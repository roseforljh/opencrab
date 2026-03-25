
import React, { useMemo } from 'react';
import { Empty } from '@douyinfe/semi-ui';
import { DataTable } from '../../ui/data-table';
import { getModelsColumns } from './ModelsColumnDefs';

const ModelsTable = (modelsData) => {
  const {
    models,
    loading,
    activePage,
    pageSize,
    modelCount,
    compactMode,
    handlePageChange,
    handlePageSizeChange,
    rowSelection,
    handleRow,
    manageModel,
    setEditingModel,
    setShowEdit,
    refresh,
    vendorMap,
    t,
  } = modelsData;

  // Get all columns
  const columns = useMemo(() => {
    return getModelsColumns({
      t,
      manageModel,
      setEditingModel,
      setShowEdit,
      refresh,
      vendorMap,
    });
  }, [t, manageModel, setEditingModel, setShowEdit, refresh, vendorMap]);

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
        data={models}
        loading={loading}
        emptyMessage={
          <Empty
            image={
              <div className='flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
                ⌕
              </div>
            }
            title={t('暂无匹配模型')}
            description={t('可以尝试供应商名称、模型关键字，或直接清空筛选。')}
            style={{ padding: 30 }}
          />
        }
      />
    </div>
  );
};

export default ModelsTable;
