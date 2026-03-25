import React from 'react';
import { Search, Info } from 'lucide-react';
import { copy, showError, showInfo, showSuccess } from '../../../../helpers';
import { MODEL_TABLE_PAGE_SIZE } from '../../../../constants';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { DataTable } from '../../../ui/data-table';

const ModelTestModal = ({
  showModelTestModal,
  currentTestChannel,
  handleCloseModal,
  isBatchTesting,
  batchTestModels,
  modelSearchKeyword,
  setModelSearchKeyword,
  selectedModelKeys,
  setSelectedModelKeys,
  modelTestResults,
  testingModels,
  testChannel,
  modelTablePage,
  setModelTablePage,
  selectedEndpointType,
  setSelectedEndpointType,
  isStreamTest,
  setIsStreamTest,
  allSelectingRef,
  isMobile,
  t,
}) => {
  const hasChannel = Boolean(currentTestChannel);
  const streamToggleDisabled = [
    'embeddings',
    'image-generation',
    'jina-rerank',
    'openai-response-compact',
  ].includes(selectedEndpointType);

  React.useEffect(() => {
    if (streamToggleDisabled && isStreamTest) {
      setIsStreamTest(false);
    }
  }, [streamToggleDisabled, isStreamTest, setIsStreamTest]);

  const filteredModels = hasChannel
    ? currentTestChannel.models
        .split(',')
        .filter((model) =>
          model.toLowerCase().includes(modelSearchKeyword.toLowerCase()),
        )
    : [];

  const endpointTypeOptions = [
    { value: '', label: t('自动检测') },
    { value: 'openai', label: 'OpenAI (/v1/chat/completions)' },
    { value: 'openai-response', label: 'OpenAI Response (/v1/responses)' },
    {
      value: 'openai-response-compact',
      label: 'OpenAI Response Compaction (/v1/responses/compact)',
    },
    { value: 'anthropic', label: 'Anthropic (/v1/messages)' },
    {
      value: 'gemini',
      label: 'Gemini (/v1beta/models/{model}:generateContent)',
    },
    { value: 'jina-rerank', label: 'Jina Rerank (/v1/rerank)' },
    {
      value: 'image-generation',
      label: t('图像生成') + ' (/v1/images/generations)',
    },
    { value: 'embeddings', label: 'Embeddings (/v1/embeddings)' },
  ];

  const handleCopySelected = () => {
    if (selectedModelKeys.length === 0) {
      showError(t('请先选择模型！'));
      return;
    }
    copy(selectedModelKeys.join(',')).then((ok) => {
      if (ok) {
        showSuccess(
          t('已复制 ${count} 个模型').replace(
            '${count}',
            selectedModelKeys.length,
          ),
        );
      } else {
        showError(t('复制失败，请手动复制'));
      }
    });
  };

  const handleSelectSuccess = () => {
    if (!currentTestChannel) return;
    const successKeys = currentTestChannel.models
      .split(',')
      .filter((m) => m.toLowerCase().includes(modelSearchKeyword.toLowerCase()))
      .filter((m) => {
        const result = modelTestResults[`${currentTestChannel.id}-${m}`];
        return result && result.success;
      });
    if (successKeys.length === 0) {
      showInfo(t('暂无成功模型'));
    }
    setSelectedModelKeys(successKeys);
  };

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
      id: 'status',
      header: t('状态'),
      cell: ({ row }) => {
        const record = row.original;
        const testResult =
          modelTestResults[`${currentTestChannel.id}-${record.model}`];
        const isTesting = testingModels.has(record.model);

        if (isTesting) {
          return (
            <Badge className='border-blue-500/20 bg-blue-500/15 text-blue-200'>
              {t('测试中')}
            </Badge>
          );
        }

        if (!testResult) {
          return (
            <Badge className='border-white/10 bg-white/10 text-white/70'>
              {t('未开始')}
            </Badge>
          );
        }

        return (
          <div className='flex items-center gap-2'>
            <Badge
              className={
                testResult.success
                  ? 'border-green-500/20 bg-green-500/15 text-green-200'
                  : 'border-red-500/20 bg-red-500/15 text-red-200'
              }
            >
              {testResult.success ? t('成功') : t('失败')}
            </Badge>
            {testResult.success && (
              <span className='text-sm text-white/45'>
                {t('请求时长: ${time}s').replace(
                  '${time}',
                  testResult.time.toFixed(2),
                )}
              </span>
            )}
          </div>
        );
      },
    },
    {
      id: 'operate',
      header: '',
      cell: ({ row }) => {
        const record = row.original;
        const isTesting = testingModels.has(record.model);
        return (
          <Button
            type='button'
            variant='secondary'
            size='sm'
            onClick={() =>
              testChannel(
                currentTestChannel,
                record.model,
                selectedEndpointType,
                isStreamTest,
              )
            }
            loading={isTesting}
          >
            {t('测试')}
          </Button>
        );
      },
    },
  ];

  const dataSource = (() => {
    if (!hasChannel) return [];
    const start = (modelTablePage - 1) * MODEL_TABLE_PAGE_SIZE;
    const end = start + MODEL_TABLE_PAGE_SIZE;
    return filteredModels.slice(start, end).map((model) => ({
      model,
      key: model,
    }));
  })();

  return (
    <Dialog
      open={showModelTestModal}
      onOpenChange={(open) => !open && handleCloseModal?.()}
    >
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[980px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          {hasChannel ? (
            <div className='flex flex-col gap-2'>
              <DialogTitle>
                {currentTestChannel.name} {t('渠道的模型测试')}
              </DialogTitle>
              <div className='text-sm text-white/45'>
                {t('共')} {currentTestChannel.models.split(',').length}{' '}
                {t('个模型')}
              </div>
            </div>
          ) : (
            <DialogTitle>{t('模型测试')}</DialogTitle>
          )}
        </DialogHeader>

        {hasChannel && (
          <div className='model-test-scroll'>
            <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
              <div className='flex items-center gap-2 flex-1 min-w-0'>
                <span className='shrink-0 font-medium'>{t('端点类型')}:</span>
                <Select
                  value={selectedEndpointType}
                  onValueChange={setSelectedEndpointType}
                >
                  <SelectTrigger className='w-full min-w-0 border-white/10 bg-white/6 text-white'>
                    <SelectValue placeholder={t('选择端点类型')} />
                  </SelectTrigger>
                  <SelectContent className='border-white/10 bg-black text-white'>
                    {endpointTypeOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className='flex items-center justify-between sm:justify-end gap-2 shrink-0'>
                <span className='shrink-0 font-medium'>{t('流式')}:</span>
                <Switch
                  checked={isStreamTest}
                  onCheckedChange={setIsStreamTest}
                  size='small'
                  disabled={streamToggleDisabled}
                  aria-label={t('流式')}
                />
              </div>
            </div>

            <div className='mb-2 rounded-xl border border-blue-500/20 bg-blue-500/10 p-3 text-sm text-blue-100'>
              <div className='flex gap-2'>
                <Info className='mt-0.5 h-4 w-4 shrink-0' />
                <span>
                  {t(
                    '说明：本页测试为非流式请求；若渠道仅支持流式返回，可能出现测试失败，请以实际使用为准。',
                  )}
                </span>
              </div>
            </div>

            <div className='flex flex-col sm:flex-row sm:items-center gap-2 w-full mb-2'>
              <div className='relative w-full sm:flex-1'>
                <Search className='pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-white/40' />
                <Input
                  placeholder={t('搜索模型...')}
                  value={modelSearchKeyword}
                  onChange={(e) => {
                    setModelSearchKeyword(e.target.value);
                    setModelTablePage(1);
                  }}
                  className='w-full border-white/10 bg-white/6 pl-9 text-white'
                />
              </div>

              <div className='flex items-center justify-end gap-2'>
                <Button
                  type='button'
                  variant='secondary'
                  onClick={handleCopySelected}
                >
                  {t('复制已选')}
                </Button>
                <Button
                  type='button'
                  variant='secondary'
                  onClick={handleSelectSuccess}
                >
                  {t('选择成功')}
                </Button>
              </div>
            </div>

            <DataTable
              columns={columns}
              data={dataSource}
              emptyMessage={t('暂无匹配模型')}
            />
            <div className='mt-3 flex items-center justify-between'>
              <label className='flex items-center gap-2 text-sm'>
                <input
                  type='checkbox'
                  checked={
                    filteredModels.length > 0 &&
                    filteredModels.every((model) =>
                      selectedModelKeys.includes(model),
                    )
                  }
                  onChange={(e) => {
                    allSelectingRef.current = true;
                    setSelectedModelKeys(
                      e.target.checked ? filteredModels : [],
                    );
                  }}
                />
                <span>{t('全选当前结果')}</span>
              </label>
              <div className='flex gap-2'>
                <Button
                  type='button'
                  variant='secondary'
                  size='sm'
                  onClick={() =>
                    setModelTablePage(Math.max(1, modelTablePage - 1))
                  }
                  disabled={modelTablePage === 1}
                >
                  {t('上一页')}
                </Button>
                <Button
                  type='button'
                  variant='secondary'
                  size='sm'
                  onClick={() =>
                    setModelTablePage((page) =>
                      page * MODEL_TABLE_PAGE_SIZE < filteredModels.length
                        ? page + 1
                        : page,
                    )
                  }
                  disabled={
                    modelTablePage * MODEL_TABLE_PAGE_SIZE >=
                    filteredModels.length
                  }
                >
                  {t('下一页')}
                </Button>
              </div>
            </div>
          </div>
        )}

        <DialogFooter className='border-white/10 bg-transparent'>
          {isBatchTesting ? (
            <Button
              type='button'
              variant='destructive'
              onClick={handleCloseModal}
            >
              {t('停止测试')}
            </Button>
          ) : (
            <Button
              type='button'
              variant='secondary'
              onClick={handleCloseModal}
            >
              {t('取消')}
            </Button>
          )}
          <Button
            type='button'
            onClick={batchTestModels}
            disabled={isBatchTesting}
          >
            {isBatchTesting
              ? t('测试中...')
              : t('批量测试${count}个模型').replace(
                  '${count}',
                  filteredModels.length,
                )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ModelTestModal;
