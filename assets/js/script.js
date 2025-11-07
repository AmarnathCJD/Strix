// Season and Episode Selection Functionality
document.addEventListener('DOMContentLoaded', function() {
    // Theme Picker
    const themePickerBtn = document.getElementById('themePickerBtn');
    const themePickerModal = document.getElementById('themePickerModal');
    const themeCloseBtn = document.getElementById('themeCloseBtn');
    const themeOptions = document.querySelectorAll('.theme-option');

    // Load saved theme
    const savedTheme = localStorage.getItem('theme') || 'red';
    document.documentElement.setAttribute('data-theme', savedTheme);
    
    // Set active theme option
    themeOptions.forEach(option => {
        if (option.dataset.theme === savedTheme) {
            option.classList.add('active');
        }
    });

    // Open theme picker
    if (themePickerBtn) {
        themePickerBtn.addEventListener('click', () => {
            themePickerModal.classList.add('active');
        });
    }

    // Close theme picker
    if (themeCloseBtn) {
        themeCloseBtn.addEventListener('click', () => {
            themePickerModal.classList.remove('active');
        });
    }

    // Close on backdrop click
    themePickerModal?.addEventListener('click', (e) => {
        if (e.target === themePickerModal) {
            themePickerModal.classList.remove('active');
        }
    });

    // Theme selection
    themeOptions.forEach(option => {
        option.addEventListener('click', () => {
            const theme = option.dataset.theme;
            
            // Update active state
            themeOptions.forEach(opt => opt.classList.remove('active'));
            option.classList.add('active');
            
            // Apply theme
            document.documentElement.setAttribute('data-theme', theme);
            localStorage.setItem('theme', theme);
            
            // Close modal after selection
            setTimeout(() => {
                themePickerModal.classList.remove('active');
            }, 300);
        });
    });

    const seasonSelector = document.getElementById('season');
    const episodesGrid = document.getElementById('episodesGrid');
    
    // Sample episode data for different seasons
    const episodeData = {
        1: [
            { num: 1, title: "The Vanishing of Will Byers", duration: "48m", desc: "On his way home from a friend's house, young Will sees something terrifying. Nearby, a sinister secret lurks in the depths of a government lab." },
            { num: 2, title: "The Weirdo on Maple Street", duration: "56m", desc: "Lucas, Mike and Dustin try to talk to the girl they found in the woods. Hopper questions an anxious Joyce about an unsettling phone call." },
            { num: 3, title: "Holly, Jolly", duration: "52m", desc: "An increasingly concerned Nancy looks for Barb and finds out what Jonathan's been up to. Joyce is convinced Will is trying to talk to her." },
            { num: 4, title: "The Body", duration: "50m", desc: "Refusing to believe Will is dead, Joyce tries to connect with her son. The boys give Eleven a makeover. Nancy and Jonathan form an unlikely alliance." },
            { num: 5, title: "The Flea and the Acrobat", duration: "53m", desc: "Hopper breaks into the lab while Nancy and Jonathan confront the force that took Will. The boys ask Mr. Clarke how to travel to another dimension." },
            { num: 6, title: "The Monster", duration: "47m", desc: "A frantic Jonathan looks for Nancy in the darkness, but Steve's looking for her, too. Hopper and Joyce uncover the truth about the lab's experiments." }
        ],
        2: [
            { num: 1, title: "MADMAX", duration: "49m", desc: "As the town preps for Halloween, a high-scoring rival shakes things up at the arcade, and a skeptical Hopper inspects a field of rotting pumpkins." },
            { num: 2, title: "Trick or Treat, Freak", duration: "56m", desc: "After Will sees something terrible on trick-or-treat night, Mike wonders whether Eleven is still out there. Nancy wrestles with the truth about Barb." },
            { num: 3, title: "The Pollywog", duration: "51m", desc: "Dustin adopts a strange new pet, and Eleven grows increasingly impatient. A well-meaning Bob urges Will to stand up to his fears." },
            { num: 4, title: "Will the Wise", duration: "46m", desc: "An ailing Will opens up to Joyce -- with disturbing results. While Hopper digs for the truth, Eleven unearths a surprising discovery." },
            { num: 5, title: "Dig Dug", duration: "57m", desc: "Nancy and Jonathan swap conspiracy theories with a new ally as Eleven searches for someone from her past. 'Bob the Brain' tackles a difficult problem." }
        ],
        3: [
            { num: 1, title: "Suzie, Do You Copy?", duration: "50m", desc: "Summer brings new jobs and budding romance. But the mood shifts when Dustin's radio picks up a Russian broadcast, and Will senses something is wrong." },
            { num: 2, title: "The Mall Rats", duration: "49m", desc: "Nancy and Jonathan follow a lead, Steve and Robin sign on to a secret mission, and Max and Eleven go shopping. A rattled Billy has troubling visions." },
            { num: 3, title: "The Case of the Missing Lifeguard", duration: "51m", desc: "With El and Max looking for Billy, Will declares a day without girls. Steve and Dustin go on a stakeout, and Joyce and Hopper return to Hawkins Lab." },
            { num: 4, title: "The Sauna Test", duration: "52m", desc: "A code red brings the gang back together to face a frighteningly familiar evil. Karen urges Nancy to keep digging, and Robin finds a useful map." },
            { num: 5, title: "The Flayed", duration: "53m", desc: "Strange surprises lurk inside an old farmhouse and deep beneath the Starcourt Mall. Meanwhile, the Mind Flayer is gathering strength." }
        ],
        4: [
            { num: 1, title: "The Hellfire Club", duration: "77m", desc: "A new evil awakens in Hawkins, bringing strange and supernatural terrors to the small town. Meanwhile, a new girl arrives at school and joins the Hellfire Club." },
            { num: 2, title: "Vecna's Curse", duration: "78m", desc: "Nancy and Robin investigate a disturbing lead. Meanwhile, Eddie and the boys seek out a crucial connection. Steve struggles with his future plans." },
            { num: 3, title: "The Monster and the Superhero", duration: "63m", desc: "Murray and Joyce fly to Alaska, and El faces serious consequences. Robin and Nancy dig up dirt on Hawkins' history." },
            { num: 4, title: "Dear Billy", duration: "78m", desc: "Max is in grave danger and running out of time. A patient at Pennhurst asylum has visitors. Elsewhere, in Russia, Hopper is hard at work." },
            { num: 5, title: "The Nina Project", duration: "75m", desc: "Owens takes El to Nevada, where she's forced to confront her past, while the Hawkins kids comb a crumbling house for clues." },
            { num: 6, title: "The Dive", duration: "85m", desc: "Behind the Iron Curtain, a risky rescue mission gets underway. The California crew seeks help from a hacker. Steve takes one for the team." }
        ]
    };
    
    // Random image URLs for variety
    const thumbnailImages = [
        'https://images.unsplash.com/photo-1485846234645-a62644f84728?w=500&q=80',
        'https://images.unsplash.com/photo-1536440136628-849c177e76a1?w=500&q=80',
        'https://images.unsplash.com/photo-1574267432644-f02b5ab7e2c3?w=500&q=80',
        'https://images.unsplash.com/photo-1594908900066-3f47337549d8?w=500&q=80',
        'https://images.unsplash.com/photo-1616530940355-351fabd9524b?w=500&q=80',
        'https://images.unsplash.com/photo-1608270586620-248524c67de9?w=500&q=80'
    ];
    
    // Handle season change
    if (seasonSelector) {
        seasonSelector.addEventListener('change', function() {
            const selectedSeason = this.value;
            updateEpisodes(selectedSeason);
        });
    }
    
    // Update episodes based on season
    function updateEpisodes(season) {
        const episodes = episodeData[season] || episodeData[4];
        
        episodesGrid.innerHTML = '';
        
        episodes.forEach((episode, index) => {
            const episodeCard = createEpisodeCard(episode, index);
            episodesGrid.appendChild(episodeCard);
        });
    }
    
    // Create episode card element
    function createEpisodeCard(episode, index) {
        const card = document.createElement('div');
        card.className = 'episode-card';
        card.dataset.episode = episode.num;
        
        const thumbnailUrl = thumbnailImages[index % thumbnailImages.length];
        
        card.innerHTML = `
            <div class="episode-thumbnail">
                <img src="${thumbnailUrl}" alt="Episode ${episode.num}">
                <div class="episode-overlay">
                    <button class="episode-play-btn">
                        <svg width="48" height="48" viewBox="0 0 24 24" fill="currentColor">
                            <path d="M8 5v14l11-7z"/>
                        </svg>
                    </button>
                </div>
                <span class="episode-duration">${episode.duration}</span>
            </div>
            <div class="episode-info">
                <div class="episode-header">
                    <span class="episode-number">${episode.num}</span>
                    <h3 class="episode-title">${episode.title}</h3>
                    <span class="episode-runtime">${episode.duration}</span>
                </div>
                <p class="episode-description">${episode.desc}</p>
            </div>
        `;
        
        // Add click handler for streaming (will be implemented later)
        card.addEventListener('click', function(e) {
            if (!e.target.closest('.episode-play-btn')) return;
            
            const seasonNum = seasonSelector.value;
            const episodeNum = episode.num;
            console.log(`Playing Season ${seasonNum}, Episode ${episodeNum}`);
            // Navigate to stream page (to be implemented)
            // window.location.href = `stream.html?season=${seasonNum}&episode=${episodeNum}`;
        });
        
        return card;
    }
    
    // Search functionality
    const searchInput = document.querySelector('.search-input');
    if (searchInput) {
        let searchTimeout;
        
        searchInput.addEventListener('input', function(e) {
            clearTimeout(searchTimeout);
            
            searchTimeout = setTimeout(() => {
                const query = e.target.value.toLowerCase();
                performSearch(query);
            }, 300);
        });
    }
    
    function performSearch(query) {
        // This is now handled by app.js for desktop
        // Mobile search uses performMobileSearch
        if (!query || query.length < 2) return;
        console.log('Searching for:', query);
    }
    
    function displaySearchResults(results) {
        // Handled by app.js
        console.log('Displaying results:', results);
    }
    
    // Smooth scroll behavior
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
    
    // Navbar scroll effect
    let lastScroll = 0;
    const navbar = document.querySelector('.navbar');
    
    window.addEventListener('scroll', () => {
        const currentScroll = window.pageYOffset;
        
        if (currentScroll <= 0) {
            navbar.style.background = 'linear-gradient(180deg, rgba(10, 10, 10, 0.95) 0%, rgba(10, 10, 10, 0.8) 100%)';
        } else if (currentScroll > lastScroll && currentScroll > 100) {
            // Scrolling down
            navbar.style.transform = 'translateY(-100%)';
        } else {
            // Scrolling up
            navbar.style.transform = 'translateY(0)';
            navbar.style.background = 'rgba(10, 10, 10, 0.98)';
        }
        
        lastScroll = currentScroll;
    });
    
    // Add to List functionality
    const addToListBtns = document.querySelectorAll('.btn-secondary');
    addToListBtns.forEach(btn => {
        if (btn.textContent.includes('My List')) {
            btn.addEventListener('click', function(e) {
                e.stopPropagation();
                
                const svg = this.querySelector('svg path');
                const text = Array.from(this.childNodes).find(node => node.nodeType === 3);
                
                if (this.classList.contains('in-list')) {
                    this.classList.remove('in-list');
                    svg.setAttribute('d', 'M12 5v14M5 12h14');
                    if (text) text.textContent = ' My List';
                } else {
                    this.classList.add('in-list');
                    svg.setAttribute('d', 'M5 13l4 4L19 7');
                    if (text) text.textContent = ' Added';
                }
            });
        }
    });
    
    // Recommendation card interactions
    const recommendationCards = document.querySelectorAll('.recommendation-card');
    recommendationCards.forEach(card => {
        const playBtn = card.querySelector('.recommendation-play-btn');
        
        if (playBtn) {
            playBtn.addEventListener('click', function(e) {
                e.stopPropagation();
                console.log('Playing recommendation');
                // Navigate to series page or start playing
            });
        }
        
        card.addEventListener('click', function() {
            const title = this.querySelector('.recommendation-title').textContent;
            console.log('Viewing details for:', title);
            // Navigate to series detail page
        });
    });
    
    // Lazy loading for images
    if ('IntersectionObserver' in window) {
        const imageObserver = new IntersectionObserver((entries, observer) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src || img.src;
                    img.classList.add('loaded');
                    observer.unobserve(img);
                }
            });
        });
        
        document.querySelectorAll('img').forEach(img => {
            imageObserver.observe(img);
        });
    }
    
    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Space or K to play/pause (for future video player)
        if (e.code === 'Space' || e.code === 'KeyK') {
            if (e.target.tagName !== 'INPUT') {
                e.preventDefault();
                console.log('Play/Pause triggered');
            }
        }
        
        // F for fullscreen (for future video player)
        if (e.code === 'KeyF') {
            if (e.target.tagName !== 'INPUT') {
                e.preventDefault();
                console.log('Fullscreen triggered');
            }
        }
        
        // ESC to close modals or exit fullscreen
        if (e.code === 'Escape') {
            console.log('Escape pressed');
        }
        
        // Ctrl/Cmd + F to focus search
        if ((e.ctrlKey || e.metaKey) && e.code === 'KeyF') {
            e.preventDefault();
            searchInput?.focus();
        }
    });
    
    // Handle window resize
    let resizeTimeout;
    window.addEventListener('resize', function() {
        clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(() => {
            console.log('Window resized');
            // Handle responsive adjustments if needed
        }, 250);
    });
    
    // Preload critical images
    function preloadImages(urls) {
        urls.forEach(url => {
            const img = new Image();
            img.src = url;
        });
    }
    
    // Preload hero backdrop
    preloadImages([
        'https://images.unsplash.com/photo-1574267432644-f02b5ab7e2c3?w=1920&q=80'
    ]);
    
    // Quality filter functionality
    const filterButtons = document.querySelectorAll('.filter-btn');
    const fileCards = document.querySelectorAll('.file-card');
    
    filterButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            const quality = this.getAttribute('data-quality');
            
            // Update active state
            filterButtons.forEach(b => b.classList.remove('active'));
            this.classList.add('active');
            
            // Filter files
            let visibleCount = 0;
            fileCards.forEach(card => {
                const cardQuality = card.getAttribute('data-quality');
                
                if (quality === 'all') {
                    card.style.display = 'block';
                    visibleCount++;
                } else {
                    if (cardQuality === quality) {
                        card.style.display = 'block';
                        visibleCount++;
                    } else {
                        card.style.display = 'none';
                    }
                }
            });
            
            // Show toast notification
            showToast(`Showing ${visibleCount} ${quality === 'all' ? '' : quality} file${visibleCount !== 1 ? 's' : ''}`);
        });
    });
    
    // File card click handler with staggered animation
    fileCards.forEach((card, index) => {
        // Add staggered fade-in animation
        card.style.opacity = '0';
        card.style.transform = 'translateY(20px)';
        
        setTimeout(() => {
            card.style.transition = 'opacity 0.5s ease-out, transform 0.5s ease-out';
            card.style.opacity = '1';
            card.style.transform = 'translateY(0)';
        }, index * 50);
        
        card.addEventListener('click', function() {
            const fileName = this.querySelector('.file-name')?.textContent || 'File';
            const streamToken = this.dataset.streamToken;
            
            if (streamToken) {
                // Navigate to stream page with token
                window.location.href = `/play?token=${streamToken}`;
            } else {
                showToast(`No stream available for: ${fileName}`);
            }
        });
    });
    
    // Toast notification function
    function showToast(message) {
        const toast = document.getElementById('toast');
        const toastMessage = toast.querySelector('.toast-message');
        
        if (toastMessage) {
            toastMessage.textContent = message;
        }
        
        toast.classList.add('show');
        
        setTimeout(() => {
            toast.classList.remove('show');
        }, 3000);
    }
    
    // Parallax effect on hero backdrop
    const heroBackdrop = document.querySelector('.backdrop-img');
    if (heroBackdrop) {
        window.addEventListener('scroll', () => {
            const scrolled = window.pageYOffset;
            const rate = scrolled * 0.5;
            heroBackdrop.style.transform = `scale(1.1) translateY(${rate}px)`;
        });
    }
    
    // Staggered animation for recommendation cards on scroll
    const recCards = document.querySelectorAll('.recommendation-card');
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };
    
    const cardObserver = new IntersectionObserver((entries) => {
        entries.forEach((entry, index) => {
            if (entry.isIntersecting) {
                setTimeout(() => {
                    entry.target.style.opacity = '1';
                    entry.target.style.transform = 'translateY(0)';
                }, index * 50);
                cardObserver.unobserve(entry.target);
            }
        });
    }, observerOptions);
    
    recCards.forEach(card => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(30px)';
        card.style.transition = 'opacity 0.6s ease-out, transform 0.6s ease-out';
        cardObserver.observe(card);
    });
    
    // Initialize
    console.log('Strix initialized successfully');
    
    // Hamburger menu functionality
    const hamburgerMenu = document.getElementById('hamburgerMenu');
    const mobileMenu = document.getElementById('mobileMenu');
    
    if (hamburgerMenu && mobileMenu) {
        hamburgerMenu.addEventListener('click', function(e) {
            e.stopPropagation();
            this.classList.toggle('active');
            mobileMenu.classList.toggle('active');
        });
        
        // Close menu when clicking outside
        document.addEventListener('click', function(e) {
            if (!hamburgerMenu.contains(e.target) && !mobileMenu.contains(e.target)) {
                hamburgerMenu.classList.remove('active');
                mobileMenu.classList.remove('active');
            }
        });
        
        // Close menu when clicking a link
        const mobileLinks = mobileMenu.querySelectorAll('a');
        mobileLinks.forEach(link => {
            link.addEventListener('click', function() {
                hamburgerMenu.classList.remove('active');
                mobileMenu.classList.remove('active');
            });
        });
    }
    
    // Mobile search functionality
    const mobileSearchInput = document.querySelector('.mobile-search-input');
    const mobileSearchContainer = document.querySelector('.mobile-search-container');
    
    console.log('Mobile search setup:', { 
        input: !!mobileSearchInput, 
        container: !!mobileSearchContainer 
    });
    
    if (mobileSearchInput && mobileSearchContainer) {
        let mobileSearchTimeout;
        let mobileSearchDropdown = null;
        
        // Create mobile search results dropdown
        mobileSearchDropdown = document.createElement('div');
        mobileSearchDropdown.className = 'mobile-search-results-dropdown';
        mobileSearchContainer.appendChild(mobileSearchDropdown);
        
        console.log('Mobile search dropdown created');
        
        mobileSearchInput.addEventListener('input', function(e) {
            clearTimeout(mobileSearchTimeout);
            const query = e.target.value.trim();
            
            console.log('Mobile search input:', query);
            
            if (query.length >= 2) {
                showMobileSearchLoading();
                mobileSearchTimeout = setTimeout(() => performMobileSearch(query), 500);
            } else {
                hideMobileSearchResults();
            }
        });
        
        mobileSearchInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                const query = e.target.value.trim();
                if (query && query.length >= 2) {
                    performMobileSearch(query);
                }
            }
        });
        
        // Clear search on escape
        mobileSearchInput.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                this.value = '';
                this.blur();
                hideMobileSearchResults();
            }
        });
        
        // Close dropdown when clicking outside
        document.addEventListener('click', function(e) {
            if (!mobileSearchContainer.contains(e.target)) {
                hideMobileSearchResults();
            }
        });
    }
    
    function showMobileSearchLoading() {
        const dropdown = document.querySelector('.mobile-search-results-dropdown');
        if (dropdown) {
            dropdown.innerHTML = '<div class="search-loading">Searching...</div>';
            dropdown.classList.add('visible');
        }
    }
    
    function hideMobileSearchResults() {
        const dropdown = document.querySelector('.mobile-search-results-dropdown');
        if (dropdown) {
            dropdown.classList.remove('visible');
        }
    }
    
    async function performMobileSearch(query) {
        if (!query) return;
        
        console.log('Performing mobile search for:', query);
        
        try {
            const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
            const data = await response.json();
            
            console.log('Mobile search results:', data);
            
            if (data.results && data.results.length > 0) {
                displayMobileSearchResults(data.results);
            } else {
                displayMobileNoResults();
            }
        } catch (error) {
            console.error('Mobile search error:', error);
            displayMobileSearchError();
        }
    }
    
    function displayMobileSearchResults(results) {
        const dropdown = document.querySelector('.mobile-search-results-dropdown');
        if (!dropdown) return;
        
        dropdown.innerHTML = '';
        
        results.slice(0, 8).forEach(result => {
            const item = document.createElement('div');
            item.className = 'search-result-item';
            
            const posterPath = result.poster_path 
                ? `https://image.tmdb.org/t/p/w92${result.poster_path}`
                : 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="92" height="138"%3E%3Crect fill="%23333" width="92" height="138"/%3E%3C/svg%3E';
            
            const title = result.title || result.name || 'Unknown';
            const year = result.release_date ? result.release_date.split('-')[0] : 
                         result.first_air_date ? result.first_air_date.split('-')[0] : '';
            const mediaType = result.media_type || 'movie';
            const typeLabel = mediaType === 'tv' ? 'TV Series' : 'Movie';
            
            item.innerHTML = `
                <img src="${posterPath}" alt="${title}" class="search-result-poster">
                <div class="search-result-info">
                    <div class="search-result-title">${title}</div>
                    <div class="search-result-meta">${typeLabel} ${year ? `• ${year}` : ''}</div>
                </div>
            `;
            
            item.addEventListener('click', () => {
                window.location.href = `/${mediaType}/${result.id}`;
            });
            
            dropdown.appendChild(item);
        });
        
        dropdown.classList.add('visible');
    }
    
    function displayMobileNoResults() {
        const dropdown = document.querySelector('.mobile-search-results-dropdown');
        if (dropdown) {
            dropdown.innerHTML = '<div class="search-no-results">No results found</div>';
            dropdown.classList.add('visible');
        }
    }
    
    function displayMobileSearchError() {
        const dropdown = document.querySelector('.mobile-search-results-dropdown');
        if (dropdown) {
            dropdown.innerHTML = '<div class="search-error">Search error. Please try again.</div>';
            dropdown.classList.add('visible');
        }
    }
});

// Utility functions
function formatDuration(minutes) {
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;
    
    if (hours > 0) {
        return `${hours}h ${mins}m`;
    }
    return `${mins}m`;
}

function formatDate(dateString) {
    const options = { year: 'numeric', month: 'long', day: 'numeric' };
    return new Date(dateString).toLocaleDateString('en-US', options);
}

// Export for use in other scripts
window.StrixUtils = {
    formatDuration,
    formatDate
};
