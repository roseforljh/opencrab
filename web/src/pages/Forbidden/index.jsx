
import React from 'react';
import { Empty } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const Forbidden = () => {
  const { t } = useTranslation();
  return (
    <div className='flex h-screen items-center justify-center p-8 bg-black'>
      <Empty
        image={
          <div className='flex h-[160px] w-[160px] items-center justify-center rounded-[36px] border border-white/10 bg-black/50 text-5xl text-white/25'>
            ⛶
          </div>
        }
        title={t('访问受限')}
        description={t('当前账户没有权限进入此页面，如有需要请联系管理员。')}
      />
    </div>
  );
};

export default Forbidden;
