import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { nodePolyfills } from 'vite-plugin-node-polyfills'

export default defineConfig({
  plugins: [
    nodePolyfills({
      include: ['buffer', 'process', 'events', 'stream', 'util', 'crypto'],
      globals: { global: true, process: true, Buffer: true },
    }),
    react(),
    tailwindcss(),
  ],
})
