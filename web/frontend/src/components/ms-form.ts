import { LitElement, html, css, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

export interface FormSchema {
  name: string;
  label: string;
  type: "string" | "integer" | "boolean" | "select" | "password";
  required?: boolean;
  options?: { value: string; label: string }[];
  default?: unknown;
  hint?: string;
  disabled?: boolean;
}

@customElement("ms-form")
export class MsForm extends LitElement {
  @property({ attribute: false }) schema: FormSchema[] = [];
  @property({ attribute: false }) data: Record<string, unknown> = {};
  @property({ attribute: false }) error: Record<string, string> = {};

  static styles = css`
    :host {
      display: block;
    }
    .field {
      margin-bottom: var(--ha-space-4, 16px);
    }
    label {
      display: block;
      font-size: var(--ha-font-size-s, 12px);
      color: var(--secondary-text-color);
      margin-bottom: var(--ha-space-1, 4px);
      font-weight: var(--ha-font-weight-medium, 500);
    }
    input,
    select {
      width: 100%;
      box-sizing: border-box;
      padding: var(--ha-space-2) var(--ha-space-3);
      border: 1px solid var(--divider-color, #e0e0e0);
      border-radius: var(--ha-border-radius-sm, 4px);
      font-size: var(--ha-font-size-m, 14px);
      font-family: inherit;
      background: var(--card-background-color, white);
      color: var(--primary-text-color);
    }
    input:focus,
    select:focus {
      outline: 2px solid var(--primary-color);
      outline-offset: -1px;
    }
    .checkbox-row {
      display: flex;
      align-items: center;
      gap: var(--ha-space-2);
    }
    .checkbox-row input {
      width: auto;
    }
    .error-text {
      color: var(--error-color);
      font-size: var(--ha-font-size-s, 12px);
      margin-top: var(--ha-space-1, 4px);
    }
    .hint-text {
      color: var(--secondary-text-color);
      font-size: var(--ha-font-size-s, 12px);
      margin-top: var(--ha-space-1, 4px);
    }
  `;

  render() {
    return html`
      ${this.schema.map((field) => this._renderField(field))}
    `;
  }

  private _renderField(field: FormSchema) {
    const value = this.data[field.name] ?? field.default ?? "";
    const err = this.error[field.name];

    if (field.type === "boolean") {
      return html`
        <div class="field">
          <div class="checkbox-row">
            <input
              type="checkbox"
              .checked=${!!value}
              @change=${(e: Event) =>
                this._onChange(field.name, (e.target as HTMLInputElement).checked)}
            />
            <label>${field.label}</label>
          </div>
        </div>
      `;
    }

    if (field.type === "select" && field.options) {
      return html`
        <div class="field">
          <label>${field.label}</label>
          <select
            @change=${(e: Event) =>
              this._onChange(field.name, (e.target as HTMLSelectElement).value)}
          >
            ${field.options.map(
              (opt) => html`
                <option value=${opt.value} ?selected=${value === opt.value}>
                  ${opt.label}
                </option>
              `
            )}
          </select>
          ${err ? html`<div class="error-text">${err}</div>` : nothing}
        </div>
      `;
    }

    const inputType =
      field.type === "password"
        ? "password"
        : field.type === "integer"
          ? "number"
          : "text";

    return html`
      <div class="field">
        <label>${field.label}</label>
        <input
          type=${inputType}
          .value=${String(value)}
          ?required=${field.required}
          ?disabled=${field.disabled}
          @input=${(e: Event) => {
            let v: unknown = (e.target as HTMLInputElement).value;
            if (field.type === "integer") v = parseInt(v as string, 10);
            this._onChange(field.name, v);
          }}
        />
        ${field.hint ? html`<div class="hint-text">${field.hint}</div>` : nothing}
        ${err ? html`<div class="error-text">${err}</div>` : nothing}
      </div>
    `;
  }

  private _onChange(name: string, value: unknown) {
    const newData = { ...this.data, [name]: value };
    this.data = newData;
    this.dispatchEvent(
      new CustomEvent("value-changed", { detail: { value: newData } })
    );
  }
}
