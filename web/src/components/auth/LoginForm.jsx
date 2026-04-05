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
import { Card, Input } from '@douyinfe/semi-ui';
import { Button } from '@/components/ui/button';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { useActualTheme } from '../../context/Theme';
import DottedSurfaceBackground from '../ui/backgrounds/DottedSurfaceBackground';

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
  const actualTheme = useActualTheme();

  const logo = getLogo();
  const systemName = getSystemName();

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
      <DottedSurfaceBackground isDark={actualTheme === 'dark'} />

      <div className='relative z-10 flex w-full items-center justify-center px-2 login-card-wrapper'>
        <div className={`login-content-stage ${isNavigating ? 'is-navigating' : ''}`}>
          <div className='flex flex-col items-center'>
            <div className='w-full max-w-sm login-panel-shell'>
              <div className='mb-8 flex items-center justify-center gap-3'>
                <div className='login-brand-mark rounded-2xl p-1.5'>
                  <img src={logo} alt='Logo' className='h-11 w-11 rounded-xl object-cover' />
                </div>
                <div className='login-brand-copy'>
                  <Title heading={3} className='login-brand-title !mb-0 !text-white'>
                    {systemName}
                  </Title>
                  <Text className='login-brand-subtitle !text-white/55'>{t('个人 API 聚合控制台')}</Text>
                </div>
              </div>

              <Card className='login-card !overflow-hidden !rounded-[28px] !border !border-white/12 !bg-transparent !backdrop-blur-xl !shadow-[0_30px_100px_rgba(0,0,0,0.4)] !ring-0 !ring-transparent'>
                <div className='login-card-glow' />
                <div className='px-4 pt-8 pb-2 text-center relative z-[1]'>
                  <Title heading={3} className='login-title !mb-2 !text-white'>
                    {t('PIN 登录')}
                  </Title>
                  <Text className='login-subtitle !text-white/60'>{t('输入 PIN 进入你的 API 控制台')}</Text>
                </div>
                <div className='login-input-container px-5 py-8 relative z-[1]'>
                  <div className='space-y-4'>
                    <div className='space-y-2'>
                      <div className='text-sm font-medium text-white'>{t('PIN')}</div>
                      <Input
                        placeholder={t('请输入你的 PIN')}
                        name='pin'
                        mode='password'
                        value={pin}
                        onChange={(value) => handleChange('pin', value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            handleSubmit();
                          }
                        }}
                        enterKeyHint='enter'
                        prefix={<IconKey />}
                      />
                    </div>

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
                  </div>

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
