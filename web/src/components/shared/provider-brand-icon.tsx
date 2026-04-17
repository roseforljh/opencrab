import Image from "next/image";

import { cn } from "@/lib/utils";

type BrandAsset = {
  alt: string;
  src: string;
  tint?: boolean;
};

const providerIconMap: Record<string, BrandAsset> = {
  Gemini: { alt: "Gemini", src: "/brands/gemini.svg" },
  OpenAI: { alt: "OpenAI", src: "/brands/openai.svg", tint: true },
  Claude: { alt: "Claude", src: "/brands/claude.svg", tint: true },
  GLM: { alt: "GLM", src: "/brands/zhipu.svg" },
  KIMI: { alt: "Kimi", src: "/brands/moonshot.svg", tint: true },
  MiniMAX: { alt: "MiniMAX", src: "/brands/minimax.svg" },
  OpenRouter: { alt: "OpenRouter", src: "/brands/openrouter.svg", tint: true }
};

export function ProviderBrandIcon({
  provider,
  size = 18,
  boxed = true
}: {
  provider: string;
  size?: number;
  boxed?: boolean;
}) {
  const icon = providerIconMap[provider] ?? providerIconMap.OpenAI;

  const image = (
    <Image
      src={icon.src}
      alt={icon.alt}
      width={size}
      height={size}
      unoptimized
      className={cn("shrink-0 object-contain", icon.tint ? "dark:invert dark:brightness-0" : "")}
    />
  );

  if (!boxed) {
    return image;
  }

  return (
    <span className="inline-flex h-6 w-6 items-center justify-center overflow-hidden rounded-md border border-border/60 bg-card/70 text-foreground">
      {image}
    </span>
  );
}
