import { LitElement, html, css, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../api";

@customElement("ms-view-editor")
export class MsViewEditor extends LitElement {
  @state() private _yaml = "";
  @state() private _loading = true;
  @state() private _validationResult: { valid: boolean; error?: string } | null = null;
  @state() private _saving = false;

  static styles = css`
    :host { display: block; }
    textarea {
      width: 100%;
      min-height: 500px;
      box-sizing: border-box;
      padding: var(--ha-space-4, 16px);
      font-family: var(--ha-font-family-code, monospace);
      font-size: var(--ha-font-size-m, 14px);
      border: 1px solid var(--divider-color, #e0e0e0);
      border-radius: var(--ha-border-radius-md, 8px);
      background: var(--card-background-color, white);
      color: var(--primary-text-color);
      resize: vertical;
      tab-size: 2;
    }
    textarea:focus {
      outline: 2px solid var(--primary-color);
      outline-offset: -1px;
    }
    .actions {
      display: flex;
      gap: var(--ha-space-3, 12px);
      margin-top: var(--ha-space-4, 16px);
    }
    .notice {
      margin-bottom: var(--ha-space-3, 12px);
    }
    .loading { text-align: center; padding: var(--ha-space-8); color: var(--secondary-text-color); }
  `;

  connectedCallback() {
    super.connectedCallback();
    this._load();
  }

  private async _load() {
    try {
      this._yaml = await api.getRawConfig();
    } catch (e) { /* ignore */ }
    this._loading = false;
  }

  render() {
    if (this._loading) return html`<div class="loading">Loading...</div>`;

    return html`
      <ms-alert class="notice" alertType="info" title="Note">
        YAML comments are preserved when saving from this editor. Comments are only removed when using the structured API (Devices, Gateways, MQTT settings pages).
      </ms-alert>
      ${this._validationResult
        ? html`
            <ms-alert
              alertType=${this._validationResult.valid ? "success" : "error"}
              title=${this._validationResult.valid ? "Valid" : "Invalid"}
              dismissable
              @dismissed=${() => (this._validationResult = null)}
            >
              ${this._validationResult.error || "Configuration is valid."}
            </ms-alert>
          `
        : nothing}
      <textarea
        .value=${this._yaml}
        @input=${(e: Event) =>
          (this._yaml = (e.target as HTMLTextAreaElement).value)}
        @keydown=${this._onKeyDown}
      ></textarea>
      <div class="actions">
        <ms-button appearance="outlined" @click=${this._validate}>
          Validate
        </ms-button>
        <ms-button ?loading=${this._saving} @click=${this._save}>
          Save
        </ms-button>
      </div>
    `;
  }

  private _onKeyDown(e: KeyboardEvent) {
    if (e.key === "Tab") {
      e.preventDefault();
      const ta = e.target as HTMLTextAreaElement;
      const start = ta.selectionStart;
      const end = ta.selectionEnd;
      this._yaml = this._yaml.substring(0, start) + "  " + this._yaml.substring(end);
      requestAnimationFrame(() => {
        ta.selectionStart = ta.selectionEnd = start + 2;
      });
    }
  }

  private async _validate() {
    try {
      this._validationResult = await api.validateConfig(this._yaml, true);
    } catch (e: any) {
      this._validationResult = { valid: false, error: e.message };
    }
  }

  private async _save() {
    this._saving = true;
    try {
      const result = await api.putRawConfig(this._yaml);
      if (result.status === "ok") {
        this._toast("success", "Configuration saved");
        this._validationResult = null;
      } else {
        this._toast("error", result.error || "Save failed");
      }
    } catch (e: any) {
      this._toast("error", e.message);
    } finally {
      this._saving = false;
    }
  }

  private _toast(type: "success" | "error", msg: string) {
    const toast = document.querySelector("ms-toast") as any;
    toast?.show(type, msg);
  }
}
