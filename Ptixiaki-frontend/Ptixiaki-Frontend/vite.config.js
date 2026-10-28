import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    host: 'localhost',
    port: 5173,
    strictPort: true,
    proxy: {
      '/analysis': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/User': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/register': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
