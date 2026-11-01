import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

async function setup(page, available = true) {
  const errors = await open(page)
  await page.evaluate((available) => {
    const api = window.go.application.App
    const save = api.SavePreferences
    window.__update = { order: [], failSave: false }
    api.CheckUpdate = async () => ({
      current: '0.1.1',
      version: available ? '0.1.2' : '0.1.1',
      available,
      notes: '<script>never execute</script>',
      url: 'https://github.com/ICEIU-I/BambuScribe-Write-Releases/releases',
    })
    api.SavePreferences = async (p, path) => {
      window.__update.order.push('save')
      if (window.__update.failSave) throw new Error('项目保存失败，请重试。')
      return save(p, path)
    }
    api.InstallUpdate = async () => {
      window.__update.order.push('download')
      window.__tables.menu['update:progress']({ phase: 'downloading', percent: 42 })
      await new Promise((resolve, reject) => {
        window.__update.finish = resolve
        window.__update.reject = reject
      })
    }
    api.RestartUpdate = async () => {
      window.__update.order.push('restart')
      window.__tables.menu['update:progress']({ phase: 'restarting', percent: 100 })
    }
    api.CancelUpdate = async () => window.__update.reject?.(new Error('更新已取消，原版本保留。'))
  }, available)
  return errors
}

test('sidebar and Help show the current version and allow another check', async ({ page }) => {
  const errors = await setup(page, false)
  await page.getByRole('button', { name: '检查更新', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '软件更新' })
  await expect(dialog).toContainText('当前版本 0.1.1')
  await expect(dialog).toContainText('已是最新版本')
  await expect(dialog.getByRole('button', { name: '一键更新并重启' })).toHaveCount(0)
  await page.getByRole('button', { name: '关闭更新窗口' }).click()
  await page.evaluate(() => window.__tables.menu['menu:action']('checkUpdate'))
  await expect(dialog).toBeVisible()
  expect(errors).toEqual([])
})

test('save precedes download, progress locks editing, and a final save precedes restart', async ({
  page,
}, info) => {
  const errors = await setup(page)
  const input = page.locator('textarea.paper-input')
  await input.fill('更新前需要保存的项目')
  await page.getByRole('button', { name: '检查更新', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '软件更新' })
  await expect(dialog).toContainText('新版本 0.1.2')
  await dialog.locator('summary').click()
  await expect(dialog).toContainText('<script>never execute</script>')
  await dialog.getByRole('button', { name: '一键更新并重启' }).click()
  await expect(dialog.getByRole('progressbar')).toHaveAttribute('value', '42')
  await expect(dialog.getByRole('button', { name: '一键更新并重启' })).toBeDisabled()
  await expect(page.getByRole('button', { name: '关闭更新窗口' })).toBeDisabled()
  await page.evaluate(() => window.__tables.menu['menu:action']('new'))
  await expect(input).toHaveValue('更新前需要保存的项目')
  await page.screenshot({ path: info.outputPath('update-progress.png') })
  const before = await page.evaluate(() => window.__update.order)
  expect(before.indexOf('save')).toBeLessThan(before.indexOf('download'))
  await page.evaluate(() => {
    document.querySelector('.theme-toggle').click()
    window.__update.finish()
  })
  await expect(dialog).toContainText('正在重启')
  const after = await page.evaluate(() => window.__update.order)
  expect(after.lastIndexOf('save')).toBeGreaterThan(after.indexOf('download'))
  expect(after.lastIndexOf('save')).toBeLessThan(after.indexOf('restart'))
  expect(errors).toEqual([])
})

test('failed save prevents download and restart', async ({ page }) => {
  const errors = await setup(page)
  await page.locator('textarea.paper-input').fill('不能丢失这段文字')
  await page.evaluate(() => {
    window.__update.failSave = true
  })
  await page.getByRole('button', { name: '检查更新', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '软件更新' })
  await dialog.getByRole('button', { name: '一键更新并重启' }).click()
  await expect(dialog.getByRole('alert')).toContainText('保存失败')
  expect(await page.evaluate(() => window.__update.order)).not.toContain('download')
  expect(await page.evaluate(() => window.__update.order)).not.toContain('restart')
  await expect(page.locator('textarea.paper-input')).toHaveValue('不能丢失这段文字')
  expect(errors).toEqual([])
})

test('cancelled download keeps the app open and allows retry', async ({ page }) => {
  const errors = await setup(page)
  await page.getByRole('button', { name: '检查更新', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '软件更新' })
  await dialog.getByRole('button', { name: '一键更新并重启' }).click()
  await dialog.getByRole('button', { name: '取消下载' }).click()
  await expect(dialog.getByRole('alert')).toContainText('更新已取消')
  await expect(dialog.getByRole('button', { name: '一键更新并重启' })).toBeEnabled()
  expect(await page.evaluate(() => window.__update.order)).not.toContain('restart')
  expect(errors).toEqual([])
})

test('network errors stay visible and allow another check', async ({ page }) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    window.go.application.App.CheckUpdate = async () => {
      throw new Error('无法连接更新服务，请检查网络后重试。')
    }
  })
  await page.getByRole('button', { name: '检查更新', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '软件更新' })
  await expect(dialog.getByRole('alert')).toContainText('检查网络后重试')
  await expect(dialog.getByRole('button', { name: '检查更新', exact: true })).toBeEnabled()
  expect(errors).toEqual([])
})

test('a failed final save retains the prepared app and never restarts', async ({ page }) => {
  const errors = await setup(page)
  await page.getByRole('button', { name: '检查更新', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '软件更新' })
  await dialog.getByRole('button', { name: '一键更新并重启' }).click()
  await page.evaluate(() => {
    document.querySelector('.theme-toggle').click()
    window.__update.failSave = true
    window.__update.finish()
  })
  await expect(dialog.getByRole('alert')).toContainText('保存失败')
  expect(await page.evaluate(() => window.__update.order)).not.toContain('restart')
  await expect(page.getByRole('button', { name: '关闭更新窗口' })).toBeEnabled()
  expect(errors).toEqual([])
})

test('cancellation racing with download completion still prevents restart', async ({ page }) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    window.go.application.App.CancelUpdate = async () => window.__update.finish?.()
  })
  await page.getByRole('button', { name: '检查更新', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '软件更新' })
  await dialog.getByRole('button', { name: '一键更新并重启' }).click()
  await dialog.getByRole('button', { name: '取消下载' }).click()
  await expect(dialog.getByRole('alert')).toContainText('更新已取消')
  expect(await page.evaluate(() => window.__update.order)).not.toContain('restart')
  expect(errors).toEqual([])
})
