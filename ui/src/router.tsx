import { createRouter, createRootRoute, createRoute } from '@tanstack/react-router'
import { lazy, Suspense } from 'react'
import Layout from './components/Layout'

const Dashboard  = lazy(() => import('./pages/Dashboard'))
const Specs      = lazy(() => import('./pages/Specs'))
const SpecDetail = lazy(() => import('./pages/SpecDetail'))
const Stats      = lazy(() => import('./pages/Stats'))
const Chat       = lazy(() => import('./pages/Chat'))
const Resources  = lazy(() => import('./pages/Resources'))

// Thin wrapper so TanStack Router (which doesn't natively support lazy
// components) can use React.lazy pages without extra boilerplate.
function wrap(Component: React.LazyExoticComponent<() => React.ReactElement | null>) {
  return () => (
    <Suspense fallback={<div className="p-8 text-gray-400 text-sm">Loading…</div>}>
      <Component />
    </Suspense>
  )
}

const rootRoute      = createRootRoute({ component: Layout })
const indexRoute     = createRoute({ getParentRoute: () => rootRoute, path: '/',             component: wrap(Dashboard)  })
const specsRoute     = createRoute({ getParentRoute: () => rootRoute, path: '/specs',        component: wrap(Specs)      })
const specDetailRoute= createRoute({ getParentRoute: () => rootRoute, path: '/specs/$specId',component: wrap(SpecDetail) })
const resourcesRoute = createRoute({ getParentRoute: () => rootRoute, path: '/resources',    component: wrap(Resources)  })
const statsRoute     = createRoute({ getParentRoute: () => rootRoute, path: '/stats',        component: wrap(Stats)      })
const chatRoute      = createRoute({ getParentRoute: () => rootRoute, path: '/chat',         component: wrap(Chat)       })

const routeTree = rootRoute.addChildren([indexRoute, specsRoute, specDetailRoute, resourcesRoute, statsRoute, chatRoute])

export const router = createRouter({ routeTree, basepath: '/_ui' })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
