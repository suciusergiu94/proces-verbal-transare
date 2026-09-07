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
    showError('Nu s-a putut incarca lista de documente', err);
    documents = [];
  }

  const items = documents
    .map(
      (doc) => `
        <li>
          <a class="doc-link" href="#/document/${doc.id}">
            <span class="doc-nr">NR ${doc.nr}</span>
            <span class="doc-meta">${formatDateRO(doc.data)}${doc.gestiune ? ` — ${escapeHtml(doc.gestiune)}` : ''}</span>
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

/** Escapes text that goes into an innerHTML template. */
export function escapeHtml(value: string): string {
  const div = document.createElement('div');
  div.textContent = value;
  return div.innerHTML;
}
