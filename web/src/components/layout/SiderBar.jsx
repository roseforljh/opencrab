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
import { Link, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getLucideIcon } from '../../helpers/render';
import { ChevronLeft } from 'lucide-react';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import { useSidebar } from '../../hooks/common/useSidebar';
import { useMinimumLoadingTime } from '../../hooks/common/useMinimumLoadingTime';
import { isAdmin, isRoot, getLogo, getSystemName } from '../../helpers';
import SkeletonWrapper from './components/SkeletonWrapper';
import HeaderLogo from './headerbar/HeaderLogo';

import { Nav, Divider, Button } from '@douyinfe/semi-ui';

const routerMap = {
  home: '/',
  overview: '/console',
  playground: '/console/playground',
  token: '/console/token',
  log: '/console/log',
  channel: '/console/channel',
  models: '/console/models',
  deployment: '/console/deployment',
  setting: '/console/setting',
  personal: '/console/personal',
  user: '/console/user',
  task: '/console/task',
  midjourney: '/console/midjourney',
};

const SiderBar = ({ onNavigate = () => {} }) => {
  const { t } = useTranslation();
  const [collapsed, toggleCollapsed] = useSidebarCollapsed();
  const {
    isModuleVisible,
    hasSectionVisibleModules,
    loading: sidebarLoading,
  } = useSidebar();

  const showSkeleton = useMinimumLoadingTime(sidebarLoading, 200);

  const [selectedKeys, setSelectedKeys] = useState(['overview']);
  const [openedKeys, setOpenedKeys] = useState([]);
  const location = useLocation();
  const [routerMapState] = useState(routerMap);

  const logo = getLogo();
  const systemName = getSystemName();

  const overviewItems = useMemo(() => {
    const items = [
      {
        text: '总览台',
        itemKey: 'overview',
      },
    ];

    return items.filter((item) => isModuleVisible('console', item.itemKey));
  }, [t, isModuleVisible]);

  const workspaceItems = useMemo(() => {
    const items = [
      {
        text: '实验场',
        itemKey: 'playground',
      },
      {
        text: '访问密钥',
        itemKey: 'token',
      },
      {
        text: '调用记录',
        itemKey: 'log',
      },
    ];

    return items.filter((item) => isModuleVisible('console', item.itemKey));
  }, [t, isModuleVisible]);

  const resourceItems = useMemo(() => {
    const items = [
      {
        text: '渠道接入',
        itemKey: 'channel',
        className: isAdmin() ? '' : 'tableHiddle',
      },
      {
        text: '模型编排',
        itemKey: 'models',
        className: isAdmin() ? '' : 'tableHiddle',
      },
      {
        text: '部署实例',
        itemKey: 'deployment',
        className: isAdmin() ? '' : 'tableHiddle',
      },
    ];

    return items.filter((item) => isModuleVisible('admin', item.itemKey));
  }, [t, isModuleVisible]);

  const systemItems = useMemo(() => {
    const items = [
      {
        text: '站点配置',
        itemKey: 'setting',
        className: isRoot() ? '' : 'tableHiddle',
      },
      {
        text: '个人中心',
        itemKey: 'personal',
      },
      {
        text: '成员管理',
        itemKey: 'user',
        className: isAdmin() ? '' : 'tableHiddle',
      },
      {
        text: '任务记录',
        itemKey: 'task',
        className:
          localStorage.getItem('enable_task') === 'true' ? '' : 'tableHiddle',
      },
      {
        text: '绘图记录',
        itemKey: 'midjourney',
        className:
          localStorage.getItem('enable_drawing') === 'true'
            ? ''
            : 'tableHiddle',
      },
    ];

    return items.filter((item) => {
      if (item.itemKey === 'personal') {
        return isModuleVisible('personal', item.itemKey);
      }
      if (item.itemKey === 'task' || item.itemKey === 'midjourney') {
        return isModuleVisible('console', item.itemKey);
      }
      return isModuleVisible('admin', item.itemKey);
    });
  }, [
    t,
    isModuleVisible,
    localStorage.getItem('enable_task'),
    localStorage.getItem('enable_drawing'),
  ]);

  // 根据当前路径设置选中的菜单项
  useEffect(() => {
    const currentPath = location.pathname;
    const matchingKey = Object.keys(routerMapState).find(
      (key) => routerMapState[key] === currentPath,
    );

    if (matchingKey) {
      setSelectedKeys([matchingKey]);
    }
  }, [location.pathname, routerMapState]);

  // 监控折叠状态变化以更新 body class
  useEffect(() => {
    if (collapsed) {
      document.body.classList.add('sidebar-collapsed');
    } else {
      document.body.classList.remove('sidebar-collapsed');
    }
  }, [collapsed]);

  // 选中高亮颜色（统一）
  const SELECTED_COLOR = 'rgba(255, 255, 255, 0.9)';
  
  // State for the sliding hover box
  const [hoveredNode, setHoveredNode] = useState(null);
  
  // Update sliding box position based on hovered item
  const handleMouseOver = (e) => {
    const item = e.target.closest('.sidebar-nav-item-wrapper');
    if (item) {
      const parent = item.closest('.sidebar-nav-list-container');
      if (parent) {
        const itemRect = item.getBoundingClientRect();
        const parentRect = parent.getBoundingClientRect();
        setHoveredNode({
          top: itemRect.top - parentRect.top + parent.scrollTop,
          height: itemRect.height,
          opacity: 1
        });
        return;
      }
    }
  };

  const handleMouseLeave = () => {
    setHoveredNode(prev => prev ? { ...prev, opacity: 0 } : null);
  };

  // 渲染自定义菜单项
  const renderNavItem = (item) => {
    // 跳过隐藏的项目
    if (item.className === 'tableHiddle') return null;

    const isSelected = selectedKeys.includes(item.itemKey);
    const textColor = isSelected ? SELECTED_COLOR : 'inherit';
    const iconColor = isSelected ? 'text-white/90' : 'text-white/55';

    return (
      <div
        className={`sidebar-nav-item-wrapper px-2 my-1 relative ${isSelected ? 'selected' : ''}`}
        key={item.itemKey}
      >
        <Nav.Item
          itemKey={item.itemKey}
          text={
            <span
              className={`truncate text-sm font-medium z-10 relative transition-all duration-300 ${collapsed ? 'opacity-0 w-0 overflow-hidden hidden' : 'opacity-100 w-auto'}`}
              style={{ color: textColor }}
            >
              {item.text}
            </span>
          }
          icon={
            <div
              className={`sidebar-icon-container flex-shrink-0 z-10 relative transition-colors duration-200 ${iconColor}`}
            >
              {getLucideIcon(item.itemKey, isSelected)}
            </div>
          }
          className={`${item.className || ''} sidebar-nav-item-animated ${isSelected ? 'active-item' : ''}`}
        />
      </div>
    );
  };

  // 渲染子菜单项
  const renderSubItem = (item) => {
    if (item.items && item.items.length > 0) {
      const isSelected = selectedKeys.includes(item.itemKey);
      const textColor = isSelected ? SELECTED_COLOR : 'inherit';
      const iconColor = isSelected ? 'text-white/90' : 'text-white/55';

      return (
        <div
          className={`sidebar-nav-item-wrapper px-2 my-1 relative ${isSelected ? 'selected' : ''}`}
          key={item.itemKey}
        >
          <Nav.Sub
            itemKey={item.itemKey}
            text={
              <span
                className={`truncate text-sm font-medium z-10 relative transition-all duration-300 ${collapsed ? 'opacity-0 w-0 overflow-hidden hidden' : 'opacity-100 w-auto'}`}
                style={{ color: textColor }}
              >
                {item.text}
              </span>
            }
            icon={
              <div
                className={`sidebar-icon-container flex-shrink-0 z-10 relative transition-colors duration-200 ${iconColor}`}
              >
                {getLucideIcon(item.itemKey, isSelected)}
              </div>
            }
            className={`${item.className || ''} sidebar-nav-item-animated ${isSelected ? 'active-item' : ''}`}
          >
            {item.items.map((subItem) => {
              const isSubSelected = selectedKeys.includes(subItem.itemKey);
              const subTextColor = isSubSelected ? SELECTED_COLOR : 'inherit';
              const subIconColor = isSubSelected ? 'text-white/90' : 'text-white/55';

              return (
                <div
                  className={`sidebar-nav-item-wrapper px-2 my-[2px] relative ${isSubSelected ? 'selected' : ''}`}
                  key={subItem.itemKey}
                >
                  <Nav.Item
                    itemKey={subItem.itemKey}
                    text={
                      <span
                        className={`truncate text-sm font-medium z-10 relative transition-all duration-300 ${collapsed ? 'opacity-0 w-0 overflow-hidden hidden' : 'opacity-100 w-auto'}`}
                        style={{ color: subTextColor }}
                      >
                        {subItem.text}
                      </span>
                    }
                    className={`sidebar-nav-item-animated ${isSubSelected ? 'active-item' : ''}`}
                  />
                </div>
              );
            })}
          </Nav.Sub>
        </div>
      );
    } else {
      return renderNavItem(item);
    }
  };

  return (
    <div
      className={`sidebar-container ${collapsed ? 'px-2 pb-3 pt-2' : 'px-3 pb-4 pt-2'}`}
      style={{
        width: 'var(--sidebar-current-width)',
        height: '100vh',
      }}
    >
      <div className='flex h-full flex-col rounded-xl bg-transparent'>
      <SkeletonWrapper
        loading={showSkeleton}
        type='sidebar'
        className='flex-1 min-h-0'
        collapsed={collapsed}
        showAdmin={isAdmin()}
      >
        <Nav
          className='sidebar-nav !bg-transparent relative'
          defaultIsCollapsed={collapsed}
          isCollapsed={collapsed}
          onCollapseChange={toggleCollapsed}
          selectedKeys={selectedKeys}
          itemStyle='sidebar-nav-item'
          hoverStyle=''
          selectedStyle=''
          renderWrapper={({ itemElement, props }) => {
            const to =
              routerMapState[props.itemKey] || routerMap[props.itemKey];

            // 如果没有路由，直接返回元素
            if (!to) return itemElement;

            return (
              <Link
                style={{ textDecoration: 'none' }}
                to={to}
                onClick={onNavigate}
              >
                {itemElement}
              </Link>
            );
          }}
          onSelect={(key) => {
            // 如果点击的是已经展开的子菜单的父项，则收起子菜单
            if (openedKeys.includes(key.itemKey)) {
              setOpenedKeys(openedKeys.filter((k) => k !== key.itemKey));
            }

            setSelectedKeys([key.itemKey]);
          }}
          openKeys={openedKeys}
          onOpenChange={(data) => {
            setOpenedKeys(data.openKeys);
          }}
        >
          <div className='flex items-center justify-center mb-6 mt-4'>
            <HeaderLogo
              isMobile={false}
              isConsoleRoute={true}
              logo={logo}
              logoLoaded={true}
              isLoading={false}
              systemName={systemName}
              collapsed={collapsed}
            />
          </div>

          <div className='sidebar-nav-list-container relative' onMouseOver={handleMouseOver} onMouseLeave={handleMouseLeave}>
            {/* The sliding hover box */}
            {hoveredNode && (
              <div 
                className="absolute left-2 right-2 rounded-xl bg-white/5 pointer-events-none transition-all duration-300 ease-out z-0"
                style={{
                  top: hoveredNode.top,
                  height: hoveredNode.height,
                  opacity: hoveredNode.opacity,
                }}
              />
            )}
            
            {overviewItems.length > 0 && (
              <div className='sidebar-section'>
                {!collapsed && (
                  <div className='mb-2 px-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-white/35'>
                    {t('概览')}
                  </div>
                )}
                {overviewItems.map((item) => renderNavItem(item))}
              </div>
            )}

            {workspaceItems.length > 0 && (
              <>
                <Divider className='!my-3 !border-white/8' />
                <div>
                  {!collapsed && (
                    <div className='mb-2 px-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-white/35'>
                      {t('工作区')}
                    </div>
                  )}
                  {workspaceItems.map((item) => renderNavItem(item))}
                </div>
              </>
            )}

            {resourceItems.length > 0 && (
              <>
                <Divider className='!my-3 !border-white/8' />
                <div>
                  {!collapsed && (
                    <div className='mb-2 px-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-white/35'>
                      {t('资源')}
                    </div>
                  )}
                  {resourceItems.map((item) => renderNavItem(item))}
                </div>
              </>
            )}

            {systemItems.length > 0 && (
              <>
                <Divider className='!my-3 !border-white/8' />
                <div>
                  {!collapsed && (
                    <div className='mb-2 px-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-white/35'>
                      {t('系统')}
                    </div>
                  )}
                  {systemItems.map((item) => renderNavItem(item))}
                </div>
              </>
            )}
          </div>
        </Nav>
      </SkeletonWrapper>

      <div className={`${collapsed ? 'mt-3 px-0' : 'mt-3 px-3'}`}>
        <SkeletonWrapper
          loading={showSkeleton}
          type='button'
          width={collapsed ? 40 : '100%'}
          height={40}
          className='w-full'
        >
          <Button
            theme='borderless'
            type='tertiary'
            size='small'
            icon={
              <ChevronLeft
                size={16}
                strokeWidth={2.5}
                color='rgba(255,255,255,0.7)'
                style={{
                  transform: collapsed ? 'rotate(180deg)' : 'rotate(0deg)',
                }}
              />
            }
            onClick={toggleCollapsed}
            icononly={collapsed}
            className='!h-10 !rounded-xl !border !border-white/10 !bg-white/5 !text-white hover:!bg-white/10'
            style={
              collapsed
                ? { width: 40, padding: 0 }
                : { padding: '0 14px', width: '100%' }
            }
          >
            {!collapsed ? t('收起侧边栏') : null}
          </Button>
        </SkeletonWrapper>
      </div>
      </div>
    </div>
  );
};

export default SiderBar;
