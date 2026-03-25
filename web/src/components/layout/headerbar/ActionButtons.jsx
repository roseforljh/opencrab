
import React from 'react';
import ThemeToggle from './ThemeToggle';
import UserArea from './UserArea';

const ActionButtons = ({
  theme,
  onThemeToggle,
  userState,
  isLoading,
  isMobile,
  logout,
  navigate,
  t,
}) => {
  return (
    <div className='flex items-center gap-2 md:gap-3 rounded-2xl border border-white/10 bg-white/5 px-2 py-1.5 shadow-[0_10px_30px_rgba(0,0,0,0.18)]'>
      <ThemeToggle theme={theme} onThemeToggle={onThemeToggle} t={t} />

      <UserArea
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

export default ActionButtons;
