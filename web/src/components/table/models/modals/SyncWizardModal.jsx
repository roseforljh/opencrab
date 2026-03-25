import React, { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const SyncWizardModal = ({ visible, onClose, onConfirm, loading, t }) => {
  const [step, setStep] = useState(0);
  const [option, setOption] = useState('official');
  const [locale, setLocale] = useState('zh-CN');
  const isMobile = useIsMobile();

  useEffect(() => {
    if (visible) {
      setStep(0);
      setOption('official');
      setLocale('zh-CN');
    }
  }, [visible]);

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[520px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          <DialogTitle>{t('同步向导')}</DialogTitle>
        </DialogHeader>
        <div className='mb-3'>
          <div className='grid grid-cols-2 gap-3 text-sm'>
            <div
              className={`rounded-xl border px-4 py-3 ${step === 0 ? 'border-white/30 bg-white/10 text-white' : 'border-white/10 bg-white/5 text-white/50'}`}
            >
              <div className='font-medium'>{t('选择方式')}</div>
              <div className='mt-1 text-xs'>{t('选择同步来源')}</div>
            </div>
            <div
              className={`rounded-xl border px-4 py-3 ${step === 1 ? 'border-white/30 bg-white/10 text-white' : 'border-white/10 bg-white/5 text-white/50'}`}
            >
              <div className='font-medium'>{t('选择语言')}</div>
              <div className='mt-1 text-xs'>{t('选择同步语言')}</div>
            </div>
          </div>
        </div>

        {step === 0 && (
          <div className='mt-2 grid gap-3'>
            <button
              type='button'
              className={`rounded-2xl border p-4 text-left ${option === 'official' ? 'border-white/30 bg-white/10' : 'border-white/10 bg-white/5'}`}
              onClick={() => setOption('official')}
            >
              <div className='font-medium text-white'>{t('官方模型同步')}</div>
              <div className='mt-1 text-sm text-white/60'>
                {t('从官方模型库同步')}
              </div>
            </button>
            <button
              type='button'
              disabled
              className='rounded-2xl border border-white/10 bg-white/5 p-4 text-left opacity-50'
            >
              <div className='font-medium text-white'>{t('配置文件同步')}</div>
              <div className='mt-1 text-sm text-white/60'>
                {t('从配置文件同步')}
              </div>
            </button>
          </div>
        )}

        {step === 1 && (
          <div className='mt-2'>
            <div className='mb-2 text-white/60'>{t('请选择同步语言')}</div>
            <div className='grid grid-cols-2 gap-3'>
              {[
                ['en', 'English'],
                ['zh-CN', '简体中文'],
                ['zh-TW', '繁體中文'],
                ['ja', '日本語'],
              ].map(([value, label]) => (
                <button
                  key={value}
                  type='button'
                  className={`rounded-2xl border p-4 text-left ${locale === value ? 'border-white/30 bg-white/10' : 'border-white/10 bg-white/5'}`}
                  onClick={() => setLocale(value)}
                >
                  <div className='font-medium text-white'>{value}</div>
                  <div className='mt-1 text-sm text-white/60'>{label}</div>
                </button>
              ))}
            </div>
          </div>
        )}

        <DialogFooter className='border-white/10 bg-transparent'>
          {step === 1 && (
            <Button
              type='button'
              variant='secondary'
              onClick={() => setStep(0)}
            >
              {t('上一步')}
            </Button>
          )}
          <Button type='button' variant='secondary' onClick={onClose}>
            {t('取消')}
          </Button>
          {step === 0 && (
            <Button
              type='button'
              onClick={() => setStep(1)}
              disabled={option !== 'official'}
            >
              {t('下一步')}
            </Button>
          )}
          {step === 1 && (
            <Button
              type='button'
              onClick={async () => {
                await onConfirm?.({ option, locale });
              }}
              disabled={loading}
            >
              {t('开始同步')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default SyncWizardModal;
