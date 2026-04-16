import { ApiKeysClient } from "@/app/(console)/api-keys/api-keys-client";
import { toEnabledStatus } from "@/lib/admin-api";
import { getAdminApiKeys, getAdminSecondarySecurityState } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

export default async function ApiKeysPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;
  const [items, securityState] = await Promise.all([getAdminApiKeys(), getAdminSecondarySecurityState()]);

  return (
    <ApiKeysClient
      eyebrow={t("nav.apikeys")}
      title={t("apikeys.title")}
      description={t("apikeys.description")}
      initialRows={items.map((item) => ({ id: item.id, name: item.name, status: toEnabledStatus(item.enabled) }))}
      requiresSecondaryPassword={securityState.enabled}
    />
  );
}
