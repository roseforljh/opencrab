import { CapabilitiesClient } from "@/app/(console)/capabilities/capabilities-client";
import { getAdminCapabilityProfiles } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

export default async function CapabilitiesPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  const payload = await getAdminCapabilityProfiles();

  return (
    <CapabilitiesClient
      eyebrow={t("nav.capabilities")}
      title={t("capabilities.title")}
      description={t("capabilities.description")}
      initialItems={payload.items}
      catalog={payload.catalog}
    />
  );
}
