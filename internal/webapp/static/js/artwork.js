import { on } from './core.js?v=83';

export const fallbackArtwork = '/static/assets/artwork.svg?v=58';
const artworkRequests = new WeakMap();
const retryDelays = [1200, 3500, 10000];

export function setTrackArtwork(image, source, refresh = false) {
    const previous = artworkRequests.get(image);
    if (previous?.source === source && !refresh) return;
    if (previous) clearTimeout(previous.timer);
    const request = { source, attempt: 0, timer: null };
    artworkRequests.set(image, request);
    image.dataset.artworkSource = source;
    function load(retry = false) {
        if (artworkRequests.get(image) !== request) return;
        let url = source;
        if ((retry || refresh) && new URL(source, location.origin).pathname.startsWith('/thumbnail/telegram/')) {
            const fresh = new URL(source, location.origin);
            fresh.searchParams.set('artwork_retry', String(Date.now())); url = fresh.href;
        }
        image.src = url;
    }
    image.onerror = () => {
        if (artworkRequests.get(image) !== request || image.getAttribute('src') === fallbackArtwork) return;
        image.src = fallbackArtwork;
        if (request.attempt < retryDelays.length) {
            request.timer = setTimeout(() => {
                if (image.isConnected) load(true);
            }, retryDelays[request.attempt++]);
        }
    };
    load();
}

function refreshArtwork() {
    if (document.hidden) return;
    for (const image of document.querySelectorAll('img[data-artwork-source]')) {
        const source = image.dataset.artworkSource;
        if (new URL(source, location.origin).pathname.startsWith('/thumbnail/telegram/') || image.getAttribute('src') === fallbackArtwork) {
            setTrackArtwork(image, source, true);
        }
    }
}
window.addEventListener('online', refreshArtwork);
window.addEventListener('pageshow', refreshArtwork);
document.addEventListener('visibilitychange', refreshArtwork);

// Artwork is a quiet wash beneath the same bright material system.
export function createAtmosphere(layer, backdrop) {
    let source = '';
    backdrop.addEventListener('error', () => { backdrop.style.visibility = 'hidden'; });
    backdrop.addEventListener('load', () => { backdrop.style.visibility = ''; });
    on('dispose', () => { layer.hidden = true; });
    return {
        setArtwork(url) {
            if (url === source) return;
            source = url;
            setTrackArtwork(backdrop, url);
        },
        setExpanded(expanded) { layer.hidden = !expanded; }
    };
}
