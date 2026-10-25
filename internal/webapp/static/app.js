import { state, platform, telegram, hasSession, preferences, on, notify, haptic, send, connect, initializePlatform, requestWriteAccess, openInBrowser, searchAPI, toggleFullscreen, isFullscreen, canControl, canPlay, canManageSettings, formatTime, safeURL } from './js/core.js?v=86';
import { createPlayback } from './js/playback.js?v=86';
import { createVoice } from './js/voice.js?v=86';
import { createChoiceScreens } from './js/choices.js?v=86';
import { createSessionScreen } from './js/session.js?v=86';
import { createAtmosphere, setTrackArtwork, fallbackArtwork } from './js/artwork.js?v=86';
import { createLyrics } from './js/lyrics.js?v=86';
import { element, icon, actionButton, empty, trackCount, normalizeSong, avatar, loadingState } from './js/ui.js?v=86';
import { createTrackRenderer } from './js/tracks.js?v=86';
import { createAppearance } from './js/appearance.js?v=86';
import { createActionDialog } from './js/dialog.js?v=86';

const ids = ['app','page-title','workspace','queue-mount','queue-rail','connection-dot','connection-label','connection-banner','connection-detail','listener-count','header-avatar','home-artwork','home-track-status','home-track-title','home-track-artist','home-requester','home-recommendations','home-recommendation-tracks','home-playlists-meta','home-history-meta','home-listeners-meta','home-main-action','home-play-action','home-play-icon','home-play-label','home-queue-action','home-queue-count','recent-count','mix-button','mix-results','recent-tracks','library-recent-tracks','search-form','search-input','search-clear','search-summary','search-results','library-message','library-playlists','sidebar-playlists','playlist-detail','playlist-detail-title','playlist-detail-meta','playlist-tracks','account-name','account-username','account-id','account-message-state','account-message-allow','account-connection','profile-avatar','profile-name','profile-handle','profile-role','partner-summary','partner-password-status','write-access-notice','settings-title','fullscreen-button','fullscreen-hint','repeat-select','autoplay-toggle','sleep-select','sleep-status','profile-room-id','profile-connection','room-admin-settings','chat-enabled-toggle','chat-cooldown','chat-cooldown-labels','chat-cooldown-value','group-chat-settings','chat-play-mode','chat-admin-mode','chat-cmd-delete','chat-autoplay','chat-auto-delete','chat-auto-delete-minutes','chat-silent-mode','chat-language','chat-silent-permission','chat-settings-status','chat-language-row','room-listeners','queue-empty','queue-now-group','queue-next-group','queue-count','queue-current','queue-tracks','player','player-artwork','player-title','player-artist','player-status','player-mix','player-mix-title','play-button','play-icon','repeat-button','repeat-count','player-view-button','player-view-icon','player-seek','player-elapsed','player-duration','player-volume','volume-button','volume-icon','listen-banner','listeners-panel','listeners-summary','listener-participants','chat-self-avatar','chat-panel','chat-unread','chat-state','chat-messages','chat-form','chat-input','chat-send','chat-composer-state','voice-panel','voice-header-label','voice-connection','voice-admin-controls','voice-settings-button','voice-settings','voice-noise-toggle','voice-join-policy','voice-settings-note','voice-participants','voice-join','voice-mic','voice-mic-icon','voice-mic-label','voice-speaker','voice-leave','voice-mini','voice-mini-label','voice-mini-mic-icon','action-dialog','dialog-form','dialog-title','dialog-description','dialog-input-label','dialog-input','dialog-select-label','dialog-select','dialog-submit','session-notice','session-title','session-description','toast','music-audio','voice-audios'];
const dom = Object.fromEntries(ids.map(id => [id, document.getElementById(id)]));
for (const [id, element] of Object.entries(dom)) if (!element) throw new Error(`Missing application element: ${id}`);
const appearance = createAppearance({ telegram, native: platform.isTelegramWebApp, preferences });
on('dispose', () => appearance.dispose());
const playback = createPlayback(dom['music-audio']);
const lyrics = createLyrics({ player: dom.player, toggle: document.getElementById('lyrics-toggle'), stage: document.getElementById('lyrics-stage'), panel: document.getElementById('lyrics-panel'), status: document.getElementById('lyrics-status'), lines: document.getElementById('lyrics-lines'), follow: document.getElementById('lyrics-follow'), preview: document.getElementById('lyrics-preview'), previewLine: document.getElementById('lyrics-preview-line'), headerTitle: document.getElementById('player-heading'), headerSubtitle: document.getElementById('player-lyrics-artist') });
const atmosphere = createAtmosphere(document.getElementById('player-atmosphere'), document.getElementById('player-backdrop'));
const voice = createVoice(dom['voice-audios']);
const session = createSessionScreen({ startListening: () => { playback.join(); } });
const choices = createChoiceScreens(document.querySelectorAll('select'), { nativeBack: platform.supportsNativeBack, onOpen: updateBackButton, onClose: updateBackButton });
choices.connect('sleep-select', document.querySelector('[data-action="sleep-focus"]'));
const { renderTracks, trackRegistry } = createTrackRenderer(dom);
const renderKeys = new Map();
function text(id, value) { if (dom[id].textContent !== String(value)) dom[id].textContent = value; }
function setIcon(id, name) { dom[id].setAttribute('href', `/static/assets/icons.svg?v=86#${name}`); }
function artwork(img, url) { setTrackArtwork(img, safeURL(url, fallbackArtwork)); }
function changed(key, value) { const next = JSON.stringify(value); if (renderKeys.get(key) === next) return false; renderKeys.set(key, next); return true; }
function renderBotProfile() {
    const profile = state.botProfile;
    if (!profile || typeof profile !== 'object') return;
    const name = typeof profile.name === 'string' && profile.name.trim() ? profile.name.trim() : 'SyncTune';
    const username = typeof profile.username === 'string' && /^[a-zA-Z0-9_]{5,32}$/.test(profile.username) ? profile.username : '';
    const count = Number(profile.monthlyActiveUsers);
    for (const node of document.querySelectorAll('[data-bot-name]')) { node.textContent = name; node.title = name; }
    for (const node of document.querySelectorAll('[data-bot-active]')) {
        node.hidden = !Number.isSafeInteger(count) || count <= 0;
        node.textContent = node.hidden ? '' : `${new Intl.NumberFormat().format(count)} monthly users`;
    }
    for (const link of document.querySelectorAll('[data-bot-link]')) {
        link.hidden = !username; link.textContent = username ? `@${username}` : '';
        if (username) link.href = `https://t.me/${username}`; else link.removeAttribute('href');
    }
    const photo = safeURL(profile.photoUrl);
    for (const container of document.querySelectorAll('[data-bot-photo]')) {
        const fallback = container.previousElementSibling;
        if (container.dataset.source !== photo) {
            container.dataset.source = photo; container.dataset.failed = '';
            container.replaceChildren();
            if (photo) {
                const image = element('img', { src: photo, alt: '' });
                image.onerror = () => { container.dataset.failed = 'true'; container.hidden = true; fallback.removeAttribute('hidden'); };
                container.append(image);
            }
        }
        container.hidden = !photo || container.dataset.failed === 'true';
        fallback.toggleAttribute('hidden', !container.hidden);
    }
}
on('bot-profile', () => { renderBotProfile(); renderPartnerInvite(); });
function participant(user, { voiceParticipant = false } = {}) {
    const name = [user.firstName, user.lastName].filter(Boolean).join(' ') || 'Listener';
    const image = element('div', { class: 'avatar' }); avatar(image, user);
    let description = user.isAdmin ? 'Room admin' : 'Listening';
    if (voiceParticipant) description = user.allowedToSpeak === false ? (user.isAdminMuted ? 'Muted by admin' : 'Listening only') : user.isSelfMuted ? 'Muted' : user.isSpeaking ? 'Speaking' : 'Listening';
    const row = element('div', { class: `participant${user.isSpeaking ? ' is-speaking' : ''}`, 'data-user': user.userId }, [image, element('div', {}, [element('strong', { text: `${name}${user.userId === state.permissions.userId ? ' · You' : ''}` }), element('p', { text: description })])]);
    if (voiceParticipant && state.permissions.isAdmin && user.userId !== state.permissions.userId) row.append(actionButton('admin-mute', user.isAdminMuted ? 'Remove admin mute' : 'Mute participant', user.isAdminMuted ? 'mic' : 'mic-off', { 'data-user': user.userId, 'data-muted': !user.isAdminMuted }));
    return row;
}

const wideLayout = matchMedia('(min-width: 1100px)');
const viewTitles = new Map([['home', 'SyncTune'], ['search', 'Search'], ['queue', 'Queue'], ['library', 'Library'], ['profile', 'Settings'], ['settings', 'Settings'], ['mix', 'Similar tracks'], ['history', 'Recently played']]);
const settingsTitles = new Map([['account', 'Account'], ['playback', 'Playback'], ['appearance', 'Appearance'], ['room', 'Room settings'], ['partner', 'Partner access']]);
let settingsSection = 'playback';
function openSettings(section) {
    if (!settingsTitles.has(section)) return;
    settingsSection = section;
    text('settings-title', settingsTitles.get(section));
    for (const group of document.querySelectorAll('[data-settings-section]')) group.hidden = group.dataset.settingsSection !== section;
    if (section === 'room' && state.connection === 'connected') send('get_chat_settings');
    navigate('settings');
}
let expandedFocus = null;
let chatFocus = null;
let voiceFocus = null;
let listenersFocus = null;
function arrangeQueue() {
    if (wideLayout.matches) {
        dom['app'].insertBefore(dom['queue-rail'], dom['player']);
        if (state.view === 'queue') navigate('home');
    } else dom['queue-mount'].append(dom['queue-rail']);
}
function navigate(view, { focus = true } = {}) {
    const title = view === 'settings' ? settingsTitles.get(settingsSection) : viewTitles.get(view);
    if (!title) return;
    choices.close(); closeListeners(); closeChat(false); closeVoice(false); collapsePlayer();
    if (view === 'queue' && wideLayout.matches) { dom['queue-rail'].focus(); return; }
    state.view = view; text('page-title', title);
    document.body.dataset.view = view;
    for (const section of document.querySelectorAll('.workspace-view')) section.hidden = section.id !== `${view}-view`;
    const selectedView = view === 'mix' || view === 'history' ? 'home' : view === 'settings' ? 'profile' : view;
    for (const button of document.querySelectorAll('.primary-nav [data-nav], .mobile-nav [data-nav], .sidebar-bottom [data-nav]')) {
        if (button.dataset.nav === selectedView) button.setAttribute('aria-current', 'page'); else button.removeAttribute('aria-current');
    }
    dom['workspace'].scrollTop = 0;
    if (focus) dom['workspace'].focus({ preventScroll: true });
    if (view === 'search') dom['search-input'].focus({ preventScroll: true });
    if (view === 'library' && state.libraryStatus === 'idle') loadLibrary();
    ensureMix();
    updateBackButton();
}
function updateBackButton() {
    document.body.classList.toggle('conversation-open', state.chat.open || state.voice.open);
    for (const panel of ['chat', 'voice']) for (const button of document.querySelectorAll(`[data-action="open-${panel}"]`)) button.setAttribute('aria-expanded', String(state[panel].open));
    document.getElementById('workspace-back').hidden = platform.supportsNativeBack || !['settings', 'mix', 'history'].includes(state.view);
    for (const button of document.querySelectorAll('.browser-back')) button.hidden = platform.supportsNativeBack;
    if (!platform.supportsNativeBack) return;
    if (choices.isOpen() || dom['action-dialog'].open || state.expandedPlayer || state.view !== 'home' || state.chat.open || state.voice.open || !dom['listeners-panel'].hidden) telegram.BackButton?.show(); else telegram.BackButton?.hide();
}
function expandPlayer(trigger) {
    if (state.expandedPlayer) return;
    choices.close(); closeListeners();
    expandedFocus = trigger instanceof HTMLElement ? trigger : document.activeElement; state.expandedPlayer = true; dom['player'].classList.add('is-expanded');
    for (const node of document.querySelectorAll('.topbar,.sidebar,#workspace,#queue-rail,.mobile-nav')) node.inert = true;
    dom['player'].setAttribute('tabindex', '-1'); dom['player'].focus(); renderPlayerView(); renderListening(); updateBackButton();
}
function collapsePlayer() {
    if (!state.expandedPlayer) return;
    choices.close();
    state.expandedPlayer = false; dom['player'].classList.remove('is-expanded'); dom['player'].removeAttribute('tabindex');
    for (const node of document.querySelectorAll('.topbar,.sidebar,#workspace,#queue-rail,.mobile-nav')) node.inert = false;
    if (expandedFocus?.isConnected) expandedFocus.focus({ preventScroll: true }); renderPlayerView(); renderListening(); updateBackButton();
}
function renderPlayerView() {
    lyrics.viewChanged();
    const expanded = state.expandedPlayer;
    atmosphere.setExpanded(expanded);
    dom['player-view-button'].setAttribute('aria-label', expanded ? 'Minimize player' : 'Expand player');
    dom['player-view-button'].setAttribute('aria-expanded', String(expanded));
    setIcon('player-view-icon', expanded ? 'down' : 'expand');
    dom['player-seek'].tabIndex = !expanded && matchMedia('(max-width: 699px)').matches ? -1 : 0;
}
function openListeners() {
    if (!dom['listeners-panel'].hidden) return;
    closeChat(false); closeVoice(false);
    listenersFocus = document.activeElement; dom['listeners-panel'].hidden = false;
    document.body.classList.add('listeners-open');
    updateBackButton(); dom['listeners-panel'].focus({ preventScroll: true });
}
function closeListeners() {
    if (dom['listeners-panel'].hidden) return;
    dom['listeners-panel'].hidden = true; document.body.classList.remove('listeners-open');
    if (listenersFocus?.isConnected) listenersFocus.focus({ preventScroll: true });
    updateBackButton();
}
function openChat() { closeListeners(); closeVoice(false); chatFocus = document.activeElement; state.chat.open = true; state.chat.unread = 0; dom['chat-panel'].hidden = false; renderChatHistory(); renderChatStatus(); renderVoiceLocal(); dom['chat-input'].focus(); updateBackButton(); }
function closeChat(restoreFocus = true) { if (!state.chat.open) return; state.chat.open = false; dom['chat-panel'].hidden = true; renderVoiceLocal(); if (restoreFocus) chatFocus?.focus({ preventScroll: true }); updateBackButton(); }
function openVoice() { closeListeners(); closeChat(false); voiceFocus = document.activeElement; state.voice.open = true; dom['voice-panel'].hidden = false; dom['voice-panel'].querySelector('button:not([hidden])')?.focus(); renderVoiceLocal(); updateBackButton(); }
function closeVoice(restoreFocus = true) { if (!state.voice.open) return; state.voice.open = false; dom['voice-panel'].hidden = true; renderVoiceLocal(); if (restoreFocus) voiceFocus?.focus({ preventScroll: true }); updateBackButton(); }
wideLayout.addEventListener('change', arrangeQueue);
matchMedia('(max-width: 699px)').addEventListener('change', renderPlayerView);
on('back', () => {
    if (!dom['session-notice'].hidden) return;
    if (choices.close(true)) return;
    if (dom['action-dialog'].open) dom['action-dialog'].close();
    else if (!dom['listeners-panel'].hidden) closeListeners(); else if (state.voice.open) closeVoice(); else if (state.chat.open) closeChat(); else if (state.expandedPlayer) collapsePlayer(); else navigate(state.view === 'settings' ? 'profile' : 'home');
});

function renderConnection() {
    const labels = { idle: 'Offline', connecting: 'Connecting', connected: 'Connected', reconnecting: 'Reconnecting', closed: 'Disconnected' };
    const label = labels[state.connection]; text('connection-label', label); text('account-connection', label); text('profile-connection', label);
    dom['connection-dot'].classList.toggle('connected', state.connection === 'connected');
    dom['connection-banner'].hidden = state.connection === 'connected' || !hasSession;
    text('connection-detail', 'Connection lost. Your music will sync again when you reconnect.');
    renderPermissions(); renderChatStatus(); renderVoiceLocal(); ensureMix();
}
const slowModePresets = [0, 2, 4, 7, 9, 15, 30, 60];
let slowModeSteps = [...slowModePresets];
function slowModeLabel(seconds) {
    return !seconds ? 'Off' : `${seconds}s`;
}
function previewSlowMode() {
    const seconds = slowModeSteps[Number(dom['chat-cooldown'].value)];
    text('chat-cooldown-value', slowModeLabel(seconds));
    const description = !seconds ? 'Off' : seconds >= 60 && seconds % 60 === 0 ? `${seconds / 60} ${seconds === 60 ? 'minute' : 'minutes'}` : `${seconds} seconds`;
    dom['chat-cooldown'].setAttribute('aria-valuetext', description);
    for (const [index, label] of [...dom['chat-cooldown-labels'].children].entries()) label.classList.toggle('is-selected', index === Number(dom['chat-cooldown'].value));
}
function renderSlowMode(seconds) {
    seconds = Math.max(0, Math.min(60, seconds));
    const steps = slowModePresets.includes(seconds) ? slowModePresets : [...slowModePresets, seconds].sort((a, b) => a - b);
    if (dom['chat-cooldown-labels'].children.length !== steps.length || steps.some((value, index) => value !== slowModeSteps[index])) {
        dom['chat-cooldown-labels'].replaceChildren(...steps.map((value, index) => {
            const label = element('span', { text: slowModeLabel(value) });
            label.style.left = `${index / (steps.length - 1) * 100}%`; return label;
        }));
    }
    slowModeSteps = steps;
    dom['chat-cooldown'].max = String(steps.length - 1); dom['chat-cooldown'].value = String(steps.indexOf(seconds));
    previewSlowMode();
}
let chatSettingsRequest = null;
function renderChatSettings() {
    const room = state.room, settings = room?.chatSettings, group = Number(state.roomId) < 0;
    dom['group-chat-settings'].hidden = !group; dom['chat-language-row'].hidden = !group;
    const editable = canManageSettings() && (!group || Boolean(settings));
    const values = { chatEnabled: Boolean(room?.chatEnabled), chatCooldown: room?.chatCooldown || 0,
        restrictedPlay: Boolean(settings?.restrictedPlay), adminMode: settings?.adminMode || 'everyone',
        cmdDelete: Boolean(settings?.cmdDelete), autoplay: Boolean(room?.autoplay), autoDelete: Boolean(settings?.autoDelete),
        autoDeleteMinutes: settings?.autoDeleteMinutes || 2, silentMode: Boolean(settings?.silentMode) };
    for (const [id, [field]] of chatSettingControls) {
        const node = dom[id];
        node.disabled = Boolean(chatSettingsRequest) || !editable || (field === 'silentMode' && !state.permissions.canChangeSilentMode) || (field === 'autoplay' && !room?.track);
        if (chatSettingsRequest) continue;
        const value = values[field];
        if (field === 'chatCooldown') { renderSlowMode(value); continue; }
        if (node.type === 'checkbox') node.checked = value;
        else node.value = field === 'restrictedPlay' ? value ? 'admins' : 'everyone' : String(value);
    }
    dom['chat-cooldown'].closest('.slow-mode-control').setAttribute('aria-disabled', String(dom['chat-cooldown'].disabled));
    dom['chat-silent-permission'].hidden = !group || Boolean(state.permissions.canChangeSilentMode);
    const status = chatSettingsRequest ? 'Saving…' : group && !settings ? 'Settings are unavailable. Reopen Room settings to try again.' : '';
    text('chat-settings-status', status); dom['chat-settings-status'].hidden = !status;
    text('chat-language', settings?.language && settings.language !== 'en' ? settings.language : 'English');
    choices.refresh();
}
function renderPermissions() {
    const permissions = state.permissions;
    renderPartnerInvite();
    text('profile-role', permissions.isAdmin ? 'Room admin' : permissions.isAuth ? 'Authorized user' : 'Listener');
    for (const node of document.querySelectorAll('[data-permission]')) node.disabled = node.dataset.permission === 'settings' ? !canManageSettings() : node.dataset.permission === 'control' ? !canControl() : !canPlay();
    for (const node of dom['player'].querySelectorAll('[data-permission]')) node.disabled = !(node.dataset.permission === 'settings' ? canManageSettings() : canControl()) || !state.room?.track;
    dom['room-admin-settings'].hidden = !permissions.isAdmin;
    dom['voice-admin-controls'].hidden = !permissions.isAdmin;
    dom['write-access-notice'].hidden = !permissions.userId || permissions.allowsWriteToPM;
    dom['account-message-state'].hidden = !permissions.allowsWriteToPM;
    dom['account-message-allow'].hidden = permissions.allowsWriteToPM;
    document.querySelector('[data-action="stop-room"]').disabled = !canControl() || !state.room?.track;
    document.querySelector('[data-action="clear-queue"]').disabled = !canControl() || !state.room?.queue?.length;
    dom['autoplay-toggle'].disabled = !canManageSettings() || !state.room?.track;
    dom['home-play-action'].disabled = !canControl() || !state.room?.track || state.room.track.ready === false;
    renderChatSettings();
    renderQueue(); renderSearch(); renderMix(); renderHistory(); renderPlaylistDetail(); renderVoiceState(); renderChatStatus(); choices.refresh();
}
function renderRoom() {
    const room = state.room;
    if (!room) return;
    const track = room.track;
    const home = document.getElementById('home-view'), toolsTitle = document.getElementById('home-tools-title');
    if (!track && dom['home-recommendations'].nextElementSibling !== toolsTitle) home.insertBefore(dom['home-recommendations'], toolsTitle);
    else if (track && home.lastElementChild !== dom['home-recommendations']) home.append(dom['home-recommendations']);
    document.querySelector('.player-title-button').dataset.action = document.querySelector('.player-artwork').dataset.action = track ? 'expand-player' : 'find-music';
    document.querySelector('.player-artwork').setAttribute('aria-label', track ? 'Expand player' : 'Find music');
    text('listener-count', `${room.listeners?.length || 0} listening`);
    text('home-listeners-meta', room.listeners?.length ? `${room.listeners.length} listening` : "See who's listening");
    text('profile-room-id', room.roomId);
    dom['home-track-title'].closest('.now-feature').classList.toggle('is-empty', !track);
    dom['home-track-status'].hidden = !track; dom['home-requester'].hidden = !track;
    text('home-track-status', track ? track.ready === false ? 'Preparing…' : room.playback.status === 'playing' ? 'Playing' : 'Paused' : '');
    text('home-track-title', track?.title || 'What’s your next play?');
    text('home-track-artist', track?.artist || (track ? 'Music' : 'Pick a song below, or find a favorite.'));
    text('home-requester', track ? `Added by ${track.user || 'a listener'}` : '');
    artwork(dom['home-artwork'], track?.thumbnail); artwork(dom['player-artwork'], track?.thumbnail);
    atmosphere.setArtwork(safeURL(track?.thumbnail, fallbackArtwork));
    text('player-title', track?.title || 'Pick a song'); text('player-artist', track?.artist || (track ? '' : 'Tap to explore'));
    dom['home-artwork'].parentElement.disabled = !track;
    const main = dom['home-main-action'];
    main.hidden = Boolean(track);
    dom['home-play-action'].hidden = !track;
    dom['home-play-action'].disabled = !track || track.ready === false || !canControl();
    dom['home-queue-action'].hidden = !track;
    text('home-queue-count', room.queue?.length || 0);
    const playing = room.playback.status === 'playing';
    setIcon('home-play-icon', playing ? 'pause' : 'play');
    text('home-play-label', playing ? 'Pause' : 'Resume');
    dom['home-play-action'].setAttribute('aria-label', playing ? 'Pause room playback' : 'Play room playback');
    setIcon('play-icon', playing ? 'pause' : 'play'); dom['play-button'].setAttribute('aria-label', playing ? 'Pause room playback' : 'Play room playback');
    dom['repeat-button'].setAttribute('aria-pressed', String(room.loop > 0)); dom['repeat-button'].setAttribute('aria-label', room.loop ? `Repeat current track: ${room.loop} repeats remaining` : 'Repeat current track');
    text('repeat-count', room.loop || ''); dom['repeat-count'].hidden = !room.loop;
    dom['repeat-button'].title = room.loop ? `${room.loop} repeats remaining` : 'Repeat current track';
    text('player-duration', formatTime(track?.duration)); dom['player-seek'].max = track?.duration || 0;
    // Preserve valid non-preset values received from Telegram controls.
    const loop = String(room.loop || 0);
    if (![...dom['repeat-select'].options].some(o => o.value === loop)) dom['repeat-select'].append(element('option', { value: loop, text: `${loop} times` }));
    dom['repeat-select'].value = loop; dom['autoplay-toggle'].checked = room.autoplay;
    renderChatSettings();
    if (changed('listeners', room.listeners)) {
        for (const id of ['room-listeners', 'listener-participants']) dom[id].replaceChildren(...(room.listeners?.length ? room.listeners.map(user => {
            const row = participant(user);
            if (id === 'listener-participants') row.setAttribute('role', 'listitem');
            return row;
        }) : [empty('No listeners yet', '', 'user', true)]));
        const count = room.listeners?.length || 0;
        dom['listener-participants'].setAttribute('role', count ? 'list' : 'group');
        text('listeners-summary', `${count} ${count === 1 ? 'listener' : 'listeners'}`);
    }
    renderQueue(); renderPermissions(); renderListening(); renderPlayerStatus(); choices.refresh();
    ensureMix();
}
function renderQueue() {
    const room = state.room;
    const queue = room?.queue || [];
    const isEmpty = Boolean(room && !room.track && !queue.length);
    dom['queue-empty'].hidden = !isEmpty;
    dom['queue-rail'].classList.toggle('is-empty', isEmpty);
    dom['queue-now-group'].hidden = Boolean(room && !room.track);
    dom['queue-next-group'].hidden = !room || isEmpty;
    dom['queue-count'].hidden = !queue.length;
    text('queue-count', queue.length);
    document.querySelector('[data-action="clear-queue"]').hidden = !queue.length;
    if (!room) { dom['queue-current'].replaceChildren(loadingState('Loading queue', 1)); return; }
    if (room.track) renderTracks('queue-current', [room.track], { current: true });
    if (queue.length) renderTracks('queue-tracks', queue, { queue: true });
    else if (!dom['queue-tracks'].querySelector('.empty-state')) dom['queue-tracks'].replaceChildren(empty('Add a song to keep the music going.', '', 'queue', true));
}

let seeking = false;
function renderProgress(position) {
    if (seeking) return;
    dom['player-seek'].value = position; text('player-elapsed', formatTime(position));
    const duration = Number(dom['player-seek'].max);
    const percent = duration > 0 ? Math.max(0, Math.min(100, position / duration * 100)) : 0;
    dom['player-seek'].style.setProperty('--fill', `${percent}%`);
    dom['player'].style.setProperty('--playback-progress', String(percent));
    dom['player-seek'].setAttribute('aria-valuetext', `${formatTime(position)} of ${formatTime(duration)}`);
}
function renderListening() { dom['listen-banner'].hidden = !hasSession || state.joinedListening || state.stopped || state.expandedPlayer; }
function renderPlayerStatus() {
    const value = state.audioStatus === 'Paused' ? '' : state.audioStatus;
    const buffering = Boolean(state.room?.track && (state.room.track.ready === false || ['Buffering…', 'Loading the track…'].includes(value)));
    dom['player'].dataset.buffering = String(buffering);
    dom['player'].dataset.playback = state.room?.track ? state.room.playback.status : 'idle';
    dom['play-button'].setAttribute('aria-busy', String(buffering));
    dom['player'].dataset.notice = String(Boolean(value && !state.audioBlocked));
    if (value.includes('Retry')) dom['player-status'].replaceChildren(element('button', { class: 'text-button', text: 'Can’t play · Retry', 'aria-label': value, 'data-action': 'retry-audio' }));
    else text('player-status', value);
}
function renderVolume() { const { volume, muted } = playback.getVolume(); dom['player-volume'].value = volume; dom['player-volume'].style.setProperty('--fill', `${volume * 100}%`); setIcon('volume-icon', muted ? 'muted' : 'volume'); dom['volume-button'].setAttribute('aria-pressed', String(muted)); dom['volume-button'].setAttribute('aria-label', muted ? 'Unmute music' : 'Mute music'); }
function renderHistory() {
    text('recent-count', trackCount(state.history.length));
    for (const button of document.querySelectorAll('[data-action="clear-history"]')) button.hidden = !state.history.length;
    text('home-history-meta', state.history.length ? trackCount(state.history.length) : 'None yet');
    for (const id of ['recent-tracks', 'library-recent-tracks']) {
        if (state.history.length) renderTracks(id, id === 'recent-tracks' ? state.history : state.history.slice(0, 5), { history: true });
        else if (dom[id].querySelector('.track-row') || changed(id, ['empty'])) dom[id].replaceChildren(empty('No recent plays', 'Tracks you listen to will appear here.', 'clock'));
    }
}
let searchAbort = null;
let searchDebounce = null;
let searchGeneration = 0;
async function search() {
    clearTimeout(searchDebounce); searchAbort?.abort(); const generation = ++searchGeneration;
    const query = dom['search-input'].value.trim(); state.search.query = query;
    if (!query) { state.search.status = 'idle'; state.search.results = []; renderSearch(); return; }
    searchAbort = new AbortController(); state.search.status = 'loading'; renderSearch();
    const timeout = setTimeout(() => searchAbort?.abort(), 15000);
    try {
        const { results } = await searchAPI(query, searchAbort.signal);
        if (generation !== searchGeneration) return;
        state.search = { query, results, status: 'ready', error: '' };
    } catch (error) {
        if (generation !== searchGeneration) return;
        state.search = { query, results: [], status: 'error', error: error.name === 'AbortError' ? 'Search took too long. Try again.' : error.message };
    } finally { clearTimeout(timeout); if (generation === searchGeneration) renderSearch(); }
}
function renderSearch() {
    dom['search-clear'].hidden = !dom['search-input'].value;
    const search = state.search;
    dom['search-results'].setAttribute('aria-busy', String(search.status === 'loading'));
    text('search-summary', search.status === 'loading' ? 'Searching…' : search.status === 'ready' ? `${search.results.length} results` : '');
    if (search.status === 'ready' && search.results.length) { renderTracks('search-results', search.results); return; }
    if (!changed('search-results', [search.status, search.error])) return;
    if (search.status === 'loading') { dom['search-results'].replaceChildren(loadingState('Searching for music')); return; }
    if (search.status === 'idle') { dom['search-results'].replaceChildren(empty('What would you like to hear?', 'Search for a song or artist, or paste a supported music link.', 'search')); return; }
    const messages = { ready: ['No tracks found', 'Try another song, artist, or music link.'], error: ['Search unavailable', search.error] };
    const [title, detail] = messages[search.status];
    const feedback = empty(title, detail, search.status === 'error' ? 'info' : 'search');
    feedback.classList.toggle('is-error', search.status === 'error');
    if (search.status === 'error') feedback.append(element('button', { class: 'button primary', text: 'Try again', 'data-action': 'retry-search' }));
    dom['search-results'].replaceChildren(feedback);
}
function requestTrack(track, force) {
    if (!track || (force ? !canControl() : !canPlay())) return;
    playback.join(); send(force ? 'play' : 'enqueue', { track, force });
}
let discoveryAbort = null;
const mixSeed = () => state.room?.track || state.history[0];
const mixQuery = track => track ? track.platform === 'telegram' ? `${track.title || ''} ${track.artist || ''}`.trim() : track.url : 'discovery';
function ensureMix() {
    if (!state.room || !['home', 'mix'].includes(state.view) || state.connection !== 'connected') return;
    if (state.mix.status === 'idle' || state.mix.query !== mixQuery(mixSeed())) fetchMix();
}
async function fetchMix() {
    if (state.connection !== 'connected') return;
    const track = mixSeed(), query = mixQuery(track);
    if (state.mix.status === 'loading' && state.mix.query === query) return;
    discoveryAbort?.abort();
    state.mix = { results: [], status: 'loading', query }; renderMix();
    if (track) {
        state.mix.requestId = send('mix', { track, count: 10 });
        if (!state.mix.requestId) { state.mix.status = 'error'; renderMix(); }
        return;
    }
    const controller = discoveryAbort = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 15000);
    try {
        const { results } = await searchAPI('popular songs', controller.signal);
        if (controller !== discoveryAbort || state.mix.query !== query) return;
        state.mix = { results: results || [], status: 'ready', query };
    } catch (error) {
        if (controller !== discoveryAbort || state.mix.query !== query) return;
        state.mix = { results: [], status: 'error', query };
    } finally { clearTimeout(timeout); if (controller === discoveryAbort) renderMix(); }
}

function renderMix() {
    const mix = state.mix, track = state.room?.track;
    dom['player-mix'].hidden = !track;
    dom['player-mix'].disabled = state.connection !== 'connected' || mix.status === 'loading';
    text('player-mix-title', track?.title || '');
    dom['player-mix'].setAttribute('aria-label', track ? `Open ${track.title} mix` : 'Open track mix');
    dom['mix-button'].disabled = state.connection !== 'connected' || mix.status === 'loading';
    dom['home-recommendations'].hidden = !state.room;
    document.getElementById('home-recommendations-title').textContent = track ? 'Similar tracks' : state.history.length ? 'Keep the music going' : 'Find your next favorite';
    for (const id of ['mix-results', 'home-recommendation-tracks']) {
        dom[id].setAttribute('aria-busy', String(mix.status === 'loading'));
        if (mix.status === 'ready' && mix.results.length) {
            renderTracks(id, id === 'mix-results' ? mix.results : mix.results.slice(0, 6));
            continue;
        }
        if (!changed(id, [mix.status, mix.error, Boolean(track), state.connection])) continue;
        for (const key of trackRegistry.keys()) if (key.startsWith(`${id}:`)) trackRegistry.delete(key);
        const message = mix.status === 'error' ? 'Couldn’t load tracks. Try again.' : mix.status === 'ready' ? 'Find your next favorite in Search.' : state.connection !== 'connected' ? 'Reconnect to discover music.' : 'Finding your next track…';
        dom[id].replaceChildren(mix.status === 'loading' ? loadingState('Finding related tracks', 2) : element('p', { class: 'recommendation-status muted', role: 'status', text: message }));
        if (mix.status === 'error' && id === 'home-recommendation-tracks') {
            const retry = element('button', { class: 'text-button', text: 'Try again', 'data-action': 'mix' });
            retry.disabled = state.connection !== 'connected'; dom[id].append(retry);
        }
    }
}

function loadLibrary() {
    if (!state.permissions.userId) return;
    state.libraryStatus = 'loading'; renderLibrary();
    if (!send('get_playlists')) { state.libraryStatus = 'error'; renderLibrary(); }
}
function selectPlaylist(id) { state.selectedPlaylist = id; navigate('library'); renderPlaylistDetail(); dom['playlist-detail'].scrollIntoView({ block: 'start', behavior: 'auto' }); }
function collection(playlist) {
    const cover = element('div', { class: playlist.songs?.[0]?.thumbnail ? 'collection-cover' : 'collection-cover settings-icon', 'data-tone': 'cyan' }, playlist.songs?.[0]?.thumbnail ? [element('img', { src: safeURL(playlist.songs[0].thumbnail, fallbackArtwork), alt: '', loading: 'lazy' })] : [icon('settings-library')]);
    return element('button', { class: 'collection', 'data-action': 'select-playlist', 'data-playlist': playlist.id }, [cover, element('div', { class: 'collection-copy' }, [element('strong', { text: playlist.name }), element('span', { text: trackCount(playlist.songs?.length || 0) })]), icon('chevron-right')]);
}
function renderLibrary() {
    text('home-playlists-meta', state.libraryStatus === 'loading' ? 'Loading…' : state.libraryStatus === 'error' ? 'Unavailable' : state.playlists.length ? `${state.playlists.length} ${state.playlists.length === 1 ? 'playlist' : 'playlists'}` : 'None yet');
    text('library-message', state.libraryStatus === 'loading' ? 'Loading playlists…' : state.libraryStatus === 'error' ? 'Couldn’t load playlists. Try again.' : '');
    if (changed('collections', [state.playlists, state.libraryStatus, state.permissions.userId])) {
        if (state.playlists.length) dom['library-playlists'].replaceChildren(...state.playlists.slice(0, 10).map(collection));
        else if (state.libraryStatus === 'loading') dom['library-playlists'].replaceChildren(loadingState('Loading playlists', 2));
        else {
            const failed = state.libraryStatus === 'error';
            const feedback = empty(failed ? 'Playlists unavailable' : state.permissions.userId ? 'No playlists yet' : 'Open in Telegram', failed ? 'Check your connection and try again.' : state.permissions.userId ? 'Create a playlist and save your favorite tracks.' : 'Connect to see your playlists.', failed ? 'info' : 'library');
            feedback.classList.toggle('is-error', failed);
            if (failed) feedback.append(element('button', { class: 'button primary', text: 'Try again', 'data-action': 'retry-library' }));
            dom['library-playlists'].replaceChildren(feedback);
        }
        if (state.libraryStatus === 'error' && state.playlists.length) dom['library-playlists'].append(element('button', { class: 'button quiet', text: 'Try again', 'data-action': 'retry-library' }));
        dom['sidebar-playlists'].replaceChildren(...(state.playlists.length ? state.playlists.map(pl => element('button', { 'data-action': 'select-playlist', 'data-playlist': pl.id }, [icon('music'), element('span', { text: pl.name })])) : [element('p', { text: 'No playlists yet', class: 'muted' })]));
    }
    renderPlaylistDetail();
}
function selectedPlaylist() { return state.playlists.find(pl => pl.id === state.selectedPlaylist); }
function renderPlaylistDetail() {
    const playlist = selectedPlaylist(); dom['playlist-detail'].hidden = !playlist;
    if (!playlist) return;
    text('playlist-detail-title', playlist.name); text('playlist-detail-meta', trackCount(playlist.songs?.length || 0));
    if (playlist.songs?.length) renderTracks('playlist-tracks', playlist.songs.map(normalizeSong), { playlist: true });
    else {
        const feedback = empty('No tracks yet', 'Find a song and save it to this playlist.', 'heart');
        feedback.append(element('button', { class: 'button primary', text: 'Find music', 'data-nav': 'search' }));
        dom['playlist-tracks'].replaceChildren(feedback);
    }
}
const dialog = createActionDialog(dom, choices, updateBackButton);
async function createPlaylist() {
    if (!state.permissions.userId) { notify('Open the player in Telegram to create playlists.'); return; }
    const name = await dialog({ title: 'New playlist', description: '', input: '', action: 'Create playlist' });
    if (name) send('create_playlist', { playlistName: name });
}
async function renamePlaylist() {
    const playlist = selectedPlaylist(); if (!playlist) return;
    const name = await dialog({ title: 'Rename playlist', description: '', input: playlist.name, action: 'Save' });
    if (name) send('rename_playlist', { playlistId: playlist.id, playlistName: name });
}
async function deletePlaylist() {
    const playlist = selectedPlaylist(); if (!playlist) return;
    if (await dialog({ title: 'Delete playlist?', description: `“${playlist.name}” will be removed from your library.`, action: 'Delete playlist', danger: true })) send('delete_playlist', { playlistId: playlist.id });
}
async function saveTrack(track) {
    if (!track || !state.permissions.userId) { notify('Connect in Telegram to save music.'); return; }
    if (state.libraryStatus === 'loading') { notify('Your playlists are still loading. Try again in a moment.'); return; }
    if (state.libraryStatus === 'error') { loadLibrary(); return; }
    const choice = await dialog({ title: 'Save to playlist', description: state.playlists.length ? `“${track.title}”` : 'Creates “My Playlist”.', choices: state.playlists.length ? state.playlists.map(pl => ({ id: pl.id, name: pl.name })) : undefined, action: 'Save track' });
    if (choice) send('add_to_playlist', { track, playlistId: typeof choice === 'string' ? choice : '' });
}

const inviteForm = document.getElementById('partner-invite-form');
const inviteButton = document.getElementById('invite-partner');
const inviteExpiry = document.getElementById('partner-expiry');
const playerInvite = document.getElementById('player-invite');
let inviteRequest = null;
let sharingInvite = false;
let inviteMessage = '';
function botChatURL() { const username = state.botProfile?.username; return typeof username === 'string' && /^[a-zA-Z0-9_]{5,32}$/.test(username) ? `https://t.me/${username}` : ''; }
function renderPartnerInvite() {
    const owner = Number(state.roomId) > 0 && Number(state.roomId) === state.permissions.userId;
    document.getElementById('partner-access').hidden = playerInvite.hidden = !owner;
    document.getElementById('partner-context').hidden = owner;
    document.getElementById('partner-lead').hidden = !owner;
    document.getElementById('partner-note').hidden = !owner;
    document.getElementById('partner-open-chat').hidden = !botChatURL();
    document.getElementById('partner-context-title').textContent = Number(state.roomId) < 0 ? 'Your own space for two' : 'Make it your own';
    document.getElementById('partner-context-description').textContent = Number(state.roomId) < 0 ? 'Partner access is for your personal player. This group’s music stays shared with everyone here.' : 'You’re listening in someone else’s player. Open your personal player to invite a partner of your own.';
    inviteButton.disabled = state.connection !== 'connected' || !platform.isTelegramWebApp || Boolean(inviteRequest);
    playerInvite.disabled = inviteButton.disabled;
    for (const button of document.querySelectorAll('[data-invite-control], #remove-partner-password')) button.disabled = inviteButton.disabled;
    inviteButton.textContent = inviteRequest ? 'Saving…' : 'Save invite settings';
    inviteButton.title = playerInvite.title = platform.isTelegramWebApp ? 'Invite Partner' : 'Open the player in Telegram to invite a partner.';
}
function syncPartnerInvite(settings) {
    if (!settings) return;
    const expiry = settings.expiresAt ? new Date(settings.expiresAt * 1000) : null;
    inviteExpiry.value = expiry ? new Date(expiry - expiry.getTimezoneOffset() * 60000).toISOString().slice(0, 16) : '';
    document.getElementById('partner-limit').value = settings.userLimit;
    text('partner-summary', `${expiry ? `Expires ${expiry.toLocaleString()}` : 'No time limit'} · ${settings.userLimit ? `${settings.userLimit} users` : 'Unlimited users'}`);
    text('partner-password-status', settings.hasPassword ? 'Passcode is on. Save a new one to change it.' : 'Use at least 6 characters. Partners enter it from your player link.');
    document.getElementById('remove-partner-password').hidden = !settings.hasPassword;
}
function sharePartnerInvite() {
    if (playerInvite.disabled) return;
    sharingInvite = true;
    inviteRequest = send('create_partner_invite');
    renderPartnerInvite();
}
on('permissions', () => syncPartnerInvite(state.permissions.partnerInvite));
inviteForm.addEventListener('submit', event => {
    event.preventDefault();
    if (inviteButton.disabled) return;
    const expiresAt = inviteExpiry.value ? Math.floor(new Date(inviteExpiry.value).getTime() / 1000) : 0;
    if (expiresAt && expiresAt <= Date.now() / 1000) { notify('Choose a future expiry time.', 'error'); inviteExpiry.focus(); return; }
    sharingInvite = false;
    inviteMessage = 'Invite settings saved.';
    inviteRequest = send('update_partner_invite', { expiresAt, userLimit: Number(document.getElementById('partner-limit').value) });
    renderPartnerInvite();
});
function setPartnerPassword(password) {
    if (inviteButton.disabled) return;
    sharingInvite = false; inviteMessage = password ? 'Passcode saved.' : 'Passcode removed.';
    inviteRequest = send('set_partner_password', { password });
    renderPartnerInvite();
}
document.getElementById('partner-password-form').addEventListener('submit', event => {
    event.preventDefault(); setPartnerPassword(document.getElementById('partner-password').value);
});
on('partner-invite-created', data => {
    if (data.requestId !== inviteRequest) return;
    inviteRequest = null; renderPartnerInvite();
    syncPartnerInvite(data);
    document.getElementById('partner-password').value = '';
    if (!sharingInvite) { notify(inviteMessage, 'success'); return; }
    const share = new URL('https://t.me/share/url');
    share.searchParams.set('url', data.url);
    share.searchParams.set('text', 'Join my PM Player and listen together.');
    telegram.openTelegramLink(share.href);
});
function renderProfile() {
    text('profile-name', [state.user.first_name, state.user.last_name].filter(Boolean).join(' ') || 'Telegram listener');
    text('profile-handle', state.user.username ? `@${state.user.username}` : state.user.id ? `Telegram ID ${state.user.id}` : 'Open in Telegram to connect.');
    text('account-name', [state.user.first_name, state.user.last_name].filter(Boolean).join(' ') || 'Telegram listener');
    text('account-username', state.user.username ? `@${state.user.username}` : 'Not set');
    text('account-id', state.user.id || '—');
    avatar(dom['header-avatar'], state.user); avatar(dom['profile-avatar'], state.user); avatar(dom['chat-self-avatar'], state.user);
    choices.refresh(); renderFullscreen();
}
function renderFullscreen() {
    const enabled = platform.supportsBrowserFullscreen || (platform.isTelegramWebApp && platform.supportsTelegramFullscreen);
    dom['fullscreen-button'].disabled = !enabled;
    const active = isFullscreen(), action = active ? 'Exit fullscreen' : 'Enter fullscreen';
    text('fullscreen-button', active ? 'On' : 'Off');
    dom['fullscreen-button'].setAttribute('aria-label', action); dom['fullscreen-button'].title = action;
    text('fullscreen-hint', enabled ? '' : 'Unavailable on this device');
    dom['fullscreen-hint'].hidden = enabled;
}
function setSleep(minutes) { state.sleepUntil = minutes ? Date.now() + minutes * 60000 : 0; renderSleep(); }
function renderSleep() {
    text('sleep-status', state.sleepUntil ? `Music stops in ${formatTime((state.sleepUntil - Date.now()) / 1000)}` : 'Stops music on this device.');
    dom['sleep-status'].hidden = !state.sleepUntil;
    if (!state.sleepUntil) dom['sleep-select'].value = '0'; choices.refresh();
}

function messageNode(message) {
    const timestamp = Number(message.timestamp) || Date.now();
    const own = message.userId === state.permissions.userId;
    const bubble = element('div', { class: 'chat-bubble' });
    if (!own) bubble.append(element('strong', { class: 'chat-sender', text: `${message.sender || 'Listener'}${message.isAdmin ? ' · Admin' : ''}` }));
    bubble.append(element('p', { text: message.text }), element('time', { datetime: new Date(timestamp).toISOString(), text: new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }));
    const nodes = [];
    if (!own) { const image = element('div', { class: 'avatar message-avatar', 'aria-label': message.sender || 'Listener' }); avatar(image, { first_name: message.sender, photo_url: message.photoURL || message.photoUrl }); nodes.push(image); }
    return element('article', { class: `chat-message${own ? ' self' : ''}`, 'data-message': message.id }, [...nodes, bubble]);
}
function renderChatHistory() {
    dom['chat-messages'].replaceChildren(...(state.chat.messages.length ? state.chat.messages.map(messageNode) : [empty('No messages yet', 'Say hello to your room.', 'chat')]));
    dom['chat-messages'].scrollTop = dom['chat-messages'].scrollHeight;
}
function appendMessage(message) {
    if (dom['chat-messages'].querySelector(`[data-message="${CSS.escape(message.id)}"]`)) return;
    const list = dom['chat-messages']; const atBottom = list.scrollHeight - list.scrollTop - list.clientHeight < 80;
    list.querySelector('.empty-state')?.remove(); list.append(messageNode(message));
    while (list.children.length > 100) list.firstElementChild.remove();
    if (atBottom || message.userId === state.permissions.userId) list.scrollTop = list.scrollHeight;
    if (message.userId === state.permissions.userId) dom['chat-input'].value = '';
    renderChatStatus();
}
function renderChatStatus() {
    const remaining = Math.max(0, Math.ceil((state.chat.cooldownUntil - Date.now()) / 1000));
    const enabled = state.room?.chatEnabled;
    const connected = state.connection === 'connected' && state.permissions.userId;
    text('chat-state', !connected ? 'Reconnecting…' : !enabled ? 'Chat disabled by admin' : '');
    dom['chat-state'].hidden = connected && enabled;
    dom['chat-input'].disabled = !connected || !enabled;
    dom['chat-send'].disabled = !connected || !enabled || remaining > 0 || !dom['chat-input'].value.trim();
    const length = [...dom['chat-input'].value].length;
    text('chat-composer-state', remaining > 0 ? `Wait ${remaining}s` : length >= 400 ? `${length} / 500` : '');
    dom['chat-composer-state'].hidden = !dom['chat-composer-state'].textContent;
    text('chat-unread', state.chat.unread); dom['chat-unread'].hidden = state.chat.unread === 0;
}
function sendChat() {
    const value = dom['chat-input'].value.trim();
    if (!value || dom['chat-send'].disabled) return;
    if ([...value].length > 500) { notify('Messages can contain up to 500 characters.'); return; }
    if (send('chat_message', { text: value })) { state.chat.cooldownUntil = Date.now() + Math.max(.5, state.room?.chatCooldown || 0) * 1000; renderChatStatus(); }
}

function renderVoiceState() {
    const participants = state.voice.room.participants;
    if (changed('voice-participants', [participants, state.permissions.isAdmin])) dom['voice-participants'].replaceChildren(...(participants.length ? participants.map(user => participant(user, { voiceParticipant: true })) : [empty('No one’s in voice chat yet', 'Join to start the conversation.', 'mic')]));
    const restricted = state.voice.room.muteNewParticipants || participants.some(p => !p.isAdmin && p.isAdminMuted);
    dom['voice-join-policy'].dataset.muted = String(!restricted);
    text('voice-join-policy', restricted ? 'Unmute all participants' : 'Mute new participants');
    text('voice-header-label', participants.length ? `Voice · ${participants.length}` : 'Voice chat');
    renderVoiceLocal();
}
function renderVoiceSpeaking(data) {
    for (const node of dom['voice-participants'].querySelectorAll('.participant')) {
        if (String(data.userId) !== node.dataset.user) continue;
        node.classList.toggle('is-speaking', data.isSpeaking);
        const user = state.voice.room.participants.find(p => p.userId === data.userId);
        if (user?.allowedToSpeak && !user.isSelfMuted) node.querySelector('p').textContent = data.isSpeaking ? 'Speaking' : 'Listening';
    }
}
function renderVoiceLocal() {
    const vc = state.voice;
    dom['voice-settings'].hidden = !vc.settingsOpen;
    dom['voice-settings-button'].setAttribute('aria-expanded', String(vc.settingsOpen));
    const noiseAvailable = Boolean(navigator.mediaDevices?.getSupportedConstraints?.().noiseSuppression);
    dom['voice-noise-toggle'].checked = vc.noiseSuppression; dom['voice-noise-toggle'].disabled = !noiseAvailable;
    text('voice-settings-note', noiseAvailable ? '' : 'Noise suppression is unavailable on this device.');
    dom['voice-settings-note'].hidden = noiseAvailable;
    const labels = { idle: 'Not connected', joining: 'Joining…', connecting: 'Connecting…', connected: `${vc.room.participants.length} participants`, reconnecting: 'Reconnecting…', error: 'Voice connection failed. Leave and join again.', unsupported: 'Voice chat is unavailable on this device.' };
    const browserVoice = !platform.supportsVoice && platform.isTelegramWebApp;
    text('voice-connection', browserVoice ? 'Join voice chat in your browser.' : labels[vc.status]); text('voice-mini-label', vc.status === 'connected' ? `${vc.muted ? 'Mic muted' : 'Voice connected'} · ${vc.room.participants.length}` : labels[vc.status]);
    text('voice-join', browserVoice ? 'Join in browser' : vc.status === 'joining' ? 'Joining…' : 'Join voice chat');
    dom['voice-join'].hidden = vc.joined; dom['voice-join'].disabled = ['joining', 'connecting'].includes(vc.status) || state.connection !== 'connected' || !state.permissions.userId;
    dom['voice-mic'].hidden = !vc.joined; dom['voice-speaker'].hidden = !vc.joined; dom['voice-leave'].hidden = !vc.joined;
    dom['voice-mini'].hidden = !vc.joined || vc.open || state.chat.open;
    const self = vc.room.participants.find(p => p.userId === state.permissions.userId);
    dom['voice-mic'].disabled = self?.allowedToSpeak === false;
    text('voice-mic-label', self?.allowedToSpeak === false ? 'Listening only' : vc.muted ? 'Unmute mic' : 'Mute mic');
    setIcon('voice-mic-icon', vc.muted ? 'mic-off' : 'mic'); setIcon('voice-mini-mic-icon', vc.muted ? 'mic-off' : 'mic');
    dom['voice-speaker'].setAttribute('aria-pressed', String(vc.outputMuted)); dom['voice-speaker'].setAttribute('aria-label', vc.outputMuted ? 'Unmute voice chat' : 'Mute voice chat');
}

const pendingActions = new Map();
const localActions = new Set(['select-playlist','workspace-back','toggle-passcode','expand-player','toggle-player','open-listeners','close-listeners','open-chat','close-chat','open-voice','close-voice','voice-settings','clear-search','sleep-focus','cancel-dialog','open-settings','create-playlist','rename-playlist','delete-playlist','save-track','save-current','clear-queue','stop-room','clear-history','remove-history','find-music','write-access']);
const actions = {
    'find-music': () => navigate('search'),
    'workspace-back': () => navigate(state.view === 'settings' ? 'profile' : 'home'),
    'toggle-passcode': button => {
        const input = document.getElementById(button.dataset.input);
        const visible = input.type === 'password';
        input.type = visible ? 'text' : 'password';
        button.querySelector('use').setAttribute('href', `/static/assets/icons.svg?v=86#${visible ? 'eye-off' : 'eye'}`);
        button.setAttribute('aria-label', visible ? 'Hide passcode' : 'Show passcode');
        button.title = button.getAttribute('aria-label');
        button.setAttribute('aria-pressed', String(visible));
    },
    'invite-partner': () => sharePartnerInvite(),
    'remove-partner-password': () => setPartnerPassword(''),
    'open-settings': button => openSettings(button.dataset.section),
    'open-in-browser': openInBrowser,
    'open-bot-chat': () => { const url = botChatURL(); if (url) { if (platform.isTelegramWebApp) telegram.openTelegramLink(url); else window.open(url, '_blank', 'noopener,noreferrer'); } },
    'expand-player': expandPlayer, 'toggle-player': button => state.expandedPlayer ? collapsePlayer() : expandPlayer(button),
    'open-listeners': openListeners, 'close-listeners': closeListeners,
    'clear-search': () => { dom['search-input'].value = ''; search(); dom['search-input'].focus(); },
    'voice-settings': () => { state.voice.settingsOpen = !state.voice.settingsOpen; renderVoiceLocal(); },
    'voice-join-policy': button => { const muted = button.dataset.muted === 'true'; if (canManageSettings() && send('vc_admin_set_join_muted', { muted })) notify(muted ? 'New participants will join muted.' : 'Participants can choose when to unmute.', 'success'); },
    'open-chat': openChat, 'close-chat': closeChat, 'open-voice': openVoice, 'close-voice': closeVoice,
    'join-listening': () => playback.join(), 'play-pause': () => playback.toggle(), 'retry-audio': () => playback.retry(),
    'restart-track': () => playback.seek(0), 'skip-track': () => { if (canControl()) send('skip'); },
    repeat: () => { if (canManageSettings()) send('loop', { count: state.room?.loop ? 0 : 1 }); },
    'mute-music': () => playback.mute(), 'sleep-focus': button => choices.open('sleep-select', button),
    'write-access': requestWriteAccess, fullscreen: toggleFullscreen, reconnect: connect,
    'retry-search': search, mix: fetchMix, 'create-playlist': createPlaylist, 'rename-playlist': renamePlaylist, 'delete-playlist': deletePlaylist,
    'retry-library': loadLibrary, 'select-playlist': button => selectPlaylist(button.dataset.playlist),
    'play-track': button => requestTrack(trackRegistry.get(button.dataset.track), true),
    'queue-track': button => requestTrack(trackRegistry.get(button.dataset.track), false),
    'save-track': button => saveTrack(trackRegistry.get(button.dataset.track)), 'save-current': () => saveTrack(state.room?.track),
    'remove-history': button => { const track = trackRegistry.get(button.dataset.track); if (track) { state.history = state.history.filter(item => item.id !== track.id); preferences.set(`history:${state.roomId}`, state.history); renderHistory(); ensureMix(); notify('Removed from recently played', 'success'); } },
    'clear-history': async () => { if (state.history.length && await dialog({ title: 'Clear recently played?', description: 'Removes listening history for this room on this device.', action: 'Clear history', danger: true })) { state.history = []; preferences.set(`history:${state.roomId}`, []); renderHistory(); ensureMix(); notify('History cleared', 'success'); } },
    'remove-song': button => { const track = trackRegistry.get(button.dataset.track); if (track && selectedPlaylist()) send('remove_from_playlist', { playlistId: state.selectedPlaylist, trackId: track.id }); },
    'play-playlist': () => { if (selectedPlaylist() && canControl()) { playback.join(); send('play_playlist', { playlistId: state.selectedPlaylist }); } },
    'queue-playlist': () => { if (selectedPlaylist() && canPlay()) { playback.join(); send('enqueue_playlist', { playlistId: state.selectedPlaylist }); } },
    'remove-queue': button => { if (canControl()) send('remove', { index: Number(button.dataset.index) }); },
    'clear-queue': async () => { if (canControl() && state.room?.queue?.length && await dialog({ title: 'Clear the queue?', description: 'Upcoming tracks will be removed for everyone. The current track will keep playing.', action: 'Clear queue' })) send('clear_queue'); },
    'stop-room': async () => { if (canControl() && state.room?.track && await dialog({ title: 'Stop room playback?', description: 'Music will stop for all listeners and the queue will be cleared. Voice chat stays connected.', action: 'Stop music', danger: true })) send('stop'); },
    'join-voice': () => voice.join(), 'leave-voice': () => voice.leave(), 'voice-mic': () => voice.toggleMic(), 'voice-speaker': () => voice.toggleOutput(),
    'admin-mute': button => send('vc_admin_mute', { targetUserId: Number(button.dataset.user), muted: button.dataset.muted === 'true' }),
    'cancel-dialog': () => dom['action-dialog'].close()
};
document.addEventListener('click', event => {
    const button = event.target.closest('button[data-action],button[data-nav]');
    if (!button || button.disabled) return;
    if (button.dataset.nav) { haptic('selection'); navigate(button.dataset.nav); return; }
    const action = button.dataset.action;
    const key = `${action}:${button.dataset.track || button.dataset.user || ''}`;
    const previous = pendingActions.get(key) || 0;
    if (!localActions.has(action)) {
        if (Date.now() - previous < 400) return;
        pendingActions.set(key, Date.now());
    }
    haptic(localActions.has(action) ? 'selection' : 'light');
    if (pendingActions.size > 100) pendingActions.clear();
    Promise.resolve(actions[action]?.(button)).catch(error => { console.error(error); notify('That action could not finish. Please try again.', 'error'); });
});
document.getElementById('settings-view').addEventListener('click', event => {
    if (event.target.closest('button,input,select,a,label')) return;
    const control = event.target.closest('.setting-row')?.querySelector('.choice-trigger,input[type="checkbox"],button[data-action]');
    if (control && !control.disabled) control.click();
});
dom['player'].addEventListener('click', event => {
    if (!state.expandedPlayer && !event.target.closest('button,input,select,a,[role="combobox"]')) expandPlayer(dom['player'].querySelector('.player-artwork'));
});
document.addEventListener('error', event => { if (event.target instanceof HTMLImageElement) { const img = event.target; if (!img.dataset.artworkSource && !img.dataset.avatar && img.getAttribute('src') !== fallbackArtwork) img.src = fallbackArtwork; } }, true);
// Touch can leave :hover latched in WebViews, including devices with a mouse.
function updatePointerMode(event) {
    document.body.classList.toggle('using-mouse', event.pointerType === 'mouse');
}
document.addEventListener('pointerdown', event => {
    document.body.classList.remove('using-keyboard'); updatePointerMode(event);
}, true);
document.addEventListener('pointermove', updatePointerMode, { capture: true, passive: true });
document.addEventListener('keydown', event => {
    if (['Tab', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End', 'PageUp', 'PageDown', 'Enter', ' '].includes(event.key)) document.body.classList.add('using-keyboard');
}, true);
document.addEventListener('keydown', event => {
    if (!dom['session-notice'].hidden) return;
    if (event.key === 'Tab' && state.expandedPlayer && !dom['action-dialog'].open && !choices.isOpen()) {
        const surfaces = [dom['player'], dom['listeners-panel'], dom['chat-panel'], dom['voice-panel'], dom['voice-mini'], dom['listen-banner'], dom['write-access-notice']];
        const targets = surfaces.flatMap(surface => [...surface.querySelectorAll('button,input,select,a[href],[tabindex="0"]')]).filter(node => !node.disabled && node.getClientRects().length);
        const first = targets[0], last = targets.at(-1);
        if (first && event.shiftKey && (document.activeElement === first || document.activeElement === dom['player'])) { event.preventDefault(); last.focus(); }
        else if (first && !event.shiftKey && (document.activeElement === last || document.activeElement === dom['player'])) { event.preventDefault(); first.focus(); }
    }
    if (event.key === 'Escape' && !dom['action-dialog'].open && !choices.isOpen()) { if (!dom['listeners-panel'].hidden) closeListeners(); else if (state.voice.open) closeVoice(); else if (state.chat.open) closeChat(); else collapsePlayer(); }
    if (event.code === 'Space' && !event.target.closest('input,textarea,select,button,a,dialog,#lyrics-panel') && canControl()) { event.preventDefault(); playback.toggle(); }
});
dom['search-form'].addEventListener('submit', event => { event.preventDefault(); search(); });
dom['search-input'].addEventListener('input', () => { dom['search-clear'].hidden = !dom['search-input'].value; clearTimeout(searchDebounce); searchAbort?.abort(); searchGeneration++; searchDebounce = setTimeout(search, 400); });
dom['repeat-select'].addEventListener('change', () => { if (canManageSettings()) send('loop', { count: Number(dom['repeat-select'].value) }); });
dom['autoplay-toggle'].addEventListener('change', () => { if (canManageSettings() && !dom['autoplay-toggle'].disabled) send('autoplay'); });
dom['sleep-select'].addEventListener('change', () => { haptic('selection'); setSleep(Number(dom['sleep-select'].value)); });
const chatSettingControls = new Map([
    ['chat-enabled-toggle', ['chatEnabled', node => node.checked]],
    ['chat-cooldown', ['chatCooldown', node => slowModeSteps[Number(node.value)]]],
    ['chat-play-mode', ['restrictedPlay', node => node.value === 'admins']],
    ['chat-admin-mode', ['adminMode', node => node.value]],
    ['chat-cmd-delete', ['cmdDelete', node => node.checked]],
    ['chat-autoplay', ['autoplay', node => node.checked]],
    ['chat-auto-delete', ['autoDelete', node => node.checked]],
    ['chat-auto-delete-minutes', ['autoDeleteMinutes', node => Number(node.value)]],
    ['chat-silent-mode', ['silentMode', node => node.checked]]
]);
dom['chat-cooldown'].addEventListener('input', previewSlowMode);
for (const [id, [field, value]] of chatSettingControls) dom[id].addEventListener('change', () => {
    const permission = field === 'silentMode' ? state.permissions.canChangeSilentMode : state.permissions.isAdmin;
    if (!canManageSettings() || !permission || chatSettingsRequest || dom[id].disabled) { renderChatSettings(); return; }
    const selected = value(dom[id]);
    if (field === 'chatCooldown' && selected === state.room?.chatCooldown) { renderChatSettings(); return; }
    if (field === 'autoDeleteMinutes' && (!Number.isInteger(selected) || selected < 2 || selected > 7)) { notify('Choose a timer from 2 to 7 minutes.', 'error'); renderChatSettings(); return; }
    chatSettingsRequest = send('chat_settings', { [field]: selected }); renderChatSettings();
});
dom['player-volume'].addEventListener('input', () => playback.volume(Number(dom['player-volume'].value)));
dom['player-seek'].addEventListener('input', () => { seeking = true; text('player-elapsed', formatTime(dom['player-seek'].value)); dom['player-seek'].style.setProperty('--fill', `${Number(dom['player-seek'].value) / Number(dom['player-seek'].max) * 100}%`); });
dom['player-seek'].addEventListener('change', () => { seeking = false; playback.seek(Number(dom['player-seek'].value)); });
dom['player-seek'].addEventListener('blur', () => { seeking = false; });
dom['chat-form'].addEventListener('submit', event => { event.preventDefault(); sendChat(); });
dom['chat-input'].addEventListener('input', renderChatStatus);
dom['voice-noise-toggle'].addEventListener('change', () => voice.setNoiseSuppression(dom['voice-noise-toggle'].checked));
let toastTimer = null, toastExitTimer = null;
function hideToast() {
    dom.toast.classList.add('is-leaving');
    toastExitTimer = setTimeout(() => {
        dom.toast.hidden = true;
        if (dom.toast.hidePopover && dom.toast.matches(':popover-open')) dom.toast.hidePopover();
    }, 180);
}
on('notice', notice => {
    const { message, type = '' } = typeof notice === 'string' ? { message: notice } : notice;
    if (!message) return;
    clearTimeout(toastTimer); clearTimeout(toastExitTimer);
    dom.toast.classList.remove('is-leaving');
    dom.toast.dataset.tone = type;
    dom.toast.replaceChildren(icon(type === 'success' ? 'check' : 'info'), element('span', { text: message }));
    dom.toast.hidden = false;
    // Keep feedback visible above dialogs without stealing focus or intercepting taps.
    if (dom.toast.showPopover && !dom.toast.matches(':popover-open')) dom.toast.showPopover();
    const duration = type === 'success' ? 2200 : Math.min(8000, Math.max(type === 'error' || type === 'warning' ? 4500 : 2800, message.length * 45));
    toastTimer = setTimeout(hideToast, duration);
});
on('request-success', ({ type }) => {
    const message = { enqueue: 'Added to queue', enqueue_playlist: 'Playlist added to queue', remove: 'Removed from queue', clear_queue: 'Queue cleared' }[type];
    if (message) notify(message, 'success');
});
on('room', renderRoom); on('connection', renderConnection); on('permissions', () => { renderProfile(); renderPermissions(); if (state.permissions.userId && state.libraryStatus === 'idle') loadLibrary(); });
on('fullscreen', renderFullscreen);
on('progress', renderProgress); on('player-status', renderPlayerStatus); on('volume', renderVolume); on('listening', renderListening);
on('history', renderHistory); on('mix', renderMix);
on('library', event => { renderLibrary(); if (event !== 'user_playlists') notify({ playlist_created: 'Playlist created', playlist_renamed: 'Playlist renamed', playlist_deleted: 'Playlist deleted', song_added_to_playlist: 'Saved to playlist', song_removed_from_playlist: 'Removed from playlist' }[event], 'success'); });
on('chat-history', renderChatHistory); on('chat-message', appendMessage); on('chat-status', renderChatStatus);
on('voice-state', renderVoiceState); on('voice-local', renderVoiceLocal); on('voice-speaking', renderVoiceSpeaking);
on('tick', () => { renderSleep(); if (state.chat.open || state.chat.cooldownUntil > Date.now()) renderChatStatus(); });
on('chat-settings-saved', data => { if (data.requestId === chatSettingsRequest) { chatSettingsRequest = null; renderChatSettings(); notify('Settings saved', 'success'); } });
on('request-error', data => {
    if (['create_partner_invite', 'update_partner_invite', 'set_partner_password'].includes(data.type) && data.requestId === inviteRequest) { inviteRequest = null; renderPartnerInvite(); }
    if (data.type === 'chat_settings' && data.requestId === chatSettingsRequest) { chatSettingsRequest = null; renderChatSettings(); }
    if (data.type === 'get_playlists') { state.libraryStatus = 'error'; renderLibrary(); }
    if (data.type === 'mix' && data.requestId === state.mix.requestId) { state.mix.status = 'error'; state.mix.error = String(data.message); renderMix(); }
    if (data.type === 'chat_message') { state.chat.cooldownUntil = 0; renderChatStatus(); }
});
on('transport-lost', () => { inviteRequest = null; renderPartnerInvite(); chatSettingsRequest = null; renderChatSettings(); if (state.libraryStatus === 'loading') state.libraryStatus = 'error'; if (state.mix.status === 'loading') state.mix.status = 'error'; renderLibrary(); renderMix(); });
function stopSessionPlayback() { choices.close(); collapsePlayer(); state.joinedListening = false; dom['music-audio'].pause(); voice.leave(); renderListening(); }
on('session-ended', stopSessionPlayback);
on('authentication-failed', stopSessionPlayback);
on('sleep-expired', () => { renderSleep(); renderListening(); });
on('dispose', () => { searchAbort?.abort(); discoveryAbort?.abort(); clearTimeout(searchDebounce); clearTimeout(toastTimer); clearTimeout(toastExitTimer); });

initializePlatform(); arrangeQueue(); renderPlayerView(); renderProfile(); renderConnection(); renderQueue(); renderLibrary(); renderHistory(); renderSearch(); renderChatHistory(); renderChatStatus(); renderVoiceState(); renderVolume(); renderListening();
if (!hasSession) { session.show('telegram_required'); text('connection-label', 'Telegram required'); }
else if (state.roomId === '0' || !/^-?\d+$/.test(state.roomId)) { session.show('invalid_room_identity'); }
else connect();
