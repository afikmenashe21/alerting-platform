import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  // For GitHub Pages: use repo name as base path in production
  // Set VITE_BASE_PATH env var or it defaults to '/' for local dev
  base: process.env.VITE_BASE_PATH || '/',
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/alert-producer-api': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/alert-producer-api/, ''),
      },
      '/metrics-api': {
        target: 'http://localhost:8083',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/metrics-api/, ''),
      },
    },
  },
})
