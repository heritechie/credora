import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';

export default defineConfig({
  base: '/workspace/',
  plugins: [preact()],
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
    // SPA fallback: serve index.html for /workspace/* routes so that
    // direct browser navigation and refresh work with client-side routing.
    // This does NOT affect /api/* or /health proxies.
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        if (req.url && req.url.startsWith('/workspace/') && !req.url.includes('.')) {
          req.url = '/workspace/index.html';
        }
        next();
      });
    },
  },
  build: {
    outDir: 'dist',
  },
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.test.{js,jsx}'],
    globals: true,
  },
});