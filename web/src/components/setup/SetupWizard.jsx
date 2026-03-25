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

import React, { useEffect, useState, useRef } from 'react';
import { Card, Form, Button } from '@douyinfe/semi-ui';
import { API, showError, showNotice } from '../../helpers';
import { useTranslation } from 'react-i18next';

import AdminStep from './components/steps/AdminStep';

const SetupWizard = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [setupStatus, setSetupStatus] = useState({
    status: false,
    root_init: false,
  });
  const formRef = useRef(null);

  const [formData, setFormData] = useState({
    pin: '',
    confirmPin: '',
  });

  useEffect(() => {
    fetchSetupStatus();
  }, []);

  const fetchSetupStatus = async () => {
    try {
      const res = await API.get('/api/setup');
      const { success, data } = res.data;
      if (success) {
        setSetupStatus(data);
        if (data.status) {
          window.location.href = '/';
        }
      } else {
        showError(t('获取初始化状态失败'));
      }
    } catch (error) {
      console.error('Failed to fetch setup status:', error);
      showError(t('获取初始化状态失败'));
    }
  };

  const onSubmit = () => {
    if (!formRef.current) {
      showError(t('表单引用错误，请刷新页面重试'));
      return;
    }

    const values = formRef.current.getValues();

    if (!values.pin || values.pin.length < 4) {
      showError(t('PIN 长度至少为4位'));
      return;
    }

    if (values.pin !== values.confirmPin) {
      showError(t('两次输入的 PIN 不一致'));
      return;
    }

    setLoading(true);
    API.post('/api/setup', {
      pin: values.pin,
      confirmPin: values.confirmPin,
    })
      .then((res) => {
        const { success, message } = res.data;
        if (success) {
          showNotice(t('系统初始化成功，正在跳转...'));
          setTimeout(() => {
            window.location.reload();
          }, 1500);
        } else {
          showError(message || t('初始化失败，请重试'));
        }
      })
      .catch((error) => {
        console.error('API error:', error);
        showError(t('系统初始化失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <div className='min-h-screen flex items-center justify-center px-4'>
      <div className='w-full max-w-xl'>
        <Card className='!rounded-2xl shadow-sm border-0'>
          <div className='mb-4'>
            <div className='text-xl font-semibold'>{t('系统初始化')}</div>
            <div className='text-xs text-gray-600'>
              {t('设置 PIN 后开始使用系统')}
            </div>
          </div>

          <Form
            getFormApi={(formApi) => {
              formRef.current = formApi;
            }}
            initValues={formData}
          >
            <AdminStep
              setupStatus={setupStatus}
              formData={formData}
              setFormData={setFormData}
              formRef={formRef}
              t={t}
            />
            {!setupStatus.root_init && (
              <div className='flex justify-end pt-4'>
                <Button
                  type='primary'
                  onClick={onSubmit}
                  loading={loading}
                  className='!rounded-lg'
                >
                  {t('初始化系统')}
                </Button>
              </div>
            )}
          </Form>
        </Card>
      </div>
    </div>
  );
};

export default SetupWizard;
