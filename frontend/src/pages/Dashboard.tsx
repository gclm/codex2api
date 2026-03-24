import type { ReactNode } from 'react'
import { useCallback } from 'react'
import { api } from '../api'
import PageHeader from '../components/PageHeader'
import StateShell from '../components/StateShell'
import StatCard from '../components/StatCard'
import type { StatsResponse, UsageStats } from '../types'
import { useDataLoader } from '../hooks/useDataLoader'
import { Card, CardContent } from '@/components/ui/card'
import { Users, CheckCircle, XCircle, Activity, Zap, Clock, AlertTriangle, BarChart3, Database } from 'lucide-react'

export default function Dashboard() {
  const loadDashboardData = useCallback(async () => {
    const [stats, usageStats] = await Promise.all([api.getStats(), api.getUsageStats()])
    return { stats, usageStats }
  }, [])

  const { data, loading, error, reload } = useDataLoader<{
    stats: StatsResponse | null
    usageStats: UsageStats | null
  }>({
    initialData: { stats: null, usageStats: null },
    load: loadDashboardData,
  })

  const { stats, usageStats } = data
  const total = stats?.total ?? 0
  const available = stats?.available ?? 0
  const errorCount = stats?.error ?? 0
  const todayRequests = stats?.today_requests ?? 0

  const icons: Record<string, ReactNode> = {
    total: <Users className="size-[22px]" />,
    available: <CheckCircle className="size-[22px]" />,
    error: <XCircle className="size-[22px]" />,
    requests: <Activity className="size-[22px]" />,
  }

  return (
    <StateShell
      variant="page"
      loading={loading}
      error={error}
      onRetry={() => void reload()}
      loadingTitle="正在加载仪表盘"
      loadingDescription="系统统计和使用数据正在同步。"
      errorTitle="仪表盘加载失败"
    >
      <>
        <PageHeader
          title="仪表盘"
          description="系统概览"
          onRefresh={() => void reload()}
        />

        {/* 账号状态 */}
        <div className="grid grid-cols-[repeat(auto-fit,minmax(220px,1fr))] gap-4 mb-6">
          <StatCard icon={icons.total} iconClass="blue" label="总账号" value={total} />
          <StatCard
            icon={icons.available}
            iconClass="green"
            label="可用"
            value={available}
            sub={`${total ? Math.round((available / total) * 100) : 0}% 可用率`}
          />
          <StatCard icon={icons.error} iconClass="red" label="异常" value={errorCount} />
          <StatCard icon={icons.requests} iconClass="purple" label="今日请求" value={todayRequests} />
        </div>

        {/* 使用统计 */}
        {usageStats && (
          <Card>
            <CardContent className="p-6">
              <h3 className="text-base font-semibold text-foreground mb-4">使用统计</h3>
              <div className="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4">
                <StatItem
                  icon={<BarChart3 className="size-5" />}
                  iconBg="bg-blue-500/10 text-blue-500"
                  label="总请求"
                  value={usageStats.total_requests.toLocaleString()}
                />
                <StatItem
                  icon={<Zap className="size-5" />}
                  iconBg="bg-purple-500/10 text-purple-500"
                  label="总 Token"
                  value={usageStats.total_tokens.toLocaleString()}
                />
                <StatItem
                  icon={<Zap className="size-5" />}
                  iconBg="bg-emerald-500/10 text-emerald-500"
                  label="今日 Token"
                  value={usageStats.today_tokens.toLocaleString()}
                />
                <StatItem
                  icon={<Database className="size-5" />}
                  iconBg="bg-indigo-500/10 text-indigo-500"
                  label="缓存读取 Token"
                  value={usageStats.total_cached_tokens.toLocaleString()}
                />
                <StatItem
                  icon={<Activity className="size-5" />}
                  iconBg="bg-amber-500/10 text-amber-500"
                  label="RPM / TPM"
                  value={`${usageStats.rpm} / ${usageStats.tpm.toLocaleString()}`}
                />
                <StatItem
                  icon={<Clock className="size-5" />}
                  iconBg="bg-cyan-500/10 text-cyan-500"
                  label="平均延迟"
                  value={
                    usageStats.avg_duration_ms > 1000
                      ? `${(usageStats.avg_duration_ms / 1000).toFixed(1)}s`
                      : `${Math.round(usageStats.avg_duration_ms)}ms`
                  }
                />
                <StatItem
                  icon={<AlertTriangle className="size-5" />}
                  iconBg="bg-red-500/10 text-red-500"
                  label="今日错误率"
                  value={`${usageStats.error_rate.toFixed(1)}%`}
                />
              </div>
            </CardContent>
          </Card>
        )}
      </>
    </StateShell>
  )
}

function StatItem({ icon, iconBg, label, value }: { icon: ReactNode; iconBg: string; label: string; value: string }) {
  return (
    <div className="flex items-center gap-3 p-4 rounded-xl bg-muted/50">
      <div className={`flex items-center justify-center size-10 rounded-lg ${iconBg}`}>
        {icon}
      </div>
      <div>
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="text-lg font-bold">{value}</div>
      </div>
    </div>
  )
}
