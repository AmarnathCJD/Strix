// API Configuration
const API_BASE = '/api';
const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

// Global state
let currentMediaId = null;
let currentMediaType = 'tv';
let currentSeason = 1;

// Initialize app
document.addEventListener('DOMContentLoaded', function() {
    initializeApp();
    setupSearchFunctionality();
    setupEventListeners();
    setupPlayButton();
});

// Initialize application
function initializeApp() {
    // Check if mediaData is provided by server-side rendering
    if (window.mediaData) {
        currentMediaId = window.mediaData.id;
        currentMediaType = window.mediaData.type;
        
        // Fetch IMDB rating if IMDB ID is available
        if (window.mediaData.imdbId) {
            fetchIMDBRating(window.mediaData.imdbId);
        }
        
        // For server-rendered pages, only load dynamic content
        if (currentMediaType === 'tv') {
            loadSeasonEpisodesForCurrentShow();
            loadRecommendationsForMedia(currentMediaType, currentMediaId);
        } else if (currentMediaType === 'movie') {
            loadRecommendationsForMedia(currentMediaType, currentMediaId);
        }
    } else {
        // Get media ID from URL if present (fallback)
        const path = window.location.pathname;
        const tvMatch = path.match(/\/tv\/(\d+)/);
        const movieMatch = path.match(/\/movie\/(\d+)/);
        
        if (tvMatch) {
            currentMediaId = tvMatch[1];
            currentMediaType = 'tv';
            loadTVDetails(currentMediaId);
        } else if (movieMatch) {
            currentMediaId = movieMatch[1];
            currentMediaType = 'movie';
            loadMovieDetails(currentMediaId);
        } else {
            loadRecentlyAdded();
            loadTrendingContent();
        }
    }
    
    const filesGrid = document.getElementById('filesGrid');
    if (filesGrid) {
        loadAvailableFiles();
    }
}

// Load recommendations for server-rendered pages
async function loadRecommendationsForMedia(type, id) {
    try {
        const response = await fetch(`${API_BASE}/${type}/${id}`);
        const data = await response.json();
        
        if (data.recommendations && data.recommendations.results) {
            updateRecommendations(data.recommendations.results);
        }
    } catch (error) {
        console.error('Error loading recommendations:', error);
    }
}

// Load season episodes for server-rendered TV pages
function loadSeasonEpisodesForCurrentShow() {
    const seasonSelect = document.getElementById('heroSeason');
    if (seasonSelect && currentMediaId) {
        const selectedSeason = seasonSelect.value || 1;
        loadSeasonEpisodes(currentMediaId, selectedSeason);
        
        seasonSelect.addEventListener('change', function() {
            loadSeasonEpisodes(currentMediaId, this.value);
        });
    }
}

// Search Functionality
function setupSearchFunctionality() {
    const searchInput = document.querySelector('.search-input');
    const searchContainer = document.querySelector('.search-container');
    
    let searchTimeout;
    let searchDropdown = null;
    
    if (searchInput) {
        // Create search results dropdown
        searchDropdown = document.createElement('div');
        searchDropdown.className = 'search-results-dropdown';
        searchContainer.appendChild(searchDropdown);
        
        searchInput.addEventListener('input', function(e) {
            clearTimeout(searchTimeout);
            const query = e.target.value.trim();
            
            if (query.length >= 2) {
                showSearchLoading();
                searchTimeout = setTimeout(() => performSearch(query), 500);
            } else {
                hideSearchResults();
            }
        });
        
        searchInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                const query = e.target.value.trim();
                if (query) {
                    performSearch(query);
                }
            }
        });
        
        // Close dropdown when clicking outside
        document.addEventListener('click', function(e) {
            if (!searchContainer.contains(e.target)) {
                hideSearchResults();
            }
        });
        
        searchInput.addEventListener('focus', function() {
            if (searchDropdown.children.length > 0) {
                searchDropdown.classList.add('visible');
            }
        });
    }
}

function showSearchLoading() {
    const dropdown = document.querySelector('.search-results-dropdown');
    if (dropdown) {
        dropdown.innerHTML = '<div class="search-loading">Searching...</div>';
        dropdown.classList.add('visible');
    }
}

function hideSearchResults() {
    const dropdown = document.querySelector('.search-results-dropdown');
    if (dropdown) {
        dropdown.classList.remove('visible');
    }
}

async function performSearch(query) {
    if (!query) return;
    
    try {
        const response = await fetch(`${API_BASE}/search?q=${encodeURIComponent(query)}`);
        const data = await response.json();
        
        if (data.results && data.results.length > 0) {
            displaySearchResults(data.results);
        } else {
            displayNoResults();
        }
    } catch (error) {
        console.error('Search error:', error);
        displaySearchError();
    }
}

function displaySearchResults(results) {
    const dropdown = document.querySelector('.search-results-dropdown');
    if (!dropdown) return;
    
    dropdown.innerHTML = '';
    
    results.slice(0, 8).forEach(result => {
        const item = document.createElement('div');
        item.className = 'search-result-item';
        
        const posterPath = result.poster_path 
            ? `https://image.tmdb.org/t/p/w92${result.poster_path}`
            : 'https://via.placeholder.com/50x75?text=No+Image';
        
        const title = result.title || result.name || 'Unknown';
        const year = result.release_date || result.first_air_date || '';
        const yearText = year ? year.split('-')[0] : 'N/A';
        const type = result.media_type === 'tv' ? 'TV Series' : 'Movie';
        const rating = result.vote_average ? result.vote_average.toFixed(1) : 'N/A';
        
        item.innerHTML = `
            <img src="${posterPath}" alt="${title}" class="search-result-poster">
            <div class="search-result-info">
                <div class="search-result-title">${title}</div>
                <div class="search-result-meta">
                    <span class="search-result-type">${type}</span>
                    <span>${yearText}</span>
                    <div class="search-result-rating">
                        <svg viewBox="0 0 24 24">
                            <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                        </svg>
                        <span>${rating}</span>
                    </div>
                </div>
            </div>
        `;
        
        item.addEventListener('click', () => {
            if (result.media_type === 'tv') {
                window.location.href = `/tv/${result.id}`;
            } else if (result.media_type === 'movie') {
                window.location.href = `/movie/${result.id}`;
            }
        });
        
        dropdown.appendChild(item);
    });
    
    dropdown.classList.add('visible');
}

function displayNoResults() {
    const dropdown = document.querySelector('.search-results-dropdown');
    if (dropdown) {
        dropdown.innerHTML = '<div class="search-no-results">No results found</div>';
        dropdown.classList.add('visible');
    }
}

function displaySearchError() {
    const dropdown = document.querySelector('.search-results-dropdown');
    if (dropdown) {
        dropdown.innerHTML = '<div class="search-no-results">Search failed. Please try again.</div>';
        dropdown.classList.add('visible');
    }
}

// Load TV Details
async function loadTVDetails(tvId) {
    try {
        const response = await fetch(`${API_BASE}/tv/${tvId}`);
        const data = await response.json();
        
        updatePageWithTVData(data);
        
        // Load first season episodes
        if (data.seasons && data.seasons.length > 0) {
            const firstSeason = data.seasons.find(s => s.season_number > 0) || data.seasons[0];
            loadSeasonEpisodes(tvId, firstSeason.season_number);
        }
        
        // Load recommendations
        if (data.recommendations && data.recommendations.results) {
            updateRecommendations(data.recommendations.results);
        }
    } catch (error) {
        console.error('Error loading TV details:', error);
        showToast('Failed to load TV show details');
    }
}

function updatePageWithTVData(data) {
    // Update title
    const titleElement = document.querySelector('.hero-title');
    if (titleElement) titleElement.textContent = data.name;
    
    // Update description
    const descElement = document.querySelector('.hero-description');
    if (descElement) descElement.textContent = data.overview;
    
    // Update rating
    const ratingElement = document.querySelector('.rating-value');
    if (ratingElement) ratingElement.textContent = data.vote_average.toFixed(1);
    
    // Update backdrop
    const backdropImg = document.querySelector('.backdrop-img');
    if (backdropImg && data.backdrop_path) {
        backdropImg.src = `${TMDB_IMAGE_BASE}/original${data.backdrop_path}`;
    }
    
    // Update poster
    const posterImg = document.querySelector('.poster-img');
    if (posterImg && data.poster_path) {
        posterImg.src = `${TMDB_IMAGE_BASE}/w500${data.poster_path}`;
    }
    
    // Update metadata
    const yearElement = document.querySelector('.meta-item:nth-child(3)');
    if (yearElement && data.first_air_date) {
        yearElement.textContent = new Date(data.first_air_date).getFullYear();
    }
    
    const seasonsElement = document.querySelector('.meta-item:nth-child(5)');
    if (seasonsElement) {
        seasonsElement.textContent = `${data.number_of_seasons} Season${data.number_of_seasons > 1 ? 's' : ''}`;
    }
    
    // Update genres
    updateGenres(data.genres);
    
    // Update production info
    if (data.production_countries && data.production_countries.length > 0) {
        const countryElement = document.querySelector('.info-value');
        if (countryElement) {
            countryElement.textContent = data.production_countries.map(c => c.name).join(', ');
        }
    }
}

// Load Season Episodes
async function loadSeasonEpisodes(tvId, seasonNumber) {
    try {
        const response = await fetch(`${API_BASE}/tv/${tvId}/season/${seasonNumber}`);
        const data = await response.json();
        
        const availableResponse = await fetch(`${API_BASE}/media/tv/${tvId}/season/${seasonNumber}`);
        const availableFiles = await availableResponse.json();
        
        updateEpisodesDisplay(data.episodes, availableFiles);
        updateEpisodesList(data.episodes, availableFiles);
    } catch (error) {
        console.error('Error loading season:', error);
        showToast('Failed to load episodes');
    }
}

function updateEpisodesDisplay(episodes, availableFiles = {}) {
    const heroEpisodesList = document.getElementById('heroEpisodesList');
    if (!heroEpisodesList) return;
    
    heroEpisodesList.innerHTML = '';
    
    episodes.forEach(episode => {
        const episodeNum = episode.episode_number;
        const isAvailable = availableFiles[episodeNum]?.available || false;
        
        const episodeItem = document.createElement('div');
        episodeItem.className = 'hero-episode-item';
        if (!isAvailable) {
            episodeItem.classList.add('unavailable');
        }
        
        episodeItem.innerHTML = `
            <div class="hero-ep-thumb">
                <img src="${episode.still_path ? TMDB_IMAGE_BASE + '/w300' + episode.still_path : 'https://via.placeholder.com/300x169?text=No+Image'}" 
                     alt="Episode ${episodeNum}">
                <div class="hero-ep-play">
                    ${isAvailable ? 
                        '<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>' :
                        '<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>'
                    }
                </div>
                ${isAvailable ? '<div class="available-badge">✓</div>' : ''}
            </div>
            <div class="hero-ep-info">
                <span class="hero-ep-num">${episodeNum}</span>
                <span class="hero-ep-title">${episode.name}</span>
                <span class="hero-ep-duration">${episode.runtime || 45}m</span>
            </div>
        `;
        
        if (isAvailable) {
            episodeItem.style.cursor = 'pointer';
            episodeItem.addEventListener('click', () => {
                playEpisode(availableFiles[episodeNum]);
            });
        }
        
        heroEpisodesList.appendChild(episodeItem);
    });
}

function updateEpisodesList(episodes, availableFiles = {}) {
    console.log('Episodes loaded:', episodes.length);
}

// Load Movie Details
async function loadMovieDetails(movieId) {
    try {
        const response = await fetch(`${API_BASE}/movie/${movieId}`);
        const data = await response.json();
        
        updatePageWithMovieData(data);
        
        const availableResponse = await fetch(`${API_BASE}/media/movie/${movieId}`);
        const availableFile = await availableResponse.json();
        
        if (availableFile.available) {
            addPlayButton(availableFile);
        }
        
        if (data.recommendations && data.recommendations.results) {
            updateRecommendations(data.recommendations.results);
        }
    } catch (error) {
        console.error('Error loading movie details:', error);
        showToast('Failed to load movie details');
    }
}

function updatePageWithMovieData(data) {
    // Similar to updatePageWithTVData but for movies
    const titleElement = document.querySelector('.hero-title');
    if (titleElement) titleElement.textContent = data.title;
    
    const descElement = document.querySelector('.hero-description');
    if (descElement) descElement.textContent = data.overview;
    
    // Update images, ratings, etc.
    updateImages(data);
    updateGenres(data.genres);
}

// Load Trending Content
async function loadTrendingContent() {
    try {
        const response = await fetch(`${API_BASE}/trending`);
        const data = await response.json();
        
        if (data && data.results && data.results.length > 0) {
            const trendingGrid = document.getElementById('trendingGrid');
            if (!trendingGrid) return;
            
            trendingGrid.innerHTML = '';
            
            data.results.forEach(item => {
                const card = createRecommendationCard(item);
                trendingGrid.appendChild(card);
            });
        }
    } catch (error) {
        console.error('Error loading trending:', error);
    }
}

// Load Recently Added from Database
async function loadRecentlyAdded() {
    try {
        const response = await fetch(`${API_BASE}/files?limit=16`);
        const files = await response.json();
        
        if (files && files.length > 0) {
            const uniqueMedia = getUniqueMediaFromFiles(files);
            await loadTMDBDetailsForRecent(uniqueMedia);
        }
    } catch (error) {
        console.error('Error loading recently added:', error);
    }
}

function getUniqueMediaFromFiles(files) {
    const mediaMap = new Map();
    
    files.forEach(file => {
        const key = `${file.media_type}-${file.tmdb_id}`;
        if (!mediaMap.has(key)) {
            mediaMap.set(key, {
                tmdb_id: file.tmdb_id,
                media_type: file.media_type,
                title: file.title,
                created_at: file.created_at
            });
        }
    });
    
    return Array.from(mediaMap.values()).slice(0, 16);
}

async function loadTMDBDetailsForRecent(mediaList) {
    const recentlyAddedGrid = document.getElementById('recentlyAddedGrid');
    if (!recentlyAddedGrid) return;
    
    recentlyAddedGrid.innerHTML = '';
    
    for (const media of mediaList) {
        try {
            const endpoint = media.media_type === 'tv' ? 'tv' : 'movie';
            const response = await fetch(`${API_BASE}/${endpoint}/${media.tmdb_id}`);
            const data = await response.json();
            
            if (data.id) {
                data.media_type = media.media_type;
                const card = createRecommendationCard(data);
                recentlyAddedGrid.appendChild(card);
            }
        } catch (error) {
            console.error(`Error loading TMDB details for ${media.title}:`, error);
        }
    }
    
    if (recentlyAddedGrid.children.length === 0) {
        recentlyAddedGrid.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: var(--text-secondary); padding: 2rem;">No recently added content</p>';
    }
}

// Update Recommendations
function updateRecommendations(items) {
    const grid = document.getElementById('recommendationsGrid') || document.querySelector('.recommendations-grid');
    if (!grid) return;
    
    updateRecommendationsInGrid(items, grid);
}

function updateRecommendationsInGrid(items, grid) {
    grid.innerHTML = '';
    
    if (!items || items.length === 0) {
        grid.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: var(--text-secondary); padding: 2rem;">No recommendations available</p>';
        return;
    }
    
    items.slice(0, 16).forEach(item => {
        const card = createRecommendationCard(item);
        grid.appendChild(card);
    });
}

function createRecommendationCard(item) {
    const card = document.createElement('div');
    card.className = 'recommendation-card';
    
    const title = item.title || item.name;
    const year = item.release_date || item.first_air_date;
    const mediaType = item.media_type || 'tv';
    
    card.innerHTML = `
        <div class="recommendation-poster">
            <img src="${item.poster_path ? TMDB_IMAGE_BASE + '/w400' + item.poster_path : 'https://via.placeholder.com/400x600?text=No+Poster'}" 
                 alt="${title}">
            <div class="recommendation-overlay"></div>
            <div class="rating-badge">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="#ffd700">
                    <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                </svg>
                <span>${item.vote_average.toFixed(1)}</span>
            </div>
        </div>
        <div class="recommendation-details">
            <button class="recommendation-play-btn">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M8 5v14l11-7z"/>
                </svg>
            </button>
            <div class="recommendation-meta">
                <span class="recommendation-year">${year ? new Date(year).getFullYear() : 'N/A'}</span>
            </div>
            <h3 class="recommendation-title">${title}</h3>
            <p class="recommendation-description">${item.overview || 'No description available'}</p>
        </div>
    `;
    
    card.addEventListener('click', () => {
        window.location.href = `/${mediaType}/${item.id}`;
    });
    
    return card;
}

// Load Available Files
async function loadAvailableFiles() {
    try {
        const response = await fetch(`${API_BASE}/files`);
        const files = await response.json();
        
        updateFilesDisplay(files);
    } catch (error) {
        console.error('Error loading files:', error);
    }
}

function updateFilesDisplay(files) {
    const filesGrid = document.getElementById('filesGrid');
    if (!filesGrid) return;
    
    filesGrid.innerHTML = '';
    
    if (files.length === 0) {
        filesGrid.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: var(--text-secondary); padding: 2rem;">No files available</p>';
        return;
    }
    
    files.forEach(file => {
        const card = createFileCard(file);
        filesGrid.appendChild(card);
    });
}

function createFileCard(file) {
    const card = document.createElement('div');
    card.className = 'file-card';
    card.setAttribute('data-quality', file.quality || 'unknown');
    
    const qualityBadge = getQualityBadgeHTML(file.quality);
    const fileSize = formatFileSize(file.file_size || 0);
    const fileName = file.file_name || file.title || 'Unknown';
    const format = file.file_name ? file.file_name.split('.').pop() : 'mp4';
    const streamToken = file.stream_token || '';
    
    card.innerHTML = `
        ${qualityBadge}
        <div class="file-details">
            <h4 class="file-name" title="${fileName}">${fileName}</h4>
            <div class="file-meta">
                <span class="file-size">${fileSize}</span>
            </div>
        </div>
        <div class="file-specs">
            <span class="spec-badge">${format.toUpperCase()}</span>
        </div>
    `;
    
    if (streamToken) {
        card.dataset.streamToken = streamToken;
        card.style.cursor = 'pointer';
        card.addEventListener('click', () => {
            window.location.href = `/play?token=${streamToken}`;
        });
    }
    
    return card;
}

function getQualityBadgeHTML(quality) {
    if (!quality) quality = 'sd';
    const badgeClass = `quality-${quality.toLowerCase()}`;
    const label = quality === '4k' ? '4K UHD' : quality.toUpperCase();
    return `<div class="file-quality-badge ${badgeClass}">${label}</div>`;
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

// Update Genres
function updateGenres(genres) {
    const genresContainer = document.querySelector('.genre-tags');
    if (!genresContainer) return;
    
    genresContainer.innerHTML = '';
    genres.slice(0, 3).forEach(genre => {
        const tag = document.createElement('span');
        tag.className = 'genre-tag';
        tag.textContent = genre.name;
        genresContainer.appendChild(tag);
    });
}

// Update Images
function updateImages(data) {
    if (data.backdrop_path) {
        const backdrop = document.querySelector('.backdrop-img');
        if (backdrop) backdrop.src = `${TMDB_IMAGE_BASE}/original${data.backdrop_path}`;
    }
    
    if (data.poster_path) {
        const poster = document.querySelector('.poster-img');
        if (poster) poster.src = `${TMDB_IMAGE_BASE}/w500${data.poster_path}`;
    }
}

// Event Listeners
function setupEventListeners() {
    // Season selector
    const seasonSelectors = document.querySelectorAll('.hero-season-select, #season');
    seasonSelectors.forEach(selector => {
        selector.addEventListener('change', function() {
            if (currentMediaId && currentMediaType === 'tv') {
                loadSeasonEpisodes(currentMediaId, this.value);
            }
        });
    });
    
    // Quality filter buttons
    const filterButtons = document.querySelectorAll('.filter-btn');
    filterButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            const quality = this.getAttribute('data-quality');
            filterFilesByQuality(quality);
            
            // Update active state
            filterButtons.forEach(b => b.classList.remove('active'));
            this.classList.add('active');
        });
    });
    
    // Navbar scroll effect
    let lastScroll = 0;
    window.addEventListener('scroll', () => {
        const navbar = document.querySelector('.navbar');
        const currentScroll = window.pageYOffset;
        
        if (currentScroll > 100) {
            navbar.style.background = 'rgba(10, 10, 10, 0.95)';
            navbar.style.backdropFilter = 'blur(10px)';
        } else {
            navbar.style.background = 'transparent';
            navbar.style.backdropFilter = 'none';
        }
        
        lastScroll = currentScroll;
    });
}

function filterFilesByQuality(quality) {
    const fileCards = document.querySelectorAll('.file-card');
    let visibleCount = 0;
    
    fileCards.forEach(card => {
        const cardQuality = card.getAttribute('data-quality');
        
        if (quality === 'all' || cardQuality === quality) {
            card.style.display = 'block';
            visibleCount++;
        } else {
            card.style.display = 'none';
        }
    });
    
    showToast(`Showing ${visibleCount} ${quality === 'all' ? '' : quality} file${visibleCount !== 1 ? 's' : ''}`);
}

// Toast Notification
function showToast(message) {
    let toast = document.getElementById('toast');
    
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'toast';
        toast.className = 'toast';
        toast.innerHTML = '<span class="toast-message"></span>';
        document.body.appendChild(toast);
    }
    
    const toastMessage = toast.querySelector('.toast-message');
    if (toastMessage) {
        toastMessage.textContent = message;
    }
    
    toast.classList.add('show');
    
    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

// Utility Functions
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Fetch IMDB Rating asynchronously
async function fetchIMDBRating(imdbId) {
    if (!imdbId) return;
    
    try {
        const ratingElement = document.querySelector('.rating-value');
        const sourceElement = document.querySelector('.rating-source');
        
        if (ratingElement) {
            // Show loading state
            ratingElement.textContent = '...';
            ratingElement.style.opacity = '0.5';
        }
        
        const response = await fetch(`${API_BASE}/imdb/${imdbId}`);
        const data = await response.json();
        
        if (data.error || !data.rating) {
            console.error('Failed to fetch IMDB rating:', data.error);
            // Keep TMDB rating if IMDB fetch fails
            if (ratingElement) {
                ratingElement.style.opacity = '1';
            }
            return;
        }
        
        // Update UI with IMDB rating
        if (ratingElement) {
            ratingElement.textContent = data.rating;
            ratingElement.style.opacity = '1';
            
            // Add smooth animation
            ratingElement.style.transform = 'scale(1.1)';
            setTimeout(() => {
                ratingElement.style.transform = 'scale(1)';
            }, 200);
        }
        
        if (sourceElement) {
            sourceElement.textContent = 'IMDb';
        }
        
        // Add votes count if available
        if (data.votes && data.votes !== 'N/A') {
            const metaInfo = document.querySelector('.meta-info');
            if (metaInfo && !document.querySelector('.imdb-votes')) {
                const votesSpan = document.createElement('span');
                votesSpan.className = 'meta-item imdb-votes';
                votesSpan.textContent = `${data.votes} votes`;
                votesSpan.style.opacity = '0';
                metaInfo.appendChild(document.createTextNode(' '));
                metaInfo.appendChild(document.createElement('span')).className = 'meta-dot';
                metaInfo.lastChild.textContent = '•';
                metaInfo.appendChild(document.createTextNode(' '));
                metaInfo.appendChild(votesSpan);
                
                // Fade in
                setTimeout(() => {
                    votesSpan.style.transition = 'opacity 0.3s';
                    votesSpan.style.opacity = '1';
                }, 100);
            }
        }
        
    } catch (error) {
        console.error('Error fetching IMDB rating:', error);
    }
}

function playEpisode(fileInfo) {
    if (fileInfo.stream_token) {
        window.location.href = `/play?token=${fileInfo.stream_token}`;
    } else {
        showToast('Stream not available');
    }
}

function addPlayButton(fileInfo) {
    const heroActions = document.querySelector('.hero-actions');
    if (!heroActions) return;
    
    const existingBtn = document.getElementById('playMovieBtn');
    if (existingBtn) existingBtn.remove();
    
    const playBtn = document.createElement('button');
    playBtn.id = 'playMovieBtn';
    playBtn.className = 'hero-btn hero-btn-primary';
    playBtn.innerHTML = `
        <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
        </svg>
        <span>Play Now</span>
    `;
    playBtn.addEventListener('click', () => {
        if (fileInfo.stream_token) {
            window.location.href = `/play?token=${fileInfo.stream_token}`;
        } else {
            showToast('Stream not available');
        }
    });
    
    heroActions.insertBefore(playBtn, heroActions.firstChild);
}

// Setup Play Button for Movie/TV pages
function setupPlayButton() {
    const playButton = document.getElementById('playButton');
    if (!playButton) return;
    
    playButton.addEventListener('click', async () => {
        try {
            playButton.disabled = true;
            playButton.innerHTML = '<span>Loading...</span>';
            
            const endpoint = currentMediaType === 'tv' 
                ? `/api/media/tv/${currentMediaId}/season/1` 
                : `/api/media/movie/${currentMediaId}`;
            
            const response = await fetch(endpoint);
            const data = await response.json();
            
            // Handle both movie response (direct object) and TV response (array)
            let fileToPlay = null;
            
            if (currentMediaType === 'movie') {
                // For movies, the response is the file object itself
                if (data && data.stream_token) {
                    fileToPlay = data;
                } else if (data && data.available && data.stream_token) {
                    fileToPlay = data;
                }
            } else {
                // For TV shows, check files array
                if (data.files && data.files.length > 0) {
                    fileToPlay = data.files[0];
                }
            }
            
            if (fileToPlay && fileToPlay.stream_token) {
                window.location.href = `/play?token=${fileToPlay.stream_token}`;
            } else {
                console.log('API response:', data);
                showToast('No files available for this title');
                resetPlayButton(playButton);
            }
        } catch (error) {
            console.error('Error loading file:', error);
            showToast('Failed to load media');
            resetPlayButton(playButton);
        }
    });
}

function resetPlayButton(button) {
    button.disabled = false;
    button.innerHTML = `
        <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
        </svg>
        Play Now
    `;
}

window.StrixAPI = {
    loadTVDetails,
    loadMovieDetails,
    performSearch,
    loadAvailableFiles,
    fetchIMDBRating
};
