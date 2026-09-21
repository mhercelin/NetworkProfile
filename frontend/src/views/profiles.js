import { icons } from '../icons.js'
import { esc } from '../format.js'
import { adaptersStrip, ifaceIcon, modeBadge } from '../components.js'
import { filterProfiles, interfacesInUse, rowData, sortProfiles } from '../table.js'

// The spacer sits between the last value column and the actions, and absorbs
// whatever a wide window leaves over. Without it that slack fell to the name
// column, which then filled three quarters of a maximised screen for nothing.
const COLUMNS = [
  { key: 'name', label: 'PROFIL', resizable: true },
  { key: 'address', label: 'ADRESSE', resizable: true },
  { key: 'gateway', label: 'PASSERELLE', resizable: true },
  { key: 'dns', label: 'DNS', resizable: true },
  { key: 'mode', label: 'MODE', resizable: true },
  { key: 'spacer', label: '', resizable: false, sortable: false },
  { key: 'actions', label: '', resizable: false, sortable: false },
]

export function renderProfiles(state) {
  const shown = sortProfiles(
    filterProfiles(state.profiles, { query: state.search, iface: state.filterInterface }),
    state.sort,
  )

  return `
    <header class="topbar">
      <div class="topbar__title">
        <h1>Profils</h1>
        <span class="topbar__count">${countLabel(state, shown.length)}</span>
      </div>
      <div class="topbar__actions">
        ${interfaceFilter(state)}
        <label class="search">
          ${icons.search(13)}
          <input type="search" data-search placeholder="Rechercher un profil…" value="${esc(state.search)}">
        </label>
        <button class="btn btn--primary" data-action="new-profile">${icons.plus(13)} Nouveau profil</button>
      </div>
    </header>

    ${adaptersStrip(state)}

    ${
      state.error
        ? `<div class="banner">
             <span class="banner__text">${esc(state.error)}</span>
             <button class="btn btn--ghost btn--sm" data-action="dismiss-error">Fermer</button>
           </div>`
        : ''
    }

    <div class="table" style="${columnStyle(state)}">
      <div class="thead">${COLUMNS.map((c) => headerCell(c, state)).join('')}</div>
      <div class="rows">${shown.length ? shown.map((p) => row(p, state)).join('') : emptyState(state)}</div>
    </div>

    <footer class="statusbar">
      <span>${state.lastApplied?.name ? `Dernier profil appliqué : ${esc(state.lastApplied.name)}` : 'Aucun profil appliqué depuis le démarrage'}</span>
      <span class="mono">${state.lastApplied?.name ? `${esc(state.lastApplied.at)} · ${state.lastApplied.millis} ms` : ''}</span>
    </footer>
  `
}

function countLabel(state, shownCount) {
  const total = state.profiles.length
  const plural = total > 1 ? 's' : ''
  if (shownCount === total) return `${total} enregistré${plural}`
  return `${shownCount} sur ${total}`
}

function interfaceFilter(state) {
  const interfaces = interfacesInUse(state.profiles)
  if (interfaces.length < 2) return ''

  const options = interfaces
    .map((name) => `<option value="${esc(name)}"${name === state.filterInterface ? ' selected' : ''}>${esc(name)}</option>`)
    .join('')

  return `
    <select class="field field--filter${state.filterInterface ? ' field--filter-on' : ''}"
            data-field="filter-interface" title="N'afficher que les profils touchant une carte">
      <option value=""${state.filterInterface ? '' : ' selected'}>Toutes les cartes</option>
      ${options}
    </select>`
}

// The widths live in a custom property so the header and every row share one
// grid, and a drag only has to rewrite this one line.
function columnStyle(state) {
  const widths = state.columns
  // Six gaps for seven columns. The 20px inset on either end is inside the
  // first and last tracks, not beside them, so it is not added again here.
  const gaps = 14 * 6
  const total =
    widths.name + widths.address + widths.gateway + widths.dns + widths.mode + widths.actions + gaps

  return [
    ...Object.entries(widths).map(([key, width]) => `--w-${key}: ${width}px`),
    // Below this the table scrolls sideways rather than squeezing values into
    // illegibility.
    `--table-min: ${total}px`,
  ].join('; ')
}

function headerCell(column, state) {
  const sorted = state.sort?.key === column.key
  const arrow = sorted ? (state.sort.dir === 'desc' ? '↓' : '↑') : ''
  const handle = column.resizable ? `<span class="col-resize" data-resize="${column.key}" title="Redimensionner"></span>` : ''

  if (column.sortable === false) {
    return `<div class="thead__cell thead__cell--${column.key}">${handle}</div>`
  }

  return `
    <div class="thead__cell${sorted ? ' is-sorted' : ''}">
      <button class="thead__sort" data-action="sort-by" data-key="${column.key}"
              title="Trier par ${column.label.toLowerCase()}">
        ${column.label}<span class="thead__arrow">${arrow}</span>
      </button>
      ${handle}
    </div>`
}

function row(profile, state) {
  const data = rowData(profile)
  // Several profiles can be in effect at once, one per adapter.
  const active = state.activeIds.includes(profile.id)
  const busy = state.applying === profile.id

  return `
    <div class="row${active ? ' row--active' : ''}">
      <div class="col-name">
        <span class="row__name" title="${esc(data.name)}">
          <span class="row__pin${profile.pinned ? '' : ' row__pin--none'}"
                title="${profile.pinned ? 'Épinglé dans la zone de notification' : ''}">${icons.pin(12)}</span>
          ${esc(data.name)}
        </span>
        <span class="row__sub">
          ${subtitle(profile.targets ?? [], state)}
          ${active ? `<span class="row__applied" title="Les cartes portent déjà cette configuration">${icons.check(11)} en place</span>` : ''}
        </span>
      </div>
      <div class="col-value">${cell(data.address)}</div>
      <div class="col-value col-value--dim">${cell(data.gateway)}</div>
      <div class="col-value col-value--dim">${cell(data.dns)}</div>
      <div class="col-mode">${modeBadge(data.mode === 'STATIQUE')}</div>
      <div class="col-spacer"></div>
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

  // The button is always there, even on a profile believed to be in effect.
  // Deciding that a profile is already applied is an inference over the adapter
  // state, and an inference that turns out wrong must never be able to lock the
  // operator out of applying it — which is exactly what happened when a Wi-Fi
  // profile was wrongly reported as active and had no button left to click.
  //
  // The "in effect" mark belongs on the profile name, not here: a tick that
  // appears beside the button the moment it is pressed reads as "done", and one
  // that is still there an hour later then looks stuck. Beside the name it
  // reads as what it is — a property of the row.
  return `
    <span class="row__tools">
      <button class="btn btn--ghost${profile.pinned ? ' is-pinned' : ''}" data-action="toggle-pin" data-id="${esc(profile.id)}"
              title="${profile.pinned ? 'Retirer de la zone de notification' : 'Épingler dans la zone de notification'}">${icons.pin(14)}</button>
      <button class="btn btn--ghost" data-action="edit-profile" data-id="${esc(profile.id)}" title="Modifier">${icons.pencil(14)}</button>
      <button class="btn btn--ghost btn--danger" data-action="delete-profile" data-id="${esc(profile.id)}" title="Supprimer">${icons.trash(14)}</button>
    </span>
    <button class="btn btn--sm" data-action="apply-profile" data-id="${esc(profile.id)}"${busy ? ' disabled' : ''}>${busy ? 'En cours…' : 'Appliquer'}</button>`
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
  return value ? `<span title="${esc(value)}">${esc(value)}</span>` : '<span class="empty-cell">—</span>'
}

function emptyState(state) {
  if (state.search.trim() || state.filterInterface) {
    return `<div class="empty">
      <span>Aucun profil ne correspond au filtre.</span>
      <button class="btn btn--sm" data-action="clear-filters">Effacer le filtre</button>
    </div>`
  }
  return `<div class="empty">
    <span>Aucun profil enregistré.</span>
    <button class="btn btn--primary" data-action="new-profile">${icons.plus(13)} Créer le premier profil</button>
  </div>`
}
