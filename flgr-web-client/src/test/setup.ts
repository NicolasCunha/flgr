import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from '@/test/msw/server'

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

// jsdom implements neither the Pointer Events capture API nor
// scrollIntoView; Radix UI's Select (and other Radix primitives)
// call these during pointer interaction, so without a stub every
// @testing-library/user-event click on a Radix trigger throws
// "target.hasPointerCapture is not a function" instead of opening the
// component. These are inert no-op/false stubs — just enough surface
// area for Radix's interaction logic to run under jsdom — not a
// behavioral polyfill of the real Pointer Capture spec.
if (!Element.prototype.hasPointerCapture) {
  Element.prototype.hasPointerCapture = () => false
}
if (!Element.prototype.setPointerCapture) {
  Element.prototype.setPointerCapture = () => {}
}
if (!Element.prototype.releasePointerCapture) {
  Element.prototype.releasePointerCapture = () => {}
}
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {}
}

// jsdom has no ResizeObserver; Radix UI's Switch (via @radix-ui/react-use-size,
// first exercised by ServiceKeyForm's access/status toggles) calls it on
// mount to measure the thumb, which otherwise throws "ResizeObserver is not
// defined" the moment any test renders one. A minimal no-op stub is enough —
// nothing here asserts on actual resize callbacks.
if (!window.ResizeObserver) {
  window.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

// jsdom has no window.matchMedia; next-themes (via the shared <Toaster />
// mounted in App.tsx, see src/components/ui/sonner.tsx) calls it on mount
// to resolve the "system" theme, which otherwise throws
// "window.matchMedia is not a function" the moment any test renders <App />.
if (!window.matchMedia) {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })
}
