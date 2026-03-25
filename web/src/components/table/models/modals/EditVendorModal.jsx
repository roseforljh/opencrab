
import React, { useEffect, useMemo, useState } from 'react';
import { API, showError, showSuccess } from '../../../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Link as LinkIcon } from 'lucide-react';

const EditVendorModal = ({ visible, handleClose, refresh, editingVendor }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    icon: '',
    status: true,
  });

  const isMobile = useIsMobile();
  const isEdit = editingVendor && editingVendor.id !== undefined;

  const getInitValues = useMemo(
    () => () => ({
      name: '',
      description: '',
      icon: '',
      status: true,
    }),
    [],
  );

  const setField = (field, value) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const resetForm = () => {
    setFormData(getInitValues());
  };

  const iconHelpText = (
    <span className='text-xs text-white/60'>
      {t(
        "图标使用@lobehub/icons库，如：OpenAI、Claude.Color，支持链式参数：OpenAI.Avatar.type={'platform'}、OpenRouter.Avatar.shape={'square'}，查询所有可用图标请 ",
      )}
      <a
        href='https://icons.lobehub.com/components/lobe-hub'
        target='_blank'
        rel='noreferrer'
        className='inline-flex items-center gap-1 text-white underline underline-offset-4'
      >
        {t('请点击我')}
        <LinkIcon className='h-3.5 w-3.5' />
      </a>
    </span>
  );

  const handleCancel = () => {
    handleClose();
    resetForm();
  };

  const loadVendor = async () => {
    if (!isEdit || !editingVendor.id) return;

    setLoading(true);
    try {
      const res = await API.get(`/api/vendors/${editingVendor.id}`);
      const { success, message, data } = res.data;
      if (success) {
        setFormData({
          ...getInitValues(),
          ...data,
          status: data.status === 1,
        });
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('加载供应商信息失败'));
    }
    setLoading(false);
  };

  useEffect(() => {
    if (visible) {
      if (isEdit) {
        loadVendor();
      } else {
        resetForm();
      }
    } else {
      resetForm();
    }
  }, [visible, editingVendor?.id]);

  const submit = async () => {
    if (!formData.name?.trim()) {
      showError(t('请输入供应商名称'));
      return;
    }

    setLoading(true);
    try {
      const submitData = {
        ...formData,
        status: formData.status ? 1 : 0,
      };

      if (isEdit) {
        submitData.id = editingVendor.id;
        const res = await API.put('/api/vendors/', submitData);
        const { success, message } = res.data;
        if (success) {
          showSuccess(t('供应商更新成功！'));
          refresh();
          handleClose();
          resetForm();
        } else {
          showError(t(message));
        }
      } else {
        const res = await API.post('/api/vendors/', submitData);
        const { success, message } = res.data;
        if (success) {
          showSuccess(t('供应商创建成功！'));
          refresh();
          handleClose();
          resetForm();
        } else {
          showError(t(message));
        }
      }
    } catch (error) {
      showError(error.response?.data?.message || t('操作失败'));
    }
    setLoading(false);
  };

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && handleCancel()}>
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[560px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          <DialogTitle>{isEdit ? t('编辑供应商') : t('新增供应商')}</DialogTitle>
        </DialogHeader>

        <form
          className='grid gap-4'
          onSubmit={(e) => {
            e.preventDefault();
            submit();
          }}
        >
          <div className='grid gap-2'>
            <Label htmlFor='vendor-name'>{t('供应商名称')}</Label>
            <Input
              id='vendor-name'
              value={formData.name}
              onChange={(e) => setField('name', e.target.value)}
              placeholder={t('请输入供应商名称，如：OpenAI')}
              className='border-white/10 bg-white/6 text-white'
            />
          </div>

          <div className='grid gap-2'>
            <Label htmlFor='vendor-description'>{t('描述')}</Label>
            <Textarea
              id='vendor-description'
              value={formData.description}
              onChange={(e) => setField('description', e.target.value)}
              placeholder={t('请输入供应商描述')}
              rows={3}
              className='border-white/10 bg-white/6 text-white'
            />
          </div>

          <div className='grid gap-2'>
            <Label htmlFor='vendor-icon'>{t('供应商图标')}</Label>
            <Input
              id='vendor-icon'
              value={formData.icon}
              onChange={(e) => setField('icon', e.target.value)}
              placeholder={t('请输入图标名称')}
              className='border-white/10 bg-white/6 text-white'
            />
            {iconHelpText}
          </div>

          <div className='flex items-center justify-between rounded-xl border border-white/10 bg-white/5 px-3 py-2'>
            <div className='grid gap-1'>
              <Label htmlFor='vendor-status'>{t('状态')}</Label>
            </div>
            <Switch
              id='vendor-status'
              checked={formData.status}
              onCheckedChange={(checked) => setField('status', checked)}
            />
          </div>
        </form>

        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={handleCancel}>
            {t('取消')}
          </Button>
          <Button type='button' onClick={submit} disabled={loading}>
            {isEdit ? t('保存') : t('创建')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default EditVendorModal;
