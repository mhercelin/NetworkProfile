import { cidr } from './format.js'

// Column widths in pixels. The name column takes what is left, so it has no
// entry here.
export const DEFAULT_COLUMNS = {
  address: 140,
  gateway: 120,
  dns: 124,
  mode: 72,
  actions: 156,
}

export const MIN_COLUMN_WIDTH = 60

// rowData pulls out what a row shows, once, so the table and the sort cannot
// disagree about what is in a column.
//
// A profile covering several adapters has no single address to show: the
// columns stay empty and the adapters are named under the profile name.
export function rowData(profile) {
  const targets = profile.targets ?? []
  const single = targets.length === 1 ? targets[0] : null
  const dnsList = single ? (single.dns ?? []) : []

  return {
    name: profile.name ?? '',
    interfaces: targets.map((t) => t.interface),
    address: single && single.mode === 'static' ? cidr(single.address, single.mask) : '',
    gateway: single ? (single.gateway ?? '') : '',
    dns: dnsList.length > 1 ? `${dnsList[0]} +${dnsList.length - 1}` : (dnsList[0] ?? ''),
    mode: targets.length > 0 && targets.every((t) => t.mode === 'dhcp') ? 'DHCP' : 'STATIQUE',
  }
}

// compareAddresses orders IPv4 addresses by value, not as text. Sorting them as
// text puts 10.x after 9.x and 192.168.1.100 before 192.168.1.9, which is
// exactly the comparison an engineer reading this table is making.
export function compareAddresses(left, right) {
  const a = parseAddress(left)
  const b = parseAddress(right)

  // Rows with nothing in the column collect at the end, whichever way the sort
  // runs: they carry no order of their own.
  if (a === null && b === null) return 0
  if (a === null) return 1
  if (b === null) return -1

  return a - b
}

function parseAddress(value) {
  const match = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})/.exec(String(value ?? '').trim())
  if (!match) return null

  let packed = 0
  for (let i = 1; i <= 4; i++) {
    const octet = Number(match[i])
    if (octet > 255) return null
    packed = packed * 256 + octet
  }
  return packed
}

const ADDRESS_COLUMNS = new Set(['address', 'gateway', 'dns'])

// sortProfiles returns a new list; the stored order is never touched.
export function sortProfiles(profiles, sort) {
  if (!sort?.key) return profiles.slice()

  const direction = sort.dir === 'desc' ? -1 : 1

  return profiles
    .map((profile, index) => ({ profile, index, data: rowData(profile) }))
    .sort((a, b) => {
      // A row with nothing in the column has no place in the order, so it sinks
      // to the bottom either way. Reversing the sort must not float the empty
      // rows to the top and bury the ones worth reading.
      const blankA = isBlank(a.data[sort.key])
      const blankB = isBlank(b.data[sort.key])
      if (blankA !== blankB) return blankA ? 1 : -1
      if (blankA) return a.index - b.index

      const compared = compareColumn(sort.key, a.data, b.data)
      // Equal values keep the order they were stored in, so a sort never
      // reshuffles rows it has nothing to say about.
      return compared !== 0 ? compared * direction : a.index - b.index
    })
    .map((entry) => entry.profile)
}

function isBlank(value) {
  if (Array.isArray(value)) return value.length === 0
  return String(value ?? '').trim() === ''
}

function compareColumn(key, a, b) {
  if (ADDRESS_COLUMNS.has(key)) {
    return compareAddresses(a[key], b[key])
  }
  if (key === 'interfaces') {
    return a.interfaces.join(' ').localeCompare(b.interfaces.join(' '), 'fr')
  }
  return String(a[key]).localeCompare(String(b[key]), 'fr', { numeric: true })
}

// filterProfiles narrows by free text and by adapter. The text matches the
// profile name and the adapters it targets, because "Ethernet" is as natural a
// thing to type as part of a name.
export function filterProfiles(profiles, { query = '', iface = '' } = {}) {
  const needle = query.trim().toLowerCase()

  return profiles.filter((profile) => {
    const targets = profile.targets ?? []

    if (iface && !targets.some((t) => t.interface === iface)) {
      return false
    }
    if (!needle) return true

    const haystack = [profile.name ?? '', ...targets.map((t) => t.interface), ...targets.map((t) => t.ssid ?? '')]
      .join(' ')
      .toLowerCase()
    return haystack.includes(needle)
  })
}

// interfacesInUse lists the adapters the stored profiles target, for the filter
// to offer. Adapters absent from every profile would only be dead entries.
export function interfacesInUse(profiles) {
  const seen = new Set()
  for (const profile of profiles) {
    for (const target of profile.targets ?? []) {
      if (target.interface) seen.add(target.interface)
    }
  }
  return [...seen].sort((a, b) => a.localeCompare(b, 'fr', { numeric: true }))
}
