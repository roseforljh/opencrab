export type AdminChannel = {
  id: number;
  name: string;
  provider: string;
  endpoint: string;
  enabled: boolean;
  rpm_limit: number;
  max_inflight: number;
  safety_factor: number;
  enabled_for_async: boolean;
  dispatch_weight: number;
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
  cooldown_until?: string;
  last_error?: string;
};

export type AdminRequestLogSummary = {
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
  details: string;
  created_at: string;
};

export type AdminRequestLogDetail = {
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

export type AdminDashboardSummary = {
  channels_count: number;
  models_count: number;
  routes_count: number;
  api_keys_count: number;
  enabled_channels_count: number;
  default_channel: string;
  provider_count: number;
  routing_overview: AdminRoutingOverview;
  today_requests: number;
  total_requests: number;
  success_count: number;
  error_count: number;
  average_latency: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cache_hit_count: number;
  cache_hit_rate: number;
  requests_per_minute: number;
  tokens_per_minute: number;
  daily_counts: {
    label: string;
    requests: number;
    success_rate: number;
    average_latency: number;
    total_tokens: number;
  }[];
  traffic_series: {
    label: string;
    requests: number;
    success: number;
    errors: number;
  }[];
  recent_logs: {
    time: string;
    model: string;
    channel: string;
    status: string;
    latency_ms: number;
  }[];
  channel_mix: {
    label: string;
    value: number;
  }[];
  model_ranking: {
    label: string;
    value: number;
    width: number;
  }[];
  runtime_redis_enabled: boolean;
  runtime_redis_address: string;
  runtime_redis_db: number;
  runtime_redis_tls_enabled: boolean;
  runtime_redis_key_prefix: string;
  dispatch_pause: boolean;
  dispatcher_workers: number;
  queue_mode: string;
  default_queue: string;
  priority_queues: string;
  queue_ttl_s: number;
  sync_hold_ms: number;
  retry_reserve_ratio: number;
  backlog_cap: number;
  max_attempts: number;
  backoff_mode: string;
  backoff_delay_ms: number;
  dead_letter_enabled: boolean;
  metrics_enabled: boolean;
  long_wait_threshold_s: number;
  show_worker_status: boolean;
  show_queue_depth: boolean;
  show_retry_rate: boolean;
  async_enabled_channels: number;
  total_rpm_limit: number;
  total_max_inflight: number;
};

export type AdminSettingItem = {
  key: string;
  label: string;
  description: string;
  value: string;
  sensitive: boolean;
  configured: boolean;
};

export type AdminSettingGroup = {
  title: string;
  items: AdminSettingItem[];
};

export type AdminRoutingOverview = {
  active_cooldowns: number;
  sticky_bindings: number;
  sticky_hits_24h: number;
  fallbacks_24h: number;
  skipped_24h: number;
  request_count_24h: number;
  healthy_routes: number;
  total_routes: number;
  pressure_score: number;
  recent_errors: string[];
  cursor_states: {
    route_key: string;
    next_index: number;
    updated_at: string;
  }[];
};

export type AdminHealthStatus = {
  status: string;
  timestamp: string;
};

export type AdminAuthStatus = {
  initialized: boolean;
  authenticated: boolean;
};

export type AdminSecondarySecurityState = {
  enabled: boolean;
  configured: boolean;
};

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
