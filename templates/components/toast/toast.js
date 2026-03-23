import { LitElement, html } from "/assets/js/lit-all.min.js";

/**
 * <omk-toast-container> — Toast notification container.
 *
 * Unified pattern for both client-side and server-side toasts:
 *   - Client: element.add("success", "Saved!", 4000)
 *   - Server: HX-Trigger header → {"toast": {"variant": "success", "message": "Saved!"}}
 *
 * Features:
 *   - Auto-dismiss with configurable duration
 *   - Click to dismiss
 *   - Stacking (newest at bottom)
 *   - Slide-up + fade animations via CSS
 *   - HTMX compatible via htmx:trigger event
 */
export class OmkToastContainer extends LitElement {
  static properties = {
    _items: { state: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this._items = [];
    this._counter = 0;
    this._onHtmxTrigger = this._handleHtmxTrigger.bind(this);
  }

  connectedCallback() {
    super.connectedCallback();
    // Listen for HTMX-triggered toasts via HX-Trigger response header
    document.body.addEventListener("toast", this._onHtmxTrigger);
  }

  /**
   * Add a toast notification.
   * @param {string} variant — "default", "success", "destructive", "warning"
   * @param {string} message — Text to display
   * @param {number} duration — Auto-dismiss ms (default 4000, 0 = manual)
   */
  add(variant, message, duration = 4000) {
    const id = ++this._counter;
    const item = { id, variant: variant || "default", message, show: true };
    this._items = [...this._items, item];

    if (duration > 0) {
      setTimeout(() => this.dismiss(id), duration);
    }

    this.requestUpdate();
  }

  dismiss(id) {
    this._items = this._items.map((t) =>
      t.id === id ? { ...t, show: false } : t
    );
    this.requestUpdate();

    // Remove from DOM after transition
    setTimeout(() => {
      this._items = this._items.filter((t) => t.id !== id);
      this.requestUpdate();
    }, 200);
  }

  _handleHtmxTrigger(e) {
    const data = e.detail || {};
    this.add(data.variant, data.message, data.duration);
  }

  _variantClasses(variant) {
    switch (variant) {
      case "success":
        return "bg-success/10 border-success/20 text-success";
      case "destructive":
        return "bg-destructive/10 border-destructive/20 text-destructive";
      case "warning":
        return "bg-warning/10 border-warning/20 text-warning";
      default:
        return "bg-card border-border text-foreground";
    }
  }

  render() {
    return html`${this._items.map(
      (t) => html`
        <div
          class="px-4 py-3 rounded-lg text-sm font-medium border shadow-lg cursor-pointer
            ${this._variantClasses(t.variant)}
            ${t.show ? "omk-toast-enter" : "omk-toast-leave"}"
          @click=${() => this.dismiss(t.id)}
        >
          ${t.message}
        </div>
      `
    )}`;
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.body.removeEventListener("toast", this._onHtmxTrigger);
  }
}

customElements.define("omk-toast-container", OmkToastContainer);
