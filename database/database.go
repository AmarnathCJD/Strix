package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {
	client *mongo.Client
	db     *mongo.Database
}

func Init(mongoURL, dbName string) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	database := &DB{
		client: client,
		db:     client.Database(dbName),
	}

	if err := database.createIndexes(ctx); err != nil {
		return nil, err
	}

	log.Printf("MongoDB connected: %s/%s", mongoURL, dbName)
	return database, nil
}

func (d *DB) createIndexes(ctx context.Context) error {
	mediaCollection := d.db.Collection("media")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "tmdb_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "media_type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "file_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "title", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "file_name", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "title", Value: "text"},
				{Key: "file_name", Value: "text"},
			},
		},
		{
			Keys: bson.D{
				{Key: "tmdb_id", Value: 1},
				{Key: "media_type", Value: 1},
				{Key: "season", Value: 1},
				{Key: "episode", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := mediaCollection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return err
	}

	usersCollection := d.db.Collection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = usersCollection.Indexes().CreateMany(ctx, userIndexes)
	if err != nil {
		return err
	}

	authUsersCollection := d.db.Collection("auth_users")
	authIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = authUsersCollection.Indexes().CreateMany(ctx, authIndexes)
	if err != nil {
		return err
	}

	settingsCollection := d.db.Collection("settings")
	settingsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = settingsCollection.Indexes().CreateMany(ctx, settingsIndexes)
	return err
}

func (d *DB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return d.client.Disconnect(ctx)
}

func (d *DB) AddMedia(tmdbID int, mediaType, title, fileID string, messageID int, chatID int64, fileSize int64, fileName string, season, episode int, quality string, cdnBotIndex int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"tmdb_id":    tmdbID,
		"media_type": mediaType,
		"season":     season,
		"episode":    episode,
	}

	update := bson.M{
		"$set": bson.M{
			"title":         title,
			"file_id":       fileID,
			"message_id":    messageID,
			"chat_id":       chatID,
			"file_size":     fileSize,
			"file_name":     fileName,
			"quality":       quality,
			"cdn_bot_index": cdnBotIndex,
			"updated_at":    time.Now(),
		},
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (d *DB) GetMediaByTMDB(tmdbID int, mediaType string, season, episode int) (*MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"tmdb_id":    tmdbID,
		"media_type": mediaType,
		"season":     season,
		"episode":    episode,
	}

	var m MediaFile
	err := collection.FindOne(ctx, filter).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (d *DB) GetSeasonEpisodes(tmdbID int, season int) ([]MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"tmdb_id":    tmdbID,
		"season":     season,
		"media_type": "tv",
	}

	opts := options.Find().SetSort(bson.D{{Key: "episode", Value: 1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var episodes []MediaFile
	if err := cursor.All(ctx, &episodes); err != nil {
		return nil, err
	}

	return episodes, nil
}

func (d *DB) SearchMedia(query string) ([]MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"$text": bson.M{
			"$search": query,
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}}).
		SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}}).
		SetLimit(100)

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []MediaFile
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (d *DB) GetAllMedia(limit, offset int) ([]MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var media []MediaFile
	if err := cursor.All(ctx, &media); err != nil {
		return nil, err
	}

	return media, nil
}

type MediaFile struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TMDBID      int                `bson:"tmdb_id" json:"tmdb_id"`
	MediaType   string             `bson:"media_type" json:"media_type"`
	Title       string             `bson:"title" json:"title"`
	FileID      string             `bson:"file_id" json:"file_id"`
	MessageID   int                `bson:"message_id" json:"message_id"`
	ChatID      int64              `bson:"chat_id" json:"chat_id"`
	FileSize    int64              `bson:"file_size" json:"file_size"`
	FileName    string             `bson:"file_name" json:"file_name"`
	Season      int                `bson:"season" json:"season"`
	Episode     int                `bson:"episode" json:"episode"`
	Quality     string             `bson:"quality" json:"quality"`
	CDNBotIndex int                `bson:"cdn_bot_index" json:"cdn_bot_index"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    int64              `bson:"user_id" json:"user_id"`
	Username  string             `bson:"username" json:"username"`
	FirstName string             `bson:"first_name" json:"first_name"`
	LastName  string             `bson:"last_name" json:"last_name"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

type AuthUser struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    int64              `bson:"user_id" json:"user_id"`
	Username  string             `bson:"username" json:"username"`
	FirstName string             `bson:"first_name" json:"first_name"`
	AddedBy   int64              `bson:"added_by" json:"added_by"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type Settings struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key       string             `bson:"key" json:"key"`
	Value     any                `bson:"value" json:"value"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	UpdatedBy int64              `bson:"updated_by" json:"updated_by"`
}

type DBStats struct {
	TotalMovies   int64   `json:"total_movies"`
	TotalTV       int64   `json:"total_tv"`
	TotalFiles    int64   `json:"total_files"`
	TotalUsers    int64   `json:"total_users"`
	DBSizeMB      float64 `json:"db_size_mb"`
	StorageSizeMB float64 `json:"storage_size_mb"`
	FreeSpaceMB   float64 `json:"free_space_mb"`
}

func (d *DB) AddUser(userID int64, username, firstName, lastName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("users")

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"username":   username,
			"first_name": firstName,
			"last_name":  lastName,
			"updated_at": time.Now(),
		},
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (d *DB) GetStats() (*DBStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats := &DBStats{}

	mediaCollection := d.db.Collection("media")
	usersCollection := d.db.Collection("users")

	totalFiles, err := mediaCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	stats.TotalFiles = totalFiles

	moviesPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"media_type": "movie"}}},
		{{Key: "$group", Value: bson.M{"_id": "$tmdb_id"}}},
		{{Key: "$count", Value: "total"}},
	}
	moviesCursor, err := mediaCollection.Aggregate(ctx, moviesPipeline)
	if err != nil {
		return nil, err
	}
	var moviesResult []bson.M
	if err := moviesCursor.All(ctx, &moviesResult); err != nil {
		return nil, err
	}
	if len(moviesResult) > 0 {
		if count, ok := moviesResult[0]["total"].(int32); ok {
			stats.TotalMovies = int64(count)
		} else if count, ok := moviesResult[0]["total"].(int64); ok {
			stats.TotalMovies = count
		}
	}

	tvPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"media_type": "tv"}}},
		{{Key: "$group", Value: bson.M{"_id": "$tmdb_id"}}},
		{{Key: "$count", Value: "total"}},
	}
	tvCursor, err := mediaCollection.Aggregate(ctx, tvPipeline)
	if err != nil {
		return nil, err
	}
	var tvResult []bson.M
	if err := tvCursor.All(ctx, &tvResult); err != nil {
		return nil, err
	}
	if len(tvResult) > 0 {
		if count, ok := tvResult[0]["total"].(int32); ok {
			stats.TotalTV = int64(count)
		} else if count, ok := tvResult[0]["total"].(int64); ok {
			stats.TotalTV = count
		}
	}

	totalUsers, err := usersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = totalUsers

	var dbStatsResult bson.M
	err = d.db.RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&dbStatsResult)
	if err == nil {
		if dataSize, ok := dbStatsResult["dataSize"].(int64); ok {
			stats.DBSizeMB = float64(dataSize) / (1024 * 1024)
		} else if dataSize, ok := dbStatsResult["dataSize"].(int32); ok {
			stats.DBSizeMB = float64(dataSize) / (1024 * 1024)
		} else if dataSize, ok := dbStatsResult["dataSize"].(float64); ok {
			stats.DBSizeMB = dataSize / (1024 * 1024)
		}

		if storageSize, ok := dbStatsResult["storageSize"].(int64); ok {
			stats.StorageSizeMB = float64(storageSize) / (1024 * 1024)
		} else if storageSize, ok := dbStatsResult["storageSize"].(int32); ok {
			stats.StorageSizeMB = float64(storageSize) / (1024 * 1024)
		} else if storageSize, ok := dbStatsResult["storageSize"].(float64); ok {
			stats.StorageSizeMB = storageSize / (1024 * 1024)
		}

		stats.FreeSpaceMB = 512.0 - stats.StorageSizeMB
		if stats.FreeSpaceMB < 0 {
			stats.FreeSpaceMB = 0
		}
	}

	return stats, nil
}

// Auth Users Management
func (d *DB) AddAuthUser(userID int64, username, firstName string, addedBy int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("auth_users")

	authUser := AuthUser{
		UserID:    userID,
		Username:  username,
		FirstName: firstName,
		AddedBy:   addedBy,
		CreatedAt: time.Now(),
	}

	_, err := collection.InsertOne(ctx, authUser)
	if err != nil && mongo.IsDuplicateKeyError(err) {
		return nil // Already exists
	}
	return err
}

func (d *DB) RemoveAuthUser(userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("auth_users")
	_, err := collection.DeleteOne(ctx, bson.M{"user_id": userID})
	return err
}

func (d *DB) IsAuthUser(userID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("auth_users")
	count, err := collection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *DB) GetAllAuthUsers() ([]AuthUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("auth_users")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var authUsers []AuthUser
	if err := cursor.All(ctx, &authUsers); err != nil {
		return nil, err
	}
	return authUsers, nil
}

// Settings Management
func (d *DB) SetSetting(key string, value interface{}, updatedBy int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("settings")

	filter := bson.M{"key": key}
	update := bson.M{
		"$set": bson.M{
			"value":      value,
			"updated_at": time.Now(),
			"updated_by": updatedBy,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (d *DB) GetSetting(key string) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("settings")

	var setting Settings
	err := collection.FindOne(ctx, bson.M{"key": key}).Decode(&setting)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return setting.Value, nil
}

func (d *DB) GetPublicAccess() (bool, error) {
	val, err := d.GetSetting("public_access")
	if err != nil {
		return false, err
	}
	if val == nil {
		return false, nil // Default is private
	}
	if b, ok := val.(bool); ok {
		return b, nil
	}
	return false, nil
}

func (d *DB) SetPublicAccess(enabled bool, updatedBy int64) error {
	return d.SetSetting("public_access", enabled, updatedBy)
}

// Search by title (groups by title for series)
func (d *DB) SearchByTitle(query string) ([]MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	// Use text search for faster queries when query has multiple words
	// Otherwise use indexed title field with regex
	var filter bson.M
	var opts *options.FindOptions

	// Text search is faster for multi-word queries
	filter = bson.M{
		"$text": bson.M{
			"$search": query,
		},
	}

	opts = options.Find().
		SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}}).
		SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}}).
		SetLimit(100)

	log.Printf("[SEARCH] Searching by title: %s", query)

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("[SEARCH] Error executing query: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []MediaFile
	if err := cursor.All(ctx, &results); err != nil {
		log.Printf("[SEARCH] Error fetching results: %v", err)
		return nil, err
	}

	log.Printf("[SEARCH] Found %d results for query: %s", len(results), query)

	return results, nil
}

// Get available seasons for a series
func (d *DB) GetAvailableSeasons(tmdbID int) ([]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	pipeline := []bson.M{
		{"$match": bson.M{"tmdb_id": tmdbID, "media_type": "tv"}},
		{"$group": bson.M{"_id": "$season"}},
		{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var seasons []int
	for cursor.Next(ctx) {
		var result struct {
			ID int `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		seasons = append(seasons, result.ID)
	}

	return seasons, nil
}

// Get available qualities for a specific episode/movie
func (d *DB) GetAvailableQualities(tmdbID int, mediaType string, season, episode int) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"tmdb_id":    tmdbID,
		"media_type": mediaType,
	}

	if mediaType == "tv" {
		filter["season"] = season
		filter["episode"] = episode
	}

	pipeline := []bson.M{
		{"$match": filter},
		{"$group": bson.M{"_id": "$quality"}},
		{"$sort": bson.M{"_id": -1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var qualities []string
	for cursor.Next(ctx) {
		var result struct {
			ID string `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		if result.ID != "" {
			qualities = append(qualities, result.ID)
		}
	}

	return qualities, nil
}

// Get all media by TMDB ID
func (d *DB) GetMediaByTMDBID(tmdbID int, mediaType string) ([]MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"tmdb_id":    tmdbID,
		"media_type": mediaType,
	}

	opts := options.Find().SetSort(bson.D{{Key: "season", Value: 1}, {Key: "episode", Value: 1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []MediaFile
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// Get episodes by season
func (d *DB) GetEpisodesBySeason(tmdbID, season int) ([]MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"tmdb_id":    tmdbID,
		"media_type": "tv",
		"season":     season,
	}

	opts := options.Find().SetSort(bson.D{{Key: "episode", Value: 1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []MediaFile
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// Get media by specific quality
func (d *DB) GetMediaByQuality(tmdbID int, mediaType string, season, episode int, quality string) (*MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"tmdb_id":    tmdbID,
		"media_type": mediaType,
		"quality":    quality,
	}

	if mediaType == "tv" {
		filter["season"] = season
		filter["episode"] = episode
	}

	var m MediaFile
	err := collection.FindOne(ctx, filter).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (d *DB) GetMediaByChatMessage(chatID int64, messageID int) (*MediaFile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := d.db.Collection("media")

	filter := bson.M{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	var m MediaFile
	err := collection.FindOne(ctx, filter).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}
