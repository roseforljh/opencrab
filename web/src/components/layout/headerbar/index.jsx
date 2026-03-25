
import React from 'react';
import { useHeaderBar } from '../../../hooks/common/useHeaderBar';
import MobileMenuButton from './MobileMenuButton';
import HeaderLogo from './HeaderLogo';
import ActionButtons from './ActionButtons';

const HeaderBar = ({ onMobileMenuToggle, drawerOpen }) => {
  const {
    userState,
    isMobile,
    collapsed,
    logoLoaded,
    isLoading,
    systemName,
    logo,
    isConsoleRoute,
    theme,
    logout,
    handleThemeToggle,
    handleMobileMenuToggle,
    navigate,
    t,
  } = useHeaderBar({ onMobileMenuToggle, drawerOpen });

  return (
    <div className='flex items-center justify-end'>
      {isMobile && (
        <MobileMenuButton
          isConsoleRoute={isConsoleRoute}
          isMobile={isMobile}
          drawerOpen={drawerOpen}
          collapsed={collapsed}
          onToggle={handleMobileMenuToggle}
          t={t}
        />
      )}
      <ActionButtons
        theme={theme}
        onThemeToggle={handleThemeToggle}
        userState={userState}
        isLoading={isLoading}
        isMobile={isMobile}
        logout={logout}
        navigate={navigate}
        t={t}
      />
    </div>
  );
};

export default HeaderBar;
