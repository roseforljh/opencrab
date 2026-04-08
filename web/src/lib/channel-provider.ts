export const CHANNEL_PROVIDERS = ["Gemini", "OpenAI", "Claude", "GLM", "KIMI", "MiniMAX", "OpenRouter"] as const;

export type ChannelProvider = (typeof CHANNEL_PROVIDERS)[number];

export const CHANNEL_PROVIDER_DEFAULT_ENDPOINT: Record<ChannelProvider, string> = {
  Gemini: "https://generativelanguage.googleapis.com/v1beta",
  OpenAI: "https://api.openai.com/v1",
  Claude: "https://api.anthropic.com",
  GLM: "https://open.bigmodel.cn/api/paas/v4",
  KIMI: "https://api.moonshot.cn/v1",
  MiniMAX: "https://api.minimax.chat",
  OpenRouter: "https://openrouter.ai/api/v1"
};

export function getDefaultEndpointForProvider(provider: string) {
  if (provider in CHANNEL_PROVIDER_DEFAULT_ENDPOINT) {
    return CHANNEL_PROVIDER_DEFAULT_ENDPOINT[provider as ChannelProvider];
  }

  return CHANNEL_PROVIDER_DEFAULT_ENDPOINT.OpenAI;
}
