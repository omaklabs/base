import { LitElement, html } from "/assets/js/lit-all.min.js";

/**
 * <omk-tagsinput> — Multi-tag input with keyboard support.
 *
 * Features:
 *   - Enter/comma adds a tag
 *   - Backspace removes last tag when input is empty
 *   - X button removes specific tag
 *   - Hidden inputs for form submission
 *   - Duplicate prevention
 *
 * Attributes:
 *   data-name — Form field name for hidden inputs
 *
 * Data attributes:
 *   [data-tagsinput-input]   — The text input
 *   [data-tagsinput-chips]   — Container for tag badges
 *   [data-tagsinput-hidden]  — Container for hidden form inputs
 */
export class OmkTagsInput extends LitElement {
  static properties = {
    _tags: { state: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this._tags = [];
  }

  connectedCallback() {
    super.connectedCallback();

    // Read initial tags from hidden inputs
    const hiddenContainer = this.querySelector("[data-tagsinput-hidden]");
    if (hiddenContainer) {
      this._tags = [...hiddenContainer.querySelectorAll("input")].map(
        (i) => i.value
      );
    }

    const input = this.querySelector("[data-tagsinput-input]");
    if (input) {
      input.addEventListener("keydown", (e) => this._handleKeydown(e, input));
    }

    // Delegate click on remove buttons
    this.addEventListener("click", (e) => {
      const removeBtn = e.target.closest("[data-tagsinput-remove]");
      if (removeBtn) {
        const tag = removeBtn.dataset.tagsinputRemove;
        this._removeTag(tag);
      }
    });
  }

  _handleKeydown(e, input) {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      const value = input.value.trim().replace(/,/g, "");
      if (value && !this._tags.includes(value)) {
        this._tags = [...this._tags, value];
        this._updateDOM();
      }
      input.value = "";
    } else if (e.key === "Backspace" && input.value === "") {
      if (this._tags.length > 0) {
        this._tags = this._tags.slice(0, -1);
        this._updateDOM();
      }
    }
  }

  _removeTag(tag) {
    this._tags = this._tags.filter((t) => t !== tag);
    this._updateDOM();
  }

  _updateDOM() {
    const name = this.dataset.name || "tags";
    const chipsContainer = this.querySelector("[data-tagsinput-chips]");
    const hiddenContainer = this.querySelector("[data-tagsinput-hidden]");

    // Update badges
    if (chipsContainer) {
      chipsContainer.innerHTML = this._tags
        .map(
          (tag) => `
        <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-[calc(var(--radius)-2px)] text-xs font-medium bg-secondary text-secondary-foreground">
          <span>${this._escapeHtml(tag)}</span>
          <button type="button" class="hover:text-destructive cursor-pointer" data-tagsinput-remove="${this._escapeAttr(tag)}">
            <svg class="h-3 w-3 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </span>
      `
        )
        .join("");
    }

    // Update hidden inputs
    if (hiddenContainer) {
      hiddenContainer.innerHTML = this._tags
        .map(
          (tag) =>
            `<input type="hidden" name="${this._escapeAttr(name)}" value="${this._escapeAttr(tag)}">`
        )
        .join("");
    }

    this.dispatchEvent(
      new CustomEvent("change", {
        detail: { tags: this._tags },
        bubbles: true,
      })
    );
  }

  _escapeHtml(str) {
    const div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  _escapeAttr(str) {
    return str.replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
}

customElements.define("omk-tagsinput", OmkTagsInput);
