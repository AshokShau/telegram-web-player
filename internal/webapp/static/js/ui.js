// Safe DOM primitives shared by room presentation and track lists.
import { safeURL } from './core.js?v=62';

export function element(tag, attrs = {}, children = []) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(attrs)) {
        if (key === 'text') node.textContent = value;
        else if (key === 'class') node.className = value;
        else if (value !== undefined && value !== null) node.setAttribute(key, String(value));
    }
    node.append(...children); return node;
}
export function icon(name) {
    const node = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    node.setAttribute('class', 'icon'); node.setAttribute('aria-hidden', 'true');
    const use = document.createElementNS(node.namespaceURI, 'use'); use.setAttribute('href', `/static/assets/icons.svg?v=58#${name}`); node.append(use); return node;
}
export function actionButton(action, label, iconName, fields = {}) {
    return element('button', { class: 'icon-button', type: 'button', 'aria-label': label, title: label, 'data-action': action, ...fields }, [icon(iconName)]);
}

export function empty(message, detail = '', iconName = 'music', compact = false) {
    return element('div', { class: `empty-state${compact ? ' compact' : ''}` }, compact ? [element('p', { text: message })] : [icon(iconName), element('h3', { text: message }), element('p', { text: detail })]);
}

export function trackCount(count) { return `${count} ${count === 1 ? 'track' : 'tracks'}`; }
export function normalizeSong(song) { return { id: song.track_id, title: song.name, artist: song.artist || '', thumbnail: song.thumbnail || '', duration: song.duration, platform: song.platform, url: song.url }; }

export function avatar(node, user) {
    const name = user.first_name || user.firstName || 'Listener';
    const src = safeURL(user.photo_url || user.photoUrl);
    node.replaceChildren(element('span', { text: Array.from(name)[0].toUpperCase() }));
    if (src) {
        const image = element('img', { alt: '', 'data-avatar': 'true', hidden: true });
        image.addEventListener('load', () => { image.hidden = false; });
        image.addEventListener('error', () => image.remove());
        node.append(image); image.src = src;
    }
}

// A static skeleton communicates loading without continuous animation.
export function loadingState(message, count = 3) {
    return element('div', { class: 'loading-state', role: 'status' }, [
        element('span', { class: 'sr-only', text: message }),
        ...Array.from({ length: count }, () => element('div', { class: 'track-skeleton', 'aria-hidden': 'true' }, [
            element('span', { class: 'skeleton-art' }),
            element('span', { class: 'skeleton-lines' }, [element('span'), element('span')])
        ]))
    ]);
}
