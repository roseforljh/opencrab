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
import { Empty, Descriptions } from '@douyinfe/semi-ui';
import CardTable from '../../common/ui/CardTable';
import { getLogsColumns } from './UsageLogsColumnDefs';

const LogsTable = (logsData) => {
  const {
    logs,
    expandData,
    loading,
    activePage,
    pageSize,
    logCount,
    compactMode,
    visibleColumns,
    handlePageChange,
    handlePageSizeChange,
    copyText,
    showUserInfoFunc,
    openChannelAffinityUsageCacheModal,
    hasExpandableRows,
    isAdminUser,
    billingDisplayMode,
    t,
    COLUMN_KEYS,
  } = logsData;

  // Get all columns
  const allColumns = useMemo(() => {
    return getLogsColumns({
      t,
      COLUMN_KEYS,
      copyText,
      showUserInfoFunc,
      openChannelAffinityUsageCacheModal,
      isAdminUser,
      billingDisplayMode,
    });
  }, [
    t,
    COLUMN_KEYS,
    copyText,
    showUserInfoFunc,
    openChannelAffinityUsageCacheModal,
    isAdminUser,
    billingDisplayMode,
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
      ? visibleColumnsList.map(({ fixed, ...rest }) => rest)
      : visibleColumnsList;
  }, [compactMode, visibleColumnsList]);

  const expandRowRender = (record, index) => {
    return <Descriptions data={expandData[record.key]} />;
  };

  return (
    <CardTable
      columns={tableColumns}
      {...(hasExpandableRows() && {
        expandedRowRender: expandRowRender,
        expandRowByClick: true,
        rowExpandable: (record) =>
          expandData[record.key] && expandData[record.key].length > 0,
      })}
      dataSource={logs}
      rowKey='key'
      loading={loading}
      scroll={compactMode ? undefined : { x: 'max-content' }}
      className='rounded-xl overflow-hidden'
      size='small'
      empty={
        <Empty
          image={
            <div className='flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
              ⌕
            </div>
          }
          title={t('没有命中记录')}
          description={t('试试缩短时间范围、减少筛选项，或检查关键词是否准确。')}
          style={{ padding: 30 }}
        />
      }
      pagination={{
        currentPage: activePage,
        pageSize: pageSize,
        total: logCount,
        pageSizeOptions: [10, 20, 50, 100],
        showSizeChanger: true,
        onPageSizeChange: (size) => {
          handlePageSizeChange(size);
        },
        onPageChange: handlePageChange,
      }}
      hidePagination={true}
    />
  );
};

export default LogsTable;
