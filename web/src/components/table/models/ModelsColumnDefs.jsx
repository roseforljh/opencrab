
import React from 'react';
import {
  timestamp2string,
  getLobeHubIcon,
  stringToColor,
} from '../../../helpers';
import {
  renderLimitedItems,
  renderDescription,
} from '../../common/ui/RenderUtils';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';

// Render timestamp
function renderTimestamp(timestamp) {
  return <>{timestamp2string(timestamp)}</>;
}

// Render model icon column: prefer model.icon, then fallback to vendor icon
const renderModelIconCol = (record, vendorMap) => {
  const iconKey = record?.icon || vendorMap[record?.vendor_id]?.icon;
  if (!iconKey) return '-';
  return (
    <div className='flex items-center justify-center'>
      {getLobeHubIcon(iconKey, 20)}
    </div>
  );
};

// Render vendor column with icon
const renderVendorTag = (vendorId, vendorMap, t) => {
  if (!vendorId || !vendorMap[vendorId]) return '-';
  const v = vendorMap[vendorId];
  return (
    <Badge
      variant='outline'
      className='rounded-full gap-1 border-white/20 bg-white/5 text-white/80 font-normal'
    >
      {getLobeHubIcon(v.icon || 'Layers', 14)}
      {v.name}
    </Badge>
  );
};

// Render groups (enable_groups)
const renderGroups = (groups) => {
  if (!groups || groups.length === 0) return '-';
  return renderLimitedItems({
    items: groups,
    renderItem: (g, idx) => (
      <Badge key={idx} variant='secondary' className='rounded-full font-normal'>
        {g}
      </Badge>
    ),
  });
};

// Render tags
const renderTags = (text) => {
  if (!text) return '-';
  const tagsArr = text.split(',').filter(Boolean);
  return renderLimitedItems({
    items: tagsArr,
    renderItem: (tag, idx) => (
      <Badge key={idx} variant='secondary' className='rounded-full font-normal'>
        {tag}
      </Badge>
    ),
  });
};

// Render endpoints (supports object map or legacy array)
const renderEndpoints = (value) => {
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value) : value;
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      const keys = Object.keys(parsed || {});
      if (keys.length === 0) return '-';
      return renderLimitedItems({
        items: keys,
        renderItem: (key, idx) => (
          <Badge
            key={idx}
            variant='secondary'
            className='rounded-full font-normal'
          >
            {key}
          </Badge>
        ),
        maxDisplay: 3,
      });
    }
    if (Array.isArray(parsed)) {
      if (parsed.length === 0) return '-';
      return renderLimitedItems({
        items: parsed,
        renderItem: (ep, idx) => (
          <Badge
            key={idx}
            variant='outline'
            className='rounded-full border-white/20 text-white/80 font-normal'
          >
            {ep}
          </Badge>
        ),
        maxDisplay: 3,
      });
    }
    return value || '-';
  } catch (_) {
    return value || '-';
  }
};

// Render quota types (array) using common limited items renderer
const renderQuotaTypes = (arr, t) => {
  if (!Array.isArray(arr) || arr.length === 0) return '-';
  return renderLimitedItems({
    items: arr,
    renderItem: (qt, idx) => {
      if (qt === 1) {
        return (
          <Badge
            key={`${qt}-${idx}`}
            variant='secondary'
            className='rounded-full bg-teal-500/20 text-teal-400 hover:bg-teal-500/30 font-normal'
          >
            {t('按次计费')}
          </Badge>
        );
      }
      if (qt === 0) {
        return (
          <Badge
            key={`${qt}-${idx}`}
            variant='secondary'
            className='rounded-full bg-violet-500/20 text-violet-400 hover:bg-violet-500/30 font-normal'
          >
            {t('按量计费')}
          </Badge>
        );
      }
      return (
        <Badge
          key={`${qt}-${idx}`}
          variant='outline'
          className='rounded-full border-white/20 text-white/80 font-normal'
        >
          {qt}
        </Badge>
      );
    },
    maxDisplay: 3,
  });
};

// Render bound channels
const renderBoundChannels = (channels) => {
  if (!channels || channels.length === 0) return '-';
  return renderLimitedItems({
    items: channels,
    renderItem: (c, idx) => (
      <Badge
        key={idx}
        variant='outline'
        className='rounded-full border-white/20 text-white/80 font-normal'
      >
        {c.name}({c.type})
      </Badge>
    ),
  });
};

// Render operations column
const renderOperations = (
  text,
  record,
  setEditingModel,
  setShowEdit,
  manageModel,
  refresh,
  t,
) => {
  return (
    <div className='flex flex-wrap items-center gap-2'>
      {record.status === 1 ? (
        <Button
          variant='destructive'
          size='sm'
          className='h-7 px-3 bg-red-500/20 text-red-400 hover:bg-red-500/30'
          onClick={() => manageModel(record.id, 'disable', record)}
        >
          {t('禁用')}
        </Button>
      ) : (
        <Button
          variant='secondary'
          size='sm'
          className='h-7 px-3 bg-green-500/20 text-green-400 hover:bg-green-500/30'
          onClick={() => manageModel(record.id, 'enable', record)}
        >
          {t('启用')}
        </Button>
      )}

      <Button
        variant='secondary'
        size='sm'
        className='h-7 px-3 bg-white/5 hover:bg-white/10 text-white'
        onClick={() => {
          setEditingModel(record);
          setShowEdit(true);
        }}
      >
        {t('编辑')}
      </Button>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button
            variant='destructive'
            size='sm'
            className='h-7 px-3 bg-red-500/20 text-red-400 hover:bg-red-500/30'
          >
            {t('删除')}
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent className='bg-black border-white/10 text-white'>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('确定是否要删除此模型？')}</AlertDialogTitle>
            <AlertDialogDescription className='text-white/60'>
              {t('此修改将不可逆')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className='bg-white/5 hover:bg-white/10 border-0'>
              {t('取消')}
            </AlertDialogCancel>
            <AlertDialogAction
              className='bg-red-500 hover:bg-red-600 text-white'
              onClick={() => {
                (async () => {
                  await manageModel(record.id, 'delete', record);
                  await refresh();
                })();
              }}
            >
              {t('确认删除')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

// 名称匹配类型渲染（带匹配数量 Tooltip）
const renderNameRule = (rule, record, t) => {
  const map = {
    0: {
      color: 'bg-green-500/20 text-green-400 border-green-500/30',
      label: t('精确'),
    },
    1: {
      color: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
      label: t('前缀'),
    },
    2: {
      color: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
      label: t('包含'),
    },
    3: {
      color: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
      label: t('后缀'),
    },
  };
  const cfg = map[rule];
  if (!cfg) return '-';

  let label = cfg.label;
  if (rule !== 0 && record.matched_count) {
    label = `${cfg.label} ${record.matched_count}${t('个模型')}`;
  }

  const tagElement = (
    <Badge
      variant='outline'
      className={`rounded-full font-medium ${cfg.color}`}
    >
      {label}
    </Badge>
  );

  if (
    rule === 0 ||
    !record.matched_models ||
    record.matched_models.length === 0
  ) {
    return tagElement;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className='cursor-help'>{tagElement}</span>
      </TooltipTrigger>
      <TooltipContent className='bg-black/90 border-white/10 text-white max-w-xs break-words'>
        <p>{record.matched_models.join(', ')}</p>
      </TooltipContent>
    </Tooltip>
  );
};

export const getModelsColumns = ({
  t,
  manageModel,
  setEditingModel,
  setShowEdit,
  refresh,
  vendorMap,
}) => {
  return [
    {
      id: 'icon',
      header: t('图标'),
      accessorKey: 'icon',
      cell: ({ row }) => renderModelIconCol(row.original, vendorMap),
    },
    {
      id: 'model_name',
      header: t('模型名称'),
      accessorKey: 'model_name',
      cell: ({ row }) => (
        <span
          className='cursor-text select-all'
          onClick={(e) => e.stopPropagation()}
        >
          {row.original.model_name}
        </span>
      ),
    },
    {
      id: 'name_rule',
      header: t('匹配类型'),
      accessorKey: 'name_rule',
      cell: ({ row }) =>
        renderNameRule(row.original.name_rule, row.original, t),
    },
    {
      id: 'sync_official',
      header: t('参与官方同步'),
      accessorKey: 'sync_official',
      cell: ({ row }) => {
        const val = row.original.sync_official;
        return (
          <Badge
            variant='outline'
            className={`rounded-full font-medium ${val === 1 ? 'bg-green-500/20 text-green-400 border-green-500/30' : 'bg-orange-500/20 text-orange-400 border-orange-500/30'}`}
          >
            {val === 1 ? t('是') : t('否')}
          </Badge>
        );
      },
    },
    {
      id: 'description',
      header: t('描述'),
      accessorKey: 'description',
      cell: ({ row }) => renderDescription(row.original.description, 200),
    },
    {
      id: 'vendor_id',
      header: t('供应商'),
      accessorKey: 'vendor_id',
      cell: ({ row }) => renderVendorTag(row.original.vendor_id, vendorMap, t),
    },
    {
      id: 'tags',
      header: t('标签'),
      accessorKey: 'tags',
      cell: ({ row }) => renderTags(row.original.tags),
    },
    {
      id: 'endpoints',
      header: t('端点'),
      accessorKey: 'endpoints',
      cell: ({ row }) => renderEndpoints(row.original.endpoints),
    },
    {
      id: 'bound_channels',
      header: t('已绑定渠道'),
      accessorKey: 'bound_channels',
      cell: ({ row }) => renderBoundChannels(row.original.bound_channels),
    },
    {
      id: 'enable_groups',
      header: t('可用分组'),
      accessorKey: 'enable_groups',
      cell: ({ row }) => renderGroups(row.original.enable_groups),
    },
    {
      id: 'quota_types',
      header: t('计费类型'),
      accessorKey: 'quota_types',
      cell: ({ row }) => renderQuotaTypes(row.original.quota_types, t),
    },
    {
      id: 'created_time',
      header: t('创建时间'),
      accessorKey: 'created_time',
      cell: ({ row }) => (
        <div>{renderTimestamp(row.original.created_time)}</div>
      ),
    },
    {
      id: 'updated_time',
      header: t('更新时间'),
      accessorKey: 'updated_time',
      cell: ({ row }) => (
        <div>{renderTimestamp(row.original.updated_time)}</div>
      ),
    },
    {
      id: 'operate',
      header: '',
      cell: ({ row }) =>
        renderOperations(
          null,
          row.original,
          setEditingModel,
          setShowEdit,
          manageModel,
          refresh,
          t,
        ),
    },
  ];
};
