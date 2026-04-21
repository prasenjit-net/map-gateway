import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/_ui/',
  build: {
    outDir: 'dist',
    rollupOptions: {
      output: {
        manualChunks(id: string) {
          if (id.includes('node_modules/openai'))                          return 'vendor-openai'
          if (id.includes('node_modules/recharts') ||
              id.includes('node_modules/d3-'))                             return 'vendor-charts'
          if (id.includes('node_modules/lucide-react'))                   return 'vendor-icons'
          if (id.includes('node_modules/@tanstack'))                      return 'vendor-query'
          if (id.includes('node_modules/react') ||
              id.includes('node_modules/react-dom') ||
              id.includes('node_modules/scheduler'))                      return 'vendor-react'
        },
      },
    },
  },
  server: {
    proxy: {
      '/_api': 'http://localhost:9876',
      '/mcp': 'http://localhost:9876',
    },
  },
})
