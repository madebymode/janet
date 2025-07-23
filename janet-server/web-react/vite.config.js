import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../web/static',
    emptyOutDir: false,
    rollupOptions: {
      input: {
        main: './src/main.jsx'
      },
      output: {
        entryFileNames: 'js/main.js',
        chunkFileNames: 'js/[name].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name.endsWith('.css')) {
            return 'css/main.css'
          }
          return 'assets/[name][extname]'
        }
      }
    }
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
})