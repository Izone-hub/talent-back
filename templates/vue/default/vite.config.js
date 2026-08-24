import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  cacheDir: '/tmp/.vite',
  plugins: [vue()],
  test: {
    environment: 'jsdom',
  },
})
