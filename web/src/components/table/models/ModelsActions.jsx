
import React, { useState } from 'react';
import MissingModelsModal from './modals/MissingModelsModal';
import PrefillGroupManagement from './modals/PrefillGroupManagement';
import EditPrefillGroupModal from './modals/EditPrefillGroupModal';
import { Button } from '@/components/ui/button';
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
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { showSuccess, showError, copy } from '../../../helpers';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import SelectionNotification from './components/SelectionNotification';
import UpstreamConflictModal from './modals/UpstreamConflictModal';
import SyncWizardModal from './modals/SyncWizardModal';

const ModelsActions = ({
  selectedKeys,
  setSelectedKeys,
  setEditingModel,
  setShowEdit,
  batchDeleteModels,
  syncing,
  previewing,
  syncUpstream,
  previewUpstreamDiff,
  applyUpstreamOverwrite,
  compactMode,
  setCompactMode,
  t,
}) => {
  // Modal states
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showMissingModal, setShowMissingModal] = useState(false);
  const [showGroupManagement, setShowGroupManagement] = useState(false);
  const [showAddPrefill, setShowAddPrefill] = useState(false);
  const [prefillInit, setPrefillInit] = useState({ id: undefined });
  const [showConflict, setShowConflict] = useState(false);
  const [conflicts, setConflicts] = useState([]);
  const [showSyncModal, setShowSyncModal] = useState(false);
  const [syncLocale, setSyncLocale] = useState('zh');

  const handleSyncUpstream = async (locale) => {
    // 先预览
    const data = await previewUpstreamDiff?.({ locale });
    const conflictItems = data?.conflicts || [];
    if (conflictItems.length > 0) {
      setConflicts(conflictItems);
      setShowConflict(true);
      return;
    }
    // 无冲突，直接同步缺失
    await syncUpstream?.({ locale });
  };

  // Handle delete selected models with confirmation
  const handleDeleteSelectedModels = () => {
    setShowDeleteModal(true);
  };

  // Handle delete confirmation
  const handleConfirmDelete = () => {
    batchDeleteModels();
    setShowDeleteModal(false);
  };

  // Handle clear selection
  const handleClearSelected = () => {
    setSelectedKeys([]);
  };

  // Handle add selected models to prefill group
  const handleCopyNames = async () => {
    const text = selectedKeys.map((m) => m.model_name).join(',');
    if (!text) return;
    const ok = await copy(text);
    if (ok) {
      showSuccess(t('已复制模型名称'));
    } else {
      showError(t('复制失败'));
    }
  };

  const handleAddToPrefill = () => {
    // Prepare initial data
    const items = selectedKeys.map((m) => m.model_name);
    setPrefillInit({ id: undefined, type: 'model', items });
    setShowAddPrefill(true);
  };

  return (
    <>
      <div className='order-2 flex w-full flex-wrap gap-2 md:order-1 md:w-auto'>
        <Button
          className='h-11 rounded-2xl border-0 bg-white/90 px-5 text-black hover:bg-white/80'
          onClick={() => {
            setEditingModel({
              id: undefined,
            });
            setShowEdit(true);
          }}
        >
          {t('添加模型')}
        </Button>

        <Button
          variant='secondary'
          className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
          onClick={() => setShowMissingModal(true)}
        >
          {t('未配置模型')}
        </Button>

        <Popover openDelay={100} closeDelay={100}>
          <PopoverTrigger asChild>
            <Button
              variant='secondary'
              className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
              disabled={syncing || previewing}
              onClick={() => {
                setSyncLocale('zh');
                setShowSyncModal(true);
              }}
            >
              {t('同步')}
            </Button>
          </PopoverTrigger>
          <PopoverContent className='max-w-[360px] border-white/10 bg-[#0b1220] text-white'>
            <div className='p-2'>
              <div className='text-[var(--semi-color-text-2)] text-sm'>
                {t(
                  '模型社区需要大家的共同维护，如发现数据有误或想贡献新的模型数据，请访问：',
                )}
              </div>
              <a
                href='https://github.com/basellm/llm-metadata'
                target='_blank'
                rel='noreferrer'
                className='text-blue-600 underline'
              >
                https://github.com/basellm/llm-metadata
              </a>
            </div>
          </PopoverContent>
        </Popover>

        <Button
          variant='secondary'
          className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
          onClick={() => setShowGroupManagement(true)}
        >
          {t('预填组管理')}
        </Button>

        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </div>

      <SelectionNotification
        selectedKeys={selectedKeys}
        t={t}
        onDelete={handleDeleteSelectedModels}
        onAddPrefill={handleAddToPrefill}
        onClear={handleClearSelected}
        onCopy={handleCopyNames}
      />

      <AlertDialog open={showDeleteModal} onOpenChange={setShowDeleteModal}>
        <AlertDialogContent className='border-white/10 bg-black text-white'>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('批量删除模型')}</AlertDialogTitle>
            <AlertDialogDescription className='text-white/60'>
              {t('确定要删除所选的 {{count}} 个模型吗？', {
                count: selectedKeys.length,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className='border-0 bg-white/5 hover:bg-white/10'>
              {t('取消')}
            </AlertDialogCancel>
            <AlertDialogAction
              className='bg-red-500 text-white hover:bg-red-600'
              onClick={handleConfirmDelete}
            >
              {t('确认删除')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <SyncWizardModal
        visible={showSyncModal}
        onClose={() => setShowSyncModal(false)}
        loading={syncing || previewing}
        t={t}
        onConfirm={async ({ option, locale }) => {
          setSyncLocale(locale);
          if (option === 'official') {
            await handleSyncUpstream(locale);
          }
          setShowSyncModal(false);
        }}
      />

      <MissingModelsModal
        visible={showMissingModal}
        onClose={() => setShowMissingModal(false)}
        onConfigureModel={(name) => {
          setEditingModel({ id: undefined, model_name: name });
          setShowEdit(true);
          setShowMissingModal(false);
        }}
        t={t}
      />

      <PrefillGroupManagement
        visible={showGroupManagement}
        onClose={() => setShowGroupManagement(false)}
      />

      <EditPrefillGroupModal
        visible={showAddPrefill}
        onClose={() => setShowAddPrefill(false)}
        editingGroup={prefillInit}
        onSuccess={() => setShowAddPrefill(false)}
      />

      <UpstreamConflictModal
        visible={showConflict}
        onClose={() => setShowConflict(false)}
        conflicts={conflicts}
        onSubmit={async (payload) => {
          return await applyUpstreamOverwrite?.({
            overwrite: payload,
            locale: syncLocale,
          });
        }}
        t={t}
        loading={syncing}
      />
    </>
  );
};

export default ModelsActions;
