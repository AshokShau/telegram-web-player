import { state, platform, telegram, on, applyTheme } from './core.js?v=54';

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
        source = url; backdrop.src = url;
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
