import { expect } from '@playwright/test'
export async function openPrinters(page, options = {}) {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))
  await page.addInitScript((options) => {
    const baseState = {
      connected: false,
      phase: 'disconnected',
      message: '',
      progress: 0,
      jobId: '',
      documentId: '',
      page: 0,
      deviceState: '',
      totalPages: 0,
      pageOrdinal: 0,
      uploadedPages: 0,
    }
    let devices = (
      options.devices || [
        { id: 'one', name: 'A1 1', ip: '192.168.1.100', accessCode: 'visible-code' },
      ]
    ).map((d) => ({ ...d, state: { ...baseState, ...d.state, deviceId: d.id } }))
    let selectedDeviceId = devices[0]?.id || ''
    if (options.persist && localStorage.getItem('test-printers')) {
      const saved = JSON.parse(localStorage.getItem('test-printers'))
      devices = saved.devices.map((d) => ({ ...d, state: { ...baseState, deviceId: d.id } }))
      selectedDeviceId = saved.selectedDeviceId
    }
    const handlers = {},
      pending = {}
    const copy = (value) => JSON.parse(JSON.stringify(value))
    const list = () => copy({ devices, selectedDeviceId })
    const save = () => {
      if (options.persist) localStorage.setItem('test-printers', JSON.stringify(list()))
    }
    const find = (id) => {
      const d = devices.find((d) => d.id === id)
      if (!d) throw new Error('实例已删除')
      return d
    }
    const emit = (id, state) => {
      const d = find(id)
      d.state = { ...d.state, ...state, deviceId: id }
      for (const h of handlers['printer:state'] || []) h(copy(d.state))
    }
    window.__connections = {
      calls: [],
      uploads: [],
      disconnects: [],
      deletes: [],
      cancels: [],
      emit,
      list,
      succeed: (id) => {
        emit(id, { connected: true, phase: 'idle', deviceState: 'IDLE', message: '' })
        pending[id].resolve()
      },
      fail: (id) => {
        emit(id, {
          connected: false,
          phase: 'disconnected',
          message: '自动检测失败，请检查 IP 后重试',
        })
        pending[id].reject(new Error('自动检测失败，请检查 IP 后重试'))
      },
    }
    window.runtime = {
      EventsOn: (name, h) => {
        ;(handlers[name] ||= []).push(h)
        return () => {}
      },
      WindowSetTitle: () => {},
    }
    const layout = {
      templateName: '测试',
      lineMode: 'fixed',
      lineCount: 16,
      paperWidth: 163,
      paperHeight: 240,
      marginTop: 0,
      marginRight: 0,
      marginBottom: 0,
      marginLeft: 0,
      fontSize: 6,
      letterGap: 0,
      lineHeight: 15,
      paragraphGap: 0,
      indent: 0,
      align: 'left',
    }
    const preferences = {
      theme: 'light',
      language: 'zh',
      text: '实例连接测试',
      layout,
      writing: { offsetX: 0, offsetY: 0, penUp: 8, penDown: 3.4, originMode: 'auto', sound: true },
      images: [],
      tables: [],
      erasures: [],
    }
    let next = devices.length + 1
    const none = async () => {}
    window.go = {
      application: {
        App: {
          Bootstrap: async () => ({
            ...list(),
            preferences,
            printer: { ip: '', serial: '', hasAccessCode: false },
            state: findState(),
            warning: '',
          }),
          Printers: async () => list(),
          PrinterState: async () => findState(),
          SavePrinterDevice: async (d) => {
            if (!d.id) {
              d = {
                ...d,
                id: 'new-' + next,
                name: d.name || 'A1 ' + next,
                state: { ...baseState, deviceId: 'new-' + next },
              }
              next++
              devices.push(d)
            } else {
              const old = find(d.id)
              if (!d.name.trim()) throw new Error('请输入实例名称')
              const changed = old.ip !== d.ip || old.accessCode !== d.accessCode
              Object.assign(old, { name: d.name, ip: d.ip, accessCode: d.accessCode })
              if (changed && old.state.connected) {
                window.__connections.disconnects.push(d.id)
                emit(d.id, { connected: false, phase: 'disconnected' })
              }
            }
            if (!selectedDeviceId) selectedDeviceId = d.id
            save()
            return list()
          },
          DeletePrinterDevice: async (id) => {
            window.__connections.deletes.push(id)
            devices = devices.filter((d) => d.id !== id)
            if (selectedDeviceId === id) selectedDeviceId = devices[0]?.id || ''
            save()
            return list()
          },
          SelectPrinter: async (id) => {
            if (options.selectionDelay)
              await new Promise((resolve) => setTimeout(resolve, options.selectionDelay))
            find(id)
            selectedDeviceId = id
            save()
            return list()
          },
          ConnectDevice: async (id) => {
            const d = find(id)
            window.__connections.calls.push({ id, ip: d.ip, accessCode: d.accessCode })
            emit(id, { phase: 'connecting', message: '' })
            await new Promise((resolve, reject) => (pending[id] = { resolve, reject }))
          },
          DisconnectDevice: async (id) => {
            window.__connections.disconnects.push(id)
            emit(id, { connected: false, phase: 'disconnected', message: '' })
          },
          UploadDevicePages: async (deviceId, id, prefix, indices, writing) => {
            window.__connections.uploads.push({ deviceId, id, prefix, indices, writing })
            emit(deviceId, {
              phase: 'uploaded',
              totalPages: 1,
              uploadedPages: 1,
              page: 1,
              pageOrdinal: 1,
              progress: 100,
            })
            return 'upload-id'
          },
          CancelDeviceUpload: async (id) => {
            window.__connections.cancels.push(id)
            emit(id, { phase: 'idle' })
          },
          SavePreferences: async () => {},
          SetLanguage: none,
          SetMenuState: none,
          PreviewWithImages: async (text) => ({
            id: 'doc',
            settings: layout,
            missing: [],
            characterCount: text.length,
            fontName: '测试',
            pages: [
              {
                number: 1,
                lineHeight: 15,
                lineCount: 16,
                text,
                strokes: [
                  [
                    { x: 10, y: 10 },
                    { x: 12, y: 12 },
                  ],
                ],
                textStrokes: [],
                images: [],
                cells: [],
                carets: [],
                missing: [],
              },
            ],
          }),
        },
      },
    }
    function findState() {
      return devices.find((d) => d.id === selectedDeviceId)?.state || baseState
    }
  }, options)
  await page.goto('/')
  await expect(page.locator('.paper-input')).toHaveValue('实例连接测试')
  return errors
}
export const instance = (page, id) => page.locator(`.printer-instance[data-device-id="${id}"]`)
