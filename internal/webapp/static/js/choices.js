// Choice screens preserve native select values and the existing change handlers.
export function createChoiceScreens(selects, { nativeBack = false, onOpen, onClose } = {}) {
    const controls = new Map(), triggers = new WeakMap();
    const screen = document.createElement('dialog'); screen.id = 'choice-screen'; screen.className = 'choice-screen'; screen.setAttribute('aria-labelledby', 'choice-title');
    const header = document.createElement('header'); header.className = 'choice-header';
    const back = document.createElement('button'); back.type = 'button'; back.className = 'icon-button browser-back'; back.setAttribute('aria-label', 'Back'); back.hidden = nativeBack;
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg'); svg.setAttribute('class', 'icon back-icon'); svg.setAttribute('aria-hidden', 'true');
    const use = document.createElementNS(svg.namespaceURI, 'use'); use.setAttribute('href', '/static/assets/icons.svg?v=82#arrow'); svg.append(use); back.append(svg);
    const title = document.createElement('h2'); title.id = 'choice-title'; header.append(back, title);
    const content = document.createElement('div'); content.className = 'choice-content';
    const group = document.createElement('section'); group.className = 'settings-group telegram-group';
    const question = document.createElement('h3'); question.id = 'choice-question'; question.className = 'telegram-group-title';
    const menu = document.createElement('div'); menu.id = 'choice-menu'; menu.className = 'choice-menu'; menu.setAttribute('role', 'listbox'); menu.setAttribute('aria-labelledby', question.id); menu.tabIndex = 0; menu.hidden = true;
    const description = document.createElement('p'); description.className = 'settings-note telegram-note';
    group.append(question, menu); content.append(group, description); screen.append(header, content); document.body.append(screen);
    let active = null, anchor = null, highlighted = 0, signature = '', typeahead = '', typeaheadTimer;
    function close(restore = false) {
        if (!active) return false;
        const button = anchor;
        active = null; anchor = null; menu.hidden = true; menu.removeAttribute('aria-activedescendant');
        button.setAttribute('aria-expanded', 'false'); screen.close(); typeahead = ''; clearTimeout(typeaheadTimer);
        onClose?.(); if (restore && button.isConnected && !button.disabled) button.focus({ preventScroll: true });
        return true;
    }
    function highlight(index, direction = 1) {
        const options = [...menu.children];
        if (!options.some(option => !option.disabled)) return;
        highlighted = (index + options.length) % options.length;
        while (options[highlighted].disabled) highlighted = (highlighted + direction + options.length) % options.length;
        for (const [i, option] of options.entries()) option.classList.toggle('is-highlighted', i === highlighted);
        menu.setAttribute('aria-activedescendant', options[highlighted].id);
        options[highlighted].scrollIntoView({ block: 'nearest' });
    }
    function renderOptions() {
        const next = JSON.stringify([...active.options].map(option => [option.value, option.textContent]));
        if (signature !== next) {
            signature = next;
            menu.replaceChildren(...[...active.options].map((option, index) => {
                const node = document.createElement('button'); node.type = 'button'; node.id = `choice-option-${index}`; node.className = 'choice-option'; node.setAttribute('role', 'option'); node.tabIndex = -1;
                node.addEventListener('click', () => choose(option.value)); return node;
            }));
        }
        for (const [index, node] of [...menu.children].entries()) {
            const option = active.options[index]; node.textContent = option.textContent; node.disabled = option.disabled; node.setAttribute('aria-selected', String(option.selected));
        }
    }
    function refresh() {
        for (const [select, { button, value, label }] of controls) {
            const selected = select.selectedOptions[0];
            value.textContent = selected?.dataset.display || selected?.textContent || 'Choose';
            button.hidden = select.hidden; button.disabled = select.disabled; button.setAttribute('aria-label', `${label}: ${selected?.textContent || 'Choose'}`);
        }
        if (!active) return;
        if (active.disabled || ![...active.options].some(option => !option.disabled)) { close(true); return; }
        renderOptions();
        if (highlighted >= active.options.length || active.options[highlighted].disabled) highlight(Math.max(0, active.selectedIndex));
    }
    function choose(value) {
        const select = active;
        if (!select || select.disabled || ![...select.options].some(option => option.value === value && !option.disabled)) return;
        const changed = select.value !== value;
        select.value = value; close(true); if (changed) select.dispatchEvent(new Event('change', { bubbles: true })); refresh();
    }
    function open(select, button = controls.get(select).button) {
        if (select.disabled || ![...select.options].some(option => !option.disabled)) return;
        if (active === select) return;
        close(); active = select; anchor = button;
        title.textContent = controls.get(select).label; question.textContent = select.dataset.choiceQuestion || title.textContent;
        description.textContent = select.dataset.choiceDescription || ''; description.hidden = !description.textContent;
        button.setAttribute('aria-expanded', 'true'); signature = ''; renderOptions();
        menu.hidden = false; screen.showModal(); onOpen?.();
        menu.focus({ preventScroll: true }); highlight(Math.max(0, select.selectedIndex));
    }
    function connect(select, button) {
        triggers.set(button, select); button.setAttribute('aria-haspopup', 'dialog'); button.setAttribute('aria-controls', screen.id); button.setAttribute('aria-expanded', 'false');
    }
    for (const select of selects) {
        const labelNode = document.querySelector(`label[for="${select.id}"]`);
        const label = labelNode?.textContent.trim() || select.getAttribute('aria-label') || 'Choose';
        const button = document.createElement('button'); button.type = 'button'; button.className = 'choice-trigger'; button.dataset.selectId = select.id;
        connect(select, button);
        const value = document.createElement('span'); button.append(value);
        select.classList.add('select-native'); select.tabIndex = -1; select.setAttribute('aria-hidden', 'true'); select.after(button);
        controls.set(select, { button, value, label });
        button.addEventListener('click', () => open(select)); select.addEventListener('change', refresh);
        labelNode?.addEventListener('click', event => { event.preventDefault(); open(select); });
        select.closest('.setting-row')?.addEventListener('click', event => {
            if (!event.defaultPrevented && !button.contains(event.target)) open(select);
        });
    }
    back.addEventListener('click', () => close(true));
    screen.addEventListener('cancel', event => { event.preventDefault(); close(true); });
    screen.addEventListener('close', () => { if (!screen.open) close(true); });
    document.addEventListener('keydown', event => {
        if (active && event.key === 'Escape') { event.preventDefault(); event.stopImmediatePropagation(); close(true); return; }
        if (event.target === menu && active) {
            if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
                event.preventDefault(); highlight(event.key === 'Home' ? 0 : event.key === 'End' ? menu.children.length - 1 : highlighted + (event.key === 'ArrowUp' ? -1 : 1), event.key === 'ArrowUp' ? -1 : 1);
            } else if (['Enter', ' '].includes(event.key)) { event.preventDefault(); choose(active.options[highlighted].value); }
            else if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
                clearTimeout(typeaheadTimer); typeahead += event.key.toLowerCase();
                const index = [...active.options].findIndex(option => !option.disabled && option.textContent.toLowerCase().startsWith(typeahead));
                if (index >= 0) highlight(index); typeaheadTimer = setTimeout(() => { typeahead = ''; }, 600);
            }
        } else if (!active && ['ArrowDown', 'ArrowUp'].includes(event.key)) {
            const select = triggers.get(event.target); if (select && !select.disabled) { event.preventDefault(); open(select, event.target); }
        }
    });
    refresh();
    const find = id => [...controls.keys()].find(node => node.id === id);
    return { refresh, close, isOpen: () => Boolean(active), connect(id, button) { const select = find(id); if (select) connect(select, button); }, open(id, button) { const select = find(id); if (select) open(select, button); } };
}
