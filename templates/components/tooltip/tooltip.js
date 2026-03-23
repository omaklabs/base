import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-tooltip> — Accessible tooltip with delayed show and keyboard support.
 *
 * Features:
 *   - Delayed show (150ms) to avoid flicker on mouse pass-through
 *   - Show on mouseenter + focusin, hide on mouseleave + focusout
 *   - ESC hides the tooltip
 *   - role="tooltip", aria-describedby linking
 *   - Position: top (default), bottom, left, right
 *   - HTMX compatible (connectedCallback auto-initializes)
 *
 * Attributes:
 *   data-text     — Tooltip text
 *   data-position — "top" (default), "bottom", "left", "right"
 */
export class OmkTooltip extends LitElement {
  static properties = {
    _visible: { state: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this._visible = false;
    this._showTimer = null;
  }

  connectedCallback() {
    super.connectedCallback();

    this.addEventListener("mouseenter", () => this._scheduleShow());
    this.addEventListener("mouseleave", () => this._hide());
    this.addEventListener("focusin", () => this._scheduleShow());
    this.addEventListener("focusout", () => this._hide());
    this.addEventListener("keydown", (e) => {
      if (e.key === "Escape") this._hide();
    });

    // Create tooltip element
    this._tooltipEl = document.createElement("div");
    this._tooltipEl.setAttribute("role", "tooltip");
    this._tooltipEl.className = this._tooltipClasses();
    this._tooltipEl.textContent = this.dataset.text || "";
    this._tooltipEl.hidden = true;
    this.appendChild(this._tooltipEl);

    // Set up aria-describedby on the first child element
    const id = "tooltip-" + Math.random().toString(36).slice(2, 9);
    this._tooltipEl.id = id;
    const target = this.firstElementChild;
    if (target && target !== this._tooltipEl) {
      target.setAttribute("aria-describedby", id);
    }
  }

  _scheduleShow() {
    clearTimeout(this._showTimer);
    this._showTimer = setTimeout(() => {
      this._tooltipEl.hidden = false;
      this._visible = true;
    }, 150);
  }

  _hide() {
    clearTimeout(this._showTimer);
    this._tooltipEl.hidden = true;
    this._visible = false;
  }

  _tooltipClasses() {
    const pos = this.dataset.position || "top";
    const base =
      "absolute z-50 px-2 py-1 text-xs rounded-md bg-foreground text-background whitespace-nowrap pointer-events-none transition-opacity duration-100";

    let posClass;
    switch (pos) {
      case "bottom":
        posClass = "top-full left-1/2 -translate-x-1/2 mt-2";
        break;
      case "left":
        posClass = "right-full top-1/2 -translate-y-1/2 mr-2";
        break;
      case "right":
        posClass = "left-full top-1/2 -translate-y-1/2 ml-2";
        break;
      default:
        posClass = "bottom-full left-1/2 -translate-x-1/2 mb-2";
    }

    return base + " " + posClass;
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    clearTimeout(this._showTimer);
  }
}

customElements.define("omk-tooltip", OmkTooltip);
