
import React from 'react';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
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
  const [pendingAction, setPendingAction] = React.useState(null);

  const confirmAction = (config) => setPendingAction(config);

  return (
    <div className='flex flex-col gap-4'>
      <div>
        <div className='text-sm font-semibold text-white'>
          {t('接入操作台')}
        </div>
        <div className='mt-1 text-xs text-white/45'>
          {t('把常用操作放前面，把高风险或低频批量操作收进菜单。')}
        </div>
      </div>

      <div className='flex flex-wrap gap-2'>
        <Button
          disabled={!enableBatchDelete}
          variant='destructive'
          className='h-11 rounded-2xl border border-red-400/20 bg-red-500/10 px-5 text-red-100 hover:bg-red-500/20'
          onClick={() => {
            confirmAction({
              title: t('确定是否要删除所选通道？'),
              description: t('此修改将不可逆'),
              confirmLabel: t('确认删除'),
              confirmClassName: 'bg-red-500 text-white hover:bg-red-600',
              onConfirm: () => batchDeleteChannels(),
            });
          }}
        >
          {t('删除所选通道')}
        </Button>

        <Button
          disabled={!enableBatchDelete}
          variant='secondary'
          className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
          onClick={() => setShowBatchSetTag(true)}
        >
          {t('批量设置标签')}
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant='secondary'
              className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
            >
              {t('更多批量操作')}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className='border-white/10 bg-[#0b1220] text-white'>
            <DropdownMenuItem
              onClick={() =>
                confirmAction({
                  title: t('确定？'),
                  description: t('确定要测试所有未手动禁用渠道吗？'),
                  confirmLabel: t('开始测试'),
                  onConfirm: () => testAllChannels(),
                })
              }
              disabled={detectAllUpstreamUpdatesLoading}
            >
              {t('测试所有未手动禁用渠道')}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() =>
                confirmAction({
                  title: t('确定是否要修复数据库一致性？'),
                  description: t(
                    '进行该操作时，可能导致渠道访问错误，请仅在数据库出现问题时使用',
                  ),
                  confirmLabel: t('确认修复'),
                  onConfirm: () => fixChannelsAbilities(),
                })
              }
            >
              {t('修复数据库一致性')}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() =>
                confirmAction({
                  title: t('确定？'),
                  description: t('确定要更新所有已启用通道余额吗？'),
                  confirmLabel: t('确认更新'),
                  onConfirm: () => updateAllChannelsBalance(),
                })
              }
            >
              {t('更新所有已启用通道余额')}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() =>
                confirmAction({
                  title: t('确定？'),
                  description: t(
                    '确定要仅检测全部渠道上游模型更新吗？（不执行新增/删除）',
                  ),
                  confirmLabel: t('确认检测'),
                  onConfirm: () => detectAllUpstreamUpdates(),
                })
              }
            >
              {t('检测全部渠道上游更新')}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() =>
                confirmAction({
                  title: t('确定？'),
                  description: t('确定要对全部渠道执行上游模型更新吗？'),
                  confirmLabel: t('确认处理'),
                  onConfirm: () => applyAllUpstreamUpdates(),
                })
              }
              disabled={applyAllUpstreamUpdatesLoading}
            >
              {t('处理全部渠道上游更新')}
            </DropdownMenuItem>
            <DropdownMenuItem
              className='text-red-300 focus:text-red-200'
              onClick={() =>
                confirmAction({
                  title: t('确定是否要删除禁用通道？'),
                  description: t('此修改将不可逆'),
                  confirmLabel: t('确认删除'),
                  confirmClassName: 'bg-red-500 text-white hover:bg-red-600',
                  onConfirm: () => deleteAllDisabledChannels(),
                })
              }
            >
              {t('删除禁用通道')}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </div>

      <div className='grid grid-cols-1 gap-3 lg:grid-cols-4'>
        <div className='flex items-center justify-between rounded-[20px] border border-white/10 bg-[#0d1527]/80 px-4 py-3'>
          <span className='text-sm font-semibold text-white'>
            {t('使用ID排序')}
          </span>
          <Switch
            checked={idSort}
            onCheckedChange={(v) => {
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
          <span className='text-sm font-semibold text-white'>
            {t('开启批量操作')}
          </span>
          <Switch
            checked={enableBatchDelete}
            onCheckedChange={(v) => {
              localStorage.setItem('enable-batch-delete', v + '');
              setEnableBatchDelete(v);
            }}
          />
        </div>

        <div className='flex items-center justify-between rounded-[20px] border border-white/10 bg-[#0d1527]/80 px-4 py-3'>
          <span className='text-sm font-semibold text-white'>
            {t('标签聚合模式')}
          </span>
          <Switch
            checked={enableTagMode}
            onCheckedChange={(v) => {
              localStorage.setItem('enable-tag-mode', v + '');
              setEnableTagMode(v);
              setActivePage(1);
              loadChannels(1, pageSize, idSort, v);
            }}
          />
        </div>

        <div className='flex items-center justify-between rounded-[20px] border border-white/10 bg-[#0d1527]/80 px-4 py-3'>
          <span className='text-sm font-semibold text-white'>
            {t('状态筛选')}
          </span>
          <Select
            value={statusFilter}
            onValueChange={(v) => {
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
            <SelectTrigger className='h-9 w-[120px] border-white/10 bg-white/6 text-white'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent className='border-white/10 bg-[#0b1220] text-white'>
              <SelectItem value='all'>{t('全部')}</SelectItem>
              <SelectItem value='enabled'>{t('已启用')}</SelectItem>
              <SelectItem value='disabled'>{t('已禁用')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <AlertDialog
        open={Boolean(pendingAction)}
        onOpenChange={(open) => {
          if (!open) setPendingAction(null);
        }}
      >
        <AlertDialogContent className='border-white/10 bg-black text-white'>
          <AlertDialogHeader>
            <AlertDialogTitle>{pendingAction?.title}</AlertDialogTitle>
            <AlertDialogDescription className='text-white/60'>
              {pendingAction?.description}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className='border-0 bg-white/5 hover:bg-white/10'>
              {t('取消')}
            </AlertDialogCancel>
            <AlertDialogAction
              className={
                pendingAction?.confirmClassName ||
                'bg-white text-black hover:bg-white/90'
              }
              onClick={() => {
                pendingAction?.onConfirm?.();
                setPendingAction(null);
              }}
            >
              {pendingAction?.confirmLabel || t('确认')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default ChannelsActions;
