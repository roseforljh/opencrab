import { ApiKeysClient } from "@/app/(console)/api-keys/api-keys-client";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";
import { apiKeys } from "@/lib/mock/console-data";

export default async function ApiKeysPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  return (
    <ApiKeysClient
      eyebrow={t("nav.apikeys")}
      title={t("apikeys.title")}
      description={t("apikeys.description")}
      initialRows={apiKeys}
    />
  );
}
