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

import HeaderBar from './headerbar';
import { Layout } from '@douyinfe/semi-ui';
import SiderBar from './SiderBar';
import App from '../../App';
import FooterBar from './Footer';
import { ToastContainer } from 'react-toastify';
import React, { useContext, useEffect, useState } from 'react';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import {
  API,
  getLogo,
  getSystemName,
  showError,
  setStatusData,
} from '../../helpers';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { useLocation } from 'react-router-dom';
const { Sider, Content, Header } = Layout;

const PageLayout = () => {
  const [userState, userDispatch] = useContext(UserContext);
  const [, statusDispatch] = useContext(StatusContext);
  const isMobile = useIsMobile();
  const [collapsed, , setCollapsed] = useSidebarCollapsed();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const location = useLocation();

  const isConsoleRoute = location.pathname.startsWith('/console');
  const isFullWorkspace =
    location.pathname === '/console/playground' ||
    location.pathname.startsWith('/console/chat');
  const isSettingsRoute =
    location.pathname === '/console/setting' ||
    location.pathname === '/console/personal';
  const isWideTableRoute = [
    '/console/channel',
    '/console/log',
    '/console/token',
    '/console/models',
    '/console/deployment',
    '/console/user',
    '/console/task',
    '/console/midjourney',
  ].includes(location.pathname);
  const shouldHideFooter = isConsoleRoute;
  const contentPadding = isConsoleRoute
    ? isFullWorkspace
      ? '0'
      : isSettingsRoute
        ? isMobile
          ? '16px 12px 24px'
          : '24px 32px 32px'
        : isWideTableRoute
          ? isMobile
            ? '12px 8px 20px'
            : '16px 20px 24px'
          : isMobile
            ? '16px 12px 24px'
            : '20px 24px 28px'
    : '0';
  const showSider = isConsoleRoute && (!isMobile || drawerOpen);

  useEffect(() => {
    if (isMobile && drawerOpen && collapsed) {
      setCollapsed(false);
    }
  }, [isMobile, drawerOpen, collapsed, setCollapsed]);

  const loadUser = () => {
    let user = localStorage.getItem('user');
    if (user) {
      let data = JSON.parse(user);
      userDispatch({ type: 'login', payload: data });
    }
  };

  const loadStatus = async () => {
    try {
      const res = await API.get('/api/status');
      const { success, data } = res.data;
      if (success) {
        statusDispatch({ type: 'set', payload: data });
        setStatusData(data);
      } else {
        showError('Unable to connect to server');
      }
    } catch (error) {
      showError('Failed to load status');
    }
  };

  useEffect(() => {
    loadUser();
    loadStatus().catch(console.error);
    let systemName = getSystemName();
    if (systemName) {
      document.title = systemName;
    }
    let logo = getLogo();
    if (logo) {
      let linkElement = document.querySelector("link[rel~='icon']");
      if (linkElement) {
        linkElement.href = logo;
      }
    }
  }, []);

  return (
    <Layout
      className='app-layout'
      style={{
        display: 'flex',
        flexDirection: 'column',
        overflow: isMobile ? 'visible' : 'hidden',
        minHeight: '100vh',
        backgroundColor: '#000000',
        backgroundImage:
          'linear-gradient(rgba(255,255,255,0.045) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.045) 1px, transparent 1px)',
        backgroundSize: '28px 28px',
        backgroundPosition: 'center center',
      }}
    >
      <Layout
        style={{
          overflow: isMobile ? 'visible' : 'hidden',
          display: 'flex',
          flexDirection: 'row',
          flex: '1',
          position: 'relative',
        }}
      >
        {showSider && (
          <Sider
            className='app-sider'
            style={{
              position: isMobile ? 'fixed' : 'relative',
              left: 0,
              top: 0,
              zIndex: 99,
              border: 'none',
              paddingRight: '0',
              width: 'var(--sidebar-current-width)',
              flexShrink: 0,
              transition: 'width 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
              height: '100vh',
            }}
          >
            <SiderBar
              onNavigate={() => {
                if (isMobile) setDrawerOpen(false);
              }}
            />
          </Sider>
        )}
        <Layout
          style={{
            flex: '1 1 0',
            minWidth: 0,
            display: 'flex',
            flexDirection: 'column',
            position: 'relative',
            transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
          }}
        >
          <div style={{ position: 'absolute', top: '16px', right: '24px', zIndex: 100 }}>
             <HeaderBar
              onMobileMenuToggle={() => setDrawerOpen((prev) => !prev)}
              drawerOpen={drawerOpen}
            />
          </div>
          <Content
            style={{
              flex: '1 0 auto',
              overflowY: isMobile ? 'visible' : 'hidden',
              WebkitOverflowScrolling: 'touch',
              padding: contentPadding,
              paddingTop: '72px',
              position: 'relative',
              background: 'transparent',
            }}
          >
            <App />
          </Content>
          {!shouldHideFooter && (
            <Layout.Footer
              style={{
                flex: '0 0 auto',
                width: '100%',
              }}
            >
              <FooterBar />
            </Layout.Footer>
          )}
        </Layout>
      </Layout>
      <ToastContainer />
    </Layout>
  );
};

export default PageLayout;
