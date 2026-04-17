import Image from "next/image";

import { cn } from "@/lib/utils";

type BrandAsset = {
  alt: string;
  src: string;
  tint?: boolean;
  boxedClassName?: string;
  imageClassName?: string;
  imageSize?: number;
};

const providerIconMap: Record<string, BrandAsset> = {
  Gemini: { alt: "Gemini", src: "/brands/gemini.svg", tint: true },
  OpenAI: { alt: "OpenAI", src: "/brands/openai.svg", tint: true },
  Claude: { alt: "Claude", src: "/brands/claude.svg", tint: true },
  GLM: { alt: "GLM", src: "/brands/zhipu.svg" },
  KIMI: { alt: "Kimi", src: "/brands/kimi-favicon-preview.png", boxedClassName: "h-10 w-10 bg-[#111213]", imageSize: 32 },
  MiniMAX: { alt: "MiniMAX", src: "/brands/minimax.svg", boxedClassName: "h-8 w-8 bg-white", imageClassName: "scale-[1.18]", imageSize: 22 },
  OpenRouter: { alt: "OpenRouter", src: "/brands/openrouter.svg", tint: true }
};

export function ProviderBrandIcon({
  provider,
  size = 20,
  boxed = true
}: {
  provider: string;
  size?: number;
  boxed?: boolean;
}) {
  const icon = providerIconMap[provider] ?? providerIconMap.OpenAI;
  const actualSize = icon.imageSize ?? size;

  const image = (
    <Image
      src={icon.src}
      alt={icon.alt}
      width={actualSize}
      height={actualSize}
      unoptimized
      className={cn("shrink-0 object-contain", icon.tint ? "dark:invert dark:brightness-0" : "", icon.imageClassName)}
    />
  );

  if (!boxed) {
    return image;
  }

  return (
    <span className={cn("inline-flex h-7 w-7 items-center justify-center overflow-hidden rounded-lg border border-border/80 bg-secondary/55 text-foreground", icon.boxedClassName)}>
      {image}
    </span>
  );
}
