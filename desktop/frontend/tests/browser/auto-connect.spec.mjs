import { test, expect } from '@playwright/test'
import { openPrinters, instance } from './printerHarness.mjs'
const dialog = (page) => page.getByRole('dialog', { name: '编辑打印机实例' })
async function editInstance(page, id) {
  await instance(page, id)
    .getByRole('button', { name: /^编辑 / })
    .click()
  await expect(dialog(page)).toBeVisible()
  return dialog(page)
}

test('startup connects saved instances independently and keeps editing available', async ({
  page,
}) => {
  const errors = await openPrinters(page, {
    devices: [
      { id: 'one', name: '左侧 A1', ip: '192.168.1.100', accessCode: 'first-code' },
      { id: 'two', name: '右侧 A1', ip: '192.168.1.101', accessCode: 'second-code' },
    ],
  })
  await expect
    .poll(() => page.evaluate(() => window.__connections.calls))
    .toEqual([
      { id: 'one', ip: '192.168.1.100', accessCode: 'first-code' },
      { id: 'two', ip: '192.168.1.101', accessCode: 'second-code' },
    ])
  await page.locator('.paper-input').fill('两台连接期间仍可编辑')
  await page.getByRole('tab', { name: '设备台' }).click()
  await expect(instance(page, 'one')).toContainText('连接中')
  await expect(instance(page, 'two')).toContainText('连接中')
  await page.evaluate(() => window.__connections.succeed('two'))
  await expect(instance(page, 'two')).toContainText('已连接')
  await expect(instance(page, 'one')).toContainText('连接中')
  await page.evaluate(() => window.__connections.succeed('one'))
  const editor = await editInstance(page, 'one')
  await editor.getByRole('button', { name: '断开连接' }).click()
  await expect(editor.getByRole('button', { name: '连接打印机' })).toBeEnabled()
  await editor.getByRole('button', { name: '关闭实例编辑' }).click()
  await page.waitForTimeout(2200)
  expect(await page.evaluate(() => window.__connections.calls.length)).toBe(2)
  await expect(instance(page, 'two')).toContainText('已连接')
  expect(errors).toEqual([])
})
for (const [name, device] of [
  ['no IP', { ip: '', accessCode: 'code' }],
  ['no access code', { ip: '192.168.1.100', accessCode: '' }],
  [
    'already connected',
    { ip: '192.168.1.100', accessCode: 'code', state: { connected: true, phase: 'idle' } },
  ],
  [
    'already connecting',
    { ip: '192.168.1.100', accessCode: 'code', state: { phase: 'connecting' } },
  ],
])
  test(`startup skips ${name}`, async ({ page }) => {
    const errors = await openPrinters(page, { devices: [{ id: 'one', name: 'A1 1', ...device }] })
    await expect(page.locator('.paper-input')).toBeEnabled()
    expect(await page.evaluate(() => window.__connections.calls)).toEqual([])
    expect(errors).toEqual([])
  })
test('connection failures stay in the instance dialog and retry connects the same instance', async ({
  page,
}) => {
  const errors = await openPrinters(page)
  await expect.poll(() => page.evaluate(() => window.__connections.calls.length)).toBe(1)
  await page.evaluate(() => window.__connections.fail('one'))
  await page.getByRole('tab', { name: '设备台' }).click()
  await expect(instance(page, 'one')).toContainText('连接失败')
  await expect(page.locator('.notice[role="alert"]')).toHaveCount(0)
  await expect(page.locator('.job-bar .status-message')).toHaveCount(0)
  const editor = await editInstance(page, 'one')
  await expect(editor.getByRole('alert')).toContainText('自动检测失败')
  await editor.getByRole('button', { name: '连接打印机' }).click()
  await expect.poll(() => page.evaluate(() => window.__connections.calls.length)).toBe(2)
  await page.evaluate(() => window.__connections.fail('one'))
  await expect(editor.getByRole('button', { name: '连接打印机' })).toBeEnabled()
  await expect(editor.getByRole('alert')).toContainText('自动检测失败')
  await expect(page.locator('.notice[role="alert"]')).toHaveCount(0)
  await expect(page.locator('.job-bar .status-message')).toHaveCount(0)
  await editor.getByRole('button', { name: '连接打印机' }).click()
  await expect.poll(() => page.evaluate(() => window.__connections.calls.length)).toBe(3)
  await page.evaluate(() => window.__connections.succeed('one'))
  await expect(editor.getByRole('button', { name: '断开连接' })).toBeEnabled()
  expect(errors).toEqual([])
})
test('plus and minus manage persistent names IPs and visible access codes', async ({
  page,
  browserName,
}) => {
  const errors = await openPrinters(page, { devices: [], persist: true })
  await page.getByRole('tab', { name: '设备台' }).click()
  await page.getByRole('button', { name: '新增实例' }).click()
  const card = instance(page, 'new-1')
  const editor = dialog(page)
  await expect(editor).toBeVisible()
  await expect(editor.getByLabel('名称', { exact: true })).toBeFocused()
  await editor.getByLabel('名称', { exact: true }).fill('实验台 A1')
  await editor.getByLabel('IP 地址', { exact: true }).fill('192.168.1.102')
  await editor.getByLabel('访问码', { exact: true }).fill('12345678')
  await expect(editor.locator('input[type="password"]')).toHaveCount(0)
  await expect(editor.getByLabel('访问码', { exact: true })).toHaveAttribute('type', 'text')
  await expect(editor).not.toContainText('序列号')
  await editor.getByLabel('访问码', { exact: true }).press('Escape')
  await expect(editor).not.toBeVisible()
  await expect
    .poll(() => page.evaluate(() => window.__connections.list().devices[0].accessCode))
    .toBe('12345678')
  await expect(page.locator('.connection-only input:not([type="radio"])')).toHaveCount(0)
  await expect(card).toContainText('实验台 A1')
  await expect(card).not.toContainText('12345678')
  await page.reload()
  await page.getByRole('tab', { name: '设备台' }).click()
  await page.evaluate(() => window.__connections.succeed('new-1'))
  await editInstance(page, 'new-1')
  await expect(editor.getByLabel('名称', { exact: true })).toHaveValue('实验台 A1')
  await expect(editor.getByLabel('访问码', { exact: true })).toHaveValue('12345678')
  await editor.screenshot({ path: `/tmp/bambuscribe-instance-dialog-${browserName}.png` })
  await editor.getByRole('button', { name: '关闭实例编辑' }).click()
  await expect(card.getByRole('button', { name: '编辑 实验台 A1' })).toBeFocused()
  await page.getByRole('button', { name: '新增实例' }).click()
  await expect(page.locator('.printer-instance')).toHaveCount(2)
  await editor.getByRole('button', { name: '关闭实例编辑' }).click()
  await page
    .locator('.connection-only')
    .screenshot({ path: `/tmp/bambuscribe-instance-list-${browserName}.png` })
  await card.getByRole('button', { name: '删除 实验台 A1' }).click()
  await expect(page.locator('.printer-instance')).toHaveCount(1)
  await page.getByRole('button', { name: /删除 A1/ }).click()
  await expect(page.locator('.printer-instance')).toHaveCount(0)
  await page.reload()
  await page.getByRole('tab', { name: '设备台' }).click()
  await expect(page.locator('.printer-instance')).toHaveCount(0)
  expect(errors).toEqual([])
})
test('rename preserves connections and editing IP disconnects only that instance', async ({
  page,
}) => {
  const devices = [
    {
      id: 'one',
      name: 'A1 1',
      ip: '192.168.1.100',
      accessCode: 'one-code',
      state: { connected: true, phase: 'idle' },
    },
    {
      id: 'two',
      name: 'A1 2',
      ip: '192.168.1.101',
      accessCode: 'two-code',
      state: { connected: true, phase: 'idle' },
    },
  ]
  await openPrinters(page, { devices })
  await page.getByRole('tab', { name: '设备台' }).click()
  const editor = await editInstance(page, 'one')
  await editor.getByLabel('名称', { exact: true }).fill('左边')
  await editor.getByLabel('名称', { exact: true }).press('Tab')
  await expect
    .poll(() => page.evaluate(() => window.__connections.list().devices[0].name))
    .toBe('左边')
  expect(await page.evaluate(() => window.__connections.disconnects)).toEqual([])
  await editor.getByLabel('IP 地址', { exact: true }).fill('192.168.1.103')
  await editor.getByLabel('IP 地址', { exact: true }).press('Tab')
  await expect(editor.getByRole('button', { name: '连接打印机' })).toBeEnabled()
  await expect(instance(page, 'two')).toContainText('已连接')
  expect(await page.evaluate(() => window.__connections.disconnects)).toEqual(['one'])
  await editor.getByRole('button', { name: '关闭实例编辑' }).click()
  await editInstance(page, 'two')
  await expect(editor.getByLabel('名称', { exact: true })).toHaveValue('A1 2')
  await expect(editor.getByLabel('访问码', { exact: true })).toHaveValue('two-code')
})
test('upload picker routes to its selected connected instance', async ({ page }) => {
  await openPrinters(page, {
    selectionDelay: 100,
    devices: [
      {
        id: 'one',
        name: '左 A1',
        ip: '192.168.1.100',
        accessCode: 'one',
        state: { connected: true, phase: 'idle' },
      },
      {
        id: 'two',
        name: '右 A1',
        ip: '192.168.1.101',
        accessCode: 'two',
        state: { connected: true, phase: 'idle' },
      },
    ],
  })
  await page.locator('.upload-button').click()
  await page.getByLabel('上传打印机').selectOption('two')
  await expect(page.getByRole('button', { name: '上传全部', exact: true })).toBeDisabled()
  await expect(page.getByLabel('上传打印机')).toHaveValue('two')
  await page.getByRole('button', { name: '上传全部', exact: true }).click()
  await expect(page.getByRole('button', { name: '完成', exact: true })).toBeVisible()
  expect(await page.evaluate(() => window.__connections.uploads.at(-1).deviceId)).toBe('two')
  expect(await page.evaluate(() => window.__connections.list().devices[0].state.phase)).toBe('idle')
  await page.getByRole('button', { name: '完成', exact: true }).click()
  await expect(page.locator('.connection-status')).toContainText('右 A1')
})
