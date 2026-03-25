
import React, { useMemo } from 'react';
import { Empty } from '@douyinfe/semi-ui';
import { DataTable } from '../../ui/data-table';
import { getTokensColumns } from './TokensColumnDefs';

const TokensTable = (tokensData) => {
  const {
    tokens,
    loading,
    activePage,
    pageSize,
    tokenCount,
    compactMode,
    handlePageChange,
    handlePageSizeChange,
    rowSelection,
    handleRow,
    showKeys,
    resolvedTokenKeys,
    loadingTokenKeys,
    toggleTokenVisibility,
    copyTokenKey,
    manageToken,
    onOpenLink,
    setEditingToken,
    setShowEdit,
    refresh,
    t,
  } = tokensData;

  // Get all columns
  const columns = useMemo(() => {
    return getTokensColumns({
      t,
      showKeys,
      resolvedTokenKeys,
      loadingTokenKeys,
      toggleTokenVisibility,
      copyTokenKey,
      manageToken,
      onOpenLink,
      setEditingToken,
      setShowEdit,
      refresh,
    });
  }, [
    t,
    showKeys,
    resolvedTokenKeys,
    loadingTokenKeys,
    toggleTokenVisibility,
    copyTokenKey,
    manageToken,
    onOpenLink,
    setEditingToken,
    setShowEdit,
    refresh,
  ]);

  // Handle compact mode by removing fixed positioning
  const tableColumns = useMemo(() => {
    return compactMode
      ? columns.map((col) => {
          if (col.dataIndex === 'operate') {
            const { fixed, ...rest } = col;
            return rest;
          }
          return col;
        })
      : columns;
  }, [compactMode, columns]);

  return (
    <div className='flex flex-col gap-4'>
      <DataTable
        columns={tableColumns}
        data={tokens}
        loading={loading}
        emptyMessage={
          <Empty
            image={
              <div className='flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
                ⌕
              </div>
            }
            title={t('未找到匹配项')}
            description={t('换个关键词，或者放宽筛选条件后再试一次。')}
            style={{ padding: 30 }}
          />
        }
      />
    </div>
  );
};

export default TokensTable;
