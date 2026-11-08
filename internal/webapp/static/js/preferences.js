const prefix = 'synctune:';
export const preferenceKeys = ['theme', 'volume', 'music-muted', 'noise-suppression'];
const allowed = key => preferenceKeys.includes(key) || /^(?:history|history-last-track|music-shelves):-?\d+$/.test(key);

function clean(key, value) {
    if (key.startsWith('history-last-track:')) return value === null || typeof value === 'string' && value.length <= 4096 ? value : undefined;
    if (key.startsWith('music-shelves:')) {
        if (!value || typeof value !== 'object') return undefined;
        const items = (list, kind) => Array.isArray(list) ? list
            .filter(item => item && typeof item.id === 'string' && item.id.length <= 120 && typeof item.title === 'string')
            .slice(0, 24)
            .map(item => ({ id: item.id, title: item.title.slice(0, 200), artist: typeof item.artist === 'string' ? item.artist.slice(0, 200) : '', thumbnail: typeof item.thumbnail === 'string' ? item.thumbnail.slice(0, 1024) : '', kind })) : [];
        return {
            loadedAt: Number.isFinite(value.loadedAt) ? Math.min(Date.now(), Math.max(0, value.loadedAt)) : 0,
            usedHistory: Boolean(value.usedHistory),
            playlistTitle: typeof value.playlistTitle === 'string' ? value.playlistTitle.slice(0, 120) : '',
            opened: Array.isArray(value.opened) ? value.opened.filter(id => typeof id === 'string' && id.length <= 120).slice(0, 6) : [],
            albums: items(value.albums, 'album'), playlists: items(value.playlists, 'playlist'), mixes: items(value.mixes, 'mix')
        };
    }
    if (key === 'theme') return typeof value === 'string' && value.length <= 80 ? value : undefined;
    if (key === 'volume') return typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= 1 ? value : undefined;
    if (key === 'music-muted' || key === 'noise-suppression') return typeof value === 'boolean' ? value : undefined;
    if (!Array.isArray(value)) return undefined;
    return value.filter(track => track && ['string', 'number'].includes(typeof track.id) && typeof track.title === 'string').slice(0, 20).map(track => {
        const saved = { id: track.id };
        for (const field of ['title', 'artist', 'url', 'thumbnail', 'platform', 'user']) if (typeof track[field] === 'string') saved[field] = track[field].slice(0, 4096);
        if (Number.isFinite(track.duration)) saved.duration = Math.max(0, track.duration);
        return saved;
    });
}
function parse(key, raw) {
    try { return clean(key, JSON.parse(raw)); } catch { return undefined; }
}
export function telegramDeviceStorage(telegram, native) {
    if (!native) return null;
    try {
        if (telegram?.isVersionAtLeast && !telegram.isVersionAtLeast('9.0')) return null;
        const storage = telegram?.DeviceStorage;
        return typeof storage?.getItem === 'function' && typeof storage?.setItem === 'function' ? storage : null;
    } catch { return null; }
}

export function createPreferences({ deviceStorage = null, timeout = 2000 } = {}) {
    const memory = new Map(), blocked = new Set(), dirty = new Set(), writable = new Set();
    let loading = Promise.resolve(), writing = Promise.resolve();
    function cache(key, value) {
        memory.set(key, value);
        try { localStorage.setItem(prefix + key, JSON.stringify(value)); blocked.delete(key); }
        catch { blocked.add(key); }
    }
    function get(key, fallback) {
        if (!allowed(key)) return fallback;
        if (!blocked.has(key)) {
            try {
                const value = parse(key, localStorage.getItem(prefix + key));
                if (value !== undefined) return value;
                if (!deviceStorage) return fallback;
            } catch {}
        }
        return memory.get(key) ?? fallback;
    }
    function request(method, key, value) {
        return new Promise(resolve => {
            const done = (error, value) => { clearTimeout(timer); resolve(error ? null : { value }); };
            const timer = setTimeout(() => resolve(null), timeout);
            try {
                const name = 'synctune_' + key.replaceAll(':', '_');
                if (method === 'getItem') deviceStorage.getItem(name, done);
                else deviceStorage.setItem(name, JSON.stringify(value), done);
            } catch { done(true); }
        });
    }
    return {
        get,
        load(keys) {
            loading = Promise.all(keys.filter(allowed).map(async key => {
                if (!deviceStorage) return;
                const saved = get(key), result = await request('getItem', key);
                if (!result) return;
                writable.add(key);
                if (dirty.has(key)) return;
                const restored = parse(key, result.value);
                if (restored !== undefined) cache(key, restored);
                else if (saved !== undefined) { cache(key, saved); await request('setItem', key, saved); }
            }));
            return loading;
        },
        set(key, value) {
            if (!allowed(key) || (value = clean(key, value)) === undefined) return Promise.resolve(false);
            dirty.add(key); cache(key, value);
            if (!deviceStorage) return Promise.resolve(true);
            writing = writing.then(() => loading).then(async () => {
                if (!writable.has(key)) return false;
                return (await request('setItem', key, value))?.value === true;
            });
            return writing;
        },
        flush() { return Promise.all([loading, writing]); }
    };
}
