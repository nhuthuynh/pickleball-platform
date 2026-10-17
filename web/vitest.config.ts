import { fileURLToPath } from 'node:url'
import { mergeConfig, defineConfig, configDefaults } from 'vitest/config'
import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      // jsdom provides no fetch, so the global Request is Node's and cannot
      // resolve a relative URL. The API client defaults to a relative base
      // URL (#339), so without this the suite fails on something no browser
      // would do. See the file for why the environment was fixed rather
      // than the ten assertions it broke.
      setupFiles: ['src/test-support/relativeUrlRequest.ts'],
      exclude: [...configDefaults.exclude, 'e2e/**'],
      root: fileURLToPath(new URL('./', import.meta.url)),
    },
  }),
)
