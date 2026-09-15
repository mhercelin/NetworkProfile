import { describe, expect, it } from 'vitest'
import { cidr, esc, maskToPrefix, prefixToMask, slugify, uniqueSlug } from './format.js'

describe('maskToPrefix', () => {
  it.each([
    ['255.255.255.0', 24],
    ['255.255.0.0', 16],
    ['255.0.0.0', 8],
    ['255.255.255.252', 30],
    ['255.255.255.255', 32],
    ['0.0.0.0', 0],
  ])('reads %s as /%i', (mask, expected) => {
    expect(maskToPrefix(mask)).toBe(expected)
  })

  it.each(['', '24', '255.255.255', '255.255.255.256', 'abc', '255.255.255.0.0', null, undefined])(
    'refuses %o',
    (mask) => {
      expect(maskToPrefix(mask)).toBeNull()
    },
  )
})

describe('prefixToMask', () => {
  it.each([
    [24, '255.255.255.0'],
    [16, '255.255.0.0'],
    [8, '255.0.0.0'],
    [30, '255.255.255.252'],
    [32, '255.255.255.255'],
    [1, '128.0.0.0'],
  ])('expands /%i to %s', (prefix, expected) => {
    expect(prefixToMask(prefix)).toBe(expected)
  })

  it.each([0, 33, -1, 1.5, NaN, '', 'abc'])('refuses %o', (prefix) => {
    expect(prefixToMask(prefix)).toBe('')
  })

  it('round-trips every usable prefix length', () => {
    for (let prefix = 1; prefix <= 32; prefix++) {
      expect(maskToPrefix(prefixToMask(prefix))).toBe(prefix)
    }
  })
})

describe('cidr', () => {
  it('joins an address and its mask', () => {
    expect(cidr('10.10.128.20', '255.255.255.0')).toBe('10.10.128.20/24')
  })

  it('falls back to the bare address when the mask is unusable', () => {
    expect(cidr('10.10.128.20', 'nonsense')).toBe('10.10.128.20')
  })

  it('is empty without an address', () => {
    expect(cidr('', '255.255.255.0')).toBe('')
  })
})

describe('slugify', () => {
  it.each([
    ['Atelier — Ligne 1', 'atelier-ligne-1'],
    ['Défaut — DHCP', 'defaut-dhcp'],
    ['Site client B', 'site-client-b'],
    ['  Chantier   Lyon  ', 'chantier-lyon'],
    ['192.168.1.x', '192-168-1-x'],
  ])('turns %s into %s', (name, expected) => {
    expect(slugify(name)).toBe(expected)
  })

  it('never yields an empty id', () => {
    expect(slugify('———')).toBe('profil')
    expect(slugify('')).toBe('profil')
  })
})

describe('uniqueSlug', () => {
  it('keeps the plain slug when it is free', () => {
    expect(uniqueSlug('Atelier', [])).toBe('atelier')
  })

  it('numbers a collision', () => {
    expect(uniqueSlug('Atelier', ['atelier'])).toBe('atelier-2')
    expect(uniqueSlug('Atelier', ['atelier', 'atelier-2'])).toBe('atelier-3')
  })
})

describe('esc', () => {
  it('neutralises markup coming from a profile name or an SSID', () => {
    expect(esc('<img src=x onerror="alert(1)">')).toBe(
      '&lt;img src=x onerror=&quot;alert(1)&quot;&gt;',
    )
  })

  it('leaves ordinary names alone', () => {
    expect(esc('Atelier — Ligne 1')).toBe('Atelier — Ligne 1')
  })
})
