
import React, { useState } from 'react';
import { Space, SplitButtonGroup, Typography } from '@douyinfe/semi-ui';
import {
  timestamp2string,
  renderGroup,
  renderQuota,
  getModelCategories,
  showError,
} from '../../../helpers';
import {
  IconTreeTriangleDown,
  IconCopy,
  IconEyeOpened,
  IconEyeClosed,
} from '@douyinfe/semi-icons';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
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

// progress color helper
const getProgressColor = (pct) => {
  if (pct === 100) return 'bg-green-500';
  if (pct <= 10) return 'bg-red-500';
  if (pct <= 30) return 'bg-yellow-500';
  return 'bg-blue-500';
};

// Render functions
function renderTimestamp(timestamp) {
  return <>{timestamp2string(timestamp)}</>;
}

// Render status column only (no usage)
const renderStatus = (text, record, t) => {
  const enabled = text === 1;

  let tagColor = 'bg-gray-500';
  let tagText = t('未知状态');
  if (enabled) {
    tagColor = 'bg-green-500/20 text-green-400 border-green-500/30';
    tagText = t('已启用');
  } else if (text === 2) {
    tagColor = 'bg-red-500/20 text-red-400 border-red-500/30';
    tagText = t('已禁用');
  } else if (text === 3) {
    tagColor = 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
    tagText = t('已过期');
  } else if (text === 4) {
    tagColor = 'bg-gray-500/20 text-gray-400 border-gray-500/30';
    tagText = t('已耗尽');
  }

  return (
    <Badge variant='outline' className={`rounded-full font-medium ${tagColor}`}>
      {tagText}
    </Badge>
  );
};

// Render group column
const renderGroupColumn = (text, record, t) => {
  if (text === 'auto') {
    return (
      <Tooltip>
        <TooltipTrigger>
          <Badge
            variant='secondary'
            className='bg-white/10 text-white hover:bg-white/20 rounded-full cursor-help'
          >
            {t('智能熔断')}
            {record && record.cross_group_retry ? `(${t('跨分组')})` : ''}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className='bg-black/90 text-white border-white/10 text-xs'>
          <p>
            {t(
              '当前分组为 auto，会自动选择最优分组，当一个组不可用时自动降级到下一个组（熔断机制）',
            )}
          </p>
        </TooltipContent>
      </Tooltip>
    );
  }
  return renderGroup(text);
};

// Render token key column with show/hide and copy functionality
const renderTokenKey = (
  text,
  record,
  showKeys,
  resolvedTokenKeys,
  loadingTokenKeys,
  toggleTokenVisibility,
  copyTokenKey,
) => {
  const revealed = !!showKeys[record.id];
  const loading = !!loadingTokenKeys[record.id];
  const keyValue =
    revealed && resolvedTokenKeys[record.id]
      ? resolvedTokenKeys[record.id]
      : record.key || '';
  const displayedKey = keyValue ? `sk-${keyValue}` : '';

  return (
    <div className='w-[200px] relative flex items-center'>
      <Input
        readOnly
        value={displayedKey}
        className='h-8 pr-16 bg-black/20 border-white/10 text-white focus-visible:ring-white/20 text-xs'
      />
      <div className='absolute right-1 flex items-center'>
        <Button
          variant='ghost'
          size='icon'
          className='h-6 w-6 text-white/60 hover:text-white hover:bg-white/10'
          disabled={loading}
          aria-label='toggle token visibility'
          onClick={async (e) => {
            e.stopPropagation();
            await toggleTokenVisibility(record);
          }}
        >
          {revealed ? (
            <IconEyeClosed className='h-3 w-3' />
          ) : (
            <IconEyeOpened className='h-3 w-3' />
          )}
        </Button>
        <Button
          variant='ghost'
          size='icon'
          className='h-6 w-6 text-white/60 hover:text-white hover:bg-white/10'
          disabled={loading}
          aria-label='copy token key'
          onClick={async (e) => {
            e.stopPropagation();
            await copyTokenKey(record);
          }}
        >
          <IconCopy className='h-3 w-3' />
        </Button>
      </div>
    </div>
  );
};

// Render model limits column
const renderModelLimits = (text, record, t) => {
  if (record.model_limits_enabled && text) {
    const models = text.split(',').filter(Boolean);
    const categories = getModelCategories(t);

    const vendorAvatars = [];
    const matchedModels = new Set();
    Object.entries(categories).forEach(([key, category]) => {
      if (key === 'all') return;
      if (!category.icon || !category.filter) return;
      const vendorModels = models.filter((m) =>
        category.filter({ model_name: m }),
      );
      if (vendorModels.length > 0) {
        vendorAvatars.push(
          <Tooltip key={key}>
            <TooltipTrigger asChild>
              <Avatar className='h-6 w-6 border-2 border-background cursor-help'>
                <AvatarFallback className='bg-white/10 text-[10px]'>
                  {category.label?.slice(0, 2)}
                </AvatarFallback>
                <div className='w-full h-full flex items-center justify-center bg-white/5'>
                  {category.icon}
                </div>
              </Avatar>
            </TooltipTrigger>
            <TooltipContent className='bg-black/90 text-white border-white/10 text-xs'>
              <p>{vendorModels.join(', ')}</p>
            </TooltipContent>
          </Tooltip>,
        );
        vendorModels.forEach((m) => matchedModels.add(m));
      }
    });

    const unmatchedModels = models.filter((m) => !matchedModels.has(m));
    if (unmatchedModels.length > 0) {
      vendorAvatars.push(
        <Tooltip key='unknown'>
          <TooltipTrigger asChild>
            <Avatar className='h-6 w-6 border-2 border-background cursor-help'>
              <AvatarFallback className='bg-white/10 text-[10px]'>
                {t('其他')}
              </AvatarFallback>
            </Avatar>
          </TooltipTrigger>
          <TooltipContent className='bg-black/90 text-white border-white/10 text-xs'>
            <p>{unmatchedModels.join(', ')}</p>
          </TooltipContent>
        </Tooltip>,
      );
    }

    return (
      <div className='flex -space-x-2 overflow-hidden'>{vendorAvatars}</div>
    );
  } else {
    return (
      <Badge
        variant='secondary'
        className='bg-white/10 text-white hover:bg-white/20 rounded-full'
      >
        {t('无限制')}
      </Badge>
    );
  }
};

// Render IP restrictions column
const renderAllowIps = (text, t) => {
  if (!text || text.trim() === '') {
    return (
      <Badge
        variant='secondary'
        className='bg-white/10 text-white hover:bg-white/20 rounded-full'
      >
        {t('无限制')}
      </Badge>
    );
  }

  const ips = text
    .split('\n')
    .map((ip) => ip.trim())
    .filter(Boolean);

  const displayIps = ips.slice(0, 1);
  const extraCount = ips.length - displayIps.length;

  const ipTags = displayIps.map((ip, idx) => (
    <Badge
      key={idx}
      variant='outline'
      className='rounded-full border-white/20 text-white/80 font-normal'
    >
      {ip}
    </Badge>
  ));

  if (extraCount > 0) {
    ipTags.push(
      <Tooltip key='extra'>
        <TooltipTrigger asChild>
          <Badge
            variant='outline'
            className='rounded-full border-white/20 text-white/80 cursor-help font-normal'
          >
            +{extraCount}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className='bg-black/90 text-white border-white/10 text-xs max-w-xs break-words'>
          <p>{ips.slice(1).join(', ')}</p>
        </TooltipContent>
      </Tooltip>,
    );
  }

  return <div className='flex flex-wrap gap-1'>{ipTags}</div>;
};

// Render separate usage summary column
const renderQuotaUsage = (text, record, t) => {
  const used = parseInt(record.used_quota) || 0;
  const remain = parseInt(record.remain_quota) || 0;
  const total = used + remain;
  if (record.unlimited_quota) {
    return (
      <Popover>
        <PopoverTrigger asChild>
          <Badge
            variant='secondary'
            className='bg-white/10 text-white hover:bg-white/20 rounded-full cursor-help'
          >
            {t('无限使用量')}
          </Badge>
        </PopoverTrigger>
        <PopoverContent className='w-auto p-3 bg-black/90 border-white/10 text-white text-xs'>
          <div className='space-y-1'>
            <p>
              <span className='text-white/50'>{t('已用使用量')}:</span>{' '}
              {renderQuota(used)}
            </p>
          </div>
        </PopoverContent>
      </Popover>
    );
  }
  const percent = total > 0 ? (remain / total) * 100 : 0;
  return (
    <Popover>
      <PopoverTrigger asChild>
        <div className='flex flex-col items-end cursor-help group p-1 rounded hover:bg-white/5 transition-colors w-[150px]'>
          <span className='text-xs leading-none mb-1 group-hover:text-white/90 text-white/70'>
            {`${renderQuota(remain)} / ${renderQuota(total)}`}
          </span>
          <div className='w-full h-1.5 bg-white/10 rounded-full overflow-hidden'>
            <div
              className={`h-full ${getProgressColor(percent)}`}
              style={{ width: `${percent}%` }}
            />
          </div>
        </div>
      </PopoverTrigger>
      <PopoverContent className='w-auto p-3 bg-black/90 border-white/10 text-white text-xs'>
        <div className='space-y-1'>
          <p>
            <span className='text-white/50'>{t('已用使用量')}:</span>{' '}
            {renderQuota(used)}
          </p>
          <p>
            <span className='text-white/50'>{t('剩余可用量')}:</span>{' '}
            {renderQuota(remain)} ({percent.toFixed(0)}%)
          </p>
          <p>
            <span className='text-white/50'>{t('总使用量')}:</span>{' '}
            {renderQuota(total)}
          </p>
        </div>
      </PopoverContent>
    </Popover>
  );
};

// Render operations column
const renderOperations = (
  text,
  record,
  onOpenLink,
  setEditingToken,
  setShowEdit,
  manageToken,
  refresh,
  t,
) => {
  let chatsArray = [];
  try {
    const raw = localStorage.getItem('chats');
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      for (let i = 0; i < parsed.length; i++) {
        const item = parsed[i];
        const name = Object.keys(item)[0];
        if (!name) continue;
        chatsArray.push({
          node: 'item',
          key: i,
          name,
          value: item[name],
          onClick: () => onOpenLink(name, item[name], record),
        });
      }
    }
  } catch (_) {
    showError(t('聊天链接配置错误，请联系管理员'));
  }

  return (
    <div className='flex flex-wrap items-center gap-2'>
      <div className='flex -space-x-px border border-white/10 rounded-md overflow-hidden shrink-0'>
        <Button
          variant='secondary'
          size='sm'
          className='h-7 px-2 rounded-none border-r border-white/10 bg-white/5 hover:bg-white/10 text-white/80'
          onClick={() => {
            if (chatsArray.length === 0) {
              showError(t('请联系管理员配置聊天链接'));
            } else {
              const first = chatsArray[0];
              onOpenLink(first.name, first.value, record);
            }
          }}
        >
          {t('聊天')}
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant='secondary'
              size='icon'
              className='h-7 w-6 rounded-none bg-white/5 hover:bg-white/10 text-white/80'
            >
              <IconTreeTriangleDown className='h-3 w-3' />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className='bg-black/90 border-white/10 text-white'>
            {chatsArray.map((chat) => (
              <DropdownMenuItem
                key={chat.key}
                onClick={chat.onClick}
                className='hover:bg-white/10 cursor-pointer'
              >
                {chat.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {record.status === 1 ? (
        <Button
          variant='destructive'
          size='sm'
          className='h-7 px-3 bg-red-500/20 text-red-400 hover:bg-red-500/30'
          onClick={async () => {
            await manageToken(record.id, 'disable', record);
            await refresh();
          }}
        >
          {t('禁用')}
        </Button>
      ) : (
        <Button
          variant='secondary'
          size='sm'
          className='h-7 px-3 bg-green-500/20 text-green-400 hover:bg-green-500/30'
          onClick={async () => {
            await manageToken(record.id, 'enable', record);
            await refresh();
          }}
        >
          {t('启用')}
        </Button>
      )}

      <Button
        variant='secondary'
        size='sm'
        className='h-7 px-3 bg-white/5 hover:bg-white/10 text-white'
        onClick={() => {
          setEditingToken(record);
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
            <AlertDialogTitle>{t('确定是否要删除此令牌？')}</AlertDialogTitle>
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
                  await manageToken(record.id, 'delete', record);
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

export const getTokensColumns = ({
  t,
  showKeys,
  resolvedTokenKeys,
  loadingTokenKeys,
  toggleTokenVisibility,
  copyTokenKey,
  manageToken,
  onOpenLink,
  setEditingToken,
  setShowEdit,
  refresh,
}) => {
  return [
    {
      accessorKey: 'name',
      header: t('名称'),
      cell: ({ row }) => <div>{row.original.name}</div>,
    },
    {
      accessorKey: 'status',
      header: t('状态'),
      cell: ({ row }) => renderStatus(row.original.status, row.original, t),
    },
    {
      id: 'quota_usage',
      header: t('可用量/总使用量'),
      cell: ({ row }) => renderQuotaUsage(null, row.original, t),
    },
    {
      accessorKey: 'group',
      header: t('分组'),
      cell: ({ row }) => renderGroupColumn(row.original.group, row.original, t),
    },
    {
      id: 'token_key',
      header: t('密钥'),
      cell: ({ row }) =>
        renderTokenKey(
          null,
          row.original,
          showKeys,
          resolvedTokenKeys,
          loadingTokenKeys,
          toggleTokenVisibility,
          copyTokenKey,
        ),
    },
    {
      accessorKey: 'model_limits',
      header: t('可用模型'),
      cell: ({ row }) =>
        renderModelLimits(row.original.model_limits, row.original, t),
    },
    {
      accessorKey: 'allow_ips',
      header: t('IP限制'),
      cell: ({ row }) => renderAllowIps(row.original.allow_ips, t),
    },
    {
      accessorKey: 'created_time',
      header: t('创建时间'),
      cell: ({ row }) => (
        <div>{renderTimestamp(row.original.created_time)}</div>
      ),
    },
    {
      accessorKey: 'expired_time',
      header: t('过期时间'),
      cell: ({ row }) => {
        return (
          <div>
            {row.original.expired_time === -1
              ? t('永不过期')
              : renderTimestamp(row.original.expired_time)}
          </div>
        );
      },
    },
    {
      id: 'operate',
      header: '',
      cell: ({ row }) =>
        renderOperations(
          null,
          row.original,
          onOpenLink,
          setEditingToken,
          setShowEdit,
          manageToken,
          refresh,
          t,
        ),
    },
  ];
};
