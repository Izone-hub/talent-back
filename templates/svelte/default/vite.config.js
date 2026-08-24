import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  cacheDir: '/tmp/.vite',
  plugins: [svelte()],
  test: {
    environment: 'jsdom',
  },
})
