import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-sidebar> — Collapsible sidebar with keyboard shortcut.
 *
 * Features:
 *   - Collapsible to icon-only width on desktop
 *   - Offcanvas on mobile (via CSS media queries)
 *   - Keyboard shortcut: Ctrl+B to toggle
 *   - Cookie-based state persistence (sidebar_state)
 *   - CSS custom property --sidebar-width for dynamic sizing
 *
 * Attributes:
 *   collapsed — Boolean, reflected. Controls collapsed state.
 *
 * Data attributes:
 *   [data-sidebar-trigger] — Toggle button
 *   [data-sidebar-panel]   — The sidebar panel
 *   [data-sidebar-overlay] — Mobile overlay backdrop
 */
export class OmkSidebar extends LitElement {
  static properties = {
    collapsed: { type: Boolean, reflect: true },
  };

  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    // Read initial state from cookie
    const saved = this._getCookie("sidebar_state");
    this.collapsed = saved === "collapsed";
    this._onKeydown = this._handleKeydown.bind(this);
  }

  connectedCallback() {
    super.connectedCallback();

    this.querySelectorAll("[data-sidebar-trigger]").forEach((trigger) => {
      trigger.addEventListener("click", () => this.toggle());
    });

    this.querySelector("[data-sidebar-overlay]")?.addEventListener(
      "click",
      () => this.collapse()
    );

    document.addEventListener("keydown", this._onKeydown);
    this._updateWidth();
  }

  toggle() {
    this.collapsed = !this.collapsed;
    this._persist();
    this._updateWidth();
  }

  collapse() {
    this.collapsed = true;
    this._persist();
    this._updateWidth();
  }

  expand() {
    this.collapsed = false;
    this._persist();
    this._updateWidth();
  }

  _updateWidth() {
    const panel = this.querySelector("[data-sidebar-panel]");
    if (panel) {
      panel.style.width = this.collapsed ? "var(--sidebar-width-collapsed, 3.5rem)" : "var(--sidebar-width, 16rem)";
    }
  }

  _persist() {
    document.cookie = `sidebar_state=${this.collapsed ? "collapsed" : "expanded"};path=/;max-age=${60 * 60 * 24 * 365};SameSite=Lax`;
  }

  _getCookie(name) {
    const match = document.cookie.match(new RegExp("(^| )" + name + "=([^;]+)"));
    return match ? match[2] : null;
  }

  _handleKeydown(e) {
    if ((e.ctrlKey || e.metaKey) && e.key === "b") {
      e.preventDefault();
      this.toggle();
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("keydown", this._onKeydown);
  }
}

customElements.define("omk-sidebar", OmkSidebar);
