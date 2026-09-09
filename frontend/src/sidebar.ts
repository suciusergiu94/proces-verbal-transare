import { DocumentSummary, ListDocuments, ListTemplates, showError } from './api';
import type { Template } from './api';
import { showAlert } from './dialog';
import { formatDateRO } from './format';
import { navigate } from './router';
import { DRAFT_PREFIX, draftHash, draftTemplateId, isDraftHash } from './templates';

let hashchangeListenerRegistered = false;
let documentClickListenerRegistered = false;

// The templates as of the most recent renderSidebar call. renderSidebar runs
// again on every save/delete (`refreshSidebar`), rebuilding `el`'s contents
// each time, but the listeners below are registered only once — against
// `window`/`document`, which outlive any single render — so they must read
// this mutable slot at event time rather than close over one render's local
// `templates`, or they would keep acting on whichever array existed when
// they were first registered.
let currentTemplates: Template[] = [];

export { DRAFT_PREFIX } from './templates';

/**
 * The hash to match sidebar entries against. A freshly launched app has an
 * empty hash and the router falls back to the new-document route, so the same
 * fallback is applied here — otherwise the draft tab would be missing, and no
 * entry highlighted, on the one screen the app always opens on.
 */
export function currentHash(hash: string = window.location.hash): string {
  return hash || DRAFT_PREFIX;
}

/** Renders the sidebar: new-document button, document history, settings link. */
export async function renderSidebar(el: HTMLElement): Promise<void> {
  let documents: DocumentSummary[];
  let templates: Template[];
  try {
    [documents, templates] = await Promise.all([ListDocuments(), ListTemplates()]);
  } catch (err) {
    showError('Nu s-a putut încărca lista de documente', err);
    documents = [];
    templates = [];
  }
  currentTemplates = templates;

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

  // With one template there is nothing to choose, so the button goes straight
  // to it and no menu is drawn at all — an install that never adds a second
  // template behaves exactly as it did before templates existed.
  const menu =
    templates.length < 2
      ? ''
      : `<ul class="template-menu" id="template-menu" hidden>
          ${templates
            .map(
              (t) => `
            <li>
              <a class="template-link" href="${draftHash(t.id)}">${escapeHtml(t.nume)}</a>
            </li>`,
            )
            .join('')}
        </ul>`;

  // The draft entry is rendered on every pass and shown or hidden by
  // markActive, so moving in and out of #/document/new only toggles an
  // attribute instead of rebuilding a sidebar whose document list costs a
  // round trip to SQLite. It sits above the saved documents because the list
  // is newest-first and the unsaved one is newer than all of them.
  el.innerHTML = `
    <div class="new-doc-wrap">
      <button class="btn btn-primary" id="new-doc">+ Document nou</button>
      ${menu}
    </div>
    <ul class="doc-list">
      <li id="draft-item" hidden>
        <a class="doc-link" id="draft-link" href="${DRAFT_PREFIX}">
          <span class="doc-nr">Document nou</span>
          <span class="doc-meta draft-meta">Nesalvat</span>
        </a>
      </li>
      ${items}
    </ul>
    <a class="settings-link" href="#/setari">Setări</a>
  `;

  const menuEl = el.querySelector<HTMLUListElement>('#template-menu');
  el.querySelector<HTMLButtonElement>('#new-doc')!.addEventListener('click', () => {
    if (menuEl === null) {
      // One template, or none loaded: go where there is only one place to go.
      if (templates.length === 1) {
        navigate(draftHash(templates[0].id));
        return;
      }
      // No templates loaded at all: reachable only after ListDocuments or
      // ListTemplates threw above, so the user has already seen an error.
      // Doing nothing here would leave the button looking broken on top of
      // that; say so instead.
      void showAlert('Niciun șablon disponibil. Reîncărcați aplicația și încercați din nou.');
      return;
    }
    menuEl.toggleAttribute('hidden');
  });

  // Picking a template closes the menu; the href does the navigating.
  menuEl?.querySelectorAll('a').forEach((link) => {
    link.addEventListener('click', () => menuEl.setAttribute('hidden', ''));
  });

  markActive(el, templates);

  // A click anywhere else closes the menu, so it does not sit open over the
  // history. This listener lives on `document` and is registered once, like
  // the `hashchange` listener below — `el` itself is the long-lived sidebar
  // element (only its contents are replaced each render), so it is safe to
  // close over, but the menu is looked up fresh on every click rather than
  // captured, since the `<ul>` node from an earlier render is discarded by
  // the next `el.innerHTML` assignment.
  if (!documentClickListenerRegistered) {
    documentClickListenerRegistered = true;
    document.addEventListener('click', (event) => {
      const openMenu = el.querySelector<HTMLUListElement>('#template-menu');
      if (openMenu === null || openMenu.hasAttribute('hidden')) return;
      if (!el.querySelector('.new-doc-wrap')!.contains(event.target as Node)) {
        openMenu.setAttribute('hidden', '');
      }
    });
  }

  if (!hashchangeListenerRegistered) {
    hashchangeListenerRegistered = true;
    window.addEventListener('hashchange', () => markActive(el, currentTemplates));
  }
}

function markActive(el: HTMLElement, templates: Template[]): void {
  const hash = currentHash();
  const draftItem = el.querySelector('#draft-item');
  draftItem?.toggleAttribute('hidden', !isDraftHash(hash));

  // The draft entry points at whatever draft is open, and says which template
  // it came from, so two unsaved documents are never confusable.
  const draftLink = el.querySelector<HTMLAnchorElement>('#draft-link');
  if (draftLink !== null && isDraftHash(hash)) {
    draftLink.setAttribute('href', hash);
    const templateId = draftTemplateId(hash);
    const template = templates.find((t) => t.id === templateId);
    const meta = draftLink.querySelector('.draft-meta');
    if (meta !== null) {
      meta.textContent = template === undefined ? 'Nesalvat' : `Nesalvat — ${template.nume}`;
    }
  }

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
