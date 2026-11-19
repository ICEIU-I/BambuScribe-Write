import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { open } from './tableHarness.mjs'
export const scene = JSON.parse(
  readFileSync(new URL('../../../docs/diagram-fit-2026-10-09/first.json', import.meta.url)),
)
export const png = readFileSync(
  new URL('../../../internal/engine/testdata/hall-switch.png', import.meta.url),
)

export async function setup(page, delayed = false) {
  const errors = await open(page)
  await page.evaluate(
    ({ scene, delayed }) => {
      window.__diagram = { calls: 0, cancels: 0, active: 0, peak: 0, requests: [], delayed, scene }
      Object.assign(window.go.application.App, {
        AgentConfig: async () => ({
          baseURL: 'https://example.invalid/v1',
          model: 'gpt-6.1-sol',
          protocol: 'responses',
          hasKey: true,
        }),
        SaveAgentConfig: async () => {},
        RecognizeDiagram: async (source, instruction) => {
          const state = window.__diagram
          state.calls++
          state.active++
          state.peak = Math.max(state.peak, state.active)
          const request = { source, instruction }
          state.requests.push(request)
          try {
            const result = state.delayed
              ? await new Promise((resolve, reject) => {
                  Object.assign(request, { resolve, reject })
                  Object.assign(state, { resolve, reject })
                })
              : null
            return result ?? state.scene
          } finally {
            state.active--
          }
        },
        CancelDiagramRecognition: async () => {
          window.__diagram.cancels++
        },
      })
    },
    { scene, delayed },
  )
  return errors
}
export async function menu(page) {
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入图像', exact: true }).click()
}
export async function start(page) {
  await menu(page)
  await page
    .getByLabel('图像文件', { exact: true })
    .setInputFiles({ name: 'diagram.png', mimeType: 'image/png', buffer: png })
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
}
export async function settled(page) {
  await expect(page.locator('[data-reorder-enabled]').first()).toHaveAttribute(
    'data-reorder-enabled',
    'true',
  )
}
