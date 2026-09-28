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

// Element Selectors
const audio = document.getElementById('audio-element');
const joinOverlay = document.getElementById('join-overlay');
const btnJoin = document.getElementById('btn-join');
const idleView = document.getElementById('idle-view');
const activePlayerView = document.getElementById('active-player-view');
const trackTitle = document.getElementById('track-title');
const trackArtist = document.getElementById('track-artist');
const requesterName = document.getElementById('requester-name');
const trackThumb = document.getElementById('track-thumb');
const artWrapper = document.getElementById('art-wrapper');
const platformBadge = document.getElementById('platform-badge');
const ambientGlow = document.getElementById('ambient-glow');
const roleBadge = document.getElementById('role-badge');
const roleText = document.getElementById('role-text');
const listenersTrigger = document.getElementById('listeners-trigger');
const listenersCountText = document.getElementById('listeners-count-text');
const seekSlider = document.getElementById('seek-slider');
const currTime = document.getElementById('curr-time');
const totalTime = document.getElementById('total-time');
const btnAddToPlaylist = document.getElementById('btn-add-to-playlist');

// Controls
const btnPlay = document.getElementById('btn-play');
const btnSkip = document.getElementById('btn-skip');
const btnStop = document.getElementById('btn-stop');
const btnLoop = document.getElementById('btn-loop');
const loopCountBadge = document.getElementById('loop-count-badge');
const btnAutoplay = document.getElementById('btn-autoplay');
const btnMixTrigger = document.getElementById('btn-mix-trigger');
const btnMute = document.getElementById('btn-mute');
const iconVolHigh = document.getElementById('icon-vol-high');
const iconVolLow = document.getElementById('icon-vol-low');
const iconVolMin = document.getElementById('icon-vol-min');
const iconVolMute = document.getElementById('icon-vol-mute');
const volumeSlider = document.getElementById('volume-slider');
const toastMsg = document.getElementById('toast-msg');

// Artwork Ring Toggle Elements
const btnToggleArtwork = document.getElementById('btn-toggle-artwork');
const artModeText = document.getElementById('art-mode-text');
const progressRingSvg = document.getElementById('progress-ring-svg');
const progressRingCircle = document.getElementById('progress-ring-circle');

// Mini Player Elements
const miniPlayer = document.getElementById('mini-player');
const miniThumb = document.getElementById('mini-thumb');
const miniTitle = document.getElementById('mini-title');
const miniArtist = document.getElementById('mini-artist');
const miniBtnPlay = document.getElementById('mini-btn-play');
const miniBtnSkip = document.getElementById('mini-btn-skip');
const miniInfoClick = document.getElementById('mini-info-click');

// User Profile Elements
const userAvatarPlaceholder = document.getElementById('user-avatar-placeholder');
const userAvatarImg = document.getElementById('user-avatar-img');
const userPremiumBadge = document.getElementById('user-premium-badge');
const userDisplayName = document.getElementById('user-display-name');
const userHandle = document.getElementById('user-handle');

// Join Overlay User Profile & Player Elements
const overlayAvatarPlaceholder = document.getElementById('overlay-avatar-placeholder');
const overlayAvatarImg = document.getElementById('overlay-avatar-img');
const overlayDisplayName = document.getElementById('overlay-display-name');
const overlayHandle = document.getElementById('overlay-handle');
const overlaySongTitle = document.getElementById('overlay-song-title');
const overlaySongArtist = document.getElementById('overlay-song-artist');

// Sleep Timer & Profile
const sleepTimerSelect = document.getElementById('sleep-timer-select');
const sleepTimerStatus = document.getElementById('sleep-timer-status');
const btnProfilePlaylists = document.getElementById('btn-profile-playlists');
let sleepTimerId = null;
let sleepEndTime = null;

// Drawers / Backdrops
const queueBackdrop = document.getElementById('queue-backdrop');
const queueDrawer = document.getElementById('queue-drawer');
const queueCloseBtn = document.getElementById('queue-close-btn');
const queueScrollList = document.getElementById('queue-scroll-list');
const drawerQueueCount = document.getElementById('drawer-queue-count');
const btnClearQueueDrawer = document.getElementById('btn-clear-queue-drawer');

const searchBackdrop = document.getElementById('search-backdrop');
const searchDrawer = document.getElementById('search-drawer');
const searchCloseBtn = document.getElementById('search-close-btn');
const modalSearchInput = document.getElementById('modal-search-input');
const btnModalSearchClear = document.getElementById('btn-modal-search-clear');
const btnModalSearchSubmit = document.getElementById('btn-modal-search-submit');
const modalSearchResults = document.getElementById('modal-search-results');

const relatedBackdrop = document.getElementById('related-backdrop');
const relatedDrawer = document.getElementById('related-drawer');
const relatedCloseBtn = document.getElementById('related-close-btn');
const btnFetchMix = document.getElementById('btn-fetch-mix');
const relatedScrollList = document.getElementById('related-scroll-list');

const listenersBackdrop = document.getElementById('listeners-backdrop');
const listenersDrawer = document.getElementById('listeners-drawer');
const listenersCloseBtn = document.getElementById('listeners-close-btn');
const listenersDrawerCount = document.getElementById('listeners-drawer-count');
const listenersScrollList = document.getElementById('listeners-scroll-list');

const profileBackdrop = document.getElementById('profile-backdrop');
const profileDrawer = document.getElementById('profile-drawer');
const profileCloseBtn = document.getElementById('profile-close-btn');
const profileRoomId = document.getElementById('profile-room-id');
const profileRoleStatus = document.getElementById('profile-role-status');
const profileConnStatus = document.getElementById('profile-conn-status');

// Bottom Nav Items
const navItemPlayer = document.getElementById('nav-item-player');
const navItemQueue = document.getElementById('nav-item-queue');
const navItemSearch = document.getElementById('nav-item-search');
const navItemRelated = document.getElementById('nav-item-related');
const navItemProfile = document.getElementById('nav-item-profile');

// Desktop Elements
const desktopLeftSidebar = document.getElementById('desktop-left-sidebar');
const desktopRightSidebar = document.getElementById('desktop-right-sidebar');
const desktopSearchInput = document.getElementById('desktop-search-input');
const btnDesktopSearchClear = document.getElementById('btn-desktop-search-clear');
const btnDesktopSearch = document.getElementById('btn-desktop-search');
const desktopSearchResults = document.getElementById('desktop-search-results');
const btnDesktopGetMix = document.getElementById('btn-desktop-get-mix');
const desktopRelatedResults = document.getElementById('desktop-related-results');
const desktopQueueList = document.getElementById('desktop-queue-list');
const btnDesktopClearQueue = document.getElementById('btn-desktop-clear-queue');

// Player State Variables
let trackDuration = 0;
let currentPosition = 0;
let serverTimeOffset = 0;
let roomState = null;
let isUserSeeking = false;
let isAdmin = false;
let canControl = false;
let canPlay = true;
let ws = null;
let isAudioUnlocked = false;
let pendingSeekPosition = null;
let playPromise = null;
let currentAudioUrl = null;
let hls = null;
let isRoundArtMode = true; // Default to round artwork with progress ring

// SVG Circumference for Progress Ring (r=100)
const RING_CIRCUMFERENCE = 2 * Math.PI * 100; // ~628

function refreshIcons() {
    if (window.lucide) {
        lucide.createIcons();
    }
}

function triggerHaptic(style) {
    if (tg && tg.HapticFeedback) {
        tg.HapticFeedback.impactOccurred(style || 'light');
    }
}

function showToast(msg) {
    if (!toastMsg) return;
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
            hls = new Hls({ enableWorker: true, lowLatencyMode: false });
            hls.attachMedia(audio);
            hls.on(Hls.Events.MEDIA_ATTACHED, () => { hls.loadSource(url); });
            hls.on(Hls.Events.MANIFEST_PARSED, () => {
                applyPendingSeek();
                if (isAudioUnlocked && roomState && roomState.playback && roomState.playback.status === 'playing') {
                    startAudioPlayback();
                }
            });
            hls.on(Hls.Events.ERROR, (event, data) => {
                if (data.fatal) {
                    showToast('Stream error. Attempting reconnect...');
                    if (data.type === Hls.ErrorTypes.NETWORK_ERROR) hls.startLoad();
                    else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) hls.recoverMediaError();
                    else destroyHls();
                }
            });
        } else if (audio.canPlayType('application/vnd.apple.mpegurl')) {
            audio.src = url;
        } else {
            showToast('HLS playback is not supported on this browser');
        }
    } else {
        audio.src = url;
    }
}

// User Profile population
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

        const drawerUserName = document.getElementById('drawer-user-name');
        const drawerUserHandle = document.getElementById('drawer-user-handle');
        if (drawerUserName) drawerUserName.innerText = name || 'Telegram User';
        if (drawerUserHandle) drawerUserHandle.innerText = handleText;

        if (u.photo_url) {
            if (userAvatarImg) { userAvatarImg.src = u.photo_url; userAvatarImg.style.display = 'block'; }
            if (userAvatarPlaceholder) userAvatarPlaceholder.style.display = 'none';
            if (overlayAvatarImg) { overlayAvatarImg.src = u.photo_url; overlayAvatarImg.style.display = 'block'; }
            if (overlayAvatarPlaceholder) overlayAvatarPlaceholder.style.display = 'none';
            const drawerAvatarImg = document.getElementById('drawer-avatar-img');
            const drawerAvatarPlaceholder = document.getElementById('drawer-avatar-placeholder');
            if (drawerAvatarImg) { drawerAvatarImg.src = u.photo_url; drawerAvatarImg.style.display = 'block'; }
            if (drawerAvatarPlaceholder) drawerAvatarPlaceholder.style.display = 'none';
        } else {
            if (userAvatarPlaceholder) userAvatarPlaceholder.innerText = initials;
            if (overlayAvatarPlaceholder) overlayAvatarPlaceholder.innerText = initials;
            const drawerAvatarPlaceholder = document.getElementById('drawer-avatar-placeholder');
            if (drawerAvatarPlaceholder) drawerAvatarPlaceholder.innerText = initials;
        }

        if (u.is_premium && userPremiumBadge) userPremiumBadge.style.display = 'flex';
    }
}
populateUserProfile();

// Drawer Helper Functions
function openDrawer(backdrop, drawer) {
    triggerHaptic('light');
    if (!backdrop || !drawer) return;
    backdrop.style.display = 'block';
    setTimeout(() => {
        backdrop.style.opacity = '1';
        drawer.style.transform = 'translateX(-50%) translateY(0)';
    }, 10);
    updateMiniPlayerVisibility();
}

function closeDrawer(backdrop, drawer) {
    triggerHaptic('light');
    if (!backdrop || !drawer) return;
    backdrop.style.opacity = '0';
    drawer.style.transform = 'translateX(-50%) translateY(100%)';
    setTimeout(() => {
        backdrop.style.display = 'none';
        updateMiniPlayerVisibility();
    }, 300);
}

function closeAllDrawers() {
    closeDrawer(queueBackdrop, queueDrawer);
    closeDrawer(searchBackdrop, searchDrawer);
    closeDrawer(relatedBackdrop, relatedDrawer);
    closeDrawer(listenersBackdrop, listenersDrawer);
    closeDrawer(profileBackdrop, profileDrawer);
    setActiveNavItem(navItemPlayer);
}

function updateMiniPlayerVisibility() {
    if (!miniPlayer) return;
    const isAnyDrawerOpen = (queueBackdrop && queueBackdrop.style.display === 'block') ||
        (searchBackdrop && searchBackdrop.style.display === 'block') ||
        (relatedBackdrop && relatedBackdrop.style.display === 'block') ||
        (listenersBackdrop && listenersBackdrop.style.display === 'block') ||
        (profileBackdrop && profileBackdrop.style.display === 'block');

    if (isAnyDrawerOpen && roomState && roomState.track) {
        miniPlayer.classList.remove('hidden');
    } else {
        miniPlayer.classList.add('hidden');
    }
}

// Nav items active state
function setActiveNavItem(activeBtn) {
    [navItemPlayer, navItemQueue, navItemSearch, navItemRelated, navItemProfile].forEach(btn => {
        if (btn) btn.classList.remove('active');
    });
    if (activeBtn) activeBtn.classList.add('active');
}

// Event Listeners for Drawers and Bottom Nav
if (navItemPlayer) navItemPlayer.addEventListener('click', () => { closeAllDrawers(); });
if (navItemQueue) navItemQueue.addEventListener('click', () => { closeAllDrawers(); setActiveNavItem(navItemQueue); openDrawer(queueBackdrop, queueDrawer); });
if (navItemSearch) navItemSearch.addEventListener('click', () => { closeAllDrawers(); setActiveNavItem(navItemSearch); openDrawer(searchBackdrop, searchDrawer); });
if (navItemRelated) navItemRelated.addEventListener('click', () => { closeAllDrawers(); setActiveNavItem(navItemRelated); openDrawer(relatedBackdrop, relatedDrawer); triggerFetchMix(); });
if (navItemProfile) navItemProfile.addEventListener('click', () => { closeAllDrawers(); setActiveNavItem(navItemProfile); openDrawer(profileBackdrop, profileDrawer); });

if (queueCloseBtn) queueCloseBtn.addEventListener('click', () => closeDrawer(queueBackdrop, queueDrawer));
if (queueBackdrop) queueBackdrop.addEventListener('click', () => closeDrawer(queueBackdrop, queueDrawer));
if (searchCloseBtn) searchCloseBtn.addEventListener('click', () => closeDrawer(searchBackdrop, searchDrawer));
if (searchBackdrop) searchBackdrop.addEventListener('click', () => closeDrawer(searchBackdrop, searchDrawer));
if (relatedCloseBtn) relatedCloseBtn.addEventListener('click', () => closeDrawer(relatedBackdrop, relatedDrawer));
if (relatedBackdrop) relatedBackdrop.addEventListener('click', () => closeDrawer(relatedBackdrop, relatedDrawer));
if (listenersCloseBtn) listenersCloseBtn.addEventListener('click', () => closeDrawer(listenersBackdrop, listenersDrawer));
if (listenersBackdrop) listenersBackdrop.addEventListener('click', () => closeDrawer(listenersBackdrop, listenersDrawer));
if (profileCloseBtn) profileCloseBtn.addEventListener('click', () => closeDrawer(profileBackdrop, profileDrawer));
if (profileBackdrop) profileBackdrop.addEventListener('click', () => closeDrawer(profileBackdrop, profileDrawer));
if (listenersTrigger) listenersTrigger.addEventListener('click', () => openDrawer(listenersBackdrop, listenersDrawer));
if (miniInfoClick) miniInfoClick.addEventListener('click', () => closeAllDrawers());
const headerUserProfile = document.getElementById('header-user-profile');
if (headerUserProfile) headerUserProfile.addEventListener('click', () => openDrawer(profileBackdrop, profileDrawer));
const profileListenersBtn = document.getElementById('profile-listeners-btn');
if (profileListenersBtn) {
    profileListenersBtn.addEventListener('click', () => {
        closeDrawer(profileBackdrop, profileDrawer);
        openDrawer(listenersBackdrop, listenersDrawer);
    });
}

if (btnProfilePlaylists) {
    btnProfilePlaylists.addEventListener('click', () => {
        showToast('Use Telegram command /myplaylists to manage playlists');
    });
}

if (btnAddToPlaylist) {
    btnAddToPlaylist.addEventListener('click', () => {
        triggerHaptic('medium');
        if (roomState && roomState.track) {
            showToast('Use /addtoplaylist in Telegram to save track');
        } else {
            showToast('No active track playing');
        }
    });
}

// Toggle Artwork Mode (Square vs Round with SVG Ring)
function applyArtworkMode() {
    if (isRoundArtMode) {
        if (artWrapper) artWrapper.classList.add('round');
        if (progressRingSvg) progressRingSvg.classList.add('visible');
        if (artModeText) artModeText.innerText = 'Round Ring';
    } else {
        if (artWrapper) artWrapper.classList.remove('round');
        if (progressRingSvg) progressRingSvg.classList.remove('visible');
        if (artModeText) artModeText.innerText = 'Square Art';
    }
}
if (btnToggleArtwork) {
    btnToggleArtwork.addEventListener('click', () => {
        triggerHaptic('light');
        isRoundArtMode = !isRoundArtMode;
        applyArtworkMode();
    });
}
applyArtworkMode();

// Update SVG Progress Ring & Slider Fill
function updateProgressRing(position, duration) {
    if (!duration || duration <= 0) {
        if (progressRingCircle) progressRingCircle.style.strokeDashoffset = RING_CIRCUMFERENCE;
        if (seekSlider) seekSlider.style.setProperty('--seek-fill', '0%');
        return;
    }
    const fraction = Math.min(Math.max(position / duration, 0), 1);
    if (progressRingCircle) {
        const offset = RING_CIRCUMFERENCE - (fraction * RING_CIRCUMFERENCE);
        progressRingCircle.style.strokeDashoffset = offset;
    }
    if (seekSlider) {
        seekSlider.style.setProperty('--seek-fill', (fraction * 100) + '%');
    }
}

// Update Volume Slider Fill & Dynamic Lucide Volume Icon
function updateVolumeIconsAndFill(valPercentage, isMuted) {
    if (volumeSlider) {
        volumeSlider.style.setProperty('--vol-fill', valPercentage + '%');
    }

    if (iconVolHigh) iconVolHigh.style.display = 'none';
    if (iconVolLow) iconVolLow.style.display = 'none';
    if (iconVolMin) iconVolMin.style.display = 'none';
    if (iconVolMute) iconVolMute.style.display = 'none';

    if (isMuted || valPercentage <= 0) {
        if (iconVolMute) iconVolMute.style.display = 'inline-block';
    } else if (valPercentage < 30) {
        if (iconVolMin) iconVolMin.style.display = 'inline-block';
    } else if (valPercentage < 70) {
        if (iconVolLow) iconVolLow.style.display = 'inline-block';
    } else {
        if (iconVolHigh) iconVolHigh.style.display = 'inline-block';
    }
    refreshIcons();
}

// Sleep Timer Handler
if (sleepTimerSelect) {
    sleepTimerSelect.addEventListener('change', () => {
        const mins = parseInt(sleepTimerSelect.value, 10);
        if (sleepTimerId) {
            clearInterval(sleepTimerId);
            sleepTimerId = null;
        }

        if (mins <= 0) {
            sleepEndTime = null;
            if (sleepTimerStatus) sleepTimerStatus.innerText = 'Off (Max 2h)';
            showToast('Sleep timer turned off');
        } else {
            sleepEndTime = Date.now() + (mins * 60 * 1000);
            showToast('Sleep timer set for ' + mins + ' min');
            updateSleepTimerUI();

            sleepTimerId = setInterval(() => {
                const remainingSecs = Math.round((sleepEndTime - Date.now()) / 1000);
                if (remainingSecs <= 0) {
                    clearInterval(sleepTimerId);
                    sleepTimerId = null;
                    sleepEndTime = null;
                    if (sleepTimerSelect) sleepTimerSelect.value = '0';
                    if (sleepTimerStatus) sleepTimerStatus.innerText = 'Off (Max 2h)';

                    if (canControl && ws && ws.readyState === WebSocket.OPEN) {
                        ws.send(JSON.stringify({ type: 'pause' }));
                    } else if (audio) {
                        audio.pause();
                    }
                    showToast('Sleep timer finished — playback paused');
                } else {
                    updateSleepTimerUI();
                }
            }, 1000);
        }
    });
}

function updateSleepTimerUI() {
    if (!sleepEndTime || !sleepTimerStatus) return;
    const remainingSecs = Math.max(0, Math.round((sleepEndTime - Date.now()) / 1000));
    const mins = Math.floor(remainingSecs / 60);
    const secs = remainingSecs % 60;
    sleepTimerStatus.innerText = 'Pausing in ' + mins + 'm ' + (secs < 10 ? '0' : '') + secs + 's';
}

// WebSocket Connection & Logic
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
        if (profileConnStatus) profileConnStatus.innerText = 'Connected';
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
                if (roleText) roleText.innerText = isAdmin ? 'Admin' : 'Listener';
                if (roleBadge) {
                    if (isAdmin) roleBadge.classList.add('admin');
                    else roleBadge.classList.remove('admin');
                }
                if (profileRoleStatus) profileRoleStatus.innerText = isAdmin ? 'Admin (Full Control)' : (canControl ? 'Controller' : 'Listener');
                updateControlButtonsState();
            } else if (msg.event === 'search_results') {
                renderSearchResults(msg.data.results || []);
            } else if (msg.event === 'recommendations_results') {
                renderRelatedResults(msg.data.results || []);
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
        if (profileConnStatus) profileConnStatus.innerText = 'Reconnecting...';
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
    if (btnPlay) btnPlay.disabled = !canControl;
    if (btnSkip) btnSkip.disabled = !canControl;
    if (btnStop) btnStop.disabled = !canControl;
    if (btnLoop) btnLoop.disabled = !canControl;
    if (btnAutoplay) btnAutoplay.disabled = !canControl;
    if (btnMixTrigger) btnMixTrigger.disabled = !canPlay && !canControl;
    if (seekSlider) seekSlider.disabled = !canControl;
    if (miniBtnPlay) miniBtnPlay.disabled = !canControl;
    if (miniBtnSkip) miniBtnSkip.disabled = !canControl;
    if (btnClearQueueDrawer) btnClearQueueDrawer.disabled = !canControl;
    if (btnDesktopClearQueue) btnDesktopClearQueue.disabled = !canControl;
}

function setPlayingState(isPlaying) {
    if (btnPlay) {
        if (isPlaying) btnPlay.classList.add('playing');
        else btnPlay.classList.remove('playing');
    }
    if (miniBtnPlay) {
        if (isPlaying) miniBtnPlay.classList.add('playing');
        else miniBtnPlay.classList.remove('playing');
    }
}

function updateRoomState(data) {
    roomState = data;
    if (profileRoomId) profileRoomId.innerText = data.roomId || roomId;

    const listenersCount = data.listeners ? data.listeners.length : 0;
    if (listenersCountText) listenersCountText.innerText = listenersCount + (listenersCount === 1 ? ' Listener' : ' Listeners');
    if (listenersDrawerCount) listenersDrawerCount.innerText = listenersCount;
    const profileListenersCount = document.getElementById('profile-listeners-count');
    if (profileListenersCount) profileListenersCount.innerText = listenersCount;

    updateListenersList(data.listeners || []);

    const track = data.track;
    const pb = data.playback;

    if (!track) {
        if (idleView) idleView.style.display = 'flex';
        if (activePlayerView) activePlayerView.style.display = 'none';
        if (ambientGlow) ambientGlow.style.opacity = '0.1';
        setPlayingState(false);
        stopAudioPlayback(true);
        if (currTime) currTime.innerText = '0:00';
        if (totalTime) totalTime.innerText = '0:00';
        if (seekSlider) {
            seekSlider.value = 0;
            seekSlider.style.setProperty('--seek-fill', '0%');
        }
        if (overlaySongTitle) overlaySongTitle.innerText = 'Nothing Playing';
        if (overlaySongArtist) overlaySongArtist.innerText = 'No active track in room';
        updateQueue([]);
        updateMiniPlayerVisibility();
        refreshIcons();
        return;
    }

    if (idleView) idleView.style.display = 'none';
    if (activePlayerView) activePlayerView.style.display = 'flex';

    // Loop state
    const loopCount = data.loop || 0;
    if (btnLoop) {
        if (loopCount > 0) {
            btnLoop.classList.add('active');
            if (loopCountBadge) { loopCountBadge.innerText = loopCount; loopCountBadge.style.display = 'flex'; }
        } else {
            btnLoop.classList.remove('active');
            if (loopCountBadge) loopCountBadge.style.display = 'none';
        }
    }

    // Autoplay active state
    if (btnAutoplay) {
        if (data.autoplay) btnAutoplay.classList.add('active');
        else btnAutoplay.classList.remove('active');
    }

    // Song info & Platform Badge format
    const songName = track.title || 'Unknown Track';
    const artistName = track.artist || track.platform || 'Music';
    const rawPlatform = (track.platform || 'YouTube').toUpperCase();
    const platformDisplay = rawPlatform === 'YOUTUBE' ? 'YouTube' : (track.platform || 'YouTube');
    const requesterDisplay = 'Requested by ' + (track.user || 'System');

    if (trackTitle) trackTitle.innerText = songName;
    if (trackArtist) trackArtist.innerText = artistName;
    if (miniTitle) miniTitle.innerText = songName;
    if (miniArtist) miniArtist.innerText = artistName;
    if (overlaySongTitle) overlaySongTitle.innerText = songName;
    if (overlaySongArtist) overlaySongArtist.innerText = artistName;
    if (requesterName) requesterName.innerText = requesterDisplay;
    if (platformBadge) platformBadge.innerText = platformDisplay;

    const thumbUrl = track.thumbnail || 'https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg';
    if (trackThumb) trackThumb.src = thumbUrl;
    if (miniThumb) miniThumb.src = thumbUrl;

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
        updateProgressRing(currentPosition, trackDuration);
    }

    if (pb.status === 'playing') {
        setPlayingState(true);
        if (artWrapper) artWrapper.classList.add('playing');
        if (ambientGlow) ambientGlow.style.opacity = '0.35';

        if (isAudioUnlocked) {
            if (srcChanged) {
                startAudioPlayback();
            } else {
                if (pendingSeekPosition === null && Math.abs(audio.currentTime - targetPos) > 1.2) {
                    pendingSeekPosition = targetPos;
                    if (audio.readyState >= 1) {
                        try { audio.currentTime = targetPos; pendingSeekPosition = null; } catch(e) {}
                    }
                }
                if (audio.paused) startAudioPlayback();
            }
        }
    } else {
        setPlayingState(false);
        if (artWrapper) artWrapper.classList.remove('playing');
        if (ambientGlow) ambientGlow.style.opacity = '0.1';
        stopAudioPlayback(false);
    }

    updateQueue(data.queue || []);
    updateMiniPlayerVisibility();
    refreshIcons();
}

// Queue Rendering
function updateQueue(queue) {
    const countText = (queue ? queue.length : 0);
    if (drawerQueueCount) drawerQueueCount.innerText = countText;
    const navQueueCount = document.getElementById('nav-queue-count');
    if (navQueueCount) navQueueCount.innerText = countText;

    const showClear = queue && queue.length > 0 && canControl;
    if (btnClearQueueDrawer) btnClearQueueDrawer.style.display = showClear ? 'inline-block' : 'none';
    if (btnDesktopClearQueue) btnDesktopClearQueue.style.display = showClear ? 'inline-block' : 'none';

    let html = '';
    if (!queue || queue.length === 0) {
        html = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">Your queue is empty</div>';
    } else {
        queue.forEach((item, index) => {
            const thumb = item.thumbnail || (trackThumb ? trackThumb.src : '');
            html += '<div class="song-row">';
            html += '<img class="song-thumb" src="' + thumb + '" alt="thumb">';
            html += '<div class="song-info">';
            html += '<div class="song-title">' + (index + 1) + '. ' + (item.title || 'Track') + '</div>';
            html += '<div class="song-sub">' + formatTime(item.duration) + ' • Requested by ' + (item.user || 'User') + '</div>';
            html += '</div>';
            if (canControl) {
                html += '<button class="btn-action danger" onclick="removeQueueTrack(' + (index + 1) + ')">Remove</button>';
            }
            html += '</div>';
        });
    }

    if (queueScrollList) queueScrollList.innerHTML = html;
    if (desktopQueueList) desktopQueueList.innerHTML = html;
}

window.removeQueueTrack = function(index) {
    if (!canControl) return;
    triggerHaptic('medium');
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'remove', index: index }));
        showToast('Removed track from queue');
    }
};

function clearQueue() {
    if (!canControl) return;
    triggerHaptic('medium');
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'clear_queue' }));
        showToast('Queue cleared');
    }
}
if (btnClearQueueDrawer) btnClearQueueDrawer.addEventListener('click', clearQueue);
if (btnDesktopClearQueue) btnDesktopClearQueue.addEventListener('click', clearQueue);

// Listeners List Rendering
function updateListenersList(listeners) {
    let html = '';
    if (!listeners || listeners.length === 0) {
        html = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">No active listeners connected</div>';
    } else {
        listeners.forEach(l => {
            const fullName = (l.firstName || 'Listener') + (l.lastName ? ' ' + l.lastName : '');
            const handle = l.username ? '@' + l.username : (l.userId ? 'ID: ' + l.userId : 'Anonymous');
            const initial = (l.firstName ? l.firstName[0] : 'L').toUpperCase();

            html += '<div class="song-row">';
            if (l.photoUrl) {
                html += '<img class="song-thumb" style="border-radius:50%;" src="' + l.photoUrl + '" alt="avatar">';
            } else {
                html += '<div class="user-avatar" style="width:48px; height:48px; font-size:18px;">' + initial + '</div>';
            }
            html += '<div class="song-info">';
            html += '<div class="song-title">' + fullName + '</div>';
            html += '<div class="song-sub">' + handle + '</div>';
            html += '</div>';
            if (l.isAdmin) {
                html += '<div class="badge admin"><span class="badge-dot"></span><span>Admin</span></div>';
            } else {
                html += '<div class="badge"><span id="role-text">Listener</span></div>';
            }
            html += '</div>';
        });
    }
    if (listenersScrollList) listenersScrollList.innerHTML = html;
}

// Search Actions & Rendering
function performSearch(queryInput, container) {
    triggerHaptic('light');
    if (!queryInput) return;
    const query = queryInput.value.trim();
    if (!query) {
        showToast('Please enter a song name or YouTube link');
        return;
    }
    if (!canPlay && !canControl) {
        showToast('Play mode is restricted in this chat');
        return;
    }

    if (container) {
        container.innerHTML = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">Searching music...</div>';
    }

    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'search', query: query }));
    }
}

function bindSearchInputEvents(input, clearBtn) {
    if (!input || !clearBtn) return;
    input.addEventListener('input', () => {
        clearBtn.style.display = input.value.trim().length > 0 ? 'flex' : 'none';
    });
    clearBtn.addEventListener('click', () => {
        input.value = '';
        clearBtn.style.display = 'none';
        input.focus();
    });
}
bindSearchInputEvents(modalSearchInput, btnModalSearchClear);
bindSearchInputEvents(desktopSearchInput, btnDesktopSearchClear);

if (btnModalSearchSubmit) btnModalSearchSubmit.addEventListener('click', () => performSearch(modalSearchInput, modalSearchResults));
if (modalSearchInput) modalSearchInput.addEventListener('keypress', (e) => { if (e.key === 'Enter') performSearch(modalSearchInput, modalSearchResults); });
if (btnDesktopSearch) btnDesktopSearch.addEventListener('click', () => performSearch(desktopSearchInput, desktopSearchResults));
if (desktopSearchInput) desktopSearchInput.addEventListener('keypress', (e) => { if (e.key === 'Enter') performSearch(desktopSearchInput, desktopSearchResults); });

function renderSearchResults(results) {
    window._searchResults = results;
    let html = '';
    if (!results || results.length === 0) {
        html = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">No results found.</div>';
    } else {
        results.forEach((item, index) => {
            const thumb = item.thumbnail || 'https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg';
            const title = item.title || 'Unknown Track';
            const artist = item.artist || item.platform || 'Music';
            const dur = formatTime(item.duration);

            html += '<div class="song-row">';
            html += '<img class="song-thumb" src="' + thumb + '" alt="thumb">';
            html += '<div class="song-info">';
            html += '<div class="song-title">' + title + '</div>';
            html += '<div class="song-sub">' + artist + ' • ' + dur + '</div>';
            html += '</div>';
            html += '<div class="song-actions">';
            if (canControl) {
                html += '<button class="btn-action primary" onclick="handleSearchAction(' + index + ', true)">Play</button>';
            }
            html += '<button class="btn-action" onclick="handleSearchAction(' + index + ', false)">+ Queue</button>';
            html += '</div></div>';
        });
    }

    if (modalSearchResults) modalSearchResults.innerHTML = html;
    if (desktopSearchResults) desktopSearchResults.innerHTML = html;
}

window.handleSearchAction = function(index, force) {
    if (!window._searchResults || !window._searchResults[index]) return;
    requestTrack(window._searchResults[index], force);
};

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
    closeDrawer(searchBackdrop, searchDrawer);
}

// Related Mix / Recommendations Logic
function triggerFetchMix() {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    triggerHaptic('light');
    if (relatedScrollList) {
        relatedScrollList.innerHTML = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">Fetching recommendations...</div>';
    }
    if (desktopRelatedResults) {
        desktopRelatedResults.innerHTML = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">Fetching recommendations...</div>';
    }

    const currentTrack = (roomState && roomState.track) ? roomState.track : null;
    ws.send(JSON.stringify({
        type: 'mix',
        track: currentTrack,
        count: 10
    }));
}

if (btnFetchMix) btnFetchMix.addEventListener('click', triggerFetchMix);
if (btnDesktopGetMix) btnDesktopGetMix.addEventListener('click', triggerFetchMix);
if (btnMixTrigger) btnMixTrigger.addEventListener('click', () => { openDrawer(relatedBackdrop, relatedDrawer); triggerFetchMix(); });

function renderRelatedResults(results) {
    window._relatedResults = results;
    let html = '';
    if (!results || results.length === 0) {
        html = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">No recommendations available right now.</div>';
    } else {
        results.forEach((item, index) => {
            const thumb = item.thumbnail || 'https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg';
            const title = item.title || 'Unknown Track';
            const artist = item.artist || 'Music';
            const dur = formatTime(item.duration);

            html += '<div class="song-row">';
            html += '<img class="song-thumb" src="' + thumb + '" alt="thumb">';
            html += '<div class="song-info">';
            html += '<div class="song-title">' + title + '</div>';
            html += '<div class="song-sub">' + artist + ' • ' + dur + '</div>';
            html += '</div>';
            html += '<div class="song-actions">';
            if (canControl) {
                html += '<button class="btn-action primary" onclick="handleRelatedAction(' + index + ', true)">Play</button>';
            }
            html += '<button class="btn-action" onclick="handleRelatedAction(' + index + ', false)">+ Queue</button>';
            html += '</div></div>';
        });
    }

    if (relatedScrollList) relatedScrollList.innerHTML = html;
    if (desktopRelatedResults) desktopRelatedResults.innerHTML = html;
}

window.handleRelatedAction = function(index, force) {
    if (!window._relatedResults || !window._relatedResults[index]) return;
    requestTrack(window._relatedResults[index], force);
};

// Playback Control Triggers
function togglePlayPause() {
    triggerHaptic('light');
    if (!canControl || !roomState || !roomState.track) return;
    if (roomState.playback && roomState.playback.status === 'playing') {
        ws.send(JSON.stringify({ type: 'pause' }));
    } else {
        ws.send(JSON.stringify({ type: 'resume' }));
    }
}
if (btnPlay) btnPlay.addEventListener('click', togglePlayPause);
if (miniBtnPlay) miniBtnPlay.addEventListener('click', togglePlayPause);

function skipTrack() {
    triggerHaptic('medium');
    if (!canControl) return;
    ws.send(JSON.stringify({ type: 'skip' }));
}
if (btnSkip) btnSkip.addEventListener('click', skipTrack);
if (miniBtnSkip) miniBtnSkip.addEventListener('click', skipTrack);

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
        const nextLoop = currentLoop > 0 ? 0 : 1;
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

// Volume Controls
if (volumeSlider) {
    const savedVol = localStorage.getItem('tg_player_volume');
    if (savedVol !== null) {
        const v = parseFloat(savedVol);
        audio.volume = v;
        const percent = Math.round(v * 100);
        volumeSlider.value = percent;
        updateVolumeIconsAndFill(percent, audio.muted);
    } else {
        updateVolumeIconsAndFill(100, false);
    }
    volumeSlider.addEventListener('input', () => {
        const percent = parseFloat(volumeSlider.value);
        const val = percent / 100;
        audio.volume = val;
        localStorage.setItem('tg_player_volume', val);
        audio.muted = (val === 0);
        updateVolumeIconsAndFill(percent, audio.muted);
    });
}

if (btnMute) {
    btnMute.addEventListener('click', () => {
        triggerHaptic('light');
        audio.muted = !audio.muted;
        if (audio.muted) {
            updateVolumeIconsAndFill(parseFloat(volumeSlider ? volumeSlider.value : 0), true);
        } else {
            if (audio.volume === 0) {
                audio.volume = 1;
                if (volumeSlider) volumeSlider.value = 100;
                localStorage.setItem('tg_player_volume', 1);
            }
            updateVolumeIconsAndFill(parseFloat(volumeSlider ? volumeSlider.value : 100), false);
        }
    });
}

// Seek Slider
if (seekSlider) {
    seekSlider.addEventListener('input', () => {
        if (!canControl) return;
        isUserSeeking = true;
        currTime.innerText = formatTime(seekSlider.value);
        updateProgressRing(seekSlider.value, trackDuration);
    });

    seekSlider.addEventListener('change', () => {
        if (!canControl) return;
        triggerHaptic('light');
        isUserSeeking = false;
        const pos = parseFloat(seekSlider.value);
        currentPosition = pos;
        pendingSeekPosition = pos;
        if (audio.readyState >= 1) {
            try { audio.currentTime = pos; } catch(e) {}
        }
        ws.send(JSON.stringify({ type: 'seek', positionSeconds: pos }));
    });
}

// Join Stream Overlay Button
if (btnJoin) {
    btnJoin.addEventListener('click', () => {
        triggerHaptic('medium');
        isAudioUnlocked = true;
        if (joinOverlay) {
            joinOverlay.style.opacity = '0';
            joinOverlay.style.visibility = 'hidden';
            setTimeout(() => { joinOverlay.style.display = 'none'; }, 300);
        }

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

        if (roomState) updateRoomState(roomState);
    });
}

// Audio Ended Event
audio.addEventListener('ended', () => {
    if (!currentAudioUrl || (audio.src && audio.src.startsWith('data:audio/'))) return;
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'track_end' }));
    }
});

// Periodic Progress Interpolation
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
        if (seekSlider) seekSlider.value = currentPosition;
        if (currTime) currTime.innerText = formatTime(currentPosition);
        updateProgressRing(currentPosition, trackDuration);
    }
}, 250);

connectWS();
