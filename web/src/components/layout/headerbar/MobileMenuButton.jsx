
import React from 'react';
import { Button } from '@douyinfe/semi-ui';
import { IconClose, IconMenu } from '@douyinfe/semi-icons';

const MobileMenuButton = ({
  isConsoleRoute,
  isMobile,
  drawerOpen,
  collapsed,
  onToggle,
  t,
}) => {
  if (!isConsoleRoute || !isMobile) {
    return null;
  }

  return (
    <Button
      icon={
        (isMobile ? drawerOpen : collapsed) ? (
          <IconClose className='text-lg' />
        ) : (
          <IconMenu className='text-lg' />
        )
      }
      aria-label={
        (isMobile ? drawerOpen : collapsed) ? t('关闭侧边栏') : t('打开侧边栏')
      }
      onClick={onToggle}
      theme='borderless'
      type='tertiary'
      className='!rounded-2xl !border !border-white/10 !bg-white/5 !p-2.5 !text-white transition-all hover:!bg-white/10 focus:!bg-white/10'
    />
  );
};

export default MobileMenuButton;
