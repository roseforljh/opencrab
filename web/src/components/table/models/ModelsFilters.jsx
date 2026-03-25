
import React, { useEffect, useRef, useState } from 'react';
import { Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

const ModelsFilters = ({
  formInitValues,
  setFormApi,
  searchModels,
  loading,
  searching,
  t,
}) => {
  const formApiRef = useRef(null);
  const [values, setValues] = useState(formInitValues);

  useEffect(() => {
    const api = {
      getValues: () => values,
      reset: () => setValues(formInitValues),
    };
    formApiRef.current = api;
    setFormApi(api);
  }, [formInitValues, setFormApi, values]);

  const handleReset = () => {
    if (!formApiRef.current) return;
    formApiRef.current.reset();
    setTimeout(() => {
      searchModels();
    }, 100);
  };

  return (
    <form
      className='order-1 w-full md:order-2 md:w-auto'
      onSubmit={(e) => {
        e.preventDefault();
        searchModels();
      }}
    >
      <div className='flex w-full flex-col gap-3 md:w-auto'>
        <div>
          <div className='text-sm font-medium text-white'>
            {t('搜索与筛选')}
          </div>
          <div className='mt-1 text-xs text-white/45'>
            {t('按模型名称与供应商快速定位展示配置。')}
          </div>
        </div>
        <div className='flex w-full flex-col items-center gap-2 md:w-auto md:flex-row'>
          <div className='relative w-full md:w-56'>
            <Search className='pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/40' />
            <Input
              value={values.searchKeyword || ''}
              onChange={(e) =>
                setValues((prev) => ({
                  ...prev,
                  searchKeyword: e.target.value,
                }))
              }
              placeholder={t('搜索模型名称')}
              className='h-11 rounded-2xl border-white/10 bg-white/6 pl-10 text-white placeholder:text-white/35'
            />
          </div>

          <div className='relative w-full md:w-56'>
            <Search className='pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/40' />
            <Input
              value={values.searchVendor || ''}
              onChange={(e) =>
                setValues((prev) => ({
                  ...prev,
                  searchVendor: e.target.value,
                }))
              }
              placeholder={t('搜索供应商')}
              className='h-11 rounded-2xl border-white/10 bg-white/6 pl-10 text-white placeholder:text-white/35'
            />
          </div>

          <div className='flex gap-2 w-full md:w-auto'>
            <Button
              type='submit'
              disabled={loading || searching}
              className='h-11 rounded-2xl border-0 bg-white/90 px-5 text-black hover:bg-white/80 md:w-auto'
            >
              {t('查询')}
            </Button>

            <Button
              type='button'
              variant='secondary'
              onClick={handleReset}
              className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10 md:w-auto'
            >
              {t('重置')}
            </Button>
          </div>
        </div>
      </div>
    </form>
  );
};

export default ModelsFilters;
