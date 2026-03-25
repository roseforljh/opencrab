
import React from 'react';
import { Link } from 'react-router-dom';
import { Typography } from '@douyinfe/semi-ui';
import SkeletonWrapper from '../components/SkeletonWrapper';

const HeaderLogo = ({
  isMobile,
  isConsoleRoute,
  logo,
  logoLoaded,
  isLoading,
  systemName,
  collapsed = false,
}) => {
  if (isMobile && isConsoleRoute) {
    return null;
  }

  return (
    <Link to='/' className='group flex items-center gap-3'>
      <div className='relative h-10 w-10 overflow-hidden rounded-2xl border border-white/10 bg-white/10 shadow-[0_8px_32px_rgba(82,169,255,0.18)]'>
        <SkeletonWrapper loading={isLoading || !logoLoaded} type='image' />
        <img
          src={logo}
          alt='logo'
          className={`absolute inset-0 h-full w-full rounded-2xl object-cover transition-all duration-300 group-hover:scale-110 ${!isLoading && logoLoaded ? 'opacity-100' : 'opacity-0'}`}
        />
      </div>
      {!collapsed && (
        <div className='hidden md:flex items-center gap-2'>
          <SkeletonWrapper
            loading={isLoading}
            type='title'
            width={140}
            height={24}
          >
            <Typography.Title
              heading={4}
              className='!mb-0 !text-lg !font-semibold !tracking-[0.02em] !text-white'
            >
              {systemName}
            </Typography.Title>
          </SkeletonWrapper>
        </div>
      )}
    </Link>
  );
};

export default HeaderLogo;
