
import React from 'react';
import ChannelsTable from '../../components/table/channels';

const Channel = () => {
  return (
    <div className='space-y-4'>
      <div className='rounded-2xl border border-white/10 bg-white/6 px-5 py-4 md:px-6 backdrop-blur-xl'>
        <div className='flex items-start justify-between gap-4'>
          <div>
            <h1 className='text-xl font-semibold text-white'>
              渠道管理
            </h1>
            <p className='mt-1 text-sm text-white/60'>
              在这里集中处理上游渠道的新增、测试、启停、批量操作与可用性检查，优先保证真正可用的渠道排在前面。
            </p>
          </div>
        </div>
      </div>

      <ChannelsTable />
    </div>
  );
};

export default Channel;
