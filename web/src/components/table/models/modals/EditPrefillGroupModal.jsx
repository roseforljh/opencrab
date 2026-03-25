import React, { useEffect, useMemo, useState } from 'react';
import JSONEditor from '../../../common/ui/JSONEditor';
import { Layers, Save, X } from 'lucide-react';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';

// Example endpoint template for quick fill
const ENDPOINT_TEMPLATE = {
  openai: { path: '/v1/chat/completions', method: 'POST' },
  'openai-response': { path: '/v1/responses', method: 'POST' },
  'openai-response-compact': { path: '/v1/responses/compact', method: 'POST' },
  anthropic: { path: '/v1/messages', method: 'POST' },
  gemini: { path: '/v1beta/models/{model}:generateContent', method: 'POST' },
  'jina-rerank': { path: '/v1/rerank', method: 'POST' },
  'image-generation': { path: '/v1/images/generations', method: 'POST' },
};

const EditPrefillGroupModal = ({
  visible,
  onClose,
  editingGroup,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const isEdit = editingGroup && editingGroup.id !== undefined;
  const getInitialValues = useMemo(
    () => () => ({
      name: editingGroup?.name || '',
      type: editingGroup?.type || 'tag',
      description: editingGroup?.description || '',
      items: (() => {
        try {
          if (editingGroup?.type === 'endpoint') {
            return typeof editingGroup?.items === 'string'
              ? editingGroup.items
              : JSON.stringify(editingGroup.items || {}, null, 2);
          }
          return Array.isArray(editingGroup?.items) ? editingGroup.items : [];
        } catch {
          return editingGroup?.type === 'endpoint' ? '' : [];
        }
      })(),
    }),
    [editingGroup],
  );
  const [formData, setFormData] = useState(getInitialValues());
  const [tagInput, setTagInput] = useState('');

  const typeOptions = [
    { label: t('模型组'), value: 'model' },
    { label: t('标签组'), value: 'tag' },
    { label: t('端点组'), value: 'endpoint' },
  ];

  useEffect(() => {
    if (visible) {
      setFormData(getInitialValues());
      setTagInput('');
    }
  }, [visible, getInitialValues]);

  const setField = (field, value) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const addTagItems = (rawValue) => {
    const values = rawValue
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean);
    if (values.length === 0) return;
    setFormData((prev) => ({
      ...prev,
      items: [
        ...new Set([
          ...(Array.isArray(prev.items) ? prev.items : []),
          ...values,
        ]),
      ],
    }));
    setTagInput('');
  };

  const removeTagItem = (value) => {
    setFormData((prev) => ({
      ...prev,
      items: (Array.isArray(prev.items) ? prev.items : []).filter(
        (item) => item !== value,
      ),
    }));
  };

  const handleSubmit = async () => {
    if (!formData.name?.trim()) {
      showError(t('请输入组名'));
      return;
    }

    if (!formData.type) {
      showError(t('请选择组类型'));
      return;
    }

    setLoading(true);
    try {
      const submitData = {
        ...formData,
      };
      if (formData.type === 'endpoint') {
        submitData.items = formData.items || '';
      } else {
        submitData.items = Array.isArray(formData.items) ? formData.items : [];
      }

      if (editingGroup.id) {
        submitData.id = editingGroup.id;
        const res = await API.put('/api/prefill_group', submitData);
        if (res.data.success) {
          showSuccess(t('更新成功'));
          onSuccess();
        } else {
          showError(res.data.message || t('更新失败'));
        }
      } else {
        const res = await API.post('/api/prefill_group', submitData);
        if (res.data.success) {
          showSuccess(t('创建成功'));
          onSuccess();
        } else {
          showError(res.data.message || t('创建失败'));
        }
      }
    } catch (error) {
      showError(t('操作失败'));
    }
    setLoading(false);
  };

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[720px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          <div className='flex items-center gap-2'>
            <Badge
              variant='secondary'
              className='border-white/10 bg-white/10 text-white'
            >
              {isEdit ? t('更新') : t('新建')}
            </Badge>
            <DialogTitle>
              {isEdit ? t('更新预填组') : t('创建新的预填组')}
            </DialogTitle>
          </div>
        </DialogHeader>

        <Card className='border-white/10 bg-white/6 text-white'>
          <CardContent className='space-y-4 p-5'>
            <div className='flex items-center gap-3'>
              <div className='flex h-9 w-9 items-center justify-center rounded-full bg-white/10'>
                <Layers className='h-4 w-4' />
              </div>
              <div>
                <div className='text-lg font-medium'>{t('基本信息')}</div>
                <div className='text-xs text-white/45'>
                  {t('设置预填组的基本信息')}
                </div>
              </div>
            </div>

            <div className='grid gap-4'>
              <div className='grid gap-2'>
                <Label htmlFor='prefill-group-name'>{t('组名')}</Label>
                <Input
                  id='prefill-group-name'
                  value={formData.name}
                  onChange={(e) => setField('name', e.target.value)}
                  placeholder={t('请输入组名')}
                  className='border-white/10 bg-white/6 text-white'
                />
              </div>

              <div className='grid gap-2'>
                <Label>{t('类型')}</Label>
                <Select
                  value={formData.type}
                  onValueChange={(value) => setField('type', value)}
                >
                  <SelectTrigger className='w-full border-white/10 bg-white/6 text-white'>
                    <SelectValue placeholder={t('选择组类型')} />
                  </SelectTrigger>
                  <SelectContent className='border-white/10 bg-black text-white'>
                    {typeOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className='grid gap-2'>
                <Label htmlFor='prefill-group-description'>{t('描述')}</Label>
                <Textarea
                  id='prefill-group-description'
                  value={formData.description}
                  onChange={(e) => setField('description', e.target.value)}
                  placeholder={t('请输入组描述')}
                  rows={3}
                  className='border-white/10 bg-white/6 text-white'
                />
              </div>

              <div className='grid gap-2'>
                {formData.type === 'endpoint' ? (
                  <JSONEditor
                    field='items'
                    label={t('端点映射')}
                    value={formData.items}
                    onChange={(val) => setField('items', val)}
                    editorType='object'
                    placeholder={
                      '{\n  "openai": {"path": "/v1/chat/completions", "method": "POST"}\n}'
                    }
                    template={ENDPOINT_TEMPLATE}
                    templateLabel={t('填入模板')}
                    extraText={t('键为端点类型，值为路径和方法对象')}
                  />
                ) : (
                  <>
                    <Label htmlFor='prefill-group-items'>{t('项目')}</Label>
                    <div className='flex gap-2'>
                      <Input
                        id='prefill-group-items'
                        value={tagInput}
                        onChange={(e) => setTagInput(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            addTagItems(tagInput);
                          }
                        }}
                        onBlur={() => addTagItems(tagInput)}
                        placeholder={t('输入项目名称，按回车添加')}
                        className='border-white/10 bg-white/6 text-white'
                      />
                      <Button
                        type='button'
                        variant='secondary'
                        onClick={() => addTagItems(tagInput)}
                      >
                        {t('添加')}
                      </Button>
                    </div>
                    <div className='flex flex-wrap gap-2'>
                      {(Array.isArray(formData.items)
                        ? formData.items
                        : []
                      ).map((item) => (
                        <Badge
                          key={item}
                          variant='secondary'
                          className='gap-2 border-white/10 bg-white/10 text-white'
                        >
                          {item}
                          <button
                            type='button'
                            onClick={() => removeTagItem(item)}
                            className='text-white/60 hover:text-white'
                          >
                            ×
                          </button>
                        </Badge>
                      ))}
                    </div>
                  </>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={onClose}>
            <X className='mr-1 h-4 w-4' />
            {t('取消')}
          </Button>
          <Button type='button' onClick={handleSubmit} disabled={loading}>
            <Save className='mr-1 h-4 w-4' />
            {t('提交')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default EditPrefillGroupModal;
