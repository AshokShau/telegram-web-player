import { createPreferences } from './preferences.js?v=86';

// Keep the preference key and storage bridge so retired themes migrate safely.
export function createAppearance({ telegram = null, native = false, preferences = createPreferences() } = {}) {
    const root = document.documentElement;
    function paint() {
        root.dataset.theme = 'dark';
        root.dataset.scheme = 'dark';
        const canvas = getComputedStyle(root).getPropertyValue('--canvas').trim();
        const meta = document.querySelector('meta[name="theme-color"]');
        if (meta) meta.content = canvas;
        if (native) {
            for (const method of ['setHeaderColor', 'setBackgroundColor', 'setBottomBarColor']) {
                try { telegram?.[method]?.(canvas); } catch { /* Older Telegram clients may not support color APIs. */ }
            }
        }
        if (preferences.get('theme') !== 'dark') preferences.set('theme', 'dark');
    }
    const storageChanged = event => { if (event.key === 'synctune:theme' || event.key === null) paint(); };
    window.addEventListener('storage', storageChanged);
    if (native) telegram?.onEvent?.('themeChanged', paint);
    paint();
    return {
        dispose() {
            window.removeEventListener('storage', storageChanged);
            if (native) telegram?.offEvent?.('themeChanged', paint);
        }
    };
}

if (!document.getElementById('app')) createAppearance();
