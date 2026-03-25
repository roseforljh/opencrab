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

import React, { useContext, useEffect, useMemo } from 'react';
import { Card, Button, Empty, Tag } from '@douyinfe/semi-ui';
import {
  Activity,
  ArrowRight,
  CircleAlert,
  Clock3,
  KeyRound,
  LayoutPanelTop,
  Server,
  Settings,
  Sparkles,
} from 'lucide-react';
import { getRelativeTime } from '../../helpers';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import DashboardHeader from './DashboardHeader';
import StatsCards from './StatsCards';
import ChartsPanel from './ChartsPanel';
import UptimePanel from './UptimePanel';
import SearchModal from './modals/SearchModal';
import { useDashboardData } from '../../hooks/dashboard/useDashboardData';
import { useDashboardStats } from '../../hooks/dashboard/useDashboardStats';
import { useDashboardCharts } from '../../hooks/dashboard/useDashboardCharts';
import {
  CHART_CONFIG,
  CARD_PROPS,
  FLEX_CENTER_GAP2,
  ILLUSTRATION_SIZE,
  UPTIME_STATUS_MAP,
} from '../../constants/dashboard.constants';
import {
  getUptimeStatusColor,
  getUptimeStatusText,
  renderMonitorList,
} from '../../helpers/dashboard';

const QuickActionCard = ({ icon, title, description, onClick, t }) => (
  <button
    type='button'
    onClick={onClick}
    className='flex w-full items-start gap-3 rounded-2xl border border-semi-color-border bg-semi-color-bg-1 p-4 text-left transition hover:border-semi-color-primary hover:bg-semi-color-fill-0'
  >
    <div className='mt-0.5 rounded-xl bg-semi-color-fill-0 p-2 text-semi-color-primary'>
      {icon}
    </div>
    <div className='min-w-0 flex-1'>
      <div className='flex items-center justify-between gap-2'>
        <div className='font-medium text-gray-900 dark:text-gray-100'>
          {title}
        </div>
        <ArrowRight size={16} className='text-gray-400' />
      </div>
      <div className='mt-1 text-sm text-gray-500 dark:text-gray-400'>
        {description}
      </div>
    </div>
  </button>
);

const ActivityList = ({ items, emptyText, t }) => {
  if (items.length === 0) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        title={emptyText}
        description={t('当前没有需要额外处理的记录。')}
      />
    );
  }

  return (
    <div className='space-y-3'>
      {items.map((item, index) => (
        <div
          key={`${item.time}-${index}`}
          className='flex items-start justify-between gap-3 rounded-xl border border-semi-color-border bg-semi-color-bg-1 px-4 py-3'
        >
          <div className='min-w-0'>
            <div className='text-sm font-medium text-gray-900 dark:text-gray-100'>
              {item.content || t('系统活动')}
            </div>
            <div className='mt-1 text-xs text-gray-500 dark:text-gray-400'>
              {item.relative ? `${item.relative} · ${item.time}` : item.time}
            </div>
          </div>
          {item.type && (
            <Tag color='grey' size='small' shape='circle'>
              {item.type}
            </Tag>
          )}
        </div>
      ))}
    </div>
  );
};

const Dashboard = () => {
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);

  const dashboardData = useDashboardData(userState, userDispatch, statusState);

  const dashboardCharts = useDashboardCharts(
    dashboardData.dataExportDefaultTime,
    dashboardData.setTrendData,
    dashboardData.setConsumeQuota,
    dashboardData.setTimes,
    dashboardData.setConsumeTokens,
    dashboardData.setPieData,
    dashboardData.setLineData,
    dashboardData.setModelColors,
    dashboardData.t,
  );

  const { groupedStatsData } = useDashboardStats(
    userState,
    dashboardData.consumeQuota,
    dashboardData.consumeTokens,
    dashboardData.times,
    dashboardData.trendData,
    dashboardData.performanceMetrics,
    dashboardData.navigate,
    dashboardData.t,
  );

  const initChart = async () => {
    await dashboardData.loadQuotaData().then((data) => {
      if (data && data.length > 0) {
        dashboardCharts.updateChartData(data);
      }
    });
    await dashboardData.loadUptimeData();
  };

  const handleRefresh = async () => {
    const data = await dashboardData.refresh();
    if (data && data.length > 0) {
      dashboardCharts.updateChartData(data);
    }
  };

  const handleSearchConfirm = async () => {
    await dashboardData.handleSearchConfirm(dashboardCharts.updateChartData);
  };

  const announcementData = (statusState?.status?.announcements || [])
    .map((item) => {
      const pubDate = item?.publishDate ? new Date(item.publishDate) : null;
      const absoluteTime =
        pubDate && !isNaN(pubDate.getTime())
          ? `${pubDate.getFullYear()}-${String(pubDate.getMonth() + 1).padStart(2, '0')}-${String(pubDate.getDate()).padStart(2, '0')} ${String(pubDate.getHours()).padStart(2, '0')}:${String(pubDate.getMinutes()).padStart(2, '0')}`
          : item?.publishDate || '';
      return {
        ...item,
        time: absoluteTime,
        relative: getRelativeTime(item.publishDate),
      };
    })
    .slice(0, 5);

  const uptimeLegendData = Object.entries(UPTIME_STATUS_MAP).map(
    ([status, info]) => ({
      status: Number(status),
      color: info.color,
      label: dashboardData.t(info.label),
    }),
  );

  const quickActions = useMemo(
    () => [
      {
        key: 'playground',
        title: dashboardData.t('打开操练场'),
        description: dashboardData.t('快速调试模型请求与参数组合。'),
        icon: <Sparkles size={18} />,
        onClick: () => dashboardData.navigate('/console/playground'),
      },
      {
        key: 'token',
        title: dashboardData.t('管理令牌'),
        description: dashboardData.t('创建、查看并清理访问凭证。'),
        icon: <KeyRound size={18} />,
        onClick: () => dashboardData.navigate('/console/token'),
      },
      {
        key: 'log',
        title: dashboardData.t('查看日志'),
        description: dashboardData.t('排查最近请求、错误和消耗情况。'),
        icon: <Activity size={18} />,
        onClick: () => dashboardData.navigate('/console/log'),
      },
      {
        key: 'setting',
        title: dashboardData.t('系统设置'),
        description: dashboardData.t('调整运行参数、模型配置和安全选项。'),
        icon: <Settings size={18} />,
        onClick: () => dashboardData.navigate('/console/setting'),
      },
    ],
    [dashboardData.navigate, dashboardData.t],
  );

  const reminders = useMemo(() => {
    const items = [];

    if ((statusState?.status?.api_info || []).length === 0) {
      items.push({
        title: dashboardData.t('尚未配置 API 信息展示'),
        level: dashboardData.t('提醒'),
      });
    }

    if (dashboardData.uptimeEnabled && dashboardData.uptimeData.length === 0) {
      items.push({
        title: dashboardData.t('当前没有可用的服务可用性监控数据'),
        level: dashboardData.t('注意'),
      });
    }

    if (!userState?.user?.access_token) {
      items.push({
        title: dashboardData.t('当前账号未绑定默认访问令牌，请检查令牌管理'),
        level: dashboardData.t('检查'),
      });
    }

    return items;
  }, [
    dashboardData.t,
    dashboardData.uptimeData.length,
    dashboardData.uptimeEnabled,
    statusState?.status?.api_info,
    userState?.user?.access_token,
  ]);

  useEffect(() => {
    initChart();
  }, []);

  return (
    <div className='h-full'>
      <DashboardHeader
        getGreeting={dashboardData.getGreeting}
        greetingVisible={dashboardData.greetingVisible}
        showSearchModal={dashboardData.showSearchModal}
        refresh={handleRefresh}
        loading={dashboardData.loading}
        t={dashboardData.t}
      />

      <SearchModal
        searchModalVisible={dashboardData.searchModalVisible}
        handleSearchConfirm={handleSearchConfirm}
        handleCloseModal={dashboardData.handleCloseModal}
        isMobile={dashboardData.isMobile}
        isAdminUser={dashboardData.isAdminUser}
        inputs={dashboardData.inputs}
        dataExportDefaultTime={dashboardData.dataExportDefaultTime}
        timeOptions={dashboardData.timeOptions}
        handleInputChange={dashboardData.handleInputChange}
        t={dashboardData.t}
      />

      <StatsCards
        groupedStatsData={groupedStatsData}
        loading={dashboardData.loading}
        CARD_PROPS={CARD_PROPS}
      />

      <div className='mb-6 grid grid-cols-1 gap-4 xl:grid-cols-4'>
        <Card
          {...CARD_PROPS}
          className='!rounded-2xl border border-semi-color-border shadow-none xl:col-span-2'
          title={
            <div className={FLEX_CENTER_GAP2}>
              <LayoutPanelTop size={16} />
              {dashboardData.t('快捷操作')}
            </div>
          }
        >
          <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
            {quickActions.map((item) => (
              <QuickActionCard key={item.key} {...item} t={dashboardData.t} />
            ))}
          </div>
        </Card>

        <Card
          {...CARD_PROPS}
          className='!rounded-2xl border border-semi-color-border shadow-none xl:col-span-2'
          title={
            <div className={FLEX_CENTER_GAP2}>
              <CircleAlert size={16} />
              {dashboardData.t('配置提醒')}
            </div>
          }
        >
          {reminders.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              title={dashboardData.t('当前配置状态良好')}
              description={dashboardData.t('暂时没有需要优先处理的提醒。')}
            />
          ) : (
            <div className='space-y-3'>
              {reminders.map((item, index) => (
                <div
                  key={`${item.title}-${index}`}
                  className='flex items-start justify-between gap-3 rounded-xl border border-semi-color-border bg-semi-color-bg-1 px-4 py-3'
                >
                  <div className='text-sm text-gray-700 dark:text-gray-200'>
                    {item.title}
                  </div>
                  <Tag color='orange' shape='circle' size='small'>
                    {item.level}
                  </Tag>
                </div>
              ))}
            </div>
          )}
        </Card>
      </div>

      <div className='mb-6 grid grid-cols-1 gap-4 xl:grid-cols-2'>
        <Card
          {...CARD_PROPS}
          className='!rounded-2xl border border-semi-color-border shadow-none'
          title={
            <div className={FLEX_CENTER_GAP2}>
              <Clock3 size={16} />
              {dashboardData.t('最近活动')}
            </div>
          }
        >
          <ActivityList
            items={announcementData}
            emptyText={dashboardData.t('暂无最近活动')}
            t={dashboardData.t}
          />
        </Card>

        <Card
          {...CARD_PROPS}
          className='!rounded-2xl border border-semi-color-border shadow-none'
          title={
            <div className={FLEX_CENTER_GAP2}>
              <Server size={16} />
              {dashboardData.t('系统状态')}
            </div>
          }
        >
          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
            <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
              <div className='text-sm text-gray-500 dark:text-gray-400'>
                {dashboardData.t('登录用户')}
              </div>
              <div className='mt-2 text-xl font-semibold text-gray-900 dark:text-gray-100'>
                {userState?.user?.username || '-'}
              </div>
            </div>
            <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
              <div className='text-sm text-gray-500 dark:text-gray-400'>
                {dashboardData.t('可用监控分组')}
              </div>
              <div className='mt-2 text-xl font-semibold text-gray-900 dark:text-gray-100'>
                {dashboardData.uptimeData.length}
              </div>
            </div>
            <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
              <div className='text-sm text-gray-500 dark:text-gray-400'>
                {dashboardData.t('公告条目')}
              </div>
              <div className='mt-2 text-xl font-semibold text-gray-900 dark:text-gray-100'>
                {announcementData.length}
              </div>
            </div>
            <div className='rounded-xl border border-semi-color-border bg-semi-color-bg-1 p-4'>
              <div className='text-sm text-gray-500 dark:text-gray-400'>
                {dashboardData.t('统计时间粒度')}
              </div>
              <div className='mt-2 text-xl font-semibold text-gray-900 dark:text-gray-100'>
                {dashboardData.t(
                  dashboardData.dataExportDefaultTime === 'hour'
                    ? '小时'
                    : dashboardData.dataExportDefaultTime === 'week'
                      ? '周'
                      : '天',
                )}
              </div>
            </div>
          </div>
        </Card>
      </div>

      <div className='mb-6'>
        <ChartsPanel
          activeChartTab={dashboardData.activeChartTab}
          setActiveChartTab={dashboardData.setActiveChartTab}
          spec_line={dashboardCharts.spec_line}
          spec_model_line={dashboardCharts.spec_model_line}
          spec_pie={dashboardCharts.spec_pie}
          spec_rank_bar={dashboardCharts.spec_rank_bar}
          CARD_PROPS={CARD_PROPS}
          CHART_CONFIG={CHART_CONFIG}
          FLEX_CENTER_GAP2={FLEX_CENTER_GAP2}
          hasApiInfoPanel={false}
          t={dashboardData.t}
        />
      </div>

      {dashboardData.uptimeEnabled && (
        <div className='mb-4'>
          <UptimePanel
            uptimeData={dashboardData.uptimeData}
            uptimeLoading={dashboardData.uptimeLoading}
            activeUptimeTab={dashboardData.activeUptimeTab}
            setActiveUptimeTab={dashboardData.setActiveUptimeTab}
            loadUptimeData={dashboardData.loadUptimeData}
            uptimeLegendData={uptimeLegendData}
            renderMonitorList={(monitors) =>
              renderMonitorList(
                monitors,
                (status) => getUptimeStatusColor(status, UPTIME_STATUS_MAP),
                (status) =>
                  getUptimeStatusText(
                    status,
                    UPTIME_STATUS_MAP,
                    dashboardData.t,
                  ),
                dashboardData.t,
              )
            }
            CARD_PROPS={CARD_PROPS}
            ILLUSTRATION_SIZE={ILLUSTRATION_SIZE}
            t={dashboardData.t}
          />
        </div>
      )}
    </div>
  );
};

export default Dashboard;
