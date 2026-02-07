import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";
import type { FormSchema } from "../components/ms-form";

@customElement("ms-view-mqtt")
export class MsViewMqtt extends LitElement {
  @state() private _mqtt: any = {};
  @state() private _adapter: any = {};
  @state() private _logLevel = "info";
  @state() private _loading = true;
  @state() private _mqttConnected = false;

  static styles = css`
    :host { display: block; }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
      gap: var(--ha-space-4, 16px);
    }
    .loading { text-align: center; padding: var(--ha-space-8); color: var(--secondary-text-color); }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
    document.addEventListener("ws-connection_changed", this._onConn as EventListener);
    document.addEventListener("ws-initial_state", this._onInit as EventListener);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("ws-connection_changed", this._onConn as EventListener);
    document.removeEventListener("ws-initial_state", this._onInit as EventListener);
  }

  private _onConn = (e: Event) => {
    const d = (e as CustomEvent).detail;
    if (d?.type === "mqtt") this._mqttConnected = d.connected;
  };

  private _onInit = (e: Event) => {
    const d = (e as CustomEvent).detail;
    if (d?.mqtt) this._mqttConnected = d.mqtt.connected;
  };

  private async _load() {
    try {
      const [mqtt, adapter, ll, status] = await Promise.all([
        api.getMQTT(),
        api.getAdapter(),
        api.getLogLevel(),
        api.getStatus(),
      ]);
      this._mqtt = mqtt;
      this._adapter = adapter;
      this._logLevel = ll.log_level;
      this._mqttConnected = status?.mqtt?.connected || false;
    } catch (e) { /* ignore */ }
    this._loading = false;
  }

  private _mqttSchema: FormSchema[] = [
    { name: "broker", label: "Broker", type: "string", required: true },
    { name: "port", label: "Port", type: "integer", required: true },
    { name: "username", label: "Username", type: "string" },
    { name: "password", label: "Password", type: "password" },
    { name: "client_id", label: "Client ID", type: "string" },
  ];

  private _adapterSchema: FormSchema[] = [
    { name: "topic_prefix", label: "Topic Prefix", type: "string" },
    { name: "homeassistant_discovery", label: "HA Discovery", type: "boolean" },
  ];

  render() {
    if (this._loading) return html`<div class="loading">Loading...</div>`;

    return html`
      <div class="cards">
        <ms-card header="Adapter Settings">
          <div class="card-content">
            <ms-form
              .schema=${this._adapterSchema}
              .data=${{
                topic_prefix: this._adapter.topic_prefix || "",
                homeassistant_discovery: this._adapter.homeassistant_discovery ?? true,
              }}
              @value-changed=${(e: CustomEvent) => {
                this._adapter = { ...this._adapter, ...e.detail.value };
              }}
            ></ms-form>
            <ms-form
              .schema=${[{ name: "log_level", label: "Log Level", type: "select", options: [
                { value: "debug", label: "Debug" },
                { value: "info", label: "Info" },
                { value: "warn", label: "Warn" },
                { value: "error", label: "Error" },
              ]} as FormSchema]}
              .data=${{ log_level: this._logLevel }}
              @value-changed=${(e: CustomEvent) => this._logLevel = e.detail.value.log_level}
            ></ms-form>
          </div>
          <div class="card-actions">
            <ms-button @click=${this._saveAdapter}>Save</ms-button>
          </div>
        </ms-card>

        <ms-card>
          <div style="display:flex;align-items:center;gap:8px;padding:12px 16px 0">
            <h2 style="margin:0;font-size:20px;font-weight:400">MQTT</h2>
            <ms-status-dot .status=${this._mqttConnected ? "online" : "offline"}></ms-status-dot>
          </div>
          <div class="card-content">
            <ms-form
              .schema=${this._mqttSchema}
              .data=${this._mqtt}
              @value-changed=${(e: CustomEvent) => {
                this._mqtt = { ...e.detail.value };
              }}
            ></ms-form>
          </div>
          <div class="card-actions">
            <ms-button @click=${this._saveMQTT}>Save</ms-button>
          </div>
        </ms-card>
      </div>
    `;
  }

  private async _saveMQTT() {
    try {
      await api.putMQTT(this._mqtt);
      this._toast("success", "MQTT settings saved");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _saveAdapter() {
    try {
      await Promise.all([
        api.putAdapter(this._adapter),
        api.putLogLevel(this._logLevel),
      ]);
      this._toast("success", "Adapter settings saved");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private _toast(type: "success" | "error", msg: string) {
    const toast = document.querySelector("ms-toast") as any;
    toast?.show(type, msg);
  }
}
