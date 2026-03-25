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

import React, { useEffect, useMemo, useState } from 'react';
import { Layout, TabPane, Tabs } from '@douyinfe/semi-ui';
import { useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Shapes, Cog, Shield } from 'lucide-react';

import { isRoot } from '../../helpers';
import OperationSetting from '../../components/settings/OperationSetting';
import RateLimitSetting from '../../components/settings/RateLimitSetting';
import ModelSetting from '../../components/settings/ModelSetting';
import DashboardSetting from '../../components/settings/DashboardSetting';
import ChatsSetting from '../../components/settings/ChatsSetting';
import DrawingSetting from '../../components/settings/DrawingSetting';
import ModelDeploymentSetting from '../../components/settings/ModelDeploymentSetting';

const Setting = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const [tabActiveKey, setTabActiveKey] = useState('routing');

  const panes = useMemo(() => {
    if (!isRoot()) {
      return [];
    }

    return [
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
            <ModelDeploymentSetting />
            <RateLimitSetting />
          </div>
        ),
        itemKey: 'routing',
      },
      {
        tab: (
          <span style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <Shield size={18} />
            {t('安全与登录')}
          </span>
        ),
        content: (
          <div className='space-y-4'>
            <OperationSetting />
          </div>
        ),
        itemKey: 'security',
      },
      {
        tab: (
          <span style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <Cog size={18} />
            {t('高级')}
          </span>
        ),
        content: (
          <div className='space-y-4'>
            <DashboardSetting />
            <ChatsSetting />
            <DrawingSetting />
          </div>
        ),
        itemKey: 'advanced',
      },
    ];
  }, [t]);
  const onChangeTab = (key) => {
    setTabActiveKey(key);
    navigate(`?tab=${key}`);
  };
  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search);
    const tab = searchParams.get('tab');
    if (tab) {
      setTabActiveKey(tab);
    } else {
      onChangeTab('routing');
    }
  }, [location.search]);
  return (
    <div className='mt-[60px] px-2'>
      <Layout>
        <Layout.Content>
          <Tabs
            type='card'
            collapsible
            activeKey={tabActiveKey}
            onChange={(key) => onChangeTab(key)}
          >
            {panes.map((pane) => (
              <TabPane itemKey={pane.itemKey} tab={pane.tab} key={pane.itemKey}>
                {tabActiveKey === pane.itemKey && pane.content}
              </TabPane>
            ))}
          </Tabs>
        </Layout.Content>
      </Layout>
    </div>
  );
};

export default Setting;
