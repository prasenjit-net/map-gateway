import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { LayoutDashboard, FileCode2, BarChart2, MessageSquare, Database } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { getHealth } from '../lib/api'
import { cn } from '../lib/utils'
import Logo from './Logo'

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, exact: true },
  { to: '/specs', label: 'Specs', icon: FileCode2 },
  { to: '/resources', label: 'Resources', icon: Database },
  { to: '/stats', label: 'Stats', icon: BarChart2 },
  { to: '/chat', label: 'Chat', icon: MessageSquare },
]

export default function Layout() {
  const { data: health } = useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    refetchInterval: 30000,
  })
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  return (
    <div className="flex h-screen bg-gray-950 text-gray-100">
      {/* Sidebar */}
      <aside className="w-56 bg-gray-900 border-r border-gray-800 flex flex-col">
        <div className="p-4 border-b border-gray-800">
          <div className="flex items-center gap-2.5">
            <Logo size={32} />
            <div>
              <h1 className="text-sm font-bold text-white leading-tight">MCP Gateway</h1>
              <p className="text-xs text-gray-500 leading-tight">API → MCP bridge</p>
            </div>
          </div>
          <div className="flex items-center gap-1.5 mt-2.5">
            <span className={cn('w-2 h-2 rounded-full', health?.status === 'ok' ? 'bg-green-400' : 'bg-red-400')} />
            <span className="text-xs text-gray-400">{health?.status === 'ok' ? `v${health.version}` : 'offline'}</span>
          </div>
        </div>
        <nav className="flex-1 p-2 space-y-1">
          {navItems.map(({ to, label, icon: Icon, exact }) => {
            const isActive = exact ? currentPath === to : currentPath.startsWith(to) && !(to === '/' && currentPath !== '/')
            return (
              <Link
                key={to}
                to={to}
                className={cn(
                  'flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors',
                  isActive
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-400 hover:bg-gray-800 hover:text-white'
                )}
              >
                <Icon className="w-4 h-4" />
                {label}
              </Link>
            )
          })}
        </nav>
        <div className="p-3 border-t border-gray-800 text-xs text-gray-500">
          {health?.uptime && <div>Uptime: {health.uptime}</div>}
        </div>
      </aside>
      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
