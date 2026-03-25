import React, { useState, useEffect, useMemo } from 'react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useTranslation } from 'react-i18next';
import { showError } from '../../../../helpers';

const APP_CONFIGS = {
  claude: {
    label: 'Claude',
    defaultName: 'My Claude',
    modelFields: [
      { key: 'model', label: '主模型' },
      { key: 'haikuModel', label: 'Haiku 模型' },
      { key: 'sonnetModel', label: 'Sonnet 模型' },
      { key: 'opusModel', label: 'Opus 模型' },
    ],
  },
  codex: {
    label: 'Codex',
    defaultName: 'My Codex',
    modelFields: [{ key: 'model', label: '主模型' }],
  },
  gemini: {
    label: 'Gemini',
    defaultName: 'My Gemini',
    modelFields: [{ key: 'model', label: '主模型' }],
  },
};

function getServerAddress() {
  try {
    const raw = localStorage.getItem('status');
    if (raw) {
      const status = JSON.parse(raw);
      if (status.server_address) return status.server_address;
    }
  } catch (_) {}
  return window.location.origin;
}

function buildCCSwitchURL(app, name, models, apiKey) {
  const serverAddress = getServerAddress();
  const endpoint = app === 'codex' ? serverAddress + '/v1' : serverAddress;
  const params = new URLSearchParams();
  params.set('resource', 'provider');
  params.set('app', app);
  params.set('name', name);
  params.set('endpoint', endpoint);
  params.set('apiKey', apiKey);
  for (const [k, v] of Object.entries(models)) {
    if (v) params.set(k, v);
  }
  params.set('homepage', serverAddress);
  params.set('enabled', 'true');
  return `ccswitch://v1/import?${params.toString()}`;
}

export default function CCSwitchModal({
  visible,
  onClose,
  tokenKey,
  modelOptions,
}) {
  const { t } = useTranslation();
  const [app, setApp] = useState('claude');
  const [name, setName] = useState(APP_CONFIGS.claude.defaultName);
  const [models, setModels] = useState({});

  const currentConfig = APP_CONFIGS[app];

  useEffect(() => {
    if (visible) {
      setModels({});
      setApp('claude');
      setName(APP_CONFIGS.claude.defaultName);
    }
  }, [visible]);

  const handleAppChange = (val) => {
    setApp(val);
    setName(APP_CONFIGS[val].defaultName);
    setModels({});
  };

  const handleModelChange = (field, value) => {
    setModels((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = () => {
    if (!models.model) {
      showError(t('请选择主模型'));
      return;
    }
    const url = buildCCSwitchURL(app, name, models, 'sk-' + tokenKey);
    window.open(url, '_blank');
    onClose();
  };

  const fieldLabelStyle = useMemo(
    () => ({
      marginBottom: 4,
      fontSize: 13,
      color: 'var(--semi-color-text-1)',
    }),
    [],
  );

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className='max-w-[480px] border-white/10 bg-black text-white'>
        <DialogHeader>
          <DialogTitle>{t('填入 CC Switch')}</DialogTitle>
        </DialogHeader>
        <div className='flex flex-col gap-4'>
          <div>
            <div style={fieldLabelStyle}>{t('应用')}</div>
            <div className='flex flex-wrap gap-2'>
              {Object.entries(APP_CONFIGS).map(([key, cfg]) => (
                <Button
                  key={key}
                  type='button'
                  variant={app === key ? 'default' : 'secondary'}
                  className='rounded-xl'
                  onClick={() => handleAppChange(key)}
                >
                  {cfg.label}
                </Button>
              ))}
            </div>
          </div>

          <div>
            <div style={fieldLabelStyle}>{t('名称')}</div>
            <Input
              value={name}
              onChange={setName}
              placeholder={currentConfig.defaultName}
            />
          </div>

          {currentConfig.modelFields.map((field) => (
            <div key={field.key}>
              <div style={fieldLabelStyle}>
                {t(field.label)}
                {field.key === 'model' && (
                  <span className='text-red-400'> *</span>
                )}
              </div>
              <Select
                value={models[field.key] || ''}
                onValueChange={(val) => handleModelChange(field.key, val)}
              >
                <SelectTrigger className='w-full border-white/10 bg-white/6 text-white'>
                  <SelectValue placeholder={t('请选择模型')} />
                </SelectTrigger>
                <SelectContent className='border-white/10 bg-[#0b1220] text-white'>
                  {(modelOptions || []).map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ))}
        </div>
        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={onClose}>
            {t('取消')}
          </Button>
          <Button type='button' onClick={handleSubmit}>
            {t('打开 CC Switch')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
