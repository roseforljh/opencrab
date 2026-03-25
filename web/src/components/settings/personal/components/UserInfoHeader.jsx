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
import { Avatar, Card, Typography } from '@douyinfe/semi-ui';
import { stringToColor } from '../../../../helpers';

const UserInfoHeader = ({ t, userState }) => {
  const getUsername = () => {
    const username = userState?.user?.username?.trim();
    return username || t('未设置用户名');
  };

  const getAvatarText = () => {
    const username = userState?.user?.username?.trim();
    if (username) {
      return username.slice(0, 2).toUpperCase();
    }
    return '--';
  };

  return (
    <Card className='!rounded-[28px] !border !border-white/10 !bg-[linear-gradient(135deg,rgba(77,162,255,0.14),rgba(139,125,255,0.08))] !shadow-[0_24px_80px_rgba(0,0,0,0.28)] !backdrop-blur-xl'>
      <div className='flex items-center gap-4 p-2 sm:p-3'>
        <Avatar
          size='large'
          color={stringToColor(getUsername())}
          className='!shadow-[0_16px_36px_rgba(0,0,0,0.24)]'
        >
          {getAvatarText()}
        </Avatar>
        <div className='min-w-0'>
          <div className='truncate text-2xl font-semibold text-white'>
            {getUsername()}
          </div>
          <Typography.Text className='!text-white/55'>
            {t('当前本地账户设置')}
          </Typography.Text>
        </div>
      </div>
    </Card>
  );
};

export default UserInfoHeader;
