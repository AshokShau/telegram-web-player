const tg = window.Telegram ? window.Telegram.WebApp : null;
const noTgOverlay = document.getElementById('no-tg-overlay');

if (!tg || !tg.initData || tg.initData.trim() === '') {
    if (noTgOverlay) noTgOverlay.style.display = 'flex';
    const joinOv = document.getElementById('join-overlay');
    if (joinOv) joinOv.style.display = 'none';
    throw new Error('Telegram WebApp context required');
}

tg.expand();
tg.ready();

const urlParams = new URLSearchParams(window.location.search);
let startParam = (tg.initDataUnsafe && tg.initDataUnsafe.start_param) ? tg.initDataUnsafe.start_param : null;
if (!startParam) {
    startParam = urlParams.get('tgWebAppStartParam') || urlParams.get('startapp') || urlParams.get('chat_id') || urlParams.get('room');
}
let roomId = startParam || '-100000000069';

var audio = document.getElementById('audio-element');
var joinOverlay = document.getElementById('join-overlay');
var btnJoin = document.getElementById('btn-join');
var idleView = document.getElementById('idle-view');
var activePlayerView = document.getElementById('active-player-view');
var trackTitle = document.getElementById('track-title');
var trackArtist = document.getElementById('track-artist');
var requesterName = document.getElementById('requester-name');
var trackThumb = document.getElementById('track-thumb');
var artContainer = document.getElementById('art-container');
var platformBadge = document.getElementById('platform-badge');
var ambientGlow = document.getElementById('ambient-glow');
var roleBadge = document.getElementById('role-badge');
var roleText = document.getElementById('role-text');
var listenersTrigger = document.getElementById('listeners-trigger');
var listenersCountText = document.getElementById('listeners-count-text');
var seekSlider = document.getElementById('seek-slider');
var currTime = document.getElementById('curr-time');
var totalTime = document.getElementById('total-time');
var btnPlay = document.getElementById('btn-play');
var iconPlay = document.getElementById('icon-play');
var iconPause = document.getElementById('icon-pause');
var btnSkip = document.getElementById('btn-skip');
var btnStop = document.getElementById('btn-stop');
var btnLoop = document.getElementById('btn-loop');
var loopCountBadge = document.getElementById('loop-count-badge');
var btnAutoplay = document.getElementById('btn-autoplay');
var btnMute = document.getElementById('btn-mute');
var iconVolHigh = document.getElementById('icon-vol-high');
var iconVolMute = document.getElementById('icon-vol-mute');
var volumeSlider = document.getElementById('volume-slider');
var queueList = document.getElementById('queue-list');
var queueCount = document.getElementById('queue-count');
var btnClearQueue = document.getElementById('btn-clear-queue');
var toastMsg = document.getElementById('toast-msg');

// Search elements
var searchInput = document.getElementById('search-input');
var btnSearchSubmit = document.getElementById('btn-search-submit');
var searchModalBackdrop = document.getElementById('search-modal-backdrop');
var searchModalDrawer = document.getElementById('search-modal-drawer');
var searchModalCloseBtn = document.getElementById('search-modal-close-btn');
var searchResultsList = document.getElementById('search-results-list');

// User profile elements
var userAvatarPlaceholder = document.getElementById('user-avatar-placeholder');
var userAvatarImg = document.getElementById('user-avatar-img');
var userPremiumBadge = document.getElementById('user-premium-badge');
var userDisplayName = document.getElementById('user-display-name');
var userHandle = document.getElementById('user-handle');

// Overlay elements
var overlayAvatarPlaceholder = document.getElementById('overlay-avatar-placeholder');
var overlayAvatarImg = document.getElementById('overlay-avatar-img');
var overlayDisplayName = document.getElementById('overlay-display-name');
var overlayHandle = document.getElementById('overlay-handle');
var overlaySongTitle = document.getElementById('overlay-song-title');
var overlaySongArtist = document.getElementById('overlay-song-artist');

// Modal elements
var modalBackdrop = document.getElementById('modal-backdrop');
var modalDrawer = document.getElementById('modal-drawer');
var modalCloseBtn = document.getElementById('modal-close-btn');
var modalListenersCount = document.getElementById('modal-listeners-count');
var listenersScrollList = document.getElementById('listeners-scroll-list');

var trackDuration = 0;
var currentPosition = 0;
var serverTimeOffset = 0;
var roomState = null;
var isUserSeeking = false;
var isAdmin = false;
var canControl = false;
var canPlay = true;
var ws = null;
var isAudioUnlocked = false;
var pendingSeekPosition = null;
var playPromise = null;
var currentAudioUrl = null;
var hls = null;

function isHlsUrl(url) {
    if (!url) return false;
    try {
        const parsed = new URL(url, window.location.href);
        return parsed.pathname.toLowerCase().endsWith('.m3u8') || url.toLowerCase().includes('.m3u8');
    } catch (e) {
        return url.toLowerCase().includes('.m3u8');
    }
}

function destroyHls() {
    if (hls) {
        try {
            hls.detachMedia();
            hls.destroy();
        } catch (e) {
            console.error('Error destroying HLS instance:', e);
        }
        hls = null;
    }
}

function startAudioPlayback() {
    if (!audio.src && !audio.currentSrc) return;
    const promise = audio.play();
    if (promise !== undefined) {
        playPromise = promise;
        promise.then(() => {
            if (playPromise === promise) playPromise = null;
            if (!roomState || !roomState.track) {
                stopAudioPlayback(true);
            } else if (roomState.playback && roomState.playback.status !== 'playing') {
                stopAudioPlayback(false);
            }
        }).catch(e => {
            if (playPromise === promise) playPromise = null;
            console.log('Playback error:', e);
        });
    }
}

function stopAudioPlayback(fullStop) {
    audio.pause();
    if (fullStop) {
        destroyHls();
        currentAudioUrl = null;
    }
}

function applyPendingSeek() {
    if (pendingSeekPosition !== null && audio.readyState >= 1) {
        try {
            audio.currentTime = pendingSeekPosition;
            pendingSeekPosition = null;
        } catch (e) {
            console.log('Error setting currentTime:', e);
        }
    }
}

audio.addEventListener('loadedmetadata', applyPendingSeek);
audio.addEventListener('canplay', applyPendingSeek);
audio.addEventListener('canplaythrough', applyPendingSeek);
audio.addEventListener('seeked', () => { pendingSeekPosition = null; });

function loadAudioSource(url, targetPos) {
    destroyHls();
    currentAudioUrl = url;
    pendingSeekPosition = targetPos;

    if (!url) {
        audio.src = '';
        audio.removeAttribute('src');
        return;
    }

    if (isHlsUrl(url)) {
        if (typeof Hls !== 'undefined' && Hls.isSupported()) {
            console.log('Initializing HLS.js for URL:', url);
            hls = new Hls({
                enableWorker: true,
                lowLatencyMode: false,
            });

            hls.attachMedia(audio);
            hls.on(Hls.Events.MEDIA_ATTACHED, () => {
                hls.loadSource(url);
            });

            hls.on(Hls.Events.MANIFEST_PARSED, (event, data) => {
                console.log('HLS manifest parsed, levels:', data.levels ? data.levels.length : 0);
                applyPendingSeek();
                if (isAudioUnlocked && roomState && roomState.playback && roomState.playback.status === 'playing') {
                    startAudioPlayback();
                }
            });

            hls.on(Hls.Events.ERROR, (event, data) => {
                console.error('HLS error:', data.type, data.details, data);
                if (data.fatal) {
                    let reason = 'Fatal HLS error (' + data.type + ': ' + data.details + ')';
                    if (data.response && data.response.status) {
                        reason += ' - HTTP status ' + data.response.status;
                        if (data.response.status === 403 || data.response.status === 401) {
                            reason += ' (Token expired or unauthorized)';
                        } else if (data.response.status === 404) {
                            reason += ' (Playlist/Segment not found)';
                        }
                    } else if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
                        reason += ' - Network failure or CORS blocking';
                    } else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
                        reason += ' - Unsupported media format/codec or decode error';
                    }
                    console.error('Fatal HLS failure reason:', reason);
                    showToast(reason);

                    switch (data.type) {
                        case Hls.ErrorTypes.NETWORK_ERROR:
                            console.log('Attempting HLS network error recovery...');
                            hls.startLoad();
                            break;
                        case Hls.ErrorTypes.MEDIA_ERROR:
                            console.log('Attempting HLS media error recovery...');
                            hls.recoverMediaError();
                            break;
                        default:
                            destroyHls();
                            break;
                    }
                }
            });
        } else if (audio.canPlayType('application/vnd.apple.mpegurl') || audio.canPlayType('application/x-mpegURL')) {
            console.log('Using native HLS playback for URL:', url);
            audio.src = url;
        } else {
            console.error('HLS playback is not supported in this browser/environment');
            showToast('HLS audio playback is not supported on this device');
        }
    } else {
        audio.src = url;
    }
}

function triggerHaptic(style) {
    if (tg && tg.HapticFeedback) {
        tg.HapticFeedback.impactOccurred(style || 'light');
    }
}

function populateUserProfile() {
    if (!tg || !tg.initDataUnsafe) return;
    const u = tg.initDataUnsafe.user;
    if (u) {
        const name = (u.first_name || '') + (u.last_name ? ' ' + u.last_name : '');
        const handleText = u.username ? '@' + u.username : 'ID: ' + u.id;
        const initials = (u.first_name ? u.first_name[0] : 'U').toUpperCase();

        if (userDisplayName) userDisplayName.innerText = name || 'Telegram User';
        if (userHandle) userHandle.innerText = handleText;

        if (overlayDisplayName) overlayDisplayName.innerText = name || 'Telegram User';
        if (overlayHandle) overlayHandle.innerText = handleText;

        if (u.photo_url) {
            if (userAvatarImg) {
                userAvatarImg.src = u.photo_url;
                userAvatarImg.style.display = 'block';
            }
            if (userAvatarPlaceholder) userAvatarPlaceholder.style.display = 'none';

            if (overlayAvatarImg) {
                overlayAvatarImg.src = u.photo_url;
                overlayAvatarImg.style.display = 'block';
            }
            if (overlayAvatarPlaceholder) overlayAvatarPlaceholder.style.display = 'none';
        } else {
            if (userAvatarPlaceholder) userAvatarPlaceholder.innerText = initials;
            if (overlayAvatarPlaceholder) overlayAvatarPlaceholder.innerText = initials;
        }

        if (u.is_premium && userPremiumBadge) {
            userPremiumBadge.style.display = 'flex';
        }
    }
}

populateUserProfile();

function showToast(msg) {
    toastMsg.innerText = msg;
    toastMsg.style.display = 'block';
    setTimeout(() => { toastMsg.style.display = 'none'; }, 3000);
}

function formatTime(secs) {
    secs = Math.floor(secs || 0);
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return m + ':' + (s < 10 ? '0' : '') + s;
}

function openSearchModal() {
    triggerHaptic('light');
    if (searchModalBackdrop) searchModalBackdrop.style.display = 'block';
    setTimeout(() => {
        if (searchModalBackdrop) searchModalBackdrop.style.opacity = '1';
        if (searchModalDrawer) searchModalDrawer.style.transform = 'translateX(-50%) translateY(0)';
    }, 10);
}

function closeSearchModal() {
    triggerHaptic('light');
    if (searchModalBackdrop) searchModalBackdrop.style.opacity = '0';
    if (searchModalDrawer) searchModalDrawer.style.transform = 'translateX(-50%) translateY(100%)';
    setTimeout(() => {
        if (searchModalBackdrop) searchModalBackdrop.style.display = 'none';
    }, 300);
}

function openListenersModal() {
    triggerHaptic('light');
    modalBackdrop.style.display = 'block';
    setTimeout(() => {
        modalBackdrop.style.opacity = '1';
        modalDrawer.style.transform = 'translateX(-50%) translateY(0)';
    }, 10);
}

function closeListenersModal() {
    triggerHaptic('light');
    modalBackdrop.style.opacity = '0';
    modalDrawer.style.transform = 'translateX(-50%) translateY(100%)';
    setTimeout(() => {
        modalBackdrop.style.display = 'none';
    }, 300);
}

listenersTrigger.addEventListener('click', openListenersModal);
modalCloseBtn.addEventListener('click', closeListenersModal);
modalBackdrop.addEventListener('click', closeListenersModal);

if (searchModalCloseBtn) searchModalCloseBtn.addEventListener('click', closeSearchModal);
if (searchModalBackdrop) searchModalBackdrop.addEventListener('click', closeSearchModal);

function performSearch() {
    triggerHaptic('light');
    if (!searchInput) return;
    const query = searchInput.value.trim();
    if (!query) {
        showToast('Please enter a song name or link');
        return;
    }

    if (!canPlay && !canControl) {
        showToast('Play mode is restricted in this chat');
        return;
    }

    openSearchModal();
    if (searchResultsList) {
        searchResultsList.innerHTML = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">Searching music...</div>';
    }

    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'search', query: query }));
    } else {
        fetch('/api/search?q=' + encodeURIComponent(query))
            .then(res => res.json())
            .then(data => {
                renderSearchResults(data.results || []);
            })
            .catch(err => {
                if (searchResultsList) searchResultsList.innerHTML = '<div style="font-size: 13px; color: #fca5a5; text-align: center; padding: 20px;">Search failed. Try again.</div>';
            });
    }
}

if (btnSearchSubmit) btnSearchSubmit.addEventListener('click', performSearch);
if (searchInput) searchInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') performSearch();
});

function requestTrack(track, force) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        showToast('Connection lost. Please try again.');
        return;
    }

    triggerHaptic('medium');
    ws.send(JSON.stringify({
        type: force ? 'play' : 'enqueue',
        track: track,
        force: force
    }));

    showToast(force ? 'Playing ' + (track.title || 'track') : 'Added to queue: ' + (track.title || 'track'));
    closeSearchModal();
}

function renderSearchResults(results) {
    if (!searchResultsList) return;
    if (!results || results.length === 0) {
        searchResultsList.innerHTML = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">No results found.</div>';
        return;
    }

    let html = '';
    window._searchResults = results;
    results.forEach((item, index) => {
        const thumb = item.thumbnail || 'https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg';
        const title = item.title || 'Unknown Track';
        const artist = item.artist || item.platform || 'Music';
        const dur = formatTime(item.duration);

        html += '<div class="search-item">';
        html += '<img class="search-thumb" src="' + thumb + '" alt="thumb">';
        html += '<div class="search-details">';
        html += '<div class="search-track-title">' + title + '</div>';
        html += '<div class="search-track-sub">' + artist + ' • ' + dur + '</div>';
        html += '</div>';
        html += '<div class="search-actions">';
        if (canControl) {
            html += '<button class="btn-search-action primary" onclick="handleSearchAction(' + index + ', true)">Play</button>';
        }
        html += '<button class="btn-search-action" onclick="handleSearchAction(' + index + ', false)">+ Queue</button>';
        html += '</div></div>';
    });

    searchResultsList.innerHTML = html;
}

window.handleSearchAction = function(index, force) {
    if (!window._searchResults || !window._searchResults[index]) return;
    requestTrack(window._searchResults[index], force);
};

window.removeQueueTrack = function(index) {
    if (!canControl) return;
    triggerHaptic('medium');
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'remove', index: index }));
    }
};

btnJoin.addEventListener('click', () => {
    triggerHaptic('medium');
    isAudioUnlocked = true;
    joinOverlay.style.opacity = '0';
    joinOverlay.style.visibility = 'hidden';
    setTimeout(() => { joinOverlay.style.display = 'none'; }, 300);

    if (!roomState || !roomState.track || !roomState.track.audioUrl) {
        audio.src = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAAZGF0YQAAAAA=';
        audio.play().catch(e => console.log('Unlock audio play error:', e));
    }

    if (roomState && roomState.track && roomState.track.audioUrl) {
        let targetPos = roomState.playback ? (roomState.playback.position || 0) : 0;
        if (roomState.playback && roomState.playback.status === 'playing') {
            const nowServer = Date.now() + serverTimeOffset;
            const elapsed = (nowServer - roomState.playback.serverTime) / 1000;
            targetPos += elapsed;
        }
        loadAudioSource(roomState.track.audioUrl, targetPos);
        if (roomState.playback && roomState.playback.status === 'playing') {
            startAudioPlayback();
        }
    }

    if (roomState) {
        updateRoomState(roomState);
    }
});

connectWS();

audio.addEventListener('ended', () => {
    if (!currentAudioUrl || (audio.src && audio.src.startsWith('data:audio/'))) {
        return;
    }
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'track_end' }));
    }
});

function connectWS() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = protocol + '//' + window.location.host + '/ws?room=' + roomId;
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        ws.send(JSON.stringify({
            type: 'join',
            roomId: roomId,
            initData: tg ? tg.initData : ''
        }));
        pingServer();
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            if (msg.event === 'room_state') {
                updateRoomState(msg.data);
            } else if (msg.event === 'user_info') {
                isAdmin = msg.data.isAdmin;
                canControl = msg.data.canControl !== undefined ? msg.data.canControl : isAdmin;
                canPlay = msg.data.canPlay !== undefined ? msg.data.canPlay : true;
                if (isAdmin) {
                    roleBadge.classList.add('admin');
                    roleText.innerText = 'Admin';
                } else {
                    roleBadge.classList.remove('admin');
                    roleText.innerText = 'Listener';
                }
                updateControlButtonsState();
            } else if (msg.event === 'search_results') {
                renderSearchResults(msg.data.results || []);
            } else if (msg.event === 'pong') {
                const now = Date.now();
                const rtt = now - msg.data.clientTime;
                const serverTime = msg.data.serverTime + (rtt / 2);
                serverTimeOffset = serverTime - now;
            } else if (msg.event === 'error') {
                showToast(msg.data);
            }
        } catch (e) {
            console.error(e);
        }
    };

    ws.onclose = () => {
        setTimeout(connectWS, 2000);
    };
}

function pingServer() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping', clientTime: Date.now() }));
    }
}

setInterval(pingServer, 10000);

function updateControlButtonsState() {
    btnPlay.disabled = !canControl;
    btnSkip.disabled = !canControl;
    if (btnStop) btnStop.disabled = !canControl;
    if (btnLoop) btnLoop.disabled = !canControl;
    if (btnAutoplay) btnAutoplay.disabled = !canControl;
    seekSlider.disabled = !canControl;
    if (btnSearchSubmit) btnSearchSubmit.disabled = !canPlay && !canControl;
    if (btnClearQueue) btnClearQueue.disabled = !canControl;
}

function updateRoomState(data) {
    roomState = data;
    const listenersCount = data.listeners ? data.listeners.length : 0;
    if (listenersCountText) listenersCountText.innerText = listenersCount + (listenersCount === 1 ? ' Listener' : ' Listeners');
    if (modalListenersCount) modalListenersCount.innerText = listenersCount;

    updateListenersList(data.listeners || []);

    const track = data.track;
    const pb = data.playback;

    if (!track) {
        if (idleView) idleView.style.display = 'flex';
        if (activePlayerView) activePlayerView.style.display = 'none';
        if (ambientGlow) ambientGlow.style.opacity = '0.1';
        if (iconPlay) iconPlay.style.display = 'block';
        if (iconPause) iconPause.style.display = 'none';
        stopAudioPlayback(true);
        if (currTime) currTime.innerText = '0:00';
        if (totalTime) totalTime.innerText = '0:00';
        if (seekSlider) seekSlider.value = 0;
        if (overlaySongTitle) overlaySongTitle.innerText = 'Nothing Playing';
        if (overlaySongArtist) overlaySongArtist.innerText = 'No active track in room';
        updateQueue([]);
        return;
    }

    if (idleView) idleView.style.display = 'none';
    if (activePlayerView) activePlayerView.style.display = 'flex';

    // Update loop state
    const loopCount = data.loop || 0;
    if (btnLoop) {
        if (loopCount > 0) {
            btnLoop.classList.add('active');
            if (loopCountBadge) {
                loopCountBadge.innerText = loopCount;
                loopCountBadge.style.display = 'flex';
            }
        } else {
            btnLoop.classList.remove('active');
            if (loopCountBadge) loopCountBadge.style.display = 'none';
        }
    }

    // Update autoplay state
    if (btnAutoplay) {
        if (data.autoplay) {
            btnAutoplay.classList.add('active');
        } else {
            btnAutoplay.classList.remove('active');
        }
    }

    if (trackTitle) trackTitle.innerText = track.title || 'Unknown Track';
    if (trackArtist) trackArtist.innerText = track.artist || track.platform || 'Music';
    if (overlaySongTitle) overlaySongTitle.innerText = track.title || 'Unknown Track';
    if (overlaySongArtist) overlaySongArtist.innerText = track.artist || track.platform || 'Music';
    if (requesterName) requesterName.innerText = 'Requested by ' + (track.user || 'User');
    if (platformBadge) platformBadge.innerText = (track.platform || 'Music').toUpperCase();
    if (trackThumb) {
        if (track.thumbnail) {
            trackThumb.src = track.thumbnail;
        } else {
            trackThumb.src = 'https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg';
        }
    }
    trackDuration = track.duration || 0;
    if (totalTime) totalTime.innerText = formatTime(trackDuration);

    let targetPos = pb.position || 0;
    if (pb.status === 'playing') {
        const nowServer = Date.now() + serverTimeOffset;
        const elapsed = (nowServer - pb.serverTime) / 1000;
        targetPos += elapsed;
    }

    if (trackDuration > 0 && targetPos > trackDuration) targetPos = trackDuration;

    const audioSrc = track.audioUrl;
    let srcChanged = false;
    if (audioSrc && currentAudioUrl !== audioSrc) {
        srcChanged = true;
        loadAudioSource(audioSrc, targetPos);
    }

    if (!isUserSeeking) {
        currentPosition = targetPos;
        if (seekSlider) {
            seekSlider.max = trackDuration;
            seekSlider.value = currentPosition;
        }
        if (currTime) currTime.innerText = formatTime(currentPosition);
    }

    if (pb.status === 'playing') {
        if (iconPlay) iconPlay.style.display = 'none';
        if (iconPause) iconPause.style.display = 'block';
        if (artContainer) artContainer.classList.add('playing');
        if (ambientGlow) ambientGlow.style.opacity = '0.35';
        if (isAudioUnlocked) {
            if (srcChanged) {
                startAudioPlayback();
            } else {
                if (pendingSeekPosition === null && Math.abs(audio.currentTime - targetPos) > 1.2) {
                    pendingSeekPosition = targetPos;
                    if (audio.readyState >= 1) {
                        try {
                            audio.currentTime = targetPos;
                            pendingSeekPosition = null;
                        } catch(e) {}
                    }
                }
                if (audio.paused) {
                    startAudioPlayback();
                }
            }
        }
    } else {
        if (iconPlay) iconPlay.style.display = 'block';
        if (iconPause) iconPause.style.display = 'none';
        if (artContainer) artContainer.classList.remove('playing');
        if (ambientGlow) ambientGlow.style.opacity = '0.1';
        stopAudioPlayback(false);
        if (!srcChanged && pendingSeekPosition === null && Math.abs(audio.currentTime - targetPos) > 1.2) {
            pendingSeekPosition = targetPos;
            if (audio.readyState >= 1) {
                try {
                    audio.currentTime = targetPos;
                    pendingSeekPosition = null;
                } catch(e) {}
            }
        }
    }

    updateQueue(data.queue || []);
}

function updateListenersList(listeners) {
    if (!listenersScrollList) return;
    if (!listeners || listeners.length === 0) {
        listenersScrollList.innerHTML = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">No active listeners connected.</div>';
        return;
    }

    let html = '';
    listeners.forEach(l => {
        const fullName = (l.firstName || 'Listener') + (l.lastName ? ' ' + l.lastName : '');
        const handle = l.username ? '@' + l.username : (l.userId ? 'ID: ' + l.userId : 'Anonymous');
        const initial = (l.firstName ? l.firstName[0] : 'L').toUpperCase();

        html += '<div class="listener-item">';
        html += '<div class="listener-user-info">';
        if (l.photoUrl) {
            html += '<img class="listener-avatar" src="' + l.photoUrl + '" alt="avatar">';
        } else {
            html += '<div class="listener-avatar">' + initial + '</div>';
        }
        html += '<div class="listener-names">';
        html += '<span class="listener-name">' + fullName + '</span>';
        html += '<span class="listener-handle">' + handle + '</span>';
        html += '</div></div>';

        if (l.isAdmin) {
            html += '<div class="badge admin"><span class="badge-dot"></span><span>Admin</span></div>';
        } else {
            html += '<div class="badge"><span id="role-text">Listener</span></div>';
        }
        html += '</div>';
    });
    listenersScrollList.innerHTML = html;
}

function updateQueue(queue) {
    if (queueCount) queueCount.innerText = (queue ? queue.length : 0) + ' tracks';
    if (btnClearQueue) {
        if (queue && queue.length > 0 && canControl) {
            btnClearQueue.style.display = 'inline-block';
            btnClearQueue.disabled = false;
        } else {
            btnClearQueue.style.display = 'none';
        }
    }

    if (!queueList) return;
    if (!queue || queue.length === 0) {
        queueList.innerHTML = '<div style="font-size: 12px; color: var(--text-muted); text-align: center; padding: 14px;">No upcoming tracks in queue</div>';
        return;
    }

    let html = '';
    queue.forEach((item, index) => {
        html += '<div class="queue-item">';
        html += '<img class="queue-thumb" src="' + (item.thumbnail || (trackThumb ? trackThumb.src : '')) + '" alt="thumb">';
        html += '<div class="queue-details">';
        html += '<div class="queue-track-title">' + (index + 1) + '. ' + (item.title || 'Track') + '</div>';
        html += '<div class="queue-track-sub">' + formatTime(item.duration) + ' • Requested by ' + (item.user || 'User') + '</div>';
        html += '</div>';
        if (canControl) {
            html += '<button class="queue-remove-btn" onclick="removeQueueTrack(' + (index + 1) + ')" title="Remove track">';
            html += '<svg viewBox="0 0 24 24"><path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/></svg>';
            html += '</button>';
        }
        html += '</div>';
    });
    queueList.innerHTML = html;
}

btnPlay.addEventListener('click', () => {
    triggerHaptic('light');
    if (!canControl || !roomState || !roomState.track) return;
    if (roomState.playback.status === 'playing') {
        ws.send(JSON.stringify({ type: 'pause' }));
    } else {
        ws.send(JSON.stringify({ type: 'resume' }));
    }
});

btnSkip.addEventListener('click', () => {
    triggerHaptic('medium');
    if (!canControl) return;
    ws.send(JSON.stringify({ type: 'skip' }));
});

if (btnStop) {
    btnStop.addEventListener('click', () => {
        triggerHaptic('medium');
        if (!canControl) return;
        ws.send(JSON.stringify({ type: 'stop' }));
    });
}

if (btnLoop) {
    btnLoop.addEventListener('click', () => {
        triggerHaptic('light');
        if (!canControl) return;
        const currentLoop = roomState ? (roomState.loop || 0) : 0;
        let nextLoop = 0;
        if (currentLoop === 0) nextLoop = 1;
        else if (currentLoop === 1) nextLoop = 2;
        else if (currentLoop === 2) nextLoop = 5;
        else nextLoop = 0;
        ws.send(JSON.stringify({ type: 'loop', count: nextLoop }));
    });
}

if (btnAutoplay) {
    btnAutoplay.addEventListener('click', () => {
        triggerHaptic('light');
        if (!canControl) return;
        ws.send(JSON.stringify({ type: 'autoplay' }));
    });
}

if (btnClearQueue) {
    btnClearQueue.addEventListener('click', () => {
        triggerHaptic('medium');
        if (!canControl) return;
        ws.send(JSON.stringify({ type: 'clear_queue' }));
    });
}

if (volumeSlider) {
    var savedVol = localStorage.getItem('tg_player_volume');
    if (savedVol !== null) {
        var v = parseFloat(savedVol);
        audio.volume = v;
        volumeSlider.value = Math.round(v * 100);
    }

    volumeSlider.addEventListener('input', () => {
        var val = parseFloat(volumeSlider.value) / 100;
        audio.volume = val;
        localStorage.setItem('tg_player_volume', val);
        if (val === 0) {
            audio.muted = true;
            if (iconVolHigh) iconVolHigh.style.display = 'none';
            if (iconVolMute) iconVolMute.style.display = 'block';
        } else {
            audio.muted = false;
            if (iconVolHigh) iconVolHigh.style.display = 'block';
            if (iconVolMute) iconVolMute.style.display = 'none';
        }
    });
}

btnMute.addEventListener('click', () => {
    triggerHaptic('light');
    audio.muted = !audio.muted;
    if (audio.muted) {
        if (iconVolHigh) iconVolHigh.style.display = 'none';
        if (iconVolMute) iconVolMute.style.display = 'block';
    } else {
        if (audio.volume === 0) {
            audio.volume = 1;
            if (volumeSlider) volumeSlider.value = 100;
            localStorage.setItem('tg_player_volume', 1);
        }
        if (iconVolHigh) iconVolHigh.style.display = 'block';
        if (iconVolMute) iconVolMute.style.display = 'none';
    }
});

seekSlider.addEventListener('input', () => {
    if (!canControl) return;
    isUserSeeking = true;
    currTime.innerText = formatTime(seekSlider.value);
});

seekSlider.addEventListener('change', () => {
    if (!canControl) return;
    triggerHaptic('light');
    isUserSeeking = false;
    const pos = parseFloat(seekSlider.value);
    currentPosition = pos;
    pendingSeekPosition = pos;
    if (audio.readyState >= 1) {
        try {
            audio.currentTime = pos;
        } catch(e) {}
    }
    ws.send(JSON.stringify({ type: 'seek', positionSeconds: pos }));
});

setInterval(() => {
    if (roomState && roomState.playback && roomState.playback.status === 'playing' && !isUserSeeking) {
        if (isAudioUnlocked && !audio.paused && audio.readyState >= 2 && pendingSeekPosition === null) {
            currentPosition = audio.currentTime;
        } else {
            const nowServer = Date.now() + serverTimeOffset;
            const elapsed = (nowServer - roomState.playback.serverTime) / 1000;
            let pos = (roomState.playback.position || 0) + elapsed;
            if (pos > trackDuration) pos = trackDuration;
            currentPosition = pos;
        }
        seekSlider.value = currentPosition;
        currTime.innerText = formatTime(currentPosition);
    }
}, 250);
