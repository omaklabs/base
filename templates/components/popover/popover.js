import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-popover> — Accessible popover positioned relative to trigger.
 *
 * Features:
 *   - Click to toggle (or manual via .show()/.hide())
 *   - Click-outside dismiss
 *   - ESC dismiss
 *   - Focus management
 *   - Positioned via CSS (relative parent + absolute content)
 *
 * Attributes:
 *   open — Boolean, reflected. Controls visibility.
 *
 * Data attributes:
 *   [data-popover-trigger] — Element that toggles the popover
 *   [data-popover-content] — The popover panel
 */
export class OmkPopover extends LitElement {
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
    this.querySelector("[data-popover-trigger]")?.addEventListener("click", (e) => {
      e.stopPropagation();
      this.toggle();
    });
  }

  toggle() {
    if (this.open) this.hide();
    else this.show();
  }

  show() {
    this.open = true;
    document.addEventListener("click", this._onDocClick);
    document.addEventListener("keydown", this._onKeydown);

    requestAnimationFrame(() => {
      const content = this.querySelector("[data-popover-content]");
      const focusable = content?.querySelector(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      focusable?.focus();
    });
  }

  hide() {
    this.open = false;
    document.removeEventListener("click", this._onDocClick);
    document.removeEventListener("keydown", this._onKeydown);
  }

  _handleDocClick(e) {
    if (!this.contains(e.target)) {
      this.hide();
    }
  }

  _handleKeydown(e) {
    if (e.key === "Escape") {
      e.preventDefault();
      this.hide();
      this.querySelector("[data-popover-trigger]")?.querySelector("button")?.focus();
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("click", this._onDocClick);
    document.removeEventListener("keydown", this._onKeydown);
  }
}

customElements.define("omk-popover", OmkPopover);
