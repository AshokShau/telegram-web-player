import { createPreferences } from './preferences.js?v=111';

export function createAppearance({ telegram = null, native = false, preferences = createPreferences(), onSelect = () => {} } = {}) {
    const root = document.documentElement;
    const inputs = [...document.querySelectorAll('input[name="theme"]')];
    const systemTheme = matchMedia('(prefers-color-scheme: light)');
    function paint() {
        const choice = ['light', 'dark'].includes(preferences.get('theme')) ? preferences.get('theme') : 'system';
        const selected = choice === 'system' ? (native && telegram?.colorScheme ? telegram.colorScheme : systemTheme.matches ? 'light' : 'dark') : choice;
        root.dataset.theme = selected === 'light' ? 'light' : 'dark';
        root.dataset.scheme = root.dataset.theme;
        for (const input of inputs) input.checked = input.value === choice;
        const canvas = getComputedStyle(root).getPropertyValue('--canvas').trim();
        const meta = document.querySelector('meta[name="theme-color"]');
        if (meta) meta.content = canvas;
        if (native) {
            for (const method of ['setHeaderColor', 'setBackgroundColor', 'setBottomBarColor']) {
                try { telegram?.[method]?.(canvas); } catch { /* Older Telegram clients may not support color APIs. */ }
            }
        }
    }
    function choose(event) {
        if (!event.target.checked) return;
        preferences.set('theme', event.target.value);
        paint(); onSelect(true);
    }
    for (const input of inputs) input.addEventListener('change', choose);
    const refresh = () => { paint(); onSelect(false); };
    const storageChanged = event => { if (event.key === 'synctune:theme' || event.key === null) refresh(); };
    window.addEventListener('storage', storageChanged);
    systemTheme.addEventListener('change', refresh);
    if (native) telegram?.onEvent?.('themeChanged', refresh);
    paint();
    return {
        dispose() {
            for (const input of inputs) input.removeEventListener('change', choose);
            window.removeEventListener('storage', storageChanged);
            systemTheme.removeEventListener('change', refresh);
            if (native) telegram?.offEvent?.('themeChanged', refresh);
        }
    };
}

if (!document.getElementById('app')) createAppearance();
