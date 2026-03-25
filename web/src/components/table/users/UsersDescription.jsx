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
import { Typography } from '@douyinfe/semi-ui';
import { IconUserAdd } from '@douyinfe/semi-icons';
import CompactModeToggle from '../../common/ui/CompactModeToggle';

const { Text } = Typography;

const UsersDescription = ({ compactMode, setCompactMode, t }) => {
  return (
    <div className='flex w-full flex-col gap-3 md:flex-row md:items-center md:justify-between'>
      <div className='flex items-start gap-3'>
        <div className='rounded-2xl border border-white/10 bg-[linear-gradient(135deg,rgba(77,162,255,0.18),rgba(139,125,255,0.12))] p-2 text-white shadow-[0_12px_24px_rgba(77,162,255,0.15)]'>
          <IconUserAdd className='mr-0' />
        </div>
        <div>
          <Text className='!font-semibold !text-white'>{t('成员中心')}</Text>
          <div className='mt-1 text-xs text-white/45'>
            {t('集中管理用户账户、权限和基础身份信息。')}
          </div>
        </div>
      </div>
      <CompactModeToggle
        compactMode={compactMode}
        setCompactMode={setCompactMode}
        t={t}
      />
    </div>
  );
};

export default UsersDescription;
