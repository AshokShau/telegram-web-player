// Shared state, platform capabilities, preferences, API and room transport.
import { readLaunch, browserSessionURL } from './launch.js?v=23';
export const telegram = window.Telegram?.WebApp ?? null;
const launch = readLaunch(telegram);
let browserHandoffComplete = false;
let browserHandoffTakeOver = launch.isBrowserHandoff;
export const hasSession = Boolean(launch.initData);
export const platform = Object.freeze({
    isTelegramWebApp: launch.isTelegramWebApp,
    supportsBrowserFullscreen: Boolean(document.fullscreenEnabled && document.documentElement.requestFullscreen),
    supportsTelegramFullscreen: Boolean(telegram?.isVersionAtLeast?.('8.0') && telegram.requestFullscreen && telegram.exitFullscreen),
    supportsVoice: Boolean(window.RTCPeerConnection), supportsMicrophone: Boolean(navigator.mediaDevices?.getUserMedia)
});
export const preferences = {
    get(key, fallback) { try { return JSON.parse(localStorage.getItem(`synctune:${key}`)) ?? fallback; } catch { return fallback; } },
    set(key, value) { try { localStorage.setItem(`synctune:${key}`, JSON.stringify(value)); } catch { /* Storage may be unavailable in a WebView. */ } }
};
const roomId = launch.roomId;
export const state = {
    roomId, room: null, user: { ...launch.user },
    permissions: { userId: 0, isAdmin: false, isAuth: false, canControl: false, canPlay: false, allowsWriteToPM: false },
    connection: 'idle', stopped: false, sessionFailure: null, joinedListening: false, audioStatus: '', view: 'home', expandedPlayer: false,
    offset: 0, revision: 0, vcRevision: 0, theme: preferences.get('theme', 'full-dark'),
    playlists: [], libraryStatus: 'idle', selectedPlaylist: null,
    search: { query: '', results: [], status: 'idle', error: '' }, mix: { results: [], status: 'idle' },
    chat: { messages: [], open: false, unread: 0, cooldownUntil: 0, status: 'loading' },
    voice: { room: { participants: [], muteNewParticipants: false }, status: 'idle', joined: false, muted: true, outputMuted: false, open: false, settingsOpen: false, noiseSuppression: preferences.get('noise-suppression', true) !== false },
    sleepUntil: 0, history: preferences.get(`history:${roomId}`, [])
};
const bus = new EventTarget();
export function on(type, fn) { bus.addEventListener(type, e => fn(e.detail)); }
export function emit(type, detail) { bus.dispatchEvent(new CustomEvent(type, { detail })); }
export function haptic(type = 'selection') {
    if (!platform.isTelegramWebApp) return;
    const feedback = telegram?.HapticFeedback;
    if (!feedback) return;
    try {
        if (['success', 'warning', 'error'].includes(type)) feedback.notificationOccurred?.(type);
        else if (type === 'selection') feedback.selectionChanged?.();
        else feedback.impactOccurred?.(type);
    } catch { /* Haptics are optional and differ between Telegram clients. */ }
}
export function notify(message, type = '') { if (type) haptic(type); emit('notice', String(message)); }
export function canControl() { return state.connection === 'connected' && state.permissions.canControl && state.permissions.allowsWriteToPM; }
export function canPlay() { return state.connection === 'connected' && state.permissions.canPlay && state.permissions.allowsWriteToPM; }
export function canManageSettings() { return state.connection === 'connected' && state.permissions.isAdmin; }
export function playbackPosition(room = state.room, now = Date.now() + state.offset) {
    if (!room?.track) return 0;
    const pb = room.playback;
    const elapsed = pb?.status === 'playing' && room.track.ready !== false ? Math.max(0, now - pb.serverTime) / 1000 : 0;
    const position = Math.max(0, (Number(pb?.position) || 0) + elapsed);
    return room.track.duration > 0 ? Math.min(position, room.track.duration) : position;
}
export function formatTime(seconds) {
    const n = Math.max(0, Math.floor(Number(seconds) || 0));
    return n >= 3600 ? `${Math.floor(n / 3600)}:${String(Math.floor(n / 60) % 60).padStart(2, '0')}:${String(n % 60).padStart(2, '0')}` : `${Math.floor(n / 60)}:${String(n % 60).padStart(2, '0')}`;
}
export function safeURL(value, fallback = '') {
    if (typeof value !== 'string' || !value.trim()) return fallback;
    try { const url = new URL(value, location.origin); return ['http:', 'https:'].includes(url.protocol) ? url.href : fallback; } catch { return fallback; }
}

// Theme has one preference and one resolved color scheme.
const systemTheme = matchMedia('(prefers-color-scheme: dark)');
export function applyTheme(value = state.theme) {
    state.theme = ['system', 'dark', 'light', 'full-dark'].includes(value) ? value : 'full-dark';
    const resolved = state.theme === 'system' ? (systemTheme.matches ? 'dark' : 'light') : state.theme;
    document.documentElement.dataset.theme = resolved;
    preferences.set('theme', state.theme);
    const color = { dark: '#111318', light: '#f6f7fb', 'full-dark': '#000000' }[resolved];
    document.querySelector('meta[name="theme-color"]').content = color;
    if (platform.isTelegramWebApp) {
        for (const method of ['setHeaderColor', 'setBackgroundColor', 'setBottomBarColor']) {
            try { telegram[method]?.(color); } catch { /* Older Telegram clients may reject a color API. */ }
        }
    }
    emit('theme');
}
systemTheme.addEventListener('change', () => { if (state.theme === 'system') applyTheme(); });
function syncViewport() {
    const fullscreen = isFullscreen();
    document.documentElement.dataset.fullscreen = String(fullscreen);
    for (const side of ['top', 'bottom', 'left', 'right']) {
        for (const [name, value] of [['device', telegram?.safeAreaInset?.[side]], ['content', telegram?.contentSafeAreaInset?.[side]]]) {
            // Normal Telegram windows already exclude native chrome. In fullscreen,
            // device insets and Telegram's content insets reserve separate areas.
            const inset = fullscreen && platform.isTelegramWebApp && Number.isFinite(Number(value)) ? Math.max(0, Number(value)) : 0;
            document.documentElement.style.setProperty(`--${name}-safe-${side}`, `${inset}px`);
        }
    }
    const visual = window.visualViewport;
    const useVisual = visual && visual.scale === 1;
    const top = useVisual ? Math.max(0, visual.offsetTop) : 0;
    const heights = [window.innerHeight - top, useVisual ? visual.height : 0,
        platform.isTelegramWebApp ? telegram.viewportHeight : 0,
        platform.isTelegramWebApp ? telegram.viewportStableHeight : 0]
        .map(Number).filter(value => Number.isFinite(value) && value > 0);
    const height = Math.min(...heights);
    document.documentElement.style.setProperty('--app-height', `${height}px`);
    document.documentElement.style.setProperty('--viewport-top', `${top}px`);
    emit('fullscreen');
}
export function isFullscreen() { return Boolean(document.fullscreenElement || (platform.isTelegramWebApp && telegram?.isFullscreen)); }
export async function toggleFullscreen() {
    try {
        if (document.fullscreenElement) await document.exitFullscreen();
        else if (platform.isTelegramWebApp && telegram?.isFullscreen) telegram.exitFullscreen();
        else if (platform.isTelegramWebApp && platform.supportsTelegramFullscreen) telegram.requestFullscreen();
        else if (platform.supportsBrowserFullscreen) await document.documentElement.requestFullscreen();
        else notify('Fullscreen is unavailable on this platform.', 'warning');
    } catch { notify('Fullscreen could not be opened. You can keep listening here.', 'error'); }
    emit('fullscreen');
}
export function initializePlatform() {
    applyTheme(); syncViewport();
    document.addEventListener('fullscreenchange', syncViewport);
    window.addEventListener('resize', syncViewport);
    window.visualViewport?.addEventListener('resize', syncViewport);
    window.visualViewport?.addEventListener('scroll', syncViewport);
    if (!platform.isTelegramWebApp) return;
    telegram.ready(); telegram.expand();
    for (const event of ['safeAreaChanged', 'contentSafeAreaChanged', 'viewportChanged', 'fullscreenChanged']) telegram.onEvent?.(event, syncViewport);
    telegram.onEvent?.('fullscreenFailed', () => { notify('Telegram could not enter fullscreen.', 'error'); emit('fullscreen'); });
    telegram.onEvent?.('themeChanged', () => { if (state.theme === 'system') applyTheme(); });
    telegram.BackButton?.onClick(() => emit('back'));
    if (['android', 'ios'].includes(String(telegram.platform).toLowerCase()) && platform.supportsTelegramFullscreen && !isFullscreen()) toggleFullscreen();
}

export function requestWriteAccess() {
    if (!platform.isTelegramWebApp || !telegram?.requestWriteAccess) { notify('Open the player in Telegram to allow bot messages.', 'warning'); return; }
    telegram.requestWriteAccess(granted => {
        if (granted) send('write_access_granted');
        else notify('Bot message access was declined. You can still listen.', 'warning');
    });
}

export function openInBrowser() {
    if (browserHandoffComplete) return;
    const url = browserSessionURL(launch);
    if (platform.isTelegramWebApp && typeof telegram.openLink === 'function') {
        let opened = false;
        try { telegram.openLink(url, { try_instant_view: false }); opened = true; } catch { /* Try a normal browser window. */ }
        if (opened) {
            browserHandoffComplete = true;
            state.stopped = true;
            emit('dispose');
            disconnect();
            try { telegram.disableClosingConfirmation?.(); } catch { /* Optional on older clients. */ }
            try { telegram.close?.(); } catch { /* The old session remains stopped. */ }
            return;
        }
    }
    window.open(url, '_blank', 'noopener,noreferrer');
}
export async function searchAPI(query, signal) {
    const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`, { signal, headers: { 'X-Telegram-Init-Data': launch.initData } });
    if (!response.ok) throw new Error(response.status === 401 ? 'Open SyncTune in Telegram to search music.' : `Search is unavailable (${response.status}). Try again.`);
    const data = await response.json();
    if (data.error) throw new Error(data.error);
    return data.results || [];
}

// One socket, heartbeat and reconnect timer. Commands never mutate room state optimistically.
let socket = null;
let reconnectTimer = null;
let heartbeat = null;
let attempt = 0;
let requestNumber = 0;
let lastRoomTrack = null;
const pending = new Map();
function connection(value) { state.connection = value; emit('connection'); }
function settle(requestId) {
    if (!requestId) return;
    const item = pending.get(requestId);
    if (item) { clearTimeout(item.timer); pending.delete(requestId); }
}
export function send(type, fields = {}) {
    if (state.stopped) return false;
    if (!socket || socket.readyState !== WebSocket.OPEN || state.connection !== 'connected') { notify('Room connection is unavailable. Reconnect and try again.', 'error'); return false; }
    const requestId = String(++requestNumber);
    socket.send(JSON.stringify({ type, requestId, ...fields }));
    if (!['ping', 'vc_candidate', 'vc_speaking', 'vc_answer', 'vc_offer', 'get_vc_state'].includes(type)) {
        const timer = setTimeout(() => {
            pending.delete(requestId);
            emit('request-error', { type, message: 'The room did not respond. Try again.' });
        }, 20000);
        pending.set(requestId, { type, timer });
    }
    return requestId;
}
function ping() { if (socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: 'ping', clientTime: Date.now() })); }
function receive(message) {
    settle(message.requestId);
    const data = message.data;
    switch (message.event) {
        case 'room_state': {
            if (data.revision && data.revision <= state.revision) return;
            state.revision = data.revision || 0;
            // Until the first pong, avoid interpolating against an uncalibrated device clock.
            if (!state.room) state.offset = data.playback.serverTime - Date.now();
            state.room = data;
            if (data.vc) acceptVoiceState(data.vc);
            if (data.track && data.track.ready !== false && data.track.id !== lastRoomTrack) {
                lastRoomTrack = data.track.id;
                state.history = [data.track, ...state.history.filter(t => t.id !== data.track.id)].slice(0, 20).map(({ audioUrl, ...track }) => track);
                preferences.set(`history:${state.roomId}`, state.history);
                emit('history');
            }
            emit('room'); break;
        }
        case 'user_info': state.permissions = data; state.user.id = data.userId; state.sessionFailure = null; emit('permissions'); emit('session-ready'); break;
        case 'pong': {
            const rtt = Date.now() - data.clientTime;
            if (rtt >= 0 && rtt < 5000) state.offset = data.serverTime + rtt / 2 - Date.now();
            emit('clock'); break;
        }
        case 'duplicate_session':
            state.sessionFailure = { code: 'session_moved', message: data };
            state.stopped = true; disconnect(); emit('session-ended', data); break;
        case 'vc_state': acceptVoiceState(data); break;
        case 'vc_user_speaking': {
            const participant = state.voice.room.participants.find(p => p.userId === data.userId);
            if (participant) participant.isSpeaking = data.isSpeaking;
            emit('voice-speaking', data); break;
        }
        case 'vc_offer': case 'vc_answer': case 'vc_candidate': emit(message.event, data); break;
        case 'chat_history': state.chat.messages = data.messages || []; state.chat.status = 'ready'; emit('chat-history'); break;
        case 'chat_message':
            if (!state.chat.messages.some(m => m.id === data.id)) { state.chat.messages.push(data); state.chat.messages = state.chat.messages.slice(-100); }
            if (!state.chat.open) state.chat.unread++;
            if (data.userId === state.permissions.userId) state.chat.cooldownUntil = Date.now() + (state.room?.chatCooldown || 0) * 1000;
            emit('chat-message', data); break;
        case 'chat_error': state.chat.cooldownUntil = Date.now() + (data.remainingSeconds || 0) * 1000; notify(data.message, 'error'); emit('chat-status'); break;
        case 'user_playlists': case 'playlist_created': case 'playlist_deleted': case 'playlist_renamed': case 'song_added_to_playlist': case 'song_removed_from_playlist':
            state.playlists = data.playlists || []; state.libraryStatus = 'ready'; emit('library', message.event); break;
        case 'recommendations_results': state.mix = { results: data.results || [], status: data.error ? 'error' : 'ready', error: data.error }; emit('mix'); break;
        case 'error':
            if (message.command === 'join') {
                state.sessionFailure = { code: message.code || 'telegram_authentication_required', message: typeof data === 'string' ? data : data?.message };
                state.stopped = true; disconnect(); emit('authentication-failed', state.sessionFailure);
                break;
            }
            notify(typeof data === 'string' ? data : data?.message || 'The action failed.', 'error');
            emit('request-error', { type: message.command, message: data }); break;
        case 'ack': break;
        default: break;
    }
}
function acceptVoiceState(data) {
    if (data.revision && data.revision <= state.vcRevision) return;
    state.vcRevision = data.revision || 0;
    state.voice.room = { ...data, participants: data.participants || [] };
    emit('voice-state');
}
export function connect({ takeOver = false } = {}) {
    if (state.stopped || !hasSession || !/^-?\d+$/.test(state.roomId) || state.roomId === '0') return;
    clearTimeout(reconnectTimer);
    if (socket && [WebSocket.OPEN, WebSocket.CONNECTING].includes(socket.readyState)) return;
    connection(attempt ? 'reconnecting' : 'connecting');
    const candidate = new WebSocket(`${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws?room=${encodeURIComponent(state.roomId)}`);
    socket = candidate;
    const openTimeout = setTimeout(() => candidate.close(), 15000);
    candidate.onopen = () => {
        if (socket !== candidate || state.stopped) { clearTimeout(openTimeout); candidate.close(); return; }
        clearTimeout(openTimeout); state.revision = 0; state.vcRevision = 0;
        candidate.send(JSON.stringify({ type: 'join', roomId: state.roomId, initData: launch.initData, ...(takeOver || browserHandoffTakeOver ? { takeOver: true } : {}) }));
        browserHandoffTakeOver = false;
        // Mark transport usable; permissions still gate commands until user_info arrives.
        connection('connected'); attempt = 0; ping();
        clearInterval(heartbeat); heartbeat = setInterval(ping, 15000);
    };
    candidate.onmessage = event => { if (socket !== candidate || state.stopped) return; try { receive(JSON.parse(event.data)); } catch (error) { console.error('Invalid room event', error); } };
    candidate.onerror = () => candidate.close();
    candidate.onclose = () => {
        clearTimeout(openTimeout);
        if (socket !== candidate) return;
        clearInterval(heartbeat);
        socket = null;
        state.permissions = { userId: 0, isAdmin: false, isAuth: false, canControl: false, canPlay: false, allowsWriteToPM: false };
        for (const { timer } of pending.values()) clearTimeout(timer);
        pending.clear(); emit('permissions'); emit('transport-lost');
        if (state.stopped) { connection('closed'); return; }
        connection('reconnecting');
        reconnectTimer = setTimeout(connect, Math.min(15000, 800 * 2 ** attempt++) + Math.random() * 500);
    };
}
export function disconnect() { clearTimeout(reconnectTimer); clearInterval(heartbeat); socket?.close(); }
window.addEventListener('pagehide', event => { state.stopped = true; disconnect(); if (!event.persisted) emit('dispose'); });
window.addEventListener('pageshow', e => { if (e.persisted && !state.sessionFailure && !browserHandoffComplete) { state.stopped = false; connect(); } });
window.addEventListener('online', connect);
