import './style.css'
import {
  ApplyProfile,
  ApplyTarget,
  DeleteProfile,
  Interfaces,
  KnownWiFiNetworks,
  Profiles,
  SaveProfile,
} from '../wailsjs/go/gui/App'

import { icons } from './icons.js'
import { clockTime, esc, prefixToMask, uniqueSlug } from './format.js'
import { renderProfiles } from './views/profiles.js'
import { renderForm } from './views/form.js'
import { renderQuick } from './views/quick.js'

const root = document.getElementById('app')

const state = {
  view: 'profiles',
  profiles: [],
  interfaces: [],
  wifiNetworks: {},
  showVirtual: false,
  search: '',
  editing: null,
  formError: '',
  confirmDelete: '',
  quick: blankQuick(),
  quickError: '',
  quickDone: '',
  applying: '',
  lastApplied: null,
  fatal: '',
}

// Nearly every network this tool is pointed at is a /24.
const DEFAULT_MASK = '255.255.255.0'

function blankQuick() {
  return { interface: '', mode: 'dhcp', ssid: '', address: '', mask: '', gateway: '', dns: '' }
}

// A profile is almost always named after the adapter it drives, so the name
// starts there and the operator only types what distinguishes it.
function prefixedName(iface) {
  return iface ? `${iface} — ` : ''
}

/* ---------------- rendering ---------------- */

const NAV = [
  { view: 'profiles', label: 'Profils', icon: 'profiles' },
  { view: 'quick', label: 'Changement rapide', icon: 'bolt' },
]

function rail() {
  const current = state.view === 'form' ? 'profiles' : state.view

  return `
    <aside class="rail">
      <div class="rail__brand">${icons.brand(20)} NetworkProfile</div>
      <div class="rail__label">NAVIGATION</div>
      <nav class="rail__nav">
        ${NAV.map(
          (item) => `
          <button class="rail__item" data-action="navigate" data-view="${item.view}"
                  ${current === item.view ? 'aria-current="page"' : ''}>
            ${icons[item.icon](15)} ${item.label}
          </button>`,
        ).join('')}
      </nav>
      <div style="flex: 1"></div>
      <div class="rail__foot">
        <label class="rail__version" style="display: flex; align-items: center; gap: 6px; cursor: pointer;">
          <input type="checkbox" data-action="toggle-virtual" ${state.showVirtual ? 'checked' : ''}>
          Cartes virtuelles
        </label>
        <div class="rail__elevated">${icons.shield(13)} Mode administrateur</div>
        <div class="rail__version">v0.1.0</div>
      </div>
    </aside>`
}

function viewHtml() {
  if (state.fatal) {
    return `<div class="pane"><div class="pane__inner"><div class="notice">${esc(state.fatal)}</div></div></div>`
  }
  if (state.view === 'form') return renderForm(state)
  if (state.view === 'quick') return renderQuick(state)
  return renderProfiles(state)
}

// A full re-render would drop the caret of whatever field is being typed in, so
// the focused field is located again afterwards by its data-field name.
function captureFocus() {
  const el = document.activeElement
  if (!el || !('selectionStart' in el)) return null
  return {
    field: el.dataset.field ?? (el.hasAttribute('data-search') ? '@search' : ''),
    index: el.dataset.index ?? '',
    start: el.selectionStart,
    end: el.selectionEnd,
  }
}

function restoreFocus(saved) {
  if (!saved?.field) return

  const selector =
    saved.field === '@search'
      ? '[data-search]'
      : `[data-field="${saved.field}"]${saved.index ? `[data-index="${saved.index}"]` : ''}`
  const el = root.querySelector(selector)
  if (!el) return

  el.focus()
  if ('setSelectionRange' in el && saved.start !== null) {
    try {
      el.setSelectionRange(saved.start, saved.end)
    } catch {
      /* non-text input types refuse a selection range */
    }
  }
}

function render() {
  const focus = captureFocus()
  root.innerHTML = `${rail()}<section class="main">${viewHtml()}</section>`
  restoreFocus(focus)
}

/* ---------------- data ---------------- */

async function loadAll() {
  try {
    const [profiles, interfaces] = await Promise.all([Profiles(), Interfaces()])
    state.profiles = profiles ?? []
    state.interfaces = interfaces ?? []
    state.fatal = ''
  } catch (err) {
    state.fatal = `Lecture impossible : ${err}`
    render()
    return
  }

  if (!state.quick.interface) {
    state.quick.interface = defaultInterface()
  }
  render()
  await loadWiFiNetworks()
}

async function loadWiFiNetworks() {
  const wifi = state.interfaces.filter((iface) => iface.kind === 'wifi' && !iface.virtual)
  let changed = false

  for (const iface of wifi) {
    try {
      state.wifiNetworks[iface.name] = (await KnownWiFiNetworks(iface.name)) ?? []
      changed = true
    } catch {
      /* an adapter that has never joined a network simply has none */
    }
  }
  if (changed) render()
}

function defaultInterface() {
  const visible = state.interfaces.filter((iface) => !iface.virtual)
  return (visible[0] ?? state.interfaces[0])?.name ?? ''
}

/* ---------------- actions ---------------- */

async function applyProfile(id) {
  const profile = state.profiles.find((p) => p.id === id)
  state.applying = id
  render()

  const started = performance.now()
  try {
    await ApplyProfile(id)
    state.lastApplied = {
      name: profile?.name ?? id,
      at: clockTime(),
      ms: Math.round(performance.now() - started),
    }
    state.fatal = ''
  } catch (err) {
    state.fatal = `${profile?.name ?? id} : ${err}`
  } finally {
    state.applying = ''
  }

  await reloadInterfaces()
}

async function reloadInterfaces() {
  try {
    state.interfaces = (await Interfaces()) ?? []
  } catch (err) {
    state.fatal = `Lecture des cartes réseau : ${err}`
  }
  render()
}

function draftFromProfile(profile) {
  return {
    isNew: false,
    id: profile.id,
    name: profile.name,
    // An existing name is the operator's own: never rewrite it.
    nameTouched: true,
    targets: (profile.targets ?? []).map((target) => ({ ...target, dns: [...(target.dns ?? [])] })),
  }
}

function blankDraft() {
  const iface = defaultInterface()
  return {
    isNew: true,
    id: '',
    name: prefixedName(iface),
    nameTouched: false,
    targets: [blankTarget(iface)],
  }
}

// A new profile starts static: switching an adapter to a fixed address is what
// this tool is for, and DHCP is one click away.
function blankTarget(iface) {
  return {
    interface: iface,
    mode: 'static',
    ssid: '',
    address: '',
    mask: DEFAULT_MASK,
    gateway: '',
    dns: [],
  }
}

function parseDNS(value) {
  return String(value ?? '')
    .split(/[\s,;]+/)
    .filter(Boolean)
}

function toProfile(draft) {
  const takenIDs = state.profiles.filter((p) => p.id !== draft.id).map((p) => p.id)

  // A name left at just the adapter prefix keeps the adapter, not the dash.
  const name = draft.name.replace(/\s*—\s*$/, '').trim()

  return {
    id: draft.id || uniqueSlug(name, takenIDs),
    name,
    targets: draft.targets.map((target) => {
      const base = { interface: target.interface, mode: target.mode }
      if (target.ssid) base.ssid = target.ssid
      if (target.mode !== 'static') return base

      return {
        ...base,
        address: target.address?.trim() ?? '',
        mask: target.mask?.trim() ?? '',
        gateway: target.gateway?.trim() ?? '',
        dns: Array.isArray(target.dns) ? target.dns : parseDNS(target.dns),
      }
    }),
  }
}

async function saveProfile() {
  const draft = state.editing
  if (!draft.name.replace(/\s*—\s*$/, '').trim()) {
    state.formError = 'Le nom du profil est obligatoire.'
    render()
    return
  }

  try {
    await SaveProfile(toProfile(draft))
    state.editing = null
    state.formError = ''
    state.profiles = (await Profiles()) ?? []
    state.view = 'profiles'
  } catch (err) {
    state.formError = String(err)
  }
  render()
}

async function deleteProfile(id) {
  try {
    await DeleteProfile(id)
    state.profiles = (await Profiles()) ?? []
    state.confirmDelete = ''
  } catch (err) {
    state.fatal = String(err)
  }
  render()
}

async function applyQuick() {
  const draft = state.quick
  const target = { interface: draft.interface, mode: draft.mode }
  if (draft.ssid) target.ssid = draft.ssid
  if (draft.mode === 'static') {
    target.address = draft.address.trim()
    target.mask = draft.mask.trim()
    target.gateway = draft.gateway.trim()
    target.dns = parseDNS(draft.dns)
  }

  state.applying = 'quick'
  state.quickError = ''
  state.quickDone = ''
  render()

  try {
    await ApplyTarget(target)
    state.quickDone = `${draft.interface} reconfigurée à ${clockTime()}.`
  } catch (err) {
    state.quickError = String(err)
  } finally {
    state.applying = ''
  }

  await reloadInterfaces()
}

function quickAsProfile() {
  const draft = state.quick
  const target = { interface: draft.interface, mode: draft.mode, ssid: draft.ssid }
  if (draft.mode === 'static') {
    Object.assign(target, {
      address: draft.address,
      mask: draft.mask,
      gateway: draft.gateway,
      dns: parseDNS(draft.dns),
    })
  }

  state.editing = {
    isNew: true,
    id: '',
    name: prefixedName(draft.interface),
    nameTouched: false,
    targets: [target],
  }
  state.formError = ''
  state.view = 'form'
  render()
}

/* ---------------- events ---------------- */

root.addEventListener('click', (event) => {
  const trigger = event.target.closest('[data-action]')
  if (!trigger) return
  const { action, view, id, index, mode } = trigger.dataset

  switch (action) {
    case 'navigate':
      state.view = view
      state.editing = null
      render()
      break

    case 'toggle-virtual':
      state.showVirtual = trigger.checked
      render()
      break

    case 'new-profile':
      state.editing = blankDraft()
      state.formError = ''
      state.view = 'form'
      render()
      break

    case 'edit-profile': {
      const profile = state.profiles.find((p) => p.id === id)
      if (!profile) return
      state.editing = draftFromProfile(profile)
      state.formError = ''
      state.view = 'form'
      render()
      break
    }

    case 'delete-profile':
      state.confirmDelete = id
      render()
      break

    case 'confirm-delete':
      deleteProfile(id)
      break

    case 'cancel-delete':
      state.confirmDelete = ''
      render()
      break

    case 'apply-profile':
      applyProfile(id)
      break

    case 'cancel-form':
      state.editing = null
      state.formError = ''
      state.view = 'profiles'
      render()
      break

    case 'save-profile':
      saveProfile()
      break

    case 'add-target':
      state.editing.targets.push(blankTarget(defaultInterface()))
      render()
      break

    case 'remove-target':
      state.editing.targets.splice(Number(index), 1)
      render()
      break

    case 'set-mode': {
      const target = state.editing.targets[Number(index)]
      target.mode = mode
      if (mode === 'static' && !target.mask) target.mask = DEFAULT_MASK
      render()
      break
    }

    case 'quick-mode':
      state.quick.mode = mode
      if (mode === 'static' && !state.quick.mask) state.quick.mask = DEFAULT_MASK
      render()
      break

    case 'quick-apply':
      applyQuick()
      break

    case 'quick-save-as-profile':
      quickAsProfile()
      break
  }
})

// Text fields update the model without re-rendering, which is what keeps typing
// smooth; selects and structural controls re-render through the click handler.
root.addEventListener('input', (event) => {
  const el = event.target
  if (el.hasAttribute('data-search')) {
    state.search = el.value
    render()
    return
  }

  const { field, index } = el.dataset
  if (!field) return

  if (field.startsWith('quick-')) {
    state.quick[field.slice('quick-'.length)] = el.value
    return
  }
  if (field === 'name') {
    state.editing.name = el.value
    state.editing.nameTouched = true
    return
  }

  const target = state.editing?.targets?.[Number(index)]
  if (!target) return
  target[field] = field === 'dns' ? parseDNS(el.value) : el.value
})

root.addEventListener('change', (event) => {
  const el = event.target
  const { field, index } = el.dataset
  if (!field) return

  // "24" is how an engineer says 255.255.255.0.
  if (field.endsWith('mask') && /^\d{1,2}$/.test(el.value.trim())) {
    const expanded = prefixToMask(el.value.trim())
    if (expanded) el.value = expanded
  }

  if (field === 'quick-interface' || field === 'quick-ssid') {
    state.quick[field.slice('quick-'.length)] = el.value
    if (field === 'quick-interface') state.quick.ssid = ''
    render()
    return
  }
  if (field.startsWith('quick-')) {
    state.quick[field.slice('quick-'.length)] = el.value
    return
  }

  const target = state.editing?.targets?.[Number(index)]
  if (!target) return

  if (field === 'interface') {
    target.interface = el.value
    target.ssid = ''
    // The prefix follows the first adapter until the operator types a name of
    // their own, which is then left alone.
    if (Number(index) === 0 && !state.editing.nameTouched) {
      state.editing.name = prefixedName(el.value)
    }
    render()
    return
  }
  if (field === 'ssid') {
    target.ssid = el.value
    render()
    return
  }
  target[field] = field === 'dns' ? parseDNS(el.value) : el.value
})

loadAll()
