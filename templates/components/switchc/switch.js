import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-switch> — Accessible toggle switch.
 *
 * Features:
 *   - Click and Space/Enter toggle
 *   - aria-checked state
 *   - Hidden checkbox for form submission
 *   - Disabled state
 *   - HTMX compatible (connectedCallback auto-initializes)
 *
 * Attributes:
 *   checked  — Boolean, reflected. Current state.
 *   disabled — Boolean, reflected. Prevents interaction.
 *
 * Data attributes on children:
 *   [data-track]  — The switch track element
 *   [data-thumb]  — The switch thumb element
 *   input[data-input] — Hidden checkbox for form submission
 */
export class OmkSwitch extends LitElement {
  static properties = {
    checked: { type: Boolean, reflect: true },
    disabled: { type: Boolean, reflect: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this.checked = false;
    this.disabled = false;
  }

  connectedCallback() {
    super.connectedCallback();

    const track = this.querySelector("[data-track]");
    if (track) {
      track.addEventListener("click", () => this.toggle());
      track.addEventListener("keydown", (e) => {
        if (e.key === " " || e.key === "Enter") {
          e.preventDefault();
          this.toggle();
        }
      });
    }

    this._updateState();
  }

  toggle() {
    if (this.disabled) return;
    this.checked = !this.checked;
    this._updateState();

    // Dispatch change event for form integration
    this.dispatchEvent(
      new Event("change", { bubbles: true, composed: true })
    );
  }

  _updateState() {
    const track = this.querySelector("[data-track]");
    const thumb = this.querySelector("[data-thumb]");
    const input = this.querySelector("[data-input]");

    if (track) {
      track.setAttribute("aria-checked", this.checked ? "true" : "false");
      if (this.checked) {
        track.classList.add("bg-primary");
        track.classList.remove("bg-secondary");
      } else {
        track.classList.remove("bg-primary");
        track.classList.add("bg-secondary");
      }
    }

    if (thumb) {
      if (this.checked) {
        thumb.classList.add("translate-x-5");
        thumb.classList.remove("translate-x-0");
      } else {
        thumb.classList.remove("translate-x-5");
        thumb.classList.add("translate-x-0");
      }
    }

    if (input) {
      input.checked = this.checked;
    }
  }
}

customElements.define("omk-switch", OmkSwitch);
