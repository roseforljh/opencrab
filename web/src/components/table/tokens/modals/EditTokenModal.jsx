import React, { useEffect, useState, useContext } from 'react';
import {
  API,
  showError,
  showSuccess,
  timestamp2string,
  renderGroupOption,
  renderQuotaWithPrompt,
  getModelCategories,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../../../context/Status';
import { KeyRound } from 'lucide-react';

const EditTokenModal = (props) => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [loading, setLoading] = useState(false);
  const isMobile = useIsMobile();
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const getInitValues = () => ({
    name: '',
    remain_quota: 0,
    expired_time: -1,
    unlimited_quota: true,
    model_limits_enabled: false,
    model_limits: [],
    allow_ips: '',
    group: '',
    cross_group_retry: false,
    tokenCount: 1,
  });
  const [values, setValues] = useState(getInitValues());
  const isEdit = props.editingToken.id !== undefined;

  const handleCancel = () => {
    props.handleClose();
  };

  const setExpiredTime = (month, day, hour, minute) => {
    const now = new Date();
    let timestamp = now.getTime() / 1000;
    let seconds = month * 30 * 24 * 60 * 60;
    seconds += day * 24 * 60 * 60;
    seconds += hour * 60 * 60;
    seconds += minute * 60;
    if (seconds !== 0) {
      timestamp += seconds;
      setValues((prev) => ({
        ...prev,
        expired_time: timestamp2string(timestamp),
      }));
    } else {
      setValues((prev) => ({
        ...prev,
        expired_time: -1,
      }));
    }
  };

  const loadModels = async () => {
    const res = await API.get('/api/user/models');
    const { success, message, data } = res.data;
    if (success) {
      const categories = getModelCategories(t);
      const modelsData = Array.isArray(data) ? data : [];
      const localModelOptions = modelsData.map((model) => {
        let icon = null;
        for (const [key, category] of Object.entries(categories)) {
          if (key !== 'all' && category.filter({ model_name: model })) {
            icon = category.icon;
            break;
          }
        }
        return {
          label: (
            <span className='flex items-center gap-1'>
              {icon}
              {model}
            </span>
          ),
          value: model,
        };
      });
      setModels(localModelOptions);
    } else {
      showError(t(message));
    }
  };

  const loadGroups = async () => {
    const res = await API.get('/api/user/self/groups');
    const { success, message, data } = res.data;
    if (success) {
      const groupsData = data && typeof data === 'object' ? data : {};
      let localGroupOptions = Object.entries(groupsData).map(([group, info]) => ({
        label: info.desc,
        value: group,
        ratio: info.ratio,
      }));
      if (statusState?.status?.default_use_auto_group) {
        if (localGroupOptions.some((group) => group.value === 'auto')) {
          localGroupOptions.sort((a, b) => (a.value === 'auto' ? -1 : 1));
        }
      }
      setGroups(localGroupOptions);
    } else {
      showError(t(message));
    }
  };

  const loadToken = async () => {
    setLoading(true);
    const res = await API.get(`/api/token/${props.editingToken.id}`);
    const { success, message, data } = res.data;
    if (success) {
      if (data.expired_time !== -1) {
        data.expired_time = timestamp2string(data.expired_time);
      }
      data.model_limits =
        data.model_limits !== '' ? data.model_limits.split(',') : [];
      setValues({ ...getInitValues(), ...data });
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (!props.visiable) {
      return;
    }
    loadModels();
    loadGroups();
  }, [props.visiable, props.editingToken.id]);

  useEffect(() => {
    if (props.visiable) {
      if (isEdit) {
        loadToken();
      } else {
        setValues(getInitValues());
      }
    } else {
      setValues(getInitValues());
    }
  }, [props.visiable, props.editingToken.id]);

  const generateRandomSuffix = () => {
    const characters =
      'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < 6; i++) {
      result += characters.charAt(
        Math.floor(Math.random() * characters.length),
      );
    }
    return result;
  };

  const submit = async (values) => {
    setLoading(true);
    if (isEdit) {
      const { tokenCount: _tc, ...localInputs } = values;
      localInputs.remain_quota = parseInt(localInputs.remain_quota);
      if (localInputs.expired_time !== -1) {
        const time = Date.parse(localInputs.expired_time);
        if (isNaN(time)) {
          showError(t('过期时间格式错误！'));
          setLoading(false);
          return;
        }
        localInputs.expired_time = Math.ceil(time / 1000);
      }
      localInputs.model_limits = localInputs.model_limits_enabled
        ? localInputs.model_limits.join(',')
        : '';
      const res = await API.put('/api/token/', localInputs);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('令牌更新成功！'));
        props.refresh();
        props.handleClose();
      } else {
        showError(message);
      }
      setLoading(false);
      return;
    }

    const localInputs = { ...values };
    localInputs.remain_quota = parseInt(localInputs.remain_quota);
    if (localInputs.expired_time !== -1) {
      const time = Date.parse(localInputs.expired_time);
      if (isNaN(time)) {
        showError(t('过期时间格式错误！'));
        setLoading(false);
        return;
      }
      localInputs.expired_time = Math.ceil(time / 1000);
    }
    localInputs.model_limits = localInputs.model_limits_enabled
      ? localInputs.model_limits.join(',')
      : '';

    if (localInputs.tokenCount > 1) {
      const baseName = localInputs.name || 'token';
      const count = Math.min(parseInt(localInputs.tokenCount), 100);
      for (let i = 0; i < count; i++) {
        await API.post('/api/token/', {
          ...localInputs,
          name: `${baseName}-${generateRandomSuffix()}`,
        });
      }
      showSuccess(t('批量令牌创建成功！'));
    } else {
      const res = await API.post('/api/token/', localInputs);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('令牌创建成功！'));
      } else {
        showError(message);
        setLoading(false);
        return;
      }
    }
    props.refresh();
    props.handleClose();
    setLoading(false);
  };

  return (
    <Dialog
      open={props.visiable}
      onOpenChange={(open) => !open && handleCancel()}
    >
      <DialogContent
        className={`border-white/10 bg-[#08101d] text-white ${isMobile ? 'max-w-[95vw]' : 'max-w-[720px]'}`}
      >
        <DialogHeader>
          <DialogTitle>{isEdit ? t('编辑令牌') : t('新建令牌')}</DialogTitle>
        </DialogHeader>
        <Card className='border border-white/10 bg-white/6 shadow-none'>
          <CardContent className='pt-6'>
            <div className='mb-4 flex items-center gap-3'>
              <div className='flex h-10 w-10 items-center justify-center rounded-full bg-white/90 text-black'>
                <KeyRound className='h-5 w-5' />
              </div>
              <div>
                <div className='text-lg font-semibold'>
                  {isEdit ? t('编辑访问令牌') : t('创建新的访问令牌')}
                </div>
                <div className='text-sm text-white/60'>
                  {t('设置令牌名称、额度、过期时间和模型范围')}
                </div>
              </div>
            </div>

            <div className='grid gap-4 md:grid-cols-2'>
              <div className='space-y-2'>
                <Label>{t('名称')}</Label>
                <Input
                  value={values.name}
                  onChange={(e) =>
                    setValues((prev) => ({ ...prev, name: e.target.value }))
                  }
                  placeholder={t('输入令牌名称')}
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('分组')}</Label>
                <Select
                  value={values.group || ''}
                  onValueChange={(value) =>
                    setValues((prev) => ({ ...prev, group: value }))
                  }
                >
                  <SelectTrigger className='border-white/10 bg-white/6 text-white'>
                    <SelectValue placeholder={t('选择分组')} />
                  </SelectTrigger>
                  <SelectContent className='border-white/10 bg-[#0b1220] text-white'>
                    {groups.map((group) => (
                      <SelectItem key={group.value} value={group.value}>
                        {group.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className='space-y-2'>
                <Label>{t('额度')}</Label>
                <Input
                  type='number'
                  value={values.remain_quota}
                  onChange={(e) =>
                    setValues((prev) => ({
                      ...prev,
                      remain_quota: e.target.value,
                    }))
                  }
                />
              </div>
              <div className='space-y-2'>
                <Label>{t('过期时间')}</Label>
                <Input
                  value={values.expired_time}
                  onChange={(e) =>
                    setValues((prev) => ({
                      ...prev,
                      expired_time: e.target.value,
                    }))
                  }
                  placeholder={t('不填则永不过期')}
                />
              </div>
            </div>

            <div className='mt-4 flex flex-wrap gap-2'>
              <Button
                type='button'
                variant='secondary'
                onClick={() => setExpiredTime(1, 0, 0, 0)}
              >
                {t('1个月')}
              </Button>
              <Button
                type='button'
                variant='secondary'
                onClick={() => setExpiredTime(0, 7, 0, 0)}
              >
                {t('7天')}
              </Button>
              <Button
                type='button'
                variant='secondary'
                onClick={() => setExpiredTime(0, 1, 0, 0)}
              >
                {t('1天')}
              </Button>
              <Button
                type='button'
                variant='secondary'
                onClick={() => setExpiredTime(0, 0, 1, 0)}
              >
                {t('1小时')}
              </Button>
            </div>

            {!isEdit && (
              <div className='mt-4 space-y-2'>
                <Label>{t('创建数量')}</Label>
                <Input
                  type='number'
                  min={1}
                  max={100}
                  value={values.tokenCount}
                  onChange={(e) =>
                    setValues((prev) => ({
                      ...prev,
                      tokenCount: e.target.value,
                    }))
                  }
                />
              </div>
            )}

            <div className='mt-4 space-y-4'>
              <div className='flex items-center justify-between rounded-2xl border border-white/10 bg-white/6 px-4 py-3'>
                <Label>{t('无限额度')}</Label>
                <Switch
                  checked={values.unlimited_quota}
                  onCheckedChange={(checked) =>
                    setValues((prev) => ({
                      ...prev,
                      unlimited_quota: checked,
                    }))
                  }
                />
              </div>
              <div className='flex items-center justify-between rounded-2xl border border-white/10 bg-white/6 px-4 py-3'>
                <Label>{t('启用模型限制')}</Label>
                <Switch
                  checked={values.model_limits_enabled}
                  onCheckedChange={(checked) =>
                    setValues((prev) => ({
                      ...prev,
                      model_limits_enabled: checked,
                    }))
                  }
                />
              </div>
            </div>

            <div className='mt-4 space-y-2'>
              <Label>{t('允许模型')}</Label>
              <Textarea
                value={(values.model_limits || []).join(',')}
                onChange={(e) =>
                  setValues((prev) => ({
                    ...prev,
                    model_limits: e.target.value
                      .split(',')
                      .map((item) => item.trim())
                      .filter(Boolean),
                  }))
                }
                placeholder={models
                  .map((m) => m.value)
                  .slice(0, 5)
                  .join(', ')}
              />
              <div className='flex flex-wrap gap-2'>
                {models.slice(0, 12).map((model) => (
                  <Badge
                    key={model.value}
                    variant='secondary'
                    className='cursor-pointer rounded-full'
                    onClick={() =>
                      setValues((prev) => ({
                        ...prev,
                        model_limits: Array.from(
                          new Set([...(prev.model_limits || []), model.value]),
                        ),
                      }))
                    }
                  >
                    {model.value}
                  </Badge>
                ))}
              </div>
            </div>

            <div className='mt-4 space-y-2'>
              <Label>{t('允许 IP')}</Label>
              <Textarea
                value={values.allow_ips}
                onChange={(e) =>
                  setValues((prev) => ({
                    ...prev,
                    allow_ips: e.target.value,
                  }))
                }
                placeholder={t('多个 IP 用逗号分隔')}
              />
            </div>

            <div className='mt-4 flex items-center justify-between rounded-2xl border border-white/10 bg-white/6 px-4 py-3'>
              <Label>{t('跨分组重试')}</Label>
              <Switch
                checked={values.cross_group_retry}
                onCheckedChange={(checked) =>
                  setValues((prev) => ({
                    ...prev,
                    cross_group_retry: checked,
                  }))
                }
              />
            </div>
          </CardContent>
        </Card>
        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={handleCancel}>
            {t('取消')}
          </Button>
          <Button
            type='button'
            onClick={() => submit(values)}
            disabled={loading}
          >
            {isEdit ? t('保存修改') : t('创建令牌')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default EditTokenModal;
