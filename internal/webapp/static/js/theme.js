// Theme palettes share semantic tokens and contrast-aware color resolution.
const validColor = value => typeof value === 'string' && /^#[\da-f]{6}$/i.test(value) ? value.toLowerCase() : null;
const channels = color => [1, 3, 5].map(offset => parseInt(color.slice(offset, offset + 2), 16));
function mix(base, tint, amount) {
    const a = channels(base), b = channels(tint);
    return '#' + a.map((value, index) => Math.round(value * (1 - amount) + b[index] * amount).toString(16).padStart(2, '0')).join('');
}
function luminance(color) {
    const values = channels(color).map(value => { const s = value / 255; return s <= .04045 ? s / 12.92 : ((s + .055) / 1.055) ** 2.4; });
    return values[0] * .2126 + values[1] * .7152 + values[2] * .0722;
}
function contrast(a, b) { const x = luminance(a), y = luminance(b); return (Math.max(x, y) + .05) / (Math.min(x, y) + .05); }
function readable(color, backgrounds, minimum = 4.5) {
    const score = value => Math.min(...backgrounds.map(background => contrast(value, background)));
    if (score(color) >= minimum) return color;
    const target = score('#ffffff') > score('#000000') ? '#ffffff' : '#000000';
    for (let step = 1; step <= 20; step++) { const candidate = mix(color, target, step / 20); if (score(candidate) >= minimum) return candidate; }
    return target;
}
export const themeTokens = ['canvas', 'surface', 'raised', 'hover', 'text', 'muted', 'line', 'accent', 'section-heading', 'switch-off', 'switch-on', 'accent-text', 'action', 'danger', 'selection', 'shadow'];
export function telegramPalette(params = {}, scheme = 'dark') {
    params ||= {};
    const light = scheme === 'light';
    const pick = (...values) => values.map(validColor).find(Boolean);
    // Telegram recommends section_bg_color on secondary_bg_color for grouped settings.
    // https://core.telegram.org/bots/webapps#themeparams
    const canvas = pick(params.secondary_bg_color, params.bg_color, light ? '#f6f7fb' : '#10151c');
    const foreground = pick(params.text_color, light ? '#1d2130' : '#f1f3f6');
    const surface = pick(params.section_bg_color, params.bg_color, params.secondary_bg_color, mix(canvas, foreground, .035));
    const raised = mix(surface, foreground, .055), hover = mix(surface, foreground, .1);
    const backgrounds = [canvas, surface, raised, hover];
    const text = readable(foreground, backgrounds);
    const action = pick(params.button_color, light ? '#32689e' : '#528bc5');
    const accent = readable(pick(params.accent_text_color, params.link_color, action), backgrounds);
    return {
        canvas, surface, raised, hover, text,
        muted: readable(pick(params.subtitle_text_color, params.hint_color, mix(surface, text, .65)), backgrounds),
        line: pick(params.section_separator_color, mix(surface, text, .22)), accent, action,
        'section-heading': readable(pick(params.section_header_text_color, params.accent_text_color, params.link_color, action), [surface]),
        // Native switch colors are not exposed by ThemeParams; derive visible controls.
        'switch-off': readable(mix(surface, text, .35), [surface], 3),
        'switch-on': readable(action, [surface], 3),
        'accent-text': readable(pick(params.button_text_color, '#ffffff'), [action]),
        danger: readable(pick(params.destructive_text_color, light ? '#a52e2a' : '#ffaaa5'), backgrounds),
        selection: mix(canvas, accent, .2),
        shadow: `0 18px 55px rgb(0 0 0 / ${light ? '14%' : '32%'})`
    };
}

const dark = {
    canvas: '#10151c', surface: '#18232e', raised: '#222f3b', hover: '#2b3c4c', text: '#f1f3f6', muted: '#97a7b7', line: '#263440',
    accent: '#78aadd', action: '#528bc5', 'accent-text': '#07131f', danger: '#ffaaa5', selection: '#254463', shadow: '0 18px 55px rgb(2 5 14 / 32%)'
};
const light = {
    canvas: '#f6f7fb', surface: '#ffffff', raised: '#eceef6', hover: '#e1e4f0', text: '#1d2130', muted: '#596174', line: '#d4d8e4',
    accent: '#32689e', action: '#32689e', 'accent-text': '#ffffff', danger: '#a52e2a', selection: '#d8e6f5', shadow: '0 18px 55px rgb(30 38 74 / 14%)'
};
function preset(id, name, scheme, canvas, surface, action) {
    return { id, name, scheme, palette: telegramPalette({ bg_color: canvas, section_bg_color: surface, button_color: action, accent_text_color: action }, scheme) };
}
export const themePresets = [
    { id: 'telegram', name: 'Telegram' }, { id: 'system', name: 'System' },
    { id: 'light', name: 'Light', scheme: 'light', palette: light },
    { id: 'dark', name: 'Dark', scheme: 'dark', palette: dark },
    { id: 'full-dark', name: 'Full Dark', scheme: 'dark', palette: { ...dark, canvas: '#000000', surface: '#0a0a0b', raised: '#17171a', hover: '#24242a', text: '#f4f4f7', muted: '#a8a8b2', line: '#303038', shadow: '0 18px 55px rgb(0 0 0 / 36%)' } },
    preset('classic', 'Classic', 'light', '#dce6dc', '#f1f5e9', '#48744b'),
    preset('arctic', 'Arctic', 'light', '#eef5f7', '#ffffff', '#327e95'),
    preset('night', 'Night', 'dark', '#18191b', '#25272a', '#72a9dd'),
    preset('amoled-violet', 'AMOLED Violet', 'dark', '#000000', '#0b0811', '#c4a2f0')
];
export const accentPresets = [
    { id: 'default', name: 'Theme default' }, { id: 'blue', name: 'Blue', color: '#528bc5' },
    { id: 'green', name: 'Green', color: '#279e69' }, { id: 'cyan', name: 'Cyan', color: '#2d91b5' },
    { id: 'violet', name: 'Violet', color: '#a664cf' }, { id: 'teal', name: 'Teal', color: '#169d93' },
    { id: 'rose', name: 'Rose', color: '#c84f82' }, { id: 'amber', name: 'Amber', color: '#c69131' }
];
export function withAccent(palette, id) {
    const color = accentPresets.find(item => item.id === id)?.color;
    if (!color) return palette;
    return { ...palette, action: color, accent: readable(color, [palette.canvas, palette.surface, palette.raised, palette.hover]),
        'section-heading': readable(color, [palette.surface]), 'switch-on': readable(color, [palette.surface], 3),
        'accent-text': readable('#ffffff', [color]), selection: mix(palette.canvas, color, .2) };
}
export function validateCustomTheme(value) {
    if (!value || typeof value.name !== 'string') return null;
    const name = value.name.trim().slice(0, 32), canvas = validColor(value.canvas), accent = validColor(value.accent);
    return name && canvas && accent ? { name, canvas, accent } : null;
}
export function customThemePreset(value) {
    const custom = validateCustomTheme(value);
    if (!custom) return null;
    const scheme = contrast(custom.canvas, '#000000') > contrast(custom.canvas, '#ffffff') ? 'light' : 'dark';
    // Mid-tone backgrounds need surfaces to move away from the text's color;
    // lightening every dark surface can leave neither white nor black readable.
    const foreground = scheme === 'light' ? '#000000' : '#ffffff';
    const tint = scheme === 'light' || luminance(custom.canvas) < .08 ? '#ffffff' : '#000000';
    const surface = mix(custom.canvas, tint, .035), raised = mix(custom.canvas, tint, .09), hover = mix(custom.canvas, tint, .13);
    const backgrounds = [custom.canvas, surface, raised, hover];
    const palette = telegramPalette({ bg_color: custom.canvas, button_color: custom.accent }, scheme);
    return { id: 'custom', name: custom.name, scheme, palette: {
        ...palette, surface, raised, hover, text: readable(foreground, backgrounds),
        muted: readable(mix(surface, foreground, .65), backgrounds), accent: readable(custom.accent, backgrounds),
        'section-heading': readable(custom.accent, [surface]),
        'switch-off': readable(mix(surface, foreground, .35), [surface], 3), 'switch-on': readable(custom.accent, [surface], 3),
        line: mix(surface, foreground, .22), danger: readable(scheme === 'light' ? '#a52e2a' : '#ffaaa5', backgrounds)
    } };
}
