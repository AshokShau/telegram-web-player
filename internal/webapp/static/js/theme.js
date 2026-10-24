// Semantic palette derivation stays independent of the DOM and room state.
export const themes = Object.freeze([
    { id: 'telegram', name: 'Telegram', note: 'Match Telegram’s colors', automatic: true },
    { id: 'system', name: 'System', note: 'Follow device settings', automatic: true },
    { id: 'light', name: 'Light', note: 'Telegram blue & white', scheme: 'light', canvas: '#f0f2f5', surface: '#ffffff', text: '#202124', action: '#2178b9', accent: '#1673b5' },
    { id: 'dark', name: 'Dark', note: 'Black & Telegram blue', scheme: 'dark', canvas: '#000000', surface: '#181818', raised: '#252525', muted: '#a0a0a0', line: '#272727', text: '#f5f5f5', action: '#62ade4', accent: '#7abbed' },
    { id: 'midnight', name: 'Midnight', note: 'Ink & cool blue', scheme: 'dark', canvas: '#17212b', surface: '#1f2c38', text: '#edf3fc', action: '#69b4ee' },
    { id: 'forest', name: 'Forest', note: 'Sage & soft green', scheme: 'light', canvas: '#e6ece2', surface: '#f4f8ee', text: '#29392b', action: '#9fc27f', accent: '#3f672f' },
    { id: 'rose', name: 'Rose', note: 'Blush & mulberry', scheme: 'light', canvas: '#f0e7e8', surface: '#fff6f6', text: '#3c2d35', action: '#dda5b7', accent: '#8b425b' },
    { id: 'dusk', name: 'Dusk', note: 'Smoky plum & lilac', scheme: 'dark', canvas: '#19151e', surface: '#27212e', text: '#f4eff8', action: '#ceb2ed' },
    { id: 'amoled', name: 'AMOLED', note: 'True black & blue', scheme: 'dark', canvas: '#000000', surface: '#141414', text: '#f5f5f5', action: '#62ade4' }
]);

export function validColor(value) {
    return typeof value === 'string' && /^#[\da-f]{6}$/i.test(value) ? value.toLowerCase() : null;
}
const channels = color => [1, 3, 5].map(offset => parseInt(color.slice(offset, offset + 2), 16));
function mix(a, b, amount) {
    const target = channels(b);
    return '#' + channels(a).map((value, index) => Math.round(value + (target[index] - value) * amount).toString(16).padStart(2, '0')).join('');
}
const alpha = (color, opacity) => `rgba(${channels(color).join(', ')}, ${opacity})`;
function luminance(color) {
    return channels(color).map(value => value / 255).map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4)
        .reduce((sum, value, index) => sum + value * [.2126, .7152, .0722][index], 0);
}
export function contrast(a, b) {
    const values = [luminance(a), luminance(b)].sort((x, y) => x - y);
    return (values[1] + .05) / (values[0] + .05);
}
function readable(color, backgrounds, minimum = 4.5) {
    const score = candidate => Math.min(...backgrounds.map(background => contrast(candidate, background)));
    if (score(color) >= minimum) return color;
    const target = score('#ffffff') > score('#000000') ? '#ffffff' : '#000000';
    for (let step = 1; step <= 100; step++) {
        const candidate = mix(color, target, step / 100);
        if (score(candidate) >= minimum) return candidate;
    }
    return target;
}

// Migrate valid saved choices without bringing back the previous theme engine.
export function normalizeTheme(id, fallback = 'system') {
    const migrated = ({ 'full-dark': 'amoled', 'amoled-violet': 'amoled', classic: 'forest', arctic: 'light', night: 'midnight' })[id] || id;
    return themes.some(theme => theme.id === migrated) ? migrated : fallback;
}

export function resolveTheme(id, { systemDark = false, native = false, telegram = null } = {}) {
    id = normalizeTheme(id);
    const followsTelegram = id === 'telegram' && native;
    const automaticDark = followsTelegram ? telegram?.colorScheme === 'dark' : systemDark;
    const selected = themes.find(theme => theme.id === id);
    const fallback = themes.find(theme => theme.id === (automaticDark ? 'dark' : 'light'));
    const seed = selected.automatic ? fallback : selected;
    const params = followsTelegram ? telegram?.themeParams || {} : {};
    // Native colors keep their own palette and accessibility corrections.
    const light = seed.scheme === 'light';
    const canvas = validColor(params.secondary_bg_color) || validColor(params.bg_color) || seed.canvas;
    const requestedText = validColor(params.text_color) || seed.text;
    let surface = validColor(params.section_bg_color) || validColor(params.bg_color) || seed.surface;
    // Opposite-luminance section backgrounds cannot share a readable foreground.
    // Keep native color intent while narrowing only inaccessible extremes.
    if (Math.max(Math.min(contrast('#ffffff', canvas), contrast('#ffffff', surface)), Math.min(contrast('#000000', canvas), contrast('#000000', surface))) < 4.5) surface = mix(canvas, surface, .12);
    let text = readable(requestedText, [canvas, surface]);
    if ([canvas, surface].some(background => contrast(text, background) < 4.5)) {
        surface = canvas;
        text = readable(requestedText, [canvas]);
    }
    const fitSurface = candidate => {
        if (contrast(text, candidate) >= 4.5) return candidate;
        for (let step = 1; step <= 100; step++) {
            const fitted = mix(candidate, canvas, step / 100);
            if (contrast(text, fitted) >= 4.5) return fitted;
        }
        return canvas;
    };
    const raised = fitSurface((!followsTelegram && seed.raised) || mix(surface, text, .045));
    const hover = fitSurface(mix(surface, text, light ? .085 : .075));
    const action = validColor(params.button_color) || seed.action;
    const messageSelf = fitSurface(mix(surface, action, light ? .3 : .12));
    const backgrounds = [canvas, surface, raised, hover, messageSelf];
    const muted = readable(validColor(params.subtitle_text_color) || validColor(params.hint_color) || seed.muted || mix(surface, text, light ? .72 : .65), backgrounds);
    const accent = readable(validColor(params.accent_text_color) || validColor(params.link_color) || seed.accent || action, backgrounds);
    const danger = readable(validColor(params.destructive_text_color) || (light ? '#a63732' : '#f3a6a0'), backgrounds);
    const switchOff = readable(mix(surface, text, .5), backgrounds, 3);
    const actionText = readable(validColor(params.button_text_color) || '#ffffff', [action]);
    return {
        id, scheme: seed.scheme, name: selected.name,
        tokens: {
            canvas, surface, raised, hover, text, muted, line: (!followsTelegram && seed.line) || mix(surface, text, light ? .2 : .11), accent, action,
            'accent-text': actionText,
            danger, 'danger-text': readable('#ffffff', [danger]),
            success: readable(light ? '#287359' : '#90d3ad', backgrounds),
            selection: fitSurface(mix(surface, action, light ? .3 : .12)), 'message-self': messageSelf,
            'section-heading': readable(validColor(params.section_header_text_color) || accent, backgrounds),
            'switch-off': switchOff, 'switch-on': readable(action, backgrounds, 3),
            // Keep existing semantic names for public pages and older consumers.
            glass: surface, 'glass-strong': surface, 'glass-edge': 'transparent', ambient: 'none',
            'player-surface': light ? canvas : surface,
            control: raised, 'control-action': action,
            'control-border': 'transparent', 'control-edge': readable(muted, backgrounds, 3), 'range-track': switchOff,
            'control-highlight': 'transparent', 'shadow-control': 'none',
            'shadow-action': 'none', 'shadow-pressed': 'none', 'shadow-inset': 'none', 'shadow-surface': 'none',
            shadow: `0 18px 55px ${alpha('#000000', light ? .18 : .5)}`
        }
    };
}
