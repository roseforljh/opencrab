
import React, { useState, useEffect } from 'react';
import { Download, Plus, RefreshCw, Search, Trash2, Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  authHeader,
  getUserIdFromLocalStorage,
  showError,
  showSuccess,
} from '../../../../helpers';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

const CHANNEL_TYPE_OLLAMA = 4;

const parseMaybeJSON = (value) => {
  if (!value) return null;
  if (typeof value === 'object') return value;
  if (typeof value === 'string') {
    try {
      return JSON.parse(value);
    } catch (error) {
      return null;
    }
  }
  return null;
};

const resolveOllamaBaseUrl = (info) => {
  if (!info) {
    return '';
  }

  const direct = typeof info.base_url === 'string' ? info.base_url.trim() : '';
  if (direct) {
    return direct;
  }

  const alt =
    typeof info.ollama_base_url === 'string' ? info.ollama_base_url.trim() : '';
  if (alt) {
    return alt;
  }

  const parsed = parseMaybeJSON(info.other_info);
  if (parsed && typeof parsed === 'object') {
    const candidate =
      (typeof parsed.base_url === 'string' && parsed.base_url.trim()) ||
      (typeof parsed.public_url === 'string' && parsed.public_url.trim()) ||
      (typeof parsed.api_url === 'string' && parsed.api_url.trim());
    if (candidate) {
      return candidate;
    }
  }

  return '';
};

const normalizeModels = (items) => {
  if (!Array.isArray(items)) {
    return [];
  }

  return items
    .map((item) => {
      if (!item) {
        return null;
      }

      if (typeof item === 'string') {
        return {
          id: item,
          owned_by: 'ollama',
        };
      }

      if (typeof item === 'object') {
        const candidateId =
          item.id || item.ID || item.name || item.model || item.Model;
        if (!candidateId) {
          return null;
        }

        const metadata = item.metadata || item.Metadata;
        const normalized = {
          ...item,
          id: candidateId,
          owned_by: item.owned_by || item.ownedBy || 'ollama',
        };

        if (typeof item.size === 'number' && !normalized.size) {
          normalized.size = item.size;
        }
        if (metadata && typeof metadata === 'object') {
          if (typeof metadata.size === 'number' && !normalized.size) {
            normalized.size = metadata.size;
          }
          if (!normalized.digest && typeof metadata.digest === 'string') {
            normalized.digest = metadata.digest;
          }
          if (
            !normalized.modified_at &&
            typeof metadata.modified_at === 'string'
          ) {
            normalized.modified_at = metadata.modified_at;
          }
          if (metadata.details && !normalized.details) {
            normalized.details = metadata.details;
          }
        }

        return normalized;
      }

      return null;
    })
    .filter(Boolean);
};

const OllamaModelModal = ({
  visible,
  onCancel,
  channelId,
  channelInfo,
  onModelsUpdate,
  onApplyModels,
}) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [models, setModels] = useState([]);
  const [filteredModels, setFilteredModels] = useState([]);
  const [searchValue, setSearchValue] = useState('');
  const [pullModelName, setPullModelName] = useState('');
  const [pullLoading, setPullLoading] = useState(false);
  const [pullProgress, setPullProgress] = useState(null);
  const [eventSource, setEventSource] = useState(null);
  const [selectedModelIds, setSelectedModelIds] = useState([]);
  const [deleteTarget, setDeleteTarget] = useState(null);

  const handleApplyAllModels = () => {
    if (!onApplyModels || selectedModelIds.length === 0) {
      return;
    }
    onApplyModels({ mode: 'append', modelIds: selectedModelIds });
  };

  const handleToggleModel = (modelId, checked) => {
    if (!modelId) {
      return;
    }
    setSelectedModelIds((prev) => {
      if (checked) {
        if (prev.includes(modelId)) {
          return prev;
        }
        return [...prev, modelId];
      }
      return prev.filter((id) => id !== modelId);
    });
  };

  const handleSelectAll = () => {
    setSelectedModelIds(models.map((item) => item?.id).filter(Boolean));
  };

  const handleClearSelection = () => {
    setSelectedModelIds([]);
  };

  // 获取模型列表
  const fetchModels = async () => {
    const channelType = Number(channelInfo?.type ?? CHANNEL_TYPE_OLLAMA);
    const shouldTryLiveFetch = channelType === CHANNEL_TYPE_OLLAMA;
    const resolvedBaseUrl = resolveOllamaBaseUrl(channelInfo);

    setLoading(true);
    let liveFetchSucceeded = false;
    let fallbackSucceeded = false;
    let lastError = '';
    let nextModels = [];

    try {
      if (shouldTryLiveFetch && resolvedBaseUrl) {
        try {
          const payload = {
            base_url: resolvedBaseUrl,
            type: CHANNEL_TYPE_OLLAMA,
            key: channelInfo?.key || '',
          };

          const res = await API.post('/api/channel/fetch_models', payload, {
            skipErrorHandler: true,
          });

          if (res?.data?.success) {
            nextModels = normalizeModels(res.data.data);
            liveFetchSucceeded = true;
          } else if (res?.data?.message) {
            lastError = res.data.message;
          }
        } catch (error) {
          const message = error?.response?.data?.message || error.message;
          if (message) {
            lastError = message;
          }
        }
      } else if (shouldTryLiveFetch && !resolvedBaseUrl && !channelId) {
        lastError = t('请先填写 Ollama API 地址');
      }

      if ((!liveFetchSucceeded || nextModels.length === 0) && channelId) {
        try {
          const res = await API.get(`/api/channel/fetch_models/${channelId}`, {
            skipErrorHandler: true,
          });

          if (res?.data?.success) {
            nextModels = normalizeModels(res.data.data);
            fallbackSucceeded = true;
            lastError = '';
          } else if (res?.data?.message) {
            lastError = res.data.message;
          }
        } catch (error) {
          const message = error?.response?.data?.message || error.message;
          if (message) {
            lastError = message;
          }
        }
      }

      if (!liveFetchSucceeded && !fallbackSucceeded && lastError) {
        showError(`${t('获取模型列表失败')}: ${lastError}`);
      }

      const normalized = nextModels;
      setModels(normalized);
      setFilteredModels(normalized);
      setSelectedModelIds((prev) => {
        if (!normalized || normalized.length === 0) {
          return [];
        }
        if (!prev || prev.length === 0) {
          return normalized.map((item) => item.id).filter(Boolean);
        }
        const available = prev.filter((id) =>
          normalized.some((item) => item.id === id),
        );
        return available.length > 0
          ? available
          : normalized.map((item) => item.id).filter(Boolean);
      });
    } finally {
      setLoading(false);
    }
  };

  // 拉取模型 (流式，支持进度)
  const pullModel = async () => {
    if (!pullModelName.trim()) {
      showError(t('请输入模型名称'));
      return;
    }

    setPullLoading(true);
    setPullProgress({ status: 'starting', completed: 0, total: 0 });

    let hasRefreshed = false;
    const refreshModels = async () => {
      if (hasRefreshed) return;
      hasRefreshed = true;
      await fetchModels();
      if (onModelsUpdate) {
        onModelsUpdate({ silent: true });
      }
    };

    try {
      // 关闭之前的连接
      if (eventSource) {
        eventSource.close();
        setEventSource(null);
      }

      const controller = new AbortController();
      const closable = {
        close: () => controller.abort(),
      };
      setEventSource(closable);

      // 使用 fetch 请求 SSE 流
      const authHeaders = authHeader();
      const userId = getUserIdFromLocalStorage();
      const fetchHeaders = {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
        'New-API-User': String(userId),
        ...authHeaders,
      };

      const response = await fetch('/api/channel/ollama/pull/stream', {
        method: 'POST',
        headers: fetchHeaders,
        body: JSON.stringify({
          channel_id: channelId,
          model_name: pullModelName.trim(),
        }),
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      // 读取 SSE 流
      const processStream = async () => {
        try {
          while (true) {
            const { done, value } = await reader.read();

            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop() || '';

            for (const line of lines) {
              if (!line.startsWith('data: ')) {
                continue;
              }

              try {
                const eventData = line.substring(6);
                if (eventData === '[DONE]') {
                  setPullLoading(false);
                  setPullProgress(null);
                  setEventSource(null);
                  return;
                }

                const data = JSON.parse(eventData);

                if (data.status) {
                  // 处理进度数据
                  setPullProgress(data);
                } else if (data.error) {
                  // 处理错误
                  showError(data.error);
                  setPullProgress(null);
                  setPullLoading(false);
                  setEventSource(null);
                  return;
                } else if (data.message) {
                  // 处理成功消息
                  showSuccess(data.message);
                  setPullModelName('');
                  setPullProgress(null);
                  setPullLoading(false);
                  setEventSource(null);
                  await fetchModels();
                  if (onModelsUpdate) {
                    onModelsUpdate({ silent: true });
                  }
                  await refreshModels();
                  return;
                }
              } catch (e) {
                console.error('Failed to parse SSE data:', e);
              }
            }
          }
          // 正常结束流
          setPullLoading(false);
          setPullProgress(null);
          setEventSource(null);
          await refreshModels();
        } catch (error) {
          if (error?.name === 'AbortError') {
            setPullProgress(null);
            setPullLoading(false);
            setEventSource(null);
            return;
          }
          console.error('Stream processing error:', error);
          showError(t('数据传输中断'));
          setPullProgress(null);
          setPullLoading(false);
          setEventSource(null);
          await refreshModels();
        }
      };

      await processStream();
    } catch (error) {
      if (error?.name !== 'AbortError') {
        showError(t('模型拉取失败: {{error}}', { error: error.message }));
      }
      setPullLoading(false);
      setPullProgress(null);
      setEventSource(null);
      await refreshModels();
    }
  };

  // 删除模型
  const deleteModel = async (modelName) => {
    try {
      const res = await API.delete('/api/channel/ollama/delete', {
        data: {
          channel_id: channelId,
          model_name: modelName,
        },
      });

      if (res.data.success) {
        showSuccess(t('模型删除成功'));
        await fetchModels(); // 重新获取模型列表
        if (onModelsUpdate) {
          onModelsUpdate({ silent: true }); // 通知父组件更新
        }
      } else {
        showError(res.data.message || t('模型删除失败'));
      }
    } catch (error) {
      showError(t('模型删除失败: {{error}}', { error: error.message }));
    }
  };

  // 搜索过滤
  useEffect(() => {
    if (!searchValue) {
      setFilteredModels(models);
    } else {
      const filtered = models.filter((model) =>
        model.id.toLowerCase().includes(searchValue.toLowerCase()),
      );
      setFilteredModels(filtered);
    }
  }, [models, searchValue]);

  useEffect(() => {
    if (!visible) {
      setSelectedModelIds([]);
      setPullModelName('');
      setPullProgress(null);
      setPullLoading(false);
    }
  }, [visible]);

  // 组件加载时获取模型列表
  useEffect(() => {
    if (!visible) {
      return;
    }

    if (channelId || Number(channelInfo?.type) === CHANNEL_TYPE_OLLAMA) {
      fetchModels();
    }
  }, [
    visible,
    channelId,
    channelInfo?.type,
    channelInfo?.base_url,
    channelInfo?.other_info,
    channelInfo?.ollama_base_url,
  ]);

  // 组件卸载时清理 EventSource
  useEffect(() => {
    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [eventSource]);

  const formatModelSize = (size) => {
    if (!size) return '-';
    const gb = size / (1024 * 1024 * 1024);
    return gb >= 1
      ? `${gb.toFixed(1)} GB`
      : `${(size / (1024 * 1024)).toFixed(0)} MB`;
  };

  return (
    <>
      <Dialog open={visible} onOpenChange={(open) => !open && onCancel?.()}>
        <DialogContent className='max-w-[820px] border-white/10 bg-black text-white'>
          <DialogHeader>
            <DialogTitle>{t('Ollama 模型管理')}</DialogTitle>
          </DialogHeader>

          <div className='space-y-4'>
            <div className='text-sm text-white/50'>
              {channelInfo?.name ? `${channelInfo.name} - ` : ''}
              {t('管理 Ollama 模型的拉取和删除')}
            </div>

            <Card className='border-white/10 bg-white/5 py-0'>
              <CardContent className='space-y-4 p-4'>
                <div className='text-base font-semibold'>{t('拉取新模型')}</div>
                <div className='grid gap-3 md:grid-cols-[1fr_180px]'>
                  <Input
                    placeholder={t('请输入模型名称，例如: llama3.2, qwen2.5:7b')}
                    value={pullModelName}
                    onChange={(e) => setPullModelName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        pullModel();
                      }
                    }}
                    disabled={pullLoading}
                    className='border-white/10 bg-white/6 text-white'
                  />
                  <Button
                    type='button'
                    onClick={pullModel}
                    disabled={!pullModelName.trim() || pullLoading}
                  >
                    <Download className='mr-2 h-4 w-4' />
                    {pullLoading ? t('拉取中...') : t('拉取模型')}
                  </Button>
                </div>

                {pullProgress &&
                  (() => {
                    const completedBytes = Number(pullProgress.completed) || 0;
                    const totalBytes = Number(pullProgress.total) || 0;
                    const hasTotal =
                      Number.isFinite(totalBytes) && totalBytes > 0;
                    const safePercent = hasTotal
                      ? Math.min(
                          100,
                          Math.max(
                            0,
                            Math.round((completedBytes / totalBytes) * 100),
                          ),
                        )
                      : null;
                    const percentText =
                      hasTotal && safePercent !== null
                        ? `${safePercent.toFixed(0)}%`
                        : pullProgress.status || t('处理中');

                    return (
                      <div className='space-y-2 rounded-xl border border-white/10 bg-white/5 p-3'>
                        <div className='flex items-center justify-between text-sm'>
                          <span className='font-medium'>{t('拉取进度')}</span>
                          <span className='text-white/50'>{percentText}</span>
                        </div>

                        {hasTotal && safePercent !== null ? (
                          <>
                            <div className='h-2 overflow-hidden rounded-full bg-white/10'>
                              <div
                                className='h-full rounded-full bg-blue-500 transition-all'
                                style={{ width: `${safePercent}%` }}
                              />
                            </div>
                            <div className='flex justify-between text-xs text-white/45'>
                              <span>
                                {(completedBytes / (1024 * 1024 * 1024)).toFixed(2)} GB
                              </span>
                              <span>
                                {(totalBytes / (1024 * 1024 * 1024)).toFixed(2)} GB
                              </span>
                            </div>
                          </>
                        ) : (
                          <div className='flex items-center gap-2 text-xs text-white/45'>
                            <Loader2 className='h-4 w-4 animate-spin' />
                            <span>{t('准备中...')}</span>
                          </div>
                        )}
                      </div>
                    );
                  })()}

                <div className='text-sm text-white/45'>
                  {t(
                    '支持拉取 Ollama 官方模型库中的所有模型，拉取过程可能需要几分钟时间',
                  )}
                </div>
              </CardContent>
            </Card>

            <Card className='border-white/10 bg-white/5 py-0'>
              <CardContent className='space-y-4 p-4'>
                <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
                  <div className='flex items-center gap-2'>
                    <div className='text-base font-semibold'>{t('已有模型')}</div>
                    {models.length > 0 ? (
                      <Badge className='border-blue-500/20 bg-blue-500/15 text-blue-200'>
                        {models.length}
                      </Badge>
                    ) : null}
                  </div>
                  <div className='flex flex-wrap gap-2'>
                    <div className='relative min-w-[220px] flex-1'>
                      <Search className='pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-white/40' />
                      <Input
                        placeholder={t('搜索模型...')}
                        value={searchValue}
                        onChange={(e) => setSearchValue(e.target.value)}
                        className='border-white/10 bg-white/6 pl-9 text-white'
                      />
                    </div>
                    <Button
                      type='button'
                      variant='secondary'
                      size='sm'
                      onClick={handleSelectAll}
                      disabled={models.length === 0}
                    >
                      {t('全选')}
                    </Button>
                    <Button
                      type='button'
                      variant='secondary'
                      size='sm'
                      onClick={handleClearSelection}
                      disabled={selectedModelIds.length === 0}
                    >
                      {t('清空')}
                    </Button>
                    <Button
                      type='button'
                      size='sm'
                      onClick={handleApplyAllModels}
                      disabled={selectedModelIds.length === 0}
                    >
                      <Plus className='mr-2 h-4 w-4' />
                      {t('加入渠道')}
                    </Button>
                    <Button
                      type='button'
                      variant='secondary'
                      size='sm'
                      onClick={fetchModels}
                      disabled={loading}
                    >
                      <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                      {t('刷新')}
                    </Button>
                  </div>
                </div>

                {loading ? (
                  <div className='flex items-center justify-center py-10 text-sm text-white/50'>
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    {t('加载中...')}
                  </div>
                ) : filteredModels.length === 0 ? (
                  <div className='rounded-xl border border-white/10 bg-white/5 py-10 text-center'>
                    <div className='font-medium'>
                      {searchValue ? t('未找到匹配的模型') : t('暂无模型')}
                    </div>
                    <div className='mt-1 text-sm text-white/45'>
                      {searchValue
                        ? t('请尝试其他搜索关键词')
                        : t('您可以在上方拉取需要的模型')}
                    </div>
                  </div>
                ) : (
                  <div className='space-y-2'>
                    {filteredModels.map((model) => (
                      <div
                        key={model.id}
                        className='flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3'
                      >
                        <div className='flex min-w-0 flex-1 items-center gap-3'>
                          <Checkbox
                            checked={selectedModelIds.includes(model.id)}
                            onCheckedChange={(checked) =>
                              handleToggleModel(model.id, Boolean(checked))
                            }
                          />
                          <div className='min-w-0 flex-1'>
                            <div className='truncate font-medium text-white'>
                              {model.id}
                            </div>
                            <div className='mt-1 flex items-center gap-2 text-sm'>
                              <Badge className='border-cyan-500/20 bg-cyan-500/15 text-cyan-200'>
                                {model.owned_by || 'ollama'}
                              </Badge>
                              {model.size && (
                                <span className='text-white/45'>
                                  {formatModelSize(model.size)}
                                </span>
                              )}
                            </div>
                          </div>
                        </div>
                        <Button
                          type='button'
                          size='icon'
                          variant='ghost'
                          className='text-red-300 hover:bg-red-500/10 hover:text-red-200'
                          onClick={() => setDeleteTarget(model)}
                        >
                          <Trash2 className='h-4 w-4' />
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          <DialogFooter className='border-white/10 bg-transparent'>
            <Button type='button' onClick={onCancel}>
              {t('关闭')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent className='border-white/10 bg-black text-white'>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('确认删除模型')}</AlertDialogTitle>
            <AlertDialogDescription className='text-white/60'>
              {t('删除后无法恢复，确定要删除模型 "{{name}}" 吗？', {
                name: deleteTarget?.id || '',
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('取消')}</AlertDialogCancel>
            <AlertDialogAction
              className='bg-red-600 text-white hover:bg-red-700'
              onClick={() => {
                if (deleteTarget?.id) {
                  deleteModel(deleteTarget.id);
                }
                setDeleteTarget(null);
              }}
            >
              {t('确认')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};

export default OllamaModelModal;
