
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
          <div className='space-y-4'>
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
          <div className='space-y-4'>
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
    <div className='px-2'>
      <div className='w-full'>
        <Tabs
          value={tabActiveKey}
          onValueChange={(key) => navigate(`?tab=${key}`)}
          className='w-full'
        >
          <TabsList className='bg-white/5 border border-white/10 mb-4 h-auto p-1'>
            {panes.map((pane) => (
              <TabsTrigger
                key={pane.itemKey}
                value={pane.itemKey}
                className='data-[state=active]:bg-white/10 data-[state=active]:text-white text-white/60 py-2 px-4'
              >
                {pane.tab}
              </TabsTrigger>
            ))}
          </TabsList>
          {panes.map((pane) => (
            <TabsContent
              key={pane.itemKey}
              value={pane.itemKey}
              className='m-0 focus-visible:outline-none'
            >
              {pane.content}
            </TabsContent>
          ))}
        </Tabs>
      </div>
    </div>
  );
};

export default Setting;
