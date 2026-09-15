const ESCAPES = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }

// Profile names and SSIDs come from the user and from the surrounding networks,
// and both end up inside innerHTML.
export function esc(value) {
  return String(value ?? '').replace(/[&<>"']/g, (c) => ESCAPES[c])
}

export function maskToPrefix(mask) {
  const parts = String(mask ?? '').split('.')
  if (parts.length !== 4) return null

  let bits = 0
  for (const part of parts) {
    const octet = Number(part)
    if (!/^\d{1,3}$/.test(part) || octet > 255) return null
    bits += octet.toString(2).replace(/0/g, '').length
  }
  return bits
}

export function prefixToMask(prefix) {
  const bits = Number(prefix)
  if (!Number.isInteger(bits) || bits < 1 || bits > 32) return ''

  const value = bits === 32 ? 0xffffffff : (0xffffffff << (32 - bits)) >>> 0
  return [24, 16, 8, 0].map((shift) => (value >>> shift) & 0xff).join('.')
}

// "10.10.128.20/24" — the form engineers actually read.
export function cidr(address, mask) {
  if (!address) return ''
  const prefix = maskToPrefix(mask)
  return prefix === null ? address : `${address}/${prefix}`
}

export function slugify(name) {
  const slug = String(name ?? '')
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return slug || 'profil'
}

// uniqueSlug keeps ids stable and collision-free without showing the user a
// field they would have to understand.
export function uniqueSlug(name, taken) {
  const base = slugify(name)
  if (!taken.includes(base)) return base

  for (let n = 2; ; n++) {
    const candidate = `${base}-${n}`
    if (!taken.includes(candidate)) return candidate
  }
}

export function clockTime(date = new Date()) {
  return date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
}
