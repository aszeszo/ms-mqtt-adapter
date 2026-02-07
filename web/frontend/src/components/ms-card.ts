import { LitElement, html, css, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

@customElement("ms-card")
export class MsCard extends LitElement {
  @property() header?: string;

  static styles = css`
    :host {
      background: var(--ha-card-background, var(--card-background-color, white));
      border-radius: var(--ha-card-border-radius, var(--ha-border-radius-lg));
      border: var(--ha-card-border-width, 1px) solid
        var(--ha-card-border-color, var(--divider-color));
      box-shadow: var(--ha-card-box-shadow, none);
      color: var(--primary-text-color);
      display: block;
      position: relative;
    }
    .card-header {
      color: var(--primary-text-color);
      font-size: var(--ha-font-size-2xl, 24px);
      line-height: 2;
      padding: var(--ha-space-3) var(--ha-space-4) 0;
      font-weight: var(--ha-font-weight-normal);
    }
    ::slotted(.card-content) {
      padding: var(--ha-space-4);
    }
    ::slotted(.card-actions) {
      border-top: 1px solid var(--divider-color);
      padding: var(--ha-space-2) var(--ha-space-4);
      display: flex;
      gap: var(--ha-space-2);
    }
  `;

  render() {
    return html`
      ${this.header
        ? html`<h1 class="card-header">${this.header}</h1>`
        : nothing}
      <slot></slot>
    `;
  }
}
