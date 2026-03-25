
import React, { useEffect, useMemo, useState, useCallback } from 'react';
import { MousePointerClick, Search } from 'lucide-react';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { MODEL_TABLE_PAGE_SIZE } from '../../../../constants';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Badge } from '@/components/ui/badge';
import { DataTable } from '../../../ui/data-table';

const FIELD_LABELS = {
  description: '描述',
  icon: '图标',
  tags: '标签',
  vendor: '供应商',
  name_rule: '命名规则',
  status: '状态',
};
const FIELD_KEYS = Object.keys(FIELD_LABELS);

const UpstreamConflictModal = ({
  visible,
  onClose,
  conflicts = [],
  onSubmit,
  t,
  loading = false,
}) => {
  const [selections, setSelections] = useState({});
  const isMobile = useIsMobile();
  const [currentPage, setCurrentPage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState('');

  const formatValue = (v) => {
    if (v === null || v === undefined) return '-';
    if (typeof v === 'string') return v || '-';
    try {
      return JSON.stringify(v, null, 2);
    } catch (_) {
      return String(v);
    }
  };

  useEffect(() => {
    if (visible) {
      const init = {};
      conflicts.forEach((item) => {
        init[item.model_name] = new Set();
      });
      setSelections(init);
      setCurrentPage(1);
      setSearchKeyword('');
    } else {
      setSelections({});
    }
  }, [visible, conflicts]);

  const toggleField = useCallback((modelName, field, checked) => {
    setSelections((prev) => {
      const next = { ...prev };
      const set = new Set(next[modelName] || []);
      if (checked) set.add(field);
      else set.delete(field);
      next[modelName] = set;
      return next;
    });
  }, []);

  // 构造数据源与过滤后的数据源
  const dataSource = useMemo(
    () =>
      (conflicts || []).map((c) => ({
        key: c.model_name,
        model_name: c.model_name,
        fields: c.fields || [],
      })),
    [conflicts],
  );

  const filteredDataSource = useMemo(() => {
    const kw = (searchKeyword || '').toLowerCase();
    if (!kw) return dataSource;
    return dataSource.filter((item) =>
      (item.model_name || '').toLowerCase().includes(kw),
    );
  }, [dataSource, searchKeyword]);

  // 列头工具：当前过滤范围内可操作的行集合/勾选状态/批量设置
  const getPresentRowsForField = useCallback(
    (fieldKey) =>
      (filteredDataSource || []).filter((row) =>
        (row.fields || []).some((f) => f.field === fieldKey),
      ),
    [filteredDataSource],
  );

  const getHeaderState = useCallback(
    (fieldKey) => {
      const presentRows = getPresentRowsForField(fieldKey);
      const selectedCount = presentRows.filter((row) =>
        selections[row.model_name]?.has(fieldKey),
      ).length;
      const allCount = presentRows.length;
      return {
        headerChecked: allCount > 0 && selectedCount === allCount,
        headerIndeterminate: selectedCount > 0 && selectedCount < allCount,
        hasAny: allCount > 0,
      };
    },
    [getPresentRowsForField, selections],
  );

  const applyHeaderChange = useCallback(
    (fieldKey, checked) => {
      setSelections((prev) => {
        const next = { ...prev };
        getPresentRowsForField(fieldKey).forEach((row) => {
          const set = new Set(next[row.model_name] || []);
          if (checked) set.add(fieldKey);
          else set.delete(fieldKey);
          next[row.model_name] = set;
        });
        return next;
      });
    },
    [getPresentRowsForField],
  );

  const columns = useMemo(() => {
    const base = [
      {
        id: 'model_name',
        header: t('模型'),
        accessorKey: 'model_name',
        cell: ({ row }) => (
          <span className='font-semibold text-white'>
            {row.original.model_name}
          </span>
        ),
      },
    ];

    const cols = FIELD_KEYS.map((fieldKey) => {
      const rawLabel = FIELD_LABELS[fieldKey] || fieldKey;
      const label = t(rawLabel);

      const { headerChecked, headerIndeterminate, hasAny } =
        getHeaderState(fieldKey);
      if (!hasAny) return null;
      return {
        id: fieldKey,
        header: (
          <div className='flex items-center gap-2'>
            <Checkbox
              checked={headerChecked}
              indeterminate={headerIndeterminate ? true : undefined}
              onCheckedChange={(checked) =>
                applyHeaderChange(fieldKey, Boolean(checked))
              }
            />
            <span>{label}</span>
          </div>
        ),
        cell: ({ row }) => {
          const record = row.original;
          const f = (record.fields || []).find((x) => x.field === fieldKey);
          if (!f) return <span className='text-white/45'>-</span>;
          const checked = selections[record.model_name]?.has(fieldKey) || false;
          return (
            <div className='flex items-center gap-2'>
              <Checkbox
                checked={checked}
                onCheckedChange={(value) =>
                  toggleField(record.model_name, fieldKey, Boolean(value))
                }
              />
              <Popover>
                <PopoverTrigger asChild>
                  <button type='button'>
                    <Badge
                      variant='secondary'
                      className='gap-1 border-white/10 bg-white/10 text-white hover:bg-white/15'
                    >
                      <MousePointerClick size={14} />
                      {t('点击查看差异')}
                    </Badge>
                  </button>
                </PopoverTrigger>
                <PopoverContent
                  side='top'
                  className='max-w-[520px] border-white/10 bg-black text-white'
                >
                  <div className='space-y-3'>
                    <div>
                      <div className='mb-1 text-xs text-white/60'>
                        {t('本地')}
                      </div>
                      <pre className='m-0 whitespace-pre-wrap text-xs text-white/80'>
                        {formatValue(f.local)}
                      </pre>
                    </div>
                    <div>
                      <div className='mb-1 text-xs text-white/60'>
                        {t('官方')}
                      </div>
                      <pre className='m-0 whitespace-pre-wrap text-xs text-white/80'>
                        {formatValue(f.upstream)}
                      </pre>
                    </div>
                  </div>
                </PopoverContent>
              </Popover>
            </div>
          );
        },
      };
    });

    return [...base, ...cols.filter(Boolean)];
  }, [
    t,
    selections,
    getHeaderState,
    applyHeaderChange,
    toggleField,
  ]);

  const pagedDataSource = useMemo(() => {
    const start = (currentPage - 1) * MODEL_TABLE_PAGE_SIZE;
    const end = start + MODEL_TABLE_PAGE_SIZE;
    return filteredDataSource.slice(start, end);
  }, [filteredDataSource, currentPage]);

  const handleOk = async () => {
    const payload = Object.entries(selections)
      .map(([modelName, set]) => ({
        model_name: modelName,
        fields: Array.from(set || []),
      }))
      .filter((x) => x.fields.length > 0);

    const ok = await onSubmit?.(payload);
    if (ok) onClose?.();
  };

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onClose?.()}>
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[1000px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          <DialogTitle>{t('选择要覆盖的冲突项')}</DialogTitle>
        </DialogHeader>

        {dataSource.length === 0 ? (
          <div className='py-6 text-center text-white/60'>{t('无冲突项')}</div>
        ) : (
          <>
            <div className='text-sm text-white/60'>
              {t('仅会覆盖你勾选的字段，未勾选的字段保持本地不变。')}
            </div>
            <div className='flex items-center justify-end gap-2'>
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

            {filteredDataSource.length > 0 ? (
              <>
                <DataTable
                  columns={columns}
                  data={pagedDataSource}
                  loading={loading}
                />
                {filteredDataSource.length > MODEL_TABLE_PAGE_SIZE && (
                  <div className='mt-4 flex justify-end gap-2'>
                    <Button
                      type='button'
                      variant='secondary'
                      onClick={() =>
                        setCurrentPage((page) => Math.max(1, page - 1))
                      }
                      disabled={currentPage === 1}
                    >
                      {t('上一页')}
                    </Button>
                    <Button
                      type='button'
                      variant='secondary'
                      onClick={() =>
                        setCurrentPage((page) =>
                          page * MODEL_TABLE_PAGE_SIZE < filteredDataSource.length
                            ? page + 1
                            : page,
                        )
                      }
                      disabled={
                        currentPage * MODEL_TABLE_PAGE_SIZE >=
                        filteredDataSource.length
                      }
                    >
                      {t('下一页')}
                    </Button>
                  </div>
                )}
              </>
            ) : (
              <div className='py-6 text-center text-white/60'>
                {searchKeyword ? t('未找到匹配的模型') : t('无冲突项')}
              </div>
            )}
          </>
        )}

        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={onClose}>
            {t('取消')}
          </Button>
          <Button type='button' onClick={handleOk} disabled={loading}>
            {t('应用覆盖')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default UpstreamConflictModal;
