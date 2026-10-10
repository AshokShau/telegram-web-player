// Telegram custom palettes use the existing semantic tokens and layout.
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
function readable(color, backgrounds) {
    const score = value => Math.min(...backgrounds.map(background => contrast(value, background)));
    if (score(color) >= 4.5) return color;
    const target = score('#ffffff') > score('#000000') ? '#ffffff' : '#000000';
    for (let step = 1; step <= 20; step++) { const candidate = mix(color, target, step / 20); if (score(candidate) >= 4.5) return candidate; }
    return target;
}
export const themeTokens = ['canvas', 'surface', 'raised', 'hover', 'text', 'muted', 'line', 'accent', 'accent-text', 'action', 'danger', 'selection', 'shadow'];
export function telegramPalette(params = {}, scheme = 'dark') {
    params ||= {};
    const light = scheme === 'light';
    const pick = (...values) => values.map(validColor).find(Boolean);
    const canvas = pick(params.bg_color, light ? '#f6f7fb' : '#111318');
    const foreground = pick(params.text_color, light ? '#1d2130' : '#f2f3f8');
    const surface = pick(params.section_bg_color, params.secondary_bg_color, mix(canvas, foreground, .035));
    const raised = mix(surface, foreground, .055), hover = mix(surface, foreground, .1);
    const backgrounds = [canvas, surface, raised, hover];
    const text = readable(foreground, backgrounds);
    const action = pick(params.button_color, light ? '#4659b8' : '#a7b5ff');
    const accent = readable(pick(params.accent_text_color, params.link_color, action), backgrounds);
    return {
        canvas, surface, raised, hover, text,
        muted: readable(pick(params.subtitle_text_color, params.hint_color, mix(surface, text, .65)), backgrounds),
        line: pick(params.section_separator_color, mix(surface, text, .22)), accent, action,
        'accent-text': readable(pick(params.button_text_color, '#ffffff'), [action]),
        danger: readable(pick(params.destructive_text_color, light ? '#a52e2a' : '#ffaaa5'), backgrounds),
        selection: mix(canvas, accent, .2),
        shadow: `0 18px 55px rgb(0 0 0 / ${light ? '14%' : '32%'})`
    };
}
