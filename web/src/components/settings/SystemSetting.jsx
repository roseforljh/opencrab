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
import { Card, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const SystemSetting = () => {
  const { t } = useTranslation();

  return (
    <div className='rounded-[28px] border border-white/10 bg-white/6 p-5 text-white shadow-[0_24px_80px_rgba(0,0,0,0.3)] backdrop-blur-xl md:p-6'>
      <div className='mb-3 text-sm uppercase tracking-[0.18em] text-white/40'>
        {t('站点配置')}
      </div>
      <Card className='!rounded-[24px] !border !border-white/10 !bg-[#0d1527]/80 !text-white !shadow-none'>
        <Text className='!text-white/72'>
          {t(
            '当前版本已移除大部分系统级登录与运维配置，默认采用 PIN 登录与精简设置。',
          )}
        </Text>
      </Card>
    </div>
  );
};

export default SystemSetting;
