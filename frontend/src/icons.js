// Stroke-based icons on a 16px grid, drawn inline so they inherit currentColor
// and stay crisp at any DPI.

const wrap = (body, size = 15, extra = '') =>
  `<svg width="${size}" height="${size}" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" ${extra}>${body}</svg>`

export const icons = {
  brand: (size = 20) =>
    `<svg width="${size}" height="${size}" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
       <rect x="7.4" y="2.4" width="5.2" height="5.2" rx="1.2"></rect>
       <rect x="1.6" y="12.4" width="5.2" height="5.2" rx="1.2"></rect>
       <rect x="13.2" y="12.4" width="5.2" height="5.2" rx="1.2"></rect>
       <path d="M10 7.6v2.6M4.2 12.4v-2.2h11.6v2.2"></path>
     </svg>`,

  profiles: (size) =>
    wrap('<rect x="2.4" y="3" width="11.2" height="3.4" rx="1"></rect><rect x="2.4" y="9.6" width="11.2" height="3.4" rx="1"></rect>', size, 'stroke-linejoin="round"'),

  bolt: (size) => wrap('<path d="M9.2 1.9L3.6 9.1h3.5l-.5 5 5.6-7.4H8.5z"></path>', size, 'stroke-linejoin="round"'),

  ethernet: (size) =>
    wrap('<rect x="2.4" y="4.6" width="11.2" height="7.6" rx="1.2"></rect><path d="M5.6 4.6v3M8 4.6v3M10.4 4.6v3"></path>', size, 'stroke-linejoin="round"'),

  wifi: (size) =>
    wrap('<path d="M2.2 6.3a8.6 8.6 0 0 1 11.6 0"></path><path d="M4.6 8.9a5.2 5.2 0 0 1 6.8 0"></path><circle cx="8" cy="11.8" r="0.95" fill="currentColor" stroke="none"></circle>', size, 'stroke-linecap="round"'),

  shield: (size) =>
    wrap('<path d="M8 1.9l4.9 1.9v3.9c0 2.9-2.1 5.2-4.9 6.3-2.8-1.1-4.9-3.4-4.9-6.3V3.8z"></path><path d="M6.1 7.9l1.5 1.5 2.7-2.9"></path>', size, 'stroke-linecap="round" stroke-linejoin="round"'),

  plus: (size) => wrap('<path d="M8 3.4v9.2M3.4 8h9.2"></path>', size, 'stroke-width="1.8" stroke-linecap="round"'),

  search: (size) => wrap('<circle cx="7" cy="7" r="4.4"></circle><path d="M10.3 10.3L14 14"></path>', size, 'stroke-linecap="round"'),

  check: (size) => wrap('<path d="M3.2 8.4l3 3 6.6-6.8"></path>', size, 'stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"'),

  pencil: (size) =>
    wrap('<path d="M11.1 2.9l2 2-7.2 7.2-2.6.6.6-2.6z"></path>', size, 'stroke-linecap="round" stroke-linejoin="round"'),

  trash: (size) =>
    wrap('<path d="M3.4 4.6h9.2M6.4 4.6V3.2h3.2v1.4M4.8 4.6l.5 8h5.4l.5-8"></path>', size, 'stroke-linecap="round" stroke-linejoin="round"'),

  back: (size) => wrap('<path d="M9.8 3.6L5.4 8l4.4 4.4"></path>', size, 'stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"'),

  pin: (size) =>
    wrap('<path d="M6.1 2.3h3.8l-.5 4.4 2.3 2v1.2H4.3V8.7l2.3-2z"></path><path d="M8 9.9v3.8"></path>', size, 'stroke-linecap="round" stroke-linejoin="round"'),

  refresh: (size) =>
    wrap('<path d="M13 8a5 5 0 1 1-1.6-3.7"></path><path d="M13.3 2.6v2.9h-2.9"></path>', size, 'stroke-linecap="round" stroke-linejoin="round"'),
}
