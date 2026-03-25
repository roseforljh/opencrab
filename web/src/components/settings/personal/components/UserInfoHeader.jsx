
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
