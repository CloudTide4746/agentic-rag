import { defineConfig } from './archive/frontend-legacy-20260921/node_modules/vite/dist/node/index.js'
import vue from './archive/frontend-legacy-20260921/node_modules/@vitejs/plugin-vue/dist/index.mjs'
import tailwindcss from './archive/frontend-legacy-20260921/node_modules/@tailwindcss/vite/dist/index.mjs'
import { fileURLToPath } from 'node:url'

const projectRoot = fileURLToPath(new URL('./archive/frontend-legacy-20260921/', import.meta.url))
const cacheRoot = fileURLToPath(new URL('./.vite-cache/frontend-legacy-20260921/', import.meta.url))

export default defineConfig({
  root: projectRoot,
  cacheDir: cacheRoot,
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./archive/frontend-legacy-20260921/src/', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
})
