
import React, { useEffect, useRef, useState } from 'react';
import { Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

const TokensFilters = ({
  formInitValues,
  setFormApi,
  searchTokens,
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
      searchTokens();
    }, 100);
  };

  return (
    <form
      className='w-full'
      onSubmit={(e) => {
        e.preventDefault();
        searchTokens(1);
      }}
    >
      <div className='flex flex-col gap-3'>
        <div>
          <div className='text-sm font-semibold text-white'>
            {t('搜索令牌')}
          </div>
          <div className='mt-1 text-xs text-white/45'>
            {t('按名称或密钥快速定位令牌，减少在表格中逐行查找。')}
          </div>
        </div>

        <div className='grid grid-cols-1 gap-2 md:grid-cols-2'>
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
              placeholder={t('搜索关键字')}
              className='h-11 rounded-2xl border-white/10 bg-white/6 pl-10 text-white placeholder:text-white/35'
            />
          </div>

          <div className='relative'>
            <Search className='pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/40' />
            <Input
              value={values.searchToken || ''}
              onChange={(e) =>
                setValues((prev) => ({
                  ...prev,
                  searchToken: e.target.value,
                }))
              }
              placeholder={t('密钥')}
              className='h-11 rounded-2xl border-white/10 bg-white/6 pl-10 text-white placeholder:text-white/35'
            />
          </div>
        </div>

        <div className='flex flex-wrap gap-2'>
          <Button
            type='submit'
            disabled={loading || searching}
            className='h-11 rounded-2xl border-0 bg-white/90 px-5 text-black hover:bg-white/80'
          >
            {t('查询令牌')}
          </Button>

          <Button
            type='button'
            variant='secondary'
            className='h-11 rounded-2xl border border-white/10 bg-white/6 px-5 text-white hover:bg-white/10'
            onClick={handleReset}
          >
            {t('重置条件')}
          </Button>
        </div>
      </div>
    </form>
  );
};

export default TokensFilters;
