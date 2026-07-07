import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, RefreshCw, Search, ShieldCheck } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { apiFetch, createAuthHeaders } from '../lib/api'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Input } from './ui/input'

interface PromptAuditEvent {
  source: string
  request_id: string
  user_id: number
  token_id: number
  model: string
  group: string
  channel_id: number
  prompt: string
  prompt_sha256: string
  prompt_truncated: boolean
  matched_keywords?: string[]
  created_at: number
  received_at: number
}

interface PromptAuditList {
  items: PromptAuditEvent[]
  total: number
  limit: number
  offset: number
  path: string
}

function formatTime(ts: number) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN')
}

function shortHash(hash: string) {
  if (!hash) return '-'
  return hash.length > 16 ? `${hash.slice(0, 12)}...${hash.slice(-6)}` : hash
}

export function PromptAudit() {
  const { token } = useAuth()
  const [data, setData] = useState<PromptAuditList | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [limit, setLimit] = useState(50)

  const apiUrl = import.meta.env.VITE_API_URL || ''

  const loadEvents = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError('')
    try {
      const response = await apiFetch(`${apiUrl}/api/prompt-audit/events?limit=${limit}&offset=0`, {
        headers: createAuthHeaders(token),
      })
      const json = await response.json()
      if (!json.success) {
        throw new Error(json.error?.message || '加载提示词审查记录失败')
      }
      setData(json.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载提示词审查记录失败')
    } finally {
      setLoading(false)
    }
  }, [apiUrl, limit, token])

  useEffect(() => {
    void loadEvents()
  }, [loadEvents])

  const events = data?.items || []
  const filteredEvents = events.filter(event => {
    const q = query.trim().toLowerCase()
    if (!q) return true
    return [
      event.request_id,
      String(event.user_id || ''),
      String(event.token_id || ''),
      event.model,
      event.group,
      event.prompt,
      event.prompt_sha256,
      ...(event.matched_keywords || []),
    ].some(value => String(value || '').toLowerCase().includes(q))
  })

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <ShieldCheck className="w-6 h-6 text-primary" />
            提示词审查
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            展示 NewAPI hook 命中关键词后上报的提示词记录。这里只做后置审查，不影响用户请求。
          </p>
        </div>
        <Button onClick={loadEvents} disabled={loading} variant="outline">
          <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>总命中记录</CardDescription>
            <CardTitle>{data?.total ?? 0}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>当前显示</CardDescription>
            <CardTitle>{filteredEvents.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>存储文件</CardDescription>
            <CardTitle className="text-sm font-mono truncate" title={data?.path || ''}>
              {data?.path || '-'}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>命中事件</CardTitle>
              <CardDescription>按接收时间倒序显示，仅包含命中 `PROMPT_AUDIT_KEYWORDS` 的请求。</CardDescription>
            </div>
            <div className="flex flex-col sm:flex-row gap-2">
              <div className="relative">
                <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={e => setQuery(e.target.value)}
                  placeholder="搜索用户、模型、关键词、prompt"
                  className="pl-9 sm:w-80"
                />
              </div>
              <select
                value={limit}
                onChange={e => setLimit(Number(e.target.value))}
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
              >
                <option value={20}>20 条</option>
                <option value={50}>50 条</option>
                <option value={100}>100 条</option>
                <option value={200}>200 条</option>
              </select>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {error && (
            <div className="mb-4 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive flex items-center gap-2">
              <AlertTriangle className="w-4 h-4" />
              {error}
            </div>
          )}

          <div className="space-y-4">
            {filteredEvents.map(event => (
              <div key={`${event.request_id}-${event.received_at}-${event.prompt_sha256}`} className="rounded-lg border bg-background p-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                  <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      {(event.matched_keywords || []).map(keyword => (
                        <Badge key={keyword} variant="destructive">{keyword}</Badge>
                      ))}
                      {event.prompt_truncated && <Badge variant="secondary">已截断</Badge>}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      用户 ID {event.user_id || '-'} · Token ID {event.token_id || '-'} · 模型 {event.model || '-'} · 分组 {event.group || '-'}
                    </div>
                    <div className="text-xs text-muted-foreground font-mono">
                      request_id: {event.request_id || '-'} · sha256: {shortHash(event.prompt_sha256)}
                    </div>
                  </div>
                  <div className="text-xs text-muted-foreground lg:text-right">
                    <div>请求时间：{formatTime(event.created_at)}</div>
                    <div>接收时间：{formatTime(event.received_at)}</div>
                  </div>
                </div>
                <pre className="mt-4 max-h-56 overflow-auto whitespace-pre-wrap rounded-md bg-muted p-3 text-sm leading-relaxed">
                  {event.prompt}
                </pre>
              </div>
            ))}

            {!loading && filteredEvents.length === 0 && (
              <div className="rounded-lg border border-dashed p-10 text-center text-muted-foreground">
                暂无提示词审查记录。确认 NewAPI 已配置 `PROMPT_AUDIT_ENABLED=true` 且关键词命中后再刷新。
              </div>
            )}

            {loading && (
              <div className="rounded-lg border border-dashed p-10 text-center text-muted-foreground">
                正在加载提示词审查记录...
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export default PromptAudit
