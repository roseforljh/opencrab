
import React from 'react';
import { Typography } from '@douyinfe/semi-ui';
import { Key } from 'lucide-react';
import CompactModeToggle from '../../common/ui/CompactModeToggle';

const { Text } = Typography;

const TokensDescription = ({ compactMode, setCompactMode, t }) => {
  return (
    <div className='flex w-full flex-col gap-3 md:flex-row md:items-center md:justify-between'>
      <div className='flex items-start gap-3'>
        <div className='rounded-2xl border border-white/10 bg-[linear-gradient(135deg,rgba(77,162,255,0.18),rgba(139,125,255,0.12))] p-2 text-white shadow-[0_12px_24px_rgba(77,162,255,0.15)]'>
          <Key size={18} />
        </div>
        <div>
          <Text className='!font-semibold !text-white'>{t('访问密钥')}</Text>
          <div className='mt-1 text-xs text-white/45'>
            {t('优先处理新增、复制、删除与搜索，减少无关说明和平台化装饰。')}
          </div>
        </div>
      </div>

      <CompactModeToggle
        compactMode={compactMode}
        setCompactMode={setCompactMode}
        t={t}
      />
    </div>
  );
};

export default TokensDescription;
