import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-tabs> — Accessible tabbed interface with keyboard navigation.
 *
 * Features:
 *   - Left/Right arrow keys switch tabs
 *   - Home/End jump to first/last tab
 *   - ARIA: role="tablist"/"tab"/"tabpanel", aria-selected, aria-controls
 *   - Active tab set via data-default attribute
 *   - HTMX compatible (connectedCallback auto-initializes)
 *
 * Attributes:
 *   data-default — Value of the initially active tab
 *
 * Data attributes on children:
 *   [data-tab-list]    — The tab button strip (role="tablist")
 *   [data-tab-trigger] — Individual tab buttons, with data-value="..."
 *   [data-tab-content] — Tab panels, with data-value="..."
 */
export class OmkTabs extends LitElement {
  static properties = {
    active: { type: String, reflect: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this.active = "";
  }

  connectedCallback() {
    super.connectedCallback();

    // Set initial active tab from data-default
    if (!this.active && this.dataset.default) {
      this.active = this.dataset.default;
    }

    // Wire up trigger clicks
    this.querySelectorAll("[data-tab-trigger]").forEach((trigger) => {
      trigger.addEventListener("click", () => {
        this.select(trigger.dataset.value);
      });

      trigger.addEventListener("keydown", (e) => this._handleKeydown(e));
    });

    this._updateState();
  }

  select(value) {
    this.active = value;
    this._updateState();
  }

  _getTriggers() {
    return [...this.querySelectorAll("[data-tab-trigger]")];
  }

  _updateState() {
    const triggers = this._getTriggers();
    const panels = [...this.querySelectorAll("[data-tab-content]")];

    triggers.forEach((trigger) => {
      const isActive = trigger.dataset.value === this.active;
      trigger.setAttribute("aria-selected", isActive ? "true" : "false");
      trigger.setAttribute("tabindex", isActive ? "0" : "-1");

      // Update visual classes
      if (isActive) {
        trigger.classList.add("border-primary", "text-foreground");
        trigger.classList.remove(
          "border-transparent",
          "text-muted-foreground"
        );
      } else {
        trigger.classList.remove("border-primary", "text-foreground");
        trigger.classList.add("border-transparent", "text-muted-foreground");
      }
    });

    panels.forEach((panel) => {
      const isActive = panel.dataset.value === this.active;
      panel.hidden = !isActive;
    });
  }

  _handleKeydown(e) {
    const triggers = this._getTriggers();
    const current = triggers.indexOf(e.target);
    let next = -1;

    switch (e.key) {
      case "ArrowRight":
        e.preventDefault();
        next = current < triggers.length - 1 ? current + 1 : 0;
        break;
      case "ArrowLeft":
        e.preventDefault();
        next = current > 0 ? current - 1 : triggers.length - 1;
        break;
      case "Home":
        e.preventDefault();
        next = 0;
        break;
      case "End":
        e.preventDefault();
        next = triggers.length - 1;
        break;
    }

    if (next >= 0) {
      this.select(triggers[next].dataset.value);
      triggers[next].focus();
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
  }
}

customElements.define("omk-tabs", OmkTabs);
