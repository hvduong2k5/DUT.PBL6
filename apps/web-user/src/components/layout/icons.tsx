// Line icons taken verbatim from the Gói Huế kit prototype (mon-02/home-v02/index.html): search, cart,
// home, categories grid and gift. No other icons are used in the storefront shell.
const base = { width: 20, height: 20, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.8, "aria-hidden": true } as const;

export function SearchIcon() {
  return <svg {...base}><circle cx="11" cy="11" r="7" /><path d="M16 16l4.5 4.5" strokeLinecap="round" /></svg>;
}
export function CartIcon() {
  return <svg {...base}><path d="M4 8h16l-1.2 12H5.2L4 8z" /><path d="M8 8V6a4 4 0 0 1 8 0v2" /></svg>;
}
export function HomeIcon() {
  return <svg {...base}><path d="M3 10.5L12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1v-9.5z" /></svg>;
}
export function GridIcon() {
  return <svg {...base}><rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="3" y="14" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /></svg>;
}
export function GiftIcon() {
  return <svg {...base}><rect x="3" y="8" width="18" height="13" rx="1" /><path d="M12 8v13M3 12h18M12 8c-2-3-5-3-5-1s3 1 5 1 5 1 5-1-3-2-5 1z" /></svg>;
}
