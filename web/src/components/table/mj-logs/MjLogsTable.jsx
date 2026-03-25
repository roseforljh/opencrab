/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useMemo } from 'react';
import { Empty } from '@douyinfe/semi-ui';
import CardTable from '../../common/ui/CardTable';
import { getMjLogsColumns } from './MjLogsColumnDefs';

const MjLogsTable = (mjLogsData) => {
  const {
    logs,
    loading,
    activePage,
    pageSize,
    logCount,
    compactMode,
    visibleColumns,
    handlePageChange,
    handlePageSizeChange,
    copyText,
    openContentModal,
    openImageModal,
    isAdminUser,
    t,
    COLUMN_KEYS,
  } = mjLogsData;

  // Get all columns
  const allColumns = useMemo(() => {
    return getMjLogsColumns({
      t,
      COLUMN_KEYS,
      copyText,
      openContentModal,
      openImageModal,
      isAdminUser,
    });
  }, [t, COLUMN_KEYS, copyText, openContentModal, openImageModal, isAdminUser]);

  // Filter columns based on visibility settings
  const getVisibleColumns = () => {
    return allColumns.filter((column) => visibleColumns[column.key]);
  };

  const visibleColumnsList = useMemo(() => {
    return getVisibleColumns();
  }, [visibleColumns, allColumns]);

  const tableColumns = useMemo(() => {
    return compactMode
      ? visibleColumnsList.map(({ fixed, ...rest }) => rest)
      : visibleColumnsList;
  }, [compactMode, visibleColumnsList]);

  return (
    <CardTable
      columns={tableColumns}
      dataSource={logs}
      rowKey='key'
      loading={loading}
      scroll={compactMode ? undefined : { x: 'max-content' }}
      className='rounded-xl overflow-hidden'
      size='middle'
      empty={
        <Empty
          image={
            <div className='flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
              ⌕
            </div>
          }
          title={t('没有绘图记录')}
          description={t('试着缩小时间范围，或检查任务状态筛选是否过严。')}
          style={{ padding: 30 }}
        />
      }
      pagination={{
        currentPage: activePage,
        pageSize: pageSize,
        total: logCount,
        pageSizeOptions: [10, 20, 50, 100],
        showSizeChanger: true,
        onPageSizeChange: handlePageSizeChange,
        onPageChange: handlePageChange,
      }}
      hidePagination={true}
    />
  );
};

export default MjLogsTable;
