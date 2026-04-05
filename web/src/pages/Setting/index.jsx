
import React, { useMemo } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Shapes, Shield } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';

import OperationSetting from '../../components/settings/OperationSetting';
import RateLimitSetting from '../../components/settings/RateLimitSetting';
import ModelSetting from '../../components/settings/ModelSetting';

const Setting = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();

  const panes = useMemo(
    () => [
      {
        tab: (
          <span style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <Shapes size={18} />
            {t('模型与路由')}
          </span>
        ),
        content: (
          <div className='space-y-6'>
            <ModelSetting />
            <RateLimitSetting />
          </div>
        ),
        itemKey: 'routing',
      },
      {
        tab: (
          <span style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <Shield size={18} />
            {t('系统与安全')}
          </span>
        ),
        content: (
          <div className='space-y-6'>
            <OperationSetting />
          </div>
        ),
        itemKey: 'security',
      },
    ],
    [t],
  );

  const searchParams = new URLSearchParams(location.search);
  const tabActiveKey = searchParams.get('tab') || 'routing';

  return (
    <div className='pl-2 pr-0 settings-page-shell'>
      <Tabs
        value={tabActiveKey}
        onValueChange={(key) => navigate(`?tab=${key}`)}
        className='w-full flex flex-col gap-6'
      >
        <div className='w-full rounded-[32px] border border-white/10 bg-white/5 shadow-[0_30px_100px_rgba(0,0,0,0.34)] backdrop-blur-2xl p-6'>
          <div className='mb-6'>
            <h1 className='text-2xl font-semibold text-white mb-2'>{t('设置')}</h1>
            <p className='text-white/60'>{t('管理系统配置、模型路由与安全选项')}</p>
          </div>
          <TabsList className='bg-white/5 border border-white/10 h-auto p-1 rounded-2xl'>
            {panes.map((pane) => (
              <TabsTrigger
                key={pane.itemKey}
                value={pane.itemKey}
                className='data-[state=active]:bg-white/10 data-[state=active]:text-white text-white/60 py-2 px-4 rounded-xl transition-all'
              >
                {pane.tab}
              </TabsTrigger>
            ))}
          </TabsList>
        </div>

        <div className='w-full rounded-[32px] border border-white/10 bg-white/5 shadow-[0_30px_100px_rgba(0,0,0,0.34)] backdrop-blur-2xl p-6 settings-content-shell'>
          {panes.map((pane) => (
            <TabsContent
              key={pane.itemKey}
              value={pane.itemKey}
              className='m-0 focus-visible:outline-none'
            >
              {pane.content}
            </TabsContent>
          ))}
        </div>
      </Tabs>
    </div>
  );
};

export default Setting;
