import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-sheet> — Accessible side panel / drawer.
 *
 * Same pattern as Dialog: focus trapping, ESC close, backdrop click,
 * body scroll lock. Slides from left or right (controlled by data-side).
 *
 * Attributes:
 *   open      — Boolean, reflected. Controls visibility.
 *   data-side — "left" or "right" (default)
 *
 * Data attributes on children:
 *   [data-trigger]  — Element that opens the sheet
 *   [data-backdrop] — Overlay element
 *   [data-panel]    — The sheet panel
 *   [data-close]    — Elements that close the sheet
 */
export class OmkSheet extends LitElement {
  static properties = {
    open: { type: Boolean, reflect: true },
  };

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

customElements.define("omk-sheet", OmkSheet);
