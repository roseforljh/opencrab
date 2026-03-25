
import React, { useEffect, useState, useRef, useMemo } from 'react';
import { Card } from '@douyinfe/semi-ui';
import { Button } from '@/components/ui/button';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import {
  API,
  showError,
  showNotice,
  getLogo,
  getSystemName,
} from '../../helpers';
import { useTranslation } from 'react-i18next';

import AdminStep from './components/steps/AdminStep';

const FLOATING_DOTS_COUNT = 250;

const SetupWizard = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [setupStatus, setSetupStatus] = useState({
    status: false,
    root_init: false,
  });
  const formRef = useRef(null);

  const logo = getLogo();
  const systemName = getSystemName();

  const floatingDots = useMemo(
    () =>
      Array.from({ length: FLOATING_DOTS_COUNT }, (_, i) => ({
        id: i,
        top: `${Math.random() * 100}%`,
        left: `${Math.random() * 100}%`,
        size: `${Math.random() * 3.5 + 1.5}px`,
        opacity: (Math.random() * 0.6 + 0.3).toFixed(2),
        duration: `${Math.random() * 8 + 6}s`,
        delay: `${Math.random() * 4}s`,
        driftX: `${Math.random() * 200 - 100}px`,
        driftY: `${Math.random() * 150 - 75}px`,
      })),
    [],
  );

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
    const values = formData;

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
    <div
      className='relative flex min-h-screen items-center justify-center overflow-hidden bg-black px-4 py-12 sm:px-6 lg:px-8'
      style={{
        minHeight: '100vh',
        backgroundColor: '#000000',
        backgroundImage:
          'linear-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(255, 255, 255, 0.05) 1px, transparent 1px)',
        backgroundSize: '32px 32px',
        backgroundPosition: 'center center',
      }}
    >
      <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(0,0,0,0)_0%,rgba(0,0,0,0.8)_100%)]'></div>
      <div className='login-enter-overlay is-active'></div>
      <div className='login-grid-overlay'></div>
      <div className='floating-dots-container'>
        {floatingDots.map((dot) => (
          <div
            key={dot.id}
            className='floating-dot'
            style={{
              top: dot.top,
              left: dot.left,
              width: dot.size,
              height: dot.size,
              opacity: dot.opacity,
              animationDuration: dot.duration,
              animationDelay: dot.delay,
              '--float-x': dot.driftX,
              '--float-y': dot.driftY,
            }}
          />
        ))}
      </div>

      <div className='relative z-10 flex w-full items-center justify-center px-2 login-card-wrapper'>
        <div className='login-content-stage w-full max-w-sm'>
          <div className='flex flex-col items-center'>
            <div className='w-full login-panel-shell'>
              <div className='mb-8 flex items-center justify-center gap-3'>
                <div className='rounded-2xl border border-white/10 bg-white/10 p-1.5 shadow-[0_16px_48px_rgba(34,124,255,0.2)]'>
                  <img
                    src={logo}
                    alt='Logo'
                    className='h-11 w-11 rounded-xl object-cover'
                  />
                </div>
                <div>
                  <Title heading={3} className='!mb-0 !text-white'>
                    {systemName}
                  </Title>
                  <Text className='!text-white/55'>
                    {t('系统首次部署初始化')}
                  </Text>
                </div>
              </div>

              <Card className='login-card !overflow-hidden !rounded-[28px] !border !border-white/10 !bg-white/6 !backdrop-blur-xl !shadow-[0_30px_100px_rgba(0,0,0,0.4)]'>
                <div className='px-3 pt-8 pb-2 text-center'>
                  <Title heading={3} className='login-title !mb-2 !text-white'>
                    {t('初始化设置')}
                  </Title>
                  <Text className='!text-white/60'>
                    {t('设置初始管理 PIN 以继续')}
                  </Text>
                </div>

                <div className='login-input-container px-4 py-8'>
                  <form
                    className='space-y-4'
                    onSubmit={(e) => {
                      e.preventDefault();
                      onSubmit();
                    }}
                  >
                    <AdminStep
                      setupStatus={setupStatus}
                      formData={formData}
                      setFormData={setFormData}
                      formRef={formRef}
                      t={t}
                    />
                    {!setupStatus.root_init && (
                      <div className='pt-3'>
                        <Button
                          className='login-btn !h-12 !w-full !rounded-2xl !border-0 !bg-white !text-black !shadow-md hover:!bg-gray-200'
                          type='submit'
                          disabled={loading}
                        >
                          {t('保存并进入系统')}
                        </Button>
                      </div>
                    )}
                  </form>
                </div>
              </Card>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SetupWizard;
