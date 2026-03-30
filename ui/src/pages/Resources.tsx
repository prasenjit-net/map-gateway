import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listResources, createResourceFile, createResourceText, createResourceUpstream,
  deleteResource, type ResourceRecord
} from '../lib/api'
import { Plus, Trash2, X, Upload, FileText, Link2, Shield, Cookie, Copy, Eye } from 'lucide-react'
import { cn } from '../lib/utils'

function TypeBadge({ type }: { type: string }) {
  const styles: Record<string, string> = {
    file: 'bg-blue-900/50 text-blue-300 border-blue-800',
    text: 'bg-green-900/50 text-green-300 border-green-800',
    upstream: 'bg-purple-900/50 text-purple-300 border-purple-800',
  }
  return (
    <span className={cn('px-1.5 py-0.5 text-xs rounded border', styles[type] ?? 'bg-gray-800 text-gray-300 border-gray-700')}>
      {type}
    </span>
  )
}

function MCPUri({ record }: { record: ResourceRecord }) {
  const [copied, setCopied] = useState(false)
  const uri = record.is_template && record.uri_template
    ? record.uri_template
    : `gateway://resources/${record.id}`
  const copy = () => {
    navigator.clipboard.writeText(uri).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1500) })
  }
  return (
    <div className="flex items-center gap-1 max-w-xs">
      <span className="text-xs font-mono text-gray-400 truncate">{uri}</span>
      <button onClick={copy} className="text-gray-600 hover:text-gray-300 flex-shrink-0">
        <Copy className="w-3 h-3" />
      </button>
      {copied && <span className="text-xs text-green-400">copied</span>}
    </div>
  )
}

type TabType = 'file' | 'text' | 'upstream'

function AddDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<TabType>('file')
  const [error, setError] = useState('')

  const [fileName, setFileName] = useState('')
  const [fileDesc, setFileDesc] = useState('')
  const [fileMime, setFileMime] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [dragOver, setDragOver] = useState(false)

  const [textName, setTextName] = useState('')
  const [textDesc, setTextDesc] = useState('')
  const [textMime, setTextMime] = useState('text/plain')
  const [textContent, setTextContent] = useState('')

  const [upName, setUpName] = useState('')
  const [upDesc, setUpDesc] = useState('')
  const [upUrl, setUpUrl] = useState('')
  const [upMime, setUpMime] = useState('application/json')
  const [upUri, setUpUri] = useState('')
  const [upAuth, setUpAuth] = useState(false)
  const [upCookies, setUpCookies] = useState(false)
  const [upHeaders, setUpHeaders] = useState('')

  const onSuccess = () => {
    void queryClient.invalidateQueries({ queryKey: ['resources'] })
    onClose()
  }

  const fileMutation = useMutation({
    mutationFn: () => createResourceFile(file!, { name: fileName, description: fileDesc, mime_type: fileMime || undefined }),
    onSuccess,
    onError: (e: Error) => setError(e.message),
  })

  const textMutation = useMutation({
    mutationFn: () => createResourceText({ name: textName, description: textDesc, mime_type: textMime, content: textContent }),
    onSuccess,
    onError: (e: Error) => setError(e.message),
  })

  const upstreamMutation = useMutation({
    mutationFn: () => createResourceUpstream({
      name: upName, description: upDesc, mime_type: upMime,
      upstream_url: upUrl, uri_template: upUri || undefined,
      passthrough_auth: upAuth, passthrough_cookies: upCookies,
      passthrough_headers: upHeaders.split(',').map(h => h.trim()).filter(Boolean),
    }),
    onSuccess,
    onError: (e: Error) => setError(e.message),
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (tab === 'file') {
      if (!file) { setError('Please select a file'); return }
      if (!fileName) { setError('Name is required'); return }
      fileMutation.mutate()
    } else if (tab === 'text') {
      if (!textName) { setError('Name is required'); return }
      if (!textContent) { setError('Content is required'); return }
      textMutation.mutate()
    } else {
      if (!upName) { setError('Name is required'); return }
      if (!upUrl) { setError('Upstream URL is required'); return }
      upstreamMutation.mutate()
    }
  }

  const isPending = fileMutation.isPending || textMutation.isPending || upstreamMutation.isPending

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault(); setDragOver(false)
    const f = e.dataTransfer.files[0]
    if (f) { setFile(f); if (!fileName) setFileName(f.name.replace(/\.[^.]+$/, '')) }
  }, [fileName])

  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative w-full max-w-lg bg-gray-900 border-l border-gray-700 shadow-2xl flex flex-col h-full overflow-y-auto">
        <div className="flex items-center justify-between p-5 border-b border-gray-800">
          <h3 className="text-lg font-semibold text-white">Add Resource</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white"><X className="w-5 h-5" /></button>
        </div>

        <div className="flex border-b border-gray-800">
          {(['file', 'text', 'upstream'] as TabType[]).map(t => (
            <button key={t} onClick={() => setTab(t)}
              className={cn('flex-1 py-3 text-sm font-medium transition-colors capitalize flex items-center justify-center gap-1.5',
                tab === t ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-200')}>
              {t === 'file' && <Upload className="w-4 h-4" />}
              {t === 'text' && <FileText className="w-4 h-4" />}
              {t === 'upstream' && <Link2 className="w-4 h-4" />}
              {t === 'file' ? 'File Upload' : t === 'text' ? 'Text Block' : 'Upstream URL'}
            </button>
          ))}
        </div>

        <form onSubmit={handleSubmit} className="flex-1 p-5 space-y-4">
          <div>
            <label className="text-sm text-gray-400 block mb-1">Name <span className="text-red-400">*</span></label>
            <input value={tab === 'file' ? fileName : tab === 'text' ? textName : upName}
              onChange={e => tab === 'file' ? setFileName(e.target.value) : tab === 'text' ? setTextName(e.target.value) : setUpName(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500" />
          </div>
          <div>
            <label className="text-sm text-gray-400 block mb-1">Description</label>
            <input value={tab === 'file' ? fileDesc : tab === 'text' ? textDesc : upDesc}
              onChange={e => tab === 'file' ? setFileDesc(e.target.value) : tab === 'text' ? setTextDesc(e.target.value) : setUpDesc(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500" />
          </div>

          {tab === 'file' && (
            <>
              <div>
                <label className="text-sm text-gray-400 block mb-1">MIME type <span className="text-gray-600">(auto-detected)</span></label>
                <input value={fileMime} onChange={e => setFileMime(e.target.value)}
                  placeholder="e.g. text/plain, image/png"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500" />
              </div>
              <div onDrop={handleDrop} onDragOver={e => { e.preventDefault(); setDragOver(true) }} onDragLeave={() => setDragOver(false)}
                className={cn('border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors',
                  dragOver ? 'border-blue-500 bg-blue-900/20' : 'border-gray-700 hover:border-gray-500')}
                onClick={() => document.getElementById('resource-file-input')?.click()}>
                <Upload className="w-8 h-8 text-gray-500 mx-auto mb-2" />
                {file ? <p className="text-sm text-green-400">{file.name}</p>
                  : <p className="text-sm text-gray-400">Drop any file or click to browse</p>}
                <input id="resource-file-input" type="file" className="hidden"
                  onChange={e => { const f = e.target.files?.[0]; if (f) { setFile(f); if (!fileName) setFileName(f.name.replace(/\.[^.]+$/, '')) } }} />
              </div>
            </>
          )}

          {tab === 'text' && (
            <>
              <div>
                <label className="text-sm text-gray-400 block mb-1">MIME type</label>
                <select value={textMime} onChange={e => setTextMime(e.target.value)}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500">
                  <option value="text/plain">text/plain</option>
                  <option value="text/markdown">text/markdown</option>
                  <option value="application/json">application/json</option>
                  <option value="text/html">text/html</option>
                  <option value="application/xml">application/xml</option>
                  <option value="text/csv">text/csv</option>
                </select>
              </div>
              <div>
                <label className="text-sm text-gray-400 block mb-1">Content <span className="text-red-400">*</span></label>
                <textarea value={textContent} onChange={e => setTextContent(e.target.value)} rows={10}
                  placeholder="Paste or type your content here..."
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-blue-500 resize-y" />
              </div>
            </>
          )}

          {tab === 'upstream' && (
            <>
              <div>
                <label className="text-sm text-gray-400 block mb-1">Upstream URL <span className="text-red-400">*</span></label>
                <input value={upUrl} onChange={e => setUpUrl(e.target.value)}
                  placeholder="https://api.example.com/data/{id}"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-blue-500" />
                {upUrl.includes('{') && (
                  <p className="text-xs text-purple-400 mt-1">⚡ Template detected — will be exposed as resource template</p>
                )}
              </div>
              <div>
                <label className="text-sm text-gray-400 block mb-1">MCP URI Template <span className="text-gray-600">(optional)</span></label>
                <input value={upUri} onChange={e => setUpUri(e.target.value)}
                  placeholder="gateway://my-resource/{id}"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-blue-500" />
              </div>
              <div>
                <label className="text-sm text-gray-400 block mb-1">Response MIME type</label>
                <input value={upMime} onChange={e => setUpMime(e.target.value)}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500" />
              </div>
              <div className="space-y-2">
                <label className="flex items-center gap-3 cursor-pointer">
                  <input type="checkbox" checked={upAuth} onChange={e => setUpAuth(e.target.checked)} className="w-4 h-4 rounded border-gray-600 bg-gray-800" />
                  <span className="text-sm text-gray-300 flex items-center gap-1.5"><Shield className="w-4 h-4 text-blue-400" /> Forward Authorization header</span>
                </label>
                <label className="flex items-center gap-3 cursor-pointer">
                  <input type="checkbox" checked={upCookies} onChange={e => setUpCookies(e.target.checked)} className="w-4 h-4 rounded border-gray-600 bg-gray-800" />
                  <span className="text-sm text-gray-300 flex items-center gap-1.5"><Cookie className="w-4 h-4 text-blue-400" /> Forward Cookie header</span>
                </label>
                <div>
                  <label className="text-sm text-gray-400 block mb-1">Additional headers (comma-separated)</label>
                  <input value={upHeaders} onChange={e => setUpHeaders(e.target.value)} placeholder="X-Custom-Header"
                    className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-blue-500" />
                </div>
              </div>
            </>
          )}

          {error && <div className="text-red-400 text-sm bg-red-900/20 border border-red-800 rounded p-2">{error}</div>}
          <button type="submit" disabled={isPending}
            className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 text-white rounded-lg py-2.5 text-sm font-medium transition-colors">
            {isPending ? 'Saving…' : 'Add Resource'}
          </button>
        </form>
      </div>
    </div>
  )
}

function PreviewPanel({ record, onClose }: { record: ResourceRecord; onClose: () => void }) {
  const [content, setContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const fetchContent = async () => {
    if (record.type === 'upstream') return
    setLoading(true)
    try {
      const res = await fetch(`/_api/resources/${record.id}/content`)
      if (res.ok) {
        if (record.mime_type.startsWith('image/')) {
          const blob = await res.blob()
          setContent(URL.createObjectURL(blob))
        } else {
          setContent(await res.text())
        }
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative bg-gray-900 border border-gray-700 rounded-xl w-full max-w-2xl mx-4 flex flex-col max-h-[80vh]">
        <div className="flex items-center justify-between p-4 border-b border-gray-800">
          <div>
            <h3 className="font-semibold text-white">{record.name}</h3>
            <p className="text-xs text-gray-500 font-mono mt-0.5">
              {record.is_template && record.uri_template ? record.uri_template : `gateway://resources/${record.id}`}
            </p>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-white"><X className="w-5 h-5" /></button>
        </div>
        <div className="p-4 overflow-auto flex-1">
          {record.type === 'upstream' ? (
            <div className="space-y-2">
              <p className="text-sm text-gray-400">Upstream URL:</p>
              <p className="text-sm font-mono text-purple-300 break-all">{record.upstream_url}</p>
              {record.is_template && <span className="text-xs bg-purple-900/50 text-purple-300 border border-purple-800 px-1.5 py-0.5 rounded">template</span>}
            </div>
          ) : content === null ? (
            <button onClick={fetchContent} disabled={loading}
              className="flex items-center gap-2 text-sm text-blue-400 hover:text-blue-300">
              <Eye className="w-4 h-4" /> {loading ? 'Loading…' : 'Load preview'}
            </button>
          ) : record.mime_type.startsWith('image/') ? (
            <img src={content} className="max-w-full rounded" />
          ) : (
            <pre className="text-xs text-gray-300 whitespace-pre-wrap font-mono">{content}</pre>
          )}
        </div>
      </div>
    </div>
  )
}

export default function Resources() {
  const queryClient = useQueryClient()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [preview, setPreview] = useState<ResourceRecord | null>(null)

  const { data: resources, isLoading, error } = useQuery({
    queryKey: ['resources'],
    queryFn: listResources,
  })

  const deleteMutation = useMutation({
    mutationFn: deleteResource,
    onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ['resources'] }); setDeleteId(null) },
  })

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-white">Resources</h2>
          <p className="text-sm text-gray-400 mt-0.5">MCP resources exposed to LLM clients as readable context</p>
        </div>
        <button onClick={() => setDrawerOpen(true)}
          className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
          <Plus className="w-4 h-4" /> Add Resource
        </button>
      </div>

      {isLoading && <div className="text-gray-400">Loading resources…</div>}
      {error && <div className="text-red-400">Failed to load resources</div>}

      {resources && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-800 text-left">
                <th className="px-4 py-3 text-xs font-medium text-gray-400 uppercase tracking-wider">Name</th>
                <th className="px-4 py-3 text-xs font-medium text-gray-400 uppercase tracking-wider">Type</th>
                <th className="px-4 py-3 text-xs font-medium text-gray-400 uppercase tracking-wider">MCP URI</th>
                <th className="px-4 py-3 text-xs font-medium text-gray-400 uppercase tracking-wider">MIME</th>
                <th className="px-4 py-3 text-xs font-medium text-gray-400 uppercase tracking-wider">Created</th>
                <th className="px-4 py-3 text-xs font-medium text-gray-400 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              {resources.length === 0 && (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No resources yet. Add one to get started.</td></tr>
              )}
              {resources.map((res: ResourceRecord) => (
                <tr key={res.id} className="hover:bg-gray-800/50 transition-colors">
                  <td className="px-4 py-3">
                    <div className="text-white font-medium">{res.name}</div>
                    {res.description && <div className="text-xs text-gray-500 mt-0.5">{res.description}</div>}
                  </td>
                  <td className="px-4 py-3"><TypeBadge type={res.type} /></td>
                  <td className="px-4 py-3"><MCPUri record={res} /></td>
                  <td className="px-4 py-3"><span className="text-xs text-gray-400 font-mono">{res.mime_type || '—'}</span></td>
                  <td className="px-4 py-3 text-gray-400 text-sm">{new Date(res.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <button onClick={() => setPreview(res)} className="p-1.5 text-gray-400 hover:text-blue-400 transition-colors rounded hover:bg-gray-700">
                        <Eye className="w-4 h-4" />
                      </button>
                      <button onClick={() => setDeleteId(res.id)} className="p-1.5 text-gray-400 hover:text-red-400 transition-colors rounded hover:bg-gray-700">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <AddDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />

      {preview && <PreviewPanel record={preview} onClose={() => setPreview(null)} />}

      {deleteId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60" onClick={() => setDeleteId(null)} />
          <div className="relative bg-gray-900 border border-gray-700 rounded-xl p-6 max-w-sm w-full mx-4">
            <h3 className="text-lg font-semibold text-white mb-2">Delete Resource</h3>
            <p className="text-gray-400 text-sm mb-5">This will permanently remove the resource and any stored file.</p>
            <div className="flex gap-3">
              <button onClick={() => setDeleteId(null)} className="flex-1 bg-gray-800 hover:bg-gray-700 text-white rounded-lg py-2 text-sm transition-colors">Cancel</button>
              <button onClick={() => deleteMutation.mutate(deleteId)} disabled={deleteMutation.isPending}
                className="flex-1 bg-red-600 hover:bg-red-700 disabled:bg-red-800 text-white rounded-lg py-2 text-sm transition-colors">
                {deleteMutation.isPending ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
