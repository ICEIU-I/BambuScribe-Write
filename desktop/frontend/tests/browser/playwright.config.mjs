import { defineConfig } from '@playwright/test'
import { tmpdir } from 'node:os'
export default defineConfig({
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
    { name: 'webkit', use: { browserName: 'webkit' } },
  ],
  testDir: '.',
  testMatch: '*.spec.mjs',
  workers: 1,
  outputDir: `${tmpdir()}/bambuscribe-eraser-tests`,
  // Pin the OS locale: the interface follows it by default, and the
  // assertions below are written against the Chinese wording.
  use: {
    baseURL: 'http://127.0.0.1:5176',
    viewport: { width: 1360, height: 1000 },
    locale: 'zh-CN',
  },
  webServer: {
    command: 'npm run dev -- --port 5176 --strictPort',
    url: 'http://127.0.0.1:5176',
    reuseExistingServer: false,
  },
})
