import { themes, normalizeTheme, resolveTheme } from './theme.js?v=83';
import { createPreferences } from './preferences.js?v=83';

const browserPreferences = createPreferences();

// Owns device appearance only; changing colors never touches the room or media.
export function createAppearance({ telegram = null, native = false, preferences = browserPreferences, onSelect = () => {} } = {}) {
    const root = document.documentElement;
    const system = matchMedia('(prefers-color-scheme: dark)');
    const fallback = 'system';
    let selected = normalizeTheme(preferences.get('theme', fallback), fallback);
    let group = null;
    const cards = new Map();
    const resolve = id => resolveTheme(id, { systemDark: system.matches, native, telegram });

    function paint() {
        const theme = resolve(selected);
        for (const [token, value] of Object.entries(theme.tokens)) root.style.setProperty(`--${token}`, value);
        root.dataset.theme = selected;
        root.dataset.scheme = theme.scheme;
        root.style.colorScheme = theme.scheme;
        const meta = document.querySelector('meta[name="theme-color"]');
        if (meta) meta.content = theme.tokens.canvas;
        if (native) {
            for (const method of ['setHeaderColor', 'setBackgroundColor', 'setBottomBarColor']) {
                try { telegram[method]?.(theme.tokens.canvas); } catch { /* Color APIs vary across Telegram clients. */ }
            }
        }
        for (const [id, card] of cards) {
            const palette = resolve(id).tokens;
            for (const token of ['canvas', 'surface', 'raised', 'action', 'text', 'muted']) card.style.setProperty(`--sample-${token}`, palette[token]);
            card.setAttribute('aria-checked', String(id === selected));
            card.tabIndex = id === selected ? 0 : -1;
        }
        const value = document.getElementById('profile-theme-value');
        if (value) value.textContent = theme.name;
        const summary = document.getElementById('appearance-summary');
        if (summary) {
            summary.textContent = selected === 'telegram'
                ? native ? 'Uses your Telegram theme.' : 'Telegram colors are available in the Mini App. Here, your device appearance is used.'
                : selected === 'system' ? `Uses your device’s ${theme.scheme} mode.`
                : `${theme.name} theme · Saved on this device`;
        }
    }
    function choose(id) {
        const next = normalizeTheme(id, fallback);
        preferences.set('theme', next);
        if (next === selected) return;
        selected = next;
        paint(); onSelect();
    }
    function handleKeys(event) {
        const ids = themes.map(theme => theme.id);
        const current = ids.indexOf(event.target.dataset.themeChoice);
        if (current < 0 || !['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return;
        event.preventDefault();
        const columns = getComputedStyle(group).gridTemplateColumns.split(' ').length;
        const step = ({ ArrowLeft: -1, ArrowRight: 1, ArrowUp: -columns, ArrowDown: columns })[event.key];
        const next = event.key === 'Home' ? 0 : event.key === 'End' ? ids.length - 1 : (current + step + ids.length) % ids.length;
        choose(ids[next]); cards.get(ids[next]).focus({ preventScroll: true });
        cards.get(ids[next]).scrollIntoView({ block: 'nearest' });
    }
    const systemChanged = () => { if (selected === 'system' || (selected === 'telegram' && !native)) paint(); else paintCards(); };
    const telegramChanged = () => { if (selected === 'telegram') paint(); else paintCards(); };
    function paintCards() {
        // Also update the automatic-mode previews when a fixed palette is active.
        for (const id of ['telegram', 'system']) {
            const card = cards.get(id);
            if (!card) continue;
            const palette = resolve(id).tokens;
            for (const token of ['canvas', 'surface', 'raised', 'action', 'text', 'muted']) card.style.setProperty(`--sample-${token}`, palette[token]);
        }
    }
    const storageChanged = event => { if (event.key === 'synctune:theme' || event.key === null) { selected = normalizeTheme(preferences.get('theme', fallback), fallback); paint(); } };
    system.addEventListener('change', systemChanged);
    window.addEventListener('storage', storageChanged);
    if (native) telegram.onEvent?.('themeChanged', telegramChanged);
    paint();

    return {
        mount({ element, icon }) {
            group = document.getElementById('theme-presets');
            for (const theme of themes) {
                const miniature = element('span', { class: 'theme-miniature', 'aria-hidden': 'true' }, [
                    element('span', { class: 'theme-mini-sidebar' }),
                    element('span', { class: 'theme-mini-art' }, [icon('music')]),
                    element('span', { class: 'theme-mini-lines' }, [element('i'), element('i')]),
                    element('span', { class: 'theme-mini-transport' }, [icon('play')])
                ]);
                const card = element('button', { type: 'button', class: 'theme-choice', role: 'radio', 'data-theme-choice': theme.id, 'aria-label': `${theme.name}: ${theme.note}` }, [
                    miniature,
                    element('span', { class: 'theme-choice-copy' }, [element('strong', { text: theme.name }), element('small', { text: theme.note })]),
                    element('span', { class: 'theme-choice-check', 'aria-hidden': 'true' }, [icon('check')])
                ]);
                card.addEventListener('click', () => choose(theme.id));
                cards.set(theme.id, card); group.append(card);
            }
            group.addEventListener('keydown', handleKeys);
            paint();
        },
        dispose() {
            system.removeEventListener('change', systemChanged);
            window.removeEventListener('storage', storageChanged);
            if (native) telegram.offEvent?.('themeChanged', telegramChanged);
            group?.removeEventListener('keydown', handleKeys);
        }
    };
}

// The same preference follows listeners onto the homepage and privacy document.
if (!document.getElementById('app')) createAppearance();
