// Palette data and material derivation are independent of the DOM and room state.
export const themes = Object.freeze([
    { id: 'telegram', name: 'Telegram', note: 'Match your Telegram app', automatic: true },
    { id: 'system', name: 'System', note: 'Follow your device', automatic: true },
    { id: 'light', name: 'Light', note: 'Ivory & honey', scheme: 'light', canvas: '#eeede9', surface: '#faf9f6', text: '#2b302e', action: '#e8ac55', accent: '#875118' },
    { id: 'dark', name: 'Dark', note: 'Charcoal & amber', scheme: 'dark', canvas: '#101215', surface: '#1c1f24', text: '#f2f3f5', action: '#edb25b' },
    { id: 'midnight', name: 'Midnight', note: 'Ink & cool blue', scheme: 'dark', canvas: '#0c1420', surface: '#182435', text: '#edf3fc', action: '#91c7f0' },
    { id: 'forest', name: 'Forest', note: 'Sage & soft green', scheme: 'light', canvas: '#e6ece2', surface: '#f4f8ee', text: '#29392b', action: '#9fc27f', accent: '#3f672f' },
    { id: 'rose', name: 'Rose', note: 'Blush & mulberry', scheme: 'light', canvas: '#f0e7e8', surface: '#fff6f6', text: '#3c2d35', action: '#dda5b7', accent: '#8b425b' },
    { id: 'dusk', name: 'Dusk', note: 'Smoky plum & lilac', scheme: 'dark', canvas: '#19151e', surface: '#27212e', text: '#f4eff8', action: '#ceb2ed' },
    { id: 'amoled', name: 'AMOLED', note: 'True black & mint', scheme: 'dark', canvas: '#000000', surface: '#111615', text: '#f0f5f3', action: '#8bdbb4' }
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
    // Native colors keep their own palette; the seed determines the material treatment.
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
    const raised = fitSurface(mix(surface, text, .045)), hover = fitSurface(mix(surface, text, light ? .085 : .075));
    const action = validColor(params.button_color) || seed.action;
    const ambientA = id === 'amoled' ? canvas : fitSurface(mix(canvas, light ? action : surface, light ? .045 : .4));
    const ambientB = id === 'amoled' ? canvas : fitSurface(mix(canvas, light ? text : surface, light ? .018 : .15));
    const controlTop = fitSurface(mix(surface, '#ffffff', light ? .6 : .035));
    const controlBottom = fitSurface(mix(surface, light ? text : '#000000', light ? .025 : .045));
    const messageSelf = fitSurface(mix(surface, action, light ? .3 : .12));
    const backgrounds = [canvas, surface, raised, hover, ambientA, ambientB, controlTop, controlBottom, messageSelf];
    const muted = readable(validColor(params.subtitle_text_color) || validColor(params.hint_color) || mix(surface, text, light ? .72 : .65), backgrounds);
    const accent = readable(validColor(params.accent_text_color) || validColor(params.link_color) || seed.accent || action, backgrounds);
    const danger = readable(validColor(params.destructive_text_color) || (light ? '#a63732' : '#f3a6a0'), backgrounds);
    const switchOff = readable(mix(surface, text, .5), backgrounds, 3);
    const actionText = readable(validColor(params.button_text_color) || '#172018', [action]);
    const darkInk = luminance(actionText) < luminance(action);
    const actionTop = darkInk ? mix(action, '#ffffff', .1) : action;
    const actionBottom = darkInk ? action : mix(action, '#000000', .1);
    const highlight = alpha('#ffffff', light ? .9 : .065);
    const controlShadow = `inset 0 1px 0 ${highlight}, 0 2px 3px ${alpha('#000000', light ? .13 : .24)}, 0 5px 10px ${alpha('#000000', light ? .045 : .08)}`;
    return {
        id, scheme: seed.scheme, name: selected.name,
        tokens: {
            canvas, surface, raised, hover, text, muted, line: mix(surface, text, light ? .2 : .11), accent, action,
            'accent-text': actionText,
            danger, 'danger-text': readable('#ffffff', [danger]),
            success: readable(light ? '#287359' : '#90d3ad', backgrounds),
            selection: mix(surface, action, light ? .3 : .2), 'message-self': messageSelf,
            'section-heading': readable(validColor(params.section_header_text_color) || muted, backgrounds),
            'switch-off': switchOff, 'switch-on': readable(action, backgrounds, 3),
            glass: alpha(surface, light ? .76 : .94), 'glass-strong': alpha(surface, light ? .97 : .98),
            'glass-edge': alpha('#ffffff', light ? .9 : .055),
            ambient: id === 'amoled' ? 'none' : `radial-gradient(ellipse at 0% 0%, ${ambientA} 0, transparent 48%), radial-gradient(ellipse at 100% 75%, ${ambientB} 0, transparent 50%)`,
            'player-surface': alpha(light ? canvas : surface, id === 'amoled' ? 1 : light ? .97 : .98),
            control: `linear-gradient(180deg, ${controlTop}, ${controlBottom})`,
            'control-action': `linear-gradient(180deg, ${actionTop}, ${actionBottom})`,
            'control-border': alpha(text, light ? .16 : .09), 'control-edge': readable(muted, backgrounds, 3), 'range-track': switchOff,
            'control-highlight': highlight, 'shadow-control': controlShadow,
            'shadow-action': `inset 0 1px 0 ${alpha('#ffffff', light ? .3 : .17)}, 0 2px 3px ${alpha('#000000', light ? .18 : .22)}, 0 5px 12px ${alpha('#000000', .1)}`,
            'shadow-pressed': `inset 0 2px 4px ${alpha('#000000', light ? .2 : .5)}, 0 1px 0 ${highlight}`,
            'shadow-inset': `inset 0 2px 4px ${alpha('#000000', light ? .1 : .4)}, 0 1px 0 ${highlight}`,
            'shadow-surface': `inset 0 1px 0 ${alpha('#ffffff', light ? .9 : .055)}, 0 8px 24px ${alpha('#000000', light ? .05 : .14)}`,
            shadow: `0 18px 55px ${alpha('#000000', light ? .18 : .5)}`
        }
    };
}
