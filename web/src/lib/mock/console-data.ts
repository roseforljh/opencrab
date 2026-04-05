export const dashboardMetrics = [
  { label: "今日请求", value: "12,480", hint: "较昨日 +8.4%" },
  { label: "成功率", value: "99.92%", hint: "错误率维持低位" },
  { label: "平均耗时", value: "642 ms", hint: "高峰时段 710 ms" },
  { label: "活跃渠道", value: "3", hint: "1 个渠道处于待验证" }
];

export const dashboardRecentLogs = [
  { time: "2026-04-06 20:14:22.128", model: "gpt-4.1", channel: "openai-main", status: "成功", latency: "532 ms" },
  { time: "2026-04-06 20:13:41.002", model: "claude-3.7-sonnet", channel: "anthropic-proxy", status: "成功", latency: "811 ms" },
  { time: "2026-04-06 20:12:08.667", model: "gemini-2.5-pro", channel: "google-bridge", status: "异常", latency: "1,204 ms" }
];

export const channels = [
  { name: "openai-main", provider: "OpenAI Compatible", status: "启用", endpoint: "https://api.openai.com/v1", models: 12 },
  { name: "anthropic-proxy", provider: "Anthropic Compatible", status: "启用", endpoint: "https://proxy.example.com/anthropic", models: 5 },
  { name: "google-bridge", provider: "Gemini Compatible", status: "待验证", endpoint: "https://proxy.example.com/gemini", models: 4 }
];

export const modelRoutes = [
  { alias: "default-chat", target: "gpt-4.1", channel: "openai-main", priority: "P1", fallback: "claude-3.7-sonnet" },
  { alias: "long-context", target: "claude-3.7-sonnet", channel: "anthropic-proxy", priority: "P2", fallback: "gpt-4.1" },
  { alias: "fast-draft", target: "gemini-2.5-pro", channel: "google-bridge", priority: "P3", fallback: "gpt-4.1-mini" }
];

export const apiKeys = [
  { name: "web-console", preview: "sk-opencrab-12••••89", status: "启用", usage: "3,842 请求", lastUsed: "刚刚" },
  { name: "obs-bot", preview: "sk-opencrab-45••••33", status: "禁用", usage: "0 请求", lastUsed: "2 天前" },
  { name: "local-client", preview: "sk-opencrab-98••••17", status: "启用", usage: "1,220 请求", lastUsed: "12 分钟前" }
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
