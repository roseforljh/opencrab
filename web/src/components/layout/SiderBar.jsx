
import React, { useEffect, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import { getLogo, getSystemName } from '../../helpers';
import HeaderLogo from './headerbar/HeaderLogo';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';

const routerMap = {
  home: '/',
  channel: '/console/channel',
  models: '/console/models',
  token: '/console/token',
  setting: '/console/setting',
  personal: '/console/personal',
};

const SiderBar = ({ onNavigate = () => {} }) => {
  const [collapsed, toggleCollapsed] = useSidebarCollapsed();
  const [selectedKeys, setSelectedKeys] = useState(['channel']);
  const location = useLocation();
  const [routerMapState] = useState(routerMap);

  const logo = getLogo();
  const systemName = getSystemName();

  const items = useMemo(() => {
    return [
      {
        text: '渠道管理',
        itemKey: 'channel',
      },
      {
        text: '模型管理',
        itemKey: 'models',
      },
      {
        text: '令牌管理',
        itemKey: 'token',
      },
      {
        text: '个人中心',
        itemKey: 'personal',
      },
      {
        text: '系统设置',
        itemKey: 'setting',
      },
    ];
  }, []);

  useEffect(() => {
    const currentPath = location.pathname;
    const matchingKey = Object.keys(routerMapState).find(
      (key) => routerMapState[key] === currentPath,
    );

    if (matchingKey) {
      setSelectedKeys([matchingKey]);
    }
  }, [location.pathname, routerMapState]);

  useEffect(() => {
    if (collapsed) {
      document.body.classList.add('sidebar-collapsed');
    } else {
      document.body.classList.remove('sidebar-collapsed');
    }
  }, [collapsed]);

  return (
    <div className='flex h-full flex-col border-r border-white/10 bg-[#050816]/95 text-white backdrop-blur-xl'>
      <div className='flex items-center justify-between px-4 py-4'>
        <HeaderLogo
          collapsed={collapsed}
          logo={logo}
          logoLoaded={true}
          systemName={systemName}
          isLoading={false}
        />
        <Button
          variant='ghost'
          size='icon'
          onClick={toggleCollapsed}
          className='text-white/80 hover:bg-white/10 hover:text-white'
        >
          <span>{collapsed ? '→' : '←'}</span>
        </Button>
      </div>

      <Separator className='my-3 bg-white/10' />

      <div className='flex-1 overflow-y-auto px-2'>
        <nav className='flex flex-col space-y-1'>
          {items.map((item) => {
            const isActive = selectedKeys.includes(item.itemKey);
            return (
              <button
                key={item.itemKey}
                onClick={() => {
                  const target = routerMapState[item.itemKey];
                  if (target && location.pathname !== target) {
                    window.location.href = target;
                    onNavigate();
                  }
                }}
                className={`flex items-center w-full px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
                  isActive
                    ? 'bg-white/15 text-white'
                    : 'text-white/70 hover:bg-white/10 hover:text-white'
                } ${collapsed ? 'justify-center' : 'justify-start'}`}
                title={collapsed ? item.text : undefined}
              >
                {!collapsed && <span>{item.text}</span>}
                {collapsed && (
                  <span className='text-xs truncate'>
                    {item.text.slice(0, 2)}
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      </div>
    </div>
  );
};

export default SiderBar;
