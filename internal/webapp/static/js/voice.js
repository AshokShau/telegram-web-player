// Audio-only WebRTC with serialized signaling and explicit media cleanup.
import { state, on, emit, notify, haptic, send, platform, preferences } from './core.js?v=24';
export function createVoice(container) {
    let pc = null;
    let stream = null;
    let microphoneSender = null;
    let context = null;
    let speakingTimer = null;
    let joinTimer = null;
    let disconnectTimer = null;
    let incomingCandidates = [];
    let outgoingCandidates = [];
    let offerSent = false;
    let signaling = Promise.resolve();
    let generation = 0;
    let wantsToJoin = false;
    let wasSpeaking = false;
    let microphonePending = false;
    function status(value) { state.voice.status = value; emit('voice-local'); }
    function self() { return state.voice.room.participants.find(p => p.userId === state.permissions.userId); }
    function allowedToSpeak() { return self()?.allowedToSpeak ?? (state.permissions.isAdmin || !state.voice.room.muteNewParticipants); }
    function applyMicrophone() {
        const allowed = state.voice.joined && !state.voice.muted && allowedToSpeak();
        stream?.getAudioTracks().forEach(track => { track.enabled = allowed; });
        if (!allowed && wasSpeaking) { wasSpeaking = false; send('vc_speaking', { speaking: false }); }
        emit('voice-local');
    }
    function analyze() {
        if (!stream || context) return;
        const AudioContext = window.AudioContext || window.webkitAudioContext;
        if (!AudioContext) return;
        try {
            context = new AudioContext();
            const source = context.createMediaStreamSource(stream);
            const analyzer = context.createAnalyser(); analyzer.fftSize = 512; source.connect(analyzer);
            const samples = new Uint8Array(analyzer.fftSize);
            speakingTimer = setInterval(() => {
                let speaking = false;
                if (!state.voice.muted && allowedToSpeak() && state.voice.joined) {
                    analyzer.getByteTimeDomainData(samples);
                    let energy = 0;
                    for (const sample of samples) energy += ((sample - 128) / 128) ** 2;
                    speaking = Math.sqrt(energy / samples.length) > .035;
                }
                if (speaking !== wasSpeaking) { wasSpeaking = speaking; send('vc_speaking', { speaking }); }
            }, 180);
        } catch { /* Speaking visualization is optional; media transmission is independent. */ }
    }
    async function microphone() {
        if (stream) return stream;
        if (!platform.supportsMicrophone) throw new Error('Microphone access is unavailable. You can listen to voice chat.');
        const currentGeneration = generation;
        const result = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: state.voice.noiseSuppression, autoGainControl: true }, video: false });
        if (currentGeneration !== generation || !pc) { result.getTracks().forEach(track => track.stop()); throw new Error('Voice chat was closed.'); }
        stream = result; stream.getAudioTracks().forEach(track => { track.enabled = false; });
        analyze(); return stream;
    }
    function cleanup() {
        generation++;
        clearInterval(speakingTimer); clearTimeout(joinTimer); clearTimeout(disconnectTimer);
        context?.close().catch(() => {}); context = null;
        stream?.getTracks().forEach(track => track.stop()); stream = null;
        if (pc) { pc.onconnectionstatechange = null; pc.onicecandidate = null; pc.ontrack = null; pc.close(); pc = null; }
        for (const audio of container.querySelectorAll('audio')) { audio.pause(); audio.srcObject = null; }
        container.replaceChildren(); microphoneSender = null; wasSpeaking = false;
        incomingCandidates = []; outgoingCandidates = []; offerSent = false;
        state.voice.joined = false; state.voice.muted = true;
    }
    async function join(listenOnly = false) {
        if (state.voice.joined || ['joining', 'connecting'].includes(state.voice.status)) return;
        if (!platform.supportsVoice) { status('unsupported'); notify('Voice chat is unavailable on this device.', 'error'); return; }
        if (state.connection !== 'connected' || !state.permissions.userId) { notify('Connect to the Telegram room before joining voice chat.', 'warning'); return; }
        wantsToJoin = true; cleanup(); const currentGeneration = generation; status('joining');
        try {
            pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });
            const peer = pc;
            // A stable sendrecv transceiver allows listen-only users to enable their mic later with replaceTrack.
            microphoneSender = peer.addTransceiver('audio', { direction: 'sendrecv' }).sender;
            peer.onicecandidate = event => {
                if (!event.candidate || peer !== pc) return;
                const candidate = JSON.stringify(event.candidate.toJSON());
                if (offerSent) send('vc_candidate', { candidate }); else outgoingCandidates.push(candidate);
            };
            peer.ontrack = event => {
                if (peer !== pc || event.track.kind !== 'audio') return;
                const audio = document.createElement('audio');
                audio.autoplay = true; audio.setAttribute('playsinline', '');
                audio.srcObject = new MediaStream([event.track]); audio.muted = state.voice.outputMuted;
                container.append(audio);
                audio.play().catch(() => notify('Tap the voice audio button to enable sound.'));
                event.track.onended = () => { audio.pause(); audio.srcObject = null; audio.remove(); };
            };
            peer.onconnectionstatechange = () => {
                if (peer !== pc) return;
                clearTimeout(disconnectTimer);
                if (peer.connectionState === 'connected') { clearTimeout(joinTimer); status('connected'); haptic('success'); }
                else if (peer.connectionState === 'failed') { cleanup(); status('error'); notify('Voice connection failed. Try joining again.', 'error'); }
                else if (peer.connectionState === 'disconnected') {
                    status('reconnecting');
                    disconnectTimer = setTimeout(() => { if (peer === pc) { leave(); status('error'); notify('Voice connection was lost. Join again to reconnect.', 'error'); } }, 10000);
                }
            };
            if (!listenOnly && allowedToSpeak()) {
                try { const local = await microphone(); await microphoneSender.replaceTrack(local.getAudioTracks()[0]); }
                catch (error) { if (currentGeneration !== generation) return; notify(`${error.message || 'Microphone unavailable.'} Joined as a listener.`); }
            }
            if (currentGeneration !== generation || peer !== pc) return;
            if (!send('vc_join')) { cleanup(); status('error'); return; }
            const offer = await peer.createOffer(); await peer.setLocalDescription(offer);
            if (peer !== pc) return;
            send('vc_offer', { sdp: peer.localDescription.sdp }); offerSent = true;
            for (const candidate of outgoingCandidates) send('vc_candidate', { candidate }); outgoingCandidates = [];
            state.voice.joined = true; state.voice.muted = true; status('connecting');
            send('vc_mute_self', { muted: true });
            joinTimer = setTimeout(() => { if (peer === pc && peer.connectionState !== 'connected') { leave(); status('error'); notify('Voice chat could not connect. Check your network and try again.', 'error'); } }, 20000);
        } catch (error) { if (currentGeneration !== generation) return; cleanup(); status('error'); notify(error.message || 'Could not join voice chat. Try again.', 'error'); }
    }
    function leave() { wantsToJoin = false; if (state.voice.joined) send('vc_leave'); cleanup(); status('idle'); }
    async function toggleMic() {
        if (!state.voice.joined || microphonePending) return;
        if (!allowedToSpeak()) { notify('Speaking is restricted by the room admin.', 'warning'); return; }
        if (state.voice.muted && !stream) {
            microphonePending = true;
            try { const local = await microphone(); await microphoneSender.replaceTrack(local.getAudioTracks()[0]); }
            catch (error) { notify(error.message || 'Allow microphone access to speak.', 'error'); return; }
            finally { microphonePending = false; }
        }
        if (!state.voice.joined) return;
        state.voice.muted = !state.voice.muted;
        if (!state.voice.muted) await context?.resume().catch(() => {});
        send('vc_mute_self', { muted: state.voice.muted }); applyMicrophone();
    }
    async function toggleOutput() {
        state.voice.outputMuted = !state.voice.outputMuted;
        for (const audio of container.querySelectorAll('audio')) { audio.muted = state.voice.outputMuted; if (!audio.muted) await audio.play().catch(() => notify('Voice audio is blocked by this device. Try again.')); }
        emit('voice-local');
    }
    function signal(task) {
        const currentGeneration = generation;
        signaling = signaling.then(async () => { if (currentGeneration === generation && pc) await task(pc); }).catch(error => { status('error'); notify(`Voice signaling failed: ${error.message}. Leave and join again.`, 'error'); });
    }
    async function flushCandidates(peer) { for (const candidate of incomingCandidates) await peer.addIceCandidate(candidate); incomingCandidates = []; }
    on('vc_answer', data => signal(async peer => { await peer.setRemoteDescription({ type: 'answer', sdp: data.sdp }); await flushCandidates(peer); }));
    on('vc_offer', data => signal(async peer => {
        await peer.setRemoteDescription({ type: 'offer', sdp: data.sdp }); await flushCandidates(peer);
        await peer.setLocalDescription(await peer.createAnswer()); send('vc_answer', { sdp: peer.localDescription.sdp });
    }));
    on('vc_candidate', data => signal(async peer => {
        const candidate = JSON.parse(data.candidate);
        if (peer.remoteDescription) await peer.addIceCandidate(candidate); else incomingCandidates.push(candidate);
    }));
    on('voice-state', () => {
        if (!allowedToSpeak() && !state.voice.muted) { state.voice.muted = true; if (state.voice.joined) send('vc_mute_self', { muted: true }); }
        applyMicrophone();
    });
    on('transport-lost', () => { cleanup(); status(wantsToJoin ? 'reconnecting' : 'idle'); });
    on('permissions', () => { if (state.permissions.userId && wantsToJoin && !state.voice.joined && state.voice.status === 'reconnecting') { state.voice.status = 'idle'; join(true); } });
    on('sleep-expired', leave);
    on('dispose', () => { wantsToJoin = false; cleanup(); });
    on('session-ended', () => { wantsToJoin = false; cleanup(); status('idle'); });
    on('request-error', data => { if (data.type?.startsWith('vc_') && ['joining', 'connecting'].includes(state.voice.status)) { cleanup(); status('error'); } });
    async function setNoiseSuppression(enabled) {
        const previous = state.voice.noiseSuppression;
        try {
            if (!navigator.mediaDevices?.getSupportedConstraints?.().noiseSuppression) throw new Error('Noise suppression is unavailable on this device.');
            for (const track of stream?.getAudioTracks() || []) await track.applyConstraints({ noiseSuppression: enabled });
            state.voice.noiseSuppression = enabled; preferences.set('noise-suppression', enabled);
            notify(`Noise suppression ${enabled ? 'enabled' : 'disabled'}.`, 'success');
        } catch (error) { state.voice.noiseSuppression = previous; notify(error.message || 'The microphone setting could not be changed.', 'error'); }
        emit('voice-local');
    }
    return { join, leave, toggleMic, toggleOutput, setNoiseSuppression };
}
