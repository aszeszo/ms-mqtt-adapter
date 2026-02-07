import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

interface TrafficMessage {
  timestamp: string;
  gateway: string;
  direction: string;
  raw: string;
  node_id: number;
  child_id: number;
  type: string;
  ack: boolean;
  sub_type: number;
  payload: string;
}

@customElement("ms-view-traffic")
export class MsViewTraffic extends LitElement {
  @state() private _messages: TrafficMessage[] = [];
  @state() private _paused = false;
  @state() private _filter = "";
  @state() private _maxMessages = 500;
  @state() private _ws: WebSocket | null = null;

  static styles = css`
    :host { display: block; height: 100%; display: flex; flex-direction: column; }

    .toolbar {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px;
      border-bottom: 1px solid var(--divider-color);
      background: var(--card-background-color);
      flex-shrink: 0;
    }
    .toolbar input {
      flex: 1;
      min-width: 200px;
      padding: 6px 10px;
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-sm, 4px);
      background: var(--primary-background-color);
      color: var(--primary-text-color);
      font-size: 13px;
    }
    .toolbar input::placeholder {
      color: var(--secondary-text-color);
    }
    .count {
      font-size: 12px;
      color: var(--secondary-text-color);
      white-space: nowrap;
    }

    .traffic-container {
      flex: 1;
      overflow: auto;
      background: var(--primary-background-color);
      font-family: var(--ha-font-family-code, monospace);
      font-size: 12px;
      line-height: 1.4;
    }
    .message-row {
      display: grid;
      grid-template-columns: 140px 80px 40px 380px auto;
      gap: 8px;
      padding: 4px 12px;
      border-bottom: 1px solid color-mix(in srgb, var(--divider-color) 50%, transparent);
      align-items: start;
    }
    .message-row:hover {
      background: color-mix(in srgb, var(--primary-color) 5%, transparent);
    }
    .message-row.rx {
      border-left: 3px solid var(--info-color);
    }
    .message-row.tx {
      border-left: 3px solid var(--warning-color);
    }

    .timestamp {
      color: var(--secondary-text-color);
      font-size: 11px;
    }
    .gateway {
      color: var(--primary-text-color);
      font-weight: 500;
    }
    .direction {
      font-weight: 700;
      font-size: 10px;
      text-transform: uppercase;
    }
    .direction.rx { color: var(--info-color); }
    .direction.tx { color: var(--warning-color); }
    .raw {
      color: var(--primary-text-color);
      word-break: break-all;
    }
    .details {
      font-size: 11px;
      color: var(--secondary-text-color);
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .detail-item {
      white-space: nowrap;
    }
    .detail-label {
      color: var(--disabled-text-color);
    }

    .status {
      padding: 12px;
      text-align: center;
      color: var(--secondary-text-color);
      font-size: 13px;
      background: var(--card-background-color);
      border-bottom: 1px solid var(--divider-color);
    }
    .status.paused {
      background: var(--warning-color);
      color: white;
    }
    .empty {
      padding: 32px;
      text-align: center;
      color: var(--secondary-text-color);
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._connect();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    this._disconnect();
  }

  private _connect() {
    const basePath = (window as any).__BASE_PATH__ || "";
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}${basePath}/ws/traffic`;

    this._ws = new WebSocket(wsUrl);

    this._ws.onmessage = (evt) => {
      if (this._paused) return;

      try {
        const event = JSON.parse(evt.data);
        if (event.event === "mysensors_message" && event.data) {
          this._messages = [event.data, ...this._messages].slice(0, this._maxMessages);
        }
      } catch (e) {
        console.error("Failed to parse WebSocket message", e);
      }
    };

    this._ws.onerror = (error) => {
      console.error("WebSocket error", error);
    };

    this._ws.onclose = () => {
      // Reconnect after 2 seconds
      setTimeout(() => {
        if (this.isConnected) {
          this._connect();
        }
      }, 2000);
    };
  }

  private _disconnect() {
    if (this._ws) {
      this._ws.close();
      this._ws = null;
    }
  }

  private get _filtered(): TrafficMessage[] {
    if (!this._filter) return this._messages;
    const q = this._filter.toLowerCase();
    return this._messages.filter(
      (m) =>
        m.raw.toLowerCase().includes(q) ||
        m.gateway.toLowerCase().includes(q) ||
        m.type.toLowerCase().includes(q) ||
        m.payload.toLowerCase().includes(q)
    );
  }

  render() {
    const filtered = this._filtered;

    return html`
      <div class="toolbar">
        <input
          type="text"
          placeholder="Filter messages..."
          .value=${this._filter}
          @input=${(e: Event) => { this._filter = (e.target as HTMLInputElement).value; }}
        />
        <span class="count">${filtered.length} messages</span>
        <ms-button
          appearance="outlined"
          variant="neutral"
          size="small"
          @click=${() => { this._paused = !this._paused; }}
        >
          ${this._paused ? "Resume" : "Pause"}
        </ms-button>
        <ms-button
          appearance="outlined"
          variant="neutral"
          size="small"
          @click=${this._clear}
        >
          Clear
        </ms-button>
      </div>

      ${this._paused
        ? html`<div class="status paused">⏸ PAUSED - Click Resume to continue</div>`
        : nothing}

      ${filtered.length === 0
        ? html`<div class="empty">
            ${this._messages.length === 0
              ? "Waiting for MySensors traffic..."
              : "No messages match filter"}
          </div>`
        : html`
          <div class="traffic-container">
            ${filtered.map((msg) => this._renderMessage(msg))}
          </div>
        `}
    `;
  }

  private _renderMessage(msg: TrafficMessage) {
    const time = new Date(msg.timestamp).toLocaleTimeString();

    return html`
      <div class="message-row ${msg.direction}">
        <div class="timestamp">${time}</div>
        <div class="gateway">${msg.gateway}</div>
        <div class="direction ${msg.direction}">${msg.direction}</div>
        <div class="raw">${msg.raw}</div>
        <div class="details">
          <span class="detail-item">
            <span class="detail-label">Node:</span> ${msg.node_id}
          </span>
          <span class="detail-item">
            <span class="detail-label">Child:</span> ${msg.child_id}
          </span>
          <span class="detail-item">
            <span class="detail-label">Type:</span> ${msg.type}
          </span>
          ${msg.ack ? html`<span class="detail-item">ACK</span>` : nothing}
          ${msg.payload
            ? html`<span class="detail-item">
                <span class="detail-label">Payload:</span> ${msg.payload}
              </span>`
            : nothing}
        </div>
      </div>
    `;
  }

  private _clear() {
    this._messages = [];
  }
}
