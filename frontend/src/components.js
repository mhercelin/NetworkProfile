import { icons } from './icons.js'
import { cidr, esc } from './format.js'

export function ifaceIcon(kind, size = 12) {
  const wifi = kind === 'wifi'
  return `<span class="iface-icon iface-icon--${wifi ? 'wifi' : 'ethernet'}">${
    wifi ? icons.wifi(size) : icons.ethernet(size)
  }</span>`
}

export function modeBadge(isStatic) {
  return isStatic
    ? '<span class="badge badge--static">STATIQUE</span>'
    : '<span class="badge">DHCP</span>'
}

export function adaptersStrip(state) {
  const shown = state.interfaces.filter((iface) => state.showVirtual || !iface.virtual)
  if (!shown.length) {
    return '<div class="adapters"><span class="hint">Aucune carte réseau détectée.</span></div>'
  }
  return `<div class="adapters">${shown.map(adapterCell).join('<span class="adapters__sep"></span>')}</div>`
}

function adapterCell(iface) {
  const address = cidr(iface.address, iface.mask)
  const connected = iface.up && address

  return `<div class="adapter">
    <span class="adapter__dot${connected ? '' : ' adapter__dot--off'}"></span>
    ${ifaceIcon(iface.kind, 13)}
    <span class="${connected ? '' : 'adapter__name--off'}">${esc(iface.name)}</span>
    ${iface.ssid ? `<span class="adapter__ssid">${esc(iface.ssid)}</span>` : ''}
    ${
      address
        ? `<span class="adapter__addr${connected ? '' : ' adapter__addr--off'}">${esc(address)}</span>
           ${modeBadge(!iface.dhcp)}`
        : '<span class="adapter__addr adapter__addr--off">non connecté</span>'
    }
    ${iface.virtual ? '<span class="badge badge--virtual">VIRTUELLE</span>' : ''}
  </div>`
}

export function interfaceOptions(state, selected, kind) {
  return state.interfaces
    .filter((iface) => (state.showVirtual || !iface.virtual) && (!kind || iface.kind === kind))
    .map(
      (iface) =>
        `<option value="${esc(iface.name)}"${iface.name === selected ? ' selected' : ''}>${esc(iface.name)}</option>`,
    )
    .join('')
}
