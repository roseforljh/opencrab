/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';
import SettingsGeneral from '../../pages/Setting/Operation/SettingsGeneral';
import SettingsSensitiveWords from '../../pages/Setting/Operation/SettingsSensitiveWords';
import SettingsLog from '../../pages/Setting/Operation/SettingsLog';
import SettingsMonitoring from '../../pages/Setting/Operation/SettingsMonitoring';
import { API, showError, toBoolean } from '../../helpers';

const OperationSetting = () => {
  let [inputs, setInputs] = useState({
    /* 实例基础行为设置 */
    'general_setting.docs_link': '',
    RetryTimes: 0,
    DefaultCollapseSidebar: false,

    /* 敏感词设置 */
    CheckSensitiveEnabled: false,
    CheckSensitiveOnPromptEnabled: false,
    SensitiveWords: '',

    /* 日志设置 */
    LogConsumeEnabled: false,

    /* 监控设置 */
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

  let [loading, setLoading] = useState(false);

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
    <>
      <Spin spinning={loading} size='large'>
        {/* 实例基础行为设置 */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsGeneral options={inputs} refresh={onRefresh} />
        </Card>
        {/* 屏蔽词过滤设置 */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsSensitiveWords options={inputs} refresh={onRefresh} />
        </Card>
        {/* 日志设置 */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsLog options={inputs} refresh={onRefresh} />
        </Card>
        {/* 监控设置 */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsMonitoring options={inputs} refresh={onRefresh} />
        </Card>
      </Spin>
    </>
  );
};

export default OperationSetting;
