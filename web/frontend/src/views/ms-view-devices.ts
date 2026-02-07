import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";

const ENTITY_TYPES = [
  "switch","light","dimmer","cover","text","number","select",
  "binary_sensor","sensor","temperature","humidity","battery",
  "voltage","current","pressure","weight","distance","light_level",
  "watt","kwh","flow","volume","ph","orp","ec","var","va",
  "power_factor","custom","position","uv","rain","rainrate",
  "wind","gust","direction","impedance","climate","rgb_light","rgbw_light",
];

@customElement("ms-view-devices")
export class MsViewDevices extends LitElement {
  @state() private _devices: any[] = [];
  @state() private _entities: Record<string, string> = {};
  @state() private _aliases: Record<string, number> = {};
  @state() private _loading = true;
  @state() private _expanded: string | null = null;

  // Dialog state
  @state() private _showDeviceDialog = false;
  @state() private _editingDevice: any = null;
  @state() private _isNewDevice = false;
  @state() private _showEntityDialog = false;
  @state() private _editingEntity: any = null;
  @state() private _editingEntityDeviceId = "";
  @state() private _isNewEntity = false;
  @state() private _showConfirmDialog = false;
  @state() private _confirmAction: (() => void) | null = null;
  @state() private _confirmMessage = "";

  static styles = css`
    :host { display: block; }
    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: var(--ha-space-4, 16px);
    }
    .quick-add {
      display: flex;
      gap: var(--ha-space-2, 8px);
      align-items: center;
    }
    .quick-add-label {
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
      font-weight: var(--ha-font-weight-medium, 500);
      text-transform: uppercase;
    }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
      gap: var(--ha-space-4, 16px);
    }
    .device-header {
      display: flex; justify-content: space-between; align-items: center;
    }
    .badge {
      background: var(--primary-color);
      color: white;
      border-radius: var(--ha-border-radius-pill);
      padding: 2px 8px;
      font-size: var(--ha-font-size-s, 12px);
    }
    .meta { font-size: var(--ha-font-size-s, 12px); color: var(--secondary-text-color); margin-top: 4px; }
    .entity-list {
      margin-top: var(--ha-space-3, 12px);
      border-top: 1px solid var(--divider-color);
      padding-top: var(--ha-space-2, 8px);
    }
    .entity-row {
      display: flex; justify-content: space-between; align-items: center;
      padding: var(--ha-space-1, 4px) 0;
      font-size: var(--ha-font-size-m, 14px);
    }
    .entity-name { display: flex; align-items: center; gap: var(--ha-space-2, 8px); }
    .entity-actions { display: flex; gap: var(--ha-space-1, 4px); }
    .entity-actions button {
      background: none; border: none; cursor: pointer;
      color: var(--secondary-text-color); font-size: 12px; padding: 4px 6px;
    }
    .entity-actions button:hover { color: var(--primary-text-color); }
    .entity-state {
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
      font-family: var(--ha-font-family-code, monospace);
    }
    .toggle-btn {
      background: none; border: none; cursor: pointer;
      color: var(--primary-color); font-size: var(--ha-font-size-s); padding: 0;
    }
    .dialog-field { margin-bottom: var(--ha-space-3, 12px); }
    .dialog-field label {
      display: block; font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color); margin-bottom: 4px;
    }
    .dialog-field input, .dialog-field select {
      width: 100%; box-sizing: border-box;
      padding: var(--ha-space-2) var(--ha-space-3);
      border: 1px solid var(--divider-color); border-radius: var(--ha-border-radius-sm);
      font-size: var(--ha-font-size-m); font-family: inherit;
      background: var(--card-background-color); color: var(--primary-text-color);
    }
    .dialog-field .hint {
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
      margin-top: 2px;
    }
    .dialog-section {
      font-size: var(--ha-font-size-s, 12px);
      font-weight: var(--ha-font-weight-medium, 500);
      color: var(--secondary-text-color);
      text-transform: uppercase;
      margin: var(--ha-space-4, 16px) 0 var(--ha-space-2, 8px);
      padding-top: var(--ha-space-2, 8px);
      border-top: 1px solid var(--divider-color);
    }
    .dialog-body { max-height: 60vh; overflow-y: auto; }
    .loading { text-align: center; padding: var(--ha-space-8, 32px); color: var(--secondary-text-color); }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
    document.addEventListener("ws-entity_state_changed", this._onEntityState as EventListener);
    document.addEventListener("ws-initial_state", this._onInitialState as EventListener);
    document.addEventListener("ws-config_reloaded", (() => this._load()) as EventListener);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    document.removeEventListener("ws-entity_state_changed", this._onEntityState as EventListener);
    document.removeEventListener("ws-initial_state", this._onInitialState as EventListener);
    document.removeEventListener("ws-config_reloaded", (() => this._load()) as EventListener);
  }

  private _onEntityState = (e: Event) => {
    const d = (e as CustomEvent).detail;
    if (d?.unique_id) {
      this._entities = { ...this._entities, [d.unique_id]: d.state };
    }
  };

  private _onInitialState = (e: Event) => {
    const d = (e as CustomEvent).detail;
    if (d?.entities) this._entities = d.entities;
  };

  private async _load() {
    try {
      const [devices, status, aliases] = await Promise.all([
        api.getDevices(),
        api.getEntityStates(),
        api.getAliases().catch(() => ({})),
      ]);
      this._devices = devices;
      this._entities = status;
      this._aliases = aliases;
    } catch (e) {
      // ignore
    } finally {
      this._loading = false;
    }
  }

  private _aliasHint(): string {
    const names = Object.keys(this._aliases);
    if (names.length === 0) return "Numeric ID or alias name";
    return `Numeric ID or alias: ${names.slice(0, 5).join(", ")}${names.length > 5 ? ", ..." : ""}`;
  }

  render() {
    if (this._loading) return html`<div class="loading">Loading...</div>`;

    return html`
      <div class="toolbar">
        <div class="quick-add">
          <span class="quick-add-label">Quick Add:</span>
          <ms-button appearance="outlined" variant="neutral" @click=${this._addNippyRelay9}>
            nippy DIN Relay 9
          </ms-button>
          <ms-button appearance="outlined" variant="neutral" @click=${this._addNippyRelay3}>
            nippy DIN Relay 3
          </ms-button>
          <ms-button appearance="outlined" variant="neutral" @click=${this._addNippyInput6}>
            nippy DIN Input 6
          </ms-button>
          <ms-button appearance="outlined" variant="neutral" @click=${this._addNippyInput6LP}>
            nippy DIN Input 6 LP
          </ms-button>
          <ms-button appearance="outlined" variant="neutral" @click=${this._addNippyRS485Gateway}>
            nippy RS-485 Gateway v1
          </ms-button>
          <ms-button appearance="outlined" variant="neutral" @click=${this._addNippyRelay9AutoSync}>
            nippy DIN Relay 9 (auto-sync)
          </ms-button>
          <ms-button appearance="outlined" variant="neutral" @click=${this._addNippyRelay3AutoSync}>
            nippy DIN Relay 3 (auto-sync)
          </ms-button>
        </div>
        <ms-button @click=${this._addDevice}>Add Device</ms-button>
      </div>
      <div class="cards">
        ${this._devices.map((dev) => this._renderDeviceCard(dev))}
      </div>
      ${this._renderDeviceDialog()}
      ${this._renderEntityDialog()}
      ${this._renderConfirmDialog()}
    `;
  }

  private _renderDeviceCard(dev: any) {
    const expanded = this._expanded === dev.id;
    return html`
      <ms-card>
        <div class="card-content">
          <div class="device-header">
            <div>
              <strong>${dev.name}</strong>
              <span class="badge">${dev.entities?.length || 0}</span>
            </div>
            <div class="entity-actions">
              <button @click=${() => this._editDevice(dev)} title="Edit">Edit</button>
              <button @click=${() => this._confirmDelete("Delete device '" + dev.name + "'?", () => this._deleteDevice(dev))} title="Delete">Del</button>
            </div>
          </div>
          <div class="meta">
            ${dev.manufacturer || ""} ${dev.model || ""}${dev.gateway ? ` | GW: ${dev.gateway}` : ""} | Node: ${dev.node_id ?? "per-entity"} | ID: ${dev.id}
          </div>
          <button class="toggle-btn" @click=${() => this._expanded = expanded ? null : dev.id}>
            ${expanded ? "Hide entities" : "Show entities"}
          </button>
          ${expanded ? this._renderEntityList(dev) : nothing}
        </div>
      </ms-card>
    `;
  }

  private _renderEntityList(dev: any) {
    return html`
      <div class="entity-list">
        ${(dev.entities || []).map((e: any) => {
          const uid = e.unique_id || `${dev.id}_${e.id}`;
          const state = this._entities[uid] ?? "";
          return html`
            <div class="entity-row">
              <span class="entity-name">
                ${e.name}
                <span class="entity-state">${state}</span>
              </span>
              <span class="entity-actions">
                <button @click=${() => this._editEntity(dev.id, e)}>Edit</button>
                <button @click=${() => this._confirmDelete("Delete entity '" + e.name + "'?", () => this._deleteEntity(dev.id, e.id))}>Del</button>
              </span>
            </div>
          `;
        })}
        <ms-button appearance="outlined" variant="neutral" @click=${() => this._addEntity(dev.id)} style="margin-top:8px">
          Add Entity
        </ms-button>
      </div>
    `;
  }

  // --- Device dialog ---
  private _addDevice() {
    this._editingDevice = { name: "", id: "", node_id: null, gateway: "", manufacturer: "", model: "", sw_version: "", hw_version: "", configuration_url: "", suggested_area: "", via_device: "", entities: [] };
    this._isNewDevice = true;
    this._showDeviceDialog = true;
  }

  private _editDevice(dev: any) {
    this._editingDevice = { ...dev };
    this._isNewDevice = false;
    this._showDeviceDialog = true;
  }

  private _generateRandomSuffix(): string {
    const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
    let result = "";
    for (let i = 0; i < 4; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  }

  private async _addNippyRelay9() {
    const suffix = this._generateRandomSuffix();
    const suggestedName = `nippy_relay9_${suffix}`;
    const name = prompt("Enter device name:", suggestedName);
    if (!name) return;

    await this._ensureRelay9Aliases();
    const deviceId = name;
    const nodeId = this._getRandomAvailableNodeId();
    const device = {
      name: name,
      id: deviceId,
      node_id: nodeId,
      manufacturer: "nippy",
      model: "DIN Relay 9",
      sw_version: "1.0",
      hw_version: "1.0",
      entities: [
        { name: "Ambient temperature", id: "ambient_temperature", child_id: 9, entity_type: "temperature", read_only: true, entity_category: "diagnostic", device_class: "temperature", icon: "mdi:thermometer" },
        { name: "P2P Toggle", id: "p2p_toggle", child_id: 10, entity_type: "switch", initial_value: "0", icon: "mdi:toggle-switch", sync_period: 30000000000 },
        ...Array.from({ length: 9 }, (_, i) => ({
          name: `Relay ${i + 1}`,
          id: `relay_${i + 1}`,
          child_id: `relay9_ch${i}`,
          entity_type: "switch",
          initial_value: "0",
          icon: "hue:socket-eu"
        }))
      ]
    };
    try {
      const devices = [...this._devices, device];
      await api.putDevices(devices);
      await this._load();
      this._toast("success", `Added ${name} device`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addNippyRelay3() {
    const suffix = this._generateRandomSuffix();
    const suggestedName = `nippy_relay3_${suffix}`;
    const name = prompt("Enter device name:", suggestedName);
    if (!name) return;

    await this._ensureRelay3Aliases();
    const deviceId = name;
    const nodeId = this._getRandomAvailableNodeId();
    const device = {
      name: name,
      id: deviceId,
      node_id: nodeId,
      manufacturer: "nippy",
      model: "DIN Relay 3",
      sw_version: "1.0",
      hw_version: "1.0",
      entities: [
        ...Array.from({ length: 3 }, (_, i) => ({
          name: `Relay ${i + 1}`,
          id: `relay_${i + 1}`,
          child_id: `relay3_ch${i}`,
          entity_type: "switch",
          initial_value: "0",
          icon: "hue:socket-eu"
        })),
        { name: "Ambient temperature", id: "ambient_temperature", child_id: 3, entity_type: "temperature", read_only: true, entity_category: "diagnostic", device_class: "temperature", icon: "mdi:thermometer" },
        { name: "P2P Toggle", id: "p2p_toggle", child_id: 4, entity_type: "switch", initial_value: "0", icon: "mdi:toggle-switch" },
      ]
    };
    try {
      const devices = [...this._devices, device];
      await api.putDevices(devices);
      await this._load();
      this._toast("success", `Added ${name} device`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addNippyRelay9AutoSync() {
    const suffix = this._generateRandomSuffix();
    const suggestedName = `nippy_relay9_${suffix}`;
    const name = prompt("Enter device name:", suggestedName);
    if (!name) return;

    await this._ensureRelay9Aliases();
    const deviceId = name;
    const nodeId = this._getRandomAvailableNodeId();
    const device = {
      name: name,
      id: deviceId,
      node_id: nodeId,
      manufacturer: "nippy",
      model: "DIN Relay 9",
      sw_version: "1.0",
      hw_version: "1.0",
      entities: [
        { name: "Ambient temperature", id: "ambient_temperature", child_id: 9, entity_type: "temperature", read_only: true, entity_category: "diagnostic", device_class: "temperature", icon: "mdi:thermometer" },
        { name: "P2P Toggle", id: "p2p_toggle", child_id: 10, entity_type: "switch", initial_value: "0", icon: "mdi:toggle-switch" },
        ...Array.from({ length: 9 }, (_, i) => ({
          name: `Relay ${i + 1}`,
          id: `relay_${i + 1}`,
          child_id: `relay9_ch${i}`,
          entity_type: "switch",
          initial_value: "0",
          icon: "hue:socket-eu",
          sync_period: 30 * 1e9  // 30 seconds
        }))
      ]
    };
    try {
      const devices = [...this._devices, device];
      await api.putDevices(devices);
      await this._load();
      this._toast("success", `Added ${name} device with auto-sync`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addNippyRelay3AutoSync() {
    const suffix = this._generateRandomSuffix();
    const suggestedName = `nippy_relay3_${suffix}`;
    const name = prompt("Enter device name:", suggestedName);
    if (!name) return;

    await this._ensureRelay3Aliases();
    const deviceId = name;
    const nodeId = this._getRandomAvailableNodeId();
    const device = {
      name: name,
      id: deviceId,
      node_id: nodeId,
      manufacturer: "nippy",
      model: "DIN Relay 3",
      sw_version: "1.0",
      hw_version: "1.0",
      entities: [
        ...Array.from({ length: 3 }, (_, i) => ({
          name: `Relay ${i + 1}`,
          id: `relay_${i + 1}`,
          child_id: `relay3_ch${i}`,
          entity_type: "switch",
          initial_value: "0",
          icon: "hue:socket-eu",
          sync_period: 30 * 1e9  // 30 seconds
        })),
        { name: "Ambient temperature", id: "ambient_temperature", child_id: 3, entity_type: "temperature", read_only: true, entity_category: "diagnostic", device_class: "temperature", icon: "mdi:thermometer" },
        { name: "P2P Toggle", id: "p2p_toggle", child_id: 4, entity_type: "switch", initial_value: "0", icon: "mdi:toggle-switch" },
      ]
    };
    try {
      const devices = [...this._devices, device];
      await api.putDevices(devices);
      await this._load();
      this._toast("success", `Added ${name} device with auto-sync`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addNippyInput6() {
    const suffix = this._generateRandomSuffix();
    const suggestedName = `nippy_input6_${suffix}`;
    const name = prompt("Enter device name:", suggestedName);
    if (!name) return;

    const deviceId = name;
    const nodeId = this._getRandomAvailableNodeId();
    const device = {
      name: name,
      id: deviceId,
      node_id: nodeId,
      manufacturer: "nippy",
      model: "DIN Input 6",
      sw_version: "1.9.36",
      hw_version: "1.0",
      entities: [
        // Regular inputs (child 0-5)
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `Input ${letter}`,
          id: `input_${letter.toLowerCase()}`,
          child_id: idx,
          entity_type: "binary_sensor",
          read_only: true,
          icon: "hue:friends-of-hue-senic"
        })),
        // Target configs (child 6-11) - internal
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `${letter} Target`,
          id: `${letter.toLowerCase()}_target`,
          child_id: 6 + idx,
          entity_type: "text",
          initial_value: "0",
          entity_category: "config",
          object_id: "",
          icon: "mdi:target"
        })),
        // Target child configs (child 12-17) - internal
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `${letter} Target child`,
          id: `${letter.toLowerCase()}_target_child`,
          child_id: 12 + idx,
          entity_type: "text",
          initial_value: "0",
          entity_category: "config",
          object_id: "",
          icon: "mdi:target-variant"
        })),
        // Ambient temperature (child 37)
        { name: "Ambient temperature", id: "ambient_temperature", child_id: 37, entity_type: "temperature", read_only: true, entity_category: "diagnostic", device_class: "temperature", icon: "mdi:thermometer" },
        // MSG Interval config (child 38) - internal
        { name: "MSG Int. (0-255 ms)", id: "msg_int", child_id: 38, entity_type: "text", initial_value: "200", entity_category: "config", object_id: "", icon: "mdi:clock-edit" },
      ]
    };
    try {
      const devices = [...this._devices, device];
      await api.putDevices(devices);
      await this._load();
      this._toast("success", `Added ${name} device`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addNippyInput6LP() {
    const suffix = this._generateRandomSuffix();
    const suggestedName = `nippy_input6lp_${suffix}`;
    const name = prompt("Enter device name:", suggestedName);
    if (!name) return;

    const deviceId = name;
    const nodeId = this._getRandomAvailableNodeId();
    const device = {
      name: name,
      id: deviceId,
      node_id: nodeId,
      manufacturer: "nippy",
      model: "DIN Input 6 LP",
      sw_version: "1.9.36",
      hw_version: "1.0",
      entities: [
        // Regular inputs (child 0-5)
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `Input ${letter}`,
          id: `input_${letter.toLowerCase()}`,
          child_id: idx,
          entity_type: "binary_sensor",
          read_only: true,
          icon: "hue:friends-of-hue-senic"
        })),
        // Target configs (child 6-11) - internal
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `${letter} Target`,
          id: `${letter.toLowerCase()}_target`,
          child_id: 6 + idx,
          entity_type: "text",
          initial_value: "0",
          entity_category: "config",
          object_id: "",
          icon: "mdi:target"
        })),
        // Target child configs (child 12-17) - internal
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `${letter} Target child`,
          id: `${letter.toLowerCase()}_target_child`,
          child_id: 12 + idx,
          entity_type: "text",
          initial_value: "0",
          entity_category: "config",
          object_id: "",
          icon: "mdi:target-variant"
        })),
        // LP Target configs (child 18-23) - internal
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `${letter} LP Target`,
          id: `${letter.toLowerCase()}_lp_target`,
          child_id: 18 + idx,
          entity_type: "text",
          initial_value: "0",
          entity_category: "config",
          object_id: "",
          icon: "mdi:target-variant"
        })),
        // LP Target child configs (child 24-29) - internal
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `${letter} LP Target child`,
          id: `${letter.toLowerCase()}_lp_target_child`,
          child_id: 24 + idx,
          entity_type: "text",
          initial_value: "0",
          entity_category: "config",
          object_id: "",
          icon: "mdi:target-variant"
        })),
        // LP inputs (child 30-35)
        ...["A", "B", "C", "D", "E", "F"].map((letter, idx) => ({
          name: `Input LP ${letter}`,
          id: `input_lp_${letter.toLowerCase()}`,
          child_id: 30 + idx,
          entity_type: "binary_sensor",
          read_only: true,
          icon: "hue:friends-of-hue-senic"
        })),
        // LP Time config (child 36) - internal
        { name: "LP Time (1-5 s)", id: "lp_time", child_id: 36, entity_type: "text", initial_value: "3", entity_category: "config", object_id: "", icon: "mdi:timer" },
        // Ambient temperature (child 37)
        { name: "Ambient temperature", id: "ambient_temperature", child_id: 37, entity_type: "temperature", read_only: true, entity_category: "diagnostic", device_class: "temperature", icon: "mdi:thermometer" },
        // MSG Interval config (child 38) - internal
        { name: "MSG Int. (0-255 ms)", id: "msg_int", child_id: 38, entity_type: "text", initial_value: "200", entity_category: "config", object_id: "", icon: "mdi:clock-edit" },
      ]
    };
    try {
      const devices = [...this._devices, device];
      await api.putDevices(devices);
      await this._load();
      this._toast("success", `Added ${name} device`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addNippyRS485Gateway() {
    const suffix = this._generateRandomSuffix();
    const suggestedName = `nippy_rs485gw_${suffix}`;
    const name = prompt("Enter device name:", suggestedName);
    if (!name) return;

    const deviceId = name;
    const device = {
      name: name,
      id: deviceId,
      node_id: 0,
      manufacturer: "nippy",
      model: "RS-485 Gateway",
      sw_version: "2.3.2",
      hw_version: "1.0",
      entities: [
        { name: "Ambient temperature", id: "ambient_temperature", child_id: 1, entity_type: "temperature", read_only: true, entity_category: "diagnostic", device_class: "temperature", icon: "mdi:thermometer", availability_topic: "none" },
      ]
    };
    try {
      const devices = [...this._devices, device];
      await api.putDevices(devices);
      await this._load();
      this._toast("success", `Added ${name} device`);
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private _getRandomAvailableNodeId(): number {
    const usedIds = new Set(this._devices.map(d => d.node_id).filter(id => id !== null && id !== undefined));
    let nodeId: number;
    do {
      nodeId = Math.floor(Math.random() * 254) + 1; // 1-254, avoiding 0 and 255
    } while (usedIds.has(nodeId));
    return nodeId;
  }

  private async _ensureRelay9Aliases() {
    const relay9Aliases: Record<string, number> = {
      relay9_ch0: 0, relay9_ch1: 1, relay9_ch2: 2, relay9_ch3: 8,
      relay9_ch4: 7, relay9_ch5: 6, relay9_ch6: 5, relay9_ch7: 4, relay9_ch8: 3,
    };
    const missing = Object.keys(relay9Aliases).filter(k => !(k in this._aliases));
    if (missing.length > 0) {
      const aliases = { ...this._aliases, ...relay9Aliases };
      await api.putAliases(aliases);
      this._aliases = aliases;
    }
  }

  private async _ensureRelay3Aliases() {
    const relay3Aliases: Record<string, number> = {
      relay3_ch0: 0, relay3_ch1: 1, relay3_ch2: 2,
    };
    const missing = Object.keys(relay3Aliases).filter(k => !(k in this._aliases));
    if (missing.length > 0) {
      const aliases = { ...this._aliases, ...relay3Aliases };
      await api.putAliases(aliases);
      this._aliases = aliases;
    }
  }

  private _renderDeviceDialog() {
    if (!this._showDeviceDialog || !this._editingDevice) return nothing;
    const d = this._editingDevice;
    const isNew = this._isNewDevice;
    return html`
      <ms-dialog .open=${true} .headerTitle=${isNew ? "Add Device" : "Edit Device"} @closed=${() => this._showDeviceDialog = false}>
        <div class="dialog-body">
          ${this._field("Name", d.name, (v: string) => d.name = v)}
          ${this._field("ID", d.id, (v: string) => d.id = v, { disabled: !isNew })}
          ${this._idField("Node ID", d.node_id, (v: any) => d.node_id = v)}
          ${this._field("Gateway", d.gateway || "", (v: string) => d.gateway = v, { hint: "Leave empty for default gateway" })}

          <div class="dialog-section">Device Info</div>
          ${this._field("Manufacturer", d.manufacturer, (v: string) => d.manufacturer = v)}
          ${this._field("Model", d.model, (v: string) => d.model = v)}
          ${this._field("SW Version", d.sw_version, (v: string) => d.sw_version = v)}
          ${this._field("HW Version", d.hw_version, (v: string) => d.hw_version = v)}

          <div class="dialog-section">Optional</div>
          ${this._field("Configuration URL", d.configuration_url || "", (v: string) => d.configuration_url = v)}
          ${this._field("Suggested Area", d.suggested_area || "", (v: string) => d.suggested_area = v)}
          ${this._field("Via Device", d.via_device || "", (v: string) => d.via_device = v, { hint: "Device ID of the parent device" })}
        </div>
        <ms-button slot="footer" variant="neutral" appearance="plain" @click=${() => this._showDeviceDialog = false}>Cancel</ms-button>
        <ms-button slot="footer" @click=${() => this._saveDevice(isNew)}>Save</ms-button>
      </ms-dialog>
    `;
  }

  private async _saveDevice(isNew: boolean) {
    const dev = this._editingDevice;
    if (!dev.id) {
      this._toast("error", "Device ID is required");
      return;
    }
    try {
      if (isNew) {
        await api.postDevice(dev);
      } else {
        await api.putDevice(dev.id, dev);
      }
      this._showDeviceDialog = false;
      await this._load();
      this._toast("success", "Device saved");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _deleteDevice(dev: any) {
    try {
      if (dev.id) {
        await api.deleteDevice(dev.id);
      } else {
        // Device has no ID; fall back to bulk PUT excluding it by index
        const idx = this._devices.indexOf(dev);
        const remaining = this._devices.filter((_, i) => i !== idx);
        await api.putDevices(remaining);
      }
      await this._load();
      this._toast("success", "Device deleted");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  // --- Entity dialog ---
  private _addEntity(deviceId: string) {
    this._editingEntity = { name: "", id: "", child_id: 0, entity_type: "switch", icon: "", initial_value: "" };
    this._editingEntityDeviceId = deviceId;
    this._isNewEntity = true;
    this._showEntityDialog = true;
  }

  private _editEntity(deviceId: string, entity: any) {
    this._editingEntity = { ...entity };
    this._editingEntityDeviceId = deviceId;
    this._isNewEntity = false;
    this._showEntityDialog = true;
  }

  private _renderEntityDialog() {
    if (!this._showEntityDialog || !this._editingEntity) return nothing;
    const e = this._editingEntity;
    const isNew = this._isNewEntity;
    return html`
      <ms-dialog .open=${true} .headerTitle=${isNew ? "Add Entity" : "Edit Entity"} width="large" @closed=${() => this._showEntityDialog = false}>
        <div class="dialog-body">
          ${this._field("Name", e.name, (v: string) => e.name = v)}
          ${this._field("ID", e.id, (v: string) => e.id = v, { disabled: !isNew })}
          ${this._idField("Child ID", e.child_id, (v: any) => e.child_id = v)}
          ${this._selectField("Entity Type", e.entity_type, ENTITY_TYPES, (v: string) => e.entity_type = v)}
          ${this._field("Icon", e.icon || "", (v: string) => e.icon = v, { hint: "e.g. mdi:lightbulb" })}

          <div class="dialog-section">Optional Overrides</div>
          ${this._idField("Node ID", e.node_id, (v: any) => e.node_id = v, { hint: "Override device Node ID" })}
          ${this._field("Gateway", e.gateway || "", (v: string) => e.gateway = v, { hint: "Override device gateway" })}
          ${this._field("Initial Value", e.initial_value || "", (v: string) => e.initial_value = v)}
          ${this._field("Object ID", e.object_id ?? "", (v: string) => e.object_id = v === "" ? undefined : v, { hint: "Custom HA entity ID, empty to exclude from discovery" })}

          <div class="dialog-section">Capabilities</div>
          ${this._checkboxField("Read Only", e.read_only, (v: boolean) => e.read_only = v)}
          ${this._checkboxField("Write Only", e.write_only, (v: boolean) => e.write_only = v)}

          <div class="dialog-section">Number Entity Settings</div>
          ${this._numField("Min Value", e.min_value, (v: any) => e.min_value = v)}
          ${this._numField("Max Value", e.max_value, (v: any) => e.max_value = v)}
          ${this._numField("Step", e.step, (v: any) => e.step = v)}
          ${this._field("Unit", e.unit_of_measurement || "", (v: string) => e.unit_of_measurement = v)}

          <div class="dialog-section">Select Entity Settings</div>
          ${this._field("Options (comma-separated)", (e.options || []).join(", "), (v: string) => e.options = v ? v.split(",").map((s: string) => s.trim()) : [])}

          <div class="dialog-section">Home Assistant</div>
          ${this._field("Device Class", e.device_class || "", (v: string) => e.device_class = v)}
          ${this._selectField("Entity Category", e.entity_category || "", ["", "config", "diagnostic"], (v: string) => e.entity_category = v)}
          ${this._field("Variable Type Override", e.variable_type || "", (v: string) => e.variable_type = v, { hint: "e.g. V_STATUS, V_TEXT" })}

          <div class="dialog-section">Sync</div>
          ${this._field("Sync Period", e.sync_period ? `${e.sync_period / 1e9}s` : "", (v: string) => {
            const match = v.match(/^(\d+(?:\.\d+)?)\s*(s|m)?$/);
            if (match) {
              const val = parseFloat(match[1]);
              e.sync_period = match[2] === "m" ? val * 6e10 : val * 1e9;
            } else {
              e.sync_period = undefined;
            }
          }, { hint: "e.g. 30s, 5m. Empty to disable" })}
        </div>
        <ms-button slot="footer" variant="neutral" appearance="plain" @click=${() => this._showEntityDialog = false}>Cancel</ms-button>
        <ms-button slot="footer" @click=${() => this._saveEntity(isNew)}>Save</ms-button>
      </ms-dialog>
    `;
  }

  private async _saveEntity(isNew: boolean) {
    const entity = this._editingEntity;
    const deviceId = this._editingEntityDeviceId;
    if (!entity.id) {
      this._toast("error", "Entity ID is required");
      return;
    }
    try {
      if (isNew) {
        await api.postEntity(deviceId, entity);
      } else {
        await api.putEntity(deviceId, entity.id, entity);
      }
      this._showEntityDialog = false;
      await this._load();
      this._toast("success", "Entity saved");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _deleteEntity(deviceId: string, entityId: string) {
    try {
      await api.deleteEntity(deviceId, entityId);
      await this._load();
      this._toast("success", "Entity deleted");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  // --- Helper: text input field ---
  private _field(label: string, value: any, onChange: (v: string) => void, opts?: { hint?: string; disabled?: boolean }) {
    return html`
      <div class="dialog-field">
        <label>${label}</label>
        <input .value=${value ?? ""} ?disabled=${opts?.disabled} @input=${(ev: Event) => onChange((ev.target as HTMLInputElement).value)}>
        ${opts?.hint ? html`<div class="hint">${opts.hint}</div>` : nothing}
      </div>
    `;
  }

  // --- Helper: ID field (int or alias string) ---
  private _idField(label: string, value: any, onChange: (v: any) => void, opts?: { hint?: string }) {
    return html`
      <div class="dialog-field">
        <label>${label}</label>
        <input .value=${value ?? ""} @input=${(ev: Event) => {
          const v = (ev.target as HTMLInputElement).value;
          if (v === "") { onChange(null); return; }
          onChange(isNaN(Number(v)) ? v : Number(v));
        }}>
        <div class="hint">${opts?.hint || this._aliasHint()}</div>
      </div>
    `;
  }

  // --- Helper: select field ---
  private _selectField(label: string, value: string, options: string[], onChange: (v: string) => void) {
    return html`
      <div class="dialog-field">
        <label>${label}</label>
        <select @change=${(ev: Event) => onChange((ev.target as HTMLSelectElement).value)}>
          ${options.map(t => html`<option value=${t} ?selected=${value === t}>${t || "(none)"}</option>`)}
        </select>
      </div>
    `;
  }

  // --- Helper: checkbox field ---
  private _checkboxField(label: string, value: any, onChange: (v: boolean) => void) {
    return html`
      <div class="dialog-field" style="display:flex;align-items:center;gap:8px">
        <input type="checkbox" .checked=${!!value} @change=${(ev: Event) => onChange((ev.target as HTMLInputElement).checked)} style="width:auto">
        <label style="margin:0">${label}</label>
      </div>
    `;
  }

  // --- Helper: number field ---
  private _numField(label: string, value: any, onChange: (v: any) => void) {
    return html`
      <div class="dialog-field">
        <label>${label}</label>
        <input type="number" .value=${value != null ? String(value) : ""} @input=${(ev: Event) => {
          const v = (ev.target as HTMLInputElement).value;
          onChange(v === "" ? undefined : parseFloat(v));
        }}>
      </div>
    `;
  }

  // --- Confirm dialog ---
  private _confirmDelete(msg: string, action: () => void) {
    this._confirmMessage = msg;
    this._confirmAction = action;
    this._showConfirmDialog = true;
  }

  private _renderConfirmDialog() {
    if (!this._showConfirmDialog) return nothing;
    return html`
      <ms-dialog .open=${true} headerTitle="Confirm" width="small" @closed=${() => this._showConfirmDialog = false}>
        <p>${this._confirmMessage}</p>
        <ms-button slot="footer" variant="neutral" appearance="plain" @click=${() => this._showConfirmDialog = false}>Cancel</ms-button>
        <ms-button slot="footer" variant="danger" @click=${() => { this._showConfirmDialog = false; this._confirmAction?.(); }}>Delete</ms-button>
      </ms-dialog>
    `;
  }

  private _toast(type: "success" | "error", msg: string) {
    const toast = document.querySelector("ms-toast") as any;
    toast?.show(type, msg);
  }
}
