// A single audio element follows the room clock. Local preferences never change room playback.
import { state, on, emit, notify, preferences, playbackPosition, canControl, send, safeURL } from './core.js?v=23';
export function createPlayback(audio) {
    let source = '';
    let mediaAvailable = false;
    let hls = null;
    let generation = 0;
    let retryTimer = null;
    let retries = 0;
    let playPending = false;
    let lastSync = 0;
    let disposed = false;
    let observedTime = 0;
    const bufferingStatuses = new Set(['Buffering…', 'Waiting for the stream…']);
    function clearBuffering() {
        if (bufferingStatuses.has(state.audioStatus) && !audio.seeking && audio.readyState >= 2) status('');
    }
    const savedVolume = Number(preferences.get('volume', 1));
    audio.volume = Number.isFinite(savedVolume) ? Math.min(1, Math.max(0, savedVolume)) : 1;
    audio.muted = preferences.get('music-muted', false);
    function status(value) { if (state.audioStatus !== value) { state.audioStatus = value; emit('player-status'); } }
    function destroySource() { hls?.destroy(); hls = null; clearTimeout(retryTimer); }
    function readyToPlay() { return state.connection === 'connected' && state.permissions.userId && state.joinedListening && state.room?.playback.status === 'playing' && state.room?.track?.ready !== false && source && mediaAvailable && !disposed; }
    async function play() {
        if (!readyToPlay() || playPending || !audio.paused) return;
        const currentGeneration = generation;
        playPending = true;
        try {
            await audio.play();
            if (currentGeneration === generation && !readyToPlay()) audio.pause();
        } catch (error) {
            if (currentGeneration !== generation) return;
            if (error.name === 'NotAllowedError') {
                state.joinedListening = false; status('Tap Start listening to enable audio'); emit('listening');
            } else if (error.name !== 'AbortError') status('Playback could not start. Tap Retry.');
        } finally { if (currentGeneration === generation) playPending = false; }
    }
    function seekToRoom() {
        if (audio.readyState < 1 || audio.seeking || !source) return;
        const target = playbackPosition();
        if (Math.abs(audio.currentTime - target) > 1.2) {
            try { audio.currentTime = target; } catch { /* The stream may not be seekable until metadata arrives. */ }
        }
    }
    function load(url) {
        destroySource(); generation++; source = url; mediaAvailable = false; playPending = false; retries = 0;
        audio.pause(); audio.removeAttribute('src'); audio.load(); status(state.joinedListening ? 'Loading music…' : 'Tap Start listening to enable audio');
        if (!url) { audio.removeAttribute('src'); audio.load(); status(''); return; }
        const parsed = new URL(url, location.origin);
        const isHls = /\.m3u8$/i.test(parsed.pathname) || /(?:format|type)=hls/i.test(parsed.search);
        if (isHls && !audio.canPlayType('application/vnd.apple.mpegurl')) {
            if (!window.Hls?.isSupported()) { status('This device cannot play this stream.'); return; }
            mediaAvailable = true;
            hls = new window.Hls({ enableWorker: true });
            hls.attachMedia(audio);
            hls.on(window.Hls.Events.MEDIA_ATTACHED, () => hls?.loadSource(url));
            hls.on(window.Hls.Events.MANIFEST_PARSED, () => { seekToRoom(); play(); });
            hls.on(window.Hls.Events.ERROR, (_, data) => {
                if (!data.fatal) return;
                if (retries++ < 2) {
                    if (data.type === window.Hls.ErrorTypes.NETWORK_ERROR) hls?.startLoad();
                    else if (data.type === window.Hls.ErrorTypes.MEDIA_ERROR) hls?.recoverMediaError();
                    else status('Stream unavailable. Tap Retry.');
                } else status('Stream unavailable. Tap Retry.');
            });
        } else { mediaAvailable = true; audio.src = url; audio.load(); }
    }
    function synchronize() {
        if (disposed) return;
        const track = state.room?.track;
        const url = track && track.ready !== false ? safeURL(track.audioUrl) : '';
        if (url !== source) load(url);
        if (track?.ready === false) { status('Preparing the track…'); return; }
        if (!track) { status(''); return; }
        if (!state.joinedListening) { audio.pause(); status('Tap Start listening to enable audio'); return; }
        seekToRoom();
        if (readyToPlay()) play(); else { audio.pause(); if (state.room.playback.status !== 'playing') status('Paused'); }
        emit('progress', playbackPosition());
    }
    async function join() {
        if (disposed || state.stopped || state.connection !== 'connected' || !state.permissions.userId) return;
        state.joinedListening = true;
        if (state.room?.track) { synchronize(); await play(); }
        emit('listening');
    }
    function tick() {
        if (disposed) return;
        if (Date.now() - lastSync > 3000) { lastSync = Date.now(); if (readyToPlay()) { seekToRoom(); play(); } }
        emit('progress', playbackPosition());
        if (state.sleepUntil && Date.now() >= state.sleepUntil) {
            state.sleepUntil = 0; state.joinedListening = false; audio.pause(); emit('sleep-expired'); emit('listening');
            notify('Sleep timer finished. Listening stopped on this device.');
        }
        emit('tick');
    }
    const clock = setInterval(tick, 250);
    audio.addEventListener('loadedmetadata', synchronize);
    audio.addEventListener('canplay', () => { clearBuffering(); seekToRoom(); play(); });
    audio.addEventListener('seeked', () => { observedTime = audio.currentTime; clearBuffering(); play(); });
    audio.addEventListener('timeupdate', () => { if (!audio.paused && Math.abs(audio.currentTime - observedTime) > .05) clearBuffering(); observedTime = audio.currentTime; });
    audio.addEventListener('playing', () => status(''));
    audio.addEventListener('waiting', () => { if (readyToPlay()) status('Buffering…'); });
    audio.addEventListener('stalled', () => { if (readyToPlay() && audio.readyState < 3 && !audio.seeking) status('Waiting for the stream…'); });
    audio.addEventListener('ended', () => {
        // Track transitions belong to the server timer. Client-ended events must not race it.
        status('Waiting for the next track…');
    });
    audio.addEventListener('error', () => {
        if (!source || !mediaAvailable || disposed) return;
        status('Stream unavailable. Tap Retry.');
        if (retries++ < 2) {
            const currentGeneration = generation;
            retryTimer = setTimeout(() => {
                if (currentGeneration !== generation || !source || hls) return;
                audio.load(); play();
            }, 1800);
        }
    });
    audio.addEventListener('volumechange', () => { preferences.set('volume', audio.volume); preferences.set('music-muted', audio.muted); emit('volume'); });
    on('room', synchronize); on('clock', seekToRoom); on('listening', synchronize);
    on('transport-lost', () => { audio.pause(); status('Reconnecting to the room…'); });
    document.addEventListener('visibilitychange', () => { if (!document.hidden) synchronize(); });
    on('dispose', () => { disposed = true; clearInterval(clock); destroySource(); audio.pause(); });
    window.addEventListener('pageshow', e => { if (e.persisted) synchronize(); });
    return {
        join,
        retry() { if (source) { const url = source; source = ''; load(url); synchronize(); } },
        toggle() { if (!canControl() || !state.room?.track) return; if (!state.joinedListening) join(); send(state.room.playback.status === 'playing' ? 'pause' : 'resume'); },
        seek(seconds) { if (canControl()) send('seek', { positionSeconds: seconds }); },
        volume(value) { audio.volume = value; audio.muted = value === 0; },
        mute() { audio.muted = !audio.muted; if (!audio.muted && audio.volume === 0) audio.volume = .7; },
        getVolume() { return { volume: audio.volume, muted: audio.muted }; }
    };
}
