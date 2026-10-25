/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { StatusBadge, type StatusBadgeProps } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { RefreshCw } from 'lucide-react'
import dayjs from '@/lib/dayjs'
import { formatDateTimeStr, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { SubscriptionUsageResponse } from '../../api'

type CodingPlanUsageDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelName?: string
  channelId?: number
  response: SubscriptionUsageResponse | null
  onRefresh?: () => void | Promise<void>
  isRefreshing?: boolean
}

type ZhipuUsageLimit = {
  type?: string
  unit?: number
  number?: number
  usage?: number
  currentValue?: number
  remaining?: number
  percentage?: number
  nextResetTime?: number
}

type ZhipuUsagePayload = {
  level?: string
  limits?: ZhipuUsageLimit[]
}

type OpenCodeUsageWindow = {
  status?: string
  percent?: number
  resetsAt?: string
}

type OpenCodeUsagePayload = {
  usage?: {
    rolling?: OpenCodeUsageWindow
    weekly?: OpenCodeUsageWindow
    monthly?: OpenCodeUsageWindow
  }
}

type UsageWindowView = {
  title: string
  percent: number
  resetAt: string
  usedTotal?: string
  remaining?: string
}

function clampPercent(value: unknown): number {
  const v = Number(value)
  return Number.isFinite(v) ? Math.max(0, Math.min(100, v)) : 0
}

function formatCount(value: unknown): string {
  const v = Number(value)
  return Number.isFinite(v) ? v.toLocaleString() : '-'
}

// Zhipu resets are epoch milliseconds; formatTimestampToDate expects seconds.
function formatEpochMs(value: unknown): string {
  const v = Number(value)
  if (!Number.isFinite(v) || v <= 0) {
    return '-'
  }
  return formatTimestampToDate(Math.floor(v / 1000))
}

function formatIso(value: unknown): string {
  if (typeof value !== 'string' || value.trim() === '') {
    return '-'
  }
  const d = dayjs(value)
  return d.isValid() ? formatDateTimeStr(d.toDate()) : value
}

function percentVariant(percent: number): StatusBadgeProps['variant'] {
  if (percent >= 95) {
    return 'danger'
  }
  if (percent >= 80) {
    return 'warning'
  }
  return 'info'
}

const percentTextClassName: Record<string, string> = {
  danger: 'text-destructive',
  warning: 'text-warning',
  info: 'text-info',
}

// resolveUsageWindows normalizes the two upstream payload shapes (Zhipu coding
// plan quota limits / OpenCode Go usage windows) into a common view model.
function resolveUsageWindows(
  data: Record<string, unknown> | undefined,
  t: (key: string) => string
): UsageWindowView[] {
  if (!data || typeof data !== 'object') {
    return []
  }

  const zhipu = data as unknown as ZhipuUsagePayload
  if (Array.isArray(zhipu.limits) && zhipu.limits.length > 0) {
    return zhipu.limits.map((limit) => ({
      title:
        limit.unit === 3
          ? t('5-Hour Window')
          : limit.unit === 6
            ? t('Weekly Window')
            : `${t('Window')} #${limit.unit ?? '-'}`,
      percent: clampPercent(limit.percentage),
      resetAt: formatEpochMs(limit.nextResetTime),
      usedTotal: `${formatCount(limit.currentValue)} / ${formatCount(limit.usage)}`,
      remaining: formatCount(limit.remaining),
    }))
  }

  const openCode = data as unknown as OpenCodeUsagePayload
  const windows = openCode.usage
  if (!windows || typeof windows !== 'object') {
    return []
  }
  return (
    [
      ['rolling', t('Rolling Window')],
      ['weekly', t('Weekly Window')],
      ['monthly', t('Monthly Window')],
    ] as const
  )
    .filter(([key]) => windows[key])
    .map(([key, title]) => ({
      title,
      percent: clampPercent(windows[key]?.percent),
      resetAt: formatIso(windows[key]?.resetsAt),
    }))
}

export function CodingPlanUsageDialog({
  open,
  onOpenChange,
  channelName,
  channelId,
  response,
  onRefresh,
  isRefreshing,
}: CodingPlanUsageDialogProps) {
  const { t } = useTranslation()

  const payload = useMemo(
    () => (response?.data ? (response.data as Record<string, unknown>) : null),
    [response?.data]
  )
  const windows = useMemo(
    () => resolveUsageWindows(payload ?? undefined, t),
    [payload, t]
  )
  const planLevel = (payload as ZhipuUsagePayload | null)?.level
  const channelLabel = `${channelName ?? '-'}${channelId ? ` (#${channelId})` : ''}`

  const errorMessage =
    response?.success === false
      ? response?.message?.trim() || t('Failed to fetch usage')
      : ''

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Subscription Usage')}
      contentClassName='sm:max-w-[720px]'
      contentHeight='auto'
      bodyClassName='flex flex-col gap-4'
      footer={
        <Button
          type='button'
          variant='outline'
          onClick={() => onOpenChange(false)}
        >
          {t('Close')}
        </Button>
      }
    >
      {errorMessage && (
        <div className='rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400'>
          {errorMessage}
        </div>
      )}

      <Card size='sm' className='bg-muted/30 gap-0 py-0'>
        <CardHeader className='p-4 pb-2'>
          <CardTitle className='text-muted-foreground text-xs font-medium'>
            {t('Channel')} · {channelLabel}
          </CardTitle>
          {onRefresh ? (
            <CardAction>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={onRefresh}
                disabled={Boolean(isRefreshing)}
              >
                <RefreshCw data-icon='inline-start' />
                {t('Refresh')}
              </Button>
            </CardAction>
          ) : null}
        </CardHeader>
        <CardContent className='flex flex-wrap items-center gap-2 p-4 pt-2'>
          {planLevel ? (
            <StatusBadge
              label={String(planLevel)}
              variant='purple'
              copyable={false}
            />
          ) : null}
          <StatusBadge
            label={`HTTP ${response?.upstream_status ?? '-'}`}
            variant='neutral'
            copyable={false}
          />
        </CardContent>
      </Card>

      {windows.length > 0 ? (
        <div className='grid grid-cols-1 gap-3 md:grid-cols-2'>
          {windows.map((window) => {
            const variant = percentVariant(window.percent)
            return (
              <Card key={window.title} size='sm' className='gap-0 py-0'>
                <CardHeader className='p-3 pb-2'>
                  <CardTitle className='text-sm font-semibold'>
                    {window.title}
                  </CardTitle>
                  <CardAction>
                    <div
                      className={cn(
                        'text-xl leading-none font-semibold tabular-nums',
                        percentTextClassName[variant]
                      )}
                    >
                      {window.percent}%
                    </div>
                  </CardAction>
                </CardHeader>
                <CardContent className='p-3 pt-0'>
                  <Progress
                    value={window.percent}
                    aria-label={`${window.title} usage: ${window.percent}%`}
                    className='mt-1'
                  />
                  <div className='mt-3 grid grid-cols-1 gap-2 text-xs sm:grid-cols-2'>
                    <div className='min-w-0'>
                      <div className='text-muted-foreground text-[11px]'>
                        {t('Reset at:')}
                      </div>
                      <div className='break-all tabular-nums'>
                        {window.resetAt}
                      </div>
                    </div>
                    {window.usedTotal ? (
                      <div className='min-w-0 sm:text-right'>
                        <div className='text-muted-foreground text-[11px]'>
                          {t('Used:')}
                        </div>
                        <div className='tabular-nums'>{window.usedTotal}</div>
                      </div>
                    ) : null}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      ) : (
        !errorMessage && (
          <div className='text-muted-foreground text-sm'>
            {t('Failed to fetch usage')}
          </div>
        )
      )}
    </Dialog>
  )
}
