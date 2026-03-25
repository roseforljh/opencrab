import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { API, copy, showError, showSuccess } from '../../../../helpers';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { AlertTriangle } from 'lucide-react';

const CodexOAuthModal = ({ visible, onCancel, onSuccess }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [authorizeUrl, setAuthorizeUrl] = useState('');
  const [input, setInput] = useState('');

  const startOAuth = async () => {
    setLoading(true);
    try {
      const res = await API.post(
        '/api/channel/codex/oauth/start',
        {},
        { skipErrorHandler: true },
      );
      if (!res?.data?.success) {
        console.error('Codex OAuth start failed:', res?.data?.message);
        throw new Error(t('启动授权失败'));
      }
      const url = res?.data?.data?.authorize_url || '';
      if (!url) {
        console.error(
          'Codex OAuth start response missing authorize_url:',
          res?.data,
        );
        throw new Error(t('响应缺少授权链接'));
      }
      setAuthorizeUrl(url);
      window.open(url, '_blank', 'noopener,noreferrer');
      showSuccess(t('已打开授权页面'));
    } catch (error) {
      showError(error?.message || t('启动授权失败'));
    } finally {
      setLoading(false);
    }
  };

  const completeOAuth = async () => {
    if (!input || !input.trim()) {
      showError(t('请先粘贴回调 URL'));
      return;
    }

    setLoading(true);
    try {
      const res = await API.post(
        '/api/channel/codex/oauth/complete',
        { input },
        { skipErrorHandler: true },
      );
      if (!res?.data?.success) {
        console.error('Codex OAuth complete failed:', res?.data?.message);
        throw new Error(t('授权失败'));
      }

      const key = res?.data?.data?.key || '';
      if (!key) {
        console.error('Codex OAuth complete response missing key:', res?.data);
        throw new Error(t('响应缺少凭据'));
      }

      onSuccess && onSuccess(key);
      showSuccess(t('已生成授权凭据'));
      onCancel && onCancel();
    } catch (error) {
      showError(error?.message || t('授权失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) return;
    setAuthorizeUrl('');
    setInput('');
  }, [visible]);

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onCancel?.()}>
      <DialogContent className='max-w-[720px] border-white/10 bg-black text-white'>
        <DialogHeader>
          <DialogTitle>{t('Codex 授权')}</DialogTitle>
        </DialogHeader>
        <div className='flex flex-col gap-3'>
          <div className='rounded-xl border border-blue-500/20 bg-blue-500/10 p-3 text-sm text-blue-100'>
            <div className='flex gap-2'>
              <AlertTriangle className='mt-0.5 h-4 w-4 shrink-0' />
              <span>
                {t(
                  '1) 点击「打开授权页面」完成登录；2) 浏览器会跳转到 localhost（页面打不开也没关系）；3) 复制地址栏完整 URL 粘贴到下方；4) 点击「生成并填入」。',
                )}
              </span>
            </div>
          </div>

          <div className='flex flex-wrap gap-2'>
            <Button type='button' onClick={startOAuth} disabled={loading}>
              {t('打开授权页面')}
            </Button>
            <Button
              type='button'
              variant='secondary'
              disabled={!authorizeUrl || loading}
              onClick={() => copy(authorizeUrl)}
            >
              {t('复制授权链接')}
            </Button>
          </div>

          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={t('请粘贴完整回调 URL（包含 code 与 state）')}
            className='border-white/10 bg-white/6 text-white'
          />

          <div className='text-xs text-white/60'>
            {t(
              '说明：生成结果是可直接粘贴到渠道密钥里的 JSON（包含 access_token / refresh_token / account_id）。',
            )}
          </div>
        </div>

        <DialogFooter className='border-white/10 bg-transparent'>
          <Button
            type='button'
            variant='secondary'
            onClick={onCancel}
            disabled={loading}
          >
            {t('取消')}
          </Button>
          <Button type='button' onClick={completeOAuth} disabled={loading}>
            {t('生成并填入')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default CodexOAuthModal;
