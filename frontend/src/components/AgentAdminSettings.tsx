import { useCallback, useEffect, useState } from 'react'
import { BarChart3, Megaphone, Pencil, Plus, Save, Trash2 } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Select } from './ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table'

interface RateRow {
  user_id: number
  username?: string | null
  display_name?: string | null
  commission_rate: number
  second_commission_enabled?: number
  second_commission_rate?: number
  updated_at: number
}

interface CommissionAgentRow {
  user_id: number
  username?: string | null
  display_name?: string | null
  power_agent?: number
  commission_rate?: number
  second_commission_enabled?: number
  second_commission_rate?: number
  invitee_count?: number
  period_success_money?: number
  period_success_count?: number
  period_first_commission?: number
  period_second_commission?: number
  period_commission_estimate?: number
  period_settled_amount?: number
  period_redeemed_amount?: number
  period_pending_amount?: number
  total_success_money?: number
  total_success_count?: number
  total_commission_estimate?: number
  settled_amount?: number
  redeemed_amount?: number
  pending_amount?: number
  last_settled_at?: number
}

interface CommissionStats {
  summary?: Record<string, unknown>
  agents?: CommissionAgentRow[]
}

interface SettlementRecord {
  id: number
  user_id: number
  username?: string | null
  display_name?: string | null
  period_type?: string
  period_start?: string
  period_end?: string
  settled_amount?: number
  status?: string
  settled_at?: number
  note?: string | null
}

interface CommissionRedemptionRecord {
  id: number
  user_id: number
  username?: string | null
  display_name?: string | null
  redemption_id: number
  redemption_key?: string | null
  redemption_name?: string | null
  quota?: number
  commission_amount?: number
  status?: string
  created_at?: number
}

type AdminPanel = 'settings' | 'commission' | 'settlement' | 'redemption'
type CommissionMode = 'day' | 'week' | 'month'
type SettlementFilterMode = 'all' | 'day' | 'week' | 'month' | 'custom'
type SettlementAgentFilter = 'all' | 'settled' | 'pending' | 'power'

export function AgentAdminSettings() {
  const { token } = useAuth()
  const [announcementHtml, setAnnouncementHtml] = useState('')
  const [defaultRate, setDefaultRate] = useState('0.1')
  const [secondRate, setSecondRate] = useState('0.03')
  const [redemptionQuotaUnit, setRedemptionQuotaUnit] = useState('10')
  const [redemptionCommissionUnit, setRedemptionCommissionUnit] = useState('3')
  const [rates, setRates] = useState<RateRow[]>([])
  const [userId, setUserId] = useState('')
  const [rate, setRate] = useState('0.1')
  const [userSecondEnabled, setUserSecondEnabled] = useState(false)
  const [userSecondRate, setUserSecondRate] = useState('0.03')
  const [commissionStats, setCommissionStats] = useState<CommissionStats | null>(null)
  const [settlementRecords, setSettlementRecords] = useState<SettlementRecord[]>([])
  const [redemptionRecords, setRedemptionRecords] = useState<CommissionRedemptionRecord[]>([])
  const [activePanel, setActivePanel] = useState<AdminPanel>('settings')
  const [commissionMode, setCommissionMode] = useState<CommissionMode>('day')
  const [settlementFilter, setSettlementFilter] = useState<SettlementFilterMode>('all')
  const [settlementAgentFilter, setSettlementAgentFilter] = useState<SettlementAgentFilter>('all')
  const [agentKeyword, setAgentKeyword] = useState('')
  const [agentPage, setAgentPage] = useState(1)
  const [settlementKeyword, setSettlementKeyword] = useState('')
  const [settlementPage, setSettlementPage] = useState(1)
  const [recordKeyword, setRecordKeyword] = useState('')
  const [recordPage, setRecordPage] = useState(1)
  const [recordPageSize, setRecordPageSize] = useState('20')
  const [redemptionKeyword, setRedemptionKeyword] = useState('')
  const [redemptionPage, setRedemptionPage] = useState(1)
  const [commissionRange, setCommissionRange] = useState(() => todayRange())
  const [saving, setSaving] = useState(false)
  const [deletingSettlementId, setDeletingSettlementId] = useState<number | null>(null)
  const apiUrl = import.meta.env.VITE_API_URL || ''

  const headers = useCallback(() => ({
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  }), [token])

  const load = useCallback(async () => {
    const res = await fetch(`${apiUrl}/api/agent-admin/settings`, { headers: headers() })
    const data = await res.json()
    if (data.success) {
      setAnnouncementHtml(String(data.data?.settings?.announcement_html || ''))
      setDefaultRate(String(Number(data.data?.settings?.default_commission_rate || 0.1)))
      setSecondRate(String(Number(data.data?.settings?.second_commission_rate || 0.03)))
      setRedemptionQuotaUnit(String(Number(data.data?.settings?.redemption_quota_unit || 5000000) / 500000))
      setRedemptionCommissionUnit(String(Number(data.data?.settings?.redemption_commission_unit || 3)))
      setRates(Array.isArray(data.data?.rates) ? data.data.rates : [])
      setCommissionStats(data.data?.commission_stats || null)
    }
  }, [apiUrl, headers])

  const loadCommissionStats = useCallback(async (mode: CommissionMode, range = commissionRange) => {
    const params = new URLSearchParams({ mode })
    if (range.start && range.end) {
      params.set('start_date', range.start)
      params.set('end_date', range.end)
    }
    const res = await fetch(`${apiUrl}/api/agent-admin/commission-stats?${params}`, { headers: headers() })
    const data = await res.json()
    if (data.success) setCommissionStats(data.data || null)
  }, [apiUrl, commissionRange, headers])

  const loadSettlementRecords = useCallback(async (mode = settlementFilter) => {
    const params = new URLSearchParams({ mode })
    const res = await fetch(`${apiUrl}/api/agent-admin/settlements?${params}`, { headers: headers() })
    const data = await res.json()
    if (data.success) setSettlementRecords(Array.isArray(data.data) ? data.data : [])
  }, [apiUrl, headers, settlementFilter])

  const loadRedemptionRecords = useCallback(async () => {
    const res = await fetch(`${apiUrl}/api/agent-admin/commission-redemptions?limit=500`, { headers: headers() })
    const data = await res.json()
    if (data.success) setRedemptionRecords(Array.isArray(data.data) ? data.data : [])
  }, [apiUrl, headers])

  useEffect(() => { void load() }, [load])
  useEffect(() => { if (activePanel === 'commission') void loadCommissionStats(commissionMode, commissionRange) }, [activePanel, commissionMode, commissionRange, loadCommissionStats])
  useEffect(() => { if (activePanel === 'settlement') { void loadCommissionStats(commissionMode, commissionRange); void loadSettlementRecords(settlementFilter) } }, [activePanel, commissionMode, commissionRange, settlementFilter, loadCommissionStats, loadSettlementRecords])
  useEffect(() => { if (activePanel === 'redemption') void loadRedemptionRecords() }, [activePanel, loadRedemptionRecords])
  useEffect(() => { setAgentPage(1) }, [agentKeyword, commissionStats])
  useEffect(() => { setSettlementPage(1) }, [settlementKeyword, settlementAgentFilter, commissionStats])
  useEffect(() => { setRecordPage(1) }, [recordKeyword, recordPageSize, settlementFilter, settlementRecords])
  useEffect(() => { setRedemptionPage(1) }, [redemptionKeyword, redemptionRecords])

  const saveSettings = async () => {
    setSaving(true)
    try {
      await fetch(`${apiUrl}/api/agent-admin/settings`, {
        method: 'POST',
        headers: headers(),
        body: JSON.stringify({
          announcement_html: announcementHtml,
          default_commission_rate: Number(defaultRate) || 0,
          second_commission_enabled: false,
          second_commission_rate: Number(secondRate) || 0.03,
          redemption_quota_unit: Number(redemptionQuotaUnit) || 10,
          redemption_commission_unit: Number(redemptionCommissionUnit) || 3,
        }),
      })
      await load()
    } finally {
      setSaving(false)
    }
  }

  const editRate = (row: RateRow) => {
    setUserId(String(row.user_id))
    setRate(String(Number(row.commission_rate || 0)))
    setUserSecondEnabled(Number(row.second_commission_enabled || 0) === 1)
    setUserSecondRate(String(Number(row.second_commission_rate || 0.03)))
  }

  const saveRate = async () => {
    if (!userId) return
    await fetch(`${apiUrl}/api/agent-admin/rates`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ user_id: Number(userId), commission_rate: Number(rate) || 0, second_commission_enabled: userSecondEnabled, second_commission_rate: Number(userSecondRate) || 0.03 }),
    })
    setUserId('')
    setUserSecondEnabled(false)
    setUserSecondRate('0.03')
    await load()
    await loadCommissionStats(commissionMode, commissionRange)
  }

  const deleteRate = async (id: number) => {
    await fetch(`${apiUrl}/api/agent-admin/rates/${id}`, { method: 'DELETE', headers: headers() })
    await load()
    await loadCommissionStats(commissionMode, commissionRange)
  }

  const settleAgent = async (row: CommissionAgentRow) => {
    const amount = Number(row.period_commission_estimate || 0)
    if (!row.user_id || amount <= 0) return
    if (!window.confirm(`确认结算 ${row.display_name || row.username || row.user_id} 的 ${formatAdminMoney(amount)} 吗？`)) return
    await fetch(`${apiUrl}/api/agent-admin/settlements`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({
        user_id: row.user_id,
        period_type: String(statsSummary.period_mode || commissionMode),
        period_start: String(statsSummary.period_start || commissionRange.start),
        period_end: String(statsSummary.period_end || commissionRange.end),
        settled_amount: amount,
        note: `佣金统计页${String(statsSummary.period_label || '')}结算`,
      }),
    })
    await loadCommissionStats(commissionMode, commissionRange)
    await loadSettlementRecords(settlementFilter)
  }

  const updateSettlementStatus = async (id: number, status: string) => {
    await fetch(`${apiUrl}/api/agent-admin/settlements/${id}/status`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ status }),
    })
    await loadCommissionStats(commissionMode, commissionRange)
    await loadSettlementRecords(settlementFilter)
  }

  const deleteSettlement = async (id: number) => {
    if (!window.confirm('确认删除这条结算记录吗？删除后会重新计算已结算和待结算金额。')) return
    setDeletingSettlementId(id)
    try {
      await fetch(`${apiUrl}/api/agent-admin/settlements/${id}`, { method: 'DELETE', headers: headers() })
      await loadCommissionStats(commissionMode, commissionRange)
      await loadSettlementRecords(settlementFilter)
    } finally {
      setDeletingSettlementId(null)
    }
  }

  const togglePowerAgent = async (row: CommissionAgentRow) => {
    await fetch(`${apiUrl}/api/agent-admin/power-agents/${row.user_id}`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ power_agent: Number(row.power_agent || 0) !== 1 }),
    })
    await loadCommissionStats(commissionMode, commissionRange)
  }

  const statsSummary = commissionStats?.summary || {}
  const statsAgents = Array.isArray(commissionStats?.agents) ? commissionStats.agents : []
  const filteredAgents = statsAgents.filter(row => {
    const keyword = agentKeyword.trim().toLowerCase()
    if (!keyword) return true
    return [row.user_id, row.username, row.display_name].some(value => String(value || '').toLowerCase().includes(keyword))
  })
  const agentPageSize = 10
  const agentTotalPages = Math.max(1, Math.ceil(filteredAgents.length / agentPageSize))
  const safeAgentPage = Math.min(agentPage, agentTotalPages)
  const pagedAgents = filteredAgents.slice((safeAgentPage - 1) * agentPageSize, safeAgentPage * agentPageSize)
  const settlementFilteredBase = statsAgents.filter(row => {
    if (Number(row.power_agent || 0) !== 1) return false
    const keyword = settlementKeyword.trim().toLowerCase()
    if (!keyword) return true
    return [row.user_id, row.username, row.display_name].some(value => String(value || '').toLowerCase().includes(keyword))
  })
  const settlementAgents = settlementFilteredBase.filter(row => {
    if (settlementAgentFilter === 'settled') return Number(row.period_settled_amount || 0) > 0
    if (settlementAgentFilter === 'pending') return Number(row.period_commission_estimate || 0) > 0 && Number(row.period_pending_amount || 0) > 0
    if (settlementAgentFilter === 'power') return Number(row.power_agent || 0) === 1
    return true
  })
  const settlementPageSize = 10
  const settlementTotalPages = Math.max(1, Math.ceil(settlementAgents.length / settlementPageSize))
  const safeSettlementPage = Math.min(settlementPage, settlementTotalPages)
  const pagedSettlementAgents = settlementAgents.slice((safeSettlementPage - 1) * settlementPageSize, safeSettlementPage * settlementPageSize)
  const filteredSettlementRecords = settlementRecords.filter(row => {
    const keyword = recordKeyword.trim().toLowerCase()
    if (!keyword) return true
    return [row.user_id, row.username, row.display_name, row.period_type, row.period_start, row.period_end, row.settled_amount, row.status].some(value => String(value || '').toLowerCase().includes(keyword))
  })
  const parsedRecordPageSize = Number(recordPageSize) || 20
  const recordTotalPages = Math.max(1, Math.ceil(filteredSettlementRecords.length / parsedRecordPageSize))
  const safeRecordPage = Math.min(recordPage, recordTotalPages)
  const pagedSettlementRecords = filteredSettlementRecords.slice((safeRecordPage - 1) * parsedRecordPageSize, safeRecordPage * parsedRecordPageSize)
  const filteredRedemptionRecords = redemptionRecords.filter(row => {
    const keyword = redemptionKeyword.trim().toLowerCase()
    if (!keyword) return true
    return [row.user_id, row.username, row.display_name, row.redemption_name, row.redemption_key, row.commission_amount, row.status].some(value => String(value || '').toLowerCase().includes(keyword))
  })
  const redemptionPageSize = 20
  const redemptionTotalPages = Math.max(1, Math.ceil(filteredRedemptionRecords.length / redemptionPageSize))
  const safeRedemptionPage = Math.min(redemptionPage, redemptionTotalPages)
  const pagedRedemptionRecords = filteredRedemptionRecords.slice((safeRedemptionPage - 1) * redemptionPageSize, safeRedemptionPage * redemptionPageSize)

  return (
    <div className="space-y-6">
      <div className="flex gap-2 overflow-x-auto">
        <Button variant={activePanel === 'settings' ? 'default' : 'outline'} onClick={() => setActivePanel('settings')}>代理设置</Button>
        <Button variant={activePanel === 'commission' ? 'default' : 'outline'} onClick={() => setActivePanel('commission')}>佣金统计</Button>
        <Button variant={activePanel === 'settlement' ? 'default' : 'outline'} onClick={() => setActivePanel('settlement')}>结算记录</Button>
        <Button variant={activePanel === 'redemption' ? 'default' : 'outline'} onClick={() => setActivePanel('redemption')}>兑换记录</Button>
      </div>

      {activePanel === 'commission' && <Card>
        <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle className="flex items-center gap-2"><BarChart3 className="h-5 w-5" />佣金统计</CardTitle>
          <div className="flex items-center gap-2 flex-wrap"><Select value={commissionMode} onChange={e => setCommissionMode(e.target.value as CommissionMode)} className="w-28"><option value="day">日</option><option value="week">周</option><option value="month">月</option></Select><Input type="date" value={commissionRange.start} onChange={e => setCommissionRange({ ...commissionRange, start: e.target.value })} className="w-40" /><span className="text-sm text-muted-foreground">至</span><Input type="date" value={commissionRange.end} onChange={e => setCommissionRange({ ...commissionRange, end: e.target.value })} className="w-40" /><Button variant="outline" onClick={() => loadCommissionStats(commissionMode, commissionRange)}>刷新</Button></div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-sm text-muted-foreground">统计周期：{String(statsSummary.period_label || '-')}。这里只统计已添加到“专属佣金比例”列表里的用户，未添加用户不视为代理商。</div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <StatBox title="周期代理佣金" value={formatAdminMoney(statsSummary.period_commission_estimate)} hint={`一级 ${formatAdminMoney(statsSummary.period_first_commission)} · 二级 ${formatAdminMoney(statsSummary.period_second_commission)}`} />
            <StatBox title="周期代理充值" value={formatAdminMoney(statsSummary.period_success_money)} hint={`${formatAdminNumber(statsSummary.period_success_count)} 笔成功订单`} />
            <StatBox title="已结算金额" value={formatAdminMoney(statsSummary.settled_amount)} hint={`覆盖 ${formatAdminNumber(statsSummary.agent_count)} 个代理商`} />
            <StatBox title="待结算金额" value={formatAdminMoney(statsSummary.pending_amount)} hint={`累计佣金 ${formatAdminMoney(statsSummary.total_commission_estimate)} · 已兑换 ${formatAdminMoney(statsSummary.redeemed_amount)}`} />
          </div>
          <div className="space-y-3 border-t pt-4">
            <div className="flex items-center justify-between gap-3 flex-wrap"><div className="text-sm font-semibold">结算佣金</div><div className="flex items-center gap-2 flex-wrap"><Input value={settlementKeyword} onChange={e => setSettlementKeyword(e.target.value)} placeholder="搜索实力代理ID / 用户名 / 昵称" className="w-72" /><Select value={settlementAgentFilter} onChange={e => setSettlementAgentFilter(e.target.value as SettlementAgentFilter)} className="w-36"><option value="all">全部实力代理</option><option value="settled">已结算</option><option value="pending">待结算</option></Select></div></div>
            <div className="text-sm text-muted-foreground">共 {settlementAgents.length} 个可结算代理商，当前第 {safeSettlementPage} / {settlementTotalPages} 页</div>
            <div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>代理商</TableHead><TableHead>标签</TableHead><TableHead className="text-right">本期佣金</TableHead><TableHead className="text-right">本期已结算</TableHead><TableHead className="text-right">本期已兑换</TableHead><TableHead className="text-right">本期待结算</TableHead><TableHead>最近结算</TableHead><TableHead className="text-right">操作</TableHead></TableRow></TableHeader><TableBody>{pagedSettlementAgents.map(row => <TableRow key={row.user_id}><TableCell><div className="font-medium">{row.display_name || row.username || '-'}</div><div className="text-xs text-muted-foreground">ID {row.user_id}</div></TableCell><TableCell>{Number(row.power_agent || 0) === 1 ? '实力代理' : '-'}</TableCell><TableCell className="text-right font-semibold">{formatAdminMoney(row.period_commission_estimate)}</TableCell><TableCell className="text-right">{formatAdminMoney(row.period_settled_amount)}</TableCell><TableCell className="text-right">{formatAdminMoney(row.period_redeemed_amount)}</TableCell><TableCell className="text-right">{formatAdminMoney(row.period_pending_amount)}</TableCell><TableCell>{formatAdminTime(row.last_settled_at)}</TableCell><TableCell className="text-right"><Button size="sm" variant="outline" disabled={Number(row.period_commission_estimate || 0) <= 0} onClick={() => settleAgent(row)}>标记结算</Button></TableCell></TableRow>)}{pagedSettlementAgents.length === 0 && <TableRow><TableCell colSpan={8} className="text-center text-muted-foreground py-8">暂无可结算代理商</TableCell></TableRow>}</TableBody></Table></div>
            <div className="flex items-center justify-end gap-2"><Button variant="outline" size="sm" disabled={safeSettlementPage <= 1} onClick={() => setSettlementPage(p => Math.max(1, p - 1))}>上一页</Button><Button variant="outline" size="sm" disabled={safeSettlementPage >= settlementTotalPages} onClick={() => setSettlementPage(p => Math.min(settlementTotalPages, p + 1))}>下一页</Button></div>
          </div>
          <div className="flex items-center justify-between gap-3 flex-wrap border-t pt-4"><Input value={agentKeyword} onChange={e => setAgentKeyword(e.target.value)} placeholder="搜索代理商ID / 用户名 / 昵称" className="max-w-sm" /><div className="text-sm text-muted-foreground">共 {filteredAgents.length} 个代理商，当前第 {safeAgentPage} / {agentTotalPages} 页</div></div>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader><TableRow><TableHead>代理商</TableHead><TableHead>标签</TableHead><TableHead className="text-right">周期充值</TableHead><TableHead className="text-right">周期佣金</TableHead><TableHead className="text-right">一级佣金</TableHead><TableHead className="text-right">二级佣金</TableHead><TableHead className="text-right">已结算</TableHead><TableHead className="text-right">已兑换</TableHead><TableHead className="text-right">待结算</TableHead><TableHead className="text-right">累计佣金</TableHead><TableHead>最近结算</TableHead><TableHead className="text-right">操作</TableHead></TableRow></TableHeader>
              <TableBody>{pagedAgents.map(row => <TableRow key={row.user_id}><TableCell><div className="font-medium">{row.display_name || row.username || '-'}</div><div className="text-xs text-muted-foreground">ID {row.user_id} · 比例 {formatAdminPercent(row.commission_rate)}</div></TableCell><TableCell>{Number(row.power_agent || 0) === 1 ? '实力代理' : '-'}</TableCell><TableCell className="text-right">{formatAdminMoney(row.period_success_money)}</TableCell><TableCell className="text-right font-semibold">{formatAdminMoney(row.period_commission_estimate)}</TableCell><TableCell className="text-right">{formatAdminMoney(row.period_first_commission)}</TableCell><TableCell className="text-right">{Number(row.second_commission_enabled || 0) === 1 ? formatAdminMoney(row.period_second_commission) : '未启用'}</TableCell><TableCell className="text-right">{formatAdminMoney(row.settled_amount)}</TableCell><TableCell className="text-right">{formatAdminMoney(row.redeemed_amount)}</TableCell><TableCell className="text-right">{formatAdminMoney(row.pending_amount)}</TableCell><TableCell className="text-right">{formatAdminMoney(row.total_commission_estimate)}</TableCell><TableCell>{formatAdminTime(row.last_settled_at)}</TableCell><TableCell className="text-right"><Button size="sm" variant="outline" onClick={() => togglePowerAgent(row)}>{Number(row.power_agent || 0) === 1 ? '取消实力' : '设为实力'}</Button></TableCell></TableRow>)}{pagedAgents.length === 0 && <TableRow><TableCell colSpan={12} className="text-center text-muted-foreground py-8">暂无代理佣金数据</TableCell></TableRow>}</TableBody>
            </Table>
          </div>
          <div className="flex items-center justify-end gap-2"><Button variant="outline" size="sm" disabled={safeAgentPage <= 1} onClick={() => setAgentPage(p => Math.max(1, p - 1))}>上一页</Button><Button variant="outline" size="sm" disabled={safeAgentPage >= agentTotalPages} onClick={() => setAgentPage(p => Math.min(agentTotalPages, p + 1))}>下一页</Button></div>
        </CardContent>
      </Card>}

      {activePanel === 'settlement' && <Card>
        <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>结算记录</CardTitle>
          <div className="flex items-center gap-2 flex-wrap"><Input value={recordKeyword} onChange={e => setRecordKeyword(e.target.value)} placeholder="搜索代理商 / 周期 / 金额 / 状态" className="w-72" /><Select value={settlementFilter} onChange={e => setSettlementFilter(e.target.value as SettlementFilterMode)} className="w-32"><option value="all">全部</option><option value="day">日结</option><option value="week">周结</option><option value="month">月结</option><option value="custom">自定义</option></Select><Select value={recordPageSize} onChange={e => setRecordPageSize(e.target.value)} className="w-28"><option value="20">20条</option><option value="50">50条</option><option value="100">100条</option></Select></div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-sm text-muted-foreground">共 {filteredSettlementRecords.length} 条记录，当前第 {safeRecordPage} / {recordTotalPages} 页</div>
          <div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>代理商</TableHead><TableHead>结算类型</TableHead><TableHead>周期</TableHead><TableHead className="text-right">金额</TableHead><TableHead>结算时间</TableHead><TableHead>状态</TableHead><TableHead className="text-right">操作</TableHead></TableRow></TableHeader><TableBody>{pagedSettlementRecords.map(row => <TableRow key={row.id}><TableCell><div className="font-medium">{row.display_name || row.username || '-'}</div><div className="text-xs text-muted-foreground">ID {row.user_id}</div></TableCell><TableCell>{settlementModeLabel(row.period_type)}</TableCell><TableCell>{row.period_start} 至 {row.period_end}</TableCell><TableCell className="text-right">{formatAdminMoney(row.settled_amount)}</TableCell><TableCell>{formatAdminTime(row.settled_at)}</TableCell><TableCell><Select value={row.status || 'settled'} onChange={e => updateSettlementStatus(row.id, e.target.value)} className="w-28"><option value="settled">已结算</option><option value="unsettled">未结算</option></Select></TableCell><TableCell className="text-right"><Button variant="ghost" size="sm" onClick={() => deleteSettlement(row.id)} disabled={deletingSettlementId === row.id} title="删除"><Trash2 className="h-4 w-4" /></Button></TableCell></TableRow>)}{pagedSettlementRecords.length === 0 && <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground py-8">暂无结算记录</TableCell></TableRow>}</TableBody></Table></div>
          <div className="flex items-center justify-end gap-2"><Button variant="outline" size="sm" disabled={safeRecordPage <= 1} onClick={() => setRecordPage(p => Math.max(1, p - 1))}>上一页</Button><Button variant="outline" size="sm" disabled={safeRecordPage >= recordTotalPages} onClick={() => setRecordPage(p => Math.min(recordTotalPages, p + 1))}>下一页</Button></div>
        </CardContent>
      </Card>}

      {activePanel === 'redemption' && <Card>
        <CardHeader className="gap-3 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>兑换记录</CardTitle>
          <div className="flex items-center gap-2 flex-wrap"><Input value={redemptionKeyword} onChange={e => setRedemptionKeyword(e.target.value)} placeholder="搜索代理商 / 兑换码 / 金额 / 状态" className="w-80" /><Button variant="outline" onClick={loadRedemptionRecords}>刷新</Button></div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <StatBox title="兑换记录数" value={formatAdminNumber(filteredRedemptionRecords.length)} hint="当前筛选范围" />
            <StatBox title="消耗佣金" value={formatAdminMoney(filteredRedemptionRecords.reduce((sum, row) => sum + Number(row.commission_amount || 0), 0))} hint="代理已兑换消耗" />
            <StatBox title="兑换额度" value={formatAdminQuota(filteredRedemptionRecords.reduce((sum, row) => sum + Number(row.quota || 0), 0))} hint="兑换码额度合计" />
          </div>
          <div className="text-sm text-muted-foreground">共 {filteredRedemptionRecords.length} 条记录，当前第 {safeRedemptionPage} / {redemptionTotalPages} 页</div>
          <div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>代理商</TableHead><TableHead>兑换码</TableHead><TableHead className="text-right">额度</TableHead><TableHead className="text-right">消耗佣金</TableHead><TableHead>状态</TableHead><TableHead>兑换时间</TableHead></TableRow></TableHeader><TableBody>{pagedRedemptionRecords.map(row => <TableRow key={row.id}><TableCell><div className="font-medium">{row.display_name || row.username || '-'}</div><div className="text-xs text-muted-foreground">ID {row.user_id}</div></TableCell><TableCell><div className="font-medium">{row.redemption_name || '-'}</div><div className="text-xs text-muted-foreground break-all">{row.redemption_key || '-'}</div></TableCell><TableCell className="text-right">{formatAdminQuota(row.quota)}</TableCell><TableCell className="text-right font-semibold">{formatAdminMoney(row.commission_amount)}</TableCell><TableCell>{redemptionStatusLabel(row.status)}</TableCell><TableCell>{formatAdminTime(row.created_at)}</TableCell></TableRow>)}{pagedRedemptionRecords.length === 0 && <TableRow><TableCell colSpan={6} className="text-center text-muted-foreground py-8">暂无兑换记录</TableCell></TableRow>}</TableBody></Table></div>
          <div className="flex items-center justify-end gap-2"><Button variant="outline" size="sm" disabled={safeRedemptionPage <= 1} onClick={() => setRedemptionPage(p => Math.max(1, p - 1))}>上一页</Button><Button variant="outline" size="sm" disabled={safeRedemptionPage >= redemptionTotalPages} onClick={() => setRedemptionPage(p => Math.min(redemptionTotalPages, p + 1))}>下一页</Button></div>
        </CardContent>
      </Card>}

      {activePanel === 'settings' && <>
      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><Megaphone className="h-5 w-5" />代理公告</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <textarea value={announcementHtml} onChange={e => setAnnouncementHtml(e.target.value)} placeholder="支持 HTML，例如 <strong>公告</strong><br><a href='...'>链接</a>" className="w-full min-h-40 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring" />
          <div className="rounded-md border bg-muted/30 p-3 text-sm" dangerouslySetInnerHTML={{ __html: announcementHtml || '公告预览' }} />
          <Button onClick={saveSettings} disabled={saving}><Save className="h-4 w-4 mr-2" />保存公告</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>佣金比例</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-[1fr_auto] gap-3 items-end">
            <div>
              <label className="text-sm font-medium">默认佣金比例</label>
              <Input value={defaultRate} onChange={e => setDefaultRate(e.target.value)} placeholder="0.1 表示 10%" />
            </div>
            <Button onClick={saveSettings} disabled={saving}><Save className="h-4 w-4 mr-2" />保存默认佣金</Button>
          </div>
          <div className="rounded-md border border-emerald-200 bg-emerald-50/60 p-3 space-y-3">
            <div>
              <div className="text-sm font-semibold">佣金兑换比例</div>
              <p className="text-xs text-muted-foreground mt-1">控制代理用佣金兑换 yj 兑换码时的消耗比例，默认 10 额度消耗 3 佣金。</p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr_auto] gap-3 items-end">
              <div>
                <label className="text-sm font-medium">兑换码额度</label>
                <Input value={redemptionQuotaUnit} onChange={e => setRedemptionQuotaUnit(e.target.value)} placeholder="10" />
              </div>
              <div className="hidden md:block pb-2 text-sm text-muted-foreground">:</div>
              <div>
                <label className="text-sm font-medium">消耗佣金</label>
                <Input value={redemptionCommissionUnit} onChange={e => setRedemptionCommissionUnit(e.target.value)} placeholder="3" />
              </div>
              <Button onClick={saveSettings} disabled={saving}><Save className="h-4 w-4 mr-2" />保存兑换比例</Button>
            </div>
            <div className="text-xs text-muted-foreground">当前比例：{Number(redemptionQuotaUnit || 0) || 10} 额度 : {Number(redemptionCommissionUnit || 0) || 3} 佣金</div>
          </div>
          <div className="rounded-md border p-3 text-sm text-muted-foreground">二级代理统计默认不启用。需要在下面编辑具体用户，勾选“当前用户启用二级代理统计”后才展示二级佣金；默认比例为 3%。</div>
          <div className="grid grid-cols-1 md:grid-cols-[1fr_1fr_auto] gap-3 items-end">
            <div><label className="text-sm font-medium">用户ID</label><Input value={userId} onChange={e => setUserId(e.target.value)} placeholder="代理商 users.id" /></div>
            <div><label className="text-sm font-medium">专属佣金比例</label><Input value={rate} onChange={e => setRate(e.target.value)} placeholder="0.15 表示 15%" /></div>
            <Button onClick={saveRate}><Plus className="h-4 w-4 mr-2" />添加/更新</Button>
          </div>
          <div className="rounded-md border border-primary/30 bg-primary/5 p-3 space-y-3">
            <div className="text-sm font-semibold">当前用户二级代理设置</div>
            <div className="grid grid-cols-1 md:grid-cols-[auto_1fr] gap-3 items-end">
              <label className="flex items-center gap-2 text-sm font-medium pb-2"><input type="checkbox" checked={userSecondEnabled} onChange={e => setUserSecondEnabled(e.target.checked)} />启用二级代理统计</label>
              <div><label className="text-sm font-medium">二级代理佣金比例</label><Input value={userSecondRate} onChange={e => setUserSecondRate(e.target.value)} placeholder="0.03 表示 3%" /></div>
            </div>
          </div>
          <Table>
            <TableHeader><TableRow><TableHead>用户ID</TableHead><TableHead>用户</TableHead><TableHead>佣金比例</TableHead><TableHead>二级代理</TableHead><TableHead className="text-right">操作</TableHead></TableRow></TableHeader>
            <TableBody>{rates.map(row => <TableRow key={row.user_id}><TableCell>{row.user_id}</TableCell><TableCell>{row.display_name || row.username || '-'}</TableCell><TableCell>{(Number(row.commission_rate || 0) * 100).toFixed(1)}%</TableCell><TableCell>{Number(row.second_commission_enabled || 0) === 1 ? `${(Number(row.second_commission_rate || 0) * 100).toFixed(1)}%` : '未启用'}</TableCell><TableCell className="text-right"><Button variant="ghost" size="sm" onClick={() => editRate(row)} title="编辑"><Pencil className="h-4 w-4" /></Button><Button variant="ghost" size="sm" onClick={() => deleteRate(row.user_id)} title="删除"><Trash2 className="h-4 w-4" /></Button></TableCell></TableRow>)}{rates.length === 0 && <TableRow><TableCell colSpan={5} className="text-center text-muted-foreground py-8">暂无专属佣金配置</TableCell></TableRow>}</TableBody>
          </Table>
        </CardContent>
      </Card>
      </>}
    </div>
  )
}

function StatBox({ title, value, hint }: { title: string; value: string; hint: string }) {
  return <div className="rounded-md border bg-background p-4"><div className="text-sm text-muted-foreground">{title}</div><div className="mt-1 text-2xl font-bold">{value}</div><div className="mt-1 text-xs text-muted-foreground">{hint}</div></div>
}

function formatAdminMoney(value: unknown) {
  const num = Number(value || 0)
  return `¥${(Number.isFinite(num) ? num : 0).toFixed(2)}`
}

function formatAdminNumber(value: unknown) {
  const num = Number(value || 0)
  return (Number.isFinite(num) ? num : 0).toLocaleString()
}

function formatAdminQuota(value: unknown) {
  const num = Number(value || 0) / 500000
  return (Number.isFinite(num) ? num : 0).toLocaleString(undefined, { maximumFractionDigits: 2 })
}

function redemptionStatusLabel(value: unknown) {
  const status = String(value || '').toLowerCase()
  if (status === 'success') return '已兑换'
  if (status === 'failed') return '失败'
  return status || '-'
}

function formatAdminPercent(value: unknown) {
  const num = Number(value || 0)
  return `${((Number.isFinite(num) ? num : 0) * 100).toFixed(1)}%`
}

function formatAdminTime(value: unknown) {
  const num = Number(value || 0)
  if (!num) return '-'
  return new Date(num * 1000).toLocaleString('zh-CN', { hour12: false })
}

function settlementModeLabel(value: unknown) {
  if (value === 'day') return '日结'
  if (value === 'week') return '周结'
  if (value === 'month') return '月结'
  if (value === 'custom') return '自定义'
  return '-'
}

function todayRange() {
  const now = new Date()
  const day = formatDateInput(now)
  return { start: day, end: day }
}

function formatDateInput(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export default AgentAdminSettings
