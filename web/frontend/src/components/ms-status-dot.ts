import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

@customElement("ms-status-dot")
export class MsStatusDot extends LitElement {
  @property() status: "online" | "offline" | "unknown" = "unknown";

  static styles = css`
    :host {
      display: inline-block;
    }
    .dot {
      width: 10px;
      height: 10px;
      border-radius: 50%;
      display: inline-block;
    }
    .dot.online {
      background: var(--success-color, #4caf50);
    }
    .dot.offline {
      background: var(--error-color, #db4437);
    }
    .dot.unknown {
      background: var(--disabled-text-color, #bdbdbd);
    }
  `;

  render() {
    return html`<span class="dot ${this.status}"></span>`;
  }
}
