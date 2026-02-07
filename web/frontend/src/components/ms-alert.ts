import { LitElement, html, css, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

@customElement("ms-alert")
export class MsAlert extends LitElement {
  @property() alertType: "error" | "warning" | "info" | "success" = "info";
  @property() title?: string;
  @property({ type: Boolean }) dismissable = false;

  static styles = css`
    :host {
      display: block;
      margin-bottom: var(--ha-space-3, 12px);
    }
    .alert {
      display: flex;
      align-items: flex-start;
      gap: var(--ha-space-3, 12px);
      padding: var(--ha-space-3) var(--ha-space-4);
      border-radius: var(--ha-border-radius-md, 8px);
      font-size: var(--ha-font-size-m, 14px);
      color: var(--primary-text-color);
    }
    .alert.error {
      background: #fce4ec;
      border-left: 4px solid var(--error-color);
    }
    .alert.warning {
      background: #fff3e0;
      border-left: 4px solid var(--warning-color);
    }
    .alert.info {
      background: #e1f5fe;
      border-left: 4px solid var(--info-color);
    }
    .alert.success {
      background: #e8f5e9;
      border-left: 4px solid var(--success-color);
    }
    .title {
      font-weight: var(--ha-font-weight-medium, 500);
    }
    .dismiss {
      margin-left: auto;
      cursor: pointer;
      background: none;
      border: none;
      font-size: 18px;
      color: var(--secondary-text-color);
    }
    .body {
      flex: 1;
    }
  `;

  render() {
    return html`
      <div class="alert ${this.alertType}">
        <div class="body">
          ${this.title
            ? html`<div class="title">${this.title}</div>`
            : nothing}
          <slot></slot>
        </div>
        ${this.dismissable
          ? html`<button class="dismiss" @click=${this._dismiss}>x</button>`
          : nothing}
      </div>
    `;
  }

  private _dismiss() {
    this.dispatchEvent(new CustomEvent("dismissed"));
    this.remove();
  }
}
