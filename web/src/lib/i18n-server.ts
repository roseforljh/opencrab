import { cookies } from "next/headers";

import type { Language } from "@/lib/i18n-shared";

export async function getServerLanguage(): Promise<Language> {
  const cookieStore = await cookies();
  const lang = cookieStore.get("opencrab-ui-lang")?.value;
  return lang === "en-US" ? "en-US" : "zh-CN";
}
