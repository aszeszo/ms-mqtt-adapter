export type Route = "dashboard" | "devices" | "gateways" | "mqtt" | "aliases" | "editor" | "logs" | "mqtt-topics";

export function currentRoute(): Route {
  const h = location.hash.replace("#", "") || "dashboard";
  const valid: Route[] = ["dashboard", "devices", "gateways", "mqtt", "aliases", "editor", "logs", "mqtt-topics"];
  return valid.includes(h as Route) ? (h as Route) : "dashboard";
}

export function navigate(route: Route) {
  location.hash = route;
}
