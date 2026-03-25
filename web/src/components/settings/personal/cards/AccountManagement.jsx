
import React from 'react';
import { UserPlus } from 'lucide-react';
import { IconKey, IconLock, IconDelete } from '@douyinfe/semi-icons';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

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
    <Card className='!rounded-[28px] border border-white/10 bg-white/6 shadow-[0_24px_80px_rgba(0,0,0,0.28)] backdrop-blur-xl text-white'>
      <CardContent className='p-5'>
        <div className='mb-5 flex items-center'>
          <div className='mr-3 flex h-8 w-8 items-center justify-center rounded-full bg-white/80 text-black shadow-[0_10px_24px_rgba(0,0,0,0.15)]'>
            <UserPlus size={16} />
          </div>
          <div>
            <div className='text-lg font-medium text-white'>
              {t('账户中枢')}
            </div>
            <div className='text-xs text-white/45'>
              {t('单用户登录与恢复相关设置')}
            </div>
          </div>
        </div>

        <div className='flex flex-col gap-4 w-full'>
          <Card className='w-full !rounded-[24px] border border-white/10 bg-[#0d1527]/80 shadow-none text-white'>
            <CardContent className='p-4'>
              <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
                <div className='flex w-full items-start sm:w-auto'>
                  <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/10 bg-white/6'>
                    <IconKey className='text-white/75 text-xl' />
                  </div>
                  <div>
                    <h6 className='mb-1 font-medium text-white'>{t('PIN')}</h6>
                    <div className='text-sm text-white/55'>
                      {t('用于当前实例的单用户登录入口，可随时更新')}
                    </div>
                  </div>
                </div>
                <Button
                  onClick={() => setShowPinModal(true)}
                  className='h-11 w-full rounded-2xl border-0 bg-white/90 hover:bg-white/80 px-5 text-black sm:w-auto gap-2'
                >
                  <IconKey />
                  {t('修改 PIN')}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card className='w-full !rounded-[24px] border border-white/10 bg-[#0d1527]/80 shadow-none text-white'>
            <CardContent className='p-4'>
              <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
                <div className='flex w-full items-start sm:w-auto'>
                  <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/10 bg-white/6'>
                    <IconKey className='text-white/75 text-xl' />
                  </div>
                  <div className='flex-1'>
                    <h6 className='mb-1 font-medium text-white'>
                      {t('系统访问令牌')}
                    </h6>
                    <div className='text-sm text-white/55'>
                      {t('用于 API 调用的系统令牌，重新生成后请立即妥善保存')}
                    </div>
                    {systemToken && (
                      <div className='mt-3 relative'>
                        <div className='absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none'>
                          <IconKey className='text-white/50' />
                        </div>
                        <Input
                          readOnly
                          value={systemToken}
                          onClick={handleSystemTokenClick}
                          className='pl-10 h-10 bg-black/20 border-white/10 text-white focus-visible:ring-white/20'
                        />
                      </div>
                    )}
                  </div>
                </div>
                <Button
                  onClick={generateAccessToken}
                  className='h-11 w-full rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10 sm:w-auto gap-2'
                >
                  <IconKey />
                  {systemToken ? t('重新生成') : t('生成令牌')}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card className='w-full !rounded-[24px] border border-white/10 bg-[#0d1527]/80 shadow-none text-white'>
            <CardContent className='p-4'>
              <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
                <div className='flex w-full items-start sm:w-auto'>
                  <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-white/10 bg-white/6'>
                    <IconLock className='text-white/75 text-xl' />
                  </div>
                  <div>
                    <h6 className='mb-1 font-medium text-white'>
                      {t('密码管理')}
                    </h6>
                    <div className='text-sm text-white/55'>
                      {t(
                        '作为兼容恢复入口保留，当 PIN 不可用时可用于找回访问能力',
                      )}
                    </div>
                  </div>
                </div>
                <Button
                  onClick={() => setShowChangePasswordModal(true)}
                  className='h-11 w-full rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10 sm:w-auto gap-2'
                >
                  <IconLock />
                  {t('修改密码')}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card className='w-full !rounded-[24px] border border-red-400/20 bg-[rgba(80,20,20,0.35)] shadow-none'>
            <CardContent className='p-4'>
              <div className='flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between'>
                <div className='flex w-full items-start sm:w-auto'>
                  <div className='mr-4 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl border border-red-400/20 bg-red-500/10'>
                    <IconDelete className='text-red-200 text-xl' />
                  </div>
                  <div>
                    <h6 className='mb-1 font-medium text-red-100'>
                      {t('删除账户')}
                    </h6>
                    <div className='text-sm text-red-100/65'>
                      {t('此操作不可逆，所有本地账户数据将被永久删除')}
                    </div>
                  </div>
                </div>
                <Button
                  variant='destructive'
                  onClick={() => setShowAccountDeleteModal(true)}
                  className='h-11 w-full rounded-2xl border-0 bg-red-900/80 hover:bg-red-900/70 px-5 text-white sm:w-auto gap-2'
                >
                  <IconDelete />
                  {t('删除账户')}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </CardContent>
    </Card>
  );
};

export default AccountManagement;
