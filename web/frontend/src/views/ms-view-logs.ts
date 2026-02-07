import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";

@customElement("ms-view-logs")
export class MsViewLogs extends LitElement {
  @state() private _logs: string[] = [];
  @state() private _connected = false;
  @state() private _autoscroll = true;
  private _ws: WebSocket | null = null;
  private _logContainer: HTMLElement | null = null;

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      height: 100%;
    }
    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: var(--ha-space-3, 12px);
      padding: var(--ha-space-2, 8px) var(--ha-space-3, 12px);
      background: var(--card-background-color, white);
      border-radius: var(--ha-border-radius-md, 8px);
      border: 1px solid var(--divider-color);
    }
    .status {
      display: flex;
      align-items: center;
      gap: var(--ha-space-2, 8px);
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
    }
    .actions {
      display: flex;
      gap: var(--ha-space-2, 8px);
    }
    .log-container {
      flex: 1;
      overflow-y: auto;
      background: var(--card-background-color, white);
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-md, 8px);
      padding: var(--ha-space-3, 12px);
      font-family: var(--ha-font-family-code, monospace);
      font-size: var(--ha-font-size-s, 12px);
      line-height: 1.5;
    }
    .log-line {
      margin: 0;
      padding: 2px 0;
      white-space: pre-wrap;
      word-break: break-all;
    }
    .log-line.debug { color: var(--secondary-text-color); }
    .log-line.info { color: var(--primary-text-color); }
    .log-line.warn { color: var(--warning-color, #ff9800); }
    .log-line.error { color: var(--error-color, #db4437); }
    .checkbox-label {
      display: flex;
      align-items: center;
      gap: var(--ha-space-1, 4px);
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
      cursor: pointer;
    }
    .checkbox-label input {
      cursor: pointer;
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
    const protocol = location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${protocol}//${location.host}${basePath}/ws/logs`;

    this._ws = new WebSocket(url);

    this._ws.onopen = () => {
      this._connected = true;
    };

    this._ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data);
        if (msg.event === "log" && msg.data) {
          this._logs = [...this._logs, msg.data];
          if (this._logs.length > 1000) {
            this._logs = this._logs.slice(-1000);
          }
          if (this._autoscroll) {
            requestAnimationFrame(() => this._scrollToBottom());
          }
        }
      } catch (e) {
        // ignore parse errors
      }
    };

    this._ws.onclose = () => {
      this._connected = false;
      // Reconnect after 2 seconds
      setTimeout(() => this._connect(), 2000);
    };

    this._ws.onerror = () => {
      this._connected = false;
    };
  }

  private _disconnect() {
    if (this._ws) {
      this._ws.close();
      this._ws = null;
    }
  }

  private _scrollToBottom() {
    if (this._logContainer) {
      this._logContainer.scrollTop = this._logContainer.scrollHeight;
    }
  }

  private _clear() {
    this._logs = [];
  }

  private _getLogLevel(line: string): string {
    if (line.includes(" DEBUG ")) return "debug";
    if (line.includes(" INFO ")) return "info";
    if (line.includes(" WARN ")) return "warn";
    if (line.includes(" ERROR ")) return "error";
    return "info";
  }

  render() {
    return html`
      <div class="toolbar">
        <div class="status">
          <ms-status-dot .status=${this._connected ? "online" : "offline"}></ms-status-dot>
          ${this._connected ? "Connected" : "Disconnected"}
          <span>•</span>
          <span>${this._logs.length} lines</span>
        </div>
        <div class="actions">
          <label class="checkbox-label">
            <input
              type="checkbox"
              .checked=${this._autoscroll}
              @change=${(e: Event) => this._autoscroll = (e.target as HTMLInputElement).checked}
            />
            Auto-scroll
          </label>
          <ms-button appearance="outlined" variant="neutral" @click=${this._clear}>
            Clear
          </ms-button>
        </div>
      </div>
      <div
        class="log-container"
        ${(el: Element) => {
          this._logContainer = el as HTMLElement;
        }}
      >
        ${this._logs.map(
          (line) => html`
            <div class="log-line ${this._getLogLevel(line)}">${line}</div>
          `
        )}
      </div>
    `;
  }
}
