import React, { useState, useEffect } from 'react';
import { Layers, Plus } from 'lucide-react';
import {
  API,
  showError,
  showSuccess,
  stringToColor,
} from '../../../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import EditPrefillGroupModal from './EditPrefillGroupModal';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
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
import { DataTable } from '../../../ui/data-table';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';

const PrefillGroupManagement = ({ visible, onClose }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [groups, setGroups] = useState([]);
  const [showEdit, setShowEdit] = useState(false);
  const [editingGroup, setEditingGroup] = useState({ id: undefined });
  const [deletingGroup, setDeletingGroup] = useState(null);

  const typeOptions = [
    { label: t('模型组'), value: 'model' },
    { label: t('标签组'), value: 'tag' },
    { label: t('端点组'), value: 'endpoint' },
  ];

  // 加载组列表
  const loadGroups = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/prefill_group');
      if (res.data.success) {
        setGroups(res.data.data || []);
      } else {
        showError(res.data.message || t('获取组列表失败'));
      }
    } catch (error) {
      showError(t('获取组列表失败'));
    }
    setLoading(false);
  };

  // 删除组
  const deleteGroup = async (id) => {
    try {
      const res = await API.delete(`/api/prefill_group/${id}`);
      if (res.data.success) {
        showSuccess(t('删除成功'));
        loadGroups();
      } else {
        showError(res.data.message || t('删除失败'));
      }
    } catch (error) {
      showError(t('删除失败'));
    }
  };

  // 编辑组
  const handleEdit = (group = {}) => {
    setEditingGroup(group);
    setShowEdit(true);
  };

  // 关闭编辑
  const closeEdit = () => {
    setShowEdit(false);
    setTimeout(() => {
      setEditingGroup({ id: undefined });
    }, 300);
  };

  // 编辑成功回调
  const handleEditSuccess = () => {
    closeEdit();
    loadGroups();
  };

  const renderLimitedBadges = (items) => {
    if (!items || items.length === 0) {
      return <span className='text-white/45'>{t('暂无项目')}</span>;
    }

    const displayItems = items.slice(0, 3);
    const remainingItems = items.slice(3);

    return (
      <div className='flex flex-wrap items-center gap-2'>
        {displayItems.map((item) => (
          <Badge
            key={item}
            variant='secondary'
            className='border-white/10 bg-white/10 text-white'
            style={{ backgroundColor: `${stringToColor(item)}20` }}
          >
            {item}
          </Badge>
        ))}
        {remainingItems.length > 0 && (
          <Popover>
            <PopoverTrigger asChild>
              <button type='button'>
                <Badge
                  variant='secondary'
                  className='border-white/10 bg-white/10 text-white'
                >
                  +{remainingItems.length}
                </Badge>
              </button>
            </PopoverTrigger>
            <PopoverContent className='max-w-xs border-white/10 bg-black text-white'>
              <div className='flex flex-wrap gap-2'>
                {remainingItems.map((item) => (
                  <Badge
                    key={item}
                    variant='secondary'
                    className='border-white/10 bg-white/10 text-white'
                    style={{ backgroundColor: `${stringToColor(item)}20` }}
                  >
                    {item}
                  </Badge>
                ))}
              </div>
            </PopoverContent>
          </Popover>
        )}
      </div>
    );
  };

  const columns = [
    {
      id: 'name',
      header: t('组名'),
      accessorKey: 'name',
      cell: ({ row }) => (
        <div className='flex items-center gap-2'>
          <span className='font-semibold text-white'>{row.original.name}</span>
          <Badge
            variant='secondary'
            className='border-white/10 bg-white/10 text-white'
          >
            {typeOptions.find((opt) => opt.value === row.original.type)
              ?.label ||
              row.original.type}
          </Badge>
        </div>
      ),
    },
    {
      id: 'description',
      header: t('描述'),
      accessorKey: 'description',
      cell: ({ row }) => (
        <span className='line-clamp-2 max-w-[220px] text-white/80'>
          {row.original.description || '-'}
        </span>
      ),
    },
    {
      id: 'items',
      header: t('项目内容'),
      accessorKey: 'items',
      cell: ({ row }) => {
        const { items, type } = row.original;
        try {
          if (type === 'endpoint') {
            const obj =
              typeof items === 'string'
                ? JSON.parse(items || '{}')
                : items || {};
            const keys = Object.keys(obj);
            return renderLimitedBadges(keys);
          }
          const itemsArray =
            typeof items === 'string' ? JSON.parse(items) : items;
          return renderLimitedBadges(
            Array.isArray(itemsArray) ? itemsArray : [],
          );
        } catch {
          return <span className='text-white/45'>{t('数据格式错误')}</span>;
        }
      },
    },
    {
      id: 'action',
      header: '',
      cell: ({ row }) => (
        <div className='flex gap-2'>
          <Button
            type='button'
            variant='secondary'
            size='sm'
            onClick={() => handleEdit(row.original)}
          >
            {t('编辑')}
          </Button>
          <Button
            type='button'
            variant='destructive'
            size='sm'
            onClick={() => setDeletingGroup(row.original)}
          >
            {t('删除')}
          </Button>
        </div>
      ),
    },
  ];

  useEffect(() => {
    if (visible) {
      loadGroups();
    }
  }, [visible]);

  return (
    <>
      <Dialog open={visible} onOpenChange={(open) => !open && onClose()}>
        <DialogContent
          className={
            isMobile
              ? 'max-w-[95vw] border-white/10 bg-black text-white'
              : 'max-w-[980px] border-white/10 bg-black text-white'
          }
        >
          <DialogHeader>
            <div className='flex items-center gap-2'>
              <Badge
                variant='secondary'
                className='border-white/10 bg-white/10 text-white'
              >
                {t('管理')}
              </Badge>
              <DialogTitle>{t('预填组管理')}</DialogTitle>
            </div>
          </DialogHeader>

          <Card className='border-white/10 bg-white/6 text-white'>
            <CardContent className='space-y-4 p-5'>
              <div className='flex items-center justify-between gap-4'>
                <div className='flex items-center gap-3'>
                  <div className='flex h-9 w-9 items-center justify-center rounded-full bg-white/10'>
                    <Layers className='h-4 w-4' />
                  </div>
                  <div>
                    <div className='text-lg font-medium'>{t('组列表')}</div>
                    <div className='text-xs text-white/45'>
                      {t('管理模型、标签、端点等预填组')}
                    </div>
                  </div>
                </div>
                <Button type='button' onClick={() => handleEdit()}>
                  <Plus className='mr-1 h-4 w-4' />
                  {t('新建组')}
                </Button>
              </div>

              {groups.length > 0 ? (
                <DataTable columns={columns} data={groups} loading={loading} />
              ) : (
                <div className='flex min-h-[180px] flex-col items-center justify-center rounded-2xl border border-white/10 bg-black/30 text-center'>
                  <div className='mb-3 flex h-[88px] w-[88px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
                    ○
                  </div>
                  <div className='font-medium'>{t('暂无预填组')}</div>
                  <div className='mt-1 text-sm text-white/45'>
                    {t('当前还没有创建任何预填组。')}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </DialogContent>
      </Dialog>

      <EditPrefillGroupModal
        visible={showEdit}
        onClose={closeEdit}
        editingGroup={editingGroup}
        onSuccess={handleEditSuccess}
      />

      <AlertDialog
        open={Boolean(deletingGroup)}
        onOpenChange={(open) => !open && setDeletingGroup(null)}
      >
        <AlertDialogContent className='border-white/10 bg-black text-white'>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('确定删除此组？')}</AlertDialogTitle>
            <AlertDialogDescription className='text-white/60'>
              {deletingGroup?.name}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter className='border-white/10 bg-transparent'>
            <AlertDialogCancel
              variant='secondary'
              onClick={() => setDeletingGroup(null)}
            >
              {t('取消')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                await deleteGroup(deletingGroup.id);
                setDeletingGroup(null);
              }}
            >
              {t('删除')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

export default PrefillGroupManagement;
