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
import { API, showError, toBoolean } from '../../helpers';
import SettingsDataDashboard from '../../pages/Setting/Dashboard/SettingsDataDashboard';

const DashboardSetting = () => {
  let [inputs, setInputs] = useState({
    DataExportEnabled: false,
    DataExportDefaultTime: 'hour',
    DataExportInterval: 5,
  });

  let [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      setInputs((prevInputs) => {
        const nextInputs = { ...prevInputs };

        data.forEach((item) => {
          if (item.key === 'DataExportEnabled') {
            nextInputs.DataExportEnabled = toBoolean(item.value);
          }
          if (item.key === 'DataExportDefaultTime') {
            nextInputs.DataExportDefaultTime =
              item.value ?? prevInputs.DataExportDefaultTime;
          }
          if (item.key === 'DataExportInterval') {
            const interval = Number(item.value);
            nextInputs.DataExportInterval = Number.isNaN(interval)
              ? prevInputs.DataExportInterval
              : interval;
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
    <Spin spinning={loading} size='large'>
      <Card style={{ marginTop: '10px' }}>
        <SettingsDataDashboard options={inputs} refresh={onRefresh} />
      </Card>
    </Spin>
  );
};

export default DashboardSetting;
