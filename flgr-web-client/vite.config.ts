import { fileURLToPath } from 'node:url'
import { coverageConfigDefaults, defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      // Keeps the browser same-origin with the backend locally, so the
      // session cookie behaves exactly as it does in production.
      // See docs/architecture/adr/0012-local-development-environment.md.
      '/api': {
        target: process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    // jsdom has no real page origin by default, which breaks relative
    // fetch() URLs (e.g. RTK Query's "/api/v1/health"); give it one.
    environmentOptions: {
      jsdom: { url: 'http://localhost:3000' },
    },
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      // shadcn/ui's generated primitives (src/components/ui) are vendored,
      // in-repo Radix wrappers — exercised through the app's own component
      // tests, but chasing every internal branch (e.g. every Select
      // sub-part, every Dialog prop default) inside code we didn't author
      // adds no real signal. Test harness files under src/test are
      // likewise infrastructure, not app logic. Both are excluded from the
      // 100% threshold below; everything else (features, app shell,
      // shared lib code) is still held to it with no exceptions.
      exclude: [...coverageConfigDefaults.exclude, 'src/components/ui/**', 'src/test/**'],
      thresholds: {
        lines: 100,
        branches: 100,
        functions: 100,
        statements: 100,
      },
    },
  },
})
