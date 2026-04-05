
import React, { useEffect, useState } from 'react';
import SiderBar from './SiderBar';
import App from '../../App';
import { ToastContainer } from 'react-toastify';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useSidebarCollapsed } from '../../hooks/common/useSidebarCollapsed';
import {
  API,
  getLogo,
  getSystemName,
  setStatusData,
  showError,
} from '../../helpers';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { useLocation } from 'react-router-dom';
import ShaderBackground from '../ui/backgrounds/ShaderBackground';

const PageLayout = () => {
  const [, userDispatch] = React.useContext(UserContext);
  const [, statusDispatch] = React.useContext(StatusContext);
  const isMobile = useIsMobile();
  const [collapsed, , setCollapsed] = useSidebarCollapsed();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const location = useLocation();

  const isConsoleRoute = location.pathname.startsWith('/console');
  const isAuthRoute =
    location.pathname === '/' || location.pathname === '/login';
  const showSider = isConsoleRoute && (!isMobile || drawerOpen);

  useEffect(() => {
    if (isMobile && drawerOpen && collapsed) {
      setCollapsed(false);
    }
  }, [isMobile, drawerOpen, collapsed, setCollapsed]);

  useEffect(() => {
    const user = localStorage.getItem('user');
    if (user) {
      userDispatch({ type: 'login', payload: JSON.parse(user) });
    }
  }, [userDispatch]);

  useEffect(() => {
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

    loadStatus().catch(console.error);

    const systemName = getSystemName();
    if (systemName) {
      document.title = systemName;
    }
    const logo = getLogo();
    if (logo) {
      const linkElement = document.querySelector("link[rel~='icon']");
      if (linkElement) {
        linkElement.href = logo;
      }
    }
  }, [statusDispatch]);

  return (
    <div className='app-layout min-h-screen flex flex-col relative isolate'>
      <div className='fixed inset-0 z-0 bg-black pointer-events-none' />
      
      {isConsoleRoute && <ShaderBackground />}
      
      <div 
        className='fixed inset-0 z-0 pointer-events-none'
        style={{
          backgroundImage:
            'linear-gradient(rgba(255,255,255,0.045) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.045) 1px, transparent 1px)',
          backgroundSize: '28px 28px',
          backgroundPosition: 'center center',
        }}
      />

      <div className='flex min-h-screen flex-row relative z-10'>
        {showSider && (
          <aside
            className='app-sider'
            style={{
              position: isMobile ? 'fixed' : 'sticky',
              left: 0,
              top: 0,
              zIndex: 99,
              width: 'var(--sidebar-current-width)',
              flexShrink: 0,
              height: '100vh',
              transition: 'width 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
            }}
          >
            <SiderBar
              onNavigate={() => {
                if (isMobile) setDrawerOpen(false);
              }}
            />
          </aside>
        )}
        <div className='flex flex-1 flex-col min-w-0'>
          <main
            className='flex-1'
            style={{
              padding: isConsoleRoute
                ? isMobile
                  ? '12px 8px 20px'
                  : '16px 20px 24px'
                : '0',
              paddingTop: isConsoleRoute ? '16px' : '0',
              background: 'transparent',
            }}
          >
            <App />
          </main>
        </div>
      </div>
      <ToastContainer />
    </div>
  );
};

export default PageLayout;
