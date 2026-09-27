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

const audio = document.getElementById('audio-element');
const joinOverlay = document.getElementById('join-overlay');
const btnJoin = document.getElementById('btn-join');
const idleView = document.getElementById('idle-view');
const activePlayerView = document.getElementById('active-player-view');
const trackTitle = document.getElementById('track-title');
const trackArtist = document.getElementById('track-artist');
const requesterName = document.getElementById('requester-name');
const trackThumb = document.getElementById('track-thumb');
const artContainer = document.getElementById('art-container');
const platformBadge = document.getElementById('platform-badge');
const ambientGlow = document.getElementById('ambient-glow');
const roleBadge = document.getElementById('role-badge');
const roleText = document.getElementById('role-text');
const listenersTrigger = document.getElementById('listeners-trigger');
const listenersCountText = document.getElementById('listeners-count-text');
const seekSlider = document.getElementById('seek-slider');
const currTime = document.getElementById('curr-time');
const totalTime = document.getElementById('total-time');
const btnPlay = document.getElementById('btn-play');
const iconPlay = document.getElementById('icon-play');
const iconPause = document.getElementById('icon-pause');
const btnSkip = document.getElementById('btn-skip');
const btnMute = document.getElementById('btn-mute');
const iconVolHigh = document.getElementById('icon-vol-high');
const iconVolMute = document.getElementById('icon-vol-mute');
const queueList = document.getElementById('queue-list');
const queueCount = document.getElementById('queue-count');
const toastMsg = document.getElementById('toast-msg');

// User profile elements
const userAvatarPlaceholder = document.getElementById('user-avatar-placeholder');
const userAvatarImg = document.getElementById('user-avatar-img');
const userPremiumBadge = document.getElementById('user-premium-badge');
const userDisplayName = document.getElementById('user-display-name');
const userHandle = document.getElementById('user-handle');

// Modal elements
const modalBackdrop = document.getElementById('modal-backdrop');
const modalDrawer = document.getElementById('modal-drawer');
const modalCloseBtn = document.getElementById('modal-close-btn');
const modalListenersCount = document.getElementById('modal-listeners-count');
const listenersScrollList = document.getElementById('listeners-scroll-list');

let trackDuration = 0;
let currentPosition = 0;
let serverTimeOffset = 0;
let roomState = null;
let isUserSeeking = false;
let isAdmin = false;
let canControl = false;
let ws = null;
let isAudioUnlocked = false;
let pendingSeekPosition = null;
let playPromise = null;
let currentAudioUrl = null;
let hls = null;

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
        audio.src = '';
        audio.removeAttribute('src');
        audio.load();
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
        userDisplayName.innerText = name || 'Telegram User';
        userHandle.innerText = u.username ? '@' + u.username : 'ID: ' + u.id;

        if (u.photo_url) {
            userAvatarImg.src = u.photo_url;
            userAvatarImg.style.display = 'block';
            userAvatarPlaceholder.style.display = 'none';
        } else {
            const initials = (u.first_name ? u.first_name[0] : 'U').toUpperCase();
            userAvatarPlaceholder.innerText = initials;
        }

        if (u.is_premium) {
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

btnJoin.addEventListener('click', () => {
    triggerHaptic('medium');
    isAudioUnlocked = true;
    audio.src = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAAZGF0YQAAAAA=';
    startAudioPlayback();
    joinOverlay.style.opacity = '0';
    joinOverlay.style.visibility = 'hidden';
    setTimeout(() => { joinOverlay.style.display = 'none'; }, 300);
    if (roomState) {
        updateRoomState(roomState);
    }
});

connectWS();

audio.addEventListener('ended', () => {
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
                if (isAdmin) {
                    roleBadge.classList.add('admin');
                    roleText.innerText = 'Admin';
                } else {
                    roleBadge.classList.remove('admin');
                    roleText.innerText = 'Listener';
                }
                updateControlButtonsState();
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
    seekSlider.disabled = !canControl;
}

function updateRoomState(data) {
    roomState = data;
    const listenersCount = data.listeners ? data.listeners.length : 0;
    listenersCountText.innerText = listenersCount + (listenersCount === 1 ? ' Listener' : ' Listeners');
    modalListenersCount.innerText = listenersCount;

    updateListenersList(data.listeners || []);

    const track = data.track;
    const pb = data.playback;

    if (!track) {
        idleView.style.display = 'flex';
        activePlayerView.style.display = 'none';
        ambientGlow.style.opacity = '0.1';
        iconPlay.style.display = 'block';
        iconPause.style.display = 'none';
        stopAudioPlayback(true);
        currTime.innerText = '0:00';
        totalTime.innerText = '0:00';
        seekSlider.value = 0;
        updateQueue([]);
        return;
    }

    idleView.style.display = 'none';
    activePlayerView.style.display = 'flex';

    trackTitle.innerText = track.title || 'Unknown Track';
    trackArtist.innerText = track.artist || track.platform || 'Music';
    requesterName.innerText = 'Requested by ' + (track.user || 'User');
    platformBadge.innerText = (track.platform || 'Music').toUpperCase();
    if (track.thumbnail) {
        trackThumb.src = track.thumbnail;
    } else {
        trackThumb.src = 'https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg';
    }
    trackDuration = track.duration || 0;
    totalTime.innerText = formatTime(trackDuration);

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
        seekSlider.max = trackDuration;
        seekSlider.value = currentPosition;
        currTime.innerText = formatTime(currentPosition);
    }

    if (pb.status === 'playing') {
        iconPlay.style.display = 'none';
        iconPause.style.display = 'block';
        artContainer.classList.add('playing');
        ambientGlow.style.opacity = '0.35';
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
        iconPlay.style.display = 'block';
        iconPause.style.display = 'none';
        artContainer.classList.remove('playing');
        ambientGlow.style.opacity = '0.1';
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
    queueCount.innerText = queue.length + ' tracks';
    if (!queue || queue.length === 0) {
        queueList.innerHTML = '<div style="font-size: 12px; color: var(--text-muted); text-align: center; padding: 14px;">No upcoming tracks in queue</div>';
        return;
    }

    let html = '';
    queue.forEach((item, index) => {
        html += '<div class="queue-item">';
        html += '<img class="queue-thumb" src="' + (item.thumbnail || trackThumb.src) + '" alt="thumb">';
        html += '<div class="queue-details">';
        html += '<div class="queue-track-title">' + (index + 1) + '. ' + (item.title || 'Track') + '</div>';
        html += '<div class="queue-track-sub">' + formatTime(item.duration) + ' • Requested by ' + (item.user || 'User') + '</div>';
        html += '</div></div>';
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

btnMute.addEventListener('click', () => {
    triggerHaptic('light');
    audio.muted = !audio.muted;
    if (audio.muted) {
        iconVolHigh.style.display = 'none';
        iconVolMute.style.display = 'block';
    } else {
        iconVolHigh.style.display = 'block';
        iconVolMute.style.display = 'none';
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
