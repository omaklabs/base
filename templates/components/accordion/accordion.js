import { LitElement } from "/assets/js/lit-all.min.js";

/**
 * <omk-accordion> — Accessible accordion with smooth height animation.
 *
 * Features:
 *   - Single or multiple open items (data-multiple attribute)
 *   - Smooth height animation via CSS transitions
 *   - aria-expanded, role="region", aria-controls
 *   - Keyboard: Enter/Space toggle, Up/Down navigate
 *   - HTMX compatible (connectedCallback auto-initializes)
 *
 * Data attributes:
 *   [data-accordion-item]    — Item container
 *   [data-accordion-trigger] — Toggle button, with data-value="..."
 *   [data-accordion-content] — Collapsible content, with data-value="..."
 */
export class OmkAccordion extends LitElement {
  createRenderRoot() {
    return this;
  }

  constructor() {
    super();
    this._handlers = [];
  }

  connectedCallback() {
    super.connectedCallback();
    this._multiple = this.hasAttribute("data-multiple");

    this.querySelectorAll("[data-accordion-trigger]").forEach((trigger) => {
      const onClick = () => this._toggle(trigger.dataset.value);
      const onKeydown = (e) => this._handleKeydown(e);
      trigger.addEventListener("click", onClick);
      trigger.addEventListener("keydown", onKeydown);
      this._handlers.push({ el: trigger, click: onClick, keydown: onKeydown });
    });

    // Close all content initially
    this.querySelectorAll("[data-accordion-content]").forEach((content) => {
      content.style.maxHeight = "0px";
      content.style.overflow = "hidden";
    });
  }

  _toggle(value) {
    const isOpen = this._isOpen(value);

    if (!this._multiple) {
      // Close all others
      this.querySelectorAll("[data-accordion-trigger]").forEach((t) => {
        if (t.dataset.value !== value) {
          this._close(t.dataset.value);
        }
      });
    }

    if (isOpen) {
      this._close(value);
    } else {
      this._open(value);
    }
  }

  _isOpen(value) {
    const trigger = this.querySelector(
      `[data-accordion-trigger][data-value="${value}"]`
    );
    return trigger?.getAttribute("aria-expanded") === "true";
  }

  _open(value) {
    const trigger = this.querySelector(
      `[data-accordion-trigger][data-value="${value}"]`
    );
    const content = this.querySelector(
      `[data-accordion-content][data-value="${value}"]`
    );
    if (!trigger || !content) return;

    trigger.setAttribute("aria-expanded", "true");
    content.hidden = false;
    content.style.maxHeight = content.scrollHeight + "px";
  }

  _close(value) {
    const trigger = this.querySelector(
      `[data-accordion-trigger][data-value="${value}"]`
    );
    const content = this.querySelector(
      `[data-accordion-content][data-value="${value}"]`
    );
    if (!trigger || !content) return;

    trigger.setAttribute("aria-expanded", "false");
    content.style.maxHeight = "0px";
  }

  _handleKeydown(e) {
    const triggers = [
      ...this.querySelectorAll("[data-accordion-trigger]"),
    ];
    const current = triggers.indexOf(e.target);

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        triggers[(current + 1) % triggers.length]?.focus();
        break;
      case "ArrowUp":
        e.preventDefault();
        triggers[(current - 1 + triggers.length) % triggers.length]?.focus();
        break;
      case "Home":
        e.preventDefault();
        triggers[0]?.focus();
        break;
      case "End":
        e.preventDefault();
        triggers[triggers.length - 1]?.focus();
        break;
    }
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    this._handlers.forEach(({ el, click, keydown }) => {
      el.removeEventListener("click", click);
      el.removeEventListener("keydown", keydown);
    });
    this._handlers = [];
  }
}

customElements.define("omk-accordion", OmkAccordion);
