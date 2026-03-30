export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'tool_call' | 'tool_result'
  content: string
  toolName?: string
  toolArgs?: Record<string, unknown>
  isStreaming?: boolean
}

export interface ChatSession {
  id: string
  title: string
  messages: ChatMessage[]
  model: string
  transport: 'sse' | 'http'
  createdAt: number
  updatedAt: number
}

const STORAGE_KEY = 'mcp_chat_sessions'
const MAX_SESSIONS = 50

export function loadSessions(): ChatSession[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const sessions = JSON.parse(raw) as ChatSession[]
    return sessions.sort((a, b) => b.updatedAt - a.updatedAt)
  } catch {
    return []
  }
}

export function saveSession(session: ChatSession): void {
  const sessions = loadSessions()
  const idx = sessions.findIndex(s => s.id === session.id)
  if (idx >= 0) {
    sessions[idx] = session
  } else {
    sessions.unshift(session)
  }
  const trimmed = sessions
    .sort((a, b) => b.updatedAt - a.updatedAt)
    .slice(0, MAX_SESSIONS)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(trimmed))
}

export function deleteSession(id: string): void {
  const sessions = loadSessions().filter(s => s.id !== id)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions))
}

export function createSession(model: string, transport: 'sse' | 'http'): ChatSession {
  const now = Date.now()
  return {
    id: crypto.randomUUID(),
    title: 'New Chat',
    messages: [],
    model,
    transport,
    createdAt: now,
    updatedAt: now,
  }
}

export function generateTitle(firstUserMessage: string): string {
  return firstUserMessage.replace(/\n/g, ' ').slice(0, 60)
}
