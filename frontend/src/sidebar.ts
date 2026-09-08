import { DocumentSummary, ListDocuments, showError } from './api';
import { formatDateRO } from './format';
import { navigate } from './router';

let hashchangeListenerRegistered = false;

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

  el.innerHTML = `
    <button class="btn btn-primary" id="new-doc">+ Document nou</button>
    ${documents.length > 0 ? `<ul class="doc-list">${items}</ul>` : ''}
    <a class="settings-link" href="#/setari">Setări</a>
  `;

  el.querySelector<HTMLButtonElement>('#new-doc')!.addEventListener('click', () => {
    navigate('#/document/new');
  });

  markActive(el);

  if (!hashchangeListenerRegistered) {
    hashchangeListenerRegistered = true;
    window.addEventListener('hashchange', () => markActive(el));
  }
}

function markActive(el: HTMLElement): void {
  el.querySelectorAll('a').forEach((link) => {
    link.classList.toggle('active', link.getAttribute('href') === window.location.hash);
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
