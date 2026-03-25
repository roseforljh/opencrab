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
