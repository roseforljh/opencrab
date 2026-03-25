import React from 'react';
import { getChannelsColumns } from '../ChannelsColumnDefs';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';

const ColumnSelectorModal = ({
  showColumnSelector,
  setShowColumnSelector,
  visibleColumns,
  handleColumnVisibilityChange,
  handleSelectAll,
  initDefaultColumns,
  COLUMN_KEYS,
  t,
  // Props needed for getChannelsColumns
  updateChannelBalance,
  manageChannel,
  manageTag,
  submitTagEdit,
  testChannel,
  setCurrentTestChannel,
  setShowModelTestModal,
  setEditingChannel,
  setShowEdit,
  setShowEditTag,
  setEditingTag,
  copySelectedChannel,
  refresh,
  activePage,
  channels,
}) => {
  // Get all columns for display in selector
  const allColumns = getChannelsColumns({
    t,
    COLUMN_KEYS,
    updateChannelBalance,
    manageChannel,
    manageTag,
    submitTagEdit,
    testChannel,
    setCurrentTestChannel,
    setShowModelTestModal,
    setEditingChannel,
    setShowEdit,
    setShowEditTag,
    setEditingTag,
    copySelectedChannel,
    refresh,
    activePage,
    channels,
  });

  return (
    <Dialog
      open={showColumnSelector}
      onOpenChange={(open) => !open && setShowColumnSelector(false)}
    >
      <DialogContent className='max-w-[720px] border-white/10 bg-black text-white'>
        <DialogHeader>
          <DialogTitle>{t('列设置')}</DialogTitle>
        </DialogHeader>
        <div className='mb-5'>
          <label className='flex items-center gap-2 text-sm'>
            <Checkbox
              checked={Object.values(visibleColumns).every((v) => v === true)}
              indeterminate={
                Object.values(visibleColumns).some((v) => v === true) &&
                !Object.values(visibleColumns).every((v) => v === true)
                  ? true
                  : undefined
              }
              onCheckedChange={(checked) => handleSelectAll(Boolean(checked))}
            />
            <span>{t('全选')}</span>
          </label>
        </div>
        <div className='flex max-h-96 flex-wrap overflow-y-auto rounded-lg border border-white/10 p-4'>
          {allColumns.map((column) => {
            if (!column.title) {
              return null;
            }

            return (
              <div key={column.key} className='mb-4 w-1/2 pr-2'>
                <label className='flex items-center gap-2 text-sm'>
                  <Checkbox
                    checked={!!visibleColumns[column.key]}
                    onCheckedChange={(checked) =>
                      handleColumnVisibilityChange(column.key, Boolean(checked))
                    }
                  />
                  <span>{column.title}</span>
                </label>
              </div>
            );
          })}
        </div>
        <DialogFooter className='border-white/10 bg-transparent'>
          <Button
            type='button'
            variant='secondary'
            onClick={initDefaultColumns}
          >
            {t('重置')}
          </Button>
          <Button
            type='button'
            variant='secondary'
            onClick={() => setShowColumnSelector(false)}
          >
            {t('取消')}
          </Button>
          <Button type='button' onClick={() => setShowColumnSelector(false)}>
            {t('确定')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ColumnSelectorModal;
