# Strix - Streaming File Server

A modern streaming file server with TMDB integration built with Go and vanilla JavaScript.

## Features

- 🎬 **TMDB Integration** - Fetch real-time movie and TV series data
- 🔍 **Search Functionality** - Search both TMDB content and local files
- 📁 **File Management** - Browse and stream local media files
- 🎨 **Modern UI** - Beautiful, responsive design with smooth animations
- ⚡ **Fast Performance** - Built with Go for optimal speed

## Prerequisites

- Go 1.21 or higher
- TMDB API Key (get it from https://www.themoviedb.org/settings/api)

## Setup Instructions

### 1. Clone the repository

```bash
cd "c:\Users\Amarnath\Programs\My Projects\Strix"
```

### 2. Install Go dependencies

```bash
go mod download
```

### 3. Configure environment variables

Create a `.env` file in the root directory:

```env
TMDB_API_KEY=your_api_key_here
PORT=8080
FILES_DIR=./media
```

### 4. Create media directory

```bash
mkdir media
```

Place your video files in the `media` directory. The server supports:
- MP4, MKV, AVI, MOV, WMV, FLV, WebM, M4V

### 5. Run the server

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Search
```
GET /api/search?q=stranger+things
```

### TV Series Details
```
GET /api/tv/{id}
```

### Season Details
```
GET /api/tv/{id}/season/{season_number}
```

### Movie Details
```
GET /api/movie/{id}
```

### Trending
```
GET /api/trending?type=all&time=week
```

### List Files
```
GET /api/files
```

### Search Files
```
GET /api/files/search?q=stranger
```

## Project Structure

```
Strix/
├── main.go              # Go server
├── go.mod               # Go dependencies
├── .env                 # Environment variables
├── index.html           # Main HTML template
├── styles.css           # Styles
├── script.js            # Client-side JavaScript
├── media/               # Video files directory
└── README.md            # This file
```

## Usage

### Browse Content
Visit `http://localhost:8080` to browse trending content

### Search
Use the search bar to find movies and TV shows from TMDB

### View Details
Click on any title to view detailed information including:
- Cast and crew
- Episodes (for TV shows)
- Recommendations
- Available local files

### Stream Files
Click on file cards to stream available video files

## Development

### Adding New Features

1. Add new routes in `setupRoutes()` function
2. Create handler functions
3. Update frontend JavaScript to call new endpoints

### Customization

- Modify `styles.css` for UI changes
- Update `script.js` for behavior changes
- Edit `index.html` for structure changes

## License

MIT License

## Credits

- Movie/TV data provided by [TMDB](https://www.themoviedb.org/)
- UI inspired by modern streaming platforms
