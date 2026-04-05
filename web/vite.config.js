import react from '@vitejs/plugin-react';
import { defineConfig, transformWithEsbuild } from 'vite';
import path from 'path';
import { codeInspectorPlugin } from 'code-inspector-plugin';

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@douyinfe/semi-ui/lib/es/typography/text': path.resolve(
        __dirname,
        './src/lib/semi-typography-text.jsx',
      ),
      '@douyinfe/semi-ui/lib/es/typography/title': path.resolve(
        __dirname,
        './src/lib/semi-typography-title.jsx',
      ),
      '@douyinfe/semi-ui': path.resolve(
        __dirname,
        './src/lib/semi-ui-compat.jsx',
      ),
      '@douyinfe/semi-icons': path.resolve(
        __dirname,
        './src/lib/semi-icons-compat.jsx',
      ),
    },
  },
  plugins: [
    codeInspectorPlugin({
      bundler: 'vite',
    }),
    {
      name: 'treat-js-files-as-jsx',
      async transform(code, id) {
        if (!/src\/.*\.js$/.test(id)) {
          return null;
        }

        // Use the exposed transform from vite, instead of directly
        // transforming with esbuild
        return transformWithEsbuild(code, id, {
          loader: 'jsx',
          jsx: 'automatic',
        });
      },
    },
    react(),
  ],
  optimizeDeps: {
    force: true,
    esbuildOptions: {
      loader: {
        '.js': 'jsx',
        '.json': 'json',
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'react-core': ['react', 'react-dom', 'react-router-dom'],
          tools: ['axios', 'history', 'marked'],
          'react-components': [
            'react-dropzone',
            'react-fireworks',
            'react-telegram-login',
            'react-toastify',
            'react-turnstile',
          ],
          i18n: [
            'i18next',
            'react-i18next',
            'i18next-browser-languagedetector',
          ],
        },
      },
    },
  },
  server: {
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://localhost:5946',
        changeOrigin: true,
      },
      '/mj': {
        target: 'http://localhost:5946',
        changeOrigin: true,
      },
      '/pg': {
        target: 'http://localhost:5946',
        changeOrigin: true,
      },
    },
  },
});
