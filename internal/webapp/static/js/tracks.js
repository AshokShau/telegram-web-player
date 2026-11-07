import { state, allowedDuration, hasQueueSpace, canControl, canPlay, formatTime, safeURL } from './core.js?v=111';
import { element, icon, actionButton } from './ui.js?v=111';
import { setTrackArtwork, fallbackArtwork } from './artwork.js?v=114';

export function createTrackRenderer(dom) {
    const trackRegistry = new Map();
    const renderKeys = new Map();
    function trackRow(track, scope, index, { queue = false, current = false, playlist = false, history = false } = {}) {
        const key = `${scope}:${index}:${track.id}`; trackRegistry.set(key, track);
        const image = element('img', { class: 'track-art', alt: '', width: '48', height: '48', loading: 'lazy' });
        setTrackArtwork(image, safeURL(track.thumbnail, fallbackArtwork));
        const artist = track.artist || (track.user ? `Added by ${track.user}` : '');
        const meta = element('p', { text: artist || track.platform || 'Music' });
        const copy = element('div', { class: 'track-copy' }, [element('strong', { text: track.title || 'Untitled track', title: track.title }), element('div', { class: 'track-meta' }, [meta, element('span', { class: 'track-duration', text: formatTime(track.duration) })])]);
        const actions = element('div', { class: 'track-actions' });
        const canQueue = !current && !queue && canPlay() && allowedDuration(track) && hasQueueSpace();
        if (canQueue || state.permissions.userId || history || playlist || (queue && canControl())) {
            actions.append(actionButton('track-menu', `More options for ${track.title || 'track'}`, 'more-vertical', { 'data-track': key, 'data-scope': scope, 'data-index': queue ? index + 1 : undefined, 'aria-haspopup': 'dialog', 'aria-expanded': 'false', 'aria-controls': 'track-actions-menu' }));
        }
        const playable = current || (!queue && canControl() && allowedDuration(track) && hasQueueSpace());
        const cover = playable ? element('span', { class: 'track-art-shell' }, [image, icon(current ? 'expand' : 'play')]) : image;
        const main = element(playable ? 'button' : 'div', { class: 'track-main', type: playable ? 'button' : undefined, 'data-action': playable ? current ? 'expand-player' : 'play-track' : undefined, 'data-track': playable && !current ? key : undefined, 'aria-label': playable ? current ? `Open now playing: ${track.title || 'track'}` : `Play ${track.title || 'track'}${artist ? ` by ${artist}` : ''} now` : undefined }, [cover, copy]);
        return element('div', { class: `track-row${current ? ' current' : queue ? ' queued' : ' actionable'}` }, [main, actions]);
    }
    function renderTracks(id, tracks, options = {}) {
        if (!options.queue && !options.current && !options.history && !options.playlist) tracks = tracks.filter(allowedDuration);
        const next = JSON.stringify([tracks, canControl(), canPlay(), state.permissions.userId, state.limits.maxTrackDuration, hasQueueSpace(), options]);
        const previous = renderKeys.get(id), mount = dom[id];
        // Feedback can temporarily replace rows (loading, empty, or error). The same
        // results must render again after that transition, even if the data is equal.
        if (previous?.signature === next && previous.nodes.length === mount.childNodes.length && previous.nodes.every((node, index) => node === mount.childNodes[index])) return;
        for (const key of trackRegistry.keys()) if (key.startsWith(`${id}:`)) trackRegistry.delete(key);
        mount.replaceChildren(...tracks.map((track, index) => trackRow(track, id, index, options)));
        renderKeys.set(id, { signature: next, nodes: [...mount.childNodes] });
    }

    return { renderTracks, trackRegistry };
}
