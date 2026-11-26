import {on} from './core.js';

export const fallbackArtwork = '/static/assets/artwork.svg?v=92';
const artworkRequests = new WeakMap();
const retryDelays = [1200, 3500, 10000];

function themedFallback() {
    const light = document.documentElement.dataset.theme === 'light';
    const [sleeve, record, groove, shine, label, spindle] = light
        ? ['#e8edf3', '#d4dde9', '#9fb1c6', '#ffffff', '#1676b9', '#e8edf3']
        : ['#272c35', '#1c2129', '#435061', '#8393a7', '#339fec', '#20242c'];
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="400" viewBox="0 0 400 400"><rect width="400" height="400" fill="${sleeve}"/><circle cx="200" cy="200" r="139" fill="${record}"/><g fill="none" stroke="${groove}" stroke-width="2"><circle cx="200" cy="200" r="111"/><circle cx="200" cy="200" r="94"/><circle cx="200" cy="200" r="77"/></g><path d="M105 99a139 139 0 0 1 180-4" fill="none" stroke="${shine}" stroke-width="3" opacity="${light ? '.7' : '.32'}"/><circle cx="200" cy="200" r="30" fill="${label}"/><circle cx="200" cy="200" r="7" fill="${spindle}"/></svg>`;
    return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

export function refreshFallbackArtwork() {
    const fallback = themedFallback();
    for (const image of document.querySelectorAll('img[data-artwork-source], img[src^="/static/assets/artwork.svg"]')) {
        if (image.dataset.artworkSource === fallbackArtwork || image.getAttribute('src')?.startsWith('data:image/svg+xml') || image.getAttribute('src')?.startsWith('/static/assets/artwork.svg')) image.src = fallback;
    }
}

export function setTrackArtwork(image, source, refresh = false) {
    const previous = artworkRequests.get(image);
    if (previous?.source === source && !refresh) return;
    if (previous) clearTimeout(previous.timer);
    const request = {source, attempt: 0, timer: null};
    artworkRequests.set(image, request);
    image.dataset.artworkSource = source;
    image.draggable = false;

    function load(retry = false) {
        if (artworkRequests.get(image) !== request) return;
        let url = source;
        if ((retry || refresh) && new URL(source, location.origin).pathname.startsWith('/thumbnail/telegram/')) {
            const fresh = new URL(source, location.origin);
            fresh.searchParams.set('artwork_retry', String(Date.now()));
            url = fresh.href;
        }
        image.src = source === fallbackArtwork ? themedFallback() : url;
    }

    image.onerror = () => {
        if (artworkRequests.get(image) !== request || image.getAttribute('src')?.startsWith('data:')) return;
        image.src = themedFallback();
        if (request.attempt < retryDelays.length) {
            request.timer = setTimeout(() => {
                if (image.isConnected) load(true);
            }, retryDelays[request.attempt++]);
        }
    };
    load();
}

document.addEventListener('contextmenu', event => {
    if (event.target instanceof HTMLImageElement && event.target.matches('[data-artwork-source]')) event.preventDefault();
});
document.addEventListener('dragstart', event => {
    if (event.target instanceof HTMLImageElement && event.target.matches('[data-artwork-source]')) event.preventDefault();
});

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

export function createAtmosphere(layer, backdrop, onColor) {
    let source = '', generation = 0, palette = null, paletteLoading = false;
    let retryTimer;
    const paletteCache = new Map();
    const neutral = {base: '#242731', primary: '#384253', secondary: '#202b3a'};
    const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)');

    function paletteSource(raw) {
        if (raw === fallbackArtwork) return '';
        const url = new URL(raw, location.origin);
        if (url.pathname.toLowerCase().endsWith('.svg')) return '';
        return ['http:', 'https:'].includes(url.protocol) ? url.href : '';
    }

    function samplePalette(image) {
        try {
            const canvas = document.createElement('canvas');
            canvas.width = canvas.height = 24;
            const context = canvas.getContext('2d', {willReadFrequently: true});
            context.drawImage(image, 0, 0, 24, 24);
            const pixels = context.getImageData(0, 0, 24, 24).data;
            const buckets = new Map();
            for (let index = 0; index < pixels.length; index += 4) {
                if (pixels[index + 3] < 200) continue;
                const rgb = [pixels[index], pixels[index + 1], pixels[index + 2]];
                const high = Math.max(...rgb), low = Math.min(...rgb);
                if (high > 245 && low > 215 || high < 35) continue;
                const key = rgb.map(value => Math.floor(value / 40)).join(':');
                const bucket = buckets.get(key) || {sum: [0, 0, 0], weight: 0};
                const weight = .7 + (high - low) / 255;
                rgb.forEach((value, channel) => bucket.sum[channel] += value * weight);
                bucket.weight += weight;
                buckets.set(key, bucket);
            }
            const colors = [...buckets.values()].filter(bucket => bucket.weight >= 2)
                .sort((a, b) => b.weight - a.weight)
                .map(bucket => bucket.sum.map(value => value / bucket.weight));
            if (!colors.length) return neutral;
            const first = colors[0];
            const second = colors.find(color => Math.hypot(...color.map((value, i) => value - first[i])) > 65) ||
                [first[2], first[0], first[1]];
            const shade = (rgb, strength) => `#${rgb.map((value, i) =>
                Math.round(Math.min(90, Math.max(20, value * strength + [8, 12, 18][i]))).toString(16).padStart(2, '0')).join('')}`;
            return {base: shade(first, .28), primary: shade(first, .46), secondary: shade(second, .38)};
        } catch {
            return neutral;
        }
    }

    function paint() {
        const colors = palette || neutral;
        layer.style.setProperty('--atmosphere-base', colors.base);
        layer.style.setProperty('--atmosphere-primary', colors.primary);
        layer.style.setProperty('--atmosphere-secondary', colors.secondary);
        onColor?.(colors.base);
    }

    function setBackdrop(url) {
        if (backdrop.src === url) return;
        const old = !layer.hidden && !reducedMotion.matches && backdrop.complete && backdrop.naturalWidth ? backdrop.cloneNode() : null;
        if (old) {
            old.removeAttribute('id');
            old.classList.add('player-backdrop-old');
            backdrop.after(old);
        }
        backdrop.src = url;
        if (old) old.animate([{opacity: 1}, {opacity: 0}], {duration: 650, easing: 'ease-out'})
            .finished.catch(() => {}).finally(() => old.remove());
    }

    function loadPalette(id, attempt = 0) {
        clearTimeout(retryTimer);
        const url = paletteSource(source);
        if (!url) {
            palette = neutral;
            paint();
            return;
        }
        if (paletteCache.has(url)) {
            palette = paletteCache.get(url);
            paint();
            return;
        }
        paletteLoading = true;
        const image = new Image();
        image.crossOrigin = 'anonymous';
        image.onload = () => {
            if (id !== generation) return;
            paletteLoading = false;
            clearTimeout(retryTimer);
            palette = samplePalette(image);
            paletteCache.set(url, palette);
            if (paletteCache.size > 48) paletteCache.delete(paletteCache.keys().next().value);
            paint();
        };
        image.onerror = () => {
            if (id !== generation) return;
            paletteLoading = false;
            palette = neutral;
            paint();
            if (attempt < retryDelays.length) retryTimer = setTimeout(() => loadPalette(id, attempt + 1), retryDelays[attempt]);
        };
        image.src = url;
    }

    paint();
    on('dispose', () => {
        generation++;
        clearTimeout(retryTimer);
        layer.hidden = true;
    });
    const refresh = () => {
        paint();
        if (source && !palette && !paletteLoading) loadPalette(generation);
    };
    window.addEventListener('online', refresh);
    window.addEventListener('pageshow', refresh);
    document.addEventListener('visibilitychange', () => {
        if (!document.hidden) refresh();
    });
    return {
        setArtwork(url) {
            if (url === source) return;
            source = url;
            generation++;
            clearTimeout(retryTimer);
            const id = generation;
            if (url === fallbackArtwork) {
                setBackdrop(themedFallback());
                palette = neutral;
                paint();
                return;
            }
            palette = paletteCache.get(paletteSource(url)) || palette;
            paint();
            const image = new Image();
            image.onload = () => {
                if (id === generation) setBackdrop(image.src);
            };
            image.onerror = () => {
                if (id === generation) setBackdrop(themedFallback());
            };
            image.src = url;
            loadPalette(id);
        },
        setExpanded(expanded) {
            const wasHidden = layer.hidden;
            layer.hidden = !expanded;
            if (expanded) {
                paint();
                if (wasHidden && source && !paletteLoading && !paletteCache.has(paletteSource(source))) loadPalette(generation);
            }
        },
        refresh
    };
}
