export const dashboardMetrics = [
  { label: "今日请求", value: "12,480", hint: "较昨日 +8.4%", trend: [22, 24, 21, 29, 35, 32, 38] },
  { label: "成功率", value: "99.92%", hint: "错误率维持低位", trend: [98, 98.8, 99.1, 99.4, 99.2, 99.6, 99.9] },
  { label: "平均耗时", value: "642 ms", hint: "高峰时段 710 ms", trend: [710, 688, 702, 674, 651, 644, 642] },
  { label: "活跃渠道", value: "3", hint: "1 个渠道处于待验证", trend: [1, 2, 2, 3, 3, 3, 3] }
];

export const dashboardTrafficSeries = [
  { label: "00:00", requests: 320, success: 302, errors: 18 },
  { label: "04:00", requests: 410, success: 398, errors: 12 },
  { label: "08:00", requests: 860, success: 831, errors: 29 },
  { label: "12:00", requests: 1180, success: 1140, errors: 40 },
  { label: "16:00", requests: 980, success: 944, errors: 36 },
  { label: "20:00", requests: 1240, success: 1196, errors: 44 }
];

export const dashboardSystemStatus = [
  { label: "运行时长", value: "6 天 07 小时", accent: "var(--chart-1)" },
  { label: "并发请求", value: "51", accent: "var(--chart-5)" },
  { label: "活跃模型", value: "12", accent: "var(--chart-2)" },
  { label: "系统版本", value: "构建 0.1.0-beta", accent: "var(--chart-4)" }
];

export const dashboardNetworkStatus = [
  { label: "OpenAI 延迟", value: "213 ms", accent: "var(--chart-4)" },
  { label: "Anthropic 延迟", value: "48 ms", accent: "var(--chart-5)" },
  { label: "Gemini 延迟", value: "96 ms", accent: "var(--chart-2)" },
  { label: "默认网关", value: "openai-main", accent: "var(--chart-1)" }
];

export const dashboardWeeklyTraffic = [
  { label: "Mon", value: 180 },
  { label: "Tue", value: 520 },
  { label: "Wed", value: 870 },
  { label: "Thu", value: 920 },
  { label: "Fri", value: 460 },
  { label: "Sat", value: 90 },
  { label: "Sun", value: 35 }
];

export const dashboardRanking = [
  { label: "gpt-4.1", value: "17.9 万 tokens", width: 100 },
  { label: "claude-3.7-sonnet", value: "14.9 万 tokens", width: 82 },
  { label: "gemini-2.5-pro", value: "7.3 万 tokens", width: 55 },
  { label: "gpt-4.1-mini", value: "2.5 万 tokens", width: 22 },
  { label: "text-embedding-3-large", value: "1.3 万 tokens", width: 11 }
];

export const dashboardTrafficSummary = {
  total: "49.3 万 tokens",
  upload: "16.3 万 prompt",
  download: "33.1 万 completion",
  direct: "2.5 万 direct",
  proxy: "46.8 万 routed"
};

export const dashboardChannelMix = [
  { label: "OpenAI", value: 48, color: "var(--chart-1)" },
  { label: "Anthropic", value: 27, color: "var(--chart-2)" },
  { label: "Gemini", value: 17, color: "var(--chart-3)" },
  { label: "Fallback", value: 8, color: "var(--chart-4)" }
];

export const dashboardRecentLogs = [
  { time: "2026-04-06 20:14:22.128", model: "gpt-4.1", channel: "openai-main", status: "成功", latency: "532 ms" },
  { time: "2026-04-06 20:13:41.002", model: "claude-3.7-sonnet", channel: "anthropic-proxy", status: "成功", latency: "811 ms" },
  { time: "2026-04-06 20:12:08.667", model: "gemini-2.5-pro", channel: "google-bridge", status: "异常", latency: "1,204 ms" }
];

export const channels = [
  {
    name: "openai-main",
    provider: "OpenAI Compatible",
    status: "启用",
    endpoint: "https://api.openai.com/v1",
    models: 12,
    modelIds: ["gpt-4.1", "gpt-4.1-mini", "o3-mini", "text-embedding-3-large"]
  },
  {
    name: "anthropic-proxy",
    provider: "Anthropic Compatible",
    status: "启用",
    endpoint: "https://proxy.example.com/anthropic",
    models: 5,
    modelIds: ["claude-3.7-sonnet", "claude-3.5-haiku", "claude-3-opus"]
  },
  {
    name: "google-bridge",
    provider: "Gemini Compatible",
    status: "待验证",
    endpoint: "https://proxy.example.com/gemini",
    models: 4,
    modelIds: ["gemini-2.5-pro", "gemini-2.0-flash", "embedding-001"]
  }
];

export const modelRoutes = [
  { alias: "default-chat", target: "gpt-4.1", channel: "openai-main", priority: "P1", fallback: "claude-3.7-sonnet" },
  { alias: "long-context", target: "claude-3.7-sonnet", channel: "anthropic-proxy", priority: "P2", fallback: "gpt-4.1" },
  { alias: "fast-draft", target: "gemini-2.5-pro", channel: "google-bridge", priority: "P3", fallback: "gpt-4.1-mini" }
];

export const apiKeys = [
  { name: "web-console", rawKey: "sk-opencrab-web-console-12ab34cd56ef78gh", preview: "sk-opencrab-12••••89", status: "启用", usage: "3,842 请求", lastUsed: "刚刚" },
  { name: "obs-bot", rawKey: "sk-opencrab-obs-bot-45xy67zt89uv33aa", preview: "sk-opencrab-45••••33", status: "禁用", usage: "0 请求", lastUsed: "2 天前" },
  { name: "local-client", rawKey: "sk-opencrab-local-client-98mn45pq17rs22tt", preview: "sk-opencrab-98••••17", status: "启用", usage: "1,220 请求", lastUsed: "12 分钟前" }
];

export const logRows = [
  { requestId: "req_01JQ9N87AB2", time: "2026-04-06 20:14:22.128", model: "gpt-4.1", channel: "openai-main", status: "200", latency: "532 ms" },
  { requestId: "req_01JQ9N44TR8", time: "2026-04-06 20:13:41.002", model: "claude-3.7-sonnet", channel: "anthropic-proxy", status: "200", latency: "811 ms" },
  { requestId: "req_01JQ9MY4FS0", time: "2026-04-06 20:12:08.667", model: "gemini-2.5-pro", channel: "google-bridge", status: "502", latency: "1,204 ms" },
  { requestId: "req_01JQ9MTD9B1", time: "2026-04-06 20:11:11.100", model: "gpt-4.1-mini", channel: "openai-main", status: "200", latency: "418 ms" }
];

export const settingsGroups = [
  {
    title: "基础设置",
    items: [
      { label: "服务名称", value: "OpenCrab Personal Gateway" },
      { label: "默认超时", value: "60 秒" },
      { label: "默认日志保留", value: "7 天" }
    ]
  },
  {
    title: "运行策略",
    items: [
      { label: "最大并发数", value: "128" },
      { label: "流式中断释放", value: "启用" },
      { label: "错误脱敏", value: "启用" }
    ]
  }
];

export const dashboardStates = {
  loading: false,
  hasErrors: false
};
