
import React, { useMemo } from 'react';
import { Empty } from '@douyinfe/semi-ui';
import { DataTable } from '../../ui/data-table';
import { getChannelsColumns } from './ChannelsColumnDefs';

const ChannelsTable = (channelsData) => {
  const {
    channels,
    loading,
    searching,
    activePage,
    pageSize,
    channelCount,
    enableBatchDelete,
    compactMode,
    visibleColumns,
    setSelectedChannels,
    handlePageChange,
    handlePageSizeChange,
    handleRow,
    t,
    COLUMN_KEYS,
    // Column functions and data
    updateChannelBalance,
    manageChannel,
    manageTag,
    submitTagEdit,
    testChannel,
    setCurrentTestChannel,
    setShowModelTestModal,
    setEditingChannel,
    setShowEdit,
    setShowEditTag,
    setEditingTag,
    copySelectedChannel,
    refresh,
    checkOllamaVersion,
    // Multi-key management
    setShowMultiKeyManageModal,
    setCurrentMultiKeyChannel,
    openUpstreamUpdateModal,
    detectChannelUpstreamUpdates,
  } = channelsData;

  // Get all columns
  const allColumns = useMemo(() => {
    return getChannelsColumns({
      t,
      COLUMN_KEYS,
      updateChannelBalance,
      manageChannel,
      manageTag,
      submitTagEdit,
      testChannel,
      setCurrentTestChannel,
      setShowModelTestModal,
      setEditingChannel,
      setShowEdit,
      setShowEditTag,
      setEditingTag,
      copySelectedChannel,
      refresh,
      activePage,
      channels,
      checkOllamaVersion,
      setShowMultiKeyManageModal,
      setCurrentMultiKeyChannel,
      openUpstreamUpdateModal,
      detectChannelUpstreamUpdates,
    });
  }, [
    t,
    COLUMN_KEYS,
    updateChannelBalance,
    manageChannel,
    manageTag,
    submitTagEdit,
    testChannel,
    setCurrentTestChannel,
    setShowModelTestModal,
    setEditingChannel,
    setShowEdit,
    setShowEditTag,
    setEditingTag,
    copySelectedChannel,
    refresh,
    activePage,
    channels,
    checkOllamaVersion,
    setShowMultiKeyManageModal,
    setCurrentMultiKeyChannel,
    openUpstreamUpdateModal,
    detectChannelUpstreamUpdates,
  ]);

  // Filter columns based on visibility settings
  const getVisibleColumns = () => {
    return allColumns.filter((column) => visibleColumns[column.key]);
  };

  const visibleColumnsList = useMemo(() => {
    return getVisibleColumns();
  }, [visibleColumns, allColumns]);

  const tableColumns = useMemo(() => {
    return compactMode
      ? visibleColumnsList
      : visibleColumnsList;
  }, [compactMode, visibleColumnsList]);

  return (
    <div className='flex flex-col gap-4'>
      <DataTable
        columns={tableColumns}
        data={channels}
        loading={loading || searching}
        emptyMessage={
          <Empty
            image={
              <div className='flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
                ⌕
              </div>
            }
            title={t('暂无命中渠道')}
            description={t('可以尝试地址关键字、模型关键字，或切换筛选状态后再试。')}
            style={{ padding: 30 }}
          />
        }
      />
    </div>
  );
};

export default ChannelsTable;
