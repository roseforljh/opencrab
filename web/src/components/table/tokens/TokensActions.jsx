
import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { showError } from '../../../helpers';
import CopyTokensModal from './modals/CopyTokensModal';
import DeleteTokensModal from './modals/DeleteTokensModal';

const TokensActions = ({
  selectedKeys,
  setEditingToken,
  setShowEdit,
  batchCopyTokens,
  batchDeleteTokens,
  t,
}) => {
  const [showCopyModal, setShowCopyModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const handleCopySelectedTokens = () => {
    if (selectedKeys.length === 0) {
      showError(t('请至少选择一个令牌！'));
      return;
    }
    setShowCopyModal(true);
  };

  const handleDeleteSelectedTokens = () => {
    if (selectedKeys.length === 0) {
      showError(t('请至少选择一个令牌！'));
      return;
    }
    setShowDeleteModal(true);
  };

  const handleConfirmDelete = () => {
    batchDeleteTokens();
    setShowDeleteModal(false);
  };

  return (
    <>
      <div className='flex w-full flex-wrap gap-2'>
        <Button
          className='h-11 rounded-2xl border-0 bg-white/90 px-5 text-black hover:bg-white/80'
          onClick={() => {
            setEditingToken({
              id: undefined,
            });
            setShowEdit(true);
          }}
        >
          {t('新建令牌')}
        </Button>

        <Button
          variant='secondary'
          className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
          onClick={handleCopySelectedTokens}
        >
          {t('复制所选')}
        </Button>

        <Button
          variant='destructive'
          className='h-11 rounded-2xl border border-red-400/20 bg-red-500/10 px-5 text-red-100 hover:bg-red-500/20'
          onClick={handleDeleteSelectedTokens}
        >
          {t('删除所选')}
        </Button>
      </div>

      <CopyTokensModal
        visible={showCopyModal}
        onCancel={() => setShowCopyModal(false)}
        batchCopyTokens={batchCopyTokens}
        t={t}
      />

      <DeleteTokensModal
        visible={showDeleteModal}
        onCancel={() => setShowDeleteModal(false)}
        onConfirm={handleConfirmDelete}
        selectedKeys={selectedKeys}
        t={t}
      />
    </>
  );
};

export default TokensActions;
