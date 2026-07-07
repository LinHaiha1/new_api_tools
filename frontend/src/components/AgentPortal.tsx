import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Copy, CreditCard, Link as LinkIcon, Loader2, LogOut, Search, UserPlus, Users } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import { Input } from './ui/input'
import { Select } from './ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table'

const AGENT_TOKEN_KEY = 'newapi_tools_agent_token'
const AGENT_EXPIRY_KEY = 'newapi_tools_agent_token_expiry'
const AGENT_PUBLIC_BASE_URL = 'https://newapi.omgteam.me'

interface AgentUser {
  id: number
  username: string
  display_name?: string | null
  aff_code?: string | null
  invite_url?: string | null
}

interface Summary {
  total?: Metrics
  period?: Metrics
  daily_success?: DailyPoint[]
  top_users?: TopUser[]
  funnel?: Funnel
  settings?: AgentSettings
  daily_compare?: DailyCompare
}

interface AgentSettlementStats {
  summary?: Record<string, unknown>
  trend?: Array<{ label: string; commission_amount: number; settled_amount: number; pending_amount: number }>
  records?: Array<{ id: number; period_type: string; period_start: string; period_end: string; settled_amount: number; settled_at: number; status?: string; note?: string | null }>
  available_redemptions?: Array<{ id: number; name: string; quota: number; commission_amount: number; stock: number; expired_time?: number | null }>
  redemption_records?: Array<{ id: number; redemption_id: number; redemption_key: string; redemption_name: string; quota: number; commission_amount: number; status: string; created_at: number }>
}

interface TrendPoint {
  label: string
  success_money: number
  success_count: number
}

interface AgentSettings {
  announcement_html?: string
  commission_rate?: number
  default_commission_rate?: number
  second_commission_enabled?: number | boolean
  second_commission_rate?: number
  user_second_commission_enabled?: number | boolean
  user_second_commission_rate?: number
}

interface Metrics {
  invitee_count?: number
  active_count?: number
  banned_count?: number
  qualified_invitee_count?: number
  unqualified_invitee_count?: number
  paying_user_count?: number
  repeat_user_count?: number
  conversion_rate?: number
  repeat_rate?: number
  topup_count?: number
  success_money?: number
  success_count?: number
  commission_rate?: number
  commission_estimate?: number
}

interface DailyPoint {
  date: string
  success_count: number
  success_money: number
}

interface TopUser {
  user_id: number
  username?: string | null
  display_name?: string | null
  success_count: number
  success_money: number
  sub_invitees?: number
  sub_success_money?: number
}

interface Funnel {
  invitee_count?: number
  qualified_invitee_count?: number
  paying_user_count?: number
  repeat_user_count?: number
}

interface DailyCompare {
  today_success_money?: number
  yesterday_success_money?: number
  today_commission_estimate?: number
  yesterday_commission_estimate?: number
}

interface Invitee {
  id: number
  username?: string | null
  display_name?: string | null
  email?: string | null
  status: number
  request_count: number
  success_money: number
  topup_count: number
  created_at?: number | null
  sub_invitees?: number
  sub_qualified_invitees?: number
  sub_success_money?: number
  commission_rate?: number
  second_commission_rate?: number
  second_commission_estimate?: number
}

interface TopUp {
  id: number
  user_id: number
  username?: string | null
  display_name?: string | null
  trade_no?: string | null
  money: number
  payment_method?: string | null
  payment_provider?: string | null
  create_time?: number | null
  complete_time?: number | null
  status_bucket: string
}

interface PageData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

type View = 'overview' | 'invitees' | 'topups'

function asNumber(value: unknown) {
  const num = Number(value || 0)
  return Number.isFinite(num) ? num : 0
}

function formatMoney(value: unknown) {
  return `¥${asNumber(value).toFixed(2)}`
}

function formatPlainAmount(value: unknown) {
  return String(Math.round(asNumber(value)))
}

function formatTokenQuota(value: unknown) {
  return `${Math.round(asNumber(value) / 500000)} USD`
}

function formatTime(value: unknown) {
  const num = asNumber(value)
  if (!num) return '-'
  return new Date(num * 1000).toLocaleString('zh-CN', { hour12: false })
}

function decodeName(value: unknown) {
  const text = String(value || '')
  if (!text) return '-'
  try {
    if (!/^[A-Za-z0-9+/=]+$/.test(text) || text.length % 4 !== 0) return text
    const bytes = Uint8Array.from(atob(text), c => c.charCodeAt(0))
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes) || text
  } catch {
    return text
  }
}

function statusLabel(status: string) {
  if (status === 'success') return '成功'
  if (status === 'pending') return '待支付'
  if (status === 'failed') return '失败'
  if (status === 'expired') return '过期'
  return status || '未知'
}

function statusVariant(status: string): 'success' | 'warning' | 'destructive' | 'outline' {
  if (status === 'success') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'failed') return 'destructive'
  return 'outline'
}

function recent30Range() {
  const now = new Date()
  const start = new Date(now)
  start.setDate(now.getDate() - 29)
  return { start: formatDateInput(start), end: formatDateInput(now) }
}

function formatDateInput(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function AgentPortal() {
  const [token, setToken] = useState<string | null>(() => {
    const saved = localStorage.getItem(AGENT_TOKEN_KEY)
    const expiry = Number(localStorage.getItem(AGENT_EXPIRY_KEY) || 0)
    if (saved && Date.now() < expiry) return saved
    localStorage.removeItem(AGENT_TOKEN_KEY)
    localStorage.removeItem(AGENT_EXPIRY_KEY)
    return null
  })
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loginError, setLoginError] = useState('')
  const [loginLoading, setLoginLoading] = useState(false)
  const [agent, setAgent] = useState<AgentUser | null>(null)
  const [summary, setSummary] = useState<Summary | null>(null)
  const [view, setView] = useState<View>('overview')
  const [invitees, setInvitees] = useState<PageData<Invitee> | null>(null)
  const [topups, setTopups] = useState<PageData<TopUp> | null>(null)
  const [inviteePage, setInviteePage] = useState(1)
  const [topupPage, setTopupPage] = useState(1)
  const [inviteeKeyword, setInviteeKeyword] = useState('')
  const [topupKeyword, setTopupKeyword] = useState('')
  const [topupStatus, setTopupStatus] = useState('')
  const [rankLimit, setRankLimit] = useState('10')
  const [rankDays, setRankDays] = useState('30')
  const [trendMode, setTrendMode] = useState<'day' | 'week' | 'month'>('day')
  const [settlementMode, setSettlementMode] = useState<'day' | 'week' | 'month'>('day')
  const [settlementRecordLimit, setSettlementRecordLimit] = useState('10')
  const [redeemingId, setRedeemingId] = useState<number | null>(null)
  const [ranking, setRanking] = useState<TopUser[]>([])
  const [trend, setTrend] = useState<TrendPoint[]>([])
  const [settlementStats, setSettlementStats] = useState<AgentSettlementStats | null>(null)
  const [dateRange, setDateRange] = useState(() => recent30Range())
  const [loading, setLoading] = useState(false)
  const apiUrl = import.meta.env.VITE_API_URL || ''

  const authHeaders = useCallback(() => ({
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  }), [token])

  const fetchJson = useCallback(async (path: string) => {
    const response = await fetch(`${apiUrl}${path}`, { headers: authHeaders() })
    if (response.status === 401) {
      handleLogout()
      throw new Error('登录已过期')
    }
    const data = await response.json()
    if (!data.success) throw new Error(data.error?.message || data.message || '请求失败')
    return data.data
  }, [apiUrl, authHeaders])

  const loadBaseData = useCallback(async () => {
    if (!token) return
    setLoading(true)
    try {
      const [me, stats] = await Promise.all([
        fetchJson('/api/agent/me'),
        fetchJson(`/api/agent/summary?${buildDateParams(dateRange)}`),
      ])
		setAgent(normalizeAgent(me))
      setSummary(stats)
    } finally {
      setLoading(false)
    }
  }, [dateRange, fetchJson, token])

  const loadInvitees = useCallback(async () => {
    if (!token) return
    const params = new URLSearchParams({ page: String(inviteePage), page_size: '20' })
    if (inviteeKeyword.trim()) params.set('keyword', inviteeKeyword.trim())
    setInvitees(await fetchJson(`/api/agent/invitees?${params}`))
  }, [fetchJson, inviteeKeyword, inviteePage, token])

  const loadTopups = useCallback(async () => {
    if (!token) return
    const params = new URLSearchParams({ page: String(topupPage), page_size: '20' })
    appendDateParams(params, dateRange)
    if (topupKeyword.trim()) params.set('keyword', topupKeyword.trim())
    if (topupStatus) params.set('status', topupStatus)
    setTopups(await fetchJson(`/api/agent/top-ups?${params}`))
  }, [dateRange, fetchJson, token, topupKeyword, topupPage, topupStatus])

  const loadInviteeCharts = useCallback(async () => {
    if (!token) return
    const today = formatDateInput(new Date())
    const [rankData, trendData, settlementData, topupData] = await Promise.all([
      fetchJson(`/api/agent/invitee-ranking?limit=${rankLimit}&days=${rankDays}`),
      fetchJson(`/api/agent/invitee-trend?mode=${trendMode}`),
      fetchJson(`/api/agent/settlement-stats?mode=${settlementMode}&limit=${settlementRecordLimit}`),
      fetchJson(`/api/agent/top-ups?page=1&page_size=10&status=success&start_date=${today}&end_date=${today}`),
    ])
    setRanking(Array.isArray(rankData) ? rankData : [])
    setTrend(Array.isArray(trendData) ? trendData : [])
    setSettlementStats(settlementData || null)
    setTopups(topupData || null)
  }, [fetchJson, rankDays, rankLimit, settlementMode, settlementRecordLimit, token, trendMode])

  useEffect(() => { loadBaseData().catch(console.error) }, [loadBaseData])
  useEffect(() => { if (view === 'invitees') loadInvitees().catch(console.error) }, [view, loadInvitees])
  useEffect(() => { if (view === 'invitees') loadInviteeCharts().catch(console.error) }, [view, loadInviteeCharts])
  useEffect(() => { if (view === 'topups') loadTopups().catch(console.error) }, [view, loadTopups])

  async function redeemCommissionCode(id: number) {
    if (!token) return
    setRedeemingId(id)
    try {
      const response = await fetch(`${apiUrl}/api/agent/commission-redemptions/${id}/redeem`, { method: 'POST', headers: authHeaders() })
      const data = await response.json()
      if (!response.ok || !data.success) throw new Error(data.error?.message || data.message || '兑换失败')
      await loadInviteeCharts()
      await loadBaseData()
    } catch (error) {
      alert(error instanceof Error ? error.message : '兑换失败')
    } finally {
      setRedeemingId(null)
    }
  }

  async function handleLogin(e: FormEvent) {
    e.preventDefault()
    setLoginError('')
    setLoginLoading(true)
    try {
      const response = await fetch(`${apiUrl}/api/agent/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const data = await response.json()
      if (!response.ok || !data.success || !data.token) {
        setLoginError(data.message || data.error?.message || '账号或密码错误')
        return
      }
      const expiry = data.expires_at ? new Date(data.expires_at).getTime() : Date.now() + 86400 * 1000
      localStorage.setItem(AGENT_TOKEN_KEY, data.token)
      localStorage.setItem(AGENT_EXPIRY_KEY, String(expiry))
      setToken(data.token)
		setAgent(normalizeAgent(data.data))
      setPassword('')
    } catch {
      setLoginError('登录失败，请稍后重试')
    } finally {
      setLoginLoading(false)
    }
  }

  function handleLogout() {
    localStorage.removeItem(AGENT_TOKEN_KEY)
    localStorage.removeItem(AGENT_EXPIRY_KEY)
    setToken(null)
    setAgent(null)
    setSummary(null)
  }

  if (!token) {
    return (
      <div className="min-h-screen bg-gradient-to-b from-teal-50 to-background flex items-center justify-center px-4">
        <Card className="w-full max-w-md shadow-lg">
          <CardHeader>
				<CardTitle className="text-2xl">Omgt运营中心</CardTitle>
            <p className="text-sm text-muted-foreground">使用 NewAPI 账号登录，只展示你的邀请用户和充值统计。</p>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium" htmlFor="agent-username">账号</label>
                <Input id="agent-username" value={username} onChange={e => setUsername(e.target.value)} placeholder="NewAPI 用户名" autoFocus />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium" htmlFor="agent-password">密码</label>
                <Input id="agent-password" type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="NewAPI 登录密码" />
              </div>
              {loginError && <p className="text-sm text-destructive">{loginError}</p>}
              <Button type="submit" className="w-full" disabled={loginLoading}>
                {loginLoading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                登录
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-teal-50 to-background">
      <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div>
				<h1 className="text-lg font-bold">Omgt运营中心</h1>
            <p className="text-xs text-muted-foreground">{decodeName(agent?.display_name || agent?.username)} · ID {agent?.id}</p>
          </div>
          <Button variant="ghost" onClick={handleLogout}><LogOut className="h-4 w-4 mr-2" />退出</Button>
        </div>
      </header>
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
        <div className="flex gap-2 overflow-x-auto">
          {(['overview', 'invitees', 'topups'] as View[]).map(item => (
            <Button key={item} variant={view === item ? 'default' : 'outline'} onClick={() => setView(item)}>
              {item === 'overview' ? '数据概览' : item === 'invitees' ? '邀请用户' : '充值记录'}
            </Button>
          ))}
        </div>
        {loading ? <div className="py-24 flex justify-center"><Loader2 className="h-8 w-8 animate-spin" /></div> : null}
        {view === 'overview' && <Overview agent={agent} summary={summary} dateRange={dateRange} setDateRange={(next) => { setDateRange(next); setInviteePage(1); setTopupPage(1) }} />}
        {view === 'invitees' && <div className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <TopUsersRank data={ranking} showSub={isEnabled(summary?.settings?.user_second_commission_enabled)} limit={rankLimit} setLimit={setRankLimit} days={rankDays} setDays={setRankDays} />
            <TrendBars data={trend} mode={trendMode} setMode={setTrendMode} />
          </div>
          <SettlementPanel data={settlementStats} mode={settlementMode} setMode={setSettlementMode} recordLimit={settlementRecordLimit} setRecordLimit={setSettlementRecordLimit} compare={summary?.daily_compare || {}} topups={topups} onRedeem={redeemCommissionCode} redeemingId={redeemingId} />
          <InviteesView data={invitees} showSecondCommission={isEnabled(summary?.settings?.user_second_commission_enabled)} keyword={inviteeKeyword} setKeyword={setInviteeKeyword} reload={() => { setInviteePage(1); void loadInvitees() }} page={inviteePage} setPage={setInviteePage} />
        </div>}
        {view === 'topups' && <div className="space-y-4">
          <DateRangeFilter value={dateRange} onChange={(next) => { setDateRange(next); setTopupPage(1) }} />
          <TopUpsView data={topups} keyword={topupKeyword} setKeyword={setTopupKeyword} status={topupStatus} setStatus={(value) => { setTopupStatus(value); setTopupPage(1) }} reload={() => { setTopupPage(1); void loadTopups() }} page={topupPage} setPage={setTopupPage} />
        </div>}
      </main>
    </div>
  )
}

function buildDateParams(range: { start: string; end: string }) {
  const params = new URLSearchParams()
  appendDateParams(params, range)
  return params.toString()
}

function appendDateParams(params: URLSearchParams, range: { start: string; end: string }) {
  if (range.start) params.set('start_date', range.start)
  if (range.end) params.set('end_date', range.end)
}

function buildInviteLink(agent: AgentUser | null) {
  const raw = agent?.invite_url || (agent?.aff_code ? `/register?aff=${agent.aff_code}` : '')
  if (!raw) return ''
  try {
    const url = new URL(raw, AGENT_PUBLIC_BASE_URL)
    if (['localhost', '127.0.0.1', '0.0.0.0'].includes(url.hostname)) {
      return `${AGENT_PUBLIC_BASE_URL}${url.pathname}${url.search}`
    }
    return url.toString()
  } catch {
    return raw.startsWith('/register') ? `${AGENT_PUBLIC_BASE_URL}${raw}` : raw
  }
}

function normalizeAgent(agent: AgentUser | null): AgentUser | null {
  if (!agent) return null
  return { ...agent, invite_url: buildInviteLink(agent) }
}

function InviteLinkCard({ agent }: { agent: AgentUser | null }) {
  const inviteLink = buildInviteLink(agent)
  const [copied, setCopied] = useState(false)
  const handleCopy = async () => {
    if (!inviteLink) return
    await navigator.clipboard.writeText(inviteLink)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1500)
  }
  return (
    <div className="rounded-md border bg-background p-3 flex flex-col gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2 text-sm font-medium"><LinkIcon className="h-4 w-4 text-primary" />邀请链接</div>
          <div className="mt-1 text-sm text-muted-foreground break-all">{inviteLink || '当前账号暂无邀请链接'}</div>
        </div>
        <Button variant="outline" onClick={handleCopy} disabled={!inviteLink} className="shrink-0">
          <Copy className="h-4 w-4 mr-2" />{copied ? '已复制' : '复制链接'}
        </Button>
    </div>
  )
}

function Announcement({ html }: { html: string }) {
  const hasContent = html.trim().length > 0
  return (
    <Card className="h-[120px] shrink-0 border-amber-200 bg-amber-50/80">
      <CardHeader className="py-3"><CardTitle className="text-base">公告栏</CardTitle></CardHeader>
      <CardContent className="max-h-[70px] overflow-auto px-4 pb-3 pt-0 text-sm text-amber-950 [&_a]:underline [&_strong]:font-semibold">
        {hasContent ? <div dangerouslySetInnerHTML={{ __html: html }} /> : <span className="text-amber-900/60">暂无公告</span>}
      </CardContent>
    </Card>
  )
}

function DateRangeFilter({ value, onChange }: { value: { start: string; end: string }; onChange: (value: { start: string; end: string }) => void }) {
  return (
    <div className="flex items-center gap-2 flex-wrap">
      <Input type="date" value={value.start} onChange={e => onChange({ ...value, start: e.target.value })} className="w-40" />
      <span className="text-sm text-muted-foreground">至</span>
      <Input type="date" value={value.end} onChange={e => onChange({ ...value, end: e.target.value })} className="w-40" />
      <Button variant="outline" onClick={() => onChange(recent30Range())}>近30天</Button>
    </div>
  )
}

function Overview({ agent, summary, dateRange, setDateRange }: { agent: AgentUser | null; summary: Summary | null; dateRange: { start: string; end: string }; setDateRange: (value: { start: string; end: string }) => void }) {
  const total = summary?.total || {}
  const period = summary?.period || {}
  return (
    <div className="space-y-6">
      <div className="flex flex-col lg:flex-row lg:items-end lg:justify-between gap-3">
        <SectionTitle title="周期数据" subtitle={`${dateRange.start} 至 ${dateRange.end}`} />
        <DateRangeFilter value={dateRange} onChange={setDateRange} />
      </div>
      <MetricGrid cards={[
        { title: '周期成功充值', value: formatMoney(period.success_money), hint: `${asNumber(period.success_count)} 笔成功订单`, icon: CreditCard },
        { title: '周期充值用户', value: `${asNumber(period.paying_user_count)} 人`, hint: `${asNumber(period.topup_count)} 笔总订单`, icon: Users },
        { title: '周期复购用户', value: `${asNumber(period.repeat_user_count)} 人`, hint: `复购率 ${formatPercent(period.repeat_rate)}`, icon: UserPlus },
        { title: '周期佣金预估', value: formatMoney(period.commission_estimate), hint: `按 ${formatPercent(period.commission_rate)} 预估`, icon: CreditCard },
      ]} />
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 items-stretch">
        <AgentInfoPanel agent={agent} summary={summary} />
        <div className="space-y-4 h-full flex flex-col">
          <Announcement html={String(summary?.settings?.announcement_html || '')} />
          <DailyMoneyChart data={summary?.daily_success || []} />
        </div>
      </div>

      <SectionTitle title="总数据" subtitle="不受日期筛选影响" />
      <MetricGrid cards={[
        { title: '总邀请用户', value: `${asNumber(total.invitee_count)} 人`, hint: `达标 ${asNumber(total.qualified_invitee_count)} · 未达标 ${asNumber(total.unqualified_invitee_count)}`, icon: Users },
        { title: '累计成功充值', value: formatMoney(total.success_money), hint: `${asNumber(total.success_count)} 笔成功订单`, icon: CreditCard },
        { title: '充值用户', value: `${asNumber(total.paying_user_count)} 人`, hint: `转化率 ${formatPercent(total.conversion_rate)}`, icon: UserPlus },
        { title: '佣金预估', value: formatMoney(total.commission_estimate), hint: `按 ${formatPercent(total.commission_rate)} 预估`, icon: CreditCard },
      ]} />
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <InviteChart total={total} />
        <FunnelChart data={summary?.funnel || {}} />
      </div>
    </div>
  )
}

function AgentInfoPanel({ agent, summary }: { agent: AgentUser | null; summary: Summary | null }) {
  const compare = summary?.daily_compare || {}
  return (
    <Card className="h-full">
      <CardHeader><CardTitle>个人信息</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        <div className="rounded-md border bg-muted/20 p-3 text-sm space-y-1">
          <div className="font-medium">{decodeName(agent?.display_name || agent?.username)}</div>
          <div className="text-muted-foreground">用户ID：{agent?.id || '-'}</div>
        </div>
        <InviteLinkCard agent={agent} />
        <div className="grid grid-cols-2 gap-3">
          <MiniMetric title="今日充值" value={formatMoney(compare.today_success_money)} />
          <MiniMetric title="今日佣金" value={formatMoney(compare.today_commission_estimate)} />
          <MiniMetric title="昨日充值" value={formatMoney(compare.yesterday_success_money)} />
          <MiniMetric title="昨日佣金" value={formatMoney(compare.yesterday_commission_estimate)} />
        </div>
      </CardContent>
    </Card>
  )
}

function MiniMetric({ title, value }: { title: string; value: string }) {
  return <div className="rounded-md border bg-background p-3"><div className="text-xs text-muted-foreground">{title}</div><div className="text-lg font-semibold mt-1">{value}</div></div>
}

function SectionTitle({ title, subtitle }: { title: string; subtitle: string }) {
  return <div><h2 className="text-base font-semibold">{title}</h2><p className="text-sm text-muted-foreground">{subtitle}</p></div>
}

function MetricGrid({ cards }: { cards: Array<{ title: string; value: string; hint: string; icon: typeof Users }> }) {
  return <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">{cards.map(({ title, value, hint, icon: Icon }) => <Card key={title}><CardContent className="p-5"><div className="flex items-center justify-between"><p className="text-sm text-muted-foreground">{title}</p><Icon className="h-5 w-5 text-primary" /></div><div className="text-2xl font-bold mt-2">{value}</div><p className="text-xs text-muted-foreground mt-1">{hint}</p></CardContent></Card>)}</div>
}

function InviteChart({ total }: { total: Metrics }) {
  const qualified = asNumber(total.qualified_invitee_count)
  const unqualified = asNumber(total.unqualified_invitee_count)
  const all = Math.max(qualified + unqualified, 1)
  const percent = Math.round((qualified / all) * 100)
  return <Card><CardHeader><CardTitle>达标邀请率</CardTitle></CardHeader><CardContent className="flex flex-col sm:flex-row items-center gap-6"><div className="relative h-36 w-36 rounded-full" style={{ background: `conic-gradient(#10b981 0 ${percent}%, #f59e0b ${percent}% 100%)` }}><div className="absolute inset-5 rounded-full bg-background flex items-center justify-center text-xl font-bold">{percent}%</div></div><div className="space-y-3 flex-1 w-full"><Legend label="达标邀请" value={qualified} color="bg-emerald-500" /><Legend label="未达标邀请" value={unqualified} color="bg-amber-500" /><p className="text-xs text-muted-foreground">达标规则：被邀请用户累计成功充值金额大于 ¥10。</p></div></CardContent></Card>
}

function FunnelChart({ data }: { data: Funnel }) {
  const total = Math.max(asNumber(data.invitee_count), 1)
  const rows = [
    ['总邀请', asNumber(data.invitee_count), 1],
    ['达标邀请', asNumber(data.qualified_invitee_count), agentRatio(asNumber(data.qualified_invitee_count), total)],
    ['充值用户', asNumber(data.paying_user_count), agentRatio(asNumber(data.paying_user_count), total)],
    ['复购用户', asNumber(data.repeat_user_count), agentRatio(asNumber(data.repeat_user_count), total)],
  ] as const
  return <Card><CardHeader><CardTitle>转化漏斗</CardTitle></CardHeader><CardContent className="space-y-3">{rows.map(([label, value], index) => {
    const percent = rows[index][2]
    const width = `${Math.max(18, percent * 100)}%`
    return <div key={label} className="text-center"><div className="mx-auto rounded-md bg-teal-600/90 text-white py-2 text-sm font-medium" style={{ width }}>{label} · {value.toLocaleString()} · {formatPercent(percent)}</div>{index < rows.length - 1 && <div className="mx-auto h-4 w-px bg-border" />}</div>
  })}</CardContent></Card>
}

function DailyMoneyChart({ data }: { data: DailyPoint[] }) {
  const rawMax = Math.max(...data.map(row => asNumber(row.success_money)), 1)
  const max = Math.max(100, Math.ceil(rawMax / 100) * 100)
  const yTicks = Array.from({ length: Math.floor(max / 100) + 1 }, (_, i) => max - i * 100)
  const chartLeft = 3
  const chartRight = 97
  const chartTop = 10
  const chartBottom = 100
  const points = data.map((row, index) => {
    const x = data.length === 1 ? 50 : chartLeft + (index / (data.length - 1)) * (chartRight - chartLeft)
    const y = chartBottom - (asNumber(row.success_money) / max) * (chartBottom - chartTop)
    return { x, y, row }
  })
  const curve = smoothPath(points)
  const area = points.length ? `${curve} L ${points[points.length - 1].x} ${chartBottom} L ${points[0].x} ${chartBottom} Z` : ''
  return (
    <Card className="h-[480px] flex flex-col">
      <CardHeader className="pb-2 shrink-0"><CardTitle>最近10天成功充值金额</CardTitle></CardHeader>
      <CardContent className="flex-1 min-h-0 px-8 pt-0 pb-6">
        {data.length === 0 ? <EmptyChart /> : (
          <div className="h-full grid grid-cols-[48px_minmax(0,1fr)] grid-rows-[1fr_30px] gap-0 overflow-hidden">
            <div className="relative text-xs text-muted-foreground pr-1">
              <span className="absolute left-0 top-0 text-sm font-medium leading-none text-foreground">金额</span>
              {yTicks.map(tick => {
                const top = `${chartBottom - (tick / max) * (chartBottom - chartTop)}%`
                return <span key={tick} className="absolute right-1 -translate-y-1/2" style={{ top }}>{tick}</span>
              })}
            </div>
            <div className="overflow-hidden">
              <svg viewBox="0 0 100 100" preserveAspectRatio="none" className="h-full w-full block">
            <defs><linearGradient id="dailyMoney" x1="0" x2="0" y1="0" y2="1"><stop offset="0%" stopColor="#0891b2" stopOpacity="0.32" /><stop offset="100%" stopColor="#0891b2" stopOpacity="0.03" /></linearGradient></defs>
            {yTicks.map(tick => {
              const y = chartBottom - (tick / max) * (chartBottom - chartTop)
              return <line key={tick} x1={chartLeft} x2={chartRight} y1={y} y2={y} stroke="#eef4fa" strokeWidth="0.24" />
            })}
            {points.map(p => <line key={p.row.date} x1={p.x} x2={p.x} y1={chartTop} y2={chartBottom} stroke="#f1f5f9" strokeWidth="0.2" />)}
            <line x1={chartLeft} x2={chartRight} y1={chartBottom} y2={chartBottom} stroke="#e5edf6" strokeWidth="0.24" />
            <line x1={chartLeft} x2={chartLeft} y1={chartTop} y2={chartBottom} stroke="#e5edf6" strokeWidth="0.24" />
            <path d={area} fill="url(#dailyMoney)" />
            <path d={curve} fill="none" stroke="#0891b2" strokeWidth="0.9" strokeLinecap="round" />
            {points.map(p => <circle key={p.row.date} cx={p.x} cy={p.y} r="1.1" fill="#0891b2" />)}
              </svg>
            </div>
            <div />
            <div className="relative text-xs text-muted-foreground overflow-hidden"><div className="absolute inset-x-0 top-0 h-3">{points.map(p => <span key={p.row.date} className="absolute -translate-x-1/2 leading-none whitespace-nowrap" style={{ left: `${p.x}%` }}>{p.row.date.slice(5)}</span>)}</div><div className="absolute inset-x-0 bottom-0 text-center text-sm font-medium leading-none text-foreground">日期</div></div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function TopUsersRank({ data, showSub = false, limit, setLimit, days, setDays }: { data: TopUser[]; showSub?: boolean; limit?: string; setLimit?: (v: string) => void; days?: string; setDays?: (v: string) => void }) {
  const max = Math.max(...data.map(row => asNumber(row.success_money)), 1)
  return (
    <Card>
      <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
        <CardTitle>充值用户排行</CardTitle>
        {setLimit && setDays && <div className="flex gap-2"><Select value={limit} onChange={e => setLimit(e.target.value)} className="w-24"><option value="5">前5</option><option value="10">前10</option><option value="20">前20</option></Select><Select value={days} onChange={e => setDays(e.target.value)} className="w-28"><option value="1">今日</option><option value="7">近7天</option><option value="15">近15天</option><option value="30">近30天</option><option value="90">近90天</option></Select></div>}
      </CardHeader>
      <CardContent>{data.length === 0 ? <EmptyChart /> : <div className="space-y-3">{data.map((row, index) => <RankRow key={row.user_id} index={index} name={decodeName(row.display_name || row.username || row.user_id)} subInvitees={asNumber(row.sub_invitees)} subMoney={asNumber(row.sub_success_money)} money={asNumber(row.success_money)} max={max} showSub={showSub} />)}</div>}</CardContent>
    </Card>
  )
}

function SettlementPanel({ data, mode, setMode, recordLimit, setRecordLimit, compare, topups, onRedeem, redeemingId }: { data: AgentSettlementStats | null; mode: 'day' | 'week' | 'month'; setMode: (v: 'day' | 'week' | 'month') => void; recordLimit: string; setRecordLimit: (v: string) => void; compare: DailyCompare; topups: PageData<TopUp> | null; onRedeem: (id: number) => void; redeemingId: number | null }) {
  const summary = data?.summary || {}
  const trend = Array.isArray(data?.trend) ? data.trend : []
  const records = Array.isArray(data?.records) ? data.records : []
  const availableRedemptions = Array.isArray(data?.available_redemptions) ? data.available_redemptions : []
  const redemptionRecords = Array.isArray(data?.redemption_records) ? data.redemption_records : []
  const max = Math.max(...trend.map(row => asNumber(row.commission_amount)), 1)
  const confirmRedeem = (row: NonNullable<AgentSettlementStats['available_redemptions']>[number]) => {
    const confirmed = window.confirm(
      `确认兑换 ${row.name}？\n\n额度：${formatTokenQuota(row.quota)}\n消耗佣金：${formatMoney(row.commission_amount)}\n\n确认后将扣减可用佣金并生成兑换码。`
    )
    if (confirmed) onRedeem(row.id)
  }
  return (
    <Card>
      <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
        <CardTitle>佣金结算趋势</CardTitle>
        <div className="flex gap-2"><Button size="sm" variant={mode === 'day' ? 'default' : 'outline'} onClick={() => setMode('day')}>日</Button><Button size="sm" variant={mode === 'week' ? 'default' : 'outline'} onClick={() => setMode('week')}>周</Button><Button size="sm" variant={mode === 'month' ? 'default' : 'outline'} onClick={() => setMode('month')}>月</Button></div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-3"><MiniMetric title="累计佣金" value={formatMoney(summary.total_commission_estimate)} /><MiniMetric title="已结算金额" value={formatMoney(summary.settled_amount)} /><MiniMetric title="已兑换佣金" value={formatMoney(summary.redeemed_amount)} /><MiniMetric title="可用佣金" value={formatMoney(summary.pending_amount)} /></div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div className="rounded-md border overflow-hidden"><Table><TableHeader><TableRow><TableHead>佣金兑换</TableHead><TableHead className="text-right">库存</TableHead><TableHead className="text-right">消耗佣金</TableHead><TableHead className="text-right">操作</TableHead></TableRow></TableHeader><TableBody>{availableRedemptions.map(row => <TableRow key={`${row.name}-${row.quota}`}><TableCell><div className="font-medium">{row.name}</div><div className="text-xs text-muted-foreground">额度 {formatTokenQuota(row.quota)}</div></TableCell><TableCell className="text-right">{asNumber(row.stock)}</TableCell><TableCell className="text-right">{formatMoney(row.commission_amount)}</TableCell><TableCell className="text-right"><Button size="sm" variant="outline" disabled={redeemingId === row.id || asNumber(summary.pending_amount) < asNumber(row.commission_amount)} onClick={() => confirmRedeem(row)}>{redeemingId === row.id ? '兑换中' : '兑换'}</Button></TableCell></TableRow>)}{availableRedemptions.length === 0 && <TableRow><TableCell colSpan={4} className="text-center text-muted-foreground py-8">暂无可兑换的 yj 兑换码</TableCell></TableRow>}</TableBody></Table></div>
          <div className="rounded-md border overflow-hidden"><Table><TableHeader><TableRow><TableHead>兑换记录</TableHead><TableHead className="text-right">消耗佣金</TableHead><TableHead>时间</TableHead></TableRow></TableHeader><TableBody>{redemptionRecords.map(row => <TableRow key={row.id}><TableCell><div className="font-medium">{row.redemption_name}</div><div className="text-xs text-muted-foreground break-all">{row.redemption_key}</div></TableCell><TableCell className="text-right">{formatMoney(row.commission_amount)}</TableCell><TableCell>{formatTime(row.created_at)}</TableCell></TableRow>)}{redemptionRecords.length === 0 && <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground py-8">暂无兑换记录</TableCell></TableRow>}</TableBody></Table></div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div className="h-60 rounded-md border bg-background/60 p-3 flex items-end gap-2">
            {trend.length === 0 ? <EmptyChart /> : trend.map(row => {
              const commission = asNumber(row.commission_amount)
              const settled = Math.min(asNumber(row.settled_amount), commission)
              const pending = Math.max(0, commission - settled)
              return <div key={row.label} className="flex-1 min-w-0 flex flex-col items-center gap-2"><span className="text-[10px] text-muted-foreground whitespace-nowrap">{formatMoney(commission)}</span><div className="w-full h-36 flex flex-col justify-end rounded-t overflow-hidden bg-muted" title={`总佣金 ${formatMoney(commission)} · 已结算 ${formatMoney(settled)} · 待结算 ${formatMoney(pending)}`}><div className="w-full bg-emerald-600" style={{ height: `${Math.max(commission ? 3 : 0, (settled / max) * 144)}px` }} /><div className="w-full bg-amber-500" style={{ height: `${Math.max(commission ? 3 : 0, (pending / max) * 144)}px` }} /></div><div className="text-[10px] text-muted-foreground truncate w-full text-center">{String(row.label).slice(-5)}</div></div>
            })}
          </div>
          <div className="rounded-md border overflow-hidden">
            <Table><TableHeader><TableRow><TableHead>今日充值记录</TableHead><TableHead className="text-right">金额</TableHead><TableHead>时间</TableHead></TableRow></TableHeader><TableBody>{(topups?.items || []).slice(0, 10).map(row => <TableRow key={row.id}><TableCell><div className="font-medium">{decodeName(row.display_name || row.username)}</div><div className="text-xs text-muted-foreground">ID {row.user_id}</div></TableCell><TableCell className="text-right">{formatMoney(row.money)}</TableCell><TableCell>{formatTime(row.complete_time || row.create_time)}</TableCell></TableRow>)}{(!topups || topups.items.length === 0) && <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground py-8">暂无今日充值记录</TableCell></TableRow>}</TableBody></Table>
          </div>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3"><MiniMetric title="今日充值用户佣金" value={formatMoney(compare.today_commission_estimate)} /><MiniMetric title="昨日充值用户佣金" value={formatMoney(compare.yesterday_commission_estimate)} /></div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3"><MiniMetric title={`佣金结算周期详情（${String(summary.period_label || '')}）`} value={formatMoney(summary.period_commission_estimate)} /><MiniMetric title="周期已结算" value={formatMoney(summary.period_settled_amount)} /><MiniMetric title="周期未结算" value={formatMoney(summary.period_pending_amount)} /></div>
        <div className="space-y-3"><div className="flex items-center justify-between"><div className="text-sm font-semibold">结算记录</div><Select value={recordLimit} onChange={e => setRecordLimit(e.target.value)} className="w-28"><option value="10">10条</option><option value="50">50条</option><option value="100">100条</option></Select></div><div className="rounded-md border overflow-hidden"><Table><TableHeader><TableRow><TableHead>周期</TableHead><TableHead className="text-right">金额</TableHead><TableHead>状态</TableHead><TableHead>时间</TableHead></TableRow></TableHeader><TableBody>{records.map(row => <TableRow key={row.id}><TableCell>{row.period_start} 至 {row.period_end}</TableCell><TableCell className="text-right">{formatMoney(row.settled_amount)}</TableCell><TableCell>{row.status === 'unsettled' ? '未结算' : '已结算'}</TableCell><TableCell>{formatTime(row.settled_at)}</TableCell></TableRow>)}{records.length === 0 && <TableRow><TableCell colSpan={4} className="text-center text-muted-foreground py-8">暂无结算记录</TableCell></TableRow>}</TableBody></Table></div></div>
      </CardContent>
    </Card>
  )
}

function RankRow({ index, name, subInvitees, subMoney, money, max, showSub }: { index: number; name: string; subInvitees: number; subMoney: number; money: number; max: number; showSub: boolean }) {
  const styles = ['bg-amber-500 text-white', 'bg-slate-400 text-white', 'bg-orange-500 text-white']
  const badge = index < 3 ? styles[index] : 'bg-muted text-muted-foreground'
  return <div className="space-y-1"><div className="flex items-center justify-between gap-3 text-sm"><span className="min-w-0 flex items-center gap-2"><span className={`h-7 w-10 rounded-full flex items-center justify-center text-xs font-bold ${badge}`}>TOP{index + 1}</span><span className="truncate font-medium">{name}</span>{showSub && <span className="text-xs text-muted-foreground shrink-0">邀请{subInvitees}人 · 下级{formatMoney(subMoney)}</span>}</span><span className="font-semibold shrink-0">{formatMoney(money)}</span></div><div className="h-2 rounded-full bg-muted overflow-hidden"><div className="h-full rounded-full bg-indigo-600" style={{ width: `${Math.max(4, Math.min(100, (money / max) * 100))}%` }} /></div></div>
}

function TrendBars({ data, mode, setMode }: { data: TrendPoint[]; mode: 'day' | 'week' | 'month'; setMode: (v: 'day' | 'week' | 'month') => void }) {
  const max = Math.max(...data.map(row => asNumber(row.success_money)), 1)
  const barWidth = mode === 'day' ? 34 : 52
  return (
    <Card>
      <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
        <CardTitle>充值金额趋势</CardTitle>
        <div className="flex gap-2"><Button size="sm" variant={mode === 'day' ? 'default' : 'outline'} onClick={() => setMode('day')}>日</Button><Button size="sm" variant={mode === 'week' ? 'default' : 'outline'} onClick={() => setMode('week')}>周</Button><Button size="sm" variant={mode === 'month' ? 'default' : 'outline'} onClick={() => setMode('month')}>月</Button></div>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? <EmptyChart /> : (
          <div className="h-72 grid grid-cols-[42px_1fr] gap-0 overflow-hidden rounded-md border bg-background/60">
            <div className="flex flex-col justify-between text-xs text-muted-foreground pt-8 pb-10 pr-2 text-right border-r border-border/70 bg-background"><span>{formatPlainAmount(max)}</span><span>{formatPlainAmount(max / 2)}</span><span>0</span></div>
            <div className="overflow-x-auto overflow-y-hidden">
              <div className="h-full flex items-end gap-3 border-b border-border/70 px-3 pt-4 pb-6" style={{ minWidth: `${Math.max(360, data.length * (barWidth + 12))}px` }}>
                {data.map(row => <div key={row.label} className="flex flex-col items-center gap-2 shrink-0" style={{ width: `${barWidth}px` }}><span className="text-[10px] text-muted-foreground whitespace-nowrap">{formatPlainAmount(row.success_money)}</span><div className="w-full rounded-t bg-teal-600" style={{ height: `${Math.max(4, (asNumber(row.success_money) / max) * 180)}px` }} title={formatMoney(row.success_money)} /><div className="text-[10px] text-muted-foreground truncate w-full text-center">{String(row.label).slice(-5)}</div></div>)}
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function Legend({ label, value, color }: { label: string; value: number; color: string }) {
  return <div className="flex items-center justify-between gap-4 text-sm"><span className="flex items-center gap-2"><span className={`h-2.5 w-2.5 rounded-full ${color}`} />{label}</span><span className="font-semibold">{value.toLocaleString()}</span></div>
}

function EmptyChart() {
  return <div className="h-32 flex items-center justify-center text-sm text-muted-foreground">当前范围暂无数据</div>
}

function formatPercent(value: unknown) {
  return `${(asNumber(value) * 100).toFixed(1)}%`
}

function isEnabled(value: unknown) {
  return value === true || Number(value || 0) === 1
}

function agentRatio(numerator: number, denominator: number) {
  return denominator > 0 ? numerator / denominator : 0
}

function smoothPath(points: Array<{ x: number; y: number }>) {
  if (points.length === 0) return ''
  if (points.length === 1) return `M ${points[0].x} ${points[0].y}`
  let d = `M ${points[0].x} ${points[0].y}`
  for (let i = 0; i < points.length - 1; i++) {
    const current = points[i]
    const next = points[i + 1]
    const midX = (current.x + next.x) / 2
    d += ` C ${midX} ${current.y}, ${midX} ${next.y}, ${next.x} ${next.y}`
  }
  return d
}

function InviteesView({ data, showSecondCommission, keyword, setKeyword, reload, page, setPage }: { data: PageData<Invitee> | null; showSecondCommission: boolean; keyword: string; setKeyword: (v: string) => void; reload: () => void; page: number; setPage: (v: number) => void }) {
  return (
    <Card>
      <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
        <CardTitle>邀请用户</CardTitle>
        <div className="flex gap-2">
          <Input value={keyword} onChange={e => setKeyword(e.target.value)} placeholder="搜索用户名、昵称、邮箱" className="w-64" />
          <Button variant="outline" onClick={reload}><Search className="h-4 w-4 mr-2" />搜索</Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>用户</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">充值笔数</TableHead>
                <TableHead className="text-right">成功充值</TableHead>
                {showSecondCommission && <TableHead className="text-right">下级成功充值</TableHead>}
                {showSecondCommission && <TableHead className="text-right">二次佣金比例</TableHead>}
                {showSecondCommission && <TableHead className="text-right">他的邀请佣金</TableHead>}
                <TableHead>注册时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(data?.items || []).map(u => (
                <TableRow key={u.id}>
                  <TableCell>{u.id}</TableCell>
                  <TableCell><div className="font-medium">{decodeName(u.display_name || u.username)}</div><div className="text-xs text-muted-foreground">{u.email || u.username || '-'}</div></TableCell>
                  <TableCell><Badge variant={u.status === 1 ? 'success' : 'destructive'}>{u.status === 1 ? '正常' : '异常'}</Badge></TableCell>
                  <TableCell className="text-right">{asNumber(u.topup_count).toLocaleString()}</TableCell>
                  <TableCell className="text-right">{formatMoney(u.success_money)}</TableCell>
                  {showSecondCommission && <TableCell className="text-right">{formatMoney(u.sub_success_money)}</TableCell>}
                  {showSecondCommission && <TableCell className="text-right">{formatPercent(u.second_commission_rate)}</TableCell>}
                  {showSecondCommission && <TableCell className="text-right">{formatMoney(u.second_commission_estimate)}</TableCell>}
                  <TableCell>{formatTime(u.created_at)}</TableCell>
                </TableRow>
              ))}
              {(!data || data.items.length === 0) && <TableRow><TableCell colSpan={showSecondCommission ? 9 : 6} className="text-center text-muted-foreground py-10">暂无邀请用户</TableCell></TableRow>}
            </TableBody>
          </Table>
        </div>
        <Pager page={page} totalPages={data?.total_pages || 1} total={data?.total || 0} setPage={setPage} />
      </CardContent>
    </Card>
  )
}

function TopUpsView({ data, keyword, setKeyword, status, setStatus, reload, page, setPage }: { data: PageData<TopUp> | null; keyword: string; setKeyword: (v: string) => void; status: string; setStatus: (v: string) => void; reload: () => void; page: number; setPage: (v: number) => void }) {
  return <Card><CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between"><CardTitle>充值记录</CardTitle><div className="flex gap-2 flex-wrap"><Select value={status} onChange={e => setStatus(e.target.value)} className="w-32"><option value="">全部状态</option><option value="success">成功</option><option value="pending">待支付</option><option value="failed">失败</option><option value="expired">过期</option><option value="unknown">未知</option></Select><Input value={keyword} onChange={e => setKeyword(e.target.value)} placeholder="搜索用户或交易号" className="w-64" /><Button variant="outline" onClick={reload}><Search className="h-4 w-4 mr-2" />搜索</Button></div></CardHeader><CardContent><div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>交易号</TableHead><TableHead>用户</TableHead><TableHead>状态</TableHead><TableHead className="text-right">金额</TableHead><TableHead>支付方式</TableHead><TableHead>创建时间</TableHead><TableHead>完成时间</TableHead></TableRow></TableHeader><TableBody>{(data?.items || []).map(t => <TableRow key={t.id}><TableCell className="font-mono text-xs">{t.trade_no || '-'}</TableCell><TableCell><div className="font-medium">{decodeName(t.display_name || t.username)}</div><div className="text-xs text-muted-foreground">ID {t.user_id}</div></TableCell><TableCell><Badge variant={statusVariant(t.status_bucket)}>{statusLabel(t.status_bucket)}</Badge></TableCell><TableCell className="text-right">{formatMoney(t.money)}</TableCell><TableCell>{t.payment_provider || t.payment_method || '-'}</TableCell><TableCell>{formatTime(t.create_time)}</TableCell><TableCell>{formatTime(t.complete_time)}</TableCell></TableRow>)}{(!data || data.items.length === 0) && <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground py-10">暂无充值记录</TableCell></TableRow>}</TableBody></Table></div><Pager page={page} totalPages={data?.total_pages || 1} total={data?.total || 0} setPage={setPage} /></CardContent></Card>
}

function Pager({ page, totalPages, total, setPage }: { page: number; totalPages: number; total: number; setPage: (v: number) => void }) {
  return <div className="flex justify-end items-center gap-3 mt-4 text-sm text-muted-foreground"><Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button><span>第 {page} / {totalPages} 页，共 {total} 条</span><Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button></div>
}

export default AgentPortal
