// VITE: a ferramenta que roda a tela em modo de desenvolvimento
// ("npm run dev") e abre em http://localhost:5173.

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// O frontend chama /api/... e o Vite repassa para o backend Go.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
