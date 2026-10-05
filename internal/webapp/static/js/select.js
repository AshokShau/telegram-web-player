// Styled choices share the existing select values and change events.
export function createSelects(selects) {
    const controls = new Map();
    const triggers = new WeakMap();
    const menu = document.createElement('div'); menu.className = 'choice-menu'; menu.id = 'choice-menu'; menu.setAttribute('role', 'listbox'); menu.hidden = true;
    document.body.append(menu);
    let active = null, anchor = null, highlighted = 0, typeahead = '', typeaheadTimer;
    function close(restore = false) {
        if (!active) return;
        const button = anchor;
        button.setAttribute('aria-expanded', 'false'); button.removeAttribute('aria-activedescendant');
        active = null; anchor = null; menu.hidden = true; document.body.append(menu); typeahead = ''; clearTimeout(typeaheadTimer); if (restore) button.focus({ preventScroll: true });
    }
    function refresh() {
        for (const [select, { button, value, label }] of controls) {
            value.textContent = select.selectedOptions[0]?.textContent || 'Choose';
            button.hidden = select.hidden;
            button.disabled = select.disabled; button.setAttribute('aria-label', `${label}: ${value.textContent}`);
        }
        if (active?.disabled) close();
    }
    function highlight(index) {
        const options = [...menu.children]; highlighted = (index + options.length) % options.length;
        for (const [i, option] of options.entries()) option.classList.toggle('is-highlighted', i === highlighted);
        anchor.setAttribute('aria-activedescendant', options[highlighted].id);
        options[highlighted].scrollIntoView({ block: 'nearest' });
    }
    function position() {
        if (!active) return;
        const rect = anchor.getBoundingClientRect();
        const shell = document.getElementById('app');
        const bounds = shell.getBoundingClientRect();
        const style = getComputedStyle(shell);
        const left = Math.max(16, bounds.left + parseFloat(style.paddingLeft));
        const right = Math.min(innerWidth - 16, bounds.right - parseFloat(style.paddingRight));
        const safeTop = Math.max(16, bounds.top);
        const bottom = Math.min(innerHeight - 16, bounds.bottom);
        const width = Math.min(Math.max(rect.width, 200), right - left);
        menu.style.width = `${width}px`;
        const below = bottom - rect.bottom - 8;
        const above = rect.top - safeTop - 8;
        const desired = Math.min(288, menu.children.length * 44 + 12);
        const placeAbove = below < desired && above > below;
        const height = Math.min(desired, Math.max(44, placeAbove ? above : below));
        menu.style.maxHeight = `${height}px`;
        menu.style.left = `${Math.max(left, Math.min(rect.right - width, right - width))}px`;
        menu.style.top = `${Math.max(safeTop, Math.min(placeAbove ? rect.top - height - 8 : rect.bottom + 8, bottom - height))}px`;
    }
    function choose(value) {
        const select = active; if (!select) return;
        select.value = value; select.dispatchEvent(new Event('change', { bubbles: true })); refresh(); close(true);
    }
    function open(select, button = controls.get(select).button) {
        if (select.disabled || !select.options.length) return;
        if (active === select && anchor === button) { close(true); return; }
        close(); active = select; anchor = button;
        // A modal's controls and popup must share the browser's top layer.
        (button.closest('dialog') || document.body).append(menu);
        const { label } = controls.get(select);
        button.setAttribute('aria-expanded', 'true'); menu.setAttribute('aria-label', label);
        menu.replaceChildren(...[...select.options].map((option, index) => {
            const node = document.createElement('button'); node.type = 'button'; node.className = 'choice-option'; node.id = `choice-option-${index}`;
            node.setAttribute('role', 'option'); node.setAttribute('aria-selected', String(option.selected)); node.tabIndex = -1;
            node.disabled = option.disabled; node.textContent = option.textContent;
            node.addEventListener('click', () => choose(option.value)); return node;
        }));
        menu.hidden = false; position(); highlight(Math.max(0, select.selectedIndex)); button.focus({ preventScroll: true });
    }
    function connect(select, button) {
        triggers.set(button, select);
        button.setAttribute('role', 'combobox'); button.setAttribute('aria-haspopup', 'listbox'); button.setAttribute('aria-controls', menu.id); button.setAttribute('aria-expanded', 'false');
    }
    for (const select of selects) {
        const label = document.querySelector(`label[for="${select.id}"]`)?.textContent.trim() || select.getAttribute('aria-label') || 'Choose';
        const button = document.createElement('button'); button.type = 'button'; button.className = 'choice-trigger'; button.dataset.selectId = select.id;
        connect(select, button);
        const value = document.createElement('span');
        const icon = document.createElementNS('http://www.w3.org/2000/svg', 'svg'); icon.setAttribute('class', 'icon'); icon.setAttribute('aria-hidden', 'true');
        const use = document.createElementNS(icon.namespaceURI, 'use'); use.setAttribute('href', '/static/assets/icons.svg?v=16#down'); icon.append(use); button.append(value, icon);
        select.classList.add('select-native'); select.tabIndex = -1; select.setAttribute('aria-hidden', 'true'); select.after(button);
        controls.set(select, { button, value, label });
        button.addEventListener('click', () => open(select));
        select.addEventListener('change', refresh);
        document.querySelector(`label[for="${select.id}"]`)?.addEventListener('click', event => { event.preventDefault(); button.focus(); });
    }
    document.addEventListener('pointerdown', event => { if (active && !menu.contains(event.target) && !anchor.contains(event.target)) close(); });
    document.addEventListener('keydown', event => {
        if (active && event.key === 'Escape') { event.preventDefault(); event.stopImmediatePropagation(); close(true); }
        else if (active && event.key === 'Tab') close();
        else {
            const select = triggers.get(event.target); if (!select || select.disabled) return;
            if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
                event.preventDefault(); if (active !== select || anchor !== event.target) open(select, event.target);
                else highlight(event.key === 'Home' ? 0 : event.key === 'End' ? menu.children.length - 1 : highlighted + (event.key === 'ArrowUp' ? -1 : 1));
            } else if (active === select && anchor === event.target && ['Enter', ' '].includes(event.key)) { event.preventDefault(); choose(select.options[highlighted].value); }
            else if (active === select && anchor === event.target && event.key.length === 1 && !event.ctrlKey && !event.metaKey) {
                clearTimeout(typeaheadTimer); typeahead += event.key.toLowerCase();
                const index = [...select.options].findIndex(o => o.textContent.toLowerCase().startsWith(typeahead)); if (index >= 0) highlight(index);
                typeaheadTimer = setTimeout(() => { typeahead = ''; }, 600);
            }
        }
    });
    document.addEventListener('scroll', event => { if (anchor && event.target instanceof Element && event.target.contains(anchor)) close(); }, { capture: true, passive: true });
    window.addEventListener('resize', () => close());
    refresh();
    const find = id => [...controls.keys()].find(node => node.id === id);
    return { refresh, close, connect(id, button) { const select = find(id); if (select) connect(select, button); }, open(id, button) { const select = find(id); if (select) open(select, button); } };
}
