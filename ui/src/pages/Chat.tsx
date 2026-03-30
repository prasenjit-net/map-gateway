import { useState, useRef, useEffect, useCallback } from 'react'
import OpenAI from 'openai'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { createMCPClient, type MCPClient, type Transport } from '../lib/mcp-client'
import { Send, Settings, ChevronDown, ChevronRight, Plus, Loader2, KeyRound, Trash2 } from 'lucide-react'
import { cn } from '../lib/utils'
import {
  type ChatMessage,
  type ChatSession,
  loadSessions,
  saveSession,
  deleteSession,
  createSession,
  generateTitle,
} from '../lib/chat-storage'

interface ServerChatConfig {
  model: string
  hasKey: boolean
}

async function fetchChatConfig(): Promise<ServerChatConfig> {
  const res = await fetch('/_api/chat/config')
  if (!res.ok) throw new Error('Failed to load chat config')
  return res.json() as Promise<ServerChatConfig>
}

function ToolCallBlock({ name, args, expanded: initExpanded = false }: { name: string; args: Record<string, unknown>; expanded?: boolean }) {
  const [expanded, setExpanded] = useState(initExpanded)
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden text-sm">
      <button onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 px-3 py-2 w-full text-left hover:bg-gray-700 transition-colors">
        {expanded ? <ChevronDown className="w-3.5 h-3.5 text-orange-400" /> : <ChevronRight className="w-3.5 h-3.5 text-orange-400" />}
        <span className="text-orange-400 font-mono text-xs">tool_call</span>
        <span className="text-gray-300 font-medium">{name}</span>
      </button>
      {expanded && (
        <pre className="px-3 pb-3 text-gray-400 text-xs overflow-auto">{JSON.stringify(args, null, 2)}</pre>
      )}
    </div>
  )
}

function ToolResultBlock({ content }: { content: string }) {
  const [expanded, setExpanded] = useState(false)
  const isLong = content.length > 200
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden text-sm">
      <button onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 px-3 py-2 w-full text-left hover:bg-gray-700 transition-colors">
        {expanded ? <ChevronDown className="w-3.5 h-3.5 text-green-400" /> : <ChevronRight className="w-3.5 h-3.5 text-green-400" />}
        <span className="text-green-400 font-mono text-xs">tool_result</span>
        {!expanded && isLong && <span className="text-gray-500 text-xs">{content.slice(0, 80)}…</span>}
      </button>
      {(expanded || !isLong) && (
        <pre className="px-3 pb-3 text-gray-400 text-xs overflow-auto whitespace-pre-wrap">{content}</pre>
      )}
    </div>
  )
}

function MarkdownContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        p: ({ children }) => <p className="mb-2 last:mb-0 leading-relaxed">{children}</p>,
        h1: ({ children }) => <h1 className="text-lg font-bold mb-2 mt-3 first:mt-0">{children}</h1>,
        h2: ({ children }) => <h2 className="text-base font-bold mb-2 mt-3 first:mt-0">{children}</h2>,
        h3: ({ children }) => <h3 className="text-sm font-bold mb-1 mt-2 first:mt-0">{children}</h3>,
        ul: ({ children }) => <ul className="list-disc list-inside mb-2 space-y-0.5">{children}</ul>,
        ol: ({ children }) => <ol className="list-decimal list-inside mb-2 space-y-0.5">{children}</ol>,
        li: ({ children }) => <li className="leading-relaxed">{children}</li>,
        code: ({ className, children, ...props }) => {
          const isBlock = className?.includes('language-')
          return isBlock ? (
            <code className="block bg-gray-900 text-green-300 rounded px-3 py-2 text-xs font-mono overflow-auto my-2 whitespace-pre" {...props}>{children}</code>
          ) : (
            <code className="bg-gray-700 text-pink-300 rounded px-1 py-0.5 text-xs font-mono" {...props}>{children}</code>
          )
        },
        pre: ({ children }) => <pre className="my-2 overflow-auto">{children}</pre>,
        blockquote: ({ children }) => <blockquote className="border-l-2 border-gray-500 pl-3 italic text-gray-300 my-2">{children}</blockquote>,
        a: ({ href, children }) => <a href={href} target="_blank" rel="noopener noreferrer" className="text-blue-400 underline hover:text-blue-300">{children}</a>,
        table: ({ children }) => <div className="overflow-auto my-2"><table className="text-xs border-collapse w-full">{children}</table></div>,
        th: ({ children }) => <th className="border border-gray-600 px-2 py-1 bg-gray-700 font-semibold text-left">{children}</th>,
        td: ({ children }) => <td className="border border-gray-600 px-2 py-1">{children}</td>,
        hr: () => <hr className="border-gray-600 my-3" />,
        strong: ({ children }) => <strong className="font-semibold text-white">{children}</strong>,
        em: ({ children }) => <em className="italic text-gray-200">{children}</em>,
      }}
    >
      {content}
    </ReactMarkdown>
  )
}


function MessageBubble({ msg }: { msg: ChatMessage }) {
  if (msg.role === 'tool_call') {
    return <div className="mx-4 my-1"><ToolCallBlock name={msg.toolName!} args={msg.toolArgs ?? {}} /></div>
  }
  if (msg.role === 'tool_result') {
    return <div className="mx-4 my-1"><ToolResultBlock content={msg.content} /></div>
  }
  if (msg.role === 'user') {
    return (
      <div className="flex justify-end px-4 my-2">
        <div className="max-w-[75%] bg-blue-600 text-white rounded-2xl rounded-tr-sm px-4 py-2.5 text-sm whitespace-pre-wrap">{msg.content}</div>
      </div>
    )
  }
  return (
    <div className="flex justify-start px-4 my-2">
      <div className="max-w-[75%] bg-gray-800 text-gray-100 rounded-2xl rounded-tl-sm px-4 py-2.5 text-sm">
        <MarkdownContent content={msg.content} />
        {msg.isStreaming && <span className="inline-block w-1.5 h-4 bg-blue-400 ml-1 animate-pulse rounded-sm" />}
      </div>
    </div>
  )
}

function formatSessionDate(ts: number): string {
  const now = new Date()
  const d = new Date(ts)
  const diffDays = Math.floor((now.getTime() - d.getTime()) / 86400000)
  if (diffDays === 0) return 'Today'
  if (diffDays === 1) return 'Yesterday'
  return d.toLocaleDateString()
}

const MODELS = ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo']

export default function Chat() {
  const [serverConfig, setServerConfig] = useState<ServerChatConfig | null>(null)
  const [model, setModel] = useState(() => localStorage.getItem('chat_model') ?? 'gpt-4o')
  const [transport, setTransport] = useState<Transport>(() => (localStorage.getItem('chat_transport') as Transport) ?? 'http')
  const [systemPrompt, setSystemPrompt] = useState(() => localStorage.getItem('chat_system_prompt') ?? 'You are a helpful assistant with access to tools.')
  const [settingsOpen, setSettingsOpen] = useState(true)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [sessions, setSessions] = useState<ChatSession[]>(() => loadSessions())
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null)
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const mcpClientRef = useRef<MCPClient | null>(null)
  const idRef = useRef(0)
  const activeSessionIdRef = useRef<string | null>(null)

  useEffect(() => { activeSessionIdRef.current = activeSessionId }, [activeSessionId])

  const nextId = () => String(++idRef.current)

  // On mount: load most recent session if available
  useEffect(() => {
    const saved = loadSessions()
    setSessions(saved)
    if (saved.length > 0) {
      const latest = saved[0]
      setActiveSessionId(latest.id)
      setMessages(latest.messages)
      setModel(latest.model)
      setTransport(latest.transport as Transport)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Load server-side chat config (model default + key presence check) on mount.
  useEffect(() => {
    fetchChatConfig()
      .then(cfg => {
        setServerConfig(cfg)
        if (!localStorage.getItem('chat_model')) {
          setModel(cfg.model)
        }
      })
      .catch(() => setServerConfig({ model: 'gpt-4o', hasKey: false }))
  }, [])

  useEffect(() => { messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [messages])

  const saveSettings = () => {
    localStorage.setItem('chat_model', model)
    localStorage.setItem('chat_transport', transport)
    localStorage.setItem('chat_system_prompt', systemPrompt)
  }

  const getMCPClient = useCallback(async () => {
    if (!mcpClientRef.current) {
      mcpClientRef.current = await createMCPClient(transport)
    }
    return mcpClientRef.current
  }, [transport])

  useEffect(() => {
    mcpClientRef.current?.disconnect()
    mcpClientRef.current = null
  }, [transport])

  const saveCurrentSession = useCallback((msgs: ChatMessage[], sidOverride?: string) => {
    const sid = sidOverride ?? activeSessionIdRef.current
    if (!sid) return
    const allSessions = loadSessions()
    const session = allSessions.find(s => s.id === sid)
    if (!session) return
    const firstUser = msgs.find(m => m.role === 'user')
    const updated: ChatSession = {
      ...session,
      messages: msgs.filter(m => !m.isStreaming),
      title: session.title === 'New Chat' && firstUser ? generateTitle(firstUser.content) : session.title,
      model,
      transport,
      updatedAt: Date.now(),
    }
    saveSession(updated)
    setSessions(loadSessions())
  }, [model, transport])

  const updateMessage = (id: string, updates: Partial<ChatMessage>) => {
    setMessages(prev => prev.map(m => m.id === id ? { ...m, ...updates } : m))
  }

  const handleNewChat = () => {
    setActiveSessionId(null)
    setMessages([])
    mcpClientRef.current?.disconnect()
    mcpClientRef.current = null
  }

  const handleSelectSession = (session: ChatSession) => {
    if (session.id === activeSessionId) return
    const prevTransport = transport
    setActiveSessionId(session.id)
    setMessages(session.messages)
    setModel(session.model)
    setTransport(session.transport as Transport)
    if (session.transport !== prevTransport) {
      mcpClientRef.current?.disconnect()
      mcpClientRef.current = null
    }
  }

  const handleDeleteSession = (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    deleteSession(id)
    const updated = loadSessions()
    setSessions(updated)
    if (activeSessionId === id) {
      if (updated.length > 0) {
        handleSelectSession(updated[0])
      } else {
        setActiveSessionId(null)
        setMessages([])
      }
    }
  }

  const updateTitle = (newTitle: string) => {
    if (!activeSessionId || !newTitle.trim()) return
    const allSessions = loadSessions()
    const session = allSessions.find(s => s.id === activeSessionId)
    if (!session) return
    const updated = { ...session, title: newTitle.trim(), updatedAt: Date.now() }
    saveSession(updated)
    setSessions(loadSessions())
  }

  const handleSend = async () => {
    if (!input.trim() || loading) return
    if (!serverConfig?.hasKey) {
      setError('OpenAI API key is not configured on the server. Set openai_api_key in gateway.yaml or OPENAI_API_KEY env var.')
      return
    }
    setError('')

    // Create a new session if none is active
    let currentSessionId = activeSessionIdRef.current
    if (!currentSessionId) {
      const newSession = createSession(model, transport)
      saveSession(newSession)
      setSessions(loadSessions())
      setActiveSessionId(newSession.id)
      activeSessionIdRef.current = newSession.id
      currentSessionId = newSession.id
    }

    const userText = input.trim()
    setInput('')
    setLoading(true)

    const userMsg: ChatMessage = { id: nextId(), role: 'user', content: userText }
    const newMessages = [...messages, userMsg]
    setMessages(newMessages)
    saveCurrentSession(newMessages, currentSessionId)

    try {
      const client = await getMCPClient()
      const tools = await client.listTools()

      // The API key is managed server-side. We point the SDK at our proxy
      // endpoint (/_api/chat) which injects the real key before forwarding.
      const openaiClient = new OpenAI({
        apiKey: 'server-managed',
        baseURL: `${window.location.origin}/_api`,
        dangerouslyAllowBrowser: true,
      })

      const history: OpenAI.ChatCompletionMessageParam[] = [
        { role: 'system', content: systemPrompt },
        ...messages.filter(m => m.role === 'user' || m.role === 'assistant').map(m => ({
          role: m.role as 'user' | 'assistant',
          content: m.content,
        })),
        { role: 'user', content: userText },
      ]

      const openaiTools: OpenAI.ChatCompletionTool[] = tools.map(t => ({
        type: 'function' as const,
        function: {
          name: t.name,
          description: t.description,
          parameters: t.inputSchema as OpenAI.FunctionParameters,
        },
      }))

      let continueLoop = true
      const loopHistory = [...history]
      let currentMsgs = newMessages

      while (continueLoop) {
        const assistantId = nextId()
        const streamingMsg: ChatMessage = { id: assistantId, role: 'assistant', content: '', isStreaming: true }
        currentMsgs = [...currentMsgs, streamingMsg]
        setMessages(currentMsgs)

        let assistantContent = ''

        const stream = await openaiClient.chat.completions.create({
          model,
          messages: loopHistory,
          tools: openaiTools.length > 0 ? openaiTools : undefined,
          stream: true,
        })

        const toolCallsMap: Record<number, { id: string; name: string; args: string }> = {}

        for await (const chunk of stream) {
          const delta = chunk.choices[0]?.delta
          if (delta?.content) {
            assistantContent += delta.content
            updateMessage(assistantId, { content: assistantContent })
          }
          if (delta?.tool_calls) {
            for (const tc of delta.tool_calls) {
              const idx = tc.index
              if (!toolCallsMap[idx]) toolCallsMap[idx] = { id: tc.id ?? '', name: '', args: '' }
              if (tc.id) toolCallsMap[idx].id = tc.id
              if (tc.function?.name) toolCallsMap[idx].name += tc.function.name
              if (tc.function?.arguments) toolCallsMap[idx].args += tc.function.arguments
            }
          }
        }

        updateMessage(assistantId, { content: assistantContent, isStreaming: false })
        currentMsgs = currentMsgs.map(m => m.id === assistantId ? { ...m, content: assistantContent, isStreaming: false } : m)

        const finalToolCalls = Object.values(toolCallsMap)
        if (finalToolCalls.length === 0) {
          continueLoop = false
          loopHistory.push({ role: 'assistant', content: assistantContent })
          saveCurrentSession(currentMsgs, currentSessionId)
        } else {
          const openaiToolCalls: OpenAI.ChatCompletionMessageToolCall[] = finalToolCalls.map(tc => ({
            id: tc.id,
            type: 'function' as const,
            function: { name: tc.name, arguments: tc.args },
          }))
          loopHistory.push({ role: 'assistant', content: assistantContent || null, tool_calls: openaiToolCalls } as OpenAI.ChatCompletionMessageParam)

          for (const tc of finalToolCalls) {
            let args: Record<string, unknown> = {}
            try { args = JSON.parse(tc.args) as Record<string, unknown> } catch {}

            const tcMsg: ChatMessage = { id: nextId(), role: 'tool_call', content: '', toolName: tc.name, toolArgs: args }
            currentMsgs = [...currentMsgs, tcMsg]
            setMessages(currentMsgs)

            let result = ''
            try {
              result = await client.callTool(tc.name, args)
            } catch (e: unknown) {
              result = `Error: ${e instanceof Error ? e.message : String(e)}`
            }

            const trMsg: ChatMessage = { id: nextId(), role: 'tool_result', content: result }
            currentMsgs = [...currentMsgs, trMsg]
            setMessages(currentMsgs)
            loopHistory.push({ role: 'tool', tool_call_id: tc.id, content: result })
          }
          saveCurrentSession(currentMsgs, currentSessionId)
        }
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Unknown error'
      setError(msg)
      const errMsg: ChatMessage = { id: nextId(), role: 'assistant', content: `Error: ${msg}` }
      const withErr = [...messages, errMsg]
      setMessages(withErr)
      saveCurrentSession(withErr, currentSessionId)
    } finally {
      setLoading(false)
    }
  }

  const activeSession = sessions.find(s => s.id === activeSessionId)

  return (
    <div className="flex h-full">
      {/* Sessions sidebar */}
      <div className="w-48 border-r border-gray-800 bg-gray-900 flex flex-col flex-shrink-0">
        <div className="flex items-center justify-between px-3 py-2.5 border-b border-gray-800">
          <span className="text-sm font-medium text-gray-300">Chats</span>
          <button onClick={handleNewChat}
            className="p-1 text-gray-400 hover:text-white transition-colors rounded"
            title="New Chat">
            <Plus className="w-4 h-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          {sessions.length === 0 ? (
            <p className="text-xs text-gray-500 text-center mt-6 px-3">No chats yet</p>
          ) : (
            sessions.map(s => (
              <div key={s.id}
                onClick={() => handleSelectSession(s)}
                className={cn(
                  'group relative px-3 py-2 cursor-pointer flex flex-col gap-0.5',
                  s.id === activeSessionId
                    ? 'bg-gray-800 border-l-2 border-blue-500'
                    : 'hover:bg-gray-800/50 border-l-2 border-transparent',
                )}>
                <span className="text-xs text-gray-200 truncate pr-5">{s.title}</span>
                <span className="text-xs text-gray-500">{formatSessionDate(s.updatedAt)}</span>
                <button
                  onClick={e => handleDeleteSession(s.id, e)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-500 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Messages area */}
      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-800 gap-2">
          {editingTitle ? (
            <input
              autoFocus
              value={titleDraft}
              onChange={e => setTitleDraft(e.target.value)}
              onBlur={() => { updateTitle(titleDraft); setEditingTitle(false) }}
              onKeyDown={e => {
                if (e.key === 'Enter') { updateTitle(titleDraft); setEditingTitle(false) }
                if (e.key === 'Escape') setEditingTitle(false)
              }}
              className="flex-1 bg-gray-800 border border-gray-600 rounded px-2 py-0.5 text-sm text-white focus:outline-none focus:border-blue-500"
            />
          ) : (
            <button
              onClick={() => { setTitleDraft(activeSession?.title ?? 'New Chat'); setEditingTitle(true) }}
              className="flex-1 text-left text-sm text-gray-300 hover:text-white truncate"
              title="Click to rename">
              {activeSession?.title ?? 'New Chat'}
            </button>
          )}
          <span className="text-xs text-gray-500 flex-shrink-0">{model} · {transport.toUpperCase()}</span>
        </div>

        <div className="flex-1 overflow-y-auto py-4">
          {messages.length === 0 && (
            <div className="text-center text-gray-500 mt-20">
              <p className="text-lg font-medium">MCP Tool Tester</p>
              <p className="text-sm mt-1">Ask anything — tools from connected MCP servers are available</p>
            </div>
          )}
          {messages.map(msg => <MessageBubble key={msg.id} msg={msg} />)}
          <div ref={messagesEndRef} />
        </div>

        {error && (
          <div className="mx-4 mb-2 px-3 py-2 bg-red-900/30 border border-red-800 text-red-400 text-sm rounded-lg">{error}</div>
        )}

        <div className="p-4 border-t border-gray-800">
          <div className="flex gap-3 items-end">
            <textarea
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); void handleSend() } }}
              placeholder="Message (Enter to send, Shift+Enter for newline)"
              rows={2}
              disabled={loading}
              className="flex-1 bg-gray-800 border border-gray-700 rounded-xl px-4 py-3 text-white text-sm resize-none focus:outline-none focus:border-blue-500 disabled:opacity-50"
            />
            <button onClick={() => { void handleSend() }} disabled={loading || !input.trim()}
              className="p-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 text-white rounded-xl transition-colors">
              {loading ? <Loader2 className="w-5 h-5 animate-spin" /> : <Send className="w-5 h-5" />}
            </button>
          </div>
        </div>
      </div>

      {/* Settings panel */}
      <div className={cn('border-l border-gray-800 bg-gray-900 flex flex-col transition-all duration-200 flex-shrink-0',
        settingsOpen ? 'w-64' : 'w-10')}>
        <button onClick={() => setSettingsOpen(!settingsOpen)}
          className="flex items-center gap-2 p-3 text-gray-400 hover:text-white transition-colors border-b border-gray-800">
          <Settings className="w-5 h-5 flex-shrink-0" />
          {settingsOpen && <span className="text-sm font-medium">Settings</span>}
        </button>
        {settingsOpen && (
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {/* Key status badge */}
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-gray-800 border border-gray-700">
              <KeyRound className="w-4 h-4 flex-shrink-0 text-gray-400" />
              <div className="min-w-0">
                <p className="text-xs text-gray-400">OpenAI API Key</p>
                {serverConfig === null ? (
                  <p className="text-xs text-gray-500">Loading…</p>
                ) : serverConfig.hasKey ? (
                  <p className="text-xs text-green-400 font-medium">Configured on server ✓</p>
                ) : (
                  <p className="text-xs text-red-400 font-medium">Not set — add to gateway.yaml</p>
                )}
              </div>
            </div>
            <div>
              <label className="text-xs text-gray-400 block mb-1">Model</label>
              <select value={model} onChange={e => setModel(e.target.value)}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500">
                {MODELS.map(m => <option key={m} value={m}>{m}</option>)}
              </select>
              {serverConfig && (
                <p className="text-xs text-gray-500 mt-1">Server default: {serverConfig.model}</p>
              )}
            </div>
            <div>
              <label className="text-xs text-gray-400 block mb-2">Transport</label>
              <div className="flex gap-2">
                {(['sse', 'http'] as Transport[]).map(t => (
                  <button key={t} onClick={() => setTransport(t)}
                    className={cn('flex-1 py-1.5 text-sm rounded-lg border transition-colors',
                      transport === t ? 'bg-blue-600 border-blue-500 text-white' : 'bg-gray-800 border-gray-700 text-gray-400 hover:text-white')}>
                    {t.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="text-xs text-gray-400 block mb-1">System Prompt</label>
              <textarea value={systemPrompt} onChange={e => setSystemPrompt(e.target.value)}
                rows={4}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm resize-none focus:outline-none focus:border-blue-500" />
            </div>
            <button onClick={saveSettings}
              className="w-full bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 text-sm font-medium transition-colors">
              Save Settings
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
