
import React from 'react';
import { Banner } from '@douyinfe/semi-ui';
import { IconKey } from '@douyinfe/semi-icons';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

const AdminStep = ({ setupStatus, formData, setFormData, formRef, t }) => {
  return setupStatus.root_init ? (
    <Banner
      type='info'
      closeIcon={null}
      description={t('管理员 PIN 已初始化，请直接进入系统')}
      className='!rounded-lg mb-4'
    />
  ) : (
    <div className='space-y-4'>
      <div className='space-y-2'>
        <Label htmlFor='pin'>{t('PIN')}</Label>
        <div className='relative'>
          <div className='absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-muted-foreground'>
            <IconKey />
          </div>
          <Input
            id='pin'
            name='pin'
            type='password'
            placeholder={t('请输入 4-12 位 PIN')}
            className='pl-10'
            required
            minLength={4}
            value={formData.pin || ''}
            onChange={(e) => {
              setFormData({ ...formData, pin: e.target.value });
            }}
          />
        </div>
      </div>
      <div className='space-y-2'>
        <Label htmlFor='confirmPin'>{t('确认 PIN')}</Label>
        <div className='relative'>
          <div className='absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-muted-foreground'>
            <IconKey />
          </div>
          <Input
            id='confirmPin'
            name='confirmPin'
            type='password'
            placeholder={t('请再次输入 PIN')}
            className='pl-10'
            required
            value={formData.confirmPin || ''}
            onChange={(e) => {
              setFormData({ ...formData, confirmPin: e.target.value });
            }}
          />
        </div>
      </div>
    </div>
  );
};

export default AdminStep;
