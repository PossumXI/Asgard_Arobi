import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import electron from 'vite-plugin-electron';
import renderer from 'vite-plugin-electron-renderer';
import path from 'path';

export default defineConfig({
  plugins: [
    react(),
    electron([
      {
        entry: 'electron/main.ts',
        vite: {
          build: {
            outDir: 'dist-electron',
            rollupOptions: {
              external: ['electron', 'electron-updater', 'electron-store'],
            },
          },
        },
      },
      {
        entry: 'electron/preload.ts',
        onstart(options) {
          options.reload();
        },
        vite: {
          build: {
            outDir: 'dist-electron',
            rollupOptions: {
              external: ['electron'],
            },
          },
        },
      },
    ]),
    renderer(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@electron': path.resolve(__dirname, './electron'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'three-vendor': ['three', '@react-three/fiber', '@react-three/drei'],
          'framer-motion': ['framer-motion'],
          'react-query': ['@tanstack/react-query'],
          'media-vendor': ['hls.js', 'peerjs'],
          'radix-ui': [
            '@radix-ui/react-dialog',
            '@radix-ui/react-dropdown-menu',
            '@radix-ui/react-select',
            '@radix-ui/react-slider',
            '@radix-ui/react-tabs',
            '@radix-ui/react-tooltip',
          ],
          'utils-vendor': ['zustand', 'clsx', 'class-variance-authority', 'tailwind-merge'],
        },
      },
    },
    target: 'esnext',
    minify: 'esbuild',
  },
  server: {
    port: 3002,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
      '/api/security': {
        target: 'http://localhost:9090',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/security/, '/api'),
      },
      '/api/valkyrie': {
        target: 'http://localhost:8093',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/valkyrie/, '/api/v1'),
      },
    },
  },
});
