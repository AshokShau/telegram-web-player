const prefix = 'synctune:';
const cloudChunkLimit = 3800;
export const historyLimit = 50;
export const preferenceKeys = ['theme', 'volume', 'music-muted', 'noise-suppression'];
const allowed = key => preferenceKeys.includes(key) || /^(?:history|history-last-track|music-shelves):-?\d+$/.test(key);
const syncsToCloud = key => key === 'theme' || key.startsWith('history:');

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
    return value.filter(track => track && ['string', 'number'].includes(typeof track.id) && typeof track.title === 'string').slice(0, historyLimit).map(track => {
        const saved = { id: track.id };
        for (const field of ['title', 'artist', 'url', 'thumbnail', 'platform']) if (typeof track[field] === 'string') saved[field] = track[field].slice(0, 4096);
        if (Number.isFinite(track.duration)) saved.duration = Math.max(0, track.duration);
        return saved;
    });
}
function parse(key, raw) {
    try { return clean(key, JSON.parse(raw)); } catch { return undefined; }
}
function cloudHistoryName(key) { return `synctune_h_${key.slice('history:'.length)}`; }
function cloudHistoryPart(name, slot, index) { return `${name}_${slot}_${index}`; }
function cloudHistoryChunks(history) {
    const compact = history.map(track => [
        track.id, track.title.slice(0, 200), (track.artist || '').slice(0, 160),
        (track.url || '').slice(0, 2048), (track.thumbnail || '').slice(0, 512),
        (track.platform || '').slice(0, 60), track.duration || 0
    ]);
    const chunks = [];
    let current = [];
    for (const track of compact) {
        const next = JSON.stringify([...current, track]);
        if (next.length > cloudChunkLimit && current.length) { chunks.push(JSON.stringify(current)); current = [track]; }
        else current.push(track);
        if (JSON.stringify(current).length > cloudChunkLimit) return null;
    }
    chunks.push(JSON.stringify(current));
    return chunks;
}
function cloudHistoryValue(key, parts) {
    try {
        const records = parts.flatMap(part => JSON.parse(part));
        if (records.length > historyLimit || records.some(record => !Array.isArray(record) || ![7, 8].includes(record.length))) return undefined;
        const value = clean(key, records.map(record => {
            const [id, title, artist, url, thumbnail, platform] = record;
            return { id, title, artist, url, thumbnail, platform, duration: record[record.length - 1] };
        }));
        return value?.length === records.length ? value : undefined;
    } catch { return undefined; }
}
export function telegramDeviceStorage(telegram, native) {
    if (!native) return null;
    try {
        if (telegram?.isVersionAtLeast && !telegram.isVersionAtLeast('9.0')) return null;
        const storage = telegram?.DeviceStorage;
        return typeof storage?.getItem === 'function' && typeof storage?.setItem === 'function' ? storage : null;
    } catch { return null; }
}
export function telegramCloudStorage(telegram, native) {
    if (!native) return null;
    try {
        if (telegram?.isVersionAtLeast && !telegram.isVersionAtLeast('6.9')) return null;
        const storage = telegram?.CloudStorage;
        return typeof storage?.getItem === 'function' && typeof storage?.getItems === 'function' && typeof storage?.setItem === 'function' && typeof storage?.removeItems === 'function' ? storage : null;
    } catch { return null; }
}

export function createPreferences({ deviceStorage = null, cloudStorage = null, timeout = 2000 } = {}) {
    const memory = new Map(), blocked = new Set(), dirty = new Set(), writable = new Set(), cloudWritable = new Set(), cloudManifests = new Map(), pendingCloud = new Set();
    let loading = Promise.resolve(), writing = Promise.resolve();
    function cloudPending(key) {
        if (pendingCloud.has(key)) return true;
        try { return localStorage.getItem(prefix + key + ':cloud-pending') === '1'; } catch { return false; }
    }
    function markCloudPending(key) {
        pendingCloud.add(key);
        try { localStorage.setItem(prefix + key + ':cloud-pending', '1'); } catch {}
    }
    function clearCloudPending(key, value) {
        if (JSON.stringify(get(key)) !== JSON.stringify(value)) return;
        pendingCloud.delete(key);
        try { localStorage.removeItem(prefix + key + ':cloud-pending'); } catch {}
    }
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
    function cloudRequest(method, ...args) {
        return new Promise(resolve => {
            const timer = setTimeout(() => resolve(null), Math.max(timeout, 5000));
            try { cloudStorage[method](...args, (error, value) => { clearTimeout(timer); resolve(error ? null : { value }); }); }
            catch { clearTimeout(timer); resolve(null); }
        });
    }
    async function readCloudHistory(key) {
        const name = cloudHistoryName(key), response = await cloudRequest('getItem', name);
        if (!response) return null;
        if (!response.value) return { value: undefined, manifest: null };
        let manifest;
        try { manifest = JSON.parse(response.value); } catch { return null; }
        if (manifest?.v !== 1 || !['a', 'b'].includes(manifest.slot) || !Number.isInteger(manifest.count) || manifest.count < 1 || manifest.count > historyLimit) return null;
        const names = Array.from({ length: manifest.count }, (_, index) => cloudHistoryPart(name, manifest.slot, index));
        const parts = await cloudRequest('getItems', names);
        if (!parts?.value || typeof parts.value !== 'object') return null;
        const value = cloudHistoryValue(key, names.map(part => parts.value[part]));
        return value === undefined ? null : { value, manifest };
    }
    async function writeCloudHistory(key, history) {
        const chunks = cloudHistoryChunks(history);
        if (!chunks) return false;
        const name = cloudHistoryName(key), previous = cloudManifests.get(key);
        const slot = previous?.slot === 'a' ? 'b' : 'a';
        const stored = await Promise.all(chunks.map((chunk, index) => cloudRequest('setItem', cloudHistoryPart(name, slot, index), chunk)));
        if (stored.some(result => result?.value !== true)) return false;
        const manifest = { v: 1, slot, count: chunks.length };
        if ((await cloudRequest('setItem', name, JSON.stringify(manifest)))?.value !== true) return false;
        cloudManifests.set(key, manifest);
        if (previous) {
            const old = Array.from({ length: previous.count }, (_, index) => cloudHistoryPart(name, previous.slot, index));
            await cloudRequest('removeItems', old);
        }
        return true;
    }
    async function readCloud(key) {
        if (key.startsWith('history:')) return readCloudHistory(key);
        const response = await cloudRequest('getItem', 'synctune_theme');
        if (!response) return null;
        if (!response.value) return { value: undefined };
        const value = clean(key, response.value);
        return value === undefined ? null : { value };
    }
    async function writeCloud(key, value) {
        return key === 'theme' ? (await cloudRequest('setItem', 'synctune_theme', value))?.value === true : writeCloudHistory(key, value);
    }
    return {
        get,
        load(keys) {
            loading = Promise.all(keys.filter(allowed).map(async key => {
                const pending = syncsToCloud(key) && cloudPending(key);
                const cloudRead = cloudStorage && syncsToCloud(key) ? readCloud(key) : null;
                if (deviceStorage) {
                    const saved = get(key), result = await request('getItem', key);
                    if (result) {
                        writable.add(key);
                        if (!dirty.has(key) && !pending) {
                            const restored = parse(key, result.value);
                            if (restored !== undefined) cache(key, restored);
                            else if (saved !== undefined) { cache(key, saved); await request('setItem', key, saved); }
                        }
                    }
                }
                if (!cloudRead) return;
                const remote = await cloudRead;
                if (!remote) return;
                cloudWritable.add(key);
                if (key.startsWith('history:')) cloudManifests.set(key, remote.manifest);
                if (dirty.has(key)) return;
                if (remote.value !== undefined && !pending) cache(key, remote.value);
                else {
                    const saved = get(key);
                    if (saved !== undefined && await writeCloud(key, saved)) clearCloudPending(key, saved);
                }
            }));
            return loading;
        },
        set(key, value) {
            if (!allowed(key) || (value = clean(key, value)) === undefined) return Promise.resolve(false);
            dirty.add(key); cache(key, value);
            if (cloudStorage && syncsToCloud(key)) markCloudPending(key);
            if (!deviceStorage && !(cloudStorage && syncsToCloud(key))) return Promise.resolve(true);
            writing = writing.then(() => loading).then(async () => {
                const results = [];
                if (deviceStorage && writable.has(key)) results.push((await request('setItem', key, value))?.value === true);
                if (cloudStorage && syncsToCloud(key)) {
                    if (!cloudWritable.has(key)) {
                        const remote = await readCloud(key);
                        if (remote) {
                            cloudWritable.add(key);
                            if (key.startsWith('history:')) cloudManifests.set(key, remote.manifest);
                        }
                    }
                    if (cloudWritable.has(key)) {
                        const saved = await writeCloud(key, value);
                        if (saved) clearCloudPending(key, value);
                        results.push(saved);
                    }
                }
                return results.length > 0 && results.every(Boolean);
            });
            return writing;
        },
        flush() { return Promise.all([loading, writing]); }
    };
}
