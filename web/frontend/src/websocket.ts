const basePath = (): string => (window as any).__BASE_PATH__ || "";

class AdapterWebSocket {
  private ws: WebSocket | null = null;
  private reconnectDelay = 1000;
  private pingInterval: number | undefined;
  private _basePath = "";

  connect() {
    this._basePath = basePath();
    const protocol = location.protocol === "https:" ? "wss:" : "ws:";
    this.ws = new WebSocket(
      `${protocol}//${location.host}${this._basePath}/ws/events`
    );
    this.ws.onmessage = (ev) => this.handleMessage(JSON.parse(ev.data));
    this.ws.onclose = () => this.scheduleReconnect();
    this.ws.onopen = () => {
      this.reconnectDelay = 1000;
      this.startPing();
    };
    this.ws.onerror = () => {}; // onclose fires after onerror
  }

  private handleMessage(msg: { event: string; data?: unknown }) {
    document.dispatchEvent(
      new CustomEvent(`ws-${msg.event}`, { detail: msg.data })
    );
  }

  private startPing() {
    clearInterval(this.pingInterval);
    this.pingInterval = window.setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ event: "ping" }));
      }
    }, 30000);
  }

  private scheduleReconnect() {
    clearInterval(this.pingInterval);
    setTimeout(() => this.connect(), this.reconnectDelay);
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
  }
}

export const adapterWS = new AdapterWebSocket();
