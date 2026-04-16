import { SettingsClient } from "@/app/(console)/settings/settings-client";
import { getAdminSecondarySecurityState, getAdminSettings } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

export default async function SettingsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;
  const [groups, securityState] = await Promise.all([getAdminSettings(), getAdminSecondarySecurityState()]);

  return <SettingsClient eyebrow={t("nav.settings")} title={t("settings.title")} description={t("settings.description")} initialGroups={groups} initialSecurityState={securityState} />;
}
