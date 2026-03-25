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

const ChannelsFilters = ({
  setEditingChannel,
  setShowEdit,
  refresh,
  setShowColumnSelector,
  formInitValues,
  setFormApi,
  searchChannels,
  enableTagMode,
  formApi,
  groupOptions,
  loading,
  searching,
  t,
}) => {
  return (
    <div className='flex flex-col gap-4 w-full'>
      <div>
        <Typography.Text strong className='!text-sm !text-white'>
          {t('搜索与筛选')}
        </Typography.Text>
        <div className='mt-1 text-xs text-white/45'>
          {t('按名称、密钥、地址、模型或分组快速定位异常渠道。')}
        </div>
      </div>

      <div className='flex flex-wrap gap-2'>
        <Button
          theme='light'
          type='primary'
          className='!h-11 !rounded-2xl !border-0 !bg-white/90 hover:!bg-white/80 !px-5 !text-black'
          onClick={() => {
            setEditingChannel({
              id: undefined,
            });
            setShowEdit(true);
          }}
        >
          {t('添加渠道')}
        </Button>

        <Button
          type='tertiary'
          className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
          onClick={refresh}
        >
          {t('刷新列表')}
        </Button>

        <Button
          type='tertiary'
          className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
          onClick={() => setShowColumnSelector(true)}
        >
          {t('列设置')}
        </Button>
      </div>

      <Form
        initValues={formInitValues}
        getFormApi={(api) => setFormApi(api)}
        onSubmit={() => searchChannels(enableTagMode)}
        allowEmpty={true}
        autoComplete='off'
        layout='vertical'
        trigger='change'
        stopValidateWithError={false}
        className='w-full'
      >
        <div className='grid grid-cols-1 gap-2 lg:grid-cols-4'>
          <Form.Input
            field='searchKeyword'
            prefix={<IconSearch />}
            placeholder={t('渠道ID，名称，密钥，API地址')}
            showClear
            pure
            size='default'
          />
          <Form.Input
            field='searchModel'
            prefix={<IconSearch />}
            placeholder={t('模型关键字')}
            showClear
            pure
            size='default'
          />
          <Form.Select
            field='searchGroup'
            placeholder={t('选择分组')}
            optionList={[
              { label: t('选择分组'), value: null },
              ...groupOptions,
            ]}
            className='w-full'
            showClear
            pure
            size='default'
            onChange={() => {
              setTimeout(() => {
                searchChannels(enableTagMode);
              }, 0);
            }}
          />
          <div className='flex flex-wrap gap-2'>
            <Button
              type='primary'
              htmlType='submit'
              loading={loading || searching}
              className='!h-11 !rounded-2xl !border-0 !bg-white/90 hover:!bg-white/80 !px-5 !text-black'
            >
              {t('查询渠道')}
            </Button>
            <Button
              type='tertiary'
              className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
              onClick={() => {
                if (formApi) {
                  formApi.reset();
                  setTimeout(() => {
                    refresh();
                  }, 100);
                }
              }}
            >
              {t('重置条件')}
            </Button>
          </div>
        </div>
      </Form>
    </div>
  );
};

export default ChannelsFilters;
