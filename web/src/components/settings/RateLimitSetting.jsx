
import React, { useEffect, useState } from 'react';
import { Card } from '@douyinfe/semi-ui';
import { Loader2 } from 'lucide-react';

import { API, showError, toBoolean } from '../../helpers';
import { useTranslation } from 'react-i18next';
import RequestRateLimit from '../../pages/Setting/RateLimit/SettingsRequestRateLimit';

const RateLimitSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    ModelRequestRateLimitEnabled: false,
    ModelRequestRateLimitCount: 0,
    ModelRequestRateLimitSuccessCount: 1000,
    ModelRequestRateLimitDurationMinutes: 1,
    ModelRequestRateLimitGroup: '',
  });

  let [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key === 'ModelRequestRateLimitGroup') {
          item.value = JSON.stringify(JSON.parse(item.value), null, 2);
        }

        if (item.key.endsWith('Enabled')) {
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
        {/* AI请求速率限制 */}
        <Card className='!p-0 !rounded-[32px] !border !border-white/10 !bg-white/6 !shadow-[0_30px_100px_rgba(0,0,0,0.34)] !backdrop-blur-2xl !ring-0'>
          <RequestRateLimit options={inputs} refresh={onRefresh} />
        </Card>
      </div>
    </div>
  );
};

export default RateLimitSetting;
