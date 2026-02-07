const basePath = (): string => (window as any).__BASE_PATH__ || "";

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const url = `${basePath()}${path}`;
  const opts: RequestInit = {
    method,
    headers: { "Content-Type": "application/json" },
  };
  if (body !== undefined) {
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(url, opts);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) {
    return res.json();
  }
  return res.text() as unknown as T;
}

export const api = {
  // Config
  getConfig: () => request<any>("GET", "/api/config"),
  putConfig: (cfg: any) => request<any>("PUT", "/api/config", cfg),
  getLogLevel: () => request<any>("GET", "/api/config/log_level"),
  putLogLevel: (v: string) => request<any>("PUT", "/api/config/log_level", { log_level: v }),
  getMQTT: () => request<any>("GET", "/api/config/mqtt"),
  putMQTT: (v: any) => request<any>("PUT", "/api/config/mqtt", v),
  getAdapter: () => request<any>("GET", "/api/config/adapter"),
  putAdapter: (v: any) => request<any>("PUT", "/api/config/adapter", v),
  getAliases: () => request<any>("GET", "/api/config/id_aliases"),
  putAliases: (v: any) => request<any>("PUT", "/api/config/id_aliases", v),
  getMySensors: () => request<any>("GET", "/api/config/mysensors"),
  putMySensors: (v: any) => request<any>("PUT", "/api/config/mysensors", v),
  getGateway: (name: string) => request<any>("GET", `/api/config/mysensors/${name}`),
  putGateway: (name: string, v: any) => request<any>("PUT", `/api/config/mysensors/${name}`, v),
  deleteGateway: (name: string) => request<any>("DELETE", `/api/config/mysensors/${name}`),
  getDevices: () => request<any[]>("GET", "/api/config/devices"),
  putDevices: (v: any[]) => request<any>("PUT", "/api/config/devices", v),
  postDevice: (v: any) => request<any>("POST", "/api/config/devices", v),
  getDevice: (id: string) => request<any>("GET", `/api/config/devices/${id}`),
  putDevice: (id: string, v: any) => request<any>("PUT", `/api/config/devices/${id}`, v),
  deleteDevice: (id: string) => request<any>("DELETE", `/api/config/devices/${id}`),
  getEntities: (id: string) => request<any[]>("GET", `/api/config/devices/${id}/entities`),
  postEntity: (id: string, v: any) => request<any>("POST", `/api/config/devices/${id}/entities`, v),
  getEntity: (id: string, eid: string) => request<any>("GET", `/api/config/devices/${id}/entities/${eid}`),
  putEntity: (id: string, eid: string, v: any) => request<any>("PUT", `/api/config/devices/${id}/entities/${eid}`, v),
  deleteEntity: (id: string, eid: string) => request<any>("DELETE", `/api/config/devices/${id}/entities/${eid}`),
  validateConfig: (body: string, isYaml: boolean) =>
    fetch(`${basePath()}/api/config/validate`, {
      method: "POST",
      headers: { "Content-Type": isYaml ? "text/yaml" : "application/json" },
      body,
    }).then((r) => r.json()),
  getRawConfig: () => request<string>("GET", "/api/config/raw"),
  putRawConfig: (yaml: string) =>
    fetch(`${basePath()}/api/config/raw`, {
      method: "PUT",
      headers: { "Content-Type": "text/yaml" },
      body: yaml,
    }).then((r) => r.json()),

  // Status
  getStatus: () => request<any>("GET", "/api/status"),
  getEntityStates: () => request<any>("GET", "/api/status/entities"),
  getGatewayStatus: (name: string) => request<any>("GET", `/api/status/gateways/${name}`),

  // MQTT Topics
  getMQTTTopics: () => request<any>("GET", "/api/mqtt/topics"),
  deleteMQTTTopics: (scope: string, deviceId?: string, entityId?: string) =>
    request<any>("DELETE", "/api/mqtt/topics", {
      scope,
      device_id: deviceId,
      entity_id: entityId,
    }),
};
