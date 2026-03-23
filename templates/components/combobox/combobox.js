import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-combobox> — Searchable select with keyboard navigation.
 *
 * Features:
 *   - Text input for filtering options
 *   - Arrow key navigation (Up/Down)
 *   - Enter to select, ESC to close
 *   - Click-outside dismiss
 *   - Hidden input for form submission
 *   - aria-expanded, role="listbox", role="option"
 *
 * Data attributes:
 *   [data-combobox-input]  — Search input
 *   [data-combobox-list]   — Options container
 *   [data-combobox-option] — Individual option, with data-value and data-label
 *   [data-combobox-hidden] — Hidden input for form value
 *   [data-combobox-display] — Display text for selected value
 */
export class OmkCombobox extends LitElement {
  static properties = {
    open: { type: Boolean, reflect: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this.open = false;
    this._focusIndex = -1;
    this._onDocClick = this._handleDocClick.bind(this);
  }

  connectedCallback() {
    super.connectedCallback();

    const input = this.querySelector("[data-combobox-input]");
    if (input) {
      input.addEventListener("focus", () => this._show());
      input.addEventListener("input", () => this._filter(input.value));
      input.addEventListener("keydown", (e) => this._handleKeydown(e));
    }
  }

  _show() {
    this.open = true;
    this._focusIndex = -1;
    this._filter("");
    document.addEventListener("click", this._onDocClick);
  }

  _hide() {
    this.open = false;
    this._focusIndex = -1;
    document.removeEventListener("click", this._onDocClick);
  }

  _getVisibleOptions() {
    return [
      ...this.querySelectorAll(
        '[data-combobox-option]:not([data-filtered="true"])'
      ),
    ];
  }

  _filter(query) {
    const q = query.toLowerCase();
    this.querySelectorAll("[data-combobox-option]").forEach((opt) => {
      const label = (opt.dataset.label || opt.textContent).toLowerCase();
      const matches = !q || label.includes(q);
      opt.setAttribute("data-filtered", matches ? "false" : "true");
      opt.style.display = matches ? "" : "none";
    });
    this._focusIndex = -1;
  }

  _select(option) {
    const value = option.dataset.value;
    const label = option.dataset.label || option.textContent.trim();

    const hidden = this.querySelector("[data-combobox-hidden]");
    if (hidden) hidden.value = value;

    const input = this.querySelector("[data-combobox-input]");
    if (input) input.value = label;

    this._hide();
    this.dispatchEvent(
      new CustomEvent("change", { detail: { value, label }, bubbles: true })
    );
  }

  _handleKeydown(e) {
    const options = this._getVisibleOptions();

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        this._focusIndex = Math.min(
          this._focusIndex + 1,
          options.length - 1
        );
        this._highlightOption(options);
        break;
      case "ArrowUp":
        e.preventDefault();
        this._focusIndex = Math.max(this._focusIndex - 1, 0);
        this._highlightOption(options);
        break;
      case "Enter":
        e.preventDefault();
        if (this._focusIndex >= 0 && options[this._focusIndex]) {
          this._select(options[this._focusIndex]);
        }
        break;
      case "Escape":
        e.preventDefault();
        this._hide();
        break;
    }
  }

  _highlightOption(options) {
    options.forEach((opt, i) => {
      if (i === this._focusIndex) {
        opt.classList.add("bg-accent", "text-accent-foreground");
        opt.scrollIntoView({ block: "nearest" });
      } else {
        opt.classList.remove("bg-accent", "text-accent-foreground");
      }
    });
  }

  _handleDocClick(e) {
    if (!this.contains(e.target)) {
      this._hide();
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("click", this._onDocClick);
  }
}

customElements.define("omk-combobox", OmkCombobox);
