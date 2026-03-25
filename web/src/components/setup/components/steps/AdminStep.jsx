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
import { Banner, Form } from '@douyinfe/semi-ui';
import { IconKey } from '@douyinfe/semi-icons';

const AdminStep = ({ setupStatus, formData, setFormData, formRef, t }) => {
  return setupStatus.root_init ? (
    <Banner
      type='info'
      closeIcon={null}
      description={t('管理员 PIN 已初始化，请直接进入系统')}
      className='!rounded-lg mb-4'
    />
  ) : (
    <>
      <Form.Input
        field='pin'
        label={t('PIN')}
        placeholder={t('请输入 4-12 位 PIN')}
        prefix={<IconKey />}
        showClear
        mode='password'
        rules={[
          { required: true, message: t('请输入 PIN') },
          { min: 4, message: t('PIN 长度至少为4位') },
        ]}
        initValue={formData.pin || ''}
        onChange={(value) => {
          setFormData({ ...formData, pin: value });
        }}
      />
      <Form.Input
        field='confirmPin'
        label={t('确认 PIN')}
        placeholder={t('请再次输入 PIN')}
        prefix={<IconKey />}
        showClear
        mode='password'
        rules={[
          { required: true, message: t('请确认 PIN') },
          {
            validator: (rule, value) => {
              if (value && formRef.current) {
                const pin = formRef.current.getValue('pin');
                if (value !== pin) {
                  return Promise.reject(t('两次输入的 PIN 不一致'));
                }
              }
              return Promise.resolve();
            },
          },
        ]}
        initValue={formData.confirmPin || ''}
        onChange={(value) => {
          setFormData({ ...formData, confirmPin: value });
        }}
      />
    </>
  );
};

export default AdminStep;
