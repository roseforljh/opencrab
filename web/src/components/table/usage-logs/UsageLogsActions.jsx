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
import { Tag, Space, Skeleton } from '@douyinfe/semi-ui';
import { renderQuota } from '../../../helpers';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import { useMinimumLoadingTime } from '../../../hooks/common/useMinimumLoadingTime';

const LogsActions = ({
  stat,
  loadingStat,
  showStat,
  compactMode,
  setCompactMode,
  t,
}) => {
  const showSkeleton = useMinimumLoadingTime(loadingStat);
  const needSkeleton = !showStat || showSkeleton;

  const placeholder = (
    <Space>
      <Skeleton.Title style={{ width: 108, height: 21, borderRadius: 6 }} />
      <Skeleton.Title style={{ width: 65, height: 21, borderRadius: 6 }} />
      <Skeleton.Title style={{ width: 64, height: 21, borderRadius: 6 }} />
    </Space>
  );

  return (
    <div className='flex w-full flex-col gap-3 md:flex-row md:items-center md:justify-between'>
      <div>
        <div className='text-sm font-medium text-white'>{t('访问记录概览')}</div>
        <div className='mt-1 text-xs text-white/45'>
          {t('优先关注消耗、请求速率和令牌吞吐，以便快速判断系统是否异常。')}
        </div>
      </div>

      <div className='flex flex-col gap-3 md:items-end'>
        <Skeleton loading={needSkeleton} active placeholder={placeholder}>
          <Space wrap>
            <Tag
              color='blue'
              className='!rounded-full !border !border-white/10 !bg-[rgba(77,162,255,0.16)] !px-3 !py-1.5 !font-medium !text-white'
            >
              {t('消耗额度')}: {renderQuota(stat.quota)}
            </Tag>
            <Tag
              color='grey'
              className='!rounded-full !border !border-white/10 !bg-white/6 !px-3 !py-1.5 !font-medium !text-white'
            >
              RPM: {stat.rpm}
            </Tag>
            <Tag
              color='green'
              className='!rounded-full !border !border-white/10 !bg-[rgba(64,196,140,0.16)] !px-3 !py-1.5 !font-medium !text-white'
            >
              TPM: {stat.tpm}
            </Tag>
          </Space>
        </Skeleton>

        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </div>
    </div>
  );
};

export default LogsActions;
