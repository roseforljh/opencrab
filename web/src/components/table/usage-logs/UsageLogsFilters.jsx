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

import React from 'react';
import { Button, Form, Typography } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';

import { DATE_RANGE_PRESETS } from '../../../constants/console.constants';

const LogsFilters = ({
  formInitValues,
  setFormApi,
  refresh,
  setShowColumnSelector,
  formApi,
  setLogType,
  loading,
  isAdminUser,
  t,
}) => {
  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => setFormApi(api)}
      onSubmit={refresh}
      allowEmpty={true}
      autoComplete='off'
      layout='vertical'
      trigger='change'
      stopValidateWithError={false}
    >
      <div className='flex flex-col gap-4'>
        <div>
          <Typography.Text strong className='!text-sm !text-white'>
            {t('筛选条件')}
          </Typography.Text>
          <div className='mt-1 text-xs text-white/45'>
            {t('按时间、令牌、模型、分组或请求 ID 快速收窄范围。')}
          </div>
        </div>

        <div className='grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-4'>
          <div className='col-span-1 xl:col-span-2'>
            <Form.DatePicker
              field='dateRange'
              className='w-full'
              type='dateTimeRange'
              placeholder={[t('开始时间'), t('结束时间')]}
              showClear
              pure
              size='default'
              presets={DATE_RANGE_PRESETS.map((preset) => ({
                text: t(preset.text),
                start: preset.start(),
                end: preset.end(),
              }))}
            />
          </div>

          <Form.Input
            field='token_name'
            prefix={<IconSearch />}
            placeholder={t('令牌名称')}
            showClear
            pure
            size='default'
          />

          <Form.Input
            field='model_name'
            prefix={<IconSearch />}
            placeholder={t('模型名称')}
            showClear
            pure
            size='default'
          />

          <Form.Input
            field='group'
            prefix={<IconSearch />}
            placeholder={t('分组')}
            showClear
            pure
            size='default'
          />

          <Form.Input
            field='request_id'
            prefix={<IconSearch />}
            placeholder={t('Request ID')}
            showClear
            pure
            size='default'
          />

          {isAdminUser && (
            <>
              <Form.Input
                field='channel'
                prefix={<IconSearch />}
                placeholder={t('渠道 ID')}
                showClear
                pure
                size='default'
              />
              <Form.Input
                field='username'
                prefix={<IconSearch />}
                placeholder={t('用户名称')}
                showClear
                pure
                size='default'
              />
            </>
          )}
        </div>

        <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
          <div className='w-full lg:w-auto'>
            <Form.Select
              field='logType'
              placeholder={t('日志类型')}
              className='w-full lg:w-auto lg:min-w-[140px]'
              showClear
              pure
              onChange={() => {
                setTimeout(() => {
                  refresh();
                }, 0);
              }}
              size='default'
            >
              <Form.Select.Option value='0'>{t('全部')}</Form.Select.Option>
              <Form.Select.Option value='1'>{t('充值')}</Form.Select.Option>
              <Form.Select.Option value='2'>{t('消费')}</Form.Select.Option>
              <Form.Select.Option value='3'>{t('管理')}</Form.Select.Option>
              <Form.Select.Option value='4'>{t('系统')}</Form.Select.Option>
              <Form.Select.Option value='5'>{t('错误')}</Form.Select.Option>
              <Form.Select.Option value='6'>{t('退款')}</Form.Select.Option>
            </Form.Select>
          </div>

          <div className='flex flex-wrap justify-end gap-2'>
            <Button
              type='primary'
              htmlType='submit'
              loading={loading}
              className='!h-11 !rounded-2xl !border-0 !bg-white/90 hover:!bg-white/80 !px-5 !text-black'
            >
              {t('查询日志')}
            </Button>
            <Button
              type='tertiary'
              className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
              onClick={() => {
                if (formApi) {
                  formApi.reset();
                  setLogType(0);
                  setTimeout(() => {
                    refresh();
                  }, 100);
                }
              }}
            >
              {t('重置条件')}
            </Button>
            <Button
              type='tertiary'
              className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
              onClick={() => setShowColumnSelector(true)}
            >
              {t('列设置')}
            </Button>
          </div>
        </div>
      </div>
    </Form>
  );
};

export default LogsFilters;
