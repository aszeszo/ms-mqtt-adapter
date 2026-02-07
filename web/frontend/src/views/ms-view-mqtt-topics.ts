import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";

@customElement("ms-view-mqtt-topics")
export class MsViewMqttTopics extends LitElement {
  @state() private _topics: any = null;
  @state() private _loading = true;
  @state() private _filter = "all";
  @state() private _showConfirm = false;
  @state() private _deleteScope = "";
  @state() private _deleteDevice = "";
  @state() private _deleteEntity = "";
  @state() private _deleteMessage = "";

  static styles = css`
    :host { display: block; }
    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: var(--ha-space-4, 16px);
      flex-wrap: wrap;
      gap: var(--ha-space-3, 12px);
    }
    .filter {
      display: flex;
      gap: var(--ha-space-2, 8px);
      flex-wrap: wrap;
    }
    .section {
      margin-bottom: var(--ha-space-6, 24px);
    }
    .section-title {
      font-size: var(--ha-font-size-l, 16px);
      font-weight: var(--ha-font-weight-medium, 500);
      margin-bottom: var(--ha-space-3, 12px);
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: var(--ha-space-2, 8px);
    }
    .topic-list {
      display: flex;
      flex-direction: column;
      gap: var(--ha-space-2, 8px);
    }
    .topic-item {
      display: grid;
      grid-template-columns: auto 1fr auto auto auto;
      gap: var(--ha-space-2, 8px);
      align-items: center;
      padding: var(--ha-space-2, 8px) var(--ha-space-3, 12px);
      background: var(--card-background-color);
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-sm);
      font-family: var(--ha-font-family-code, monospace);
      font-size: var(--ha-font-size-s, 12px);
    }
    .topic-type {
      padding: 2px 8px;
      border-radius: var(--ha-border-radius-pill);
      font-size: var(--ha-font-size-xs, 11px);
      text-transform: uppercase;
      font-weight: var(--ha-font-weight-medium, 500);
      white-space: nowrap;
    }
    .topic-type.state { background: var(--info-color); color: white; }
    .topic-type.command { background: var(--warning-color); color: white; }
    .topic-type.availability { background: var(--success-color); color: white; }
    .topic-type.discovery { background: var(--primary-color); color: white; }
    .topic-type.gateway { background: var(--accent-color); color: white; }
    .topic-path {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      min-width: 0;
    }
    .topic-meta {
      font-size: var(--ha-font-size-xs, 11px);
      color: var(--secondary-text-color);
      white-space: nowrap;
    }
    .retained-badge {
      padding: 2px 6px;
      background: var(--divider-color);
      border-radius: var(--ha-border-radius-pill);
      font-size: var(--ha-font-size-xs, 10px);
      white-space: nowrap;
    }
    .topic-actions {
      display: flex;
      gap: var(--ha-space-1, 4px);
    }
    .topic-actions button {
      padding: 2px 8px;
      font-size: var(--ha-font-size-xs, 11px);
      background: none;
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-sm);
      cursor: pointer;
      color: var(--error-color);
    }
    .topic-actions button:hover {
      background: var(--error-color);
      color: white;
    }
    .danger-zone {
      margin-top: var(--ha-space-8, 32px);
      padding: var(--ha-space-4, 16px);
      border: 2px solid var(--error-color);
      border-radius: var(--ha-border-radius-m);
      background: color-mix(in srgb, var(--error-color) 10%, transparent);
    }
    .danger-title {
      font-size: var(--ha-font-size-l, 16px);
      font-weight: var(--ha-font-weight-bold, 700);
      color: var(--error-color);
      margin-bottom: var(--ha-space-2, 8px);
    }
    .danger-actions {
      display: flex;
      gap: var(--ha-space-2, 8px);
      margin-top: var(--ha-space-3, 12px);
      flex-wrap: wrap;
    }
    .device-group {
      margin-bottom: var(--ha-space-4, 16px);
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-m);
      overflow: hidden;
    }
    .device-header {
      padding: var(--ha-space-2, 8px) var(--ha-space-3, 12px);
      background: var(--secondary-background-color);
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-weight: var(--ha-font-weight-medium, 500);
    }
    .device-topics {
      padding: var(--ha-space-2, 8px);
    }
    .loading {
      text-align: center;
      padding: var(--ha-space-8, 32px);
      color: var(--secondary-text-color);
    }
    .no-mqtt {
      text-align: center;
      padding: var(--ha-space-8, 32px);
      color: var(--secondary-text-color);
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
  }

  private async _load() {
    try {
      this._topics = await api.getMQTTTopics();
    } catch (e) {
      this._toast("error", "Failed to load topics");
    }
    this._loading = false;
  }

  render() {
    if (this._loading) return html`<div class="loading">Loading...</div>`;
    if (!this._topics) return html`<div class="no-mqtt">MQTT client not available</div>`;

    return html`
      <div class="toolbar">
        <h2>MQTT Topics</h2>
        <div class="filter">
          <ms-button appearance="${this._filter === 'all' ? 'filled' : 'outlined'}" variant="neutral" @click=${() => this._filter = "all"}>All</ms-button>
          <ms-button appearance="${this._filter === 'adapter' ? 'filled' : 'outlined'}" variant="neutral" @click=${() => this._filter = "adapter"}>Adapter</ms-button>
          <ms-button appearance="${this._filter === 'discovery' ? 'filled' : 'outlined'}" variant="neutral" @click=${() => this._filter = "discovery"}>Discovery</ms-button>
          <ms-button appearance="${this._filter === 'gateway' ? 'filled' : 'outlined'}" variant="neutral" @click=${() => this._filter = "gateway"}>Gateway</ms-button>
          <ms-button appearance="outlined" variant="neutral" @click=${this._load}>Refresh</ms-button>
        </div>
      </div>

      ${this._filter === "all" || this._filter === "adapter" ? this._renderAdapterSection() : nothing}
      ${this._filter === "all" || this._filter === "discovery" ? this._renderDiscoverySection() : nothing}
      ${this._filter === "all" || this._filter === "gateway" ? this._renderGatewaySection() : nothing}

      <div class="danger-zone">
        <div class="danger-title">⚠️ Danger Zone</div>
        <p>Clearing topics will remove them from the MQTT broker. Home Assistant entities will become unavailable until republished.</p>
        <div class="danger-actions">
          <ms-button variant="danger" @click=${() => this._confirmDelete("all", "", "", "Clear ALL topics? This will remove all adapter, discovery, and gateway topics.")}>
            Clear All Topics
          </ms-button>
        </div>
      </div>

      ${this._renderConfirmDialog()}
    `;
  }

  private _renderAdapterSection() {
    const topics = this._topics.adapter_topics || [];
    const byDevice = this._groupByDevice(topics);

    return html`
      <div class="section">
        <div class="section-title">
          <span>Adapter Topics (${topics.length})</span>
          <ms-button variant="danger" appearance="outlined" size="small"
            @click=${() => this._confirmDelete("adapter", "", "", "Clear all adapter topics (state, command, availability)?")}>
            Clear All Adapter Topics
          </ms-button>
        </div>
        ${Object.entries(byDevice).map(([deviceId, deviceTopics]) => this._renderDeviceGroup(deviceId, deviceTopics as any[]))}
      </div>
    `;
  }

  private _renderDiscoverySection() {
    const topics = this._topics.discovery_topics || [];
    const byDevice = this._groupByDevice(topics);

    return html`
      <div class="section">
        <div class="section-title">
          <span>Home Assistant Discovery (${topics.length})</span>
          <ms-button variant="danger" appearance="outlined" size="small"
            @click=${() => this._confirmDelete("discovery", "", "", "Clear all Home Assistant discovery topics? Entities will disappear from HA until republished.")}>
            Clear All Discovery Topics
          </ms-button>
        </div>
        ${Object.entries(byDevice).map(([deviceId, deviceTopics]) => this._renderDeviceGroup(deviceId, deviceTopics as any[]))}
      </div>
    `;
  }

  private _renderGatewaySection() {
    const topics = this._topics.gateway_topics || [];

    return html`
      <div class="section">
        <div class="section-title">
          <span>Gateway Topics (${topics.length})</span>
        </div>
        <div class="topic-list">
          ${topics.map((t: any) => this._renderTopic(t, false))}
        </div>
      </div>
    `;
  }

  private _groupByDevice(topics: any[]): Record<string, any[]> {
    const grouped: Record<string, any[]> = {};
    for (const topic of topics) {
      const deviceId = topic.device_id || "unknown";
      if (!grouped[deviceId]) grouped[deviceId] = [];
      grouped[deviceId].push(topic);
    }
    return grouped;
  }

  private _renderDeviceGroup(deviceId: string, topics: any[]) {
    return html`
      <div class="device-group">
        <div class="device-header">
          <span>Device: ${deviceId}</span>
          <ms-button variant="danger" appearance="outlined" size="small"
            @click=${() => this._confirmDelete("device", deviceId, "", `Clear all topics for device "${deviceId}"?`)}>
            Clear Device Topics
          </ms-button>
        </div>
        <div class="device-topics">
          ${this._groupByEntity(topics)}
        </div>
      </div>
    `;
  }

  private _groupByEntity(topics: any[]) {
    const byEntity: Record<string, any[]> = {};
    for (const topic of topics) {
      const entityId = topic.entity_id || "unknown";
      if (!byEntity[entityId]) byEntity[entityId] = [];
      byEntity[entityId].push(topic);
    }

    return Object.entries(byEntity).map(([entityId, entityTopics]) => html`
      <div style="margin-bottom: var(--ha-space-3, 12px);">
        <div style="font-size: var(--ha-font-size-s, 12px); color: var(--secondary-text-color); margin-bottom: var(--ha-space-1, 4px); display: flex; justify-content: space-between; align-items: center;">
          <span>Entity: ${entityId}</span>
          <ms-button variant="danger" appearance="outlined" size="small"
            @click=${() => this._confirmDelete("entity", entityTopics[0].device_id, entityId, `Clear topics for entity "${entityId}"?`)}>
            Clear Entity
          </ms-button>
        </div>
        <div class="topic-list">
          ${entityTopics.map((t: any) => this._renderTopic(t, true))}
        </div>
      </div>
    `);
  }

  private _renderTopic(topic: any, showActions: boolean) {
    return html`
      <div class="topic-item">
        <span class="topic-type ${topic.type}">${topic.type}</span>
        <span class="topic-path" title="${topic.topic}">${topic.topic}</span>
        <span class="topic-meta">
          ${topic.gateway ? `GW: ${topic.gateway}` : ""}
        </span>
        ${topic.retained ? html`<span class="retained-badge">retained</span>` : html`<span></span>`}
        ${showActions && topic.retained ? html`
          <div class="topic-actions">
            <button @click=${() => this._confirmDelete("entity", topic.device_id, topic.entity_id, `Clear this entity's topics?`)} title="Clear this entity">
              Clear
            </button>
          </div>
        ` : html`<span></span>`}
      </div>
    `;
  }

  private _confirmDelete(scope: string, deviceId: string, entityId: string, message: string) {
    this._deleteScope = scope;
    this._deleteDevice = deviceId;
    this._deleteEntity = entityId;
    this._deleteMessage = message;
    this._showConfirm = true;
  }

  private async _executeDelete() {
    try {
      await api.deleteMQTTTopics(this._deleteScope, this._deleteDevice, this._deleteEntity);
      this._showConfirm = false;
      await this._load();
      this._toast("success", "Topics cleared successfully");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private _renderConfirmDialog() {
    if (!this._showConfirm) return nothing;
    return html`
      <ms-dialog .open=${true} headerTitle="Confirm Clear Topics" @closed=${() => this._showConfirm = false}>
        <p>${this._deleteMessage}</p>
        <p style="margin-top: var(--ha-space-3, 12px); font-size: var(--ha-font-size-s, 12px); color: var(--secondary-text-color);">
          <strong>Scope:</strong> ${this._deleteScope}
          ${this._deleteDevice ? html`<br><strong>Device:</strong> ${this._deleteDevice}` : nothing}
          ${this._deleteEntity ? html`<br><strong>Entity:</strong> ${this._deleteEntity}` : nothing}
        </p>
        <ms-button slot="footer" variant="neutral" appearance="plain" @click=${() => this._showConfirm = false}>Cancel</ms-button>
        <ms-button slot="footer" variant="danger" @click=${this._executeDelete}>Clear Topics</ms-button>
      </ms-dialog>
    `;
  }

  private _toast(type: "success" | "error", msg: string) {
    const toast = document.querySelector("ms-toast") as any;
    toast?.show(type, msg);
  }
}
