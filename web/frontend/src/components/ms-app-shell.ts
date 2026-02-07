import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { type Route, currentRoute } from "../router";

@customElement("ms-app-shell")
export class MsAppShell extends LitElement {
  @state() private _route: Route = currentRoute();
  @state() private _mqttConnected = false;
  @state() private _gatewayStatuses: Record<string, boolean> = {};

  static styles = css`
    :host {
      display: block;
      height: 100vh;
      background: var(--primary-background-color, #fafafa);
      color: var(--primary-text-color, #212121);
      font-family: var(--ha-font-family-body, Roboto, Noto, sans-serif);
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: var(--ha-space-3, 12px) var(--ha-space-4, 16px);
      background: var(--card-background-color, white);
      border-bottom: 1px solid var(--divider-color, #e0e0e0);
    }
    .header h1 {
      margin: 0;
      font-size: var(--ha-font-size-l, 16px);
      font-weight: var(--ha-font-weight-medium, 500);
    }
    .status-indicators {
      display: flex;
      gap: var(--ha-space-4, 16px);
      align-items: center;
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
    }
    .status-item {
      display: flex;
      align-items: center;
      gap: var(--ha-space-1, 4px);
    }
    .tab-bar {
      background: var(--card-background-color, white);
      border-bottom: 1px solid var(--divider-color, #e0e0e0);
      overflow-x: auto;
    }
    .main {
      overflow-y: auto;
      padding: var(--ha-space-4, 16px);
      height: calc(100vh - 98px);
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    window.addEventListener("hashchange", this._onHashChange);
    document.addEventListener("ws-initial_state", this._onInitialState as EventListener);
    document.addEventListener("ws-connection_changed", this._onConnectionChanged as EventListener);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    window.removeEventListener("hashchange", this._onHashChange);
    document.removeEventListener("ws-initial_state", this._onInitialState as EventListener);
    document.removeEventListener("ws-connection_changed", this._onConnectionChanged as EventListener);
  }

  private _onHashChange = () => {
    this._route = currentRoute();
  };

  private _onInitialState = (e: CustomEvent) => {
    const data = e.detail;
    if (data?.mqtt) this._mqttConnected = data.mqtt.connected;
    if (data?.gateways) {
      const s: Record<string, boolean> = {};
      for (const [name, gw] of Object.entries(data.gateways) as any) {
        s[name] = gw.connected;
      }
      this._gatewayStatuses = s;
    }
  };

  private _onConnectionChanged = (e: CustomEvent) => {
    const data = e.detail;
    if (data?.type === "mqtt") {
      this._mqttConnected = data.connected;
    } else if (data?.type === "gateway" && data.name) {
      this._gatewayStatuses = {
        ...this._gatewayStatuses,
        [data.name]: data.connected,
      };
    }
  };

  render() {
    return html`
      <div class="header">
        <h1>MySensors MQTT Adapter</h1>
        <div class="status-indicators">
          <span class="status-item">
            MQTT
            <ms-status-dot
              .status=${this._mqttConnected ? "online" : "offline"}
            ></ms-status-dot>
          </span>
          ${Object.entries(this._gatewayStatuses).map(
            ([name, connected]) => html`
              <span class="status-item">
                ${name}
                <ms-status-dot
                  .status=${connected ? "online" : "offline"}
                ></ms-status-dot>
              </span>
            `
          )}
        </div>
      </div>
      <div class="tab-bar">
        <ms-sidebar
          .activeRoute=${this._route}
          @route-changed=${(e: CustomEvent) => (this._route = e.detail)}
        ></ms-sidebar>
      </div>
      <div class="main">${this._renderView()}</div>
      <ms-toast id="toast"></ms-toast>
    `;
  }

  private _renderView() {
    switch (this._route) {
      case "dashboard":
        return html`<ms-view-dashboard></ms-view-dashboard>`;
      case "devices":
        return html`<ms-view-devices></ms-view-devices>`;
      case "gateways":
        return html`<ms-view-gateways></ms-view-gateways>`;
      case "mqtt":
        return html`<ms-view-mqtt></ms-view-mqtt>`;
      case "mqtt-topics":
        return html`<ms-view-mqtt-topics></ms-view-mqtt-topics>`;
      case "aliases":
        return html`<ms-view-aliases></ms-view-aliases>`;
      case "logs":
        return html`<ms-view-logs></ms-view-logs>`;
      case "editor":
        return html`<ms-view-editor></ms-view-editor>`;
      default:
        return html`<ms-view-dashboard></ms-view-dashboard>`;
    }
  }
}
