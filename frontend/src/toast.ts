// A brief confirmation that floats over the bottom of the window and fades
// out on its own. Deliberately not a window.alert: a save is the most
// frequent action in this app, and a blocking dialog would put a click
// between the user and every single save.

/** The single reused toast element, created on first use. */
let element: HTMLDivElement | undefined;
let hideTimer: number | undefined;

const VISIBLE_MS = 1800;

/**
 * Shows `message` over the bottom of the window for a moment.
 *
 * One element is reused rather than one per call, so saving twice in quick
 * succession restarts the same toast instead of stacking two of them.
 */
export function showToast(message: string): void {
  if (!element) {
    element = document.createElement('div');
    element.className = 'toast';
    // Announced by screen readers without stealing focus from the form.
    element.setAttribute('role', 'status');
    element.setAttribute('aria-live', 'polite');
    document.body.appendChild(element);
  }

  element.textContent = message;

  // Drop and re-add the class so the entry animation replays even when the
  // toast is already on screen; reading offsetWidth forces the reflow that
  // makes the browser treat this as two separate style changes.
  element.classList.remove('toast-visible');
  void element.offsetWidth;
  element.classList.add('toast-visible');

  window.clearTimeout(hideTimer);
  hideTimer = window.setTimeout(() => {
    element?.classList.remove('toast-visible');
  }, VISIBLE_MS);
}
