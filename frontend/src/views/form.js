import { icons } from '../icons.js'
import { esc } from '../format.js'
import { interfaceOptions } from '../components.js'

export function renderForm(state) {
  const draft = state.editing

  return `
    <header class="topbar">
      <div class="topbar__title">
        <button class="btn btn--ghost" data-action="cancel-form" title="Retour">${icons.back(14)}</button>
        <h1>${draft.isNew ? 'Nouveau profil' : 'Modifier le profil'}</h1>
      </div>
      <div class="topbar__actions">
        <button class="btn" data-action="cancel-form">Annuler</button>
        <button class="btn btn--primary" data-action="save-profile">Enregistrer</button>
      </div>
    </header>

    <div class="pane">
      <div class="pane__inner">
        ${state.formError ? `<div class="notice">${esc(state.formError)}</div>` : ''}

        <div class="form-row">
          <label for="profile-name">NOM DU PROFIL</label>
          <input class="field" id="profile-name" data-field="name" value="${esc(draft.name)}"
                 placeholder="Atelier — Ligne 1" autocomplete="off">
        </div>

        ${draft.targets.map((target, index) => targetCard(target, index, state)).join('')}

        <div>
          <button class="btn" data-action="add-target">${icons.plus(13)} Ajouter une interface</button>
        </div>
      </div>
    </div>
  `
}

function targetCard(target, index, state) {
  const iface = state.interfaces.find((candidate) => candidate.name === target.interface)
  const isWiFi = iface?.kind === 'wifi'
  const isStatic = target.mode === 'static'

  return `
    <div class="card">
      <div class="card__head">
        <span class="card__title">Interface ${index + 1}</span>
        ${
          state.editing.targets.length > 1
            ? `<button class="btn btn--ghost btn--danger" data-action="remove-target" data-index="${index}" title="Retirer">${icons.trash(14)}</button>`
            : ''
        }
      </div>

      <div class="form-grid">
        <div class="form-row">
          <label>CARTE RÉSEAU</label>
          <select class="field" data-field="interface" data-index="${index}">
            ${interfaceOptions(state, target.interface)}
          </select>
        </div>
        <div class="form-row">
          <label>MODE</label>
          <div class="seg">
            <button type="button" data-action="set-mode" data-index="${index}" data-mode="dhcp" aria-pressed="${!isStatic}">DHCP</button>
            <button type="button" data-action="set-mode" data-index="${index}" data-mode="static" aria-pressed="${isStatic}">Statique</button>
          </div>
        </div>
      </div>

      ${isWiFi ? wifiRow(target, index, state) : ''}
      ${isStatic ? staticRows(target, index) : '<div class="hint">Adresse, passerelle et DNS seront obtenus automatiquement.</div>'}
    </div>`
}

function wifiRow(target, index, state) {
  const known = state.wifiNetworks[target.interface] ?? []
  const options = known
    .map((ssid) => `<option value="${esc(ssid)}"${ssid === target.ssid ? ' selected' : ''}>${esc(ssid)}</option>`)
    .join('')

  return `
    <div class="form-row">
      <label>RÉSEAU WI-FI</label>
      <select class="field" data-field="ssid" data-index="${index}">
        <option value=""${target.ssid ? '' : ' selected'}>— rester sur le réseau courant —</option>
        ${options}
      </select>
      <span class="hint">Seuls les réseaux déjà enregistrés par Windows sont proposés : la clé est déjà connue.</span>
    </div>`
}

function staticRows(target, index) {
  return `
    <div class="form-grid">
      ${textField('ADRESSE IP', 'address', index, target.address, '10.10.128.20')}
      ${textField('MASQUE', 'mask', index, target.mask, '255.255.255.0')}
    </div>
    <div class="form-grid">
      ${textField('PASSERELLE', 'gateway', index, target.gateway, 'facultative')}
      ${textField('DNS', 'dns', index, (target.dns ?? []).join(', '), 'facultatifs, séparés par une virgule')}
    </div>`
}

function textField(label, field, index, value, placeholder) {
  return `
    <div class="form-row">
      <label>${label}</label>
      <input class="field field--mono" data-field="${field}" data-index="${index}"
             value="${esc(value ?? '')}" placeholder="${esc(placeholder)}" autocomplete="off" spellcheck="false">
    </div>`
}
