import React, { useState, useEffect, useMemo } from 'react';
import JSONEditor from '../../../common/ui/JSONEditor';
import {
  Save,
  X,
  FileText,
  AlertTriangle,
  Link as LinkIcon,
} from 'lucide-react';
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
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

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

const nameRuleOptions = [
  { label: '精确名称匹配', value: 0 },
  { label: '前缀名称匹配', value: 1 },
  { label: '包含名称匹配', value: 2 },
  { label: '后缀名称匹配', value: 3 },
];

const EditModelModal = (props) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const isMobile = useIsMobile();
  const isEdit = props.editingModel && props.editingModel.id !== undefined;
  const [tagInput, setTagInput] = useState('');
  const [formData, setFormData] = useState({
    model_name: props.editingModel?.model_name || '',
    description: '',
    icon: '',
    tags: [],
    vendor_id: undefined,
    vendor: '',
    vendor_icon: '',
    endpoints: '',
    name_rule: props.editingModel?.model_name ? 0 : undefined,
    status: true,
    sync_official: true,
  });

  // 供应商列表
  const [vendors, setVendors] = useState([]);

  // 预填组（标签、端点）
  const [tagGroups, setTagGroups] = useState([]);
  const [endpointGroups, setEndpointGroups] = useState([]);

  // 获取供应商列表
  const fetchVendors = async () => {
    try {
      const res = await API.get('/api/vendors/?page_size=1000'); // 获取全部供应商
      if (res.data.success) {
        const items = res.data.data.items || res.data.data || [];
        setVendors(Array.isArray(items) ? items : []);
      }
    } catch (error) {
      // ignore
    }
  };

  // 获取预填组（标签、端点）
  const fetchPrefillGroups = async () => {
    try {
      const [tagRes, endpointRes] = await Promise.all([
        API.get('/api/prefill_group?type=tag'),
        API.get('/api/prefill_group?type=endpoint'),
      ]);
      if (tagRes?.data?.success) {
        setTagGroups(tagRes.data.data || []);
      }
      if (endpointRes?.data?.success) {
        setEndpointGroups(endpointRes.data.data || []);
      }
    } catch (error) {
      // ignore
    }
  };

  useEffect(() => {
    if (props.visiable) {
      fetchVendors();
      fetchPrefillGroups();
    }
  }, [props.visiable]);

  const getInitValues = useMemo(
    () => () => ({
      model_name: props.editingModel?.model_name || '',
      description: '',
      icon: '',
      tags: [],
      vendor_id: undefined,
      vendor: '',
      vendor_icon: '',
      endpoints: '',
      name_rule: props.editingModel?.model_name ? 0 : undefined,
      status: true,
      sync_official: true,
    }),
    [props.editingModel?.model_name],
  );

  const setField = (field, value) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const handleCancel = () => {
    props.handleClose();
  };

  const loadModel = async () => {
    if (!isEdit || !props.editingModel.id) return;

    setLoading(true);
    try {
      const res = await API.get(`/api/models/${props.editingModel.id}`);
      const { success, message, data } = res.data;
      if (success) {
        // 处理tags
        if (data.tags) {
          data.tags = data.tags.split(',').filter(Boolean);
        } else {
          data.tags = [];
        }
        // endpoints 保持原始 JSON 字符串，若为空设为空串
        if (!data.endpoints) {
          data.endpoints = '';
        }
        data.status = data.status === 1;
        data.sync_official = (data.sync_official ?? 1) === 1;
        setFormData({ ...getInitValues(), ...data });
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('加载模型信息失败'));
    }
    setLoading(false);
  };

  useEffect(() => {
    if (props.visiable) {
      if (isEdit) {
        loadModel();
      } else {
        setFormData({
          ...getInitValues(),
          model_name: props.editingModel?.model_name || '',
        });
        setTagInput('');
      }
    } else {
      setFormData(getInitValues());
      setTagInput('');
    }
  }, [props.visiable, props.editingModel?.id, props.editingModel?.model_name]);

  const addTags = (rawValue) => {
    const values = rawValue
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean);
    if (values.length === 0) return;
    setFormData((prev) => ({
      ...prev,
      tags: [
        ...new Set([...(Array.isArray(prev.tags) ? prev.tags : []), ...values]),
      ],
    }));
    setTagInput('');
  };

  const submit = async () => {
    if (!formData.model_name?.trim()) {
      showError(t('请输入模型名称'));
      return;
    }
    if (formData.name_rule === undefined || formData.name_rule === null) {
      showError(t('请选择名称匹配类型'));
      return;
    }

    setLoading(true);
    try {
      const submitData = {
        ...formData,
        tags: Array.isArray(formData.tags)
          ? formData.tags.join(',')
          : formData.tags,
        endpoints: formData.endpoints || '',
        status: formData.status ? 1 : 0,
        sync_official: formData.sync_official ? 1 : 0,
      };

      if (isEdit) {
        submitData.id = props.editingModel.id;
        const res = await API.put('/api/models/', submitData);
        const { success, message } = res.data;
        if (success) {
          showSuccess(t('模型更新成功！'));
          props.refresh();
          props.handleClose();
        } else {
          showError(t(message));
        }
      } else {
        const res = await API.post('/api/models/', submitData);
        const { success, message } = res.data;
        if (success) {
          showSuccess(t('模型创建成功！'));
          props.refresh();
          props.handleClose();
        } else {
          showError(t(message));
        }
      }
    } catch (error) {
      showError(error.response?.data?.message || t('操作失败'));
    }
    setLoading(false);
    setFormData(getInitValues());
  };

  return (
    <Dialog
      open={props.visiable}
      onOpenChange={(open) => !open && handleCancel()}
    >
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
              {isEdit ? t('更新模型信息') : t('创建新的模型')}
            </DialogTitle>
          </div>
        </DialogHeader>

        <Card className='border-white/10 bg-white/6 text-white'>
          <CardContent className='space-y-4 p-5'>
            <div className='flex items-center gap-3'>
              <div className='flex h-9 w-9 items-center justify-center rounded-full bg-white/10'>
                <FileText className='h-4 w-4' />
              </div>
              <div>
                <div className='text-lg font-medium'>{t('基本信息')}</div>
                <div className='text-xs text-white/45'>
                  {t('设置模型的基本信息')}
                </div>
              </div>
            </div>

            <div className='grid gap-4'>
              <div className='grid gap-2'>
                <Label htmlFor='model-name'>{t('模型名称')}</Label>
                <Input
                  id='model-name'
                  value={formData.model_name}
                  onChange={(e) => setField('model_name', e.target.value)}
                  placeholder={t('请输入模型名称，如：gpt-4')}
                  className='border-white/10 bg-white/6 text-white'
                />
              </div>

              <div className='grid gap-2'>
                <Label>{t('名称匹配类型')}</Label>
                <Select
                  value={
                    formData.name_rule === undefined
                      ? undefined
                      : String(formData.name_rule)
                  }
                  onValueChange={(value) =>
                    setField('name_rule', Number(value))
                  }
                >
                  <SelectTrigger className='w-full border-white/10 bg-white/6 text-white'>
                    <SelectValue placeholder={t('请选择名称匹配类型')} />
                  </SelectTrigger>
                  <SelectContent className='border-white/10 bg-black text-white'>
                    {nameRuleOptions.map((option) => (
                      <SelectItem
                        key={option.value}
                        value={String(option.value)}
                      >
                        {t(option.label)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className='text-xs text-white/45'>
                  {t(
                    '根据模型名称和匹配规则查找模型元数据，优先级：精确 > 前缀 > 后缀 > 包含',
                  )}
                </p>
              </div>

              <div className='grid gap-2'>
                <Label htmlFor='model-icon'>{t('模型图标')}</Label>
                <Input
                  id='model-icon'
                  value={formData.icon}
                  onChange={(e) => setField('icon', e.target.value)}
                  placeholder={t('请输入图标名称')}
                  className='border-white/10 bg-white/6 text-white'
                />
                <p className='text-xs text-white/60'>
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
                </p>
              </div>

              <div className='grid gap-2'>
                <Label htmlFor='model-description'>{t('描述')}</Label>
                <Textarea
                  id='model-description'
                  value={formData.description}
                  onChange={(e) => setField('description', e.target.value)}
                  placeholder={t('请输入模型描述')}
                  rows={3}
                  className='border-white/10 bg-white/6 text-white'
                />
              </div>

              <div className='grid gap-2'>
                <Label htmlFor='model-tags'>{t('标签')}</Label>
                <div className='flex gap-2'>
                  <Input
                    id='model-tags'
                    value={tagInput}
                    onChange={(e) => setTagInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        addTags(tagInput);
                      }
                    }}
                    onBlur={() => addTags(tagInput)}
                    placeholder={t('输入标签或使用","分隔多个标签')}
                    className='border-white/10 bg-white/6 text-white'
                  />
                  <Button
                    type='button'
                    variant='secondary'
                    onClick={() => addTags(tagInput)}
                  >
                    {t('添加')}
                  </Button>
                </div>
                {tagGroups.length > 0 && (
                  <div className='flex flex-wrap gap-2'>
                    {tagGroups.map((group) => (
                      <Button
                        key={group.id}
                        type='button'
                        variant='secondary'
                        size='sm'
                        onClick={() =>
                          setFormData((prev) => ({
                            ...prev,
                            tags: [
                              ...new Set([
                                ...(Array.isArray(prev.tags) ? prev.tags : []),
                                ...(group.items || []),
                              ]),
                            ],
                          }))
                        }
                      >
                        {group.name}
                      </Button>
                    ))}
                  </div>
                )}
                <div className='flex flex-wrap gap-2'>
                  {(Array.isArray(formData.tags) ? formData.tags : []).map(
                    (tag) => (
                      <Badge
                        key={tag}
                        variant='secondary'
                        className='gap-2 border-white/10 bg-white/10 text-white'
                      >
                        {tag}
                        <button
                          type='button'
                          onClick={() =>
                            setFormData((prev) => ({
                              ...prev,
                              tags: (Array.isArray(prev.tags)
                                ? prev.tags
                                : []
                              ).filter((item) => item !== tag),
                            }))
                          }
                          className='text-white/60 hover:text-white'
                        >
                          ×
                        </button>
                      </Badge>
                    ),
                  )}
                </div>
              </div>

              <div className='grid gap-2'>
                <Label>{t('供应商')}</Label>
                <Select
                  value={
                    formData.vendor_id === undefined
                      ? undefined
                      : String(formData.vendor_id)
                  }
                  onValueChange={(value) => {
                    const numericValue = Number(value);
                    const vendorInfo = vendors.find(
                      (v) => v.id === numericValue,
                    );
                    setFormData((prev) => ({
                      ...prev,
                      vendor_id: numericValue,
                      vendor: vendorInfo?.name || '',
                    }));
                  }}
                >
                  <SelectTrigger className='w-full border-white/10 bg-white/6 text-white'>
                    <SelectValue placeholder={t('选择模型供应商')} />
                  </SelectTrigger>
                  <SelectContent className='border-white/10 bg-black text-white'>
                    {vendors.map((vendor) => (
                      <SelectItem key={vendor.id} value={String(vendor.id)}>
                        {vendor.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className='rounded-xl border border-amber-500/20 bg-amber-500/10 p-3 text-sm'>
                <div className='flex gap-2 text-amber-200'>
                  <AlertTriangle className='mt-0.5 h-4 w-4 shrink-0' />
                  <span>
                    {t(
                      '提示：此处配置仅用于控制「模型广场」对用户的展示效果，不会影响模型的实际调用与路由。若需配置真实调用行为，请前往「渠道管理」进行设置。',
                    )}
                  </span>
                </div>
              </div>

              <JSONEditor
                field='endpoints'
                label={t('在模型广场向用户展示的端点')}
                placeholder={
                  '{\n  "openai": {"path": "/v1/chat/completions", "method": "POST"}\n}'
                }
                value={formData.endpoints}
                onChange={(val) => setField('endpoints', val)}
                editorType='object'
                template={ENDPOINT_TEMPLATE}
                templateLabel={t('填入模板')}
                extraText={t('留空则使用默认端点；支持 {path, method}')}
                extraFooter={
                  endpointGroups.length > 0 && (
                    <div className='flex flex-wrap gap-2'>
                      {endpointGroups.map((group) => (
                        <Button
                          key={group.id}
                          type='button'
                          variant='secondary'
                          size='sm'
                          onClick={() => {
                            try {
                              const current = formData.endpoints || '';
                              let base = {};
                              if (current && current.trim()) {
                                base = JSON.parse(current);
                              }
                              const groupObj =
                                typeof group.items === 'string'
                                  ? JSON.parse(group.items || '{}')
                                  : group.items || {};
                              setField(
                                'endpoints',
                                JSON.stringify(
                                  { ...base, ...groupObj },
                                  null,
                                  2,
                                ),
                              );
                            } catch (e) {
                              try {
                                const groupObj =
                                  typeof group.items === 'string'
                                    ? JSON.parse(group.items || '{}')
                                    : group.items || {};
                                setField(
                                  'endpoints',
                                  JSON.stringify(groupObj, null, 2),
                                );
                              } catch {}
                            }
                          }}
                        >
                          {group.name}
                        </Button>
                      ))}
                    </div>
                  )
                }
              />

              <div className='flex items-center justify-between rounded-xl border border-white/10 bg-white/5 px-3 py-2'>
                <div className='grid gap-1'>
                  <Label htmlFor='sync-official'>{t('参与官方同步')}</Label>
                  <p className='text-xs text-white/45'>
                    {t('关闭后，此模型将不会被“同步官方”自动覆盖或创建')}
                  </p>
                </div>
                <Switch
                  id='sync-official'
                  checked={formData.sync_official}
                  onCheckedChange={(checked) =>
                    setField('sync_official', checked)
                  }
                />
              </div>

              <div className='flex items-center justify-between rounded-xl border border-white/10 bg-white/5 px-3 py-2'>
                <Label htmlFor='model-status'>{t('状态')}</Label>
                <Switch
                  id='model-status'
                  checked={formData.status}
                  onCheckedChange={(checked) => setField('status', checked)}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={handleCancel}>
            <X className='mr-1 h-4 w-4' />
            {t('取消')}
          </Button>
          <Button type='button' onClick={submit} disabled={loading}>
            <Save className='mr-1 h-4 w-4' />
            {t('提交')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default EditModelModal;
