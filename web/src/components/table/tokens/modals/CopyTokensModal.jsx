
import React from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

const CopyTokensModal = ({ visible, onCancel, batchCopyTokens, t }) => {
  // Handle copy with name and key format
  const handleCopyWithName = async () => {
    await batchCopyTokens('name+key');
    onCancel();
  };

  // Handle copy with key only format
  const handleCopyKeyOnly = async () => {
    await batchCopyTokens('key-only');
    onCancel();
  };

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className='border-white/10 bg-black text-white'>
        <DialogHeader>
          <DialogTitle>{t('复制令牌')}</DialogTitle>
          <DialogDescription className='text-white/60'>
            {t('请选择你的复制方式')}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className='border-white/10 bg-transparent'>
          <Button variant='secondary' onClick={handleCopyWithName}>
            {t('名称+密钥')}
          </Button>
          <Button onClick={handleCopyKeyOnly}>{t('仅密钥')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default CopyTokensModal;
