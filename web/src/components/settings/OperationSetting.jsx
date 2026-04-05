
import React, { useEffect, useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Loader2 } from 'lucide-react';
import SettingsGeneral from '../../pages/Setting/Operation/SettingsGeneral';
import SettingsMonitoring from '../../pages/Setting/Operation/SettingsMonitoring';
import { API, showError, toBoolean } from '../../helpers';

const OperationSetting = () => {
  const [inputs, setInputs] = useState({
    'general_setting.docs_link': '',
    RetryTimes: 0,
    DefaultCollapseSidebar: false,
    ChannelDisableThreshold: 0,
    QuotaRemindThreshold: 0,
    AutomaticDisableChannelEnabled: false,
    AutomaticEnableChannelEnabled: false,
    AutomaticDisableKeywords: '',
    AutomaticDisableStatusCodes: '401',
    AutomaticRetryStatusCodes:
      '100-199,300-399,401-407,409-499,500-503,505-523,525-599',
    'monitor_setting.auto_test_channel_enabled': false,
    'monitor_setting.auto_test_channel_minutes': 10,
  });

  const [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      setInputs((prev) => {
        const nextInputs = { ...prev };
        data.forEach((item) => {
          if (!(item.key in nextInputs)) {
            return;
          }
          if (typeof nextInputs[item.key] === 'boolean') {
            nextInputs[item.key] = toBoolean(item.value);
          } else {
            nextInputs[item.key] = item.value;
          }
        });
        return nextInputs;
      });
    } else {
      showError(message);
    }
  };

  async function onRefresh() {
    try {
      setLoading(true);
      await getOptions();
    } catch {
      showError('刷新失败');
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
      <Card className='!p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
        <CardContent className='p-4 md:p-6'>
          <SettingsGeneral options={inputs} refresh={onRefresh} />
        </CardContent>
      </Card>
      <Card className='mt-6 !p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
        <CardContent className='p-4 md:p-6'>
          <SettingsMonitoring options={inputs} refresh={onRefresh} />
        </CardContent>
      </Card>
    </div>
  );
};

export default OperationSetting;
