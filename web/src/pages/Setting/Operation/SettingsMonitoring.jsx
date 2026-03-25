
import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
  parseHttpStatusCodeRules,
} from '../../../helpers';
import HttpStatusCodeRulesInput from '../../../components/settings/HttpStatusCodeRulesInput';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';

export default function SettingsMonitoring(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    ChannelDisableThreshold: '',
    QuotaRemindThreshold: '',
    AutomaticDisableChannelEnabled: false,
    AutomaticEnableChannelEnabled: false,
    AutomaticDisableKeywords: '',
    AutomaticDisableStatusCodes: '401',
    AutomaticRetryStatusCodes:
      '100-199,300-399,401-407,409-499,500-503,505-523,525-599',
    'monitor_setting.auto_test_channel_enabled': false,
    'monitor_setting.auto_test_channel_minutes': 10,
  });
  const [inputsRow, setInputsRow] = useState(inputs);
  const parsedAutoDisableStatusCodes = parseHttpStatusCodeRules(
    inputs.AutomaticDisableStatusCodes || '',
  );
  const parsedAutoRetryStatusCodes = parseHttpStatusCodeRules(
    inputs.AutomaticRetryStatusCodes || '',
  );

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    if (!parsedAutoDisableStatusCodes.ok) {
      const details =
        parsedAutoDisableStatusCodes.invalidTokens &&
        parsedAutoDisableStatusCodes.invalidTokens.length > 0
          ? `: ${parsedAutoDisableStatusCodes.invalidTokens.join(', ')}`
          : '';
      return showError(`${t('自动禁用状态码格式不正确')}${details}`);
    }
    if (!parsedAutoRetryStatusCodes.ok) {
      const details =
        parsedAutoRetryStatusCodes.invalidTokens &&
        parsedAutoRetryStatusCodes.invalidTokens.length > 0
          ? `: ${parsedAutoRetryStatusCodes.invalidTokens.join(', ')}`
          : '';
      return showError(`${t('自动重试状态码格式不正确')}${details}`);
    }
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        const normalizedMap = {
          AutomaticDisableStatusCodes: parsedAutoDisableStatusCodes.normalized,
          AutomaticRetryStatusCodes: parsedAutoRetryStatusCodes.normalized,
        };
        value = normalizedMap[item.key] ?? inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
  }, [props.options]);

  return (
    <div className='space-y-6'>
      <div className='flex flex-col space-y-4'>
        <h3 className='text-lg font-medium leading-none text-white'>
          {t('监控设置')}
        </h3>

        <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6'>
          <div className='space-y-3 pt-1'>
            <div className='flex flex-col space-y-2'>
              <Label
                className='text-white/80'
                htmlFor='auto_test_channel_enabled'
              >
                {t('定时测试所有通道')}
              </Label>
              <div className='flex items-center h-9'>
                <Switch
                  id='auto_test_channel_enabled'
                  checked={inputs['monitor_setting.auto_test_channel_enabled']}
                  onCheckedChange={(value) =>
                    setInputs({
                      ...inputs,
                      'monitor_setting.auto_test_channel_enabled': value,
                    })
                  }
                  className='data-[state=checked]:bg-white data-[state=unchecked]:bg-white/20'
                />
              </div>
            </div>
          </div>

          <div className='space-y-2'>
            <Label
              className='text-white/80'
              htmlFor='auto_test_channel_minutes'
            >
              {t('自动测试所有通道间隔时间(分钟)')}
            </Label>
            <Input
              id='auto_test_channel_minutes'
              type='number'
              min={1}
              value={inputs['monitor_setting.auto_test_channel_minutes']}
              placeholder={t('每隔多少分钟测试一次所有通道')}
              onChange={(e) =>
                setInputs({
                  ...inputs,
                  'monitor_setting.auto_test_channel_minutes':
                    parseInt(e.target.value) || 10,
                })
              }
              className='bg-black/20 border-white/10 text-white placeholder:text-white/30 focus-visible:ring-white/20'
            />
            <div className='text-xs text-white/50'>
              {t('每隔多少分钟测试一次所有通道')}
            </div>
          </div>
        </div>

        <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6 pt-4'>
          <div className='space-y-2'>
            <Label className='text-white/80' htmlFor='ChannelDisableThreshold'>
              {t('测试所有渠道的最长响应时间(秒)')}
            </Label>
            <Input
              id='ChannelDisableThreshold'
              type='number'
              min={0}
              value={inputs['ChannelDisableThreshold']}
              onChange={(e) =>
                setInputs({
                  ...inputs,
                  ChannelDisableThreshold: e.target.value,
                })
              }
              className='bg-black/20 border-white/10 text-white placeholder:text-white/30 focus-visible:ring-white/20'
            />
            <div className='text-xs text-white/50'>
              {t('当运行通道全部测试时，超过此时间将自动禁用通道')}
            </div>
          </div>

          <div className='space-y-2'>
            <Label className='text-white/80' htmlFor='QuotaRemindThreshold'>
              {t('额度提醒阈值(Token)')}
            </Label>
            <Input
              id='QuotaRemindThreshold'
              type='number'
              min={0}
              value={inputs['QuotaRemindThreshold']}
              onChange={(e) =>
                setInputs({ ...inputs, QuotaRemindThreshold: e.target.value })
              }
              className='bg-black/20 border-white/10 text-white placeholder:text-white/30 focus-visible:ring-white/20'
            />
            <div className='text-xs text-white/50'>
              {t('低于此额度时将发送邮件提醒用户')}
            </div>
          </div>
        </div>

        <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6 pt-4'>
          <div className='space-y-3 pt-1'>
            <div className='flex flex-col space-y-2'>
              <Label
                className='text-white/80'
                htmlFor='AutomaticDisableChannelEnabled'
              >
                {t('失败时自动禁用通道')}
              </Label>
              <div className='flex items-center h-9'>
                <Switch
                  id='AutomaticDisableChannelEnabled'
                  checked={inputs['AutomaticDisableChannelEnabled']}
                  onCheckedChange={(value) =>
                    setInputs({
                      ...inputs,
                      AutomaticDisableChannelEnabled: value,
                    })
                  }
                  className='data-[state=checked]:bg-white data-[state=unchecked]:bg-white/20'
                />
              </div>
            </div>
          </div>

          <div className='space-y-3 pt-1'>
            <div className='flex flex-col space-y-2'>
              <Label
                className='text-white/80'
                htmlFor='AutomaticEnableChannelEnabled'
              >
                {t('成功时自动启用通道')}
              </Label>
              <div className='flex items-center h-9'>
                <Switch
                  id='AutomaticEnableChannelEnabled'
                  checked={inputs['AutomaticEnableChannelEnabled']}
                  onCheckedChange={(value) =>
                    setInputs({
                      ...inputs,
                      AutomaticEnableChannelEnabled: value,
                    })
                  }
                  className='data-[state=checked]:bg-white data-[state=unchecked]:bg-white/20'
                />
              </div>
            </div>
          </div>
        </div>

        <div className='grid grid-cols-1 md:grid-cols-2 gap-6 pt-4'>
          <div className='space-y-6'>
            <HttpStatusCodeRulesInput
              label={t('自动禁用状态码')}
              placeholder={t('例如：401, 403, 429, 500-599')}
              extraText={t('支持填写单个状态码或范围（含首尾），使用逗号分隔')}
              field={'AutomaticDisableStatusCodes'}
              value={inputs['AutomaticDisableStatusCodes']}
              onChange={(value) =>
                setInputs({ ...inputs, AutomaticDisableStatusCodes: value })
              }
              parsed={parsedAutoDisableStatusCodes}
              invalidText={t('自动禁用状态码格式不正确')}
            />

            <HttpStatusCodeRulesInput
              label={t('自动重试状态码')}
              placeholder={t('例如：401, 403, 429, 500-599')}
              extraText={t(
                '支持填写单个状态码或范围（含首尾），使用逗号分隔；504 和 524 始终不重试，不受此处配置影响',
              )}
              field={'AutomaticRetryStatusCodes'}
              value={inputs['AutomaticRetryStatusCodes']}
              onChange={(value) =>
                setInputs({ ...inputs, AutomaticRetryStatusCodes: value })
              }
              parsed={parsedAutoRetryStatusCodes}
              invalidText={t('自动重试状态码格式不正确')}
            />

            <div className='space-y-2'>
              <Label
                className='text-white/80'
                htmlFor='AutomaticDisableKeywords'
              >
                {t('自动禁用关键词')}
              </Label>
              <Textarea
                id='AutomaticDisableKeywords'
                value={inputs['AutomaticDisableKeywords']}
                placeholder={t('一行一个，不区分大小写')}
                onChange={(e) =>
                  setInputs({
                    ...inputs,
                    AutomaticDisableKeywords: e.target.value,
                  })
                }
                className='min-h-[120px] bg-black/20 border-white/10 text-white placeholder:text-white/30 focus-visible:ring-white/20 resize-y'
              />
              <div className='text-xs text-white/50'>
                {t(
                  '当上游通道返回错误中包含这些关键词时（不区分大小写），自动禁用通道',
                )}
              </div>
            </div>
          </div>
        </div>

        <div className='pt-4'>
          <Button
            onClick={onSubmit}
            disabled={loading}
            className='bg-white text-black hover:bg-white/90'
          >
            {t('保存监控设置')}
          </Button>
        </div>
      </div>
    </div>
  );
}
