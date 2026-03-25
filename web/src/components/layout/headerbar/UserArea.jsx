
import React, { useRef } from 'react';
import { Link } from 'react-router-dom';
import { Avatar, Button, Dropdown, Typography } from '@douyinfe/semi-ui';
import { ChevronDown } from 'lucide-react';
import { IconExit, IconUserSetting, IconKey } from '@douyinfe/semi-icons';
import { stringToColor } from '../../../helpers';
import SkeletonWrapper from '../components/SkeletonWrapper';

const UserArea = ({ userState, isLoading, isMobile, logout, navigate, t }) => {
  const dropdownRef = useRef(null);
  if (isLoading) {
    return (
      <SkeletonWrapper
        loading={true}
        type='userArea'
        width={50}
        isMobile={isMobile}
      />
    );
  }

  if (userState.user) {
    return (
      <div className='relative' ref={dropdownRef}>
        <Dropdown
          position='bottomRight'
          getPopupContainer={() => dropdownRef.current}
          render={
            <Dropdown.Menu className='!rounded-2xl !border !border-white/10 !bg-[#0b1220] !p-2 !shadow-[0_24px_80px_rgba(0,0,0,0.45)]'>
              <Dropdown.Item
                onClick={() => {
                  navigate('/console/personal');
                }}
                className='!rounded-xl !px-3 !py-2 !text-sm !text-white/85 hover:!bg-white/6 hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  <IconUserSetting
                    size='small'
                    className='text-white/50'
                  />
                  <span>{t('个人设置')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item
                onClick={() => {
                  navigate('/console/token');
                }}
                className='!rounded-xl !px-3 !py-2 !text-sm !text-white/85 hover:!bg-white/6 hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  <IconKey
                    size='small'
                    className='text-white/50'
                  />
                  <span>{t('令牌管理')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item
                onClick={logout}
                className='!rounded-xl !px-3 !py-2 !text-sm !text-white/85 hover:!bg-[#351515] hover:!text-white'
              >
                <div className='flex items-center gap-2'>
                  <IconExit
                    size='small'
                    className='text-white/50'
                  />
                  <span>{t('退出')}</span>
                </div>
              </Dropdown.Item>
            </Dropdown.Menu>
          }
        >
          <Button
            theme='borderless'
            type='tertiary'
            className='!flex !items-center !gap-1.5 !rounded-2xl !border !border-white/10 !bg-white/6 !p-1.5 hover:!bg-white/10'
          >
            <Avatar
              size='extra-small'
              color={stringToColor(userState.user.username)}
              className='mr-1'
            >
              {userState.user.username[0].toUpperCase()}
            </Avatar>
            <span className='hidden md:inline'>
              <Typography.Text className='!mr-1 !text-xs !font-medium !text-white'>
                {userState.user.username}
              </Typography.Text>
            </span>
            <ChevronDown
              size={14}
              className='text-xs text-white/55'
            />
          </Button>
        </Dropdown>
      </div>
    );
  } else {
    const showRegisterButton = false;

    const commonSizingAndLayoutClass =
      'flex items-center justify-center !py-[10px] !px-1.5';

    const loginButtonSpecificStyling =
      '!bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700 transition-colors';
    let loginButtonClasses = `${commonSizingAndLayoutClass} ${loginButtonSpecificStyling}`;

    let registerButtonClasses = `${commonSizingAndLayoutClass}`;

    const loginButtonTextSpanClass =
      '!text-xs !text-semi-color-text-1 dark:!text-gray-300 !p-1.5';
    const registerButtonTextSpanClass = '!text-xs !text-white !p-1.5';

    if (showRegisterButton) {
      if (isMobile) {
        loginButtonClasses += ' !rounded-full';
      } else {
        loginButtonClasses += ' !rounded-l-full !rounded-r-none';
      }
      registerButtonClasses += ' !rounded-r-full !rounded-l-none';
    } else {
      loginButtonClasses += ' !rounded-full';
    }

    return (
      <div className='flex items-center'>
        <Link to='/login' className='flex'>
          <Button
            theme='borderless'
            type='tertiary'
            className={loginButtonClasses}
          >
            <span className={loginButtonTextSpanClass}>{t('登录')}</span>
          </Button>
        </Link>
        {showRegisterButton && (
          <div className='hidden md:block'>
            <Link to='/register' className='flex -ml-px'>
              <Button
                theme='solid'
                type='primary'
                className={registerButtonClasses}
              >
                <span className={registerButtonTextSpanClass}>{t('注册')}</span>
              </Button>
            </Link>
          </div>
        )}
      </div>
    );
  }
};

export default UserArea;
