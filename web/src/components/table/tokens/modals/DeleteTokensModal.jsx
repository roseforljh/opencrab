
import React from 'react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

const DeleteTokensModal = ({
  visible,
  onCancel,
  onConfirm,
  selectedKeys,
  t,
}) => {
  return (
    <AlertDialog open={visible} onOpenChange={(open) => !open && onCancel()}>
      <AlertDialogContent className='border-white/10 bg-black text-white'>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('批量删除令牌')}</AlertDialogTitle>
          <AlertDialogDescription className='text-white/60'>
            {t('确定要删除所选的 {{count}} 个令牌吗？', {
              count: selectedKeys.length,
            })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel className='border-0 bg-white/5 hover:bg-white/10'>
            {t('取消')}
          </AlertDialogCancel>
          <AlertDialogAction
            className='bg-red-500 text-white hover:bg-red-600'
            onClick={onConfirm}
          >
            {t('确认删除')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
};

export default DeleteTokensModal;
