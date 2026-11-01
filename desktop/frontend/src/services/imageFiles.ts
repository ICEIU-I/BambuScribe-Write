export async function readImageFile(file: File): Promise<string> {
  if (!['image/png', 'image/jpeg'].includes(file.type)) throw new Error('请选择 PNG 或 JPEG 图片')
  if (file.size > 8 * 1024 * 1024) throw new Error('图片超过 8 MB，请裁剪后重试')
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(new Error('图片读取失败，请重新选择'))
    reader.readAsDataURL(file)
  })
}

export function imageDimensions(data: string): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => {
      if (image.naturalWidth < 2 || image.naturalHeight < 2) {
        reject(new Error('图片分辨率过低，请重新截图'))
        return
      }
      resolve({ width: image.naturalWidth, height: image.naturalHeight })
    }
    image.onerror = () => reject(new Error('图片无法读取，请重新截图'))
    image.src = data
  })
}
