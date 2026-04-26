"use client";

import { formatDateTime } from "@/lib/admin-api";

export function LocalDateTime({ value }: { value: string }) {
  return <span suppressHydrationWarning>{formatDateTime(value)}</span>;
}
