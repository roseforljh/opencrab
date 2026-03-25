import React, { useCallback, useEffect, useRef, useState } from 'react';
import { API, showError } from '../../../../helpers';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Progress,
  ProgressLabel,
  ProgressTrack,
  ProgressValue,
} from '@/components/ui/progress';

const clampPercent = (value) => {
  const v = Number(value);
  if (!Number.isFinite(v)) return 0;
  return Math.max(0, Math.min(100, v));
};

const pickStrokeColor = (percent) => {
  const p = clampPercent(percent);
  if (p >= 95) return '#ef4444';
  if (p >= 80) return '#f59e0b';
  return '#3b82f6';
};

const normalizePlanType = (value) => {
  if (value == null) return '';
  return String(value).trim().toLowerCase();
};

const getWindowDurationSeconds = (windowData) => {
  const value = Number(windowData?.limit_window_seconds);
  if (!Number.isFinite(value) || value <= 0) return null;
  return value;
};

const classifyWindowByDuration = (windowData) => {
  const seconds = getWindowDurationSeconds(windowData);
  if (seconds == null) return null;
  return seconds >= 24 * 60 * 60 ? 'weekly' : 'fiveHour';
};

const resolveRateLimitWindows = (data) => {
  const rateLimit = data?.rate_limit ?? {};
  const primary = rateLimit?.primary_window ?? null;
  const secondary = rateLimit?.secondary_window ?? null;
  const windows = [primary, secondary].filter(Boolean);
  const planType = normalizePlanType(data?.plan_type ?? rateLimit?.plan_type);

  let fiveHourWindow = null;
  let weeklyWindow = null;

  for (const windowData of windows) {
    const bucket = classifyWindowByDuration(windowData);
    if (bucket === 'fiveHour' && !fiveHourWindow) {
      fiveHourWindow = windowData;
      continue;
    }
    if (bucket === 'weekly' && !weeklyWindow) {
      weeklyWindow = windowData;
    }
  }

  if (planType === 'free') {
    if (!weeklyWindow) {
      weeklyWindow = primary ?? secondary ?? null;
    }
    return { fiveHourWindow: null, weeklyWindow };
  }

  if (!fiveHourWindow && !weeklyWindow) {
    return {
      fiveHourWindow: primary ?? null,
      weeklyWindow: secondary ?? null,
    };
  }

  if (!fiveHourWindow) {
    fiveHourWindow =
      windows.find((windowData) => windowData !== weeklyWindow) ?? null;
  }
  if (!weeklyWindow) {
    weeklyWindow =
      windows.find((windowData) => windowData !== fiveHourWindow) ?? null;
  }

  return { fiveHourWindow, weeklyWindow };
};

const formatDurationSeconds = (seconds, t) => {
  const tt = typeof t === 'function' ? t : (v) => v;
  const s = Number(seconds);
  if (!Number.isFinite(s) || s <= 0) return '-';
  const total = Math.floor(s);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) return `${hours}${tt('小时')} ${minutes}${tt('分钟')}`;
  if (minutes > 0) return `${minutes}${tt('分钟')} ${secs}${tt('秒')}`;
  return `${secs}${tt('秒')}`;
};

const formatUnixSeconds = (unixSeconds) => {
  const v = Number(unixSeconds);
  if (!Number.isFinite(v) || v <= 0) return '-';
  try {
    return new Date(v * 1000).toLocaleString();
  } catch (error) {
    return String(unixSeconds);
  }
};

const RateLimitWindowCard = ({ t, title, windowData }) => {
  const tt = typeof t === 'function' ? t : (v) => v;
  const hasWindowData =
    !!windowData &&
    typeof windowData === 'object' &&
    Object.keys(windowData).length > 0;
  const percent = clampPercent(windowData?.used_percent ?? 0);
  const resetAt = windowData?.reset_at;
  const resetAfterSeconds = windowData?.reset_after_seconds;
  const limitWindowSeconds = windowData?.limit_window_seconds;

  return (
    <div className='rounded-lg border border-semi-color-border bg-semi-color-bg-0 p-3'>
      <div className='flex items-center justify-between gap-2'>
        <div className='font-medium'>{title}</div>
        <div className='text-sm text-white/45'>
          {tt('重置时间：')}
          {formatUnixSeconds(resetAt)}
        </div>
      </div>

      {hasWindowData ? (
        <div className='mt-2'>
          <Progress value={percent} className='gap-2'>
            <div className='flex items-center gap-2'>
              <ProgressLabel className='text-sm text-white'>
                {tt('使用率')}
              </ProgressLabel>
              <ProgressValue className='text-sm text-white/60'>
                {percent}%
              </ProgressValue>
            </div>
            <ProgressTrack className='bg-white/10'>
              <div
                className='h-full rounded-full transition-all'
                style={{
                  width: `${percent}%`,
                  backgroundColor: pickStrokeColor(percent),
                }}
              />
            </ProgressTrack>
          </Progress>
        </div>
      ) : (
        <div className='mt-3 text-sm text-white/45'>-</div>
      )}

      <div className='mt-1 flex flex-wrap items-center gap-2 text-xs text-white/45'>
        <div>
          {tt('已使用：')}
          {hasWindowData ? `${percent}%` : '-'}
        </div>
        <div>
          {tt('距离重置：')}
          {hasWindowData ? formatDurationSeconds(resetAfterSeconds, tt) : '-'}
        </div>
        <div>
          {tt('窗口：')}
          {hasWindowData ? formatDurationSeconds(limitWindowSeconds, tt) : '-'}
        </div>
      </div>
    </div>
  );
};

const CodexUsageView = ({ t, record, payload, onCopy, onRefresh }) => {
  const tt = typeof t === 'function' ? t : (v) => v;
  const data = payload?.data ?? null;
  const rateLimit = data?.rate_limit ?? {};
  const { fiveHourWindow, weeklyWindow } = resolveRateLimitWindows(data);

  const allowed = !!rateLimit?.allowed;
  const limitReached = !!rateLimit?.limit_reached;
  const upstreamStatus = payload?.upstream_status;

  const statusTag =
    allowed && !limitReached ? (
      <Badge className='border-green-500/20 bg-green-500/15 text-green-200'>
        {tt('可用')}
      </Badge>
    ) : (
      <Badge className='border-red-500/20 bg-red-500/15 text-red-200'>
        {tt('受限')}
      </Badge>
    );

  const rawText =
    typeof data === 'string' ? data : JSON.stringify(data ?? payload, null, 2);

  return (
    <div className='flex flex-col gap-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='text-sm text-white/45'>
          {tt('渠道：')}
          {record?.name || '-'} ({tt('编号：')}
          {record?.id || '-'})
        </div>
        <div className='flex items-center gap-2'>
          {statusTag}
          <Button
            type='button'
            variant='secondary'
            size='sm'
            onClick={onRefresh}
          >
            {tt('刷新')}
          </Button>
        </div>
      </div>

      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='text-sm text-white/45'>
          {tt('上游状态码：')}
          {upstreamStatus ?? '-'}
        </div>
      </div>

      <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
        <RateLimitWindowCard
          t={tt}
          title={tt('5小时窗口')}
          windowData={fiveHourWindow}
        />
        <RateLimitWindowCard
          t={tt}
          title={tt('每周窗口')}
          windowData={weeklyWindow}
        />
      </div>

      <div>
        <div className='mb-1 flex items-center justify-between gap-2'>
          <div className='text-sm font-medium'>{tt('原始 JSON')}</div>
          <Button
            type='button'
            variant='secondary'
            size='sm'
            onClick={() => onCopy?.(rawText)}
            disabled={!rawText}
          >
            {tt('复制')}
          </Button>
        </div>
        <pre className='max-h-[50vh] overflow-auto rounded-lg bg-white/6 p-3 text-xs text-white/80'>
          {rawText}
        </pre>
      </div>
    </div>
  );
};

const CodexUsageLoader = ({ t, record, initialPayload, onCopy }) => {
  const tt = typeof t === 'function' ? t : (v) => v;
  const [loading, setLoading] = useState(!initialPayload);
  const [payload, setPayload] = useState(initialPayload ?? null);
  const hasShownErrorRef = useRef(false);
  const mountedRef = useRef(true);
  const recordId = record?.id;

  const fetchUsage = useCallback(async () => {
    if (!recordId) {
      if (mountedRef.current) setPayload(null);
      return;
    }

    if (mountedRef.current) setLoading(true);
    try {
      const res = await API.get(`/api/channel/${recordId}/codex/usage`, {
        skipErrorHandler: true,
      });
      if (!mountedRef.current) return;
      setPayload(res?.data ?? null);
      if (!res?.data?.success && !hasShownErrorRef.current) {
        hasShownErrorRef.current = true;
        showError(tt('获取用量失败'));
      }
    } catch (error) {
      if (!mountedRef.current) return;
      if (!hasShownErrorRef.current) {
        hasShownErrorRef.current = true;
        showError(tt('获取用量失败'));
      }
      setPayload({ success: false, message: String(error) });
    } finally {
      if (mountedRef.current) setLoading(false);
    }
  }, [recordId, tt]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    if (initialPayload) return;
    fetchUsage().catch(() => {});
  }, [fetchUsage, initialPayload]);

  if (loading) {
    return (
      <div className='flex items-center justify-center py-10'>
        <div className='text-sm text-white/60'>{tt('加载中...')}</div>
      </div>
    );
  }

  if (!payload) {
    return (
      <div className='flex flex-col gap-3'>
        <div className='text-sm text-red-300'>{tt('获取用量失败')}</div>
        <div className='flex justify-end'>
          <Button
            type='button'
            variant='secondary'
            size='sm'
            onClick={fetchUsage}
          >
            {tt('刷新')}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <CodexUsageView
      t={tt}
      record={record}
      payload={payload}
      onCopy={onCopy}
      onRefresh={fetchUsage}
    />
  );
};

export const openCodexUsageModal = ({ t, record, payload, onCopy }) => {
  const tt = typeof t === 'function' ? t : (v) => v;
  const container = document.createElement('div');
  document.body.appendChild(container);

  const close = async () => {
    const ReactDOM = await import('react-dom/client');
    const root = ReactDOM.createRoot(container);
    root.unmount();
    container.remove();
  };

  import('react-dom/client').then((ReactDOM) => {
    const root = ReactDOM.createRoot(container);
    root.render(
      <div className='fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4'>
        <div className='w-full max-w-[900px] rounded-xl border border-white/10 bg-black p-4 text-white shadow-2xl'>
          <div className='mb-4 flex items-center justify-between'>
            <div className='text-lg font-medium'>{tt('Codex 用量')}</div>
            <Button type='button' variant='secondary' size='sm' onClick={close}>
              {tt('关闭')}
            </Button>
          </div>
          <CodexUsageLoader
            t={tt}
            record={record}
            initialPayload={payload}
            onCopy={onCopy}
          />
        </div>
      </div>,
    );
  });
};
