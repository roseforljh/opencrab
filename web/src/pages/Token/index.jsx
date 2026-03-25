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
