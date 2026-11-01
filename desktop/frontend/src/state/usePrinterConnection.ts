import { computed, reactive, ref, type Ref } from 'vue'
import { errorText, native, snapshot } from '../services/bridge'
import { defaultPrinter } from './defaults'
import type { Bootstrap, PrinterDevice, PrinterList, PrinterSettings, PrinterState } from '../types'

export function usePrinterConnection(
  settings: Ref<PrinterSettings>,
  state: Ref<PrinterState>,
  tell: (text: string, kind?: 'error' | 'success') => void,
) {
  const devices = ref<PrinterDevice[]>([])
  const selectedDeviceId = ref('')
  const selectingDevice = ref(false)
  const deviceActions = reactive<Record<string, 'connect' | 'save' | 'delete' | undefined>>({})
  const deviceErrors = reactive<Record<string, string>>({})
  const selectedDevice = computed(() => devices.value.find((d) => d.id === selectedDeviceId.value))
  const saves = new Map<string, Promise<void>>()
  function apply(list: PrinterList) {
    devices.value = list.devices || []
    selectedDeviceId.value = list.selectedDeviceId
    state.value = selectedDevice.value?.state || defaultPrinter()
  }
  function loadPrinters(boot: Bootstrap) {
    settings.value = boot.printer
    state.value = boot.state
    if (boot.devices) apply({ devices: boot.devices, selectedDeviceId: boot.selectedDeviceId })
  }
  function observePrinter(next: PrinterState) {
    if (!next.deviceId) {
      state.value = next
      return
    }
    const device = devices.value.find((d) => d.id === next.deviceId)
    if (!device) return
    device.state = next
    if (device.id === selectedDeviceId.value) state.value = next
  }
  async function refreshPrinters() {
    apply(await native().Printers())
  }
  async function saveDevice(device: PrinterDevice) {
    const previous = saves.get(device.id) || Promise.resolve()
    const work = previous
      .catch(() => {})
      .then(async () => {
        apply(await native().SavePrinterDevice(snapshot(device)))
        delete deviceErrors[device.id]
      })
    saves.set(device.id, work)
    try {
      await work
    } catch (error) {
      deviceErrors[device.id] = errorText(error)
      throw error
    } finally {
      if (saves.get(device.id) === work) saves.delete(device.id)
    }
  }
  async function editDevice(device: PrinterDevice) {
    try {
      await saveDevice(device)
    } catch (error) {
      tell(errorText(error))
    }
  }
  async function addDevice() {
    try {
      const list = await native().SavePrinterDevice({ id: '', name: '', ip: '', accessCode: '' })
      apply(list)
      return list.devices[list.devices.length - 1]?.id
    } catch (error) {
      tell(errorText(error))
    }
  }
  async function deleteDevice(id: string) {
    if (deviceActions[id]) return
    deviceActions[id] = 'delete'
    try {
      await saves.get(id)
      apply(await native().DeletePrinterDevice(id))
      delete deviceErrors[id]
    } catch (error) {
      deviceErrors[id] = errorText(error)
      tell(errorText(error))
    } finally {
      delete deviceActions[id]
    }
  }
  async function selectDevice(id: string) {
    if (selectingDevice.value) return
    selectingDevice.value = true
    try {
      apply(await native().SelectPrinter(id))
    } catch (error) {
      tell(errorText(error))
    } finally {
      selectingDevice.value = false
    }
  }
  async function connectDevice(device: PrinterDevice, announce = true) {
    if (deviceActions[device.id] || device.state.connected || device.state.phase === 'connecting')
      return
    deviceActions[device.id] = 'connect'
    delete deviceErrors[device.id]
    try {
      await saves.get(device.id)
      await saveDevice(device)
      await native().ConnectDevice(device.id)
      await refreshPrinters()
      if (announce) tell(`${device.name} 已连接。`, 'success')
    } catch (error) {
      deviceErrors[device.id] = errorText(error)
    } finally {
      delete deviceActions[device.id]
    }
  }
  async function disconnectDevice(id: string) {
    try {
      await native().DisconnectDevice(id)
      await refreshPrinters()
    } catch (error) {
      deviceErrors[id] = errorText(error)
      tell(errorText(error))
    }
  }
  async function autoConnectSaved() {
    await Promise.allSettled(
      devices.value
        .filter(
          (d) =>
            d.ip.trim() && d.accessCode && !d.state.connected && d.state.phase !== 'connecting',
        )
        .map((d) => connectDevice(d, false)),
    )
  }
  return {
    devices,
    selectingDevice,
    selectedDeviceId,
    selectedDevice,
    deviceActions,
    deviceErrors,
    loadPrinters,
    observePrinter,
    refreshPrinters,
    editDevice,
    addDevice,
    deleteDevice,
    selectDevice,
    connectDevice,
    disconnectDevice,
    autoConnectSaved,
  }
}
