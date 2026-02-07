import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

@customElement("ms-button")
export class MsButton extends LitElement {
  @property() variant: "brand" | "neutral" | "danger" = "brand";
  @property() appearance: "filled" | "outlined" | "plain" = "filled";
  @property({ type: Boolean }) disabled = false;
  @property({ type: Boolean }) loading = false;

  static styles = css`
    button {
      border-radius: var(--ha-border-radius-pill, 9999px);
      height: 40px;
      padding: 0 var(--ha-space-6, 24px);
      font-size: var(--ha-font-size-m, 14px);
      font-weight: var(--ha-font-weight-medium, 500);
      font-family: inherit;
      cursor: pointer;
      border: none;
      transition: background var(--ha-animation-duration-fast, 150ms),
        opacity var(--ha-animation-duration-fast, 150ms);
      display: inline-flex;
      align-items: center;
      gap: var(--ha-space-2, 8px);
    }
    button:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
    /* Filled */
    button.brand.filled {
      background: var(--primary-color);
      color: white;
    }
    button.danger.filled {
      background: var(--error-color);
      color: white;
    }
    button.neutral.filled {
      background: var(--divider-color);
      color: var(--primary-text-color);
    }
    /* Outlined */
    button.brand.outlined {
      background: transparent;
      border: 1px solid var(--primary-color);
      color: var(--primary-color);
    }
    button.danger.outlined {
      background: transparent;
      border: 1px solid var(--error-color);
      color: var(--error-color);
    }
    button.neutral.outlined {
      background: transparent;
      border: 1px solid var(--divider-color);
      color: var(--primary-text-color);
    }
    /* Plain */
    button.brand.plain {
      background: transparent;
      color: var(--primary-color);
    }
    button.danger.plain {
      background: transparent;
      color: var(--error-color);
    }
    button.neutral.plain {
      background: transparent;
      color: var(--primary-text-color);
    }
    button:not(:disabled):hover {
      opacity: 0.85;
    }
  `;

  render() {
    return html`
      <button
        class="${this.variant} ${this.appearance}"
        ?disabled=${this.disabled || this.loading}
      >
        ${this.loading ? html`<span>...</span>` : ""}
        <slot></slot>
      </button>
    `;
  }
}
