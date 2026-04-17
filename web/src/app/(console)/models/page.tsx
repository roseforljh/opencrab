import { ModelsClient } from "@/app/(console)/models/models-client";
import { getAdminModels, getAdminModelRoutes } from "@/lib/admin-api-server";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

export default async function ModelsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;

  const [models, routes] = await Promise.all([getAdminModels(), getAdminModelRoutes()]);
  const modelMap = new Map(models.map((item) => [item.alias, item]));

  const initialModels = models.map((item) => ({
    id: item.id,
    alias: item.alias,
    upstreamModel: item.upstream_model,
  }));

  const initialRoutes = routes.map((route) => {
    const mapping = modelMap.get(route.model_alias);
    return {
      id: route.id,
      modelId: mapping?.id,
      alias: route.model_alias,
      target: mapping?.upstream_model ?? route.model_alias,
      channel: route.channel_name,
      invocationMode: route.invocation_mode || "auto",
      priority: route.priority,
      fallback: route.fallback_model,
    };
  });

  return (
    <ModelsClient
      eyebrow={t("nav.models")}
      title={t("models.title")}
      description={t("models.description")}
      initialRoutes={initialRoutes}
      initialModels={initialModels}
    />
  );
}
