// Polyfill ResizeObserver — used by cmdk, not implemented in jsdom
class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver = ResizeObserver;

// Polyfill scrollIntoView — used by cmdk for keyboard navigation, not in jsdom
Element.prototype.scrollIntoView = function () {};
