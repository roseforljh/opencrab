
import React from 'react';
import TokensTable from '../../components/table/tokens';

const Token = () => {
  return (
    <div className='space-y-4'>
      <div className='rounded-2xl border border-semi-color-border bg-semi-color-bg-1 px-5 py-4 md:px-6'>
        <div className='flex items-start justify-between gap-4'>
          <div>
            <h1 className='text-xl font-semibold text-gray-900 dark:text-gray-100'>
              令牌管理
            </h1>
            <p className='mt-1 text-sm text-gray-500 dark:text-gray-400'>
              在这里集中创建、筛选、复制和清理访问令牌，优先处理高频可见、低风险误操作的管理流程。
            </p>
          </div>
        </div>
      </div>

      <TokensTable />
    </div>
  );
};

export default Token;
