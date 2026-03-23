import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-dialog> — Accessible modal dialog with focus trapping.
 *
 * Features:
 *   - Focus trapping (Tab cycles within the panel)
 *   - Return focus to trigger on close
 *   - ESC key closes (scoped to this dialog)
 *   - Body scroll lock while open
 *   - HTMX compatible (connectedCallback auto-initializes)
 *
 * Attributes:
 *   open — Boolean, reflected. Controls visibility.
 *
 * Data attributes on children:
 *   [data-trigger]  — Element that opens the dialog on click
 *   [data-backdrop] — Overlay element (click to close)
 *   [data-panel]    — The dialog panel (receives focus)
 *   [data-close]    — Elements that close the dialog on click
 */
export class OmkDialog extends LitElement {
  static properties = {
    open: { type: Boolean, reflect: true },
  };

  // Light DOM — no shadow root, Tailwind classes pass through
  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this.open = false;
    this._previousFocus = null;
    this._onKeydown = this._handleKeydown.bind(this);
  }

  connectedCallback() {
    super.connectedCallback();

    this.querySelector("[data-trigger]")?.addEventListener("click", () =>
      this.show()
    );

    this.querySelector("[data-backdrop]")?.addEventListener("click", (e) => {
      // Only close if clicking the backdrop itself, not its children
      if (e.target === e.currentTarget) this.hide();
    });

    this.querySelectorAll("[data-close]").forEach((el) =>
      el.addEventListener("click", () => this.hide())
    );
  }

  show() {
    this._previousFocus = document.activeElement;
    this.open = true;
    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", this._onKeydown);

    // Focus first focusable element inside the panel
    requestAnimationFrame(() => {
      const panel = this.querySelector("[data-panel]");
      const focusable = panel?.querySelector(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      (focusable || panel)?.focus();
    });
  }

  hide() {
    this.open = false;
    document.body.style.overflow = "";
    document.removeEventListener("keydown", this._onKeydown);
    this._previousFocus?.focus();
  }

  _handleKeydown(e) {
    if (e.key === "Escape") {
      this.hide();
      return;
    }

    // Focus trap: cycle Tab within the panel
    if (e.key !== "Tab") return;
    const panel = this.querySelector("[data-panel]");
    const focusable = [
      ...(panel?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      ) || []),
    ];
    if (!focusable.length) return;

    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("keydown", this._onKeydown);
    document.body.style.overflow = "";
  }
}

customElements.define("omk-dialog", OmkDialog);
