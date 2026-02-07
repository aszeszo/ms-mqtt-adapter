import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";

@customElement("ms-view-dashboard")
export class MsViewDashboard extends LitElement {
  @state() private _status: any = null;
  @state() private _loading = true;
  @state() private _error = "";

  static styles = css`
    :host {
      display: block;
    }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
      gap: var(--ha-space-4, 16px);
    }
    .kv {
      display: flex;
      justify-content: space-between;
      padding: var(--ha-space-1, 4px) 0;
      font-size: var(--ha-font-size-m, 14px);
    }
    .kv-label {
      color: var(--secondary-text-color);
    }
    .kv-value {
      display: flex;
      align-items: center;
      gap: var(--ha-space-1, 4px);
    }
    .node-chips {
      display: flex;
      flex-wrap: wrap;
      gap: var(--ha-space-1, 4px);
      margin-top: var(--ha-space-2, 8px);
    }
    .chip {
      padding: 2px 8px;
      border-radius: var(--ha-border-radius-pill, 9999px);
      background: var(--primary-background-color, #fafafa);
      border: 1px solid var(--divider-color);
      font-size: var(--ha-font-size-s, 12px);
      display: inline-flex;
      align-items: center;
      gap: 4px;
    }
    .summary {
      margin-bottom: var(--ha-space-4, 16px);
      display: flex;
      gap: var(--ha-space-6, 24px);
      font-size: var(--ha-font-size-m, 14px);
      color: var(--secondary-text-color);
    }
    .loading {
      text-align: center;
      padding: var(--ha-space-8, 32px);
      color: var(--secondary-text-color);
    }
    .config-warning {
      margin-bottom: var(--ha-space-4, 16px);
    }
    .missing-list {
      margin: var(--ha-space-3, 12px) 0 0 0;
      padding-left: var(--ha-space-5, 20px);
    }
    .missing-list li {
      margin: var(--ha-space-1, 4px) 0;
      font-family: var(--ha-font-family-code, monospace);
      font-size: var(--ha-font-size-s, 12px);
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
    document.addEventListener("ws-initial_state", this._onWSState as EventListener);
    document.addEventListener("ws-connection_changed", this._onWSState as EventListener);
    document.addEventListener("ws-config_reloaded", (() => this._load()) as EventListener);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("ws-initial_state", this._onWSState as EventListener);
    document.removeEventListener("ws-connection_changed", this._onWSState as EventListener);
    document.removeEventListener("ws-config_reloaded", (() => this._load()) as EventListener);
  }

  private _onWSState = (e: Event) => {
    const detail = (e as CustomEvent).detail;
    if (detail) {
      this._status = { ...this._status, ...detail };
    }
  };

  private async _load() {
    try {
      this._status = await api.getStatus();
      this._error = "";
    } catch (e: any) {
      this._error = e.message;
    } finally {
      this._loading = false;
    }
  }

  render() {
    if (this._loading) return html`<div class="loading">Loading...</div>`;
    if (this._error)
      return html`<ms-alert alertType="error" title="Error"
        >${this._error}</ms-alert
      >`;

    const s = this._status;
    const entityCount = s?.entities ? Object.keys(s.entities).length : 0;
    const configStatus = s?.config_status;

    return html`
      ${configStatus && !configStatus.complete
        ? html`
            <div class="config-warning">
              <ms-alert alertType="warning" title="Configuration Incomplete">
                <p>
                  The adapter configuration is incomplete. Please configure the
                  following required settings via the
                  <strong>MQTT</strong> and <strong>Gateways</strong> tabs:
                </p>
                <ul class="missing-list">
                  ${configStatus.missing_fields.map(
                    (field: string) => html`<li>${field}</li>`
                  )}
                </ul>
              </ms-alert>
            </div>
          `
        : nothing}

      <div class="summary">
        <span>Entities: ${entityCount}</span>
        <span
          >Gateways: ${s?.gateways ? Object.keys(s.gateways).length : 0}</span
        >
      </div>
      <div class="cards">
        ${this._renderMqttCard()} ${this._renderGatewayCards()}
      </div>
    `;
  }

  private _renderMqttCard() {
    const mqtt = this._status?.mqtt;
    return html`
      <ms-card header="MQTT">
        <div class="card-content">
          <div class="kv">
            <span class="kv-label">Status</span>
            <span class="kv-value">
              <ms-status-dot
                .status=${mqtt?.connected ? "online" : "offline"}
              ></ms-status-dot>
              ${mqtt?.connected ? "Connected" : "Disconnected"}
            </span>
          </div>
          <div class="kv">
            <span class="kv-label">Broker</span>
            <span class="kv-value">${mqtt?.broker || "N/A"}</span>
          </div>
          <div class="kv">
            <span class="kv-label">Port</span>
            <span class="kv-value">${mqtt?.port || "N/A"}</span>
          </div>
        </div>
      </ms-card>
    `;
  }

  private _renderGatewayCards() {
    const gateways = this._status?.gateways;
    if (!gateways) return nothing;
    return Object.entries(gateways).map(
      ([name, gw]: [string, any]) => html`
        <ms-card header=${name}>
          <div class="card-content">
            <div class="kv">
              <span class="kv-label">Status</span>
              <span class="kv-value">
                <ms-status-dot
                  .status=${gw.connected ? "online" : "offline"}
                ></ms-status-dot>
                ${gw.connected ? "Connected" : "Disconnected"}
              </span>
            </div>
            <div class="kv">
              <span class="kv-label">Transport</span>
              <span class="kv-value">${gw.transport || "N/A"}</span>
            </div>
            <div class="kv">
              <span class="kv-label">Last Issued ID</span>
              <span class="kv-value">${gw.last_issued_node_id || "N/A"}</span>
            </div>
            ${gw.seen_nodes?.length
              ? html`
                  <div class="kv">
                    <span class="kv-label">Seen IDs</span>
                  </div>
                  <div class="node-chips">
                    ${gw.seen_nodes.map(
                      (n: number) => html`
                        <span class="chip">
                          <ms-status-dot
                            .status=${gw.node_availability?.[n]
                              ? "online"
                              : "offline"}
                          ></ms-status-dot>
                          ${n}
                        </span>
                      `
                    )}
                  </div>
                `
              : nothing}
          </div>
        </ms-card>
      `
    );
  }
}
