import { state, allowedDuration, hasQueueSpace, canControl, canPlay, formatTime, safeURL } from './core.js?v=87';
import { element, icon, actionButton } from './ui.js?v=87';
import { setTrackArtwork, fallbackArtwork } from './artwork.js?v=87';

export function createTrackRenderer(dom) {
    const trackRegistry = new Map();
    const renderKeys = new Map();
    function trackRow(track, scope, index, { queue = false, current = false, playlist = false, history = false } = {}) {
        const key = `${scope}:${index}:${track.id}`; if (!current && !queue) trackRegistry.set(key, track);
        const image = element('img', { class: 'track-art', alt: '', width: '48', height: '48', loading: 'lazy' });
        setTrackArtwork(image, safeURL(track.thumbnail, fallbackArtwork));
        const artist = track.artist || (track.user ? `Added by ${track.user}` : '');
        const meta = element('p', { text: artist || track.platform || 'Music' });
        const copy = element('div', { class: 'track-copy' }, [element('strong', { text: track.title || 'Untitled track', title: track.title }), element('div', { class: 'track-meta' }, [meta, element('span', { class: 'track-duration', text: formatTime(track.duration) })])]);
        const actions = element('div', { class: 'track-actions' });
        if (!current && !queue) {
            const enqueue = actionButton('queue-track', `Add ${track.title} to queue`, 'plus', { 'data-track': key, 'data-permission': 'play' }); enqueue.disabled = !canPlay() || !allowedDuration(track) || !hasQueueSpace();
            if (!allowedDuration(track)) enqueue.title = 'Exceeds the playback duration limit';
            else if (!hasQueueSpace()) enqueue.title = 'Queue is full'; actions.append(enqueue);
        }
        if (queue && canControl()) actions.append(actionButton('remove-queue', `Remove ${track.title} from queue`, 'trash', { 'data-index': index + 1 }));
        if (!queue && !current) actions.append(actionButton(history ? 'remove-history' : playlist ? 'remove-song' : 'save-track', history ? `Remove ${track.title} from recently played` : playlist ? `Remove ${track.title} from playlist` : `Save ${track.title} to playlist`, history || playlist ? 'trash' : 'heart', { 'data-track': key }));
        const cover = current || (!queue && canControl() && allowedDuration(track) && hasQueueSpace()) ? element('button', { class: 'track-art-button', type: 'button', 'data-action': current ? 'expand-player' : 'play-track', 'data-track': current ? undefined : key, 'aria-label': current ? 'Open now playing' : `Play ${track.title} now`, title: current ? 'Open now playing' : `Play ${track.title} now` }, [image, icon(current ? 'expand' : 'play')]) : image;
        return element('div', { class: `track-row${current ? ' current' : queue ? ' queued' : ' actionable'}` }, [cover, copy, actions]);
    }
    function renderTracks(id, tracks, options = {}) {
        if (!options.queue && !options.current && !options.history && !options.playlist) tracks = tracks.filter(allowedDuration);
        const next = JSON.stringify([tracks, canControl(), canPlay(), state.limits.maxTrackDuration, hasQueueSpace(), options]);
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
