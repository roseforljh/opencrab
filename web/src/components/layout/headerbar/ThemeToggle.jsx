
import React, { useMemo } from 'react';
import { Button, Dropdown } from '@douyinfe/semi-ui';
import { Sun, Moon, Monitor } from 'lucide-react';
import { useActualTheme } from '../../../context/Theme';

const ThemeToggle = ({ theme, onThemeToggle, t }) => {
  const actualTheme = useActualTheme();

  const themeOptions = useMemo(
    () => [
      {
        key: 'light',
        icon: <Sun size={18} />,
        buttonIcon: <Sun size={18} />,
        label: t('浅色模式'),
        description: t('始终使用浅色主题'),
      },
      {
        key: 'dark',
        icon: <Moon size={18} />,
        buttonIcon: <Moon size={18} />,
        label: t('深色模式'),
        description: t('始终使用深色主题'),
      },
      {
        key: 'auto',
        icon: <Monitor size={18} />,
        buttonIcon: <Monitor size={18} />,
        label: t('自动模式'),
        description: t('跟随系统主题设置'),
      },
    ],
    [t],
  );

  const getItemClassName = (isSelected) =>
    isSelected
      ? '!rounded-xl !bg-white/10 !font-semibold !text-white'
      : '!rounded-xl !text-white/80 hover:!bg-white/5 hover:!text-white';

  const currentButtonIcon = useMemo(() => {
    const currentOption = themeOptions.find((option) => option.key === theme);
    return currentOption?.buttonIcon || themeOptions[2].buttonIcon;
  }, [theme, themeOptions]);

  return (
    <Dropdown
      position='bottomRight'
      render={
        <Dropdown.Menu className='!rounded-2xl !border !border-white/10 !bg-[#0b1220] !p-2 !text-white !shadow-[0_24px_80px_rgba(0,0,0,0.45)]'>
          {themeOptions.map((option) => (
            <Dropdown.Item
              key={option.key}
              icon={option.icon}
              onClick={() => onThemeToggle(option.key)}
              className={getItemClassName(theme === option.key)}
            >
              <div className='flex flex-col'>
                <span>{option.label}</span>
                <span className='text-xs text-white/55'>
                  {option.description}
                </span>
              </div>
            </Dropdown.Item>
          ))}

          {theme === 'auto' && (
            <>
              <Dropdown.Divider className='!border-white/10' />
              <div className='px-3 py-2 text-xs text-white/55'>
                {t('当前跟随系统')}：
                {actualTheme === 'dark' ? t('深色') : t('浅色')}
              </div>
            </>
          )}
        </Dropdown.Menu>
      }
    >
      <Button
        icon={currentButtonIcon}
        aria-label={t('切换主题')}
        theme='borderless'
        type='tertiary'
        className='!rounded-2xl !border !border-white/10 !bg-white/6 !p-2.5 !text-white transition-all hover:!bg-white/10 focus:!bg-white/10'
      />
    </Dropdown>
  );
};

export default ThemeToggle;
