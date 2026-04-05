import { ModelsClient } from "@/app/(console)/models/models-client";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export default async function ModelsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  return <ModelsClient eyebrow={t("nav.models")} title={t("models.title")} description={t("models.description")} />;
}
