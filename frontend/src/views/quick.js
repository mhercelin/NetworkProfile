import { icons } from '../icons.js'
import { esc } from '../format.js'
import { adaptersStrip, interfaceOptions } from '../components.js'

export function renderQuick(state) {
  const draft = state.quick
  const iface = state.interfaces.find((candidate) => candidate.name === draft.interface)
  const isWiFi = iface?.kind === 'wifi'
  const isStatic = draft.mode === 'static'
  const known = state.wifiNetworks[draft.interface] ?? []

  return `
    <header class="topbar">
      <div class="topbar__title">
        <h1>Changement rapide</h1>
        <span class="topbar__count">sans enregistrer de profil</span>
      </div>
      <div class="topbar__actions">
        <button class="btn" data-action="quick-save-as-profile">Enregistrer comme profil</button>
        <button class="btn btn--primary" data-action="quick-apply"${state.applying === 'quick' ? ' disabled' : ''}>
          ${icons.bolt(13)} ${state.applying === 'quick' ? 'Application…' : 'Appliquer'}
        </button>
      </div>
    </header>

    ${adaptersStrip(state)}

    <div class="pane">
      <div class="pane__inner">
        ${state.quickError ? `<div class="notice">${esc(state.quickError)}</div>` : ''}
        ${state.quickDone ? `<div class="notice notice--ok">${esc(state.quickDone)}</div>` : ''}

        <div class="card">
          <div class="form-grid">
            <div class="form-row">
              <label>CARTE RÉSEAU</label>
              <select class="field" data-field="quick-interface">
                ${interfaceOptions(state, draft.interface)}
              </select>
            </div>
            <div class="form-row">
              <label>MODE</label>
              <div class="seg">
                <button type="button" data-action="quick-mode" data-mode="dhcp" aria-pressed="${!isStatic}">DHCP</button>
                <button type="button" data-action="quick-mode" data-mode="static" aria-pressed="${isStatic}">Statique</button>
              </div>
            </div>
          </div>

          ${
            isWiFi
              ? `<div class="form-row">
                   <label>RÉSEAU WI-FI</label>
                   <select class="field" data-field="quick-ssid">
                     <option value=""${draft.ssid ? '' : ' selected'}>— rester sur le réseau courant —</option>
                     ${known.map((ssid) => `<option value="${esc(ssid)}"${ssid === draft.ssid ? ' selected' : ''}>${esc(ssid)}</option>`).join('')}
                   </select>
                 </div>`
              : ''
          }

          ${
            isStatic
              ? `<div class="form-grid">
                   ${field('ADRESSE IP', 'quick-address', draft.address, '192.168.1.50')}
                   ${field('MASQUE', 'quick-mask', draft.mask, '255.255.255.0')}
                 </div>
                 <div class="form-grid">
                   ${field('PASSERELLE', 'quick-gateway', draft.gateway, 'facultative')}
                   ${field('DNS', 'quick-dns', draft.dns, 'facultatifs, séparés par une virgule')}
                 </div>`
              : '<div class="hint">La carte repassera en attribution automatique.</div>'
          }
        </div>

        <div class="hint">
          Cette configuration n'est pas conservée : elle disparaît au prochain changement de profil.
        </div>
      </div>
    </div>

    <footer class="statusbar">
      <span>${state.lastApplied?.name ? `Dernier profil appliqué : ${esc(state.lastApplied.name)}` : 'Aucun profil appliqué depuis le démarrage'}</span>
      <span class="mono">${state.lastApplied?.name ? `${esc(state.lastApplied.at)} · ${state.lastApplied.millis} ms` : ''}</span>
    </footer>
  `
}

function field(label, key, value, placeholder) {
  return `
    <div class="form-row">
      <label>${label}</label>
      <input class="field field--mono" data-field="${key}" value="${esc(value ?? '')}"
             placeholder="${esc(placeholder)}" autocomplete="off" spellcheck="false">
    </div>`
}
