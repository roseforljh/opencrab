const ADMIN_API_BASE = process.env.OPENCRAB_ADMIN_API_BASE ?? "http://127.0.0.1:8080";

type ListResponse<T> = {
  items: T[];
};

export type AdminChannel = {
  id: number;
  name: string;
  provider: string;
  endpoint: string;
  enabled: boolean;
  updated_at: string;
};

export type AdminApiKey = {
  id: number;
  name: string;
  enabled: boolean;
};

export type AdminCreatedApiKey = {
  id: number;
  name: string;
  raw_key: string;
  enabled: boolean;
};

export type AdminModel = {
  id: number;
  alias: string;
  upstream_model: string;
};

export type AdminModelRoute = {
  id: number;
  model_alias: string;
  channel_name: string;
  invocation_mode?: string;
  priority: number;
  fallback_model: string;
};

export type AdminRequestLog = {
  id: number;
  request_id: string;
  model: string;
  channel: string;
  status_code: number;
  latency_ms: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cache_hit: boolean;
  request_body: string;
  response_body: string;
  details: string;
  created_at: string;
};

export type AdminSettingItem = {
  key: string;
  label: string;
  description: string;
  value: string;
};

export type AdminSettingGroup = {
  title: string;
  items: AdminSettingItem[];
};

export type AdminHealthStatus = {
  status: string;
  timestamp: string;
};

async function adminFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${ADMIN_API_BASE}${path}`, {
    ...init,
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    }
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `请求失败: ${response.status}`);
  }

  return response.json() as Promise<T>;
}

export async function getAdminChannels() {
  const response = await adminFetch<ListResponse<AdminChannel>>("/api/admin/channels");
  return response.items;
}

export async function getAdminApiKeys() {
  const response = await adminFetch<ListResponse<AdminApiKey>>("/api/admin/api-keys");
  return response.items;
}

export async function getAdminModels() {
  const response = await adminFetch<ListResponse<AdminModel>>("/api/admin/models");
  return response.items;
}

export async function getAdminModelRoutes() {
  const response = await adminFetch<ListResponse<AdminModelRoute>>("/api/admin/model-routes");
  return response.items;
}

export async function getAdminLogs() {
  const response = await adminFetch<ListResponse<AdminRequestLog>>("/api/admin/logs");
  return response.items;
}

export async function getAdminSettings() {
  const response = await adminFetch<ListResponse<AdminSettingGroup>>("/api/admin/settings");
  return response.items;
}

export async function getAdminHealth() {
  return adminFetch<AdminHealthStatus>("/healthz");
}

export async function getAdminReadiness() {
  return adminFetch<AdminHealthStatus>("/readyz");
}

export function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false
  }).format(date);
}

export function formatNumber(value: number) {
  return new Intl.NumberFormat("zh-CN").format(value);
}

export function formatPercent(value: number) {
  return `${value.toFixed(2)}%`;
}

export function formatLatency(value: number) {
  return `${formatNumber(value)} ms`;
}

export function toEnabledStatus(enabled: boolean) {
  return enabled ? "启用" : "禁用";
}
