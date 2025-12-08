import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    strictPort: false,
    watch: {
      usePolling: true
    },
    // Configuration HMR pour fonctionner via Nginx
    hmr: {
      clientPort: 3300, // Port exposé par Nginx
      protocol: 'ws',
      host: 'localhost',
      path: '/@vite/client'
    },
    // Désactiver la vérification d'origine pour permettre les connexions depuis Nginx
    cors: true,
    // Configuration pour accepter les connexions depuis le proxy Nginx
    // Vite vérifie l'en-tête Host, on doit permettre les connexions depuis nginx
    allowedHosts: [
      'localhost',
      '.localhost',
      '127.0.0.1',
      'nginx',
      'frontend'
    ]
  }
})

