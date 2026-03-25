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

import React, { useContext, useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import {
  API,
  getLogo,
  getSystemName,
  setUserData,
  showError,
  showInfo,
  showSuccess,
  updateAPI,
} from '../../helpers';
import Turnstile from 'react-turnstile';
import { Button, Card, Checkbox, Form } from '@douyinfe/semi-ui';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import { IconKey } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const LoginForm = () => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [inputs, setInputs] = useState({ pin: '' });
  const { pin } = inputs;
  const [searchParams] = useSearchParams();
  const [, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [loginLoading, setLoginLoading] = useState(false);
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [hasUserAgreement, setHasUserAgreement] = useState(false);
  const [hasPrivacyPolicy, setHasPrivacyPolicy] = useState(false);

  const logo = getLogo();
  const systemName = getSystemName();

  const affCode = new URLSearchParams(window.location.search).get('aff');
  if (affCode) {
    localStorage.setItem('aff', affCode);
  }

  const status = useMemo(() => {
    if (statusState?.status) return statusState.status;
    const savedStatus = localStorage.getItem('status');
    if (!savedStatus) return {};
    try {
      return JSON.parse(savedStatus) || {};
    } catch (err) {
      return {};
    }
  }, [statusState?.status]);

  useEffect(() => {
    if (status?.turnstile_check) {
      setTurnstileEnabled(true);
      setTurnstileSiteKey(status.turnstile_site_key);
    }
    setHasUserAgreement(status?.user_agreement_enabled || false);
    setHasPrivacyPolicy(status?.privacy_policy_enabled || false);
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
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo('请稍后几秒重试，Turnstile 正在检查用户环境！');
      return;
    }
    if (!pin) {
      showError('请输入 PIN！');
      return;
    }

    setLoginLoading(true);
    try {
      const res = await API.post(
        `/api/user/pin-login?turnstile=${turnstileToken}`,
        {
          pin,
        },
      );
      const { success, message, data } = res.data;
      if (success) {
        userDispatch({ type: 'login', payload: data });
        setUserData(data);
        updateAPI();
        showSuccess('登录成功！');
        navigate('/console');
      } else {
        showError(message);
      }
    } catch (error) {
      showError('登录失败，请重试');
    } finally {
      setLoginLoading(false);
    }
  }

  const renderPinLoginForm = () => {
    return (
      <div className='flex flex-col items-center'>
        <div className='w-full max-w-md'>
          <div className='mb-8 flex items-center justify-center gap-3'>
            <div className='rounded-2xl border border-white/10 bg-white/10 p-1.5 shadow-[0_16px_48px_rgba(34,124,255,0.2)]'>
              <img src={logo} alt='Logo' className='h-11 w-11 rounded-xl object-cover' />
            </div>
            <div>
              <Title heading={3} className='!mb-0 !text-white'>
                {systemName}
              </Title>
              <Text className='!text-white/55'>
                {t('安全访问控制台')}
              </Text>
            </div>
          </div>

          <Card className='login-card !overflow-hidden !rounded-[28px] !border !border-white/10 !bg-white/6 !backdrop-blur-xl !shadow-[0_30px_100px_rgba(0,0,0,0.4)]'>
            <div className='px-3 pt-8 pb-2 text-center'>
              <Title heading={3} className='login-title !mb-2 !text-white'>
                {t('PIN 登录')}
              </Title>
              <Text className='!text-white/60'>
                {t('输入 PIN 继续访问你的工作台')}
              </Text>
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

                {(hasUserAgreement || hasPrivacyPolicy) && (
                  <div className='pt-3'>
                    <Checkbox
                      checked={agreedToTerms}
                      onChange={(e) => setAgreedToTerms(e.target.checked)}
                    >
                      <Text size='small' className='!text-white/60'>
                        {t('我已阅读并同意')}
                        {hasUserAgreement && (
                          <>
                            <a
                              href='/user-agreement'
                              target='_blank'
                              rel='noopener noreferrer'
                              className='mx-1 text-[#7cc7ff] hover:text-white'
                            >
                              {t('用户协议')}
                            </a>
                          </>
                        )}
                        {hasUserAgreement && hasPrivacyPolicy && t('和')}
                        {hasPrivacyPolicy && (
                          <>
                            <a
                              href='/privacy-policy'
                              target='_blank'
                              rel='noopener noreferrer'
                              className='mx-1 text-[#7cc7ff] hover:text-white'
                            >
                              {t('隐私政策')}
                            </a>
                          </>
                        )}
                      </Text>
                    </Checkbox>
                  </div>
                )}

                <div className='pt-3'>
                  <Button
                    theme='solid'
                    className='login-btn !h-12 !w-full !rounded-2xl !border-0 !bg-white !text-black !shadow-md hover:!bg-gray-200'
                    type='primary'
                    htmlType='submit'
                    onClick={handleSubmit}
                    loading={loginLoading}
                    disabled={
                      (hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms
                    }
                  >
                    {t('继续')}
                  </Button>
                </div>
              </Form>
            </div>
          </Card>
        </div>
      </div>
    );
  };

  return (
    <div
      className='relative flex min-h-screen items-center justify-center overflow-hidden bg-black px-4 py-12 sm:px-6 lg:px-8'
      style={{
        minHeight: '100vh',
        backgroundColor: '#000000',
        backgroundImage:
          'linear-gradient(rgba(255,255,255,0.045) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.045) 1px, transparent 1px)',
        backgroundSize: '28px 28px',
        backgroundPosition: 'center center',
      }}
    >
      <div className='pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.05),transparent_22%)]'></div>
      <div className='login-grid-overlay'></div>
      {/* Snowflakes effect */}
      <div className='stars-container'>
        {[...Array(50)].map((_, i) => (
          <div
            key={i}
            className='star'
            style={{
              top: `${Math.random() * -20}%`,
              left: `${Math.random() * 100}%`,
              width: `${Math.random() * 4 + 2}px`,
              height: `${Math.random() * 4 + 2}px`,
              animationDelay: `${Math.random() * 5}s`,
              animationDuration: `${Math.random() * 5 + 5}s`,
            }}
          ></div>
        ))}
      </div>

      <div className='relative z-10 w-full max-w-sm mt-[60px] login-card-wrapper'>
        {renderPinLoginForm()}

        {turnstileEnabled && (
          <div className='flex justify-center mt-6'>
            <Turnstile
              sitekey={turnstileSiteKey}
              onVerify={(token) => {
                setTurnstileToken(token);
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default LoginForm;
