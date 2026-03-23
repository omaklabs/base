import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-dropdown> — Accessible dropdown menu with keyboard navigation.
 *
 * Features:
 *   - Arrow key navigation (Up/Down move between items)
 *   - Home/End jump to first/last item
 *   - Enter/Space activate focused item
 *   - Click-outside dismiss
 *   - ESC closes and returns focus to trigger
 *   - ARIA: aria-expanded on trigger, role="menu" on content, role="menuitem" on items
 *   - HTMX compatible (connectedCallback auto-initializes)
 *
 * Attributes:
 *   open — Boolean, reflected. Controls visibility.
 *
 * Data attributes on children:
 *   [data-trigger]   — Element that toggles the menu
 *   [data-content]   — The menu panel
 *   [data-item]      — Menu items (keyboard-navigable)
 */
export class OmkDropdown extends LitElement {
  static properties = {
    open: { type: Boolean, reflect: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this.open = false;
    this._onDocClick = this._handleDocClick.bind(this);
    this._onKeydown = this._handleKeydown.bind(this);
  }

  connectedCallback() {
    super.connectedCallback();

    this.querySelector("[data-trigger]")?.addEventListener("click", (e) => {
      e.stopPropagation();
      this.toggle();
    });
  }

  toggle() {
    if (this.open) {
      this.hide();
    } else {
      this.show();
    }
  }

  show() {
    this.open = true;

    // Set ARIA on trigger
    const trigger = this.querySelector("[data-trigger]");
    const btn = trigger?.querySelector("button, a, [role='button']") || trigger;
    btn?.setAttribute("aria-expanded", "true");

    document.addEventListener("click", this._onDocClick);
    document.addEventListener("keydown", this._onKeydown);

    // Focus first item
    requestAnimationFrame(() => {
      const items = this._getItems();
      if (items.length) items[0].focus();
    });
  }

  hide() {
    this.open = false;

    const trigger = this.querySelector("[data-trigger]");
    const btn = trigger?.querySelector("button, a, [role='button']") || trigger;
    btn?.setAttribute("aria-expanded", "false");

    document.removeEventListener("click", this._onDocClick);
    document.removeEventListener("keydown", this._onKeydown);

    // Return focus to trigger
    const focusTarget =
      trigger?.querySelector("button, a, [role='button']") || trigger;
    focusTarget?.focus();
  }

  _getItems() {
    return [...this.querySelectorAll("[data-item]")];
  }

  _handleDocClick(e) {
    if (!this.contains(e.target)) {
      this.hide();
    }
  }

  _handleKeydown(e) {
    const items = this._getItems();
    const current = items.indexOf(document.activeElement);

    switch (e.key) {
      case "Escape":
        e.preventDefault();
        this.hide();
        break;
      case "ArrowDown":
        e.preventDefault();
        if (current < items.length - 1) items[current + 1].focus();
        else items[0].focus();
        break;
      case "ArrowUp":
        e.preventDefault();
        if (current > 0) items[current - 1].focus();
        else items[items.length - 1].focus();
        break;
      case "Home":
        e.preventDefault();
        if (items.length) items[0].focus();
        break;
      case "End":
        e.preventDefault();
        if (items.length) items[items.length - 1].focus();
        break;
      case "Tab":
        this.hide();
        break;
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("click", this._onDocClick);
    document.removeEventListener("keydown", this._onKeydown);
  }
}

customElements.define("omk-dropdown", OmkDropdown);
