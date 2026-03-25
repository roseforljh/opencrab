
import React from 'react';
import { Banner } from '@douyinfe/semi-ui';
import { IconAlertTriangle } from '@douyinfe/semi-icons';
import CardPro from '../../common/ui/CardPro';
import ChannelsTable from './ChannelsTable';
import ChannelsActions from './ChannelsActions';
import ChannelsFilters from './ChannelsFilters';
import ChannelsTabs from './ChannelsTabs';
import { useChannelsData } from '../../../hooks/channels/useChannelsData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import BatchTagModal from './modals/BatchTagModal';
import ModelTestModal from './modals/ModelTestModal';
import ColumnSelectorModal from './modals/ColumnSelectorModal';
import EditChannelModal from './modals/EditChannelModal';
import EditTagModal from './modals/EditTagModal';
import MultiKeyManageModal from './modals/MultiKeyManageModal';
import ChannelUpstreamUpdateModal from './modals/ChannelUpstreamUpdateModal';
import { createCardProPagination } from '../../../helpers/utils';

const ChannelsPage = () => {
  const channelsData = useChannelsData();
  const isMobile = useIsMobile();

  return (
    <>
      {/* Modals */}
      <ColumnSelectorModal {...channelsData} />
      <EditTagModal
        visible={channelsData.showEditTag}
        tag={channelsData.editingTag}
        handleClose={() => channelsData.setShowEditTag(false)}
        refresh={channelsData.refresh}
      />
      <EditChannelModal
        refresh={channelsData.refresh}
        visible={channelsData.showEdit}
        handleClose={channelsData.closeEdit}
        editingChannel={channelsData.editingChannel}
      />
      <BatchTagModal {...channelsData} />
      <ModelTestModal {...channelsData} />
      <MultiKeyManageModal
        visible={channelsData.showMultiKeyManageModal}
        onCancel={() => channelsData.setShowMultiKeyManageModal(false)}
        channel={channelsData.currentMultiKeyChannel}
        onRefresh={channelsData.refresh}
      />
      <ChannelUpstreamUpdateModal
        visible={channelsData.showUpstreamUpdateModal}
        addModels={channelsData.upstreamUpdateAddModels}
        removeModels={channelsData.upstreamUpdateRemoveModels}
        preferredTab={channelsData.upstreamUpdatePreferredTab}
        confirmLoading={channelsData.upstreamApplyLoading}
        onConfirm={channelsData.applyUpstreamUpdates}
        onCancel={channelsData.closeUpstreamUpdateModal}
      />

      {/* Main Content */}
      {channelsData.globalPassThroughEnabled ? (
        <Banner
          type='warning'
          closeIcon={null}
          icon={
            <IconAlertTriangle
              size='large'
              style={{ color: '#ffcf5a' }}
            />
          }
          description={channelsData.t(
            '全局请求直通已启用：参数覆写、模型重定向、渠道适配等内置能力将不再生效，这并非推荐方案；若因此产生异常，请勿提交 issue。',
          )}
          className='!mb-3 !rounded-[24px] !border !border-yellow-400/20 !bg-[rgba(120,88,20,0.28)] !text-white'
        />
      ) : null}
      <CardPro
        type='type3'
        tabsArea={<ChannelsTabs {...channelsData} />}
        actionsArea={<ChannelsActions {...channelsData} />}
        searchArea={<ChannelsFilters {...channelsData} />}
        paginationArea={createCardProPagination({
          currentPage: channelsData.activePage,
          pageSize: channelsData.pageSize,
          total: channelsData.channelCount,
          onPageChange: channelsData.handlePageChange,
          onPageSizeChange: channelsData.handlePageSizeChange,
          isMobile: isMobile,
          t: channelsData.t,
        })}
        t={channelsData.t}
      >
        <ChannelsTable {...channelsData} />
      </CardPro>
    </>
  );
};

export default ChannelsPage;
