import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  cacheDir: '/tmp/.vite',
  plugins: [react()],
  test: {
    environment: 'jsdom',
  },
})
