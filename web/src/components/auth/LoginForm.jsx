
import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import {
  API,
  getLogo,
  getSystemName,
  setUserData,
  showError,
  showSuccess,
  updateAPI,
} from '../../helpers';
import Turnstile from 'react-turnstile';
import { Card, Form } from '@douyinfe/semi-ui';
import { Button } from '@/components/ui/button';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const FLOATING_DOTS_COUNT = 250;

const LoginForm = () => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [inputs, setInputs] = useState({ pin: '' });
  const { pin } = inputs;
  const [searchParams] = useSearchParams();
  const [, userDispatch] = React.useContext(UserContext);
  const [statusState] = React.useContext(StatusContext);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [loginLoading, setLoginLoading] = useState(false);
  const [isNavigating, setIsNavigating] = useState(false);

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

  const status = useMemo(() => {
    if (statusState?.status) return statusState.status;
    const savedStatus = localStorage.getItem('status');
    if (!savedStatus) return {};
    try {
      return JSON.parse(savedStatus) || {};
    } catch {
      return {};
    }
  }, [statusState?.status]);

  useEffect(() => {
    if (status?.turnstile_check) {
      setTurnstileEnabled(true);
      setTurnstileSiteKey(status.turnstile_site_key);
    }
  }, [status]);

  useEffect(() => {
    if (searchParams.get('expired')) {
      showError(t('未登录或登录已过期，请重新登录'));
    }
  }, [searchParams, t]);

  function handleChange(name, value) {
    setInputs((prev) => ({ ...prev, [name]: value }));
  }

  async function handleSubmit() {
    if (turnstileEnabled && turnstileToken === '') {
      showError('请稍后几秒重试，Turnstile 正在检查用户环境！');
      return;
    }
    if (!pin) {
      showError('请输入 PIN！');
      return;
    }

    setLoginLoading(true);
    try {
      const res = await API.post(`/api/user/pin-login?turnstile=${turnstileToken}`, { pin });
      const { success, message, data } = res.data;
      if (success) {
        userDispatch({ type: 'login', payload: data });
        setUserData(data);
        updateAPI();
        setIsNavigating(true);
        showSuccess('登录成功！');
        setTimeout(() => {
          navigate('/console/channel');
        }, 280);
      } else {
        showError(message);
      }
    } catch {
      showError('登录失败，请重试');
    } finally {
      setLoginLoading(false);
    }
  }

  return (
    <div className='relative flex min-h-screen items-center justify-center overflow-hidden bg-black px-4 py-12 sm:px-6 lg:px-8'>
      <div className={`login-enter-overlay ${isNavigating ? 'is-active' : ''}`} />
      <div className='login-grid-overlay'></div>
      <div className='absolute inset-0 overflow-hidden'>
        {floatingDots.map((dot) => (
          <span
            key={dot.id}
            className='login-floating-dot'
            style={{
              top: dot.top,
              left: dot.left,
              width: dot.size,
              height: dot.size,
              opacity: dot.opacity,
              animationDuration: dot.duration,
              animationDelay: dot.delay,
              '--dot-drift-x': dot.driftX,
              '--dot-drift-y': dot.driftY,
            }}
          />
        ))}
      </div>

      <div className='relative z-10 flex w-full items-center justify-center px-2 login-card-wrapper'>
        <div className={`login-content-stage ${isNavigating ? 'is-navigating' : ''}`}>
          <div className='flex flex-col items-center'>
            <div className='w-full max-w-sm login-panel-shell'>
              <div className='mb-8 flex items-center justify-center gap-3'>
                <div className='rounded-2xl border border-white/10 bg-white/10 p-1.5 shadow-[0_16px_48px_rgba(34,124,255,0.2)]'>
                  <img src={logo} alt='Logo' className='h-11 w-11 rounded-xl object-cover' />
                </div>
                <div>
                  <Title heading={3} className='!mb-0 !text-white'>
                    {systemName}
                  </Title>
                  <Text className='!text-white/55'>{t('个人 API 聚合控制台')}</Text>
                </div>
              </div>

              <Card className='login-card !overflow-hidden !rounded-[28px] !border !border-white/10 !bg-white/6 !backdrop-blur-xl !shadow-[0_30px_100px_rgba(0,0,0,0.4)]'>
                <div className='px-3 pt-8 pb-2 text-center'>
                  <Title heading={3} className='login-title !mb-2 !text-white'>
                    {t('PIN 登录')}
                  </Title>
                  <Text className='!text-white/60'>{t('输入 PIN 进入你的 API 控制台')}</Text>
                </div>
                <div className='login-input-container px-4 py-8'>
                  <Form className='space-y-4'>
                    <Form.Input
                      field='pin'
                      label={t('PIN')}
                      placeholder={t('请输入你的 PIN')}
                      name='pin'
                      mode='password'
                      onChange={(value) => handleChange('pin', value)}
                      prefix={<IconKey />}
                    />

                    <div className='pt-3'>
                      <Button
                        className='login-btn !h-12 !w-full !rounded-2xl !border-0 !bg-white !text-black !shadow-md hover:!bg-gray-200'
                        type='submit'
                        onClick={handleSubmit}
                        disabled={loginLoading || isNavigating}
                      >
                        {isNavigating ? t('正在进入控制台...') : t('继续')}
                      </Button>
                    </div>
                  </Form>

                  {turnstileEnabled && turnstileSiteKey && (
                    <div className='mt-4 flex justify-center'>
                      <Turnstile
                        sitekey={turnstileSiteKey}
                        onVerify={(token) => setTurnstileToken(token)}
                      />
                    </div>
                  )}
                </div>
              </Card>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default LoginForm;
