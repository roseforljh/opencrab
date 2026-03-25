import React, { useEffect, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { DataTable } from '../../../ui/data-table';
import { API, showError } from '../../../../helpers';
import { MODEL_TABLE_PAGE_SIZE } from '../../../../constants';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { Search } from 'lucide-react';

const MissingModelsModal = ({ visible, onClose, onConfigureModel, t }) => {
  const [loading, setLoading] = useState(false);
  const [missingModels, setMissingModels] = useState([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const isMobile = useIsMobile();

  const fetchMissing = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/models/missing');
      if (res.data.success) {
        setMissingModels(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch (_) {
      showError(t('获取未配置模型失败'));
    }
    setLoading(false);
  };

  useEffect(() => {
    if (visible) {
      fetchMissing();
      setSearchKeyword('');
      setCurrentPage(1);
    } else {
      setMissingModels([]);
    }
  }, [visible]);

  // 过滤和分页逻辑
  const filteredModels = missingModels.filter((model) =>
    model.toLowerCase().includes(searchKeyword.toLowerCase()),
  );

  const dataSource = (() => {
    const start = (currentPage - 1) * MODEL_TABLE_PAGE_SIZE;
    const end = start + MODEL_TABLE_PAGE_SIZE;
    return filteredModels.slice(start, end).map((model) => ({
      model,
      key: model,
    }));
  })();

  const columns = [
    {
      id: 'model',
      header: t('模型名称'),
      accessorKey: 'model',
      cell: ({ row }) => (
        <div className='flex items-center'>
          <span className='font-semibold text-white'>{row.original.model}</span>
        </div>
      ),
    },
    {
      id: 'operate',
      header: '',
      cell: ({ row }) => (
        <Button
          type='button'
          onClick={() => onConfigureModel(row.original.model)}
        >
          {t('配置')}
        </Button>
      ),
    },
  ];

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[900px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          <DialogTitle>
            <div className='flex flex-col gap-2 w-full'>
              <div className='flex items-center gap-2'>
                <span className='text-base font-semibold text-white'>
                  {t('未配置的模型列表')}
                </span>
                <span className='text-sm text-white/60'>
                  {t('共')} {missingModels.length} {t('个未配置模型')}
                </span>
              </div>
            </div>
          </DialogTitle>
        </DialogHeader>
        {missingModels.length === 0 && !loading ? (
          <div className='py-8 text-center text-white/60'>
            {t('当前所有模型都已配置完成。')}
          </div>
        ) : (
          <div className='missing-models-content'>
            <div className='flex items-center justify-end gap-2 w-full mb-4'>
              <Search className='h-4 w-4 text-white/40' />
              <Input
                placeholder={t('搜索模型...')}
                value={searchKeyword}
                onChange={(e) => {
                  setSearchKeyword(e.target.value);
                  setCurrentPage(1);
                }}
                className='w-full border-white/10 bg-white/6 text-white'
              />
            </div>

            {filteredModels.length > 0 ? (
              <DataTable
                columns={columns}
                data={dataSource}
                loading={loading}
              />
            ) : (
              <div className='py-6 text-center text-white/60'>
                {searchKeyword
                  ? t('可尝试缩短关键词，或检查命名是否准确。')
                  : t('当前所有模型都已配置完成。')}
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default MissingModelsModal;
