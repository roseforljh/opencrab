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
import {
  Button,
  Card,
  Input,
  Space,
  Typography,
  Avatar,
} from '@douyinfe/semi-ui';
import { IconKey, IconLock, IconDelete } from '@douyinfe/semi-icons';
import { UserPlus } from 'lucide-react';

const AccountManagement = ({
  t,
  systemToken,
  generateAccessToken,
  handleSystemTokenClick,
  setShowPinModal,
  setShowChangePasswordModal,
  setShowAccountDeleteModal,
}) => {
  return (
    <Card className='!rounded-[28px] !border !border-white/10 !bg-white/6 !shadow-[0_24px_80px_rgba(0,0,0,0.28)] !backdrop-blur-xl'>
      <div className='mb-5 flex items-center'>
        <Avatar
          size='small'
          color='teal'
          className='mr-3 !bg-white/80 !text-black shadow-[0_10px_24px_rgba(0,0,0,0.15)]'
        >
          <UserPlus size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium !text-white'>
            {t('账户中枢')}
          </Typography.Text>
          <div className='text-xs text-white/45'>
            {t('单用户登录与恢复相关设置')}
          </div>
        </div>
      </div>

      <Space vertical className='w-full'>
        <Card className='!w-full !rounded-[24px] !border !border-white/10 !bg-[#0d1527]/80 !shadow-none'>
          <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
            <div className='flex w-full items-start sm:w-auto'>
              <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/10 bg-white/6'>
                <IconKey size='large' className='text-white/75' />
              </div>
              <div>
                <Typography.Title heading={6} className='!mb-1 !text-white'>
                  {t('PIN')}
                </Typography.Title>
                <Typography.Text className='!text-sm !text-white/55'>
                  {t('用于当前实例的单用户登录入口，可随时更新')}
                </Typography.Text>
              </div>
            </div>
            <Button
              type='primary'
              theme='solid'
              onClick={() => setShowPinModal(true)}
              className='!h-11 !w-full !rounded-2xl !border-0 !bg-white/90 hover:!bg-white/80 !px-5 !text-black sm:!w-auto'
              icon={<IconKey />}
            >
              {t('修改 PIN')}
            </Button>
          </div>
        </Card>

        <Card className='!w-full !rounded-[24px] !border !border-white/10 !bg-[#0d1527]/80 !shadow-none'>
          <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
            <div className='flex w-full items-start sm:w-auto'>
              <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/10 bg-white/6'>
                <IconKey size='large' className='text-white/75' />
              </div>
              <div className='flex-1'>
                <Typography.Title heading={6} className='!mb-1 !text-white'>
                  {t('系统访问令牌')}
                </Typography.Title>
                <Typography.Text className='!text-sm !text-white/55'>
                  {t('用于 API 调用的系统令牌，重新生成后请立即妥善保存')}
                </Typography.Text>
                {systemToken && (
                  <div className='mt-3'>
                    <Input
                      readonly
                      value={systemToken}
                      onClick={handleSystemTokenClick}
                      size='large'
                      prefix={<IconKey />}
                    />
                  </div>
                )}
              </div>
            </div>
            <Button
              type='primary'
              theme='solid'
              onClick={generateAccessToken}
              className='!h-11 !w-full !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10 sm:!w-auto'
              icon={<IconKey />}
            >
              {systemToken ? t('重新生成') : t('生成令牌')}
            </Button>
          </div>
        </Card>

        <Card className='!w-full !rounded-[24px] !border !border-white/10 !bg-[#0d1527]/80 !shadow-none'>
          <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
            <div className='flex w-full items-start sm:w-auto'>
              <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/10 bg-white/6'>
                <IconLock size='large' className='text-white/75' />
              </div>
              <div>
                <Typography.Title heading={6} className='!mb-1 !text-white'>
                  {t('密码管理')}
                </Typography.Title>
                <Typography.Text className='!text-sm !text-white/55'>
                  {t('作为兼容恢复入口保留，当 PIN 不可用时可用于找回访问能力')}
                </Typography.Text>
              </div>
            </div>
            <Button
              type='primary'
              theme='solid'
              onClick={() => setShowChangePasswordModal(true)}
              className='!h-11 !w-full !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10 sm:!w-auto'
              icon={<IconLock />}
            >
              {t('修改密码')}
            </Button>
          </div>
        </Card>

        <Card className='!w-full !rounded-[24px] !border !border-red-400/20 !bg-[rgba(80,20,20,0.35)] !shadow-none'>
          <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
            <div className='flex w-full items-start sm:w-auto'>
              <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-red-400/20 bg-red-500/10'>
                <IconDelete size='large' className='text-red-200' />
              </div>
              <div>
                <Typography.Title heading={6} className='!mb-1 !text-red-100'>
                  {t('删除账户')}
                </Typography.Title>
                <Typography.Text className='!text-sm !text-red-100/65'>
                  {t('此操作不可逆，所有本地账户数据将被永久删除')}
                </Typography.Text>
              </div>
            </div>
            <Button
              type='danger'
              theme='solid'
              onClick={() => setShowAccountDeleteModal(true)}
              className='!h-11 !w-full !rounded-2xl !border-0 !bg-red-900/80 hover:!bg-red-900/70 !px-5 !text-white sm:!w-auto'
              icon={<IconDelete />}
            >
              {t('删除账户')}
            </Button>
          </div>
        </Card>
      </Space>
    </Card>
  );
};

export default AccountManagement;
