import { LitElement, html, css, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

interface ToastMessage {
  id: number;
  type: "success" | "error" | "warning" | "info";
  text: string;
}

let nextId = 0;

@customElement("ms-toast")
export class MsToast extends LitElement {
  @state() private _messages: ToastMessage[] = [];

  static styles = css`
    :host {
      position: fixed;
      bottom: var(--ha-space-4, 16px);
      right: var(--ha-space-4, 16px);
      z-index: 2000;
      display: flex;
      flex-direction: column-reverse;
      gap: var(--ha-space-2, 8px);
    }
    .toast {
      padding: var(--ha-space-3) var(--ha-space-4);
      border-radius: var(--ha-border-radius-md, 8px);
      font-size: var(--ha-font-size-m, 14px);
      color: white;
      min-width: 250px;
      max-width: 400px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
      animation: slideIn var(--ha-animation-duration-normal, 250ms) ease-out;
    }
    .toast.success { background: var(--success-color, #4caf50); }
    .toast.error { background: var(--error-color, #db4437); }
    .toast.warning { background: var(--warning-color, #ff9800); }
    .toast.info { background: var(--info-color, #03a9f4); }
    @keyframes slideIn {
      from { transform: translateX(100%); opacity: 0; }
      to { transform: translateX(0); opacity: 1; }
    }
  `;

  show(type: ToastMessage["type"], text: string, duration = 4000) {
    const id = nextId++;
    this._messages = [...this._messages, { id, type, text }];
    setTimeout(() => {
      this._messages = this._messages.filter((m) => m.id !== id);
    }, duration);
  }

  render() {
    return html`
      ${this._messages.map(
        (m) => html`<div class="toast ${m.type}">${m.text}</div>`
      )}
    `;
  }
}
