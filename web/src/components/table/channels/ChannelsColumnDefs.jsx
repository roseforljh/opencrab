
import React from 'react';
import {
  timestamp2string,
  renderGroup,
  renderQuota,
  getChannelIcon,
  renderQuotaWithAmount,
  showSuccess,
  showError,
  showInfo,
} from '../../../helpers';
import {
  CHANNEL_OPTIONS,
  MODEL_FETCHABLE_CHANNEL_TYPES,
} from '../../../constants';
import { parseUpstreamUpdateMeta } from '../../../hooks/channels/upstreamUpdateUtils';
import {
  IconTreeTriangleDown,
  IconMore,
  IconAlertTriangle,
} from '@douyinfe/semi-icons';
import { FaRandom } from 'react-icons/fa';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
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

// Render functions
const renderType = (type, record = {}, t) => {
  const channelInfo = record?.channel_info;
  let type2label = new Map();
  for (let i = 0; i < CHANNEL_OPTIONS.length; i++) {
    type2label[CHANNEL_OPTIONS[i].value] = CHANNEL_OPTIONS[i];
  }
  type2label[0] = { value: 0, label: t('未知类型'), color: 'grey' };

  let icon = getChannelIcon(type);

  if (channelInfo?.is_multi_key) {
    icon =
      channelInfo?.multi_key_mode === 'random' ? (
        <div className='flex items-center gap-1'>
          <FaRandom className='text-blue-400' />
          {icon}
        </div>
      ) : (
        <div className='flex items-center gap-1'>
          <IconTreeTriangleDown className='text-blue-400' />
          {icon}
        </div>
      );
  }

  const typeTag = (
    <Badge
      variant='outline'
      className={`rounded-full gap-1 border-white/20 bg-white/5 text-white/80 font-normal`}
    >
      {icon}
      {type2label[type]?.label}
    </Badge>
  );

  let ionetMeta = null;
  if (record?.other_info) {
    try {
      const parsed = JSON.parse(record.other_info);
      if (parsed && typeof parsed === 'object' && parsed.source === 'ionet') {
        ionetMeta = parsed;
      }
    } catch (error) {
      // ignore invalid metadata
    }
  }

  if (!ionetMeta) {
    return typeTag;
  }

  const handleNavigate = (event) => {
    event?.stopPropagation?.();
    if (!ionetMeta?.deployment_id) {
      return;
    }
    const targetUrl = `/console/deployment?deployment_id=${ionetMeta.deployment_id}`;
    window.open(targetUrl, '_blank', 'noopener');
  };

  return (
    <div className='flex flex-wrap items-center gap-1.5'>
      {typeTag}
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge
            variant='secondary'
            className='bg-purple-500/20 text-purple-300 hover:bg-purple-500/30 cursor-pointer rounded-full'
            onClick={handleNavigate}
          >
            IO.NET
          </Badge>
        </TooltipTrigger>
        <TooltipContent className='bg-black/90 border-white/10 text-white max-w-xs'>
          <div>
            <div className='text-xs text-white/55'>
              {t('挂载自 IO.NET 实例')}
            </div>
            {ionetMeta?.deployment_id && (
              <div className='mt-1 text-xs text-white/45'>
                {t('部署 ID')}: {ionetMeta.deployment_id}
              </div>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </div>
  );
};

const renderTagType = (t) => {
  return (
    <Badge
      variant='secondary'
      className='bg-blue-500/20 text-blue-300 hover:bg-blue-500/30 rounded-full font-normal'
    >
      {t('标签聚合')}
    </Badge>
  );
};

const renderStatus = (status, channelInfo = undefined, t) => {
  if (channelInfo) {
    if (channelInfo.is_multi_key) {
      let keySize = channelInfo.multi_key_size;
      let enabledKeySize = keySize;
      if (channelInfo.multi_key_status_list) {
        enabledKeySize =
          keySize - Object.keys(channelInfo.multi_key_status_list).length;
      }
      return renderMultiKeyStatus(status, keySize, enabledKeySize, t);
    }
  }
  switch (status) {
    case 1:
      return (
        <Badge
          variant='outline'
          className='bg-green-500/20 text-green-400 border-green-500/30 rounded-full font-medium'
        >
          {t('已启用')}
        </Badge>
      );
    case 2:
      return (
        <Badge
          variant='outline'
          className='bg-red-500/20 text-red-400 border-red-500/30 rounded-full font-medium'
        >
          {t('已禁用')}
        </Badge>
      );
    case 3:
      return (
        <Badge
          variant='outline'
          className='bg-yellow-500/20 text-yellow-400 border-yellow-500/30 rounded-full font-medium'
        >
          {t('自动禁用')}
        </Badge>
      );
    default:
      return (
        <Badge
          variant='outline'
          className='bg-gray-500/20 text-gray-400 border-gray-500/30 rounded-full font-medium'
        >
          {t('未知状态')}
        </Badge>
      );
  }
};

const renderMultiKeyStatus = (status, keySize, enabledKeySize, t) => {
  switch (status) {
    case 1:
      return (
        <Badge
          variant='outline'
          className='bg-green-500/20 text-green-400 border-green-500/30 rounded-full font-medium'
        >
          {t('已启用')} {enabledKeySize}/{keySize}
        </Badge>
      );
    case 2:
      return (
        <Badge
          variant='outline'
          className='bg-red-500/20 text-red-400 border-red-500/30 rounded-full font-medium'
        >
          {t('已禁用')} {enabledKeySize}/{keySize}
        </Badge>
      );
    case 3:
      return (
        <Badge
          variant='outline'
          className='bg-yellow-500/20 text-yellow-400 border-yellow-500/30 rounded-full font-medium'
        >
          {t('自动禁用')} {enabledKeySize}/{keySize}
        </Badge>
      );
    default:
      return (
        <Badge
          variant='outline'
          className='bg-gray-500/20 text-gray-400 border-gray-500/30 rounded-full font-medium'
        >
          {t('未知状态')} {enabledKeySize}/{keySize}
        </Badge>
      );
  }
};

const renderResponseTime = (responseTime, t) => {
  let time = responseTime / 1000;
  time = time.toFixed(2) + t(' 秒');
  if (responseTime === 0) {
    return (
      <Badge
        variant='outline'
        className='bg-gray-500/20 text-gray-400 border-gray-500/30 rounded-full font-medium'
      >
        {t('未测试')}
      </Badge>
    );
  } else if (responseTime <= 1000) {
    return (
      <Badge
        variant='outline'
        className='bg-green-500/20 text-green-400 border-green-500/30 rounded-full font-medium'
      >
        {time}
      </Badge>
    );
  } else if (responseTime <= 3000) {
    return (
      <Badge
        variant='outline'
        className='bg-lime-500/20 text-lime-400 border-lime-500/30 rounded-full font-medium'
      >
        {time}
      </Badge>
    );
  } else if (responseTime <= 5000) {
    return (
      <Badge
        variant='outline'
        className='bg-yellow-500/20 text-yellow-400 border-yellow-500/30 rounded-full font-medium'
      >
        {time}
      </Badge>
    );
  } else {
    return (
      <Badge
        variant='outline'
        className='bg-red-500/20 text-red-400 border-red-500/30 rounded-full font-medium'
      >
        {time}
      </Badge>
    );
  }
};

const isRequestPassThroughEnabled = (record) => {
  if (!record || record.children !== undefined) {
    return false;
  }
  const settingValue = record.setting;
  if (!settingValue) {
    return false;
  }
  if (typeof settingValue === 'object') {
    return settingValue.pass_through_body_enabled === true;
  }
  if (typeof settingValue !== 'string') {
    return false;
  }
  try {
    const parsed = JSON.parse(settingValue);
    return parsed?.pass_through_body_enabled === true;
  } catch (error) {
    return false;
  }
};

const getUpstreamUpdateMeta = (record) => {
  const supported =
    !!record &&
    record.children === undefined &&
    MODEL_FETCHABLE_CHANNEL_TYPES.has(record.type);
  if (!record || record.children !== undefined) {
    return {
      supported: false,
      enabled: false,
      pendingAddModels: [],
      pendingRemoveModels: [],
    };
  }
  const parsed =
    record?.upstreamUpdateMeta && typeof record.upstreamUpdateMeta === 'object'
      ? record.upstreamUpdateMeta
      : parseUpstreamUpdateMeta(record?.settings);
  return {
    supported,
    enabled: parsed?.enabled === true,
    pendingAddModels: Array.isArray(parsed?.pendingAddModels)
      ? parsed.pendingAddModels
      : [],
    pendingRemoveModels: Array.isArray(parsed?.pendingRemoveModels)
      ? parsed.pendingRemoveModels
      : [],
  };
};

export const getChannelsColumns = ({
  t,
  COLUMN_KEYS,
  updateChannelBalance,
  manageChannel,
  manageTag,
  submitTagEdit,
  testChannel,
  setCurrentTestChannel,
  setShowModelTestModal,
  setEditingChannel,
  setShowEdit,
  setShowEditTag,
  setEditingTag,
  copySelectedChannel,
  refresh,
  activePage,
  channels,
  checkOllamaVersion,
  setShowMultiKeyManageModal,
  setCurrentMultiKeyChannel,
  openUpstreamUpdateModal,
  detectChannelUpstreamUpdates,
}) => {
  return [
    {
      id: COLUMN_KEYS.ID,
      header: t('ID'),
      accessorKey: 'id',
    },
    {
      id: COLUMN_KEYS.NAME,
      header: t('名称'),
      accessorKey: 'name',
      cell: ({ row }) => {
        const record = row.original;
        const text = record.name;
        const passThroughEnabled = isRequestPassThroughEnabled(record);
        const upstreamUpdateMeta = getUpstreamUpdateMeta(record);
        const pendingAddCount = upstreamUpdateMeta.pendingAddModels.length;
        const pendingRemoveCount =
          upstreamUpdateMeta.pendingRemoveModels.length;
        const showUpstreamUpdateTag =
          upstreamUpdateMeta.supported &&
          upstreamUpdateMeta.enabled &&
          (pendingAddCount > 0 || pendingRemoveCount > 0);
        const nameNode =
          record.remark && record.remark.trim() !== '' ? (
            <Tooltip
              content={
                <div className='flex flex-col gap-2 max-w-xs'>
                  <div className='text-sm'>{record.remark}</div>
                  <Button
                    size='small'
                    type='primary'
                    theme='outline'
                    onClick={(e) => {
                      e.stopPropagation();
                      navigator.clipboard
                        .writeText(record.remark)
                        .then(() => {
                          showSuccess(t('复制成功'));
                        })
                        .catch(() => {
                          showError(t('复制失败'));
                        });
                    }}
                  >
                    {t('复制')}
                  </Button>
                </div>
              }
              trigger='hover'
              position='topLeft'
            >
              <span>{text}</span>
            </Tooltip>
          ) : (
            <span>{text}</span>
          );

        if (!passThroughEnabled && !showUpstreamUpdateTag) {
          return nameNode;
        }

        return (
          <Space spacing={6} align='center'>
            {nameNode}
            {passThroughEnabled && (
              <Tooltip
                content={t(
                  '该渠道已开启请求透传：参数覆写、模型重定向、渠道适配等 OpenCrab 内置功能将失效，非最佳实践；如因此产生问题，请勿提交 issue 反馈。',
                )}
                trigger='hover'
                position='topLeft'
              >
                <span className='inline-flex items-center'>
                  <IconAlertTriangle
                    style={{ color: 'var(--semi-color-warning)' }}
                  />
                </span>
              </Tooltip>
            )}
            {showUpstreamUpdateTag && (
              <Space spacing={4} align='center'>
                {pendingAddCount > 0 ? (
                  <Tooltip content={t('点击处理新增模型')} position='top'>
                    <Tag
                      color='green'
                      type='light'
                      size='small'
                      shape='circle'
                      className='cursor-pointer transition-all duration-150 hover:opacity-85 hover:-translate-y-px active:scale-95'
                      onClick={(e) => {
                        e.stopPropagation();
                        openUpstreamUpdateModal(
                          record,
                          upstreamUpdateMeta.pendingAddModels,
                          upstreamUpdateMeta.pendingRemoveModels,
                          'add',
                        );
                      }}
                    >
                      +{pendingAddCount}
                    </Tag>
                  </Tooltip>
                ) : null}
                {pendingRemoveCount > 0 ? (
                  <Tooltip content={t('点击处理删除模型')} position='top'>
                    <Tag
                      color='red'
                      type='light'
                      size='small'
                      shape='circle'
                      className='cursor-pointer transition-all duration-150 hover:opacity-85 hover:-translate-y-px active:scale-95'
                      onClick={(e) => {
                        e.stopPropagation();
                        openUpstreamUpdateModal(
                          record,
                          upstreamUpdateMeta.pendingAddModels,
                          upstreamUpdateMeta.pendingRemoveModels,
                          'remove',
                        );
                      }}
                    >
                      -{pendingRemoveCount}
                    </Tag>
                  </Tooltip>
                ) : null}
              </Space>
            )}
          </Space>
        );
      },
    },
    {
      id: COLUMN_KEYS.GROUP,
      header: t('分组'),
      accessorKey: 'group',
      cell: ({ row }) => {
        const text = row.original.group;
        return (
          <div className='flex flex-wrap gap-1'>
            {text
              ?.split(',')
              .sort((a, b) => {
                if (a === 'default') return -1;
                if (b === 'default') return 1;
                return a.localeCompare(b);
              })
              .map((item, index) => (
                <span key={index}>{renderGroup(item)}</span>
              ))}
          </div>
        );
      },
    },
    {
      id: COLUMN_KEYS.TYPE,
      header: t('类型'),
      accessorKey: 'type',
      cell: ({ row }) => {
        const record = row.original;
        const text = record.type;
        if (record.children === undefined) {
          return <>{renderType(text, record, t)}</>;
        } else {
          return <>{renderTagType(t)}</>;
        }
      },
    },
    {
      id: COLUMN_KEYS.STATUS,
      header: t('状态'),
      accessorKey: 'status',
      cell: ({ row }) => {
        const record = row.original;
        const text = record.status;
        if (text === 3) {
          if (record.other_info === '') {
            record.other_info = '{}';
          }
          let otherInfo = JSON.parse(record.other_info);
          let reason = otherInfo['status_reason'];
          let time = otherInfo['status_time'];
          return (
            <Tooltip>
              <TooltipTrigger asChild>
                <div className='cursor-help'>
                  {renderStatus(text, record.channel_info, t)}
                </div>
              </TooltipTrigger>
              <TooltipContent className='bg-black/90 border-white/10 text-white max-w-xs'>
                <p>
                  {t('原因：') +
                    reason +
                    t('，时间：') +
                    timestamp2string(time)}
                </p>
              </TooltipContent>
            </Tooltip>
          );
        } else {
          return renderStatus(text, record.channel_info, t);
        }
      },
    },
    {
      id: COLUMN_KEYS.RESPONSE_TIME,
      header: t('响应时间'),
      accessorKey: 'response_time',
      cell: ({ row }) => (
        <div>{renderResponseTime(row.original.response_time, t)}</div>
      ),
    },
    {
      id: COLUMN_KEYS.BALANCE,
      header: t('已用/剩余'),
      accessorKey: 'expired_time',
      cell: ({ row }) => {
        const record = row.original;
        if (record.children === undefined) {
          return (
            <div className='flex items-center gap-1'>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge
                    variant='outline'
                    className='border-white/20 text-white/80 font-normal rounded-full cursor-help'
                  >
                    {renderQuota(record.used_quota)}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent className='bg-black/90 border-white/10 text-white'>
                  <p>{t('已用额度')}</p>
                </TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge
                    variant='outline'
                    className='border-white/20 text-white/80 font-normal rounded-full cursor-pointer hover:bg-white/10 transition-colors'
                    onClick={() => updateChannelBalance(record)}
                  >
                    {renderQuotaWithAmount(record.balance)}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent className='bg-black/90 border-white/10 text-white'>
                  <p>
                    {t('剩余额度') +
                      ': ' +
                      renderQuotaWithAmount(record.balance) +
                      t('，点击更新')}
                  </p>
                </TooltipContent>
              </Tooltip>
            </div>
          );
        } else {
          return (
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge
                  variant='outline'
                  className='border-white/20 text-white/80 font-normal rounded-full cursor-help'
                >
                  {renderQuota(record.used_quota)}
                </Badge>
              </TooltipTrigger>
              <TooltipContent className='bg-black/90 border-white/10 text-white'>
                <p>{t('已用额度')}</p>
              </TooltipContent>
            </Tooltip>
          );
        }
      },
    },
    {
      id: COLUMN_KEYS.PRIORITY,
      header: t('优先级'),
      accessorKey: 'priority',
      cell: ({ row }) => {
        const record = row.original;
        if (record.children === undefined) {
          return (
            <Input
              type='number'
              className='w-16 h-8 text-center bg-black/20 border-white/10 text-white focus-visible:ring-white/20 p-1'
              defaultValue={record.priority}
              min={-999}
              onBlur={(e) => {
                manageChannel(record.id, 'priority', record, e.target.value);
              }}
            />
          );
        } else {
          return (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Input
                  type='number'
                  className='w-16 h-8 text-center bg-black/20 border-white/10 text-white focus-visible:ring-white/20 p-1'
                  defaultValue={record.priority}
                  min={-999}
                />
              </AlertDialogTrigger>
              <AlertDialogContent className='bg-black border-white/10 text-white'>
                <AlertDialogHeader>
                  <AlertDialogTitle>{t('修改子渠道优先级')}</AlertDialogTitle>
                  <AlertDialogDescription className='text-white/60'>
                    {t('确定要修改所有子渠道优先级吗？')}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel className='bg-white/5 hover:bg-white/10 border-0'>
                    {t('取消')}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    className='bg-white text-black hover:bg-white/90'
                    onClick={(e) => {
                      const inputVal = e.currentTarget
                        .closest('[role="dialog"]')
                        ?.querySelector('input[type="number"]')?.value;
                      if (inputVal === '') return;
                      submitTagEdit('priority', {
                        tag: record.key,
                        priority: inputVal,
                      });
                    }}
                  >
                    {t('确认修改')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          );
        }
      },
    },
    {
      id: COLUMN_KEYS.WEIGHT,
      header: t('权重'),
      accessorKey: 'weight',
      cell: ({ row }) => {
        const record = row.original;
        if (record.children === undefined) {
          return (
            <Input
              type='number'
              className='w-16 h-8 text-center bg-black/20 border-white/10 text-white focus-visible:ring-white/20 p-1'
              defaultValue={record.weight}
              min={0}
              onBlur={(e) => {
                manageChannel(record.id, 'weight', record, e.target.value);
              }}
            />
          );
        } else {
          return (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Input
                  type='number'
                  className='w-16 h-8 text-center bg-black/20 border-white/10 text-white focus-visible:ring-white/20 p-1'
                  defaultValue={record.weight}
                  min={-999}
                />
              </AlertDialogTrigger>
              <AlertDialogContent className='bg-black border-white/10 text-white'>
                <AlertDialogHeader>
                  <AlertDialogTitle>{t('修改子渠道权重')}</AlertDialogTitle>
                  <AlertDialogDescription className='text-white/60'>
                    {t('确定要修改所有子渠道权重吗？')}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel className='bg-white/5 hover:bg-white/10 border-0'>
                    {t('取消')}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    className='bg-white text-black hover:bg-white/90'
                    onClick={(e) => {
                      const inputVal = e.currentTarget
                        .closest('[role="dialog"]')
                        ?.querySelector('input[type="number"]')?.value;
                      if (inputVal === '') return;
                      submitTagEdit('weight', {
                        tag: record.key,
                        weight: inputVal,
                      });
                    }}
                  >
                    {t('确认修改')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          );
        }
      },
    },
    {
      id: COLUMN_KEYS.OPERATE,
      header: '',
      accessorKey: 'operate',
      cell: ({ row }) => {
        const record = row.original;
        if (record.children === undefined) {
          const upstreamUpdateMeta = getUpstreamUpdateMeta(record);
          const moreMenuItems = [
            {
              key: 'delete',
              name: t('删除'),
              type: 'danger',
              isAlert: true,
              alertTitle: t('确定是否要删除此渠道？'),
              alertDesc: t('此修改将不可逆'),
              onConfirm: () => {
                (async () => {
                  await manageChannel(record.id, 'delete', record);
                  await refresh();
                  setTimeout(() => {
                    if (channels.length === 0 && activePage > 1) {
                      refresh(activePage - 1);
                    }
                  }, 100);
                })();
              },
            },
            {
              key: 'copy',
              name: t('复制'),
              type: 'tertiary',
              isAlert: true,
              alertTitle: t('确定是否要复制此渠道？'),
              alertDesc: t('复制渠道的所有信息'),
              onConfirm: () => copySelectedChannel(record),
            },
          ];

          if (upstreamUpdateMeta.supported) {
            moreMenuItems.push({
              key: 'detect_upstream',
              name: t('仅检测上游模型更新'),
              type: 'tertiary',
              onClick: () => {
                detectChannelUpstreamUpdates(record);
              },
            });
            moreMenuItems.push({
              key: 'handle_upstream',
              name: t('处理上游模型更新'),
              type: 'tertiary',
              onClick: () => {
                if (!upstreamUpdateMeta.enabled) {
                  showInfo(t('该渠道未开启上游模型更新检测'));
                  return;
                }
                if (
                  upstreamUpdateMeta.pendingAddModels.length === 0 &&
                  upstreamUpdateMeta.pendingRemoveModels.length === 0
                ) {
                  showInfo(t('该渠道暂无可处理的上游模型更新'));
                  return;
                }
                openUpstreamUpdateModal(
                  record,
                  upstreamUpdateMeta.pendingAddModels,
                  upstreamUpdateMeta.pendingRemoveModels,
                  upstreamUpdateMeta.pendingAddModels.length > 0
                    ? 'add'
                    : 'remove',
                );
              },
            });
          }

          if (record.type === 4) {
            moreMenuItems.unshift({
              key: 'test_alive',
              name: t('测活'),
              type: 'tertiary',
              onClick: () => checkOllamaVersion(record),
            });
          }

          return (
            <div className='flex flex-wrap items-center gap-2'>
              <div className='flex -space-x-px border border-white/10 rounded-md overflow-hidden shrink-0'>
                <Button
                  variant='secondary'
                  size='sm'
                  className='h-7 px-2 rounded-none border-r border-white/10 bg-white/5 hover:bg-white/10 text-white/80'
                  onClick={() => testChannel(record, '')}
                >
                  {t('测试')}
                </Button>
                <Button
                  variant='secondary'
                  size='icon'
                  className='h-7 w-6 rounded-none bg-white/5 hover:bg-white/10 text-white/80'
                  onClick={() => {
                    setCurrentTestChannel(record);
                    setShowModelTestModal(true);
                  }}
                >
                  <IconTreeTriangleDown className='h-3 w-3' />
                </Button>
              </div>

              {record.status === 1 ? (
                <Button
                  variant='destructive'
                  size='sm'
                  className='h-7 px-3 bg-red-500/20 text-red-400 hover:bg-red-500/30'
                  onClick={() => manageChannel(record.id, 'disable', record)}
                >
                  {t('禁用')}
                </Button>
              ) : (
                <Button
                  variant='secondary'
                  size='sm'
                  className='h-7 px-3 bg-green-500/20 text-green-400 hover:bg-green-500/30'
                  onClick={() => manageChannel(record.id, 'enable', record)}
                >
                  {t('启用')}
                </Button>
              )}

              {record.channel_info?.is_multi_key ? (
                <div className='flex -space-x-px border border-white/10 rounded-md overflow-hidden shrink-0'>
                  <Button
                    variant='secondary'
                    size='sm'
                    className='h-7 px-2 rounded-none border-r border-white/10 bg-white/5 hover:bg-white/10 text-white/80'
                    onClick={() => {
                      setEditingChannel(record);
                      setShowEdit(true);
                    }}
                  >
                    {t('编辑')}
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
                      <DropdownMenuItem
                        onClick={() => {
                          setCurrentMultiKeyChannel(record);
                          setShowMultiKeyManageModal(true);
                        }}
                        className='hover:bg-white/10 cursor-pointer'
                      >
                        {t('多密钥管理')}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              ) : (
                <Button
                  variant='secondary'
                  size='sm'
                  className='h-7 px-3 bg-white/5 hover:bg-white/10 text-white'
                  onClick={() => {
                    setEditingChannel(record);
                    setShowEdit(true);
                  }}
                >
                  {t('编辑')}
                </Button>
              )}

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant='ghost'
                    size='icon'
                    className='h-7 w-7 text-white/70 hover:text-white hover:bg-white/10'
                  >
                    <IconMore />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className='bg-black/90 border-white/10 text-white'>
                  {moreMenuItems.map((item) =>
                    item.isAlert ? (
                      <AlertDialog key={item.key}>
                        <AlertDialogTrigger asChild>
                          <DropdownMenuItem
                            onSelect={(e) => e.preventDefault()}
                            className={`cursor-pointer ${item.type === 'danger' ? 'text-red-400 hover:text-red-300 hover:bg-red-500/20' : 'hover:bg-white/10'}`}
                          >
                            {item.name}
                          </DropdownMenuItem>
                        </AlertDialogTrigger>
                        <AlertDialogContent className='bg-black border-white/10 text-white'>
                          <AlertDialogHeader>
                            <AlertDialogTitle>
                              {item.alertTitle}
                            </AlertDialogTitle>
                            <AlertDialogDescription className='text-white/60'>
                              {item.alertDesc}
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel className='bg-white/5 hover:bg-white/10 border-0'>
                              {t('取消')}
                            </AlertDialogCancel>
                            <AlertDialogAction
                              className={
                                item.type === 'danger'
                                  ? 'bg-red-500 hover:bg-red-600 text-white'
                                  : 'bg-white text-black hover:bg-white/90'
                              }
                              onClick={item.onConfirm}
                            >
                              {item.type === 'danger'
                                ? t('确认删除')
                                : t('确认')}
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    ) : (
                      <DropdownMenuItem
                        key={item.key}
                        onClick={item.onClick}
                        className='hover:bg-white/10 cursor-pointer'
                      >
                        {item.name}
                      </DropdownMenuItem>
                    ),
                  )}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          );
        } else {
          // 标签操作按钮
          return (
            <div className='flex flex-wrap gap-2'>
              <Button
                variant='secondary'
                size='sm'
                className='h-7 px-3 bg-white/5 hover:bg-white/10 text-white'
                onClick={() => manageTag(record.key, 'enable')}
              >
                {t('启用全部')}
              </Button>
              <Button
                variant='secondary'
                size='sm'
                className='h-7 px-3 bg-white/5 hover:bg-white/10 text-white'
                onClick={() => manageTag(record.key, 'disable')}
              >
                {t('禁用全部')}
              </Button>
              <Button
                variant='secondary'
                size='sm'
                className='h-7 px-3 bg-white/5 hover:bg-white/10 text-white'
                onClick={() => {
                  setShowEditTag(true);
                  setEditingTag(record.key);
                }}
              >
                {t('编辑')}
              </Button>
            </div>
          );
        }
      },
    },
  ];
};
