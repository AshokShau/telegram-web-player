// Capture launch data before removing it from the address bar. The server still
// verifies the original signed initData for every join and authenticated request.
const storageKey = 'synctune:room-session';

export function readLaunch(telegram) {
    const params = new URLSearchParams(location.search);
    const fragment = new URLSearchParams(location.hash.slice(1));
    let saved = null;
    try { saved = JSON.parse(sessionStorage.getItem(storageKey)); } catch { /* Storage is optional. */ }
    let transferred = null;
    try { transferred = JSON.parse(fragment.get('browser_session')); } catch { /* Ignore malformed handoffs. */ }
    const nativeData = telegram?.initData?.trim() ? telegram.initData : '';
    const source = transferred || (nativeData ? { initData: nativeData } : saved);
    const initData = typeof source?.initData === 'string' ? source.initData : '';
    const signed = new URLSearchParams(initData);
    let user = {};
    try { user = JSON.parse(signed.get('user')) || {}; } catch { /* Display data never grants access. */ }
    const isTelegramWebApp = Boolean(!transferred && nativeData && telegram.platform !== 'unknown');
    if (isTelegramWebApp) user = { ...telegram.initDataUnsafe?.user };
    const roomId = String(signed.get('start_param') || (nativeData && !transferred ? telegram.initDataUnsafe?.start_param : '') || params.get('tgWebAppStartParam') || params.get('startapp') || params.get('chat_id') || params.get('room') || source?.roomId || user.id || '0');
    const launch = { initData, roomId, user, isTelegramWebApp, isBrowserHandoff: Boolean(transferred) };
    try {
        if (initData) sessionStorage.setItem(storageKey, JSON.stringify({ initData, roomId }));
    } catch { /* The current session still works without storage. */ }
    if (location.search || location.hash) history.replaceState(history.state, '', location.pathname);
    return launch;
}

export function browserSessionURL(launch) {
    const url = new URL('/room', location.origin);
    // A fragment stays out of HTTP requests and is removed by readLaunch on arrival.
    url.hash = new URLSearchParams({ browser_session: JSON.stringify({ initData: launch.initData, roomId: launch.roomId }) }).toString();
    return url.href;
}
