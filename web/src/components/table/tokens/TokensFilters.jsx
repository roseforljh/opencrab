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

import React, { useRef } from 'react';
import { Form, Button, Typography } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';

const TokensFilters = ({
  formInitValues,
  setFormApi,
  searchTokens,
  loading,
  searching,
  t,
}) => {
  const formApiRef = useRef(null);

  const handleReset = () => {
    if (!formApiRef.current) return;
    formApiRef.current.reset();
    setTimeout(() => {
      searchTokens();
    }, 100);
  };

  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        setFormApi(api);
        formApiRef.current = api;
      }}
      onSubmit={() => searchTokens(1)}
      allowEmpty={true}
      autoComplete='off'
      layout='vertical'
      trigger='change'
      stopValidateWithError={false}
      className='w-full'
    >
      <div className='flex flex-col gap-3'>
        <div>
          <Typography.Text strong className='!text-sm !text-white'>
            {t('搜索令牌')}
          </Typography.Text>
          <div className='mt-1 text-xs text-white/45'>
            {t('按名称或密钥快速定位令牌，减少在表格中逐行查找。')}
          </div>
        </div>

        <div className='grid grid-cols-1 gap-2 md:grid-cols-2'>
          <Form.Input
            field='searchKeyword'
            prefix={<IconSearch />}
            placeholder={t('搜索关键字')}
            showClear
            pure
            size='default'
          />

          <Form.Input
            field='searchToken'
            prefix={<IconSearch />}
            placeholder={t('密钥')}
            showClear
            pure
            size='default'
          />
        </div>

        <div className='flex flex-wrap gap-2'>
          <Button
            type='primary'
            htmlType='submit'
            loading={loading || searching}
            className='!h-11 !rounded-2xl !border-0 !bg-white/90 hover:!bg-white/80 !px-5 !text-black'
          >
            {t('查询令牌')}
          </Button>

          <Button
            type='tertiary'
            className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
            onClick={handleReset}
          >
            {t('重置条件')}
          </Button>
        </div>
      </div>
    </Form>
  );
};

export default TokensFilters;
