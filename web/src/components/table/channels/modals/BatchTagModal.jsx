import React from 'react';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

const BatchTagModal = ({
  showBatchSetTag,
  setShowBatchSetTag,
  batchSetChannelTag,
  batchSetTagValue,
  setBatchSetTagValue,
  selectedChannels,
  t,
}) => {
  return (
    <Dialog
      open={showBatchSetTag}
      onOpenChange={(open) => !open && setShowBatchSetTag(false)}
    >
      <DialogContent className='max-w-[420px] border-white/10 bg-black text-white'>
        <DialogHeader>
          <DialogTitle>{t('批量设置标签')}</DialogTitle>
        </DialogHeader>
        <div className='mb-5 text-sm text-white/70'>
          {t('请输入要设置的标签名称')}
        </div>
        <Input
          placeholder={t('请输入标签名称')}
          value={batchSetTagValue}
          onChange={(e) => setBatchSetTagValue(e.target.value)}
          className='border-white/10 bg-white/6 text-white'
        />
        <div className='mt-4 text-sm text-white/60'>
          {t('已选择 ${count} 个渠道').replace(
            '${count}',
            selectedChannels.length,
          )}
        </div>
        <DialogFooter className='border-white/10 bg-transparent'>
          <Button
            type='button'
            variant='secondary'
            onClick={() => setShowBatchSetTag(false)}
          >
            {t('取消')}
          </Button>
          <Button type='button' onClick={batchSetChannelTag}>
            {t('确定')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default BatchTagModal;
