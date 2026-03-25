
import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';

export default function GeneralSettings(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const defaultInputs = {
    'general_setting.docs_link': '',
    RetryTimes: '',
    DefaultCollapseSidebar: false,
  };
  const allowedKeys = Object.keys(defaultInputs);
  const [inputs, setInputs] = useState(defaultInputs);
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((inputs) => ({ ...inputs, [fieldName]: value }));
    };
  }

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = inputs[item.key];
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
          if (res.includes(undefined)) return showError(t('保存失败，请重试'));
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
    const nextInputs = { ...defaultInputs };
    for (const key of allowedKeys) {
      if (props.options?.[key] !== undefined) {
        nextInputs[key] = props.options[key];
      }
    }
    setInputs(nextInputs);
    setInputsRow(structuredClone(nextInputs));
  }, [props.options]);

  return (
    <div className='space-y-6'>
      <div className='flex flex-col space-y-4'>
        <h3 className='text-lg font-medium leading-none text-white'>
          {t('实例基础行为')}
        </h3>

        <div className='grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6'>
          <div className='space-y-2'>
            <Label
              className='text-white/80'
              htmlFor='general_setting.docs_link'
            >
              {t('文档地址')}
            </Label>
            <Input
              id='general_setting.docs_link'
              value={inputs['general_setting.docs_link']}
              placeholder={t('例如 https://docs.opencrab.pro')}
              onChange={(e) =>
                handleFieldChange('general_setting.docs_link')(e.target.value)
              }
              className='bg-black/20 border-white/10 text-white placeholder:text-white/30 focus-visible:ring-white/20'
            />
          </div>

          <div className='space-y-2'>
            <Label className='text-white/80' htmlFor='RetryTimes'>
              {t('失败重试次数')}
            </Label>
            <Input
              id='RetryTimes'
              value={inputs['RetryTimes']}
              placeholder={t('失败重试次数')}
              onChange={(e) => handleFieldChange('RetryTimes')(e.target.value)}
              className='bg-black/20 border-white/10 text-white placeholder:text-white/30 focus-visible:ring-white/20'
            />
          </div>

          <div className='space-y-3 pt-1'>
            <div className='flex flex-col space-y-2'>
              <Label className='text-white/80' htmlFor='DefaultCollapseSidebar'>
                {t('默认折叠侧边栏')}
              </Label>
              <div className='flex items-center h-9'>
                <Switch
                  id='DefaultCollapseSidebar'
                  checked={inputs['DefaultCollapseSidebar']}
                  onCheckedChange={handleFieldChange('DefaultCollapseSidebar')}
                  className='data-[state=checked]:bg-white data-[state=unchecked]:bg-white/20'
                />
              </div>
            </div>
          </div>
        </div>

        <div className='pt-2'>
          <Button
            onClick={onSubmit}
            disabled={loading}
            className='bg-white text-black hover:bg-white/90'
          >
            {t('保存实例基础行为设置')}
          </Button>
        </div>
      </div>
    </div>
  );
}
