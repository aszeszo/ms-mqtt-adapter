import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";
import type { FormSchema } from "../components/ms-form";

@customElement("ms-view-gateways")
export class MsViewGateways extends LitElement {
  @state() private _gateways: Record<string, any> = {};
  @state() private _loading = true;
  @state() private _showDialog = false;
  @state() private _editingName = "";
  @state() private _editingGateway: any = null;

  static styles = css`
    :host { display: block; }
    .toolbar {
      display: flex; justify-content: flex-end;
      margin-bottom: var(--ha-space-4, 16px);
    }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
      gap: var(--ha-space-4, 16px);
    }
    .section-title {
      font-size: var(--ha-font-size-s, 12px);
      font-weight: var(--ha-font-weight-medium, 500);
      color: var(--secondary-text-color);
      text-transform: uppercase;
      margin: var(--ha-space-4, 16px) 0 var(--ha-space-2, 8px);
      padding-top: var(--ha-space-2, 8px);
      border-top: 1px solid var(--divider-color);
    }
    .section-title:first-child { border-top: none; margin-top: 0; padding-top: 0; }
    .loading { text-align: center; padding: var(--ha-space-8, 32px); color: var(--secondary-text-color); }
    .dialog-field { margin-bottom: var(--ha-space-3, 12px); }
    .dialog-field label {
      display: block; font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color); margin-bottom: 4px;
    }
    .dialog-field input {
      width: 100%; box-sizing: border-box;
      padding: var(--ha-space-2) var(--ha-space-3);
      border: 1px solid var(--divider-color); border-radius: var(--ha-border-radius-sm);
      font-size: var(--ha-font-size-m); font-family: inherit;
      background: var(--card-background-color); color: var(--primary-text-color);
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
  }

  private async _load() {
    try {
      this._gateways = await api.getMySensors();
    } catch (e) { /* ignore */ }
    this._loading = false;
  }

  render() {
    if (this._loading) return html`<div class="loading">Loading...</div>`;
    return html`
      <div class="toolbar">
        <ms-button @click=${this._addGateway}>Add Gateway</ms-button>
      </div>
      <div class="cards">
        ${Object.entries(this._gateways).map(([name, gw]) =>
          this._renderGatewayCard(name, gw)
        )}
      </div>
      ${this._renderDialog()}
    `;
  }

  private _renderGatewayCard(name: string, gw: any) {
    const transport = gw.transport || "ethernet";
    return html`
      <ms-card header=${name}>
        <div class="card-content">
          <div class="section-title">Transport</div>
          <ms-form
            .schema=${this._transportSchema()}
            .data=${{ transport }}
            @value-changed=${(e: CustomEvent) => {
              gw.transport = e.detail.value.transport;
              this._gateways = { ...this._gateways, [name]: { ...gw } };
            }}
          ></ms-form>

          ${transport === "ethernet" ? html`
            <ms-form
              .schema=${this._ethernetSchema()}
              .data=${{ host: gw.ethernet?.host || "", port: gw.ethernet?.port || 5003 }}
              @value-changed=${(e: CustomEvent) => {
                if (!gw.ethernet) gw.ethernet = {};
                gw.ethernet.host = e.detail.value.host;
                gw.ethernet.port = e.detail.value.port;
                this._gateways = { ...this._gateways, [name]: { ...gw } };
              }}
            ></ms-form>
          ` : html`
            <ms-form
              .schema=${this._rs485Schema()}
              .data=${{ device: gw.rs485?.device || "" }}
              @value-changed=${(e: CustomEvent) => {
                if (!gw.rs485) gw.rs485 = {};
                gw.rs485.device = e.detail.value.device;
                this._gateways = { ...this._gateways, [name]: { ...gw } };
              }}
            ></ms-form>
          `}

          <div class="section-title">Node ID Assignment</div>
          <ms-form
            .schema=${this._nodeIdSchema()}
            .data=${{
              id_assignment_enabled: gw.gateway?.node_id_assignment?.enabled ?? true,
              id_range_start: gw.gateway?.node_id_assignment?.node_id_range?.start ?? 1,
              id_range_end: gw.gateway?.node_id_assignment?.node_id_range?.end ?? 254,
              random_assignment: gw.gateway?.node_id_assignment?.random_id_assignment ?? false,
            }}
            @value-changed=${(e: CustomEvent) => {
              if (!gw.gateway) gw.gateway = {};
              if (!gw.gateway.node_id_assignment) gw.gateway.node_id_assignment = {};
              if (!gw.gateway.node_id_assignment.node_id_range) gw.gateway.node_id_assignment.node_id_range = {};
              gw.gateway.node_id_assignment.enabled = e.detail.value.id_assignment_enabled;
              gw.gateway.node_id_assignment.node_id_range.start = e.detail.value.id_range_start;
              gw.gateway.node_id_assignment.node_id_range.end = e.detail.value.id_range_end;
              gw.gateway.node_id_assignment.random_id_assignment = e.detail.value.random_assignment;
              this._gateways = { ...this._gateways, [name]: { ...gw } };
            }}
          ></ms-form>

          <div class="section-title">Availability &amp; Heartbeat</div>
          <ms-form
            .schema=${this._availSchema()}
            .data=${{
              availability_window: this._durationToString(gw.gateway?.availability_window),
              heartbeat_request_period: this._durationToString(gw.gateway?.heartbeat_request_period),
            }}
            @value-changed=${(e: CustomEvent) => {
              if (!gw.gateway) gw.gateway = {};
              gw.gateway.availability_window = this._parseDuration(e.detail.value.availability_window);
              gw.gateway.heartbeat_request_period = this._parseDuration(e.detail.value.heartbeat_request_period);
              this._gateways = { ...this._gateways, [name]: { ...gw } };
            }}
          ></ms-form>

          <div class="section-title">TCP Service</div>
          <ms-form
            .schema=${this._tcpSchema()}
            .data=${{
              tcp_enabled: gw.tcp_service?.enabled || false,
              tcp_port: gw.tcp_service?.port || 0,
            }}
            @value-changed=${(e: CustomEvent) => {
              if (!gw.tcp_service) gw.tcp_service = {};
              gw.tcp_service.enabled = e.detail.value.tcp_enabled;
              gw.tcp_service.port = e.detail.value.tcp_port;
              this._gateways = { ...this._gateways, [name]: { ...gw } };
            }}
          ></ms-form>
        </div>
        <div class="card-actions">
          <ms-button @click=${() => this._saveGateway(name)}>Save</ms-button>
          <ms-button variant="danger" appearance="outlined" @click=${() => this._deleteGateway(name)}>Delete</ms-button>
        </div>
      </ms-card>
    `;
  }

  private _transportSchema(): FormSchema[] {
    return [
      { name: "transport", label: "Transport", type: "select", options: [
        { value: "ethernet", label: "Ethernet" },
        { value: "rs485", label: "RS485" },
      ]},
    ];
  }

  private _ethernetSchema(): FormSchema[] {
    return [
      { name: "host", label: "Host", type: "string" },
      { name: "port", label: "Port", type: "integer" },
    ];
  }

  private _rs485Schema(): FormSchema[] {
    return [
      { name: "device", label: "Serial Device", type: "string", hint: "e.g. /dev/ttyUSB0" },
    ];
  }

  private _nodeIdSchema(): FormSchema[] {
    return [
      { name: "id_assignment_enabled", label: "Enabled", type: "boolean" },
      { name: "id_range_start", label: "Range Start", type: "integer" },
      { name: "id_range_end", label: "Range End", type: "integer" },
      { name: "random_assignment", label: "Random Assignment", type: "boolean" },
    ];
  }

  private _availSchema(): FormSchema[] {
    return [
      { name: "availability_window", label: "Availability Window", type: "string", hint: "How long to consider a node online after receiving a message (e.g. 30s, 5m). Nodes go offline in HA if no messages received within this window." },
      { name: "heartbeat_request_period", label: "Heartbeat Request Period", type: "string", hint: "How often to request heartbeat from gateway (e.g. 10s). Set to 0s to disable. Gateway responds with I_HEARTBEAT_RESPONSE to prove it's alive." },
    ];
  }

  private _tcpSchema(): FormSchema[] {
    return [
      { name: "tcp_enabled", label: "Enabled", type: "boolean" },
      { name: "tcp_port", label: "Port", type: "integer" },
    ];
  }

  // Go marshals time.Duration as nanoseconds in JSON
  private _durationToString(ns: number | undefined): string {
    if (!ns) return "0s";
    const seconds = ns / 1e9;
    if (seconds >= 60 && seconds % 60 === 0) return `${seconds / 60}m`;
    return `${seconds}s`;
  }

  private _parseDuration(s: string): number {
    if (!s) return 0;
    const match = s.match(/^(\d+(?:\.\d+)?)\s*(s|m|ms|h)?$/);
    if (!match) return 0;
    const val = parseFloat(match[1]);
    switch (match[2]) {
      case "h": return val * 3.6e12;
      case "m": return val * 6e10;
      case "ms": return val * 1e6;
      case "s": default: return val * 1e9;
    }
  }

  private async _saveGateway(name: string) {
    try {
      await api.putGateway(name, this._gateways[name]);
      this._toast("success", `Gateway '${name}' saved`);
      await this._load();
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _deleteGateway(name: string) {
    try {
      await api.deleteGateway(name);
      this._toast("success", `Gateway '${name}' deleted`);
      await this._load();
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private _addGateway() {
    this._editingName = "";
    this._editingGateway = {
      gateway: {
        node_id_assignment: {
          enabled: true,
          random_id_assignment: true,
          node_id_range: {
            start: 101,
            end: 199
          }
        }
      },
      tcp_service: {
        enabled: true,
        port: 5003
      }
    };
    this._showDialog = true;
  }

  private _renderDialog() {
    if (!this._showDialog) return nothing;
    return html`
      <ms-dialog .open=${true} headerTitle="Add Gateway" @closed=${() => this._showDialog = false}>
        <div class="dialog-field">
          <label>Gateway Name</label>
          <input .value=${this._editingName} @input=${(e: Event) => this._editingName = (e.target as HTMLInputElement).value}>
        </div>
        <p style="font-size:var(--ha-font-size-s);color:var(--secondary-text-color)">
          A gateway will be created with default settings (ethernet transport port 5003, TCP service enabled on port 5003, random node ID assignment enabled for range 101-199).
          You can configure all settings after creation.
        </p>
        <ms-button slot="footer" variant="neutral" appearance="plain" @click=${() => this._showDialog = false}>Cancel</ms-button>
        <ms-button slot="footer" @click=${this._saveNewGateway}>Create</ms-button>
      </ms-dialog>
    `;
  }

  private async _saveNewGateway() {
    if (!this._editingName) {
      this._toast("error", "Gateway name required");
      return;
    }
    try {
      // Create with minimal defaults; SetDefaults in Go will fill the rest
      await api.putGateway(this._editingName, this._editingGateway);
      this._showDialog = false;
      await this._load();
      this._toast("success", `Gateway '${this._editingName}' created`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private _toast(type: "success" | "error", msg: string) {
    const toast = document.querySelector("ms-toast") as any;
    toast?.show(type, msg);
  }
}
