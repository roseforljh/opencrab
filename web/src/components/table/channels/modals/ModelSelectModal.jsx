
import React, { useState, useEffect, useMemo } from 'react';
import { Search, Info, ChevronDown } from 'lucide-react';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { useTranslation } from 'react-i18next';
import { getModelCategories } from '../../../../helpers/render';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';

const ModelSelectModal = ({
  visible,
  models = [],
  selected = [],
  redirectModels = [],
  onConfirm,
  onCancel,
}) => {
  const { t } = useTranslation();

  const getModelName = (model) => {
    if (!model) return '';
    if (typeof model === 'string') return model;
    if (typeof model === 'object' && model.model_name) return model.model_name;
    return String(model ?? '');
  };

  const normalizedSelected = useMemo(
    () => (selected || []).map(getModelName),
    [selected],
  );

  const [checkedList, setCheckedList] = useState(normalizedSelected);
  const [keyword, setKeyword] = useState('');
  const [activeTab, setActiveTab] = useState('new');
  const [openSections, setOpenSections] = useState({});

  const isMobile = useIsMobile();
  const normalizeModelName = (model) =>
    typeof model === 'string' ? model.trim() : '';
  const normalizedRedirectModels = useMemo(
    () =>
      Array.from(
        new Set(
          (redirectModels || [])
            .map((model) => normalizeModelName(model))
            .filter(Boolean),
        ),
      ),
    [redirectModels],
  );
  const normalizedSelectedSet = useMemo(() => {
    const set = new Set();
    (selected || []).forEach((model) => {
      const normalized = normalizeModelName(model);
      if (normalized) {
        set.add(normalized);
      }
    });
    return set;
  }, [selected]);
  const classificationSet = useMemo(() => {
    const set = new Set(normalizedSelectedSet);
    normalizedRedirectModels.forEach((model) => set.add(model));
    return set;
  }, [normalizedSelectedSet, normalizedRedirectModels]);
  const redirectOnlySet = useMemo(() => {
    const set = new Set();
    normalizedRedirectModels.forEach((model) => {
      if (!normalizedSelectedSet.has(model)) {
        set.add(model);
      }
    });
    return set;
  }, [normalizedRedirectModels, normalizedSelectedSet]);

  const filteredModels = models.filter((m) =>
    String(m || '')
      .toLowerCase()
      .includes(keyword.toLowerCase()),
  );

  // 分类模型：新获取的模型和已有模型
  const isExistingModel = (model) =>
    classificationSet.has(normalizeModelName(model));
  const newModels = filteredModels.filter((model) => !isExistingModel(model));
  const existingModels = filteredModels.filter((model) =>
    isExistingModel(model),
  );

  // 同步外部选中值
  useEffect(() => {
    if (visible) {
      setCheckedList(normalizedSelected);
      setOpenSections({});
    }
  }, [visible, normalizedSelected]);

  // 当模型列表变化时，设置默认tab
  useEffect(() => {
    if (visible) {
      // 默认显示新获取模型tab，如果没有新模型则显示已有模型
      const hasNewModels = newModels.length > 0;
      setActiveTab(hasNewModels ? 'new' : 'existing');
    }
  }, [visible, newModels.length, selected]);

  const handleOk = () => {
    onConfirm && onConfirm(checkedList);
  };

  // 按厂商分类模型
  const categorizeModels = (models) => {
    const categories = getModelCategories(t);
    const categorizedModels = {};
    const uncategorizedModels = [];

    models.forEach((model) => {
      let foundCategory = false;
      for (const [key, category] of Object.entries(categories)) {
        if (key !== 'all' && category.filter({ model_name: model })) {
          if (!categorizedModels[key]) {
            categorizedModels[key] = {
              label: category.label,
              icon: category.icon,
              models: [],
            };
          }
          categorizedModels[key].models.push(model);
          foundCategory = true;
          break;
        }
      }
      if (!foundCategory) {
        uncategorizedModels.push(model);
      }
    });

    // 如果有未分类模型，添加到"其他"分类
    if (uncategorizedModels.length > 0) {
      categorizedModels['other'] = {
        label: t('其他'),
        icon: null,
        models: uncategorizedModels,
      };
    }

    return categorizedModels;
  };

  const newModelsByCategory = categorizeModels(newModels);
  const existingModelsByCategory = categorizeModels(existingModels);

  // Tab列表配置
  const tabList = [
    ...(newModels.length > 0
      ? [
          {
            tab: `${t('新获取的模型')} (${newModels.length})`,
            itemKey: 'new',
          },
        ]
      : []),
    ...(existingModels.length > 0
      ? [
          {
            tab: `${t('已有的模型')} (${existingModels.length})`,
            itemKey: 'existing',
          },
        ]
      : []),
  ];

  // 处理分类全选/取消全选
  const handleCategorySelectAll = (categoryModels, isChecked) => {
    let newCheckedList = [...checkedList];

    if (isChecked) {
      // 全选：添加该分类下所有未选中的模型
      categoryModels.forEach((model) => {
        if (!newCheckedList.includes(model)) {
          newCheckedList.push(model);
        }
      });
    } else {
      // 取消全选：移除该分类下所有已选中的模型
      newCheckedList = newCheckedList.filter(
        (model) => !categoryModels.includes(model),
      );
    }

    setCheckedList(newCheckedList);
  };

  // 检查分类是否全选
  const isCategoryAllSelected = (categoryModels) => {
    return (
      categoryModels.length > 0 &&
      categoryModels.every((model) => checkedList.includes(model))
    );
  };

  // 检查分类是否部分选中
  const isCategoryIndeterminate = (categoryModels) => {
    const selectedCount = categoryModels.filter((model) =>
      checkedList.includes(model),
    ).length;
    return selectedCount > 0 && selectedCount < categoryModels.length;
  };

  const renderModelsByCategory = (modelsByCategory, categoryKeyPrefix) => {
    const categoryEntries = Object.entries(modelsByCategory);
    if (categoryEntries.length === 0) return null;

    return (
      <div className='space-y-3'>
        {categoryEntries.map(([key, categoryData], index) => {
          const sectionKey = `${categoryKeyPrefix}_${index}`;
          const isOpen = Boolean(openSections[sectionKey]);
          const selectedCount = categoryData.models.filter((model) =>
            checkedList.includes(model),
          ).length;

          return (
            <div
              key={sectionKey}
              className='rounded-xl border border-white/10 bg-white/5'
            >
              <button
                type='button'
                className='flex w-full items-center justify-between gap-3 px-4 py-3 text-left'
                onClick={() =>
                  setOpenSections((prev) => ({
                    ...prev,
                    [sectionKey]: !prev[sectionKey],
                  }))
                }
              >
                <div className='flex items-center gap-3'>
                  <Checkbox
                    checked={isCategoryAllSelected(categoryData.models)}
                    indeterminate={isCategoryIndeterminate(categoryData.models)}
                    onCheckedChange={(checked) =>
                      handleCategorySelectAll(
                        categoryData.models,
                        Boolean(checked),
                      )
                    }
                    onClick={(e) => e.stopPropagation()}
                    aria-label={t('选择 {{name}} 分类', {
                      name: categoryData.label,
                    })}
                  />
                  <div className='flex items-center gap-2'>
                    {categoryData.icon}
                    <div>
                      <div className='font-medium text-white'>
                        {categoryData.label} ({categoryData.models.length})
                      </div>
                      <div className='text-xs text-white/45'>
                        {t('已选择 {{selected}} / {{total}}', {
                          selected: selectedCount,
                          total: categoryData.models.length,
                        })}
                      </div>
                    </div>
                  </div>
                </div>
                <ChevronDown
                  className={`h-4 w-4 shrink-0 transition-transform ${isOpen ? 'rotate-180' : ''}`}
                />
              </button>

              {isOpen && (
                <div className='grid grid-cols-1 gap-2 px-4 pb-4 sm:grid-cols-2'>
                  {categoryData.models.map((model) => {
                    const checked = checkedList.includes(model);
                    return (
                      <label
                        key={model}
                        className='my-1 flex items-center gap-2 text-sm text-white/85'
                      >
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(nextChecked) => {
                            setCheckedList((prev) => {
                              if (nextChecked) {
                                return prev.includes(model)
                                  ? prev
                                  : [...prev, model];
                              }
                              return prev.filter((item) => item !== model);
                            });
                          }}
                        />
                        <span className='flex items-center gap-2'>
                          <span>{model}</span>
                          {redirectOnlySet.has(normalizeModelName(model)) && (
                            <span
                              className='inline-flex items-center text-amber-400'
                              title={t('来自模型重定向，尚未加入模型列表')}
                            >
                              <Info className='h-4 w-4' />
                            </span>
                          )}
                        </span>
                      </label>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && onCancel?.()}>
      <DialogContent
        className={
          isMobile
            ? 'max-w-[95vw] border-white/10 bg-black text-white'
            : 'max-w-[960px] border-white/10 bg-black text-white'
        }
      >
        <DialogHeader>
          <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
            <DialogTitle>{t('选择模型')}</DialogTitle>
            <div className='flex flex-wrap gap-2'>
              {tabList.map((tab) => (
                <Button
                  key={tab.itemKey}
                  type='button'
                  variant={activeTab === tab.itemKey ? 'default' : 'secondary'}
                  size='sm'
                  onClick={() => setActiveTab(tab.itemKey)}
                >
                  {tab.tab}
                </Button>
              ))}
            </div>
          </div>
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
          {!models || models.length === 0 ? (
            <div className='flex items-center justify-center py-12 text-sm text-white/50'>
              {t('加载中...')}
            </div>
          ) : filteredModels.length === 0 ? (
            <div className='flex flex-col items-center justify-center rounded-2xl border border-white/10 bg-black/30 p-6 text-center'>
              <div className='mb-3 flex h-[120px] w-[120px] items-center justify-center rounded-[28px] border border-white/10 bg-black/50 text-3xl text-white/30'>
                ⌕
              </div>
              <div className='font-medium'>{t('暂无匹配模型')}</div>
              <div className='mt-1 text-sm text-white/45'>
                {t('请尝试供应商分类或缩短关键词后再试。')}
              </div>
            </div>
          ) : (
            <>
              {activeTab === 'new' && newModels.length > 0 && (
                <div>{renderModelsByCategory(newModelsByCategory, 'new')}</div>
              )}
              {activeTab === 'existing' && existingModels.length > 0 && (
                <div>
                  {renderModelsByCategory(existingModelsByCategory, 'existing')}
                </div>
              )}
            </>
          )}
        </div>

        <div className='flex items-center justify-between gap-3 text-sm text-white/60'>
          {(() => {
            const currentModels = activeTab === 'new' ? newModels : existingModels;
            const currentSelected = currentModels.filter((model) =>
              checkedList.includes(model),
            ).length;
            const isAllSelected =
              currentModels.length > 0 &&
              currentSelected === currentModels.length;
            const isIndeterminate =
              currentSelected > 0 && currentSelected < currentModels.length;

            return (
              <>
                <span>
                  {t('已选择 {{selected}} / {{total}}', {
                    selected: currentSelected,
                    total: currentModels.length,
                  })}
                </span>
                <div className='flex items-center gap-2'>
                  <Badge
                    variant='secondary'
                    className='border-white/10 bg-white/10 text-white/70'
                  >
                    {t('总计 {{count}}', { count: checkedList.length })}
                  </Badge>
                  <Checkbox
                    checked={isAllSelected}
                    indeterminate={isIndeterminate}
                    onCheckedChange={(checked) =>
                      handleCategorySelectAll(currentModels, Boolean(checked))
                    }
                    aria-label={t('全选当前结果')}
                  />
                </div>
              </>
            );
          })()}
        </div>

        <DialogFooter className='border-white/10 bg-transparent'>
          <Button type='button' variant='secondary' onClick={onCancel}>
            {t('取消')}
          </Button>
          <Button type='button' onClick={handleOk}>
            {t('确定')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ModelSelectModal;
