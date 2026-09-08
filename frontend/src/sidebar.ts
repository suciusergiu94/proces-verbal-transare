import { DocumentSummary, ListDocuments, showError } from './api';
import { formatDateRO } from './format';
import { navigate } from './router';

let hashchangeListenerRegistered = false;

/** The route an unsaved document lives at. */
export const DRAFT_HASH = '#/document/new';

/**
 * The hash to match sidebar entries against. A freshly launched app has an
 * empty hash and the router falls back to the new-document route, so the same
 * fallback is applied here — otherwise the draft tab would be missing, and no
 * entry highlighted, on the one screen the app always opens on.
 */
export function currentHash(hash: string = window.location.hash): string {
  return hash || DRAFT_HASH;
}

/** Renders the sidebar: new-document button, document history, settings link. */
export async function renderSidebar(el: HTMLElement): Promise<void> {
  let documents: DocumentSummary[];
  try {
    documents = await ListDocuments();
  } catch (err) {
    showError('Nu s-a putut încărca lista de documente', err);
    documents = [];
  }

  const items = documents
    .map(
      (doc) => `
        <li>
          <a class="doc-link" href="#/document/${doc.id}">
            <span class="doc-nr">NR ${doc.nr}</span>
            <span class="doc-meta">${escapeHtml(formatDateRO(doc.data))}${doc.gestiune ? ` — ${escapeHtml(doc.gestiune)}` : ''}</span>
          </a>
        </li>`,
    )
    .join('');

  // The draft entry is rendered on every pass and shown or hidden by
  // markActive, so moving in and out of #/document/new only toggles an
  // attribute instead of rebuilding a sidebar whose document list costs a
  // round trip to SQLite. It sits above the saved documents because the list
  // is newest-first and the unsaved one is newer than all of them.
  el.innerHTML = `
    <button class="btn btn-primary" id="new-doc">+ Document nou</button>
    <ul class="doc-list">
      <li id="draft-item" hidden>
        <a class="doc-link" href="${DRAFT_HASH}">
          <span class="doc-nr">Document nou</span>
          <span class="doc-meta draft-meta">Nesalvat</span>
        </a>
      </li>
      ${items}
    </ul>
    <a class="settings-link" href="#/setari">Setări</a>
  `;

  el.querySelector<HTMLButtonElement>('#new-doc')!.addEventListener('click', () => {
    navigate(DRAFT_HASH);
  });

  markActive(el);

  if (!hashchangeListenerRegistered) {
    hashchangeListenerRegistered = true;
    window.addEventListener('hashchange', () => markActive(el));
  }
}

function markActive(el: HTMLElement): void {
  const hash = currentHash();
  el.querySelector('#draft-item')?.toggleAttribute('hidden', hash !== DRAFT_HASH);
  el.querySelectorAll('a').forEach((link) => {
    link.classList.toggle('active', link.getAttribute('href') === hash);
  });
}

/**
 * Escapes text that goes into an innerHTML template. Almost every call site
 * places the result inside an HTML attribute (`value="${escapeHtml(...)}"`),
 * so quotes must be escaped too — a `div.textContent`/`innerHTML` round trip
 * escapes `&`, `<` and `>` but leaves `"` and `'` untouched, which lets a
 * quote in user data break out of the attribute (corrupting the saved value)
 * or inject an arbitrary attribute/event handler. Ampersand must be escaped
 * first so the other replacements' `&...;` sequences are not re-escaped.
 */
export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
