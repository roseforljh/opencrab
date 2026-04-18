import { ApiKeysClient } from "@/app/(console)/api-keys/api-keys-client";
import { toEnabledStatus } from "@/lib/admin-api";
import { getAdminApiKeys, getAdminChannels, getAdminModels, getAdminSecondarySecurityState } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

export default async function ApiKeysPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;
  const [items, channels, models, securityState] = await Promise.all([
    getAdminApiKeys(),
    getAdminChannels(),
    getAdminModels(),
    getAdminSecondarySecurityState()
  ]);

  return (
    <ApiKeysClient
      eyebrow={t("nav.apikeys")}
      title={t("apikeys.title")}
      description={t("apikeys.description")}
      initialRows={items.map((item) => ({
        id: item.id,
        name: item.name,
        status: toEnabledStatus(item.enabled),
        channelNames: item.channel_names ?? [],
        modelAliases: item.model_aliases ?? []
      }))}
      channelOptions={channels.map((item) => item.name)}
      modelOptions={models.map((item) => item.alias)}
      requiresSecondaryPassword={securityState.enabled}
    />
  );
}

