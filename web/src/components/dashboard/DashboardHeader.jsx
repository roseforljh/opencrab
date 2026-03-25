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
import { Button } from '@douyinfe/semi-ui';
import { RefreshCw, Search } from 'lucide-react';

const DashboardHeader = ({
  getGreeting,
  greetingVisible,
  showSearchModal,
  refresh,
  loading,
  t,
}) => {
  return (
    <div className='mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
      <div className='space-y-1'>
        <h2
          className='text-2xl font-semibold text-gray-900 transition-opacity duration-500 ease-in-out dark:text-gray-100'
          style={{ opacity: greetingVisible ? 1 : 0 }}
        >
          {getGreeting}
        </h2>
        <p className='text-sm text-gray-500 dark:text-gray-400'>
          {t('这里展示今天最值得关注的状态、快捷操作和系统提醒。')}
        </p>
      </div>
      <div className='flex items-center gap-2'>
        <Button
          type='tertiary'
          icon={<Search size={16} />}
          onClick={showSearchModal}
          className='!rounded-lg !border !border-semi-color-border !bg-transparent'
        >
          {t('筛选数据')}
        </Button>
        <Button
          type='primary'
          icon={<RefreshCw size={16} />}
          onClick={refresh}
          loading={loading}
          className='!rounded-lg'
        >
          {t('刷新')}
        </Button>
      </div>
    </div>
  );
};

export default DashboardHeader;
