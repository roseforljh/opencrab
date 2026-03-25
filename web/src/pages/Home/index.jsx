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

import React, { useContext, useEffect, useState } from 'react';
import {
  Button,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  IconGithubLogo,
  IconPlay,
  IconCopy,
} from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Cohere,
  Claude,
  Gemini,
  Suno,
  Minimax,
  Wenxin,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Midjourney,
  Grok,
  AzureAI,
  Hunyuan,
  Xinference,
} from '@lobehub/icons';

const { Text } = Typography;

const Home = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const isMobile = useIsMobile();
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = API_ENDPOINTS.map((e) => ({ value: e }));
  const [endpointIndex, setEndpointIndex] = useState(0);

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      // 如果内容是 URL，则发送主题模式
      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: actualTheme }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent('加载首页内容失败...');
    }
    setHomePageContentLoaded(true);
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <div className='relative w-full overflow-x-hidden bg-transparent text-white'>
          <div className='pointer-events-none absolute inset-0 opacity-100'>
            <div
              className='absolute inset-0'
              style={{
                backgroundImage:
                  'linear-gradient(rgba(255,255,255,0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.05) 1px, transparent 1px)',
                backgroundSize: '32px 32px',
                backgroundPosition: 'center center',
              }}
            />
            <div className='absolute left-1/2 top-0 h-[260px] w-[260px] -translate-x-1/2 rounded-full bg-white/[0.04] blur-[120px]' />
          </div>

          <div className='relative mx-auto flex min-h-[calc(100vh-72px)] w-full max-w-[1320px] items-center px-4 pb-16 pt-28 md:px-8 md:pb-24 md:pt-32'>
            <div className='grid w-full items-center gap-12 lg:grid-cols-[1.1fr_0.9fr]'>
              <div className='max-w-3xl'>
                <div className='mb-5 inline-flex items-center rounded-full border border-white/12 bg-black/55 px-4 py-2 text-sm text-white/70 shadow-[0_12px_36px_rgba(0,0,0,0.32)]'>
                  {t('统一接入层')} AI Gateway
                </div>

                <h1 className='text-5xl font-semibold leading-[1.02] tracking-[-0.04em] text-white md:text-6xl lg:text-7xl'>
                  {t('统一接入层')}
                  <br />
                  <span className='bg-clip-text text-transparent'>
                    {t('模型与能力路由中枢')}
                  </span>
                </h1>

                <p className='mt-6 max-w-2xl text-base leading-7 text-white/62 md:text-lg'>
                  {t('更稳、更顺手，切换基址后即可开始使用：')}
                </p>

                <div className='mt-8 flex w-full max-w-2xl flex-col gap-3 rounded-[28px] border border-white/10 bg-black/55 p-3 shadow-[0_30px_80px_rgba(0,0,0,0.5)] backdrop-blur-xl md:flex-row md:items-center'>
                  <div className='flex min-w-0 flex-1 items-center rounded-[22px] border border-white/10 bg-black/70 px-4 py-3'>
                    <div className='min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white/88 md:text-base'>
                      {serverAddress}
                    </div>
                    <div className='ml-3 rounded-full border border-white/10 bg-white/6 px-3 py-1 text-xs text-white/60'>
                      {endpointItems[endpointIndex]?.value}
                    </div>
                  </div>

                  <div className='flex items-center gap-3'>
                    <Button
                      theme='solid'
                      type='primary'
                      size={isMobile ? 'default' : 'large'}
                      className='!h-12 !rounded-2xl !border-0 !bg-white !text-black hover:!bg-white/90 !px-6 !shadow-[0_18px_44px_rgba(0,0,0,0.25)]'
                      icon={<IconCopy />}
                      onClick={handleCopyBaseURL}
                    >
                      {t('复制基址')}
                    </Button>
                    <Link to='/console'>
                      <Button
                        size={isMobile ? 'default' : 'large'}
                        className='!h-12 !rounded-2xl !border !border-white/10 !bg-white/6 !px-6 !text-white hover:!bg-white/10'
                        icon={<IconPlay />}
                      >
                        {t('进入控制台')}
                      </Button>
                    </Link>
                  </div>
                </div>

                <div className='mt-6 flex flex-wrap items-center gap-3 text-sm text-white/48'>
                  <span className='rounded-full border border-white/10 bg-black/55 px-3 py-1.5'>OpenAI Compatible</span>
                  <span className='rounded-full border border-white/10 bg-black/55 px-3 py-1.5'>Low Latency</span>
                  <span className='rounded-full border border-white/10 bg-black/55 px-3 py-1.5'>Unified Billing</span>
                </div>
              </div>

              <div className='relative'>
                <div className='rounded-[32px] border border-white/10 bg-black/55 p-4 shadow-[0_40px_120px_rgba(0,0,0,0.55)] backdrop-blur-2xl md:p-6'>
                  <div className='mb-4 flex items-center justify-between'>
                    <div>
                      <div className='text-sm text-white/45'>{t('支持众多的大模型供应商')}</div>
                      <div className='mt-1 text-xl font-semibold text-white'>OpenCrab Runtime</div>
                    </div>
                    {statusState?.status?.version ? (
                      <Button
                        size='small'
                        className='!rounded-xl !border !border-white/10 !bg-white/8 !text-white hover:!bg-white/12'
                        icon={<IconGithubLogo />}
                        onClick={() =>
                          window.open(
                            'https://github.com/QuantumNous/opencrab',
                            '_blank',
                          )
                        }
                      >
                        {statusState.status.version}
                      </Button>
                    ) : null}
                  </div>

                  <div className='grid grid-cols-4 gap-3 md:grid-cols-5'>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Moonshot size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><OpenAI size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><XAI size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Zhipu.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Volcengine.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Cohere.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Claude.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Gemini.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Suno size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Minimax.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Wenxin.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Spark.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Qingyan.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><DeepSeek.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Qwen.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Midjourney size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Grok size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><AzureAI.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-black/60'><Hunyuan.Color size={32} /></div>
                    <div className='flex h-16 items-center justify-center rounded-2xl border border-white/8 bg-[#0d1527]'><Xinference size={32} /></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className='w-full h-screen border-none'
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
