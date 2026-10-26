import { state, validPasscode, platform, telegram, on, haptic, connect, joinWithPassword, safeURL, formatTime } from './core.js?v=88';
import { setTrackArtwork } from './artwork.js?v=88';

export function createSessionScreen({ startListening }) {
    const screen = document.getElementById('session-notice');
    const title = document.getElementById('session-title');
    const description = document.getElementById('session-description');
    const status = document.getElementById('session-status');
    const steps = document.getElementById('session-steps');
    const primary = document.getElementById('session-primary');
    const primaryLabel = document.getElementById('session-primary-label');
    const primaryIcon = document.getElementById('session-primary-icon');
    const audioNote = document.getElementById('session-audio-note');
    const emptyPreview = document.getElementById('session-empty');
    const secondary = document.getElementById('session-secondary');
    const hint = document.getElementById('session-hint');
    const app = document.getElementById('app');
    const passwordForm = document.getElementById('partner-join-form');
    const passwordInput = document.getElementById('partner-join-password');
    const passwordError = document.getElementById('partner-join-error');
    const passwordButton = document.getElementById('partner-join-button');
    const passwordLabel = document.getElementById('partner-join-label');
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
    const writeNotice = document.getElementById('write-access-notice');
    let writePromptDismissed = false;
    let retryable = false;
    let admitted = false;
    let hadWriteAccess = false;
    let currentCode = 'connecting';
    function dismissWritePrompt() {
        writePromptDismissed = true;
        writeNotice.close();
        primary.focus({ preventScroll: true });
    }
    document.getElementById('write-access-dismiss').addEventListener('click', dismissWritePrompt);
    writeNotice.addEventListener('cancel', event => { event.preventDefault(); dismissWritePrompt(); });
    function syncWritePrompt() {
        const needed = state.permissions.userId && !state.permissions.allowsWriteToPM;
        const visible = currentCode === 'listening_required' && admitted && !screen.hidden;
        if (needed && visible && !writePromptDismissed) {
            if (!writeNotice.open) writeNotice.showModal();
        } else if (writeNotice.open) {
            writeNotice.close();
            if (visible) primary.focus({ preventScroll: true });
        }
    }
    const fromGroup = ['Return to the group this room belongs to.', 'Open its player button there, instead of a copied link.'];
    const states = {
        connecting: { title: 'Getting your\nmusic ready.', description: 'Connecting to your player and loading the latest queue.', status: 'Connecting', loading: true },
        listening_required: { title: 'The music’s\non pause.', description: 'Open the player to see what’s queued up.', status: 'Ready to listen', action: 'Go to player' },
        partner_password_required: { title: 'Listen\ntogether.', description: 'Enter the passcode your partner shared.', status: 'Private player' },
        group_launch_required: { title: 'This room opens\nfrom its group.', description: 'This player belongs to a Telegram group. Open it from that group to play music.', status: 'Group access required', steps: fromGroup },
        group_context_mismatch: { title: 'Right link.\nDifferent chat.', description: 'You opened a link for another group. Return to the group it belongs to and open the player there.', status: 'Different chat', steps: fromGroup },
        group_context_missing: { title: 'Open this room\ninside its group.', description: 'This launch didn’t include the group it came from. Open a fresh player button inside the linked Telegram group.', status: 'Group launch needed', steps: fromGroup },
        chat_binding_required: { title: 'One quick setup.\nThen press play.', description: 'This group needs to set up its player once. Everyone in the group can use it afterward.', status: 'Group setup needed', steps: ['Return to the linked Telegram group.', 'On a fresh bot message, tap Verify group for player, then Open Web App Player.'] },
        chat_binding_unavailable: { title: 'Your room is\nout of reach.', description: 'We couldn’t confirm access to this room right now. Give it a moment, then try again.', status: 'Temporarily unavailable', retry: 'Try again' },
        telegram_authentication_required: { title: 'Reconnect through\nTelegram.', description: 'Your Telegram session could not be verified. Reopen the player from a fresh bot message.', status: 'Session unavailable', steps: ['Go back to your Telegram chat.', 'Open a fresh player button from SyncTune.'] },
        private_room_owner_required: { title: 'This is someone\nelse’s room.', description: 'Personal rooms belong to the account that opened them. Open your own player from your chat with SyncTune.', status: 'Personal room', steps: ['Return to your private chat with SyncTune.', 'Send /player and open your own room.'] },
        partner_invite_expired: { title: 'Time for a\nfresh invite.', description: 'This partner invite expired or is no longer available. Ask the room owner for a new link.', status: 'Invite expired' },
        partner_invite_full: { title: 'This invite\nis full.', description: 'This partner invite has reached its user limit. Ask the room owner for a new link.', status: 'User limit reached' },
        invalid_room_identity: { title: 'Open your\nmusic player.', description: 'Open SyncTune from a player button in your Telegram group or private bot chat.', status: 'Room link needed', steps: ['Open your group or private chat with SyncTune.', 'Send /player and use its player button.'] },
        signed_room_mismatch: { title: 'Open a fresh\nroom link.', description: 'This link doesn’t match the room in your Telegram session. Open a new player button from the chat you want to join.', status: 'Room link changed', steps: fromGroup },
        telegram_required: { title: 'Good music.\nBetter together.', description: 'SyncTune rooms open inside Telegram. Use a player button in your group or private chat with the bot to join.', status: 'Open in Telegram', steps: ['Find SyncTune in your Telegram chat.', 'Send /player and open its player button.'] },
        session_in_use: { title: 'You’re already\nconnected elsewhere.', description: 'Your other tab or device can keep listening. Choose Switch to this device to switch here and stop the other session.', status: 'Another session is active', action: 'Switch to this device' },
        session_moved: { title: 'Your listening\nsession moved.', description: 'You switched to another tab or device. This session has stopped and will stay disconnected. Choose Switch to this device to switch back here.', status: 'Playing elsewhere', action: 'Switch to this device' },
        connection_unavailable: { title: 'Let’s get you\nconnected.', description: 'The room connection is taking longer than usual. We’re trying again, or you can retry now.', status: 'Reconnecting', retry: 'Try again' }
    };

    function setImage(container, value) {
        const url = safeURL(value);
        if (!url) { container.hidden = true; container.replaceChildren(); container.dataset.source = ''; container.dataset.failed = ''; return; }
        if (container.dataset.source !== url) {
            container.dataset.source = url; container.dataset.failed = '';
            const image = document.createElement('img'); image.alt = '';
            if (container === cover) setTrackArtwork(image, url); else image.src = url;
            container.replaceChildren(image);
        }
        container.hidden = container.dataset.failed === 'true';
    }
    profilePhoto.addEventListener('error', () => { profilePhoto.dataset.failed = 'true'; profilePhoto.hidden = true; profileInitial.hidden = false; }, true);
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
        const track = state.room?.track;
        const preparing = track?.ready === false;
        const playing = Boolean(track && !preparing && state.room.playback?.status === 'playing');
        const playback = !track ? 'idle' : preparing ? 'preparing' : playing ? 'playing' : 'paused';
        screen.dataset.playback = playback;
        trackPreview.hidden = !track;
        emptyPreview.hidden = Boolean(track);
        title.textContent = !track ? 'Find your\nnext track.'
            : preparing ? 'Your next track\nis on its way.'
            : playing ? 'One tap away\nfrom the music.' : 'The music’s\non pause.';
        description.textContent = !track ? 'Search, queue, and play.'
            : preparing ? 'We’re getting this track ready. You can browse the queue while you wait.'
            : playing ? 'Pick up the current track, right where everyone is.'
            : 'See what’s next in the queue, or return to this track in the player.';
        const needsWriteAccess = !state.permissions.allowsWriteToPM;
        primaryLabel.textContent = needsWriteAccess ? playing ? 'Continue listening' : 'Browse music' : !track ? 'Explore music' : playing ? 'Listen now' : 'Go to player';
        primaryIcon.setAttribute('href', `/static/assets/icons.svg?v=88#${!track ? 'search' : playing ? 'volume' : 'music'}`);
        audioNote.textContent = playing ? 'Playback stays in sync with your Telegram chat.' : !track ? 'Search for something you love.' : 'Music will play here when the track starts.';
        audioNote.hidden = false;
        if (track) {
            trackTitle.textContent = track.title || 'Untitled track'; trackTitle.title = trackTitle.textContent;
            trackMeta.textContent = [track.artist || (track.user ? `Added by ${track.user}` : track.platform), Number(track.duration) > 0 ? formatTime(track.duration) : ''].filter(Boolean).join(' · ');
            trackMeta.hidden = !trackMeta.textContent;
            trackStatus.textContent = preparing ? 'Getting ready…' : playing ? 'Now playing' : 'Paused';
            trackPreview.dataset.playing = String(playing);
        }
        setImage(cover, track?.thumbnail);
        record.toggleAttribute('hidden', !cover.hidden);
        syncWritePrompt();
    }
    on('room', renderListeningContext);
    on('permissions', () => {
        const revoked = state.permissions.userId && hadWriteAccess && !state.permissions.allowsWriteToPM;
        if (revoked) writePromptDismissed = false;
        if (state.permissions.userId) hadWriteAccess = state.permissions.allowsWriteToPM;
        if (admitted && revoked && screen.hidden) show('listening_required');
        else renderListeningContext();
    });

    function show(code, message) {
        currentCode = code;
        const view = states[code] || { ...states.telegram_authentication_required, description: typeof message === 'string' && message ? message : states.telegram_authentication_required.description };
        retryable = Boolean(view.retry);
        title.textContent = view.title;
        description.textContent = view.description;
        status.textContent = view.status;
        const passwordRequired = code === 'partner_password_required';
        screen.classList.toggle('passcode-entry', passwordRequired);
        passwordForm.hidden = !passwordRequired;
        screen.dataset.state = code === 'listening_required' ? 'listening' : view.loading ? 'connecting' : 'blocked';
        screen.setAttribute('aria-busy', String(Boolean(view.loading)));
        profile.hidden = true;
        emptyPreview.hidden = true;
        audioNote.hidden = true;
        delete screen.dataset.playback;
        trackPreview.hidden = true;
        cover.hidden = true;
        record.removeAttribute('hidden');
        steps.replaceChildren(...(view.steps || []).map(copy => { const item = document.createElement('li'); item.textContent = copy; return item; }));
        steps.hidden = !view.steps;
        primary.hidden = Boolean(view.loading) || passwordRequired;
        primary.disabled = false;
        primaryLabel.textContent = view.action || view.retry || (platform.isTelegramWebApp ? 'Back to Telegram' : 'Back to SyncTune');
        primaryIcon.setAttribute('href', `/static/assets/icons.svg?v=88#${view.retry ? 'restart' : view.action ? 'volume' : 'logout'}`);
        secondary.hidden = !view.retry && !view.action && !passwordRequired;
        secondary.textContent = platform.isTelegramWebApp ? 'Back to Telegram' : 'Back to SyncTune';
        hint.hidden = true;
        screen.hidden = false;
        app.hidden = true;
        document.body.classList.add('session-blocked');
        for (const node of document.body.children) if (node !== screen && node.tagName !== 'SCRIPT') node.inert = true;
        for (const dialog of document.querySelectorAll('dialog[open]')) dialog.close();
        renderListeningContext();
        if (!view.loading) { (code === 'listening_required' ? writeNotice.open ? writeNotice.querySelector('[data-action="write-access"]') : primary : title).focus({ preventScroll: true }); if (code !== 'listening_required' && !passwordRequired) haptic('warning'); }
    }

    function joining(value) {
        passwordForm.setAttribute('aria-busy', String(value));
        passwordInput.readOnly = value;
        passwordButton.disabled = value || !validPasscode(passwordInput.value);
        passwordLabel.textContent = value ? 'Joining…' : 'Join player';
    }
    passwordInput.addEventListener('input', () => {
        passwordInput.setAttribute('aria-invalid', 'false'); passwordError.hidden = true;
        joining(false);
    });
    passwordForm.addEventListener('submit', event => {
        event.preventDefault();
        if (passwordButton.disabled) return;
        if (!validPasscode(passwordInput.value)) { passwordError.textContent = 'Use 4–64 characters without leading or trailing spaces.'; passwordError.hidden = false; return; }
        passwordError.hidden = true;
        if (joinWithPassword(passwordInput.value)) joining(true);
    });
    on('partner-password-required', error => {
        show('partner_password_required');
        joining(false);
        passwordInput.setAttribute('aria-invalid', String(error.invalid));
        passwordError.textContent = error.invalid ? error.message : '';
        passwordError.hidden = !error.invalid;
        if (error.invalid) haptic('error');
        passwordInput.focus();
    });
    on('request-error', error => {
        if (error.type !== 'join' || currentCode !== 'partner_password_required') return;
        joining(false);
        passwordError.textContent = String(error.message); passwordError.hidden = false;
    });

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
    on('back', () => { if (writeNotice.open) dismissWritePrompt(); else if (!screen.hidden) back(); });
    function reveal() {
        if (screen.hidden) return;
        screen.hidden = true;
        if (writeNotice.open) writeNotice.close();
        app.hidden = false;
        document.body.classList.remove('session-blocked');
        for (const node of document.body.children) if (node !== screen && node.tagName !== 'SCRIPT') node.inert = false;
        if (screen.contains(document.activeElement)) document.getElementById('workspace').focus({ preventScroll: true });
    }
    on('session-ready', () => {
        passwordForm.reset();
        admitted = true;
        if (!state.joinedListening) show('listening_required');
        else reveal();
    });
    on('listening', () => {
        if (!admitted || state.stopped || state.sessionFailure) return;
        if (state.joinedListening && currentCode === 'listening_required') reveal();
        else if (!state.joinedListening && state.audioBlocked) show('listening_required');
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
