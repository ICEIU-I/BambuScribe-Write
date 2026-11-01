const digits: Record<string, number> = {
  零: 0,
  〇: 0,
  一: 1,
  二: 2,
  两: 2,
  三: 3,
  四: 4,
  五: 5,
  六: 6,
  七: 7,
  八: 8,
  九: 9,
}
const units: Record<string, number> = { 十: 10, 百: 100, 千: 1000 }

function chineseNumber(text: string): string {
  if (!/[十百千]/u.test(text)) return Array.from(text, (c) => digits[c]).join('')
  let result = 0,
    digit = 0
  for (const char of text) {
    if (char in units) {
      result += (digit || 1) * units[char]!
      digit = 0
    } else digit = digits[char]!
  }
  return String(result + digit)
}

export function defaultUploadPrefix(text: string, fallback = 'Untitled'): string {
  const line =
    text
      .normalize('NFKC')
      .split(/\r?\n/u)
      .find((line) => line.trim())
      ?.trim() || ''
  const arabic = /\d+/u.exec(line)
  const chinese = /[零〇一二三四五六七八九十百千两]+/u.exec(line)
  if (arabic && (!chinese || arabic.index < chinese.index)) return arabic[0].slice(0, 64)
  if (chinese) return chineseNumber(chinese[0])
  return (
    Array.from(line.replace(/[\/\\:*?"<>|\u0000-\u001f\u007f-\u009f]/gu, '').trim())
      .slice(0, 48)
      .join('') || fallback
  )
}

export function validUploadPrefix(prefix: string): boolean {
  const name = prefix.trim()
  return (
    !!name &&
    Array.from(name).length <= 64 &&
    !/[\/\\:*?"<>|\u0000-\u001f\u007f-\u009f]/u.test(name)
  )
}

export function uploadPageSuffix(numbers: number[]): string {
  const first = numbers[0] || 1,
    last = numbers.at(-1) || first
  return first === last ? `-page${first}` : `-page${first}~page${last}`
}
