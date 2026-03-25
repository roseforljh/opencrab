
import React, { useState, useEffect } from 'react';
import { RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../../helpers';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { DataTable } from '../../../ui/data-table';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
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
import { Card, CardContent } from '@/components/ui/card';

const MultiKeyManageModal = ({ visible, onCancel, channel, onRefresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [keyStatusList, setKeyStatusList] = useState([]);
  const [operationLoading, setOperationLoading] = useState({});

  // Pagination states
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);

  // Statistics states
  const [enabledCount, setEnabledCount] = useState(0);
  const [manualDisabledCount, setManualDisabledCount] = useState(0);
  const [autoDisabledCount, setAutoDisabledCount] = useState(0);

  // Filter states
  const [statusFilter, setStatusFilter] = useState(null); // null=all, 1=enabled, 2=manual_disabled, 3=auto_disabled

  // Load key status data
  const loadKeyStatus = async (
    page = currentPage,
    size = pageSize,
    status = statusFilter,
  ) => {
    if (!channel?.id) return;

    setLoading(true);
    try {
      const requestData = {
        channel_id: channel.id,
        action: 'get_key_status',
        page: page,
        page_size: size,
      };

      // Add status filter if specified
      if (status !== null) {
        requestData.status = status;
      }

      const res = await API.post('/api/channel/multi_key/manage', requestData);

      if (res.data.success) {
        const data = res.data.data;
        setKeyStatusList(data.keys || []);
        setTotal(data.total || 0);
        setCurrentPage(data.page || 1);
        setPageSize(data.page_size || 10);
        setTotalPages(data.total_pages || 0);

        // Update statistics (these are always the overall statistics)
        setEnabledCount(data.enabled_count || 0);
        setManualDisabledCount(data.manual_disabled_count || 0);
        setAutoDisabledCount(data.auto_disabled_count || 0);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      console.error(error);
      showError(t('获取密钥状态失败'));
    } finally {
      setLoading(false);
    }
  };

  // Disable a specific key
  const handleDisableKey = async (keyIndex) => {
    const operationId = `disable_${keyIndex}`;
    setOperationLoading((prev) => ({ ...prev, [operationId]: true }));

    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'disable_key',
        key_index: keyIndex,
      });

      if (res.data.success) {
        showSuccess(t('密钥已禁用'));
        await loadKeyStatus(currentPage, pageSize); // Reload current page
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('禁用密钥失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, [operationId]: false }));
    }
  };

  // Enable a specific key
  const handleEnableKey = async (keyIndex) => {
    const operationId = `enable_${keyIndex}`;
    setOperationLoading((prev) => ({ ...prev, [operationId]: true }));

    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'enable_key',
        key_index: keyIndex,
      });

      if (res.data.success) {
        showSuccess(t('密钥已启用'));
        await loadKeyStatus(currentPage, pageSize); // Reload current page
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('启用密钥失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, [operationId]: false }));
    }
  };

  // Enable all disabled keys
  const handleEnableAll = async () => {
    setOperationLoading((prev) => ({ ...prev, enable_all: true }));

    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'enable_all_keys',
      });

      if (res.data.success) {
        showSuccess(res.data.message || t('已启用所有密钥'));
        // Reset to first page after bulk operation
        setCurrentPage(1);
        await loadKeyStatus(1, pageSize);
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('启用所有密钥失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, enable_all: false }));
    }
  };

  // Disable all enabled keys
  const handleDisableAll = async () => {
    setOperationLoading((prev) => ({ ...prev, disable_all: true }));

    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'disable_all_keys',
      });

      if (res.data.success) {
        showSuccess(res.data.message || t('已禁用所有密钥'));
        // Reset to first page after bulk operation
        setCurrentPage(1);
        await loadKeyStatus(1, pageSize);
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('禁用所有密钥失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, disable_all: false }));
    }
  };

  // Delete all disabled keys
  const handleDeleteDisabledKeys = async () => {
    setOperationLoading((prev) => ({ ...prev, delete_disabled: true }));

    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'delete_disabled_keys',
      });

      if (res.data.success) {
        showSuccess(res.data.message);
        // Reset to first page after deletion as data structure might change
        setCurrentPage(1);
        await loadKeyStatus(1, pageSize);
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('删除禁用密钥失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, delete_disabled: false }));
    }
  };

  // Delete a specific key
  const handleDeleteKey = async (keyIndex) => {
    const operationId = `delete_${keyIndex}`;
    setOperationLoading((prev) => ({ ...prev, [operationId]: true }));

    try {
      const res = await API.post('/api/channel/multi_key/manage', {
        channel_id: channel.id,
        action: 'delete_key',
        key_index: keyIndex,
      });

      if (res.data.success) {
        showSuccess(t('密钥已删除'));
        await loadKeyStatus(currentPage, pageSize); // Reload current page
        onRefresh && onRefresh(); // Refresh parent component
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('删除密钥失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, [operationId]: false }));
    }
  };

  // Handle page change
  const handlePageChange = (page) => {
    setCurrentPage(page);
    loadKeyStatus(page, pageSize);
  };

  // Handle page size change
  const handlePageSizeChange = (size) => {
    setPageSize(size);
    setCurrentPage(1); // Reset to first page
    loadKeyStatus(1, size);
  };

  // Handle status filter change
  const handleStatusFilterChange = (status) => {
    setStatusFilter(status);
    setCurrentPage(1); // Reset to first page when filter changes
    loadKeyStatus(1, pageSize, status);
  };

  // Effect to load data when modal opens
  useEffect(() => {
    if (visible && channel?.id) {
      setCurrentPage(1); // Reset to first page when opening
      loadKeyStatus(1, pageSize);
    }
  }, [visible, channel?.id]);

  // Reset pagination when modal closes
  useEffect(() => {
    if (!visible) {
      setCurrentPage(1);
      setKeyStatusList([]);
      setTotal(0);
      setTotalPages(0);
      setEnabledCount(0);
      setManualDisabledCount(0);
      setAutoDisabledCount(0);
      setStatusFilter(null); // Reset filter
    }
  }, [visible]);

  // Percentages for progress display
  const enabledPercent =
    total > 0 ? Math.round((enabledCount / total) * 100) : 0;
  const manualDisabledPercent =
    total > 0 ? Math.round((manualDisabledCount / total) * 100) : 0;
  const autoDisabledPercent =
    total > 0 ? Math.round((autoDisabledCount / total) * 100) : 0;

  const [confirmAction, setConfirmAction] = useState(null);

  const renderStatusTag = (status) => {
    switch (status) {
      case 1:
        return (
          <Badge className='border-green-500/20 bg-green-500/15 text-green-200'>
            {t('已启用')}
          </Badge>
        );
      case 2:
        return (
          <Badge className='border-red-500/20 bg-red-500/15 text-red-200'>
            {t('已禁用')}
          </Badge>
        );
      case 3:
        return (
          <Badge className='border-amber-500/20 bg-amber-500/15 text-amber-200'>
            {t('自动禁用')}
          </Badge>
        );
      default:
        return (
          <Badge className='border-white/10 bg-white/10 text-white/70'>
            {t('未知状态')}
          </Badge>
        );
    }
  };

  const formatStatusFilterValue = (value) => {
    if (value === null || value === undefined) {
      return 'all';
    }
    return String(value);
  };

  const handleConfirmAction = async () => {
    if (!confirmAction) return;
    const action = confirmAction;
    setConfirmAction(null);
    await action.onConfirm();
  };

  const columns = [
    {
      id: 'index',
      header: t('索引'),
      accessorKey: 'index',
      cell: ({ row }) => `#${row.original.index}`,
    },
    {
      id: 'status',
      header: t('状态'),
      cell: ({ row }) => renderStatusTag(row.original.status),
    },
    {
      id: 'reason',
      header: t('禁用原因'),
      cell: ({ row }) => {
        const { reason, status } = row.original;
        if (status === 1 || !reason) {
          return <span className='text-white/35'>-</span>;
        }
        return (
          <span className='block max-w-[200px] truncate text-white/80' title={reason}>
            {reason}
          </span>
        );
      },
    },
    {
      id: 'disabled_time',
      header: t('禁用时间'),
      cell: ({ row }) => {
        const { disabled_time: time, status } = row.original;
        if (status === 1 || !time) {
          return <span className='text-white/35'>-</span>;
        }
        return (
          <span className='text-xs text-white/70' title={timestamp2string(time)}>
            {timestamp2string(time)}
          </span>
        );
      },
    },
    {
      id: 'action',
      header: t('操作'),
      cell: ({ row }) => {
        const record = row.original;
        return (
          <div className='flex flex-wrap gap-2'>
            {record.status === 1 ? (
              <Button
                type='button'
                size='sm'
                variant='destructive'
                onClick={() => handleDisableKey(record.index)}
                disabled={operationLoading[`disable_${record.index}`]}
              >
                {operationLoading[`disable_${record.index}`] ? t('处理中...') : t('禁用')}
              </Button>
            ) : (
              <Button
                type='button'
                size='sm'
                onClick={() => handleEnableKey(record.index)}
                disabled={operationLoading[`enable_${record.index}`]}
              >
                {operationLoading[`enable_${record.index}`] ? t('处理中...') : t('启用')}
              </Button>
            )}
            <Button
              type='button'
              size='sm'
              variant='destructive'
              onClick={() =>
                setConfirmAction({
                  title: t('确定要删除此密钥吗？'),
                  description: t('此操作不可撤销，将永久删除该密钥'),
                  onConfirm: () => handleDeleteKey(record.index),
                })
              }
              disabled={operationLoading[`delete_${record.index}`]}
            >
              {operationLoading[`delete_${record.index}`] ? t('处理中...') : t('删除')}
            </Button>
          </div>
        );
      },
    },
  ];

  const stats = [
    {
      label: t('已启用'),
      value: enabledCount,
      percent: enabledPercent,
      color: '#22c55e',
    },
    {
      label: t('手动禁用'),
      value: manualDisabledCount,
      percent: manualDisabledPercent,
      color: '#ef4444',
    },
    {
      label: t('自动禁用'),
      value: autoDisabledCount,
      percent: autoDisabledPercent,
      color: '#f59e0b',
    },
  ];

  return (
    <>
      <Dialog open={visible} onOpenChange={(open) => !open && onCancel?.()}>
        <DialogContent className='max-w-[1100px] border-white/10 bg-black text-white'>
          <DialogHeader>
            <div className='flex flex-wrap items-center gap-2'>
              <DialogTitle>{t('多密钥管理')}</DialogTitle>
              {channel?.name && (
                <Badge className='border-white/10 bg-white/10 text-white/80'>
                  {channel.name}
                </Badge>
              )}
              <Badge className='border-white/10 bg-white/10 text-white/80'>
                {t('总密钥数')}: {total}
              </Badge>
              {channel?.channel_info?.multi_key_mode && (
                <Badge className='border-white/10 bg-white/10 text-white/80'>
                  {channel.channel_info.multi_key_mode === 'random'
                    ? t('随机模式')
                    : t('轮询模式')}
                </Badge>
              )}
            </div>
          </DialogHeader>

          <div className='mb-5 flex flex-col gap-4'>
            <div className='grid gap-3 md:grid-cols-3'>
              {stats.map((item) => (
                <Card key={item.label} className='border-white/10 bg-white/5 py-0'>
                  <CardContent className='space-y-3 p-4'>
                    <div className='flex items-center gap-2'>
                      <span
                        className='h-2.5 w-2.5 rounded-full'
                        style={{ backgroundColor: item.color }}
                      />
                      <span className='text-sm text-white/60'>{item.label}</span>
                    </div>
                    <div className='flex items-end gap-2'>
                      <span
                        className='text-2xl font-semibold'
                        style={{ color: item.color }}
                      >
                        {item.value}
                      </span>
                      <span className='text-white/45'>/ {total}</span>
                    </div>
                    <div className='h-1.5 overflow-hidden rounded-full bg-white/10'>
                      <div
                        className='h-full rounded-full transition-all'
                        style={{
                          width: `${item.percent}%`,
                          backgroundColor: item.color,
                        }}
                      />
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>

            <Card className='border-white/10 bg-white/5 py-0'>
              <CardContent className='space-y-4 p-4'>
                <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Select
                      value={formatStatusFilterValue(statusFilter)}
                      onValueChange={(value) =>
                        handleStatusFilterChange(value === 'all' ? null : Number(value))
                      }
                    >
                      <SelectTrigger className='w-[180px] border-white/10 bg-white/6 text-white'>
                        <SelectValue placeholder={t('全部状态')} />
                      </SelectTrigger>
                      <SelectContent className='border-white/10 bg-black text-white'>
                        <SelectItem value='all'>{t('全部状态')}</SelectItem>
                        <SelectItem value='1'>{t('已启用')}</SelectItem>
                        <SelectItem value='2'>{t('手动禁用')}</SelectItem>
                        <SelectItem value='3'>{t('自动禁用')}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className='flex flex-wrap justify-end gap-2'>
                    <Button
                      type='button'
                      variant='secondary'
                      onClick={() => loadKeyStatus(currentPage, pageSize)}
                      disabled={loading}
                    >
                      <RefreshCw
                        className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`}
                      />
                      {t('刷新')}
                    </Button>
                    {manualDisabledCount + autoDisabledCount > 0 && (
                      <Button
                        type='button'
                        onClick={() =>
                          setConfirmAction({
                            title: t('确定要启用所有密钥吗？'),
                            description: '',
                            onConfirm: handleEnableAll,
                          })
                        }
                        disabled={operationLoading.enable_all}
                      >
                        {operationLoading.enable_all ? t('处理中...') : t('启用全部')}
                      </Button>
                    )}
                    {enabledCount > 0 && (
                      <Button
                        type='button'
                        variant='destructive'
                        onClick={() =>
                          setConfirmAction({
                            title: t('确定要禁用所有的密钥吗？'),
                            description: '',
                            onConfirm: handleDisableAll,
                          })
                        }
                        disabled={operationLoading.disable_all}
                      >
                        {operationLoading.disable_all ? t('处理中...') : t('禁用全部')}
                      </Button>
                    )}
                    <Button
                      type='button'
                      variant='destructive'
                      onClick={() =>
                        setConfirmAction({
                          title: t('确定要删除所有已自动禁用的密钥吗？'),
                          description: t('此操作不可撤销，将永久删除已自动禁用的密钥'),
                          onConfirm: handleDeleteDisabledKeys,
                        })
                      }
                      disabled={operationLoading.delete_disabled}
                    >
                      {operationLoading.delete_disabled
                        ? t('处理中...')
                        : t('删除自动禁用密钥')}
                    </Button>
                  </div>
                </div>

                <DataTable
                  columns={columns}
                  data={keyStatusList}
                  loading={loading}
                  emptyMessage={t('暂无密钥数据')}
                />

                <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
                  <div className='text-sm text-white/50'>
                    {t('第 {{page}} / {{pages}} 页，共 {{total}} 条', {
                      page: currentPage,
                      pages: totalPages || 1,
                      total,
                    })}
                  </div>

                  <div className='flex flex-wrap items-center gap-2'>
                    <Select
                      value={String(pageSize)}
                      onValueChange={(value) => handlePageSizeChange(Number(value))}
                    >
                      <SelectTrigger className='w-[120px] border-white/10 bg-white/6 text-white'>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent className='border-white/10 bg-black text-white'>
                        {[10, 20, 50, 100].map((size) => (
                          <SelectItem key={size} value={String(size)}>
                            {size} / page
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Button
                      type='button'
                      variant='secondary'
                      size='sm'
                      onClick={() => handlePageChange(Math.max(1, currentPage - 1))}
                      disabled={currentPage <= 1 || loading}
                    >
                      {t('上一页')}
                    </Button>
                    <Button
                      type='button'
                      variant='secondary'
                      size='sm'
                      onClick={() =>
                        handlePageChange(
                          Math.min(totalPages || 1, currentPage + 1),
                        )
                      }
                      disabled={currentPage >= totalPages || totalPages <= 1 || loading}
                    >
                      {t('下一页')}
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(confirmAction)}
        onOpenChange={(open) => {
          if (!open) {
            setConfirmAction(null);
          }
        }}
      >
        <AlertDialogContent className='border-white/10 bg-black text-white'>
          <AlertDialogHeader>
            <AlertDialogTitle>{confirmAction?.title}</AlertDialogTitle>
            {confirmAction?.description ? (
              <AlertDialogDescription className='text-white/60'>
                {confirmAction.description}
              </AlertDialogDescription>
            ) : null}
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('取消')}</AlertDialogCancel>
            <AlertDialogAction
              className='bg-red-600 text-white hover:bg-red-700'
              onClick={handleConfirmAction}
            >
              {t('确认')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

export default MultiKeyManageModal;
