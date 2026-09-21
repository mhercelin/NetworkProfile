import { describe, expect, it } from 'vitest'
import {
  clampColumn,
  compareAddresses,
  DEFAULT_COLUMNS,
  filterProfiles,
  interfacesInUse,
  MIN_COLUMN_WIDTHS,
  rowData,
  sortProfiles,
} from './table.js'

function staticProfile(id, name, iface, address, extra = {}) {
  return {
    id,
    name,
    targets: [{ interface: iface, mode: 'static', address, mask: '255.255.255.0', ...extra }],
  }
}

describe('rowData', () => {
  it('reads the columns of a single-adapter profile', () => {
    const p = staticProfile('a', 'Atelier', 'Ethernet', '192.168.1.50', {
      gateway: '192.168.1.1',
      dns: ['192.168.1.1', '9.9.9.9'],
    })

    expect(rowData(p)).toEqual({
      name: 'Atelier',
      interfaces: ['Ethernet'],
      address: '192.168.1.50/24',
      gateway: '192.168.1.1',
      dns: '192.168.1.1 +1',
      mode: 'STATIQUE',
    })
  })

  // A profile covering two adapters has no single address to show.
  it('leaves the address columns empty for a multi-adapter profile', () => {
    const p = {
      id: 'defaut',
      name: 'Défaut',
      targets: [
        { interface: 'Ethernet', mode: 'dhcp' },
        { interface: 'Wi-Fi', mode: 'dhcp' },
      ],
    }

    expect(rowData(p)).toMatchObject({
      interfaces: ['Ethernet', 'Wi-Fi'],
      address: '',
      gateway: '',
      dns: '',
      mode: 'DHCP',
    })
  })

  it('shows no address for a DHCP profile', () => {
    const p = { id: 'd', name: 'D', targets: [{ interface: 'Wi-Fi', mode: 'dhcp' }] }
    expect(rowData(p).address).toBe('')
  })
})

describe('compareAddresses', () => {
  // Sorting addresses as text is the whole reason this function exists.
  it('orders by value, not alphabetically', () => {
    expect(compareAddresses('9.0.0.1', '10.0.0.1')).toBeLessThan(0)
    expect(compareAddresses('192.168.1.100', '192.168.1.9')).toBeGreaterThan(0)
    expect(compareAddresses('10.0.0.1', '10.0.0.1')).toBe(0)
  })

  it('ignores the prefix length', () => {
    expect(compareAddresses('10.0.0.5/24', '10.0.0.6/16')).toBeLessThan(0)
  })

  it('puts empty and unparsable values last whichever way it runs', () => {
    expect(compareAddresses('', '10.0.0.1')).toBeGreaterThan(0)
    expect(compareAddresses('10.0.0.1', '')).toBeLessThan(0)
    expect(compareAddresses('—', '10.0.0.1')).toBeGreaterThan(0)
    expect(compareAddresses('', '')).toBe(0)
  })

  it('refuses an octet out of range', () => {
    expect(compareAddresses('10.0.0.300', '10.0.0.1')).toBeGreaterThan(0)
  })
})

describe('sortProfiles', () => {
  const profiles = [
    staticProfile('c', 'Chantier', 'Wi-Fi', '172.16.32.8'),
    staticProfile('a', 'Atelier', 'Ethernet', '192.168.1.50'),
    staticProfile('b', 'Bureau', 'Ethernet', '10.56.20.9'),
  ]

  it('leaves the stored order alone without a key', () => {
    expect(sortProfiles(profiles, {}).map((p) => p.id)).toEqual(['c', 'a', 'b'])
  })

  it('sorts by name', () => {
    expect(sortProfiles(profiles, { key: 'name', dir: 'asc' }).map((p) => p.id)).toEqual(['a', 'b', 'c'])
    expect(sortProfiles(profiles, { key: 'name', dir: 'desc' }).map((p) => p.id)).toEqual(['c', 'b', 'a'])
  })

  it('sorts addresses by value', () => {
    expect(sortProfiles(profiles, { key: 'address', dir: 'asc' }).map((p) => p.id)).toEqual(['b', 'c', 'a'])
  })

  // Reversing a sort must not float the rows with nothing in that column to the
  // top and bury the ones worth reading.
  it('keeps rows with an empty column at the bottom in both directions', () => {
    const mixed = [
      { id: 'dhcp', name: 'DHCP', targets: [{ interface: 'Wi-Fi', mode: 'dhcp' }] },
      staticProfile('high', 'Haut', 'Ethernet', '192.168.1.50'),
      staticProfile('low', 'Bas', 'Ethernet', '10.0.0.1'),
    ]

    expect(sortProfiles(mixed, { key: 'address', dir: 'asc' }).map((p) => p.id)).toEqual(['low', 'high', 'dhcp'])
    expect(sortProfiles(mixed, { key: 'address', dir: 'desc' }).map((p) => p.id)).toEqual(['high', 'low', 'dhcp'])
  })

  it('does not modify the list it was given', () => {
    const before = profiles.map((p) => p.id)
    sortProfiles(profiles, { key: 'name', dir: 'desc' })
    expect(profiles.map((p) => p.id)).toEqual(before)
  })

  // Rows a sort has nothing to say about must not get reshuffled.
  it('keeps the stored order between equal values', () => {
    const same = [
      { id: 'first', name: 'Même', targets: [{ interface: 'Ethernet', mode: 'dhcp' }] },
      { id: 'second', name: 'Même', targets: [{ interface: 'Wi-Fi', mode: 'dhcp' }] },
    ]
    expect(sortProfiles(same, { key: 'name', dir: 'asc' }).map((p) => p.id)).toEqual(['first', 'second'])
  })
})

describe('filterProfiles', () => {
  const profiles = [
    staticProfile('a', 'Atelier', 'Ethernet', '192.168.1.50'),
    staticProfile('w', 'Chantier', 'Wi-Fi', '172.16.32.8', { ssid: 'CHANTIER-MOB' }),
    {
      id: 'defaut',
      name: 'Défaut',
      targets: [
        { interface: 'Ethernet', mode: 'dhcp' },
        { interface: 'Wi-Fi', mode: 'dhcp' },
      ],
    },
  ]

  it('returns everything without criteria', () => {
    expect(filterProfiles(profiles).map((p) => p.id)).toEqual(['a', 'w', 'defaut'])
  })

  it('keeps only the profiles touching one adapter', () => {
    expect(filterProfiles(profiles, { iface: 'Wi-Fi' }).map((p) => p.id)).toEqual(['w', 'defaut'])
  })

  it('matches the name regardless of case', () => {
    expect(filterProfiles(profiles, { query: 'atel' }).map((p) => p.id)).toEqual(['a'])
  })

  // Typing an adapter name is as natural as typing part of a profile name.
  it('matches the adapter and the wireless network too', () => {
    expect(filterProfiles(profiles, { query: 'wi-fi' }).map((p) => p.id)).toEqual(['w', 'defaut'])
    expect(filterProfiles(profiles, { query: 'chantier-mob' }).map((p) => p.id)).toEqual(['w'])
  })

  it('applies both criteria together', () => {
    expect(filterProfiles(profiles, { query: 'défaut', iface: 'Wi-Fi' }).map((p) => p.id)).toEqual(['defaut'])
    expect(filterProfiles(profiles, { query: 'atelier', iface: 'Wi-Fi' })).toEqual([])
  })
})

describe('interfacesInUse', () => {
  it('lists the adapters the profiles target, once each', () => {
    const profiles = [
      staticProfile('a', 'A', 'Ethernet', '10.0.0.1'),
      staticProfile('b', 'B', 'Ethernet', '10.0.0.2'),
      { id: 'c', name: 'C', targets: [{ interface: 'Wi-Fi', mode: 'dhcp' }] },
    ]
    expect(interfacesInUse(profiles)).toEqual(['Ethernet', 'Wi-Fi'])
  })

  it('orders Ethernet 2 after Ethernet', () => {
    const profiles = [
      { id: 'b', name: 'B', targets: [{ interface: 'Ethernet 2', mode: 'dhcp' }] },
      { id: 'a', name: 'A', targets: [{ interface: 'Ethernet', mode: 'dhcp' }] },
    ]
    expect(interfacesInUse(profiles)).toEqual(['Ethernet', 'Ethernet 2'])
  })

  it('is empty without profiles', () => {
    expect(interfacesInUse([])).toEqual([])
  })
})

describe('clampColumn', () => {
  it('keeps a width that is wide enough', () => {
    expect(clampColumn('name', 300)).toBe(300)
  })

  it('rounds to whole pixels', () => {
    expect(clampColumn('name', 300.6)).toBe(301)
  })

  // The actions column holds three tools, a marker and a button; narrower than
  // its minimum its contents spilled left over the column beside it.
  it('holds every column at its own minimum', () => {
    for (const [key, min] of Object.entries(MIN_COLUMN_WIDTHS)) {
      expect(clampColumn(key, 10)).toBe(min)
    }
    expect(MIN_COLUMN_WIDTHS.actions).toBeGreaterThan(MIN_COLUMN_WIDTHS.mode)
  })

  // A stored width can come from an older version, or from a hand-edited file.
  it('falls back to the default for anything unreadable', () => {
    expect(clampColumn('name', undefined)).toBe(DEFAULT_COLUMNS.name)
    expect(clampColumn('name', 'wide')).toBe(DEFAULT_COLUMNS.name)
    expect(clampColumn('name', Infinity)).toBe(DEFAULT_COLUMNS.name)
  })

  it('every default is at least its own minimum', () => {
    for (const [key, min] of Object.entries(MIN_COLUMN_WIDTHS)) {
      expect(DEFAULT_COLUMNS[key]).toBeGreaterThanOrEqual(min)
    }
  })
})
