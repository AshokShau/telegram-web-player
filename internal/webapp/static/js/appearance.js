import { createPreferences } from './preferences.js?v=88';

export function createAppearance({ telegram = null, native = false, preferences = createPreferences(), onSelect = () => {} } = {}) {
    const root = document.documentElement;
    const inputs = [...document.querySelectorAll('input[name="theme"]')];
    function paint() {
        const selected = preferences.get('theme') === 'light' ? 'light' : 'dark';
        root.dataset.theme = selected;
        root.dataset.scheme = selected;
        for (const input of inputs) input.checked = input.value === selected;
        const canvas = getComputedStyle(root).getPropertyValue('--canvas').trim();
        const meta = document.querySelector('meta[name="theme-color"]');
        if (meta) meta.content = canvas;
        if (native) {
            for (const method of ['setHeaderColor', 'setBackgroundColor', 'setBottomBarColor']) {
                try { telegram?.[method]?.(canvas); } catch { /* Older Telegram clients may not support color APIs. */ }
            }
        }
        if (preferences.get('theme') !== selected) preferences.set('theme', selected);
    }
    function choose(event) {
        if (!event.target.checked) return;
        preferences.set('theme', event.target.value === 'light' ? 'light' : 'dark');
        paint(); onSelect();
    }
    for (const input of inputs) input.addEventListener('change', choose);
    const storageChanged = event => { if (event.key === 'synctune:theme' || event.key === null) paint(); };
    window.addEventListener('storage', storageChanged);
    if (native) telegram?.onEvent?.('themeChanged', paint);
    paint();
    return {
        dispose() {
            for (const input of inputs) input.removeEventListener('change', choose);
            window.removeEventListener('storage', storageChanged);
            if (native) telegram?.offEvent?.('themeChanged', paint);
        }
    };
}

if (!document.getElementById('app')) createAppearance();
