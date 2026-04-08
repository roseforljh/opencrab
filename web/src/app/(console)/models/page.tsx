import { ModelsClient } from "@/app/(console)/models/models-client";
import { getAdminChannels, getAdminModels, getAdminModelRoutes } from "@/lib/admin-api";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

export default async function ModelsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;
	const [models, routes, channels] = await Promise.all([getAdminModels(), getAdminModelRoutes(), getAdminChannels()]);
	const modelMap = new Map(models.map((item) => [item.alias, item]));
	const initialRoutes = routes.map((route) => {
		const mapping = modelMap.get(route.model_alias);
		return {
			id: route.id,
			modelId: mapping?.id,
			alias: route.model_alias,
			target: mapping?.upstream_model ?? route.model_alias,
			channel: route.channel_name,
			priority: `P${route.priority}`,
			fallback: route.fallback_model
		};
	});

  return <ModelsClient eyebrow={t("nav.models")} title={t("models.title")} description={t("models.description")} initialRoutes={initialRoutes} channelNames={channels.map((item) => item.name)} />;
}
