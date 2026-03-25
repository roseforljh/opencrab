import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Search } from 'lucide-react';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
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

const normalizeModels = (models = []) =>
  Array.from(
    new Set(
      (models || []).map((model) => String(model || '').trim()).filter(Boolean),
    ),
  );

const filterByKeyword = (models = [], keyword = '') => {
  const normalizedKeyword = String(keyword || '')
    .trim()
    .toLowerCase();
  if (!normalizedKeyword) {
    return models;
  }
  return models.filter((model) =>
    String(model).toLowerCase().includes(normalizedKeyword),
  );
};

const ChannelUpstreamUpdateModal = ({
  visible,
  addModels = [],
  removeModels = [],
  preferredTab = 'add',
  confirmLoading = false,
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();

  const normalizedAddModels = useMemo(
    () => normalizeModels(addModels),
    [addModels],
  );
  const normalizedRemoveModels = useMemo(
    () => normalizeModels(removeModels),
    [removeModels],
  );

  const [selectedAddModels, setSelectedAddModels] = useState([]);
  const [selectedRemoveModels, setSelectedRemoveModels] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [activeTab, setActiveTab] = useState('add');
  const [pendingPartialConfirm, setPendingPartialConfirm] = useState(null);

  const addTabEnabled = normalizedAddModels.length > 0;
  const removeTabEnabled = normalizedRemoveModels.length > 0;
  const filteredAddModels = useMemo(
    () => filterByKeyword(normalizedAddModels, keyword),
    [normalizedAddModels, keyword],
  );
  const filteredRemoveModels = useMemo(
    () => filterByKeyword(normalizedRemoveModels, keyword),
    [normalizedRemoveModels, keyword],
  );

  useEffect(() => {
    if (!visible) {
      return;
    }
    setSelectedAddModels([]);
    setSelectedRemoveModels([]);
    setKeyword('');
    setPendingPartialConfirm(null);
    const normalizedPreferredTab = preferredTab === 'remove' ? 'remove' : 'add';
    if (normalizedPreferredTab === 'remove' && removeTabEnabled) {
      setActiveTab('remove');
      return;
    }
    if (normalizedPreferredTab === 'add' && addTabEnabled) {
      setActiveTab('add');
      return;
    }
    setActiveTab(addTabEnabled ? 'add' : 'remove');
  }, [visible, addTabEnabled, removeTabEnabled, preferredTab]);

  const currentModels =
    activeTab === 'add' ? filteredAddModels : filteredRemoveModels;
  const currentSelectedModels =
    activeTab === 'add' ? selectedAddModels : selectedRemoveModels;
  const currentSetSelectedModels =
    activeTab === 'add' ? setSelectedAddModels : setSelectedRemoveModels;
  const selectedAddCount = selectedAddModels.length;
  const selectedRemoveCount = selectedRemoveModels.length;
  const checkedCount = currentModels.filter((model) =>
    currentSelectedModels.includes(model),
  ).length;
  const isAllChecked =
    currentModels.length > 0 && checkedCount === currentModels.length;
  const isIndeterminate =
    checkedCount > 0 && checkedCount < currentModels.length;

  const handleToggleAllCurrent = (checked) => {
    if (checked) {
      const merged = normalizeModels([
        ...currentSelectedModels,
        ...currentModels,
      ]);
      currentSetSelectedModels(merged);
      return;
    }
    const currentSet = new Set(currentModels);
    currentSetSelectedModels(
      currentSelectedModels.filter((model) => !currentSet.has(model)),
    );
  };

  const submitSelectedChanges = () => {
    onConfirm?.({
      addModels: selectedAddModels,
      removeModels: selectedRemoveModels,
    });
  };

  const handleSubmit = () => {
    const hasAnySelected = selectedAddCount > 0 || selectedRemoveCount > 0;
    if (!hasAnySelected) {
      submitSelectedChanges();
      return;
    }

    const hasBothPending = addTabEnabled && removeTabEnabled;
    const hasUnselectedAdd = addTabEnabled && selectedAddCount === 0;
    const hasUnselectedRemove = removeTabEnabled && selectedRemoveCount === 0;
    if (hasBothPending && (hasUnselectedAdd || hasUnselectedRemove)) {
      const missingTab = hasUnselectedAdd ? 'add' : 'remove';
      const missingType = hasUnselectedAdd ? t('新增') : t('删除');
      const missingCount = hasUnselectedAdd
        ? normalizedAddModels.length
        : normalizedRemoveModels.length;
      setActiveTab(missingTab);
      setPendingPartialConfirm({
        missingType,
        missingCount,
      });
      return;
    }

    submitSelectedChanges();
  };

  return (
    <>
      <Dialog open={visible} onOpenChange={(open) => !open && onCancel?.()}>
        <DialogContent
          className={
            isMobile
              ? 'max-w-[95vw] border-white/10 bg-black text-white'
              : 'max-w-[720px] border-white/10 bg-black text-white'
          }
        >
          <DialogHeader>
            <DialogTitle>{t('处理上游模型更新')}</DialogTitle>
          </DialogHeader>
          <div className='flex flex-col gap-3'>
            <div className='text-sm text-white/60'>
              {t(
                '可勾选需要执行的变更：新增会加入渠道模型列表，删除会从渠道模型列表移除。',
              )}
            </div>

            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList className='grid w-full grid-cols-2 bg-white/5'>
                <TabsTrigger value='add' disabled={!addTabEnabled}>
                  {t('新增模型')} ({selectedAddCount}/
                  {normalizedAddModels.length})
                </TabsTrigger>
                <TabsTrigger value='remove' disabled={!removeTabEnabled}>
                  {t('删除模型')} ({selectedRemoveCount}/
                  {normalizedRemoveModels.length})
                </TabsTrigger>
              </TabsList>
            </Tabs>

            <div className='flex items-center gap-3 text-xs text-white/45'>
              <span>
                {t('新增已选 {{selected}} / {{total}}', {
                  selected: selectedAddCount,
                  total: normalizedAddModels.length,
                })}
              </span>
              <span>
                {t('删除已选 {{selected}} / {{total}}', {
                  selected: selectedRemoveCount,
                  total: normalizedRemoveModels.length,
                })}
              </span>
            </div>

            <div className='relative'>
              <Search className='pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-white/40' />
              <Input
                placeholder={t('搜索模型')}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className='border-white/10 bg-white/6 pl-9 text-white'
              />
            </div>

            <div className='max-h-[320px] overflow-y-auto pr-2'>
              {currentModels.length === 0 ? (
                <div className='flex flex-col items-center justify-center rounded-2xl border border-white/10 bg-black/30 p-6 text-center'>
                  <div className='mb-3 flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
                    ⌕
                  </div>
                  <div className='font-medium'>{t('暂无匹配模型')}</div>
                  <div className='mt-1 text-sm text-white/45'>
                    {t('可尝试更换关键词，或切换新增 / 删除页签。')}
                  </div>
                </div>
              ) : (
                <div className='grid grid-cols-1 gap-x-4 md:grid-cols-2'>
                  {currentModels.map((model) => (
                    <label
                      key={`${activeTab}:${model}`}
                      className='my-1 flex items-center gap-2 text-sm'
                    >
                      <Checkbox
                        checked={currentSelectedModels.includes(model)}
                        onCheckedChange={(checked) => {
                          if (checked) {
                            currentSetSelectedModels(
                              normalizeModels([
                                ...currentSelectedModels,
                                model,
                              ]),
                            );
                          } else {
                            currentSetSelectedModels(
                              currentSelectedModels.filter(
                                (item) => item !== model,
                              ),
                            );
                          }
                        }}
                      />
                      <span>{model}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>

            <div className='flex items-center justify-end gap-2'>
              <span className='text-xs text-white/45'>
                {t('已选择 {{selected}} / {{total}}', {
                  selected: checkedCount,
                  total: currentModels.length,
                })}
              </span>
              <label className='flex items-center gap-2 text-sm'>
                <span>{t('全选')}</span>
                <Checkbox
                  checked={isAllChecked}
                  indeterminate={isIndeterminate ? true : undefined}
                  aria-label={t('全选当前列表模型')}
                  onCheckedChange={(checked) =>
                    handleToggleAllCurrent(Boolean(checked))
                  }
                />
              </label>
            </div>
          </div>

          <DialogFooter className='border-white/10 bg-transparent'>
            <Button type='button' variant='secondary' onClick={onCancel}>
              {t('取消')}
            </Button>
            <Button
              type='button'
              onClick={handleSubmit}
              disabled={confirmLoading}
            >
              {t('确定')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(pendingPartialConfirm)}
        onOpenChange={(open) => !open && setPendingPartialConfirm(null)}
      >
        <AlertDialogContent className='border-white/10 bg-black text-white'>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('仍有未处理项')}</AlertDialogTitle>
            <AlertDialogDescription className='text-white/60'>
              {pendingPartialConfirm &&
                t(
                  '你还没有处理{{type}}模型（{{count}}个）。是否仅提交当前已勾选内容？',
                  {
                    type: pendingPartialConfirm.missingType,
                    count: pendingPartialConfirm.missingCount,
                  },
                )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter className='border-white/10 bg-transparent'>
            <AlertDialogCancel
              variant='secondary'
              onClick={() => setPendingPartialConfirm(null)}
            >
              {pendingPartialConfirm
                ? t('去处理{{type}}', {
                    type: pendingPartialConfirm.missingType,
                  })
                : t('取消')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setPendingPartialConfirm(null);
                submitSelectedChanges();
              }}
            >
              {t('仅提交已勾选')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

export default ChannelUpstreamUpdateModal;
