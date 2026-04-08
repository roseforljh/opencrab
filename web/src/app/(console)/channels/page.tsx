import { PageContainer } from "@/components/layout/page-container";
import { ChannelsTable } from "@/app/(console)/channels/channels-table";
import { NewChannelForm } from "@/app/(console)/channels/new-channel-form";
import { PageHeader } from "@/components/layout/page-header";
import { DetailDrawer } from "@/components/shared/detail-drawer";
import { ErrorState } from "@/components/shared/error-state";
import { SectionCard } from "@/components/shared/section-card";
import { Button } from "@/components/ui/button";
import { getAdminChannels, getAdminModels, getAdminModelRoutes, toEnabledStatus } from "@/lib/admin-api";
import { getDictionary } from "@/lib/i18n-shared";
import { getServerLanguage } from "@/lib/i18n-server";

export const dynamic = "force-dynamic";

type ChannelRow = {
  id: number;
  name: string;
  provider: string;
  status: string;
  endpoint: string;
  models: number;
  modelIds: string[];
  updatedAt: string;
};

export default async function ChannelsPage() {
  const language = await getServerLanguage();
  const dictionary = getDictionary(language);
  const t = (key: string) => dictionary[key] ?? key;
  let rows: ChannelRow[] = [];
  let loadError: string | null = null;

  try {
    const [channels, models, routes] = await Promise.all([getAdminChannels(), getAdminModels(), getAdminModelRoutes()]);
    const modelMap = new Map(models.map((item) => [item.alias, item.upstream_model]));
    rows = channels.map((channel) => {
      const relatedRoutes = routes.filter((route) => route.channel_name === channel.name);
      const modelIds = Array.from(new Set(relatedRoutes.map((route) => modelMap.get(route.model_alias) ?? route.model_alias)));

      return {
        id: channel.id,
        name: channel.name,
        provider: channel.provider,
        status: toEnabledStatus(channel.enabled),
        endpoint: channel.endpoint,
        models: modelIds.length,
        modelIds,
        updatedAt: channel.updated_at
      };
    });
  } catch (error) {
    loadError = error instanceof Error ? error.message : "渠道数据加载失败";
  }

  return (
    <PageContainer>
      <PageHeader
        eyebrow={t("nav.channels")}
        title={t("channels.title")}
        description={t("channels.description")}
        action={
          <DetailDrawer
            title="新建渠道"
            description="新增一个上游模型渠道，填写兼容类型、请求地址和默认认证信息。"
            triggerLabel={t("common.create")}
            trigger={<Button>{t("common.create")}</Button>}
          >
            <NewChannelForm />
          </DetailDrawer>
        }
      />

      <SectionCard title="渠道列表" description="这一页采用筛选条 + 表格区 + 右侧编辑抽屉的标准模式。">
        {loadError ? <ErrorState title="渠道数据加载失败" description={loadError} /> : null}
        {!loadError ? <ChannelsTable rows={rows} /> : null}
      </SectionCard>
    </PageContainer>
  );
}
