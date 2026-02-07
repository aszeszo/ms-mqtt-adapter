import { LitElement, html, css, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

@customElement("ms-dialog")
export class MsDialog extends LitElement {
  @property({ type: Boolean, reflect: true }) open = false;
  @property() headerTitle = "";
  @property() width: "small" | "medium" | "large" = "medium";

  static styles = css`
    :host {
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      z-index: 1000;
      display: none;
    }
    :host([open]) {
      display: block;
    }
    .scrim {
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background: rgba(0, 0, 0, 0.32);
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .dialog {
      background: var(--card-background-color, white);
      border-radius: var(--ha-border-radius-xl, 16px);
      max-height: 90vh;
      overflow-y: auto;
      box-shadow: 0 8px 32px rgba(0, 0, 0, 0.24);
    }
    .dialog.small {
      width: 320px;
    }
    .dialog.medium {
      width: 560px;
    }
    .dialog.large {
      width: 720px;
    }
    .header {
      padding: var(--ha-space-4) var(--ha-space-6);
      font-size: var(--ha-font-size-xl, 20px);
      font-weight: var(--ha-font-weight-medium, 500);
      color: var(--primary-text-color);
    }
    .body {
      padding: 0 var(--ha-space-6) var(--ha-space-6);
    }
    .footer {
      display: flex;
      justify-content: flex-end;
      gap: var(--ha-space-3);
      padding: var(--ha-space-2) var(--ha-space-6) var(--ha-space-4);
    }
  `;

  render() {
    if (!this.open) return nothing;
    return html`
      <div class="scrim" @click=${this._onScrimClick}>
        <div class="dialog ${this.width}" @click=${(e: Event) => e.stopPropagation()}>
          <div class="header">${this.headerTitle}</div>
          <div class="body"><slot></slot></div>
          <div class="footer"><slot name="footer"></slot></div>
        </div>
      </div>
    `;
  }

  private _onScrimClick() {
    this.open = false;
    this.dispatchEvent(new CustomEvent("closed"));
  }
}
