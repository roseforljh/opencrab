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
import {
  Button,
  Dropdown,
  Modal,
  Switch,
  Typography,
  Select,
} from '@douyinfe/semi-ui';
import CompactModeToggle from '../../common/ui/CompactModeToggle';

const ChannelsActions = ({
  enableBatchDelete,
  batchDeleteChannels,
  setShowBatchSetTag,
  testAllChannels,
  fixChannelsAbilities,
  updateAllChannelsBalance,
  deleteAllDisabledChannels,
  applyAllUpstreamUpdates,
  detectAllUpstreamUpdates,
  detectAllUpstreamUpdatesLoading,
  applyAllUpstreamUpdatesLoading,
  compactMode,
  setCompactMode,
  idSort,
  setIdSort,
  setEnableBatchDelete,
  enableTagMode,
  setEnableTagMode,
  statusFilter,
  setStatusFilter,
  getFormValues,
  loadChannels,
  searchChannels,
  activeTypeKey,
  activePage,
  pageSize,
  setActivePage,
  t,
}) => {
  return (
    <div className='flex flex-col gap-4'>
      <div>
        <Typography.Text strong className='!text-sm !text-white'>
          {t('接入操作台')}
        </Typography.Text>
        <div className='mt-1 text-xs text-white/45'>
          {t('把常用操作放前面，把高风险或低频批量操作收进菜单。')}
        </div>
      </div>

      <div className='flex flex-wrap gap-2'>
        <Button
          disabled={!enableBatchDelete}
          type='danger'
          className='!h-11 !rounded-2xl !border !border-red-400/20 !bg-red-500/10 !px-5 !text-red-100 hover:!bg-red-500/20'
          onClick={() => {
            Modal.confirm({
              title: t('确定是否要删除所选通道？'),
              content: t('此修改将不可逆'),
              onOk: () => batchDeleteChannels(),
            });
          }}
        >
          {t('删除所选通道')}
        </Button>

        <Button
          disabled={!enableBatchDelete}
          type='tertiary'
          className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
          onClick={() => setShowBatchSetTag(true)}
        >
          {t('批量设置标签')}
        </Button>

        <Dropdown
          trigger='click'
          render={
            <Dropdown.Menu className='!rounded-2xl !border !border-white/10 !bg-[#0b1220] !p-2 !shadow-[0_24px_80px_rgba(0,0,0,0.45)]'>
              <Dropdown.Item>
                <Button
                  type='tertiary'
                  className='w-full'
                  loading={detectAllUpstreamUpdatesLoading}
                  disabled={detectAllUpstreamUpdatesLoading}
                  onClick={() => {
                    Modal.confirm({
                      title: t('确定？'),
                      content: t('确定要测试所有未手动禁用渠道吗？'),
                      onOk: () => testAllChannels(),
                      size: 'small',
                      centered: true,
                    });
                  }}
                >
                  {t('测试所有未手动禁用渠道')}
                </Button>
              </Dropdown.Item>
              <Dropdown.Item>
                <Button
                  className='w-full'
                  onClick={() => {
                    Modal.confirm({
                      title: t('确定是否要修复数据库一致性？'),
                      content: t(
                        '进行该操作时，可能导致渠道访问错误，请仅在数据库出现问题时使用',
                      ),
                      onOk: () => fixChannelsAbilities(),
                      size: 'sm',
                      centered: true,
                    });
                  }}
                >
                  {t('修复数据库一致性')}
                </Button>
              </Dropdown.Item>
              <Dropdown.Item>
                <Button
                  type='secondary'
                  className='w-full'
                  onClick={() => {
                    Modal.confirm({
                      title: t('确定？'),
                      content: t('确定要更新所有已启用通道余额吗？'),
                      onOk: () => updateAllChannelsBalance(),
                      size: 'sm',
                      centered: true,
                    });
                  }}
                >
                  {t('更新所有已启用通道余额')}
                </Button>
              </Dropdown.Item>
              <Dropdown.Item>
                <Button
                  type='tertiary'
                  className='w-full'
                  onClick={() => {
                    Modal.confirm({
                      title: t('确定？'),
                      content: t(
                        '确定要仅检测全部渠道上游模型更新吗？（不执行新增/删除）',
                      ),
                      onOk: () => detectAllUpstreamUpdates(),
                      size: 'sm',
                      centered: true,
                    });
                  }}
                >
                  {t('检测全部渠道上游更新')}
                </Button>
              </Dropdown.Item>
              <Dropdown.Item>
                <Button
                  type='primary'
                  className='w-full'
                  loading={applyAllUpstreamUpdatesLoading}
                  disabled={applyAllUpstreamUpdatesLoading}
                  onClick={() => {
                    Modal.confirm({
                      title: t('确定？'),
                      content: t('确定要对全部渠道执行上游模型更新吗？'),
                      onOk: () => applyAllUpstreamUpdates(),
                      size: 'sm',
                      centered: true,
                    });
                  }}
                >
                  {t('处理全部渠道上游更新')}
                </Button>
              </Dropdown.Item>
              <Dropdown.Item>
                <Button
                  type='danger'
                  className='w-full'
                  onClick={() => {
                    Modal.confirm({
                      title: t('确定是否要删除禁用通道？'),
                      content: t('此修改将不可逆'),
                      onOk: () => deleteAllDisabledChannels(),
                      size: 'sm',
                      centered: true,
                    });
                  }}
                >
                  {t('删除禁用通道')}
                </Button>
              </Dropdown.Item>
            </Dropdown.Menu>
          }
        >
          <Button
            theme='light'
            type='tertiary'
            className='!h-11 !rounded-2xl !border !border-white/10 !bg-white/6 !px-5 !text-white hover:!bg-white/10'
          >
            {t('更多批量操作')}
          </Button>
        </Dropdown>

        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </div>

      <div className='grid grid-cols-1 gap-3 lg:grid-cols-4'>
        <div className='flex items-center justify-between rounded-[20px] border border-white/10 bg-[#0d1527]/80 px-4 py-3'>
          <Typography.Text strong>{t('使用ID排序')}</Typography.Text>
          <Switch
            checked={idSort}
            onChange={(v) => {
              localStorage.setItem('id-sort', v + '');
              setIdSort(v);
              const { searchKeyword, searchGroup, searchModel } =
                getFormValues();
              if (
                searchKeyword === '' &&
                searchGroup === '' &&
                searchModel === ''
              ) {
                loadChannels(activePage, pageSize, v, enableTagMode);
              } else {
                searchChannels(
                  enableTagMode,
                  activeTypeKey,
                  statusFilter,
                  activePage,
                  pageSize,
                  v,
                );
              }
            }}
          />
        </div>

        <div className='flex items-center justify-between rounded-[20px] border border-white/10 bg-[#0d1527]/80 px-4 py-3'>
          <Typography.Text strong className='!text-white'>
            {t('开启批量操作')}
          </Typography.Text>
          <Switch
            checked={enableBatchDelete}
            onChange={(v) => {
              localStorage.setItem('enable-batch-delete', v + '');
              setEnableBatchDelete(v);
            }}
          />
        </div>

        <div className='flex items-center justify-between rounded-[20px] border border-white/10 bg-[#0d1527]/80 px-4 py-3'>
          <Typography.Text strong className='!text-white'>
            {t('标签聚合模式')}
          </Typography.Text>
          <Switch
            checked={enableTagMode}
            onChange={(v) => {
              localStorage.setItem('enable-tag-mode', v + '');
              setEnableTagMode(v);
              setActivePage(1);
              loadChannels(1, pageSize, idSort, v);
            }}
          />
        </div>

        <div className='flex items-center justify-between rounded-[20px] border border-white/10 bg-[#0d1527]/80 px-4 py-3'>
          <Typography.Text strong className='!text-white'>
            {t('状态筛选')}
          </Typography.Text>
          <Select
            value={statusFilter}
            style={{ width: 120 }}
            onChange={(v) => {
              localStorage.setItem('channel-status-filter', v);
              setStatusFilter(v);
              setActivePage(1);
              loadChannels(
                1,
                pageSize,
                idSort,
                enableTagMode,
                activeTypeKey,
                v,
              );
            }}
          >
            <Select.Option value='all'>{t('全部')}</Select.Option>
            <Select.Option value='enabled'>{t('已启用')}</Select.Option>
            <Select.Option value='disabled'>{t('已禁用')}</Select.Option>
          </Select>
        </div>
      </div>
    </div>
  );
};

export default ChannelsActions;
