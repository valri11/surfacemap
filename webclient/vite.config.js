
const { resolve } = require('path')
const { defineConfig } = require('vite')

module.exports = defineConfig({
  build: {
    sourcemap: true,
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        sealevel: resolve(__dirname, 'sealevel.html'),
        terra3d: resolve(__dirname, 'terra3d.html')
      }
    }
  }
})
