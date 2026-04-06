"use client";

import { Claude, Gemini, Minimax, Moonshot, OpenAI, OpenRouter, Zhipu } from "@lobehub/icons";

const providerIconMap = {
  Gemini,
  OpenAI,
  Claude,
  GLM: Zhipu,
  KIMI: Moonshot,
  MiniMAX: Minimax,
  OpenRouter
} as const;

export function ProviderBrandIcon({
  provider,
  size = 18,
  boxed = true
}: {
  provider: string;
  size?: number;
  boxed?: boolean;
}) {
  const Icon = providerIconMap[provider as keyof typeof providerIconMap] ?? OpenAI;

  if (!boxed) {
    return <Icon size={size} />;
  }

  return (
    <span className="inline-flex h-6 w-6 items-center justify-center overflow-hidden rounded-md bg-card/70">
      <Icon size={size} />
    </span>
  );
}
