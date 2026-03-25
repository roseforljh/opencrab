import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { ChevronDown, Search } from 'lucide-react';
import { getModelCategories } from '../../../../helpers/render';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';

const SingleModelSelectModal = ({
  visible,
  models = [],
  selected = '',
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();

  const normalizeModelName = (model) => String(model ?? '').trim();
  const normalizedModels = useMemo(() => {
    const list = Array.isArray(models) ? models : [];
    return Array.from(new Set(list.map(normalizeModelName).filter(Boolean)));
  }, [models]);

  const [keyword, setKeyword] = useState('');
  const [selectedModel, setSelectedModel] = useState('');
  const [openSections, setOpenSections] = useState({});

  useEffect(() => {
    if (visible) {
      setKeyword('');
      setSelectedModel(normalizeModelName(selected));
      setOpenSections({});
    }
  }, [visible, selected]);

  const filteredModels = useMemo(() => {
    const lower = keyword.trim().toLowerCase();
    if (!lower) return normalizedModels;
    return normalizedModels.filter((m) => m.toLowerCase().includes(lower));
  }, [normalizedModels, keyword]);

  const modelsByCategory = useMemo(() => {
    const categories = getModelCategories(t);
    const categorized = {};
    const uncategorized = [];

    filteredModels.forEach((model) => {
      let foundCategory = false;
      for (const [key, category] of Object.entries(categories)) {
        if (key !== 'all' && category.filter({ model_name: model })) {
          if (!categorized[key]) {
            categorized[key] = {
              label: category.label,
              icon: category.icon,
              models: [],
            };
          }
          categorized[key].models.push(model);
          foundCategory = true;
          break;
        }
      }
      if (!foundCategory) {
        uncategorized.push(model);
      }
    });

    if (uncategorized.length > 0) {
      categorized.other = {
        label: t('其他'),
        icon: null,
        models: uncategorized,
      };
    }

    return categorized;
  }, [filteredModels, t]);

  const categoryEntries = useMemo(
    () => Object.entries(modelsByCategory),
    [modelsByCategory],
  );

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onCancel?.()}>
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[900px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          <DialogTitle>{t('选择模型')}</DialogTitle>
        </DialogHeader>

        <div className='relative'>
          <Search className='pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-white/40' />
          <Input
            placeholder={t('搜索模型')}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className='border-white/10 bg-white/6 pl-9 text-white'
          />
        </div>

        <div className='max-h-[400px] overflow-y-auto pr-2'>
          {filteredModels.length === 0 ? (
            <div className='flex flex-col items-center justify-center rounded-2xl border border-white/10 bg-black/30 p-6 text-center'>
              <div className='mb-3 flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
                ⌕
              </div>
              <div className='font-medium'>{t('暂无匹配模型')}</div>
              <div className='mt-1 text-sm text-white/45'>
                {t('请尝试缩短关键词，或检查模型名称是否准确。')}
              </div>
            </div>
          ) : (
            <div className='space-y-3'>
              {categoryEntries.map(([key, categoryData], index) => {
                const sectionKey = `${key}_${index}`;
                const isOpen = Boolean(openSections[sectionKey]);
                return (
                  <div
                    key={sectionKey}
                    className='rounded-xl border border-white/10 bg-white/5'
                  >
                    <button
                      type='button'
                      className='flex w-full items-center justify-between gap-2 px-4 py-3 text-left'
                      onClick={() =>
                        setOpenSections((prev) => ({
                          ...prev,
                          [sectionKey]: !prev[sectionKey],
                        }))
                      }
                    >
                      <span className='flex items-center gap-2'>
                        {categoryData.icon}
                        <span>
                          {categoryData.label} ({categoryData.models.length})
                        </span>
                      </span>
                      <ChevronDown
                        className={`h-4 w-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
                      />
                    </button>
                    {isOpen && (
                      <div className='grid grid-cols-2 gap-x-4 px-4 pb-4'>
                        {categoryData.models.map((model) => (
                          <label
                            key={model}
                            className='my-1 flex items-center gap-2 text-sm'
                          >
                            <Checkbox
                              checked={selectedModel === model}
                              onCheckedChange={(checked) =>
                                setSelectedModel(checked ? model : '')
                              }
                            />
                            <span>{model}</span>
                          </label>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={onCancel}>
            {t('取消')}
          </Button>
          <Button
            type='button'
            onClick={() => onConfirm?.(selectedModel)}
            disabled={!selectedModel}
          >
            {t('确定')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default SingleModelSelectModal;
