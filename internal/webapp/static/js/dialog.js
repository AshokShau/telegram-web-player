import { element } from './ui.js?v=88';
import { state, haptic } from './core.js?v=88';

export function createActionDialog(dom, choices, onChange) {
    const surface = dom['action-dialog'];
    let pending = null;
    function finish(value) {
        const request = pending;
        pending = null;
        choices.close();
        request?.resolve(value);
        if (request?.focus?.isConnected) request.focus.focus({ preventScroll: true });
        onChange();
    }
    dom['dialog-form'].addEventListener('submit', event => {
        event.preventDefault();
        const value = !dom['dialog-input'].hidden ? dom['dialog-input'].value.trim()
            : !dom['dialog-select'].hidden ? dom['dialog-select'].value : true;
        if (!dom['dialog-input'].hidden && (!value || Array.from(value).length > state.limits.maxPlaylistNameLength)) { dom['dialog-input'].setCustomValidity(`Use 1–${state.limits.maxPlaylistNameLength} characters.`); dom['dialog-input'].reportValidity(); return; }
        if (!value) return;
        // Settle synchronously; the queued native close event must not cancel it.
        surface.close();
        finish(value);
    });
    dom['dialog-input'].addEventListener('input', () => dom['dialog-input'].setCustomValidity(''));
    surface.addEventListener('close', () => { if (!surface.open) finish(null); });
    return function dialog({ title, description, input, choices: options, action = 'Confirm', danger = false }) {
        if (surface.open) { surface.close(); finish(null); }
        if (danger) haptic('warning');
        const focus = document.activeElement;
        dom['dialog-title'].textContent = title;
        dom['dialog-description'].textContent = description;
        dom['dialog-submit'].textContent = action;
        dom['dialog-submit'].classList.toggle('danger', danger);
        dom['dialog-input'].hidden = dom['dialog-input-label'].hidden = input === undefined;
        dom['dialog-input'].value = input || '';
        dom['dialog-input'].setCustomValidity('');
        dom['dialog-input'].required = input !== undefined;
        dom['dialog-select'].hidden = dom['dialog-select-label'].hidden = !options;
        dom['dialog-select'].replaceChildren(...(options || []).map(choice => element('option', { value: choice.id, text: choice.name })));
        choices.close(); choices.refresh();
        const promise = new Promise(resolve => { pending = { resolve, focus }; });
        surface.showModal(); onChange();
        if (input !== undefined) { dom['dialog-input'].focus(); dom['dialog-input'].select(); }
        else if (options) document.querySelector('[data-select-id="dialog-select"]').focus();
        return promise;
    };
}
