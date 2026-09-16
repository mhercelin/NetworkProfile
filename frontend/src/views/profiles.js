import { icons } from '../icons.js'
import { cidr, esc } from '../format.js'
import { adaptersStrip, ifaceIcon, isActive, modeBadge } from '../components.js'

export function renderProfiles(state) {
  const query = state.search.trim().toLowerCase()
  const matching = state.profiles.filter((p) => !query || p.name.toLowerCase().includes(query))

  return `
    <header class="topbar">
      <div class="topbar__title">
        <h1>Profils</h1>
        <span class="topbar__count">${state.profiles.length} enregistré${state.profiles.length > 1 ? 's' : ''}</span>
      </div>
      <div class="topbar__actions">
        <label class="search">
          ${icons.search(13)}
          <input type="search" data-search placeholder="Rechercher un profil…" value="${esc(state.search)}">
        </label>
        <button class="btn btn--primary" data-action="new-profile">${icons.plus(13)} Nouveau profil</button>
      </div>
    </header>

    ${adaptersStrip(state)}

    <div class="thead">
      <div class="col-name">PROFIL</div>
      <div class="col-addr">ADRESSE</div>
      <div class="col-gw">PASSERELLE</div>
      <div class="col-dns">DNS</div>
      <div class="col-mode">MODE</div>
      <div class="col-act"></div>
    </div>

    <div class="rows">${matching.length ? matching.map((p) => row(p, state)).join('') : emptyState(state)}</div>

    <footer class="statusbar">
      <span>${state.lastApplied ? `Dernier profil appliqué : ${esc(state.lastApplied.name)}` : 'Aucun profil appliqué depuis le démarrage'}</span>
      <span class="mono">${state.lastApplied ? `${esc(state.lastApplied.at)} · ${state.lastApplied.ms} ms` : ''}</span>
    </footer>
  `
}

function row(profile, state) {
  const targets = profile.targets ?? []
  const single = targets.length === 1 ? targets[0] : null
  const allDHCP = targets.every((t) => t.mode === 'dhcp')
  const active = isActive(profile, state.interfaces)
  const busy = state.applying === profile.id

  const address = single && single.mode === 'static' ? cidr(single.address, single.mask) : ''
  const gateway = single ? single.gateway : ''
  const dnsList = single ? (single.dns ?? []) : []
  const dns = dnsList.length > 1 ? `${dnsList[0]} +${dnsList.length - 1}` : (dnsList[0] ?? '')

  return `
    <div class="row${active ? ' row--active' : ''}">
      <div class="col-name">
        <span class="row__name" title="${esc(profile.name)}">${esc(profile.name)}</span>
        <span class="row__sub">${subtitle(targets, state)}</span>
      </div>
      <div class="col-addr">${cell(address)}</div>
      <div class="col-gw">${cell(gateway)}</div>
      <div class="col-dns">${cell(dns)}</div>
      <div class="col-mode">${modeBadge(!allDHCP)}</div>
      <div class="col-act">${actions(profile, active, busy, state)}</div>
    </div>`
}

// Deleting is confirmed in place rather than through a dialog box: the row
// itself asks, and a second click is needed to go through with it.
function actions(profile, active, busy, state) {
  if (state.confirmDelete === profile.id) {
    return `
      <span class="hint">Supprimer ?</span>
      <button class="btn btn--sm btn--ghost" data-action="cancel-delete">Non</button>
      <button class="btn btn--sm btn--danger" data-action="confirm-delete" data-id="${esc(profile.id)}">Oui</button>`
  }

  return `
    <span class="row__tools">
      <button class="btn btn--ghost" data-action="edit-profile" data-id="${esc(profile.id)}" title="Modifier">${icons.pencil(14)}</button>
      <button class="btn btn--ghost btn--danger" data-action="delete-profile" data-id="${esc(profile.id)}" title="Supprimer">${icons.trash(14)}</button>
    </span>
    ${
      active
        ? `<span class="row__applied">${icons.check(13)} Actif</span>`
        : `<button class="btn btn--sm" data-action="apply-profile" data-id="${esc(profile.id)}"${busy ? ' disabled' : ''}>${busy ? 'En cours…' : 'Appliquer'}</button>`
    }`
}

function subtitle(targets, state) {
  if (targets.length === 1) {
    const [target] = targets
    const suffix = target.ssid ? ` · ${esc(target.ssid)}` : ''
    return `${ifaceIcon(kindOf(target, state))} ${esc(target.interface)}${suffix}`
  }
  return `${ifaceIcon('ethernet')} ${targets.map((t) => esc(t.interface)).join(' · ')}`
}

// A profile stores an interface name, not its type, so that it survives being
// carried to a machine where the same name is a different kind of adapter.
function kindOf(target, state) {
  const iface = state.interfaces.find((candidate) => candidate.name === target.interface)
  if (iface) return iface.kind
  return target.ssid ? 'wifi' : 'ethernet'
}

function cell(value) {
  return value ? esc(value) : '<span class="empty-cell">—</span>'
}

function emptyState(state) {
  if (state.search.trim()) {
    return `<div class="empty"><span>Aucun profil ne correspond à « ${esc(state.search)} ».</span></div>`
  }
  return `<div class="empty">
    <span>Aucun profil enregistré.</span>
    <button class="btn btn--primary" data-action="new-profile">${icons.plus(13)} Créer le premier profil</button>
  </div>`
}
