import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, ChevronLeft, ChevronRight, RefreshCw, Search, ShieldCheck } from 'lucide-react'
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

interface PromptAuditConfig {
  enabled: boolean
  version: number
  keywords: string[]
  updated_at: number
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
  const [offset, setOffset] = useState(0)
  const [configLoading, setConfigLoading] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)
  const [configEnabled, setConfigEnabled] = useState(false)
  const [configVersion, setConfigVersion] = useState(0)
  const [configUpdatedAt, setConfigUpdatedAt] = useState(0)
  const [keywordsRaw, setKeywordsRaw] = useState('')
  const [configMessage, setConfigMessage] = useState('')

  const apiUrl = import.meta.env.VITE_API_URL || ''

  const normalizeKeywords = useCallback((raw: string) => {
    const seen = new Set<string>()
    return raw
      .replace(/\r\n/g, '\n')
      .replace(/[，,；;]/g, '\n')
      .split('\n')
      .map(item => item.trim().toLowerCase())
      .filter(item => {
        if (!item || seen.has(item)) return false
        seen.add(item)
        return true
      })
  }, [])

  const loadConfig = useCallback(async () => {
    if (!token) return
    setConfigLoading(true)
    setConfigMessage('')
    try {
      const response = await apiFetch(`${apiUrl}/api/prompt-audit/config`, {
        headers: createAuthHeaders(token),
      })
      const json = await response.json()
      if (!json.success) {
        throw new Error(json.error?.message || '加载提示词审查配置失败')
      }
      const cfg = json.data as PromptAuditConfig
      setConfigEnabled(Boolean(cfg.enabled))
      setConfigVersion(Number(cfg.version || 0))
      setConfigUpdatedAt(Number(cfg.updated_at || 0))
      setKeywordsRaw((cfg.keywords || []).join('\n'))
    } catch (err) {
      setConfigMessage(err instanceof Error ? err.message : '加载提示词审查配置失败')
    } finally {
      setConfigLoading(false)
    }
  }, [apiUrl, token])

  const loadEvents = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError('')
    try {
      const response = await apiFetch(`${apiUrl}/api/prompt-audit/events?limit=${limit}&offset=${offset}`, {
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
  }, [apiUrl, limit, offset, token])

  useEffect(() => {
    void loadEvents()
  }, [loadEvents])

  useEffect(() => {
    void loadConfig()
  }, [loadConfig])

  const events = data?.items || []
  const total = data?.total ?? 0
  const currentPage = total === 0 ? 0 : Math.floor(offset / limit) + 1
  const totalPages = total === 0 ? 0 : Math.ceil(total / limit)
  const canGoPrevious = offset > 0 && !loading
  const canGoNext = offset + limit < total && !loading
  const rangeStart = total === 0 ? 0 : offset + 1
  const rangeEnd = Math.min(offset + events.length, total)
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

  const changeLimit = (nextLimit: number) => {
    setLimit(nextLimit)
    setOffset(0)
  }

  const goPrevious = () => {
    setOffset(current => Math.max(0, current - limit))
  }

  const goNext = () => {
    setOffset(current => current + limit)
  }

  const saveConfig = async () => {
    if (!token) return
    setConfigSaving(true)
    setConfigMessage('')
    try {
      const keywords = normalizeKeywords(keywordsRaw)
      const response = await apiFetch(`${apiUrl}/api/prompt-audit/config`, {
        method: 'PUT',
        headers: {
          ...createAuthHeaders(token),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          enabled: configEnabled,
          keywords,
        }),
      })
      const json = await response.json()
      if (!json.success) {
        throw new Error(json.error?.message || '保存提示词审查配置失败')
      }
      const cfg = json.data as PromptAuditConfig
      setConfigEnabled(Boolean(cfg.enabled))
      setConfigVersion(Number(cfg.version || 0))
      setConfigUpdatedAt(Number(cfg.updated_at || 0))
      setKeywordsRaw((cfg.keywords || []).join('\n'))
      setConfigMessage(`已保存 ${cfg.keywords?.length || 0} 个关键词，版本 ${cfg.version || 0}`)
    } catch (err) {
      setConfigMessage(err instanceof Error ? err.message : '保存提示词审查配置失败')
    } finally {
      setConfigSaving(false)
    }
  }

  const keywordCount = normalizeKeywords(keywordsRaw).length

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
            <CardTitle>{rangeStart}-{rangeEnd}</CardTitle>
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
          <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <CardTitle>审查词库配置</CardTitle>
              <CardDescription>
                一行一个关键词，也兼容逗号/分号分隔。保存后版本号自增，NewAPI 会按版本拉取并重建 Aho-Corasick 匹配器。
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={loadConfig} disabled={configLoading || configSaving}>
                <RefreshCw className={`w-4 h-4 mr-2 ${configLoading ? 'animate-spin' : ''}`} />
                重新加载
              </Button>
              <Button onClick={saveConfig} disabled={configSaving}>
                {configSaving ? '保存中...' : '保存词库'}
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={configEnabled}
              onChange={event => setConfigEnabled(event.target.checked)}
              className="h-4 w-4"
            />
            启用 tools 词库配置
          </label>
          <textarea
            value={keywordsRaw}
            onChange={event => setKeywordsRaw(event.target.value)}
            placeholder={'每行一个关键词，例如：\n代写\n博彩\n绕过限制'}
            className="min-h-56 w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono leading-6 outline-none focus:ring-2 focus:ring-ring"
          />
          <div className="flex flex-col gap-1 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
            <span>当前输入 {keywordCount} 个关键词，配置版本 {configVersion || 0}，更新时间 {formatTime(configUpdatedAt)}</span>
            <span>大词库建议一行一个词，避免在逗号分隔里混入说明文本。</span>
          </div>
          {configMessage && (
            <div className="rounded-md border bg-muted p-3 text-sm text-muted-foreground">
              {configMessage}
            </div>
          )}
        </CardContent>
      </Card>

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
                onChange={e => changeLimit(Number(e.target.value))}
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

          <div className="mt-6 flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="text-sm text-muted-foreground">
              第 {currentPage || 0} / {totalPages || 0} 页，当前页 {filteredEvents.length} 条，合计 {total} 条
              {query.trim() ? '（搜索仅过滤当前页）' : ''}
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={goPrevious} disabled={!canGoPrevious}>
                <ChevronLeft className="mr-1 h-4 w-4" />
                上一页
              </Button>
              <Button variant="outline" size="sm" onClick={goNext} disabled={!canGoNext}>
                下一页
                <ChevronRight className="ml-1 h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export default PromptAudit
