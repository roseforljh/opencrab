
import React from 'react';
import { Tabs, TabPane, Tag, Typography } from '@douyinfe/semi-ui';
import { CHANNEL_OPTIONS } from '../../../constants';
import { getChannelIcon } from '../../../helpers';

const ChannelsTabs = ({
  enableTagMode,
  activeTypeKey,
  setActiveTypeKey,
  channelTypeCounts,
  availableTypeKeys,
  loadChannels,
  pageSize,
  idSort,
  setActivePage,
  t,
}) => {
  if (enableTagMode) return null;

  const handleTabChange = (key) => {
    setActiveTypeKey(key);
    setActivePage(1);
    loadChannels(1, pageSize, idSort, enableTagMode, key);
  };

  return (
    <div className='space-y-3'>
      <div>
        <Typography.Text strong className='!text-sm !text-white'>
          {t('渠道类型')}
        </Typography.Text>
        <div className='mt-1 text-xs text-white/45'>
          {t('先按渠道类型收窄范围，再继续查看可用性和配置问题。')}
        </div>
      </div>

      <Tabs
        activeKey={activeTypeKey}
        type='card'
        collapsible
        onChange={handleTabChange}
        className='mb-2'
      >
        <TabPane
          itemKey='all'
          tab={
            <span className='flex items-center gap-2'>
              {t('全部')}
              <Tag
                color={activeTypeKey === 'all' ? 'blue' : 'grey'}
                shape='circle'
              >
                {channelTypeCounts['all'] || 0}
              </Tag>
            </span>
          }
        />

        {CHANNEL_OPTIONS.filter((opt) =>
          availableTypeKeys.includes(String(opt.value)),
        ).map((option) => {
          const key = String(option.value);
          const count = channelTypeCounts[option.value] || 0;
          return (
            <TabPane
              key={key}
              itemKey={key}
              tab={
                <span className='flex items-center gap-2'>
                  {getChannelIcon(option.value)}
                  {option.label}
                  <Tag
                    color={activeTypeKey === key ? 'blue' : 'grey'}
                    shape='circle'
                  >
                    {count}
                  </Tag>
                </span>
              }
            />
          );
        })}
      </Tabs>
    </div>
  );
};

export default ChannelsTabs;
