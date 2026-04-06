"use client";

import { ProviderBrandIcon } from "@/components/shared/provider-brand-icon";
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select";

const providers = [
  { value: "Gemini" },
  { value: "OpenAI" },
  { value: "Claude" },
  { value: "GLM" },
  { value: "KIMI" },
  { value: "MiniMAX" },
  { value: "OpenRouter" }
] as const;

export function ProviderSelect({ value, onValueChange }: { value: string; onValueChange: (value: string) => void }) {
  const active = providers.find((item) => item.value === value) ?? providers[1];

  return (
    <Select value={active.value} onValueChange={onValueChange}>
      <SelectTrigger>
        <div className="flex items-center gap-3">
          <ProviderBrandIcon provider={active.value} />
          <span>{active.value}</span>
        </div>
      </SelectTrigger>
      <SelectContent>
        {providers.map((provider) => (
          <SelectItem key={provider.value} value={provider.value}>
            <div className="flex items-center gap-3 pr-6">
              <ProviderBrandIcon provider={provider.value} />
              <span>{provider.value}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
