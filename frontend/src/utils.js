export function formatBytes(n) {
  if (!n || n <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1)
  const v = n / 1024 ** i
  return v.toFixed(v >= 100 || i === 0 ? 0 : 1) + ' ' + units[i]
}

export function formatDate(s) {
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function buildSrcset(img) {
  const sized = (img.variants || [])
    .filter((v) => v.width > 0 && v.label !== 'main')
    .sort((a, b) => a.width - b.width)
  if (sized.length === 0) return img.url
  return sized.map((v) => `${v.url} ${v.width}w`).join(', ')
}

export function markdown(img) {
  return `![${img.name}](${img.url})`
}

export async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
}
