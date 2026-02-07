import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";
import type { Route } from "../router";
import { navigate } from "../router";

interface NavItem {
  route: Route;
  label: string;
}

const NAV_ITEMS: NavItem[] = [
  { route: "dashboard", label: "Status" },
  { route: "devices", label: "Devices" },
  { route: "gateways", label: "Gateways" },
  { route: "mqtt", label: "MQTT" },
  { route: "mqtt-topics", label: "MQTT Topics" },
  { route: "aliases", label: "Aliases" },
  { route: "logs", label: "Logs" },
  { route: "editor", label: "Editor" },
];

@customElement("ms-sidebar")
export class MsSidebar extends LitElement {
  @property() activeRoute: Route = "dashboard";

  static styles = css`
    :host {
      display: block;
    }
    nav {
      display: flex;
      gap: var(--ha-space-1, 4px);
      padding: 0 var(--ha-space-2, 8px);
    }
    a {
      display: block;
      padding: var(--ha-space-2, 8px) var(--ha-space-3, 12px);
      color: var(--primary-text-color);
      text-decoration: none;
      font-size: var(--ha-font-size-m, 14px);
      font-weight: var(--ha-font-weight-medium, 500);
      border-bottom: 2px solid transparent;
      transition: background var(--ha-animation-duration-fast, 150ms);
      white-space: nowrap;
    }
    a:hover {
      background: var(--primary-background-color, #fafafa);
      border-radius: var(--ha-border-radius-sm, 4px) var(--ha-border-radius-sm, 4px) 0 0;
    }
    a.active {
      color: var(--primary-color);
      border-bottom-color: var(--primary-color);
    }
  `;

  render() {
    return html`
      <nav>
        ${NAV_ITEMS.map(
          (item) => html`
            <a
              class=${item.route === this.activeRoute ? "active" : ""}
              href="#${item.route}"
              @click=${(e: Event) => {
                e.preventDefault();
                navigate(item.route);
                this.dispatchEvent(
                  new CustomEvent("route-changed", {
                    detail: item.route,
                    bubbles: true,
                    composed: true,
                  })
                );
              }}
            >
              ${item.label}
            </a>
          `
        )}
      </nav>
    `;
  }
}
