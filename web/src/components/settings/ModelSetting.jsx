
import React, { useEffect, useState } from 'react';
import { Card, Tabs } from '@douyinfe/semi-ui';
import { Loader2 } from 'lucide-react';

import { API, showError, showSuccess, toBoolean } from '../../helpers';
import { useTranslation } from 'react-i18next';
import SettingGeminiModel from '../../pages/Setting/Model/SettingGeminiModel';
import SettingClaudeModel from '../../pages/Setting/Model/SettingClaudeModel';
import SettingGlobalModel from '../../pages/Setting/Model/SettingGlobalModel';
import SettingGrokModel from '../../pages/Setting/Model/SettingGrokModel';
import SettingsChannelAffinity from '../../pages/Setting/Operation/SettingsChannelAffinity';

const ModelSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    'gemini.safety_settings': '',
    'gemini.version_settings': '',
    'gemini.supported_imagine_models': '',
    'gemini.remove_function_response_id_enabled': true,
    'claude.model_headers_settings': '',
    'claude.thinking_adapter_enabled': true,
    'claude.default_max_tokens': '',
    'claude.thinking_adapter_budget_tokens_percentage': 0.8,
    'global.pass_through_request_enabled': false,
    'global.thinking_model_blacklist': '[]',
    'global.chat_completions_to_responses_policy': '{}',
    'general_setting.ping_interval_enabled': false,
    'general_setting.ping_interval_seconds': 60,
    'gemini.thinking_adapter_enabled': false,
    'gemini.thinking_adapter_budget_tokens_percentage': 0.6,
    'grok.violation_deduction_enabled': true,
    'grok.violation_deduction_amount': 0.05,
  });

  let [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (
          item.key === 'gemini.safety_settings' ||
          item.key === 'gemini.version_settings' ||
          item.key === 'claude.model_headers_settings' ||
          item.key === 'claude.default_max_tokens' ||
          item.key === 'gemini.supported_imagine_models' ||
          item.key === 'global.thinking_model_blacklist' ||
          item.key === 'global.chat_completions_to_responses_policy'
        ) {
          if (item.value !== '') {
            try {
              item.value = JSON.stringify(JSON.parse(item.value), null, 2);
            } catch (e) {
              // Keep raw value so user can fix it, and avoid crashing the page.
              console.error(`Invalid JSON for option ${item.key}:`, e);
            }
          }
        }
        // Keep boolean config keys ending with enabled/Enabled so UI parses correctly.
        if (item.key.endsWith('Enabled') || item.key.endsWith('enabled')) {
          newInputs[item.key] = toBoolean(item.value);
        } else {
          newInputs[item.key] = item.value;
        }
      });

      setInputs(newInputs);
    } else {
      showError(message);
    }
  };
  async function onRefresh() {
    try {
      setLoading(true);
      await getOptions();
      // showSuccess('刷新成功');
    } catch (error) {
      showError('刷新失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    onRefresh();
  }, []);

  return (
    <div className='relative'>
      {loading && (
        <div className='absolute inset-0 z-50 flex items-center justify-center rounded-[32px] bg-black/20 backdrop-blur-sm'>
          <Loader2 className='h-8 w-8 animate-spin text-white' />
        </div>
      )}
      <div className='flex flex-col gap-6'>
        {/* OpenAI */}
        <Card className='!p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
          <SettingGlobalModel options={inputs} refresh={onRefresh} />
        </Card>
        {/* Channel affinity */}
        <Card className='!p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
          <SettingsChannelAffinity options={inputs} refresh={onRefresh} />
        </Card>
        {/* Gemini */}
        <Card className='!p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
          <SettingGeminiModel options={inputs} refresh={onRefresh} />
        </Card>
        {/* Claude */}
        <Card className='!p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
          <SettingClaudeModel options={inputs} refresh={onRefresh} />
        </Card>
        {/* Grok */}
        <Card className='!p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
          <SettingGrokModel options={inputs} refresh={onRefresh} />
        </Card>
      </div>
    </div>
  );
};

export default ModelSetting;
