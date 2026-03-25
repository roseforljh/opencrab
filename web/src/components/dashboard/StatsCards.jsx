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
import { Card, Skeleton } from '@douyinfe/semi-ui';

const StatsCards = ({ groupedStatsData, loading, CARD_PROPS }) => {
  const flattenedStats = groupedStatsData.flatMap((group) =>
    group.items.map((item) => ({
      key: `${group.title?.props?.children?.[1] || group.title}-${item.title}`,
      title: item.title,
      value: item.value,
      description: group.title,
      onClick: item.onClick,
    })),
  );

  return (
    <div className='mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4'>
      {flattenedStats.map((item) => (
        <Card
          key={item.key}
          {...CARD_PROPS}
          className='!rounded-2xl border border-semi-color-border bg-semi-color-bg-1 shadow-none'
          bodyStyle={{ padding: 20 }}
        >
          <div
            className={`flex h-full flex-col gap-2 ${item.onClick ? 'cursor-pointer' : ''}`}
            onClick={item.onClick}
          >
            <div className='text-sm text-gray-500 dark:text-gray-400'>
              {item.title}
            </div>
            <div className='text-2xl font-semibold text-gray-900 dark:text-gray-100'>
              <Skeleton
                loading={loading}
                active
                placeholder={
                  <Skeleton.Title
                    style={{ width: '96px', height: '28px', margin: 0 }}
                  />
                }
              >
                {item.value}
              </Skeleton>
            </div>
            <div className='text-xs text-gray-400 dark:text-gray-500'>
              {item.description}
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
};

export default StatsCards;
