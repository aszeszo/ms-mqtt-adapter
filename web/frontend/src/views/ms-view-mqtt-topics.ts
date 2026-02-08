import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";

interface BrokerTopic {
  topic: string;
  payload: string;
  retained: boolean;
}

@customElement("ms-view-mqtt-topics")
export class MsViewMqttTopics extends LitElement {
  @state() private _topics: BrokerTopic[] = [];
  @state() private _loading = true;
  @state() private _search = "";
  @state() private _expandedTopic = "";
  @state() private _selected = new Set<string>();
  @state() private _showConfirm = false;
  @state() private _confirmMessage = "";
  @state() private _confirmAction: (() => Promise<void>) | null = null;

  static styles = css`
    :host { display: block; }

    .toolbar {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;
      flex-wrap: wrap;
    }
    .toolbar input {
      flex: 1;
      min-width: 200px;
      padding: 6px 10px;
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-sm, 4px);
      background: var(--card-background-color);
      color: var(--primary-text-color);
      font-size: 13px;
      font-family: inherit;
    }
    .toolbar input::placeholder {
      color: var(--secondary-text-color);
    }
    .count {
      font-size: 12px;
      color: var(--secondary-text-color);
      white-space: nowrap;
    }

    .topic-table {
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-m, 6px);
      overflow: hidden;
    }
    .topic-row {
      display: grid;
      grid-template-columns: 32px 1fr auto auto 32px;
      align-items: center;
      padding: 4px 8px;
      font-size: 12px;
      line-height: 1.3;
      border-bottom: 1px solid var(--divider-color);
      cursor: pointer;
      transition: background 0.1s;
    }
    .topic-row:last-child { border-bottom: none; }
    .topic-row:nth-child(even) { background: color-mix(in srgb, var(--secondary-background-color) 50%, transparent); }
    .topic-row:hover { background: color-mix(in srgb, var(--primary-color) 8%, transparent); }
    .topic-row.expanded { background: color-mix(in srgb, var(--primary-color) 12%, transparent); }

    .topic-row input[type="checkbox"] {
      width: 14px;
      height: 14px;
      margin: 0;
      cursor: pointer;
    }
    .topic-path {
      font-family: var(--ha-font-family-code, monospace);
      font-size: 11px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      min-width: 0;
    }
    .topic-payload-preview {
      font-size: 11px;
      color: var(--secondary-text-color);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      max-width: 300px;
      padding: 0 8px;
    }
    .retained-badge {
      padding: 1px 5px;
      background: var(--divider-color);
      border-radius: 8px;
      font-size: 10px;
      white-space: nowrap;
    }
    .delete-btn {
      padding: 2px 6px;
      font-size: 10px;
      background: none;
      border: 1px solid var(--divider-color);
      border-radius: 3px;
      cursor: pointer;
      color: var(--error-color);
      line-height: 1;
    }
    .delete-btn:hover {
      background: var(--error-color);
      color: white;
    }

    .payload-detail {
      padding: 8px 8px 8px 40px;
      border-bottom: 1px solid var(--divider-color);
      background: var(--secondary-background-color);
    }
    .payload-detail pre {
      margin: 0;
      font-family: var(--ha-font-family-code, monospace);
      font-size: 11px;
      white-space: pre-wrap;
      word-break: break-all;
      max-height: 300px;
      overflow: auto;
      color: var(--primary-text-color);
    }
    .payload-detail .payload-actions {
      margin-top: 6px;
      display: flex;
      gap: 6px;
    }
    .tree-delete-btn {
      padding: 2px 8px;
      font-size: 11px;
      background: none;
      border: 1px solid var(--error-color);
      border-radius: 3px;
      cursor: pointer;
      color: var(--error-color);
    }
    .tree-delete-btn:hover {
      background: var(--error-color);
      color: white;
    }

    .loading {
      text-align: center;
      padding: 32px;
      color: var(--secondary-text-color);
    }
    .empty {
      text-align: center;
      padding: 24px;
      color: var(--secondary-text-color);
      font-size: 13px;
    }
    .select-all-label {
      font-size: 12px;
      color: var(--secondary-text-color);
      display: flex;
      align-items: center;
      gap: 4px;
      cursor: pointer;
    }
    .select-all-label input { cursor: pointer; }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
  }

  private async _load() {
    this._loading = true;
    try {
      this._topics = await api.getMQTTTopics();
    } catch {
      this._topics = [];
      this._toast("error", "Failed to load topics");
    }
    this._loading = false;
  }

  private get _filtered(): BrokerTopic[] {
    if (!this._search) return this._topics;
    const q = this._search.toLowerCase();
    return this._topics.filter(
      (t) => t.topic.toLowerCase().includes(q) || t.payload.toLowerCase().includes(q)
    );
  }

  render() {
    if (this._loading) {
      return html`<div class="loading">Browsing MQTT broker topics (5s)...</div>`;
    }

    const filtered = this._filtered;
    const allFilteredTopics = filtered.map((t) => t.topic);
    const allSelected = filtered.length > 0 && filtered.every((t) => this._selected.has(t.topic));

    return html`
      <div class="toolbar">
        <input
          type="text"
          placeholder="Filter topics..."
          .value=${this._search}
          @input=${(e: Event) => { this._search = (e.target as HTMLInputElement).value; }}
        />
        <span class="count">
          ${filtered.length === this._topics.length
            ? `${this._topics.length} topics`
            : `${filtered.length} of ${this._topics.length} topics`}
        </span>
        <ms-button appearance="outlined" variant="neutral" size="small" @click=${this._load}>
          Refresh
        </ms-button>
        ${this._selected.size > 0
          ? html`<ms-button variant="danger" size="small" @click=${this._deleteSelected}>
              Delete Selected (${this._selected.size})
            </ms-button>`
          : nothing}
      </div>

      ${filtered.length === 0
        ? html`<div class="empty">${this._topics.length === 0 ? "No topics found on broker" : "No topics match filter"}</div>`
        : html`
          <div style="margin-bottom:6px">
            <label class="select-all-label">
              <input
                type="checkbox"
                .checked=${allSelected}
                @change=${() => this._toggleSelectAll(allFilteredTopics, !allSelected)}
              />
              Select all${this._search ? " matching" : ""}
            </label>
          </div>
          <div class="topic-table">
            ${filtered.map((t) => this._renderTopicRow(t))}
          </div>
        `}

      ${this._renderConfirmDialog()}
    `;
  }

  private _renderTopicRow(t: BrokerTopic) {
    const expanded = this._expandedTopic === t.topic;
    const preview = t.payload.length > 60 ? t.payload.slice(0, 60) + "..." : t.payload;

    // Calculate tree prefix segments for subtree deletion
    const segments = t.topic.split("/");
    const treePrefix = segments.length > 1 ? segments.slice(0, -1).join("/") : "";

    return html`
      <div
        class="topic-row ${expanded ? "expanded" : ""}"
        @click=${(e: Event) => {
          // Don't toggle on checkbox or button clicks
          if ((e.target as HTMLElement).tagName === "INPUT" || (e.target as HTMLElement).tagName === "BUTTON") return;
          this._expandedTopic = expanded ? "" : t.topic;
        }}
      >
        <input
          type="checkbox"
          .checked=${this._selected.has(t.topic)}
          @change=${(e: Event) => {
            e.stopPropagation();
            this._toggleSelect(t.topic);
          }}
        />
        <span class="topic-path" title=${t.topic}>${t.topic}</span>
        <span class="topic-payload-preview" title=${t.payload}>${preview || "(empty)"}</span>
        ${t.retained ? html`<span class="retained-badge">retained</span>` : html`<span></span>`}
        <button class="delete-btn" title="Delete this topic" @click=${(e: Event) => {
          e.stopPropagation();
          this._confirmDeleteTopics([t.topic], `Delete topic "${t.topic}"?`);
        }}>x</button>
      </div>
      ${expanded ? html`
        <div class="payload-detail">
          <pre>${this._formatPayload(t.payload)}</pre>
          <div class="payload-actions">
            ${treePrefix ? html`
              <button class="tree-delete-btn" @click=${() =>
                this._confirmDeleteTree(treePrefix, `Delete all retained topics under "${treePrefix}/"?`)
              }>
                Delete tree: ${treePrefix}/...
              </button>
            ` : nothing}
          </div>
        </div>
      ` : nothing}
    `;
  }

  private _formatPayload(payload: string): string {
    if (!payload) return "(empty)";
    try {
      const parsed = JSON.parse(payload);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return payload;
    }
  }

  private _toggleSelect(topic: string) {
    const next = new Set(this._selected);
    if (next.has(topic)) {
      next.delete(topic);
    } else {
      next.add(topic);
    }
    this._selected = next;
  }

  private _toggleSelectAll(topics: string[], select: boolean) {
    const next = new Set(this._selected);
    for (const t of topics) {
      if (select) {
        next.add(t);
      } else {
        next.delete(t);
      }
    }
    this._selected = next;
  }

  private _confirmDeleteTopics(topics: string[], message: string) {
    this._confirmMessage = message;
    this._confirmAction = async () => {
      await api.deleteMQTTTopics({ topics });
      this._selected = new Set();
      await this._load();
      this._toast("success", `Deleted ${topics.length} topic(s)`);
    };
    this._showConfirm = true;
  }

  private _confirmDeleteTree(prefix: string, message: string) {
    this._confirmMessage = message;
    this._confirmAction = async () => {
      const result = await api.deleteMQTTTopics({ prefix }) as any;
      this._selected = new Set();
      await this._load();
      this._toast("success", `Deleted ${result.deleted || 0} retained topic(s)`);
    };
    this._showConfirm = true;
  }

  private _deleteSelected() {
    const topics = Array.from(this._selected);
    this._confirmDeleteTopics(topics, `Delete ${topics.length} selected topic(s)?`);
  }

  private async _executeConfirm() {
    try {
      if (this._confirmAction) await this._confirmAction();
    } catch (e: any) {
      this._toast("error", e.message);
    }
    this._showConfirm = false;
    this._confirmAction = null;
  }

  private _renderConfirmDialog() {
    if (!this._showConfirm) return nothing;
    return html`
      <ms-dialog .open=${true} headerTitle="Confirm Delete" @closed=${() => this._showConfirm = false}>
        <p>${this._confirmMessage}</p>
        <p style="margin-top:8px;font-size:12px;color:var(--secondary-text-color)">
          This will publish empty retained messages to clear the topic(s) from the broker.
        </p>
        <ms-button slot="footer" variant="neutral" appearance="plain" @click=${() => this._showConfirm = false}>Cancel</ms-button>
        <ms-button slot="footer" variant="danger" @click=${this._executeConfirm}>Delete</ms-button>
      </ms-dialog>
    `;
  }

  private _toast(type: "success" | "error", msg: string) {
    const toast = document.querySelector("ms-toast") as any;
    toast?.show(type, msg);
  }
}
