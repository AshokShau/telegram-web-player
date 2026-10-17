import { state, platform, telegram, on, applyTheme } from './core.js?v=55';

export const fallbackArtwork = '/static/assets/artwork.svg?v=55';
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

export function createAtmosphere(layer, backdrop) {
    const fallback = '#171719', colors = new Map();
    let source = '', color = fallback, expanded = false, generation = 0;
    function paint() {
        layer.style.setProperty('--player-color', color);
        if (!expanded) return;
        document.querySelector('meta[name="theme-color"]').content = color;
        if (platform.isTelegramWebApp) {
            for (const method of ['setHeaderColor', 'setBackgroundColor', 'setBottomBarColor']) {
                try { telegram[method]?.(color); } catch { /* Older clients may reject custom chrome. */ }
            }
        }
    }
    function setArtwork(url) {
        if (url === source) return;
        source = url; setTrackArtwork(backdrop, url);
        const request = ++generation;
        color = colors.get(url) || fallback; paint();
        if (colors.has(url)) return;
        const image = new Image();
        image.crossOrigin = 'anonymous';
        image.onload = () => {
            if (request !== generation) return;
            try {
                const canvas = document.createElement('canvas');
                canvas.width = canvas.height = 16;
                const context = canvas.getContext('2d', { willReadFrequently: true });
                context.drawImage(image, 0, 0, 16, 16);
                const pixels = context.getImageData(0, 0, 16, 16).data;
                const channels = [0, 0, 0];
                let weight = 0;
                for (let i = 0; i < pixels.length; i += 4) {
                    const alpha = pixels[i + 3] / 255;
                    weight += alpha;
                    for (let c = 0; c < 3; c++) channels[c] += pixels[i + c] * alpha;
                }
                if (weight) color = '#' + channels.map(sum => Math.round(sum / weight * .38).toString(16).padStart(2, '0')).join('');
                colors.set(url, color);
                if (colors.size > 20) colors.delete(colors.keys().next().value);
                paint();
            } catch { /* The full-screen blurred image remains available without sampling. */ }
        };
        image.src = url;
    }
    backdrop.addEventListener('error', () => { backdrop.style.visibility = 'hidden'; });
    backdrop.addEventListener('load', () => { backdrop.style.visibility = ''; });
    on('theme', paint);
    on('dispose', () => { generation++; colors.clear(); layer.hidden = true; if (expanded) { expanded = false; applyTheme(state.theme); } });
    return {
        setArtwork,
        setExpanded(next) {
            if (next === expanded) return;
            expanded = next; layer.hidden = !next;
            if (next) paint(); else applyTheme(state.theme);
        }
    };
}
