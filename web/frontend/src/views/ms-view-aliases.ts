import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";

@customElement("ms-view-aliases")
export class MsViewAliases extends LitElement {
  @state() private _aliases: Record<string, number> = {};
  @state() private _loading = true;
  @state() private _newAlias = "";
  @state() private _newId = 0;

  static styles = css`
    :host { display: block; }
    table {
      width: 100%;
      border-collapse: collapse;
      background: var(--card-background-color, white);
      border-radius: var(--ha-border-radius-lg, 12px);
      overflow: hidden;
      border: 1px solid var(--divider-color);
    }
    th, td {
      padding: var(--ha-space-2, 8px) var(--ha-space-4, 16px);
      text-align: left;
      font-size: var(--ha-font-size-m, 14px);
    }
    th {
      background: var(--primary-background-color, #fafafa);
      font-weight: var(--ha-font-weight-medium, 500);
      color: var(--secondary-text-color);
      font-size: var(--ha-font-size-s, 12px);
      text-transform: uppercase;
    }
    tr:not(:last-child) td {
      border-bottom: 1px solid var(--divider-color);
    }
    .actions { display: flex; gap: var(--ha-space-1, 4px); }
    .actions button {
      background: none; border: none; cursor: pointer;
      color: var(--secondary-text-color); font-size: 12px; padding: 4px 6px;
    }
    .actions button:hover { color: var(--primary-text-color); }
    .add-row {
      display: flex; gap: var(--ha-space-2, 8px); align-items: center;
      margin-top: var(--ha-space-4, 16px);
    }
    .add-row input {
      padding: var(--ha-space-2) var(--ha-space-3);
      border: 1px solid var(--divider-color); border-radius: var(--ha-border-radius-sm);
      font-size: var(--ha-font-size-m); font-family: inherit;
      background: var(--card-background-color); color: var(--primary-text-color);
    }
    .loading { text-align: center; padding: var(--ha-space-8); color: var(--secondary-text-color); }
    .id-input { width: 80px; }
    .bulk-actions {
      display: flex;
      gap: var(--ha-space-2, 8px);
      margin-bottom: var(--ha-space-4, 16px);
      padding: var(--ha-space-3, 12px);
      background: var(--card-background-color, white);
      border: 1px solid var(--divider-color);
      border-radius: var(--ha-border-radius-md, 8px);
    }
    .bulk-actions-label {
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
      font-weight: var(--ha-font-weight-medium, 500);
      text-transform: uppercase;
      margin-right: var(--ha-space-2, 8px);
      align-self: center;
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
  }

  private async _load() {
    try {
      this._aliases = await api.getAliases();
    } catch (e) { /* ignore */ }
    this._loading = false;
  }

  render() {
    if (this._loading) return html`<div class="loading">Loading...</div>`;

    const entries = Object.entries(this._aliases).sort((a, b) =>
      a[0].localeCompare(b[0])
    );

    return html`
      <div class="bulk-actions">
        <span class="bulk-actions-label">Quick Add:</span>
        <ms-button appearance="outlined" variant="neutral" @click=${this._addRelay9Aliases}>
          nippy DIN Relay 9 Aliases
        </ms-button>
        <ms-button appearance="outlined" variant="neutral" @click=${this._addRelay3Aliases}>
          nippy DIN Relay 3 Aliases
        </ms-button>
      </div>
      <table>
        <thead>
          <tr>
            <th>Alias</th>
            <th>ID</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          ${entries.map(
            ([alias, id]) => html`
              <tr>
                <td>${alias}</td>
                <td>${id}</td>
                <td class="actions">
                  <button @click=${() => this._delete(alias)}>Delete</button>
                </td>
              </tr>
            `
          )}
        </tbody>
      </table>
      <div class="add-row">
        <input
          placeholder="Alias name"
          .value=${this._newAlias}
          @input=${(e: Event) =>
            (this._newAlias = (e.target as HTMLInputElement).value)}
        />
        <input
          class="id-input"
          type="number"
          placeholder="ID"
          .value=${String(this._newId)}
          @input=${(e: Event) =>
            (this._newId = parseInt((e.target as HTMLInputElement).value) || 0)}
        />
        <ms-button @click=${this._add}>Add</ms-button>
      </div>
    `;
  }

  private async _add() {
    if (!this._newAlias) return;
    const aliases = { ...this._aliases, [this._newAlias]: this._newId };
    try {
      await api.putAliases(aliases);
      this._aliases = aliases;
      this._newAlias = "";
      this._newId = 0;
      this._toast("success", "Alias added");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _delete(alias: string) {
    const aliases = { ...this._aliases };
    delete aliases[alias];
    try {
      await api.putAliases(aliases);
      this._aliases = aliases;
      this._toast("success", "Alias deleted");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addRelay9Aliases() {
    const relay9Aliases: Record<string, number> = {
      relay9_ch0: 0,
      relay9_ch1: 1,
      relay9_ch2: 2,
      relay9_ch3: 8,
      relay9_ch4: 7,
      relay9_ch5: 6,
      relay9_ch6: 5,
      relay9_ch7: 4,
      relay9_ch8: 3,
    };

    const aliases = { ...this._aliases, ...relay9Aliases };
    try {
      await api.putAliases(aliases);
      this._aliases = aliases;
      this._toast("success", "Added Nippy Relay 9 aliases (ch0-ch8)");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private async _addRelay3Aliases() {
    const relay3Aliases: Record<string, number> = {
      ch0: 0,
      ch1: 1,
      ch2: 2,
    };

    const aliases = { ...this._aliases, ...relay3Aliases };
    try {
      await api.putAliases(aliases);
      this._aliases = aliases;
      this._toast("success", "Added Nippy Relay 3 aliases (ch0-ch2)");
    } catch (e: any) {
      this._toast("error", e.message);
    }
  }

  private _toast(type: "success" | "error", msg: string) {
    const toast = document.querySelector("ms-toast") as any;
    toast?.show(type, msg);
  }
}
