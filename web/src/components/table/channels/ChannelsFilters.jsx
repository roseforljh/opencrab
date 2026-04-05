
import React, { useEffect, useMemo, useState } from 'react';
import { Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

const ALL_GROUP_VALUE = '__all_groups__';

const ChannelsFilters = ({
  setEditingChannel,
  setShowEdit,
  refresh,
  setShowColumnSelector,
  formInitValues,
  setFormApi,
  searchChannels,
  enableTagMode,
  formApi,
  groupOptions,
  loading,
  searching,
  t,
}) => {
  const [values, setValues] = useState(formInitValues);

  useEffect(() => {
    setFormApi({
      getValues: () => values,
      reset: () => setValues(formInitValues),
    });
  }, [formInitValues, setFormApi, values]);

  const searchableGroups = useMemo(
    () => [{ label: t('选择分组'), value: ALL_GROUP_VALUE }, ...groupOptions],
    [groupOptions, t],
  );

  return (
    <div className='flex flex-col gap-4 w-full'>
      <div>
        <div className='text-sm font-semibold text-white'>
          {t('搜索与筛选')}
        </div>
        <div className='mt-1 text-xs text-white/45'>
          {t('按名称、密钥、地址、模型或分组快速定位异常渠道。')}
        </div>
      </div>

      <div className='flex flex-wrap gap-2'>
        <Button
          className='h-11 rounded-2xl border-0 bg-white/90 px-5 text-black hover:bg-white/80'
          onClick={() => {
            setEditingChannel({
              id: undefined,
            });
            setShowEdit(true);
          }}
        >
          {t('添加渠道')}
        </Button>

        <Button
          variant='secondary'
          className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
          onClick={refresh}
        >
          {t('刷新列表')}
        </Button>

        <Button
          variant='secondary'
          className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
          onClick={() => setShowColumnSelector(true)}
        >
          {t('列设置')}
        </Button>
      </div>

      <form
        className='w-full'
        onSubmit={(e) => {
          e.preventDefault();
          searchChannels(enableTagMode);
        }}
      >
        <div className='grid grid-cols-1 gap-2 lg:grid-cols-4'>
          <div className='relative'>
            <Search className='pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/40' />
            <Input
              value={values.searchKeyword || ''}
              onChange={(e) =>
                setValues((prev) => ({
                  ...prev,
                  searchKeyword: e.target.value,
                }))
              }
              placeholder={t('渠道ID，名称，密钥，API地址')}
              className='h-11 rounded-2xl border-white/10 bg-white/6 pl-10 text-white placeholder:text-white/35'
            />
          </div>
          <div className='relative'>
            <Search className='pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/40' />
            <Input
              value={values.searchModel || ''}
              onChange={(e) =>
                setValues((prev) => ({
                  ...prev,
                  searchModel: e.target.value,
                }))
              }
              placeholder={t('模型关键字')}
              className='h-11 rounded-2xl border-white/10 bg-white/6 pl-10 text-white placeholder:text-white/35'
            />
          </div>
          <Select
            value={values.searchGroup || ALL_GROUP_VALUE}
            onValueChange={(value) => {
              setValues((prev) => ({
                ...prev,
                searchGroup: value === ALL_GROUP_VALUE ? '' : value,
              }));
              setTimeout(() => {
                searchChannels(enableTagMode);
              }, 0);
            }}
          >
            <SelectTrigger className='h-11 w-full rounded-2xl border-white/10 bg-white/6 text-white'>
              <SelectValue placeholder={t('选择分组')} />
            </SelectTrigger>
            <SelectContent className='bg-[#0b1220] border-white/10 text-white'>
              {searchableGroups.map((option) => (
                <SelectItem
                  key={option.value || 'all'}
                  value={option.value || ALL_GROUP_VALUE}
                >
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className='flex flex-wrap gap-2'>
            <Button
              type='submit'
              disabled={loading || searching}
              className='h-11 rounded-2xl border-0 bg-white/90 px-5 text-black hover:bg-white/80'
            >
              {t('查询渠道')}
            </Button>
            <Button
              type='button'
              variant='secondary'
              className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
              onClick={() => {
                if (formApi) {
                  formApi.reset();
                  setTimeout(() => {
                    refresh();
                  }, 100);
                }
              }}
            >
              {t('重置条件')}
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
};

export default ChannelsFilters;
