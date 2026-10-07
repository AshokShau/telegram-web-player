import { state, platform, telegram, on, haptic, connect, safeURL, formatTime } from './core.js?v=24';

export function createSessionScreen({ startListening }) {
    const screen = document.getElementById('session-notice');
    const title = document.getElementById('session-title');
    const description = document.getElementById('session-description');
    const status = document.getElementById('session-status');
    const steps = document.getElementById('session-steps');
    const primary = document.getElementById('session-primary');
    const secondary = document.getElementById('session-secondary');
    const hint = document.getElementById('session-hint');
    const app = document.getElementById('app');
    const profile = document.getElementById('session-profile');
    const profileName = document.getElementById('session-profile-name');
    const profileHandle = document.getElementById('session-profile-handle');
    const profilePhoto = document.getElementById('session-profile-photo');
    const profileInitial = document.getElementById('session-profile-initial');
    const trackPreview = document.getElementById('session-track');
    const trackTitle = document.getElementById('session-track-title');
    const trackMeta = document.getElementById('session-track-meta');
    const trackStatus = document.getElementById('session-track-status');
    const cover = document.getElementById('session-cover');
    const record = screen.querySelector('.session-record');
    let retryable = false;
    let admitted = false;
    let currentCode = 'connecting';
    const fromGroup = ['Return to the group this room belongs to.', 'Open its player button there, instead of a copied link.'];
    const states = {
        connecting: { title: 'Finding your\nlistening room.', description: 'Connecting to your Telegram room. Your music will be ready in a moment.', status: 'Connecting', loading: true },
        listening_required: { title: 'Your room is ready.\nTap to listen.', description: 'Tap Start listening to enable sound on this device. You’ll hear the music when your room plays.', status: 'Ready to listen', action: 'Start listening' },
        group_launch_required: { title: 'This room opens\nfrom its group.', description: 'This player belongs to a Telegram group. Open it inside that group to listen together.', status: 'Group access required', steps: fromGroup },
        group_context_mismatch: { title: 'Right link.\nDifferent chat.', description: 'You opened a link for another group. Return to the group it belongs to and open the player there.', status: 'Different chat', steps: fromGroup },
        group_context_missing: { title: 'Open this room\ninside its group.', description: 'This launch didn’t include the group it came from. Open a fresh player button inside the linked Telegram group.', status: 'Group launch needed', steps: fromGroup },
        chat_binding_required: { title: 'One quick setup.\nThen press play.', description: 'This group needs to set up its player once. Everyone in the group can use it afterward.', status: 'Group setup needed', steps: ['Return to the linked Telegram group.', 'On a fresh bot message, tap Verify group for player, then Open Web App Player.'] },
        chat_binding_unavailable: { title: 'Your room is\nout of reach.', description: 'We couldn’t confirm access to this room right now. Give it a moment, then try again.', status: 'Temporarily unavailable', retry: 'Try again' },
        telegram_authentication_required: { title: 'Reconnect through\nTelegram.', description: 'Your Telegram session could not be verified. Reopen the player from a fresh bot message.', status: 'Session unavailable', steps: ['Go back to your Telegram chat.', 'Open a fresh player button from SyncTune.'] },
        private_room_owner_required: { title: 'This is someone\nelse’s room.', description: 'Personal rooms belong to the account that opened them. Open your own player from your chat with SyncTune.', status: 'Personal room', steps: ['Return to your private chat with SyncTune.', 'Send /player and open your own room.'] },
        invalid_room_identity: { title: 'Choose your\nlistening room.', description: 'Open SyncTune from a player button in your Telegram group or private bot chat.', status: 'Room link needed', steps: ['Open your group or private chat with SyncTune.', 'Send /player and use its player button.'] },
        signed_room_mismatch: { title: 'Open a fresh\nroom link.', description: 'This link doesn’t match the room in your Telegram session. Open a new player button from the chat you want to join.', status: 'Room link changed', steps: fromGroup },
        telegram_required: { title: 'Good music.\nBetter together.', description: 'SyncTune rooms open inside Telegram. Use a player button in your group or private chat with the bot to join.', status: 'Open in Telegram', steps: ['Find SyncTune in your Telegram chat.', 'Send /player and open its player button.'] },
        session_in_use: { title: 'You’re already\nconnected elsewhere.', description: 'Your other tab or device can keep listening. Choose Use this session to switch here and stop the other session.', status: 'Another session is active', action: 'Use this session' },
        session_moved: { title: 'Your listening\nsession moved.', description: 'You switched to another tab or device. This session has stopped and will stay disconnected. Choose Use this session to switch back here.', status: 'Playing elsewhere', action: 'Use this session' },
        connection_unavailable: { title: 'Let’s get you\nconnected.', description: 'The room connection is taking longer than usual. We’re trying again, or you can retry now.', status: 'Reconnecting', retry: 'Try again' }
    };

    function setImage(container, value) {
        const url = safeURL(value);
        if (!url) { container.hidden = true; container.replaceChildren(); container.dataset.source = ''; container.dataset.failed = ''; return; }
        if (container.dataset.source !== url) {
            container.dataset.source = url; container.dataset.failed = '';
            const image = document.createElement('img'); image.alt = ''; image.src = url;
            container.replaceChildren(image);
        }
        container.hidden = container.dataset.failed === 'true';
    }
    profilePhoto.addEventListener('error', () => { profilePhoto.dataset.failed = 'true'; profilePhoto.hidden = true; profileInitial.hidden = false; }, true);
    cover.addEventListener('error', () => { cover.dataset.failed = 'true'; cover.hidden = true; record.removeAttribute('hidden'); }, true);
    function renderListeningContext() {
        if (currentCode !== 'listening_required' || !admitted || screen.hidden) return;
        const firstName = typeof state.user.first_name === 'string' ? state.user.first_name.trim() : '';
        const lastName = typeof state.user.last_name === 'string' ? state.user.last_name.trim() : '';
        const username = typeof state.user.username === 'string' ? state.user.username.trim() : '';
        const name = [firstName, lastName].filter(Boolean).join(' ') || 'Listener';
        profile.hidden = false;
        profileName.textContent = name; profileName.title = name;
        profileHandle.textContent = username ? `@${username}` : ''; profileHandle.hidden = !username;
        profileInitial.textContent = Array.from(firstName || name)[0].toUpperCase();
        setImage(profilePhoto, state.user.photo_url);
        profileInitial.hidden = !profilePhoto.hidden;
        if (firstName) {
            const greeting = document.createElement('span'); greeting.className = 'session-greeting';
            greeting.textContent = `Hi, ${firstName}.`; greeting.title = greeting.textContent;
            title.replaceChildren(greeting, document.createTextNode('\nLet’s listen.'));
        } else title.textContent = 'Your room is ready.\nTap to listen.';

        const track = state.room?.track;
        trackPreview.hidden = !track;
        const preparing = track?.ready === false;
        const playing = Boolean(track && !preparing && state.room.playback?.status === 'playing');
        description.textContent = !track ? 'Enable sound now. You’ll be ready when someone adds a track.'
            : preparing ? 'Your track is getting ready. Enable sound to hear it when it starts.'
            : playing ? 'Music is playing in your room. Tap Start listening to join in.'
            : 'Music is paused. Enable sound now to hear it when the room resumes.';
        if (track) {
            trackTitle.textContent = track.title || 'Untitled track'; trackTitle.title = trackTitle.textContent;
            trackMeta.textContent = [track.artist || (track.user ? `Added by ${track.user}` : track.platform), Number(track.duration) > 0 ? formatTime(track.duration) : ''].filter(Boolean).join(' · ');
            trackMeta.hidden = !trackMeta.textContent;
            trackStatus.textContent = preparing ? 'Preparing track' : playing ? 'Playing in your room' : 'Paused in your room';
            trackPreview.dataset.playing = String(playing);
        }
        setImage(cover, track?.thumbnail);
        record.toggleAttribute('hidden', !cover.hidden);
    }
    on('room', renderListeningContext);
    on('permissions', renderListeningContext);

    function show(code, message) {
        currentCode = code;
        const view = states[code] || { ...states.telegram_authentication_required, description: typeof message === 'string' && message ? message : states.telegram_authentication_required.description };
        retryable = Boolean(view.retry);
        title.textContent = view.title;
        description.textContent = view.description;
        status.textContent = view.status;
        screen.dataset.state = code === 'listening_required' ? 'listening' : view.loading ? 'connecting' : 'blocked';
        screen.setAttribute('aria-busy', String(Boolean(view.loading)));
        profile.hidden = true;
        trackPreview.hidden = true;
        cover.hidden = true;
        record.removeAttribute('hidden');
        steps.replaceChildren(...(view.steps || []).map(copy => { const item = document.createElement('li'); item.textContent = copy; return item; }));
        steps.hidden = !view.steps;
        primary.hidden = Boolean(view.loading);
        primary.disabled = false;
        primary.textContent = view.action || view.retry || (platform.isTelegramWebApp ? 'Back to Telegram' : 'Back to SyncTune');
        secondary.hidden = !view.retry && !view.action;
        secondary.textContent = platform.isTelegramWebApp ? 'Back to Telegram' : 'Back to SyncTune';
        hint.hidden = true;
        screen.hidden = false;
        renderListeningContext();
        app.hidden = true;
        document.body.classList.add('session-blocked');
        for (const node of document.body.children) if (node !== screen && node.tagName !== 'SCRIPT') node.inert = true;
        for (const dialog of document.querySelectorAll('dialog[open]')) dialog.close();
        if (!view.loading) { (code === 'listening_required' ? primary : title).focus({ preventScroll: true }); if (code !== 'listening_required') haptic('warning'); }
    }

    function back() {
        if (!platform.isTelegramWebApp) { location.assign('/'); return; }
        try {
            if (typeof telegram.close === 'function') { telegram.close(); return; }
        } catch { /* Let the user close the native window if this client cannot. */ }
        hint.textContent = 'Close this Mini App with Telegram’s close button, then return to your chat.';
        hint.hidden = false;
    }

    screen.addEventListener('click', event => {
        const button = event.target.closest('button[data-session-action]');
        if (!button || button.disabled) return;
        haptic('selection');
        if (button === primary && currentCode === 'listening_required') { startListening(); }
        else if (button === primary && ['session_in_use', 'session_moved'].includes(currentCode)) {
            state.sessionFailure = null;
            state.stopped = false;
            admitted = false;
            show('connecting');
            connect({ takeOver: true });
        }
        else if (button === primary && retryable) { primary.disabled = true; location.reload(); }
        else back();
    });
    on('back', () => { if (!screen.hidden) back(); });
    function reveal() {
        if (screen.hidden) return;
        screen.hidden = true;
        app.hidden = false;
        document.body.classList.remove('session-blocked');
        for (const node of document.body.children) if (node !== screen && node.tagName !== 'SCRIPT') node.inert = false;
        if (screen.contains(document.activeElement)) document.getElementById('workspace').focus({ preventScroll: true });
    }
    on('session-ready', () => {
        admitted = true;
        if (!state.joinedListening) show('listening_required');
        else reveal();
    });
    on('listening', () => {
        if (!admitted || state.stopped || state.sessionFailure) return;
        if (state.joinedListening && currentCode === 'listening_required') reveal();
        else if (!state.joinedListening && state.audioStatus === 'Tap Start listening to enable audio') show('listening_required');
    });
    on('authentication-failed', error => show(error.code === 'invalid_start_param' ? 'signed_room_mismatch' : error.code, error.message));
    on('session-ended', message => show('session_moved', message));
    on('connection', () => {
        if (admitted || state.stopped) return;
        if (state.connection === 'reconnecting') show('connection_unavailable');
        else if (['connecting', 'connected'].includes(state.connection)) show('connecting');
    });
    return { show };
}
