const tg = window.Telegram ? window.Telegram.WebApp : null;
const noTgOverlay = document.getElementById('no-tg-overlay');
const btnTgOpen = document.getElementById('btn-tg-open');

let isMiniAppEnv = !!(tg && tg.initData && tg.initData.trim() !== '');
let currentWebUser = null;
let userAllowsWriteToPM = false;

if (btnTgOpen) {
    btnTgOpen.addEventListener('click', () => {
        if (window.Telegram && window.Telegram.WebApp && typeof window.Telegram.WebApp.close === 'function' && tg && tg.initData) {
            try {
                window.Telegram.WebApp.close();
            } catch (e) {
                window.location.reload();
            }
        } else {
            window.location.reload();
        }
    });
}

if (isMiniAppEnv) {
    if (noTgOverlay) noTgOverlay.style.display = 'none';
    tg.expand();

    const currentPlatform = (tg.platform || (new URLSearchParams(window.location.search)).get('tgWebAppPlatform') || '').toLowerCase();
    const isDesktopPlatform = currentPlatform.includes('desktop') || currentPlatform.includes('macos') || currentPlatform === 'weba' || currentPlatform === 'webk';

    if (!isDesktopPlatform && typeof tg.requestFullscreen === 'function' && !tg.isFullscreen) {
        try {
            tg.requestFullscreen();
        } catch (e) {
            console.log('Telegram requestFullscreen error:', e);
        }
    }

    if (tg.setHeaderColor) {
        try {
            tg.setHeaderColor('#060811');
        } catch (e) {}
    }
    if (tg.setBackgroundColor) {
        try {
            tg.setBackgroundColor('#060811');
        } catch (e) {}
    }
    if (tg.disableClosingConfirmation) {
        try {
            tg.disableClosingConfirmation();
        } catch (e) {}
    }
    tg.ready();
} else {
    // Normal website environment
    if (noTgOverlay) noTgOverlay.style.display = 'flex';
    const joinOv = document.getElementById('join-overlay');
    if (joinOv) joinOv.style.display = 'none';
}

function updateTelegramSafeArea() {
    if (!tg) return;
    const root = document.documentElement;

    const safeTop = (tg.safeAreaInset && typeof tg.safeAreaInset.top === 'number') ? tg.safeAreaInset.top : 0;
    const safeBottom = (tg.safeAreaInset && typeof tg.safeAreaInset.bottom === 'number') ? tg.safeAreaInset.bottom : 0;
    const safeLeft = (tg.safeAreaInset && typeof tg.safeAreaInset.left === 'number') ? tg.safeAreaInset.left : 0;
    const safeRight = (tg.safeAreaInset && typeof tg.safeAreaInset.right === 'number') ? tg.safeAreaInset.right : 0;

    const contentSafeTop = (tg.contentSafeAreaInset && typeof tg.contentSafeAreaInset.top === 'number') ? tg.contentSafeAreaInset.top : safeTop;
    const contentSafeBottom = (tg.contentSafeAreaInset && typeof tg.contentSafeAreaInset.bottom === 'number') ? tg.contentSafeAreaInset.bottom : safeBottom;

    root.style.setProperty('--tg-safe-top', safeTop + 'px');
    root.style.setProperty('--tg-safe-bottom', safeBottom + 'px');
    root.style.setProperty('--tg-safe-left', safeLeft + 'px');
    root.style.setProperty('--tg-safe-right', safeRight + 'px');
    root.style.setProperty('--tg-content-safe-top', contentSafeTop + 'px');
    root.style.setProperty('--tg-content-safe-bottom', contentSafeBottom + 'px');
}

updateTelegramSafeArea();

if (tg && tg.onEvent) {
    try {
        tg.onEvent('safeAreaChanged', updateTelegramSafeArea);
        tg.onEvent('contentSafeAreaChanged', updateTelegramSafeArea);
        tg.onEvent('fullscreenChanged', updateTelegramSafeArea);
        tg.onEvent('fullscreenFailed', (err) => {
            console.log('Fullscreen failed:', err);
            updateTelegramSafeArea();
        });
        tg.onEvent('viewportChanged', updateTelegramSafeArea);
    } catch (e) {
        console.error('Error attaching Telegram safe area listeners:', e);
    }
}
window.addEventListener('resize', updateTelegramSafeArea);

const urlParams = new URLSearchParams(window.location.search);
let startParam = (tg && tg.initDataUnsafe && tg.initDataUnsafe.start_param) ? String(tg.initDataUnsafe.start_param).trim() : null;
if (!startParam) {
    const rawVal = urlParams.get('tgWebAppStartParam') || urlParams.get('startapp') || urlParams.get('chat_id') || urlParams.get('room');
    if (rawVal) {
        startParam = String(rawVal).trim();
    }
}
let roomId = startParam || null;
if (!roomId) {
    if (isMiniAppEnv && tg && tg.initDataUnsafe && tg.initDataUnsafe.user && tg.initDataUnsafe.user.id) {
        roomId = String(tg.initDataUnsafe.user.id);
    } else if (currentWebUser && currentWebUser.id) {
        roomId = String(currentWebUser.id);
    } else {
        roomId = '0';
    }
}

if (window.history && window.history.replaceState) {
    try {
        window.history.replaceState({}, document.title, window.location.pathname);
    } catch (e) {
        console.error('Failed to clean URL:', e);
    }
}

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
const btnPrev = document.getElementById('btn-prev');
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
const navAvatarPlaceholder = document.getElementById('nav-avatar-placeholder');
const navAvatarImg = document.getElementById('nav-avatar-img');
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
const sleepTimerContainer = document.getElementById('sleep-timer-container');
const sleepTimerTrigger = document.getElementById('sleep-timer-trigger');
const sleepTimerDropdown = document.getElementById('sleep-timer-dropdown');
const sleepTimerSelectedText = document.getElementById('sleep-timer-selected-text');
const btnProfilePlaylists = document.getElementById('btn-profile-playlists');
const btnPlayerSleepTimer = document.getElementById('btn-player-sleep-timer');
const playerSleepTimerBadge = document.getElementById('player-sleep-timer-badge');
const sleepTimerBackdrop = document.getElementById('sleep-timer-backdrop');
const sleepTimerDrawer = document.getElementById('sleep-timer-drawer');
const sleepTimerCloseBtn = document.getElementById('sleep-timer-close-btn');
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
const listenersSessionActions = document.getElementById('listeners-session-actions');
const btnEndSession = document.getElementById('btn-end-session');
const endSessionConfirmOverlay = document.getElementById('end-session-confirm-overlay');
const btnConfirmEndCancel = document.getElementById('btn-confirm-end-cancel');
const btnConfirmEndSubmit = document.getElementById('btn-confirm-end-submit');

const playlistBackdrop = document.getElementById('playlist-backdrop');
const playlistDrawer = document.getElementById('playlist-drawer');
const playlistCloseBtn = document.getElementById('playlist-close-btn');
let playlistScrollList = document.getElementById('playlist-scroll-list');
const btnCreatePlaylistTrigger = document.getElementById('btn-create-playlist-trigger');

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
var isRoundArtMode = true; // Default to round artwork with progress ring

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

let toastTimeoutId = null;

function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function showToast(msg, type = null, duration = 1600) {
    if (!toastMsg || !msg) return;

    if (!type) {
        const lower = String(msg).toLowerCase();
        if (lower.includes('error') || lower.includes('failed') || lower.includes('unauthorized') || lower.includes('lost') || lower.includes('restricted')) {
            type = 'error';
        } else if (lower.includes('warning') || lower.includes('interrupted') || lower.includes('expired')) {
            type = 'warning';
        } else if (lower.includes('added') || lower.includes('created') || lower.includes('playing') || lower.includes('set') || lower.includes('cleared') || lower.includes('removed') || lower.includes('renamed') || lower.includes('finished') || lower.includes('success')) {
            type = 'success';
        } else {
            type = 'info';
        }
    }

    let iconSvg = '';
    if (type === 'success') {
        iconSvg = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>';
    } else if (type === 'error') {
        iconSvg = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>';
    } else if (type === 'warning') {
        iconSvg = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>';
    } else {
        iconSvg = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>';
    }

    if (toastTimeoutId) {
        clearTimeout(toastTimeoutId);
        toastTimeoutId = null;
    }

    toastMsg.innerHTML = `<div class="toast-card toast-${type}"><div class="toast-icon">${iconSvg}</div><span class="toast-text">${escapeHtml(msg)}</span></div>`;
    toastMsg.style.display = 'block';

    toastTimeoutId = setTimeout(() => {
        const card = toastMsg.querySelector('.toast-card');
        if (card) {
            card.classList.add('toast-exit');
        }
        setTimeout(() => {
            toastMsg.style.display = 'none';
            toastMsg.innerHTML = '';
        }, 220);
    }, duration);
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

    if (url && url.startsWith('/stream?') && tg && tg.initData) {
        if (!url.includes('init_data=')) {
            url += '&init_data=' + encodeURIComponent(tg.initData);
        }
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

// User Profile population & Telegram Messaging Permission Handler
function populateUserProfile(userObj) {
    const u = userObj || currentWebUser || (tg && tg.initDataUnsafe ? tg.initDataUnsafe.user : null);
    const writePermWarning = document.getElementById('write-perm-warning');

    if (!u) {
        if (writePermWarning) writePermWarning.style.display = 'none';
        return;
    }

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

    const photoUrl = u.photo_url || u.photoUrl || u.picture;
    if (photoUrl) {
        if (userAvatarImg) { userAvatarImg.src = photoUrl; userAvatarImg.style.display = 'block'; }
        if (userAvatarPlaceholder) userAvatarPlaceholder.style.display = 'none';
        if (navAvatarImg) { navAvatarImg.src = photoUrl; navAvatarImg.style.display = 'block'; }
        if (navAvatarPlaceholder) navAvatarPlaceholder.style.display = 'none';
        if (overlayAvatarImg) { overlayAvatarImg.src = photoUrl; overlayAvatarImg.style.display = 'block'; }
        if (overlayAvatarPlaceholder) overlayAvatarPlaceholder.style.display = 'none';
        const drawerAvatarImg = document.getElementById('drawer-avatar-img');
        const drawerAvatarPlaceholder = document.getElementById('drawer-avatar-placeholder');
        if (drawerAvatarImg) { drawerAvatarImg.src = photoUrl; drawerAvatarImg.style.display = 'block'; }
        if (drawerAvatarPlaceholder) drawerAvatarPlaceholder.style.display = 'none';
    } else {
        if (userAvatarPlaceholder) userAvatarPlaceholder.innerText = initials;
        if (navAvatarPlaceholder) navAvatarPlaceholder.innerText = initials;
        if (overlayAvatarPlaceholder) overlayAvatarPlaceholder.innerText = initials;
        const drawerAvatarPlaceholder = document.getElementById('drawer-avatar-placeholder');
        if (drawerAvatarPlaceholder) drawerAvatarPlaceholder.innerText = initials;
    }

    if (u.is_premium && userPremiumBadge) userPremiumBadge.style.display = 'flex';

    if (u.allows_write_to_pm || userAllowsWriteToPM) {
        if (writePermWarning) writePermWarning.style.display = 'none';
    } else {
        if (writePermWarning) writePermWarning.style.display = 'block';
    }
}

function requestWriteAccessPermission() {
    if (tg && typeof tg.requestWriteAccess === 'function') {
        tg.requestWriteAccess((granted) => {
            if (granted) {
                userAllowsWriteToPM = true;
                if (currentWebUser) currentWebUser.allows_write_to_pm = true;
                populateUserProfile(currentWebUser);
                if (ws && ws.readyState === WebSocket.OPEN) {
                    ws.send(JSON.stringify({ type: 'write_access_granted' }));
                }
                showToast('Telegram messaging permission granted!', 'success');
            } else {
                showToast('Permission cancelled. Playback remains blocked.', 'warning');
            }
        });
    } else {
        showToast('Please open the app inside Telegram to grant messaging permission.', 'info');
    }
}

const btnRequestWritePerm = document.getElementById('btn-request-write-perm');
if (btnRequestWritePerm) btnRequestWritePerm.addEventListener('click', requestWriteAccessPermission);

populateUserProfile();

// Active drawers tracking for body scroll locking
const activeDrawersSet = new Set();

function lockBodyScroll() {
    document.body.style.overflow = 'hidden';
}

function unlockBodyScroll() {
    if (activeDrawersSet.size === 0) {
        document.body.style.overflow = '';
    }
}

// Drawer Helper Functions
function syncNavState() {
    if (activeDrawersSet.has(queueDrawer)) {
        setActiveNavItem(navItemQueue);
    } else if (activeDrawersSet.has(searchDrawer)) {
        setActiveNavItem(navItemSearch);
    } else if (activeDrawersSet.has(relatedDrawer)) {
        setActiveNavItem(navItemRelated);
    } else if (activeDrawersSet.has(profileDrawer) || activeDrawersSet.has(playlistDrawer)) {
        setActiveNavItem(navItemProfile);
    } else {
        setActiveNavItem(navItemPlayer);
    }
}

function openDrawer(backdrop, drawer) {
    triggerHaptic('light');
    if (!backdrop || !drawer) return;
    activeDrawersSet.add(drawer);
    syncNavState();
    lockBodyScroll();

    drawer.style.transition = 'transform .32s cubic-bezier(.22,.8,.22,1)';
    backdrop.style.display = 'block';
    setTimeout(() => {
        backdrop.style.opacity = '1';
        drawer.style.transform = 'translateY(0)';
    }, 10);
    updateMiniPlayerVisibility();
}

function closeDrawer(backdrop, drawer) {
    triggerHaptic('light');
    if (!backdrop || !drawer) return;
    activeDrawersSet.delete(drawer);
    syncNavState();

    drawer.style.transition = 'transform .32s cubic-bezier(.22,.8,.22,1)';
    backdrop.style.opacity = '0';
    drawer.style.transform = 'translateY(100%)';
    setTimeout(() => {
        backdrop.style.display = 'none';
        unlockBodyScroll();
        updateMiniPlayerVisibility();
    }, 300);
}

function closeAllDrawers() {
    closeDrawer(queueBackdrop, queueDrawer);
    closeDrawer(searchBackdrop, searchDrawer);
    closeDrawer(relatedBackdrop, relatedDrawer);
    closeDrawer(listenersBackdrop, listenersDrawer);
    closeDrawer(playlistBackdrop, playlistDrawer);
    closeDrawer(profileBackdrop, profileDrawer);
    closeDrawer(sleepTimerBackdrop, sleepTimerDrawer);
    syncNavState();
}

function setupSwipeToDismiss(drawer, backdrop) {
    if (!drawer || !backdrop) return;
    let startY = 0;
    let isDragging = false;
    let currentDeltaY = 0;

    drawer.addEventListener('touchstart', (e) => {
        if (e.touches.length !== 1) return;
        const touch = e.touches[0];
        const contentElem = drawer.querySelector('.modal-content');
        const isHeader = e.target.closest('.modal-header') || e.target.closest('.drawer-handle');
        const isAtTop = contentElem ? contentElem.scrollTop <= 0 : true;

        if (isHeader || isAtTop) {
            startY = touch.clientY;
            isDragging = false;
            currentDeltaY = 0;
        } else {
            startY = 0;
        }
    }, { passive: true });

    drawer.addEventListener('touchmove', (e) => {
        if (!startY) return;
        const touch = e.touches[0];
        const deltaY = touch.clientY - startY;
        const contentElem = drawer.querySelector('.modal-content');
        const isHeader = e.target.closest('.modal-header') || e.target.closest('.drawer-handle');
        const isAtTop = contentElem ? contentElem.scrollTop <= 0 : true;

        if (deltaY > 0 && (isHeader || isAtTop)) {
            isDragging = true;
            currentDeltaY = deltaY;
            drawer.style.transition = 'none';
            drawer.style.transform = 'translateY(' + deltaY + 'px)';
        }
    }, { passive: true });

    drawer.addEventListener('touchend', () => {
        if (!startY) return;
        if (isDragging) {
            if (currentDeltaY > 80) {
                closeDrawer(backdrop, drawer);
            } else {
                drawer.style.transition = 'transform .32s cubic-bezier(.22,.8,.22,1)';
                drawer.style.transform = 'translateY(0)';
            }
        }
        startY = 0;
        isDragging = false;
        currentDeltaY = 0;
    }, { passive: true });
}

setupSwipeToDismiss(queueDrawer, queueBackdrop);
setupSwipeToDismiss(searchDrawer, searchBackdrop);
setupSwipeToDismiss(relatedDrawer, relatedBackdrop);
setupSwipeToDismiss(listenersDrawer, listenersBackdrop);
setupSwipeToDismiss(playlistDrawer, playlistBackdrop);
setupSwipeToDismiss(profileDrawer, profileBackdrop);
setupSwipeToDismiss(sleepTimerDrawer, sleepTimerBackdrop);

function updateMiniPlayerVisibility() {
    if (!miniPlayer) return;
    const isAnyDrawerOpen = (queueBackdrop && queueBackdrop.style.display === 'block') ||
        (searchBackdrop && searchBackdrop.style.display === 'block') ||
        (relatedBackdrop && relatedBackdrop.style.display === 'block') ||
        (listenersBackdrop && listenersBackdrop.style.display === 'block') ||
        (playlistBackdrop && playlistBackdrop.style.display === 'block') ||
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
if (navItemQueue) navItemQueue.addEventListener('click', () => { closeAllDrawers(); openDrawer(queueBackdrop, queueDrawer); });
if (navItemSearch) navItemSearch.addEventListener('click', () => { closeAllDrawers(); openDrawer(searchBackdrop, searchDrawer); });
if (navItemRelated) navItemRelated.addEventListener('click', () => { closeAllDrawers(); openDrawer(relatedBackdrop, relatedDrawer); triggerFetchMix(); });
if (navItemProfile) navItemProfile.addEventListener('click', () => { closeAllDrawers(); openDrawer(profileBackdrop, profileDrawer); });

syncNavState();

if (queueCloseBtn) queueCloseBtn.addEventListener('click', () => closeDrawer(queueBackdrop, queueDrawer));
if (queueBackdrop) queueBackdrop.addEventListener('click', () => closeDrawer(queueBackdrop, queueDrawer));
if (searchCloseBtn) searchCloseBtn.addEventListener('click', () => closeDrawer(searchBackdrop, searchDrawer));
if (searchBackdrop) searchBackdrop.addEventListener('click', () => closeDrawer(searchBackdrop, searchDrawer));
if (relatedCloseBtn) relatedCloseBtn.addEventListener('click', () => closeDrawer(relatedBackdrop, relatedDrawer));
if (relatedBackdrop) relatedBackdrop.addEventListener('click', () => closeDrawer(relatedBackdrop, relatedDrawer));
if (listenersCloseBtn) listenersCloseBtn.addEventListener('click', () => closeDrawer(listenersBackdrop, listenersDrawer));
if (listenersBackdrop) listenersBackdrop.addEventListener('click', () => closeDrawer(listenersBackdrop, listenersDrawer));
if (playlistCloseBtn) playlistCloseBtn.addEventListener('click', () => closeDrawer(playlistBackdrop, playlistDrawer));
if (playlistBackdrop) playlistBackdrop.addEventListener('click', () => closeDrawer(playlistBackdrop, playlistDrawer));
if (profileCloseBtn) profileCloseBtn.addEventListener('click', () => closeDrawer(profileBackdrop, profileDrawer));
if (profileBackdrop) profileBackdrop.addEventListener('click', () => closeDrawer(profileBackdrop, profileDrawer));
if (sleepTimerCloseBtn) sleepTimerCloseBtn.addEventListener('click', () => closeDrawer(sleepTimerBackdrop, sleepTimerDrawer));
if (sleepTimerBackdrop) sleepTimerBackdrop.addEventListener('click', () => closeDrawer(sleepTimerBackdrop, sleepTimerDrawer));
if (listenersTrigger) listenersTrigger.addEventListener('click', () => openDrawer(listenersBackdrop, listenersDrawer));
if (miniInfoClick) miniInfoClick.addEventListener('click', () => closeAllDrawers());
const headerUserProfile = document.getElementById('header-user-profile');
if (headerUserProfile) headerUserProfile.addEventListener('click', () => openDrawer(profileBackdrop, profileDrawer));
const headerMobileTitle = document.getElementById('header-mobile-title');
if (headerMobileTitle) headerMobileTitle.addEventListener('click', () => openDrawer(profileBackdrop, profileDrawer));
const profileListenersBtn = document.getElementById('profile-listeners-btn');
if (profileListenersBtn) {
    profileListenersBtn.addEventListener('click', () => {
        closeDrawer(profileBackdrop, profileDrawer);
        openDrawer(listenersBackdrop, listenersDrawer);
    });
}

if (btnProfilePlaylists) {
    btnProfilePlaylists.addEventListener('click', () => {
        closeDrawer(profileBackdrop, profileDrawer);
        openDrawer(playlistBackdrop, playlistDrawer);
        fetchPlaylists();
    });
}

if (btnCreatePlaylistTrigger) {
    btnCreatePlaylistTrigger.addEventListener('click', () => {
        const name = prompt('Enter a name for the new playlist:');
        if (name && name.trim()) {
            createPlaylist(name.trim());
        }
    });
}

let isPlaylistActionPending = false;

function updateAddToPlaylistButtonState() {
    if (!btnAddToPlaylist) return;
    if (!roomState || !roomState.track) {
        btnAddToPlaylist.classList.remove('in-playlist');
        btnAddToPlaylist.title = "Add to playlist";
        return;
    }

    const trackId = roomState.track.trackId || roomState.track.id;
    let isInAnyPlaylist = false;

    if (window._userPlaylists && Array.isArray(window._userPlaylists)) {
        isInAnyPlaylist = window._userPlaylists.some(pl =>
            pl.songs && pl.songs.some(s => s.track_id === trackId || s.url === roomState.track.url)
        );
    }

    if (isInAnyPlaylist) {
        btnAddToPlaylist.classList.add('in-playlist');
        btnAddToPlaylist.title = "In playlist (click to remove)";
        btnAddToPlaylist.innerHTML = '<i data-lucide="heart-off"></i>';
    } else {
        btnAddToPlaylist.classList.remove('in-playlist');
        btnAddToPlaylist.title = "Add to playlist";
        btnAddToPlaylist.innerHTML = '<i data-lucide="heart-plus"></i>';
    }
    refreshIcons();
}

if (btnAddToPlaylist) {
    btnAddToPlaylist.addEventListener('click', () => {
        if (isPlaylistActionPending) return;
        triggerHaptic('medium');

        if (!roomState || !roomState.track) {
            showToast('No active track playing');
            return;
        }

        const trackId = roomState.track.trackId || roomState.track.id;
        let foundPlaylistId = null;

        if (window._userPlaylists && Array.isArray(window._userPlaylists)) {
            for (const pl of window._userPlaylists) {
                if (pl.songs && pl.songs.some(s => s.track_id === trackId || s.url === roomState.track.url)) {
                    foundPlaylistId = pl.id;
                    break;
                }
            }
        }

        isPlaylistActionPending = true;
        btnAddToPlaylist.disabled = true;

        if (foundPlaylistId) {
            removeSongFromPlaylist(foundPlaylistId, trackId);
        } else {
            addTrackToPlaylist(null, {
                id: trackId,
                title: roomState.track.title,
                url: roomState.track.url,
                duration: roomState.track.duration,
                platform: roomState.track.platform
            });
        }

        setTimeout(() => {
            isPlaylistActionPending = false;
            btnAddToPlaylist.disabled = false;
        }, 800);
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

let lastNonZeroVolume = 1.0;

// Update Volume Slider Fill & Dynamic Lucide Volume Icon
function updateVolumeIconsAndFill(valPercentage, isMuted) {
    if (volumeSlider) {
        volumeSlider.style.setProperty('--vol-fill', valPercentage + '%');
    }

    const high = document.getElementById('icon-vol-high') || iconVolHigh;
    const low = document.getElementById('icon-vol-low') || iconVolLow;
    const min = document.getElementById('icon-vol-min') || iconVolMin;
    const mute = document.getElementById('icon-vol-mute') || iconVolMute;

    if (high) high.style.display = 'none';
    if (low) low.style.display = 'none';
    if (min) min.style.display = 'none';
    if (mute) mute.style.display = 'none';

    if (isMuted || valPercentage <= 0) {
        if (mute) mute.style.display = 'inline-block';
    } else if (valPercentage < 10) {
        if (min) min.style.display = 'inline-block';
    } else if (valPercentage < 40) {
        if (low) low.style.display = 'inline-block';
    } else {
        if (high) high.style.display = 'inline-block';
    }
}

// Custom Sleep Timer Dropdown Handler
function setSleepTimerValue(mins, label) {
    if (!sleepTimerSelect) return;
    sleepTimerSelect.value = mins.toString();

    // Update selected text on trigger
    if (sleepTimerSelectedText) {
        sleepTimerSelectedText.innerText = label;
    }

    // Update active class & aria on options
    if (sleepTimerDropdown) {
        const options = sleepTimerDropdown.querySelectorAll('.custom-select-option');
        options.forEach(opt => {
            const isMatch = opt.getAttribute('data-value') === mins.toString();
            if (isMatch) {
                opt.classList.add('active');
                opt.setAttribute('aria-selected', 'true');
            } else {
                opt.classList.remove('active');
                opt.setAttribute('aria-selected', 'false');
            }
        });
    }

    document.querySelectorAll('.sleep-timer-option-btn').forEach(btn => {
        const val = parseInt(btn.getAttribute('data-value'), 10);
        if (val === mins) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }
    });

    // Process timer logic
    if (sleepTimerId) {
        clearInterval(sleepTimerId);
        sleepTimerId = null;
    }

    const rowElem = sleepTimerContainer ? sleepTimerContainer.closest('.profile-row') : null;

    if (mins <= 0) {
        sleepEndTime = null;
        if (sleepTimerStatus) sleepTimerStatus.innerText = 'Off (Max 2h)';
        if (rowElem) rowElem.classList.remove('timer-active');
        showToast('Sleep timer turned off');
        updateSleepTimerUI();
    } else {
        sleepEndTime = Date.now() + (mins * 60 * 1000);
        if (rowElem) rowElem.classList.add('timer-active');
        showToast('Sleep timer set for ' + label);
        updateSleepTimerUI();

        sleepTimerId = setInterval(() => {
            const remainingSecs = Math.round((sleepEndTime - Date.now()) / 1000);
            if (remainingSecs <= 0) {
                clearInterval(sleepTimerId);
                sleepTimerId = null;
                sleepEndTime = null;
                setSleepTimerValue(0, 'Off');

                isAudioUnlocked = false;
                if (audio) {
                    audio.pause();
                }
                showToast('Sleep timer finished — local playback paused');
                if (tg && typeof tg.close === 'function') {
                    tg.close();
                }
            } else {
                updateSleepTimerUI();
            }
        }, 1000);
    }
}

function toggleSleepTimerDropdown(open) {
    if (!sleepTimerContainer) return;
    const shouldOpen = open !== undefined ? open : !sleepTimerContainer.classList.contains('open');
    if (shouldOpen) {
        sleepTimerContainer.classList.add('open');
        if (sleepTimerTrigger) sleepTimerTrigger.setAttribute('aria-expanded', 'true');
    } else {
        sleepTimerContainer.classList.remove('open');
        if (sleepTimerTrigger) sleepTimerTrigger.setAttribute('aria-expanded', 'false');
    }
}

if (sleepTimerTrigger) {
    sleepTimerTrigger.addEventListener('click', (e) => {
        e.stopPropagation();
        toggleSleepTimerDropdown();
    });
}

if (sleepTimerDropdown) {
    sleepTimerDropdown.querySelectorAll('.custom-select-option').forEach(option => {
        option.addEventListener('click', (e) => {
            e.stopPropagation();
            const value = parseInt(option.getAttribute('data-value'), 10);
            const labelText = option.querySelector('span') ? option.querySelector('span').innerText : option.innerText.trim();
            setSleepTimerValue(value, labelText);
            toggleSleepTimerDropdown(false);
        });
    });
}

document.addEventListener('click', (e) => {
    if (sleepTimerContainer && !sleepTimerContainer.contains(e.target)) {
        toggleSleepTimerDropdown(false);
    }
});

document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && sleepTimerContainer && sleepTimerContainer.classList.contains('open')) {
        toggleSleepTimerDropdown(false);
        if (sleepTimerTrigger) sleepTimerTrigger.focus();
    }
});

function openSleepTimerModal() {
    closeAllDrawers();
    openDrawer(sleepTimerBackdrop, sleepTimerDrawer);
}

if (btnPlayerSleepTimer) {
    btnPlayerSleepTimer.addEventListener('click', (e) => {
        e.stopPropagation();
        triggerHaptic('light');
        openSleepTimerModal();
    });
}

if (sleepTimerContainer) {
    sleepTimerContainer.addEventListener('click', (e) => {
        e.stopPropagation();
        triggerHaptic('light');
        openSleepTimerModal();
    });
}

document.querySelectorAll('.sleep-timer-option-btn').forEach(btn => {
    btn.addEventListener('click', (e) => {
        e.stopPropagation();
        triggerHaptic('light');
        const value = parseInt(btn.getAttribute('data-value'), 10);
        const labelText = btn.querySelector('span') ? btn.querySelector('span').innerText.trim() : '';
        setSleepTimerValue(value, labelText);
        closeDrawer(sleepTimerBackdrop, sleepTimerDrawer);
    });
});

function updateSleepTimerUI() {
    const rowElem = sleepTimerContainer ? sleepTimerContainer.closest('.profile-row') : null;
    if (!sleepEndTime) {
        if (sleepTimerStatus) sleepTimerStatus.innerText = 'Off (Max 2h)';
        if (rowElem) rowElem.classList.remove('timer-active');
        if (btnPlayerSleepTimer) btnPlayerSleepTimer.classList.remove('timer-active');
        if (playerSleepTimerBadge) {
            playerSleepTimerBadge.style.display = 'none';
            playerSleepTimerBadge.innerText = '';
        }
        return;
    }

    const remainingSecs = Math.max(0, Math.round((sleepEndTime - Date.now()) / 1000));
    const mins = Math.ceil(remainingSecs / 60);

    let badgeText = '';
    if (mins >= 60) {
        const h = Math.floor(mins / 60);
        const m = mins % 60;
        badgeText = m > 0 ? `${h}h${m}m` : `${h}h`;
    } else if (mins >= 1) {
        badgeText = `${mins}m`;
    } else {
        badgeText = `${remainingSecs}s`;
    }

    if (sleepTimerStatus) {
        if (mins >= 1) {
            sleepTimerStatus.innerText = mins + ' min remaining';
        } else {
            sleepTimerStatus.innerText = remainingSecs + 's remaining';
        }
    }

    if (rowElem) rowElem.classList.add('timer-active');
    if (btnPlayerSleepTimer) btnPlayerSleepTimer.classList.add('timer-active');
    if (playerSleepTimerBadge) {
        playerSleepTimerBadge.style.display = 'inline-block';
        playerSleepTimerBadge.innerText = badgeText;
    }
}

let isDuplicateSession = false;

function stopAppFlowForDuplicateSession() {
    isDuplicateSession = true;
    if (audio) {
        audio.pause();
        audio.src = '';
    }
    if (ws) {
        try {
            ws.close();
        } catch (e) {}
    }
    if (joinOverlay) joinOverlay.style.display = 'none';
    const activeDrawers = document.querySelectorAll('.modal-drawer.active, .modal-backdrop.active');
    activeDrawers.forEach(el => el.classList.remove('active'));

    const dupOverlay = document.getElementById('duplicate-session-overlay');
    if (dupOverlay) dupOverlay.style.display = 'flex';
    if (window.lucide) {
        lucide.createIcons();
    }
}

const btnDuplicateRestart = document.getElementById('btn-duplicate-restart');
if (btnDuplicateRestart) {
    btnDuplicateRestart.addEventListener('click', () => {
        window.location.reload();
    });
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
            initData: (tg && tg.initData) ? tg.initData : ''
        }));
        pingServer();
        if (profileConnStatus) profileConnStatus.innerText = 'Connected';
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            if (msg.event === 'duplicate_session') {
                stopAppFlowForDuplicateSession();
                return;
            }
            if (msg.event === 'room_state') {
                updateRoomState(msg.data);
            } else if (msg.event === 'user_info') {
                isAdmin = msg.data.isAdmin;
                canControl = msg.data.canControl !== undefined ? msg.data.canControl : isAdmin;
                canPlay = msg.data.canPlay !== undefined ? msg.data.canPlay : true;
                if (msg.data.allowsWriteToPM !== undefined) {
                    userAllowsWriteToPM = !!msg.data.allowsWriteToPM;
                    if (currentWebUser) currentWebUser.allows_write_to_pm = userAllowsWriteToPM;
                }
                if (roleText) roleText.innerText = isAdmin ? 'Admin' : 'Listener';
                if (roleBadge) {
                    if (isAdmin) roleBadge.classList.add('admin');
                    else roleBadge.classList.remove('admin');
                }
                if (profileRoleStatus) profileRoleStatus.innerText = isAdmin ? 'Admin (Full Control)' : (canControl ? 'Controller' : 'Listener');
                populateUserProfile();
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
            } else if (msg.event === 'user_playlists' || msg.event === 'playlist_created' || msg.event === 'playlist_deleted' || msg.event === 'playlist_renamed' || msg.event === 'song_added_to_playlist' || msg.event === 'song_removed_from_playlist') {
                if (msg.event === 'playlist_created') showToast('Playlist created!');
                if (msg.event === 'playlist_deleted') showToast('Playlist deleted');
                if (msg.event === 'playlist_renamed') showToast('Playlist renamed!');
                if (msg.event === 'song_added_to_playlist') showToast('Song added to playlist');
                if (msg.event === 'song_removed_from_playlist') showToast('Song removed from playlist');
                if (msg.data && msg.data.playlists) {
                    window._userPlaylists = msg.data.playlists;
                    renderPlaylists(msg.data.playlists);
                } else {
                    fetchPlaylists();
                }
            } else if (msg.event === 'error') {
                showToast(msg.data);
            }
        } catch (e) {
            console.error(e);
        }
    };

    ws.onclose = () => {
        if (isDuplicateSession) {
            return;
        }
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
    if (btnPrev) btnPrev.disabled = !canControl;
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
    if (listenersSessionActions) {
        listenersSessionActions.style.display = isAdmin ? 'block' : 'none';
    }
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
    updateAddToPlaylistButtonState();
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

let pendingActionKeys = new Set();
let lastSkipTime = 0;
let lastPlayPauseTime = 0;

function animateButtonLoading(elem, duration = 800) {
    if (!elem) return;
    elem.classList.add('is-loading');
    setTimeout(() => {
        elem.classList.remove('is-loading');
    }, duration);
}

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
            html += '<div class="song-title" title="' + title.replace(/"/g, '&quot;') + '">' + title + '</div>';
            html += '<div class="song-sub">' + artist + ' • ' + dur + '</div>';
            html += '</div>';
            html += '<div class="song-actions">';
            if (canControl) {
                html += '<button class="btn-song-action primary" title="Play Now" onclick="handleSearchAction(' + index + ', true, this)"><i data-lucide="play"></i></button>';
            }
            html += '<button class="btn-song-action secondary" title="Add to Queue" onclick="handleSearchAction(' + index + ', false, this)"><i data-lucide="plus"></i></button>';
            html += '</div></div>';
        });
    }

    if (modalSearchResults) modalSearchResults.innerHTML = html;
    if (desktopSearchResults) desktopSearchResults.innerHTML = html;
    refreshIcons();
}

window.handleSearchAction = function(index, force, btnElem) {
    if (!window._searchResults || !window._searchResults[index]) return;
    requestTrack(window._searchResults[index], force, btnElem);
};

function requestTrack(track, force, btnElem) {
    if (!track) return;
    const reqKey = (track.id || track.trackId || track.title || '') + '_' + (force ? 'play' : 'enqueue');

    if (pendingActionKeys.has(reqKey)) {
        return;
    }

    if (!ws || ws.readyState !== WebSocket.OPEN) {
        showToast('Connection lost. Please try again.', 'error');
        return;
    }

    pendingActionKeys.add(reqKey);
    setTimeout(() => pendingActionKeys.delete(reqKey), 1200);

    if (btnElem) {
        btnElem.classList.add('is-loading');
        setTimeout(() => btnElem.classList.remove('is-loading'), 1200);
    }

    triggerHaptic('medium');
    ws.send(JSON.stringify({
        type: force ? 'play' : 'enqueue',
        track: track,
        force: force
    }));

    showToast(force ? 'Playing ' + (track.title || 'track') : 'Added to queue: ' + (track.title || 'track'), 'success');
    closeDrawer(searchBackdrop, searchDrawer);
    closeDrawer(relatedBackdrop, relatedDrawer);
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
            html += '<div class="song-title" title="' + title.replace(/"/g, '&quot;') + '">' + title + '</div>';
            html += '<div class="song-sub">' + artist + ' • ' + dur + '</div>';
            html += '</div>';
            html += '<div class="song-actions">';
            if (canControl) {
                html += '<button class="btn-song-action primary" title="Play Now" onclick="handleRelatedAction(' + index + ', true, this)"><i data-lucide="play"></i></button>';
            }
            html += '<button class="btn-song-action secondary" title="Add to Queue" onclick="handleRelatedAction(' + index + ', false, this)"><i data-lucide="plus"></i></button>';
            html += '</div></div>';
        });
    }

    if (relatedScrollList) relatedScrollList.innerHTML = html;
    if (desktopRelatedResults) desktopRelatedResults.innerHTML = html;
    refreshIcons();
}

window.handleRelatedAction = function(index, force, btnElem) {
    if (!window._relatedResults || !window._relatedResults[index]) return;
    requestTrack(window._relatedResults[index], force, btnElem);
};

// Playlist Management Helper Functions
function fetchPlaylists() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'get_playlists' }));
    }
}

function createPlaylist(name) {
    if (!name || !ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({ type: 'create_playlist', playlistName: name }));
}

function renamePlaylist(playlistId, currentName) {
    if (!playlistId || !ws || ws.readyState !== WebSocket.OPEN) return;
    const newName = prompt('Enter new playlist name:', currentName || '');
    if (newName && newName.trim() && newName.trim() !== currentName) {
        ws.send(JSON.stringify({
            type: 'rename_playlist',
            playlistId: playlistId,
            playlistName: newName.trim()
        }));
    }
}

function deletePlaylist(playlistId) {
    if (!playlistId || !ws || ws.readyState !== WebSocket.OPEN) return;
    if (confirm('Are you sure you want to delete this playlist?')) {
        if (confirm('Action is irreversible! Are you REALLY sure you want to permanently delete this playlist?')) {
            ws.send(JSON.stringify({ type: 'delete_playlist', playlistId: playlistId }));
        }
    }
}

function addTrackToPlaylist(playlistId, track) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({
        type: 'add_to_playlist',
        playlistId: playlistId || '',
        track: track || null
    }));
}

function removeSongFromPlaylist(playlistId, trackId) {
    if (!playlistId || !trackId || !ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({
        type: 'remove_from_playlist',
        playlistId: playlistId,
        trackId: trackId
    }));
}

function playPlaylist(playlistId, force, btnElem) {
    if (!playlistId || !ws || ws.readyState !== WebSocket.OPEN) return;
    const reqKey = 'pl_' + playlistId + '_' + (force ? 'play' : 'enqueue');
    if (pendingActionKeys.has(reqKey)) return;

    pendingActionKeys.add(reqKey);
    setTimeout(() => pendingActionKeys.delete(reqKey), 1200);

    if (btnElem) animateButtonLoading(btnElem, 1000);

    triggerHaptic('medium');
    if (force && !isAudioUnlocked) {
        isAudioUnlocked = true;
        if (joinOverlay) {
            joinOverlay.style.opacity = '0';
            joinOverlay.style.visibility = 'hidden';
            setTimeout(() => { joinOverlay.style.display = 'none'; }, 300);
        }
        audio.src = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAAZGF0YQAAAAA=';
        audio.play().catch(e => console.log('Unlock audio play error:', e));
    }
    ws.send(JSON.stringify({
        type: force ? 'play_playlist' : 'enqueue_playlist',
        playlistId: playlistId
    }));
    showToast(force ? 'Playing playlist...' : 'Added playlist to queue', 'success');
}

function playPlaylistSong(playlistId, trackId, force, btnElem) {
    if (!playlistId || !trackId || !ws || ws.readyState !== WebSocket.OPEN) return;
    const reqKey = 'song_' + playlistId + '_' + trackId + '_' + (force ? 'play' : 'enqueue');
    if (pendingActionKeys.has(reqKey)) return;

    pendingActionKeys.add(reqKey);
    setTimeout(() => pendingActionKeys.delete(reqKey), 1200);

    if (btnElem) animateButtonLoading(btnElem, 1000);

    triggerHaptic('medium');
    if (force && !isAudioUnlocked) {
        isAudioUnlocked = true;
        if (joinOverlay) {
            joinOverlay.style.opacity = '0';
            joinOverlay.style.visibility = 'hidden';
            setTimeout(() => { joinOverlay.style.display = 'none'; }, 300);
        }
        audio.src = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAAZGF0YQAAAAA=';
        audio.play().catch(e => console.log('Unlock audio play error:', e));
    }
    ws.send(JSON.stringify({
        type: force ? 'play' : 'enqueue',
        playlistId: playlistId,
        trackId: trackId,
        force: force
    }));
    showToast(force ? 'Playing song...' : 'Added song to queue', 'success');
}

function togglePlaylistAccordion(plId) {
    const listElem = document.getElementById('playlist-songs-' + plId);
    const chevronElem = document.getElementById('playlist-chevron-' + plId);
    if (!listElem) return;
    const isHidden = listElem.style.display === 'none' || listElem.style.display === '';
    if (isHidden) {
        listElem.style.display = 'flex';
        if (chevronElem) chevronElem.style.transform = 'rotate(180deg)';
    } else {
        listElem.style.display = 'none';
        if (chevronElem) chevronElem.style.transform = 'rotate(0deg)';
    }
}

window.renamePlaylist = renamePlaylist;
window.deletePlaylist = deletePlaylist;
window.removeSongFromPlaylist = removeSongFromPlaylist;
window.playPlaylist = playPlaylist;
window.playPlaylistSong = playPlaylistSong;
window.togglePlaylistAccordion = togglePlaylistAccordion;

function renderPlaylists(playlists) {
    const listContainer = document.getElementById('playlist-scroll-list');
    if (!listContainer) return;
    let html = '';
    if (!playlists || playlists.length === 0) {
        html = '<div class="playlist-empty-state" style="padding: 30px 15px; margin-top: 10px;">No playlists created yet.<br>Click <b>+ New</b> above to create your first playlist!</div>';
    } else {
        playlists.forEach((pl) => {
            const songCount = pl.songs ? pl.songs.length : 0;
            const plName = pl.name || 'Untitled Playlist';
            const escapedName = plName.replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '&quot;');

            html += '<div class="playlist-card">';

            // Header
            html += '<div class="playlist-card-header">';

            // Title group (Clicking toggles accordion)
            html += '<div class="playlist-title-group" onclick="togglePlaylistAccordion(\'' + pl.id + '\')">';
            html += '<div class="playlist-icon-badge"><i data-lucide="list-music"></i></div>';
            html += '<div class="playlist-meta">';
            html += '<div class="playlist-name" title="' + plName.replace(/"/g, '&quot;') + '">' + plName + '</div>';
            html += '<div class="playlist-song-count">' + songCount + ' ' + (songCount === 1 ? 'song' : 'songs') + '</div>';
            html += '</div>';
            html += '<i data-lucide="chevron-down" class="playlist-accordion-chevron" id="playlist-chevron-' + pl.id + '"></i>';
            html += '</div>';

            // Actions (Rename, Delete)
            html += '<div class="playlist-header-actions">';
            html += '<button class="btn-action secondary small" style="padding: 4px 8px;" onclick="event.stopPropagation(); renamePlaylist(\'' + pl.id + '\', \'' + escapedName + '\')" title="Rename Playlist"><i data-lucide="pencil" style="width: 14px; height: 14px;"></i></button>';
            html += '<button class="btn-action danger small" style="padding: 4px 8px;" onclick="event.stopPropagation(); deletePlaylist(\'' + pl.id + '\')" title="Delete Playlist"><i data-lucide="trash-2" style="width: 14px; height: 14px;"></i></button>';
            html += '</div>';

            html += '</div>'; // end playlist-card-header

            // Collapsible Songs Container
            html += '<div id="playlist-songs-' + pl.id + '" class="playlist-songs-container" style="display: none;">';

            if (songCount > 0) {
                // Top bar in accordion with + Queue All
                html += '<div class="playlist-songs-top-bar">';
                html += '<button class="btn-action primary small" style="width: 100%; justify-content: center;" onclick="playPlaylist(\'' + pl.id + '\', false)"><i data-lucide="plus"></i> <span>Queue All (' + songCount + ')</span></button>';
                html += '</div>';

                // Songs list
                pl.songs.forEach((s, idx) => {
                    const songName = s.name || 'Song';
                    const dur = formatTime(s.duration);

                    html += '<div class="song-row">';
                    html += '<div class="song-info">';
                    html += '<div class="song-title" title="' + songName.replace(/"/g, '&quot;') + '">' + (idx + 1) + '. ' + songName + '</div>';
                    html += '<div class="song-sub">' + dur + '</div>';
                    html += '</div>';

                    html += '<div class="song-actions">';
                    if (canControl) {
                        html += '<button class="btn-song-action primary" title="Play Now" onclick="playPlaylistSong(\'' + pl.id + '\', \'' + s.track_id + '\', true)"><i data-lucide="play"></i></button>';
                    }
                    html += '<button class="btn-song-action secondary" title="Add to Queue" onclick="playPlaylistSong(\'' + pl.id + '\', \'' + s.track_id + '\', false)"><i data-lucide="plus"></i></button>';
                    html += '<button class="btn-song-action secondary" title="Remove song" onclick="removeSongFromPlaylist(\'' + pl.id + '\', \'' + s.track_id + '\')"><i data-lucide="trash-2"></i></button>';
                    html += '</div>';

                    html += '</div>';
                });
            } else {
                html += '<div class="playlist-empty-state">No songs in playlist yet. Use <i data-lucide="heart-plus" style="width:14px; height:14px; vertical-align:middle; color:var(--accent-2);"></i> on player to add current song.</div>';
            }

            html += '</div>'; // end playlist-songs-container
            html += '</div>'; // end playlist-card
        });
    }
    listContainer.innerHTML = html;
    refreshIcons();
    updateAddToPlaylistButtonState();
}

// Playback Control Triggers
function togglePlayPause() {
    const now = Date.now();
    if (now - lastPlayPauseTime < 500) return;
    lastPlayPauseTime = now;

    triggerHaptic('medium');
    if (!canControl) {
        showToast('Control permission required', 'warning');
        return;
    }

    if (btnPlay) animateButtonLoading(btnPlay, 500);
    if (miniBtnPlay) animateButtonLoading(miniBtnPlay, 500);

    const isPlaying = (roomState && roomState.playback && roomState.playback.status === 'playing');
    if (ws && ws.readyState === WebSocket.OPEN) {
        if (isPlaying) {
            ws.send(JSON.stringify({ type: 'pause' }));
            showToast('Playback paused', 'info', 1500);
        } else {
            ws.send(JSON.stringify({ type: 'resume' }));
            showToast('Playback resumed', 'success', 1500);
        }
    }
}
if (btnPlay) btnPlay.addEventListener('click', togglePlayPause);
if (miniBtnPlay) miniBtnPlay.addEventListener('click', togglePlayPause);

function skipTrack() {
    const now = Date.now();
    if (now - lastSkipTime < 800) return;
    lastSkipTime = now;

    triggerHaptic('medium');
    if (!canControl) {
        showToast('Control permission required to skip', 'warning');
        return;
    }

    if (btnSkip) animateButtonLoading(btnSkip, 800);
    if (miniBtnSkip) animateButtonLoading(miniBtnSkip, 800);

    const currentTrackId = (roomState && roomState.track) ? (roomState.track.trackId || roomState.track.id || '') : '';
    if (currentTrackId) {
        lastEndedTrackId = currentTrackId;
    }
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'skip' }));
        showToast('Skipping track...', 'info', 1500);
    }
}
if (btnSkip) btnSkip.addEventListener('click', skipTrack);
if (miniBtnSkip) miniBtnSkip.addEventListener('click', skipTrack);

if (btnPrev) {
    btnPrev.addEventListener('click', () => {
        triggerHaptic('medium');
        if (!canControl) return;
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'seek', position: 0 }));
            showToast('Restarted track');
        }
    });
}

if (btnEndSession) {
    btnEndSession.addEventListener('click', () => {
        triggerHaptic('medium');
        if (!isAdmin) {
            showToast('Only admins can end the listening session', 'warning');
            return;
        }
        if (endSessionConfirmOverlay) endSessionConfirmOverlay.style.display = 'flex';
    });
}

if (btnConfirmEndCancel) {
    btnConfirmEndCancel.addEventListener('click', () => {
        triggerHaptic('light');
        if (endSessionConfirmOverlay) endSessionConfirmOverlay.style.display = 'none';
    });
}

if (btnConfirmEndSubmit) {
    btnConfirmEndSubmit.addEventListener('click', () => {
        triggerHaptic('heavy');
        if (endSessionConfirmOverlay) endSessionConfirmOverlay.style.display = 'none';
        closeDrawer(listenersBackdrop, listenersDrawer);
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'stop' }));
            showToast('Listening session ended', 'info');
        }
    });
}

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
        if (!isNaN(v) && v >= 0 && v <= 1) {
            audio.volume = v;
            audio.muted = (v === 0);
            if (v > 0) lastNonZeroVolume = v;
            const percent = Math.round(v * 100);
            volumeSlider.value = percent;
            updateVolumeIconsAndFill(percent, audio.muted);
        } else {
            updateVolumeIconsAndFill(100, false);
        }
    } else {
        updateVolumeIconsAndFill(100, false);
    }

    volumeSlider.addEventListener('input', () => {
        const percent = parseFloat(volumeSlider.value);
        const val = percent / 100;
        audio.volume = val;
        if (val > 0) {
            lastNonZeroVolume = val;
            audio.muted = false;
        } else {
            audio.muted = true;
        }
        localStorage.setItem('tg_player_volume', val);
        updateVolumeIconsAndFill(percent, audio.muted);
    });
}

if (btnMute) {
    btnMute.addEventListener('click', () => {
        triggerHaptic('light');
        audio.muted = !audio.muted;
        let percent = parseFloat(volumeSlider ? volumeSlider.value : 100);
        if (audio.muted) {
            updateVolumeIconsAndFill(percent, true);
        } else {
            if (audio.volume === 0 || percent === 0) {
                const restoreVal = lastNonZeroVolume || 0.8;
                audio.volume = restoreVal;
                percent = Math.round(restoreVal * 100);
                if (volumeSlider) volumeSlider.value = percent;
                localStorage.setItem('tg_player_volume', restoreVal);
            }
            updateVolumeIconsAndFill(percent, false);
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

let lastEndedTrackId = null;

// Audio Media Error Event
audio.addEventListener('error', (e) => {
    if (!currentAudioUrl || (audio.src && audio.src.startsWith('data:audio/'))) return;
    const err = audio.error;
    const errCode = err ? err.code : 0;
    const errMsg = err ? err.message : '';
    console.warn('Audio playback error (code ' + errCode + '): ' + errMsg);

    if (errCode === 4 || errCode === 3) {
        showToast('Stream URL expired or unplayable');
        const currentTrackId = (roomState && roomState.track) ? (roomState.track.trackId || roomState.track.id || '') : '';
        if (currentTrackId && lastEndedTrackId !== currentTrackId && canControl && ws && ws.readyState === WebSocket.OPEN) {
            lastEndedTrackId = currentTrackId;
            ws.send(JSON.stringify({ type: 'track_end', trackId: currentTrackId }));
        }
    } else {
        showToast('Playback interrupted. Retrying...');
    }
});

// Audio Ended Event
audio.addEventListener('ended', () => {
    if (!currentAudioUrl || (audio.src && audio.src.startsWith('data:audio/'))) return;
    const currentTrackId = (roomState && roomState.track) ? (roomState.track.trackId || roomState.track.id || '') : '';
    if (!currentTrackId) return;

    if (lastEndedTrackId === currentTrackId) {
        return;
    }

    if (ws && ws.readyState === WebSocket.OPEN) {
        lastEndedTrackId = currentTrackId;
        ws.send(JSON.stringify({ type: 'track_end', trackId: currentTrackId }));
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
