// Modal dialogs drawn by the page itself.
//
// window.confirm() and window.alert() are dead ends here: the macOS webview
// Wails builds on sets a WKUIDelegate that never implements the JavaScript
// panel methods, so the browser draws nothing and confirm() returns false
// straight away. A "Șterge" guarded by window.confirm therefore looked like a
// button that did nothing at all. These dialogs are ordinary elements, so they
// behave the same on every platform the app is built for.

/**
 * Builds a modal <dialog> around `message` and the given buttons, and resolves
 * with the returnValue of the button pressed. Closing with Esc leaves the
 * returnValue empty, which callers read as "no".
 */
function openModal(message: string, buttons: HTMLButtonElement[]): Promise<string> {
  return new Promise((resolve) => {
    const dialog = document.createElement('dialog');
    dialog.className = 'modal';

    const text = document.createElement('p');
    text.className = 'modal-text';
    // textContent, not innerHTML: the message can carry a document number or
    // an error string straight from Go, neither of which is escaped.
    text.textContent = message;

    const row = document.createElement('div');
    row.className = 'modal-buttons';
    buttons.forEach((button) => row.appendChild(button));

    dialog.appendChild(text);
    dialog.appendChild(row);
    document.body.appendChild(dialog);

    // One dialog per call rather than a reused element, so a dialog opened
    // from another dialog's handler cannot inherit the previous one's state.
    dialog.addEventListener('close', () => {
      const value = dialog.returnValue;
      dialog.remove();
      resolve(value);
    });

    dialog.showModal();
  });
}

/** A button that closes its dialog with `value` when pressed. */
function modalButton(label: string, value: string, className: string): HTMLButtonElement {
  const button = document.createElement('button');
  button.className = className;
  button.textContent = label;
  button.addEventListener('click', () => {
    const dialog = button.closest('dialog');
    dialog?.close(value);
  });
  return button;
}

/**
 * Asks `message` and resolves true only if the user picks "Da".
 *
 * "Nu" comes first in the DOM so showModal() lands the initial focus there:
 * the callers are destructive actions, and Enter on a freshly opened dialog
 * should back out, not go through with it.
 */
export function showConfirm(message: string): Promise<boolean> {
  const no = modalButton('Nu', 'nu', 'btn');
  const yes = modalButton('Da', 'da', 'btn btn-danger');
  return openModal(message, [no, yes]).then((value) => value === 'da');
}

/** Shows `message` with a single dismiss button. */
export function showAlert(message: string): Promise<void> {
  const ok = modalButton('OK', 'ok', 'btn btn-primary');
  return openModal(message, [ok]).then(() => undefined);
}
