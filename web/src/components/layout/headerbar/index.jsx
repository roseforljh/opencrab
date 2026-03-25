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
