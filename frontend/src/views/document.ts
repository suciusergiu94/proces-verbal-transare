import {
  DeleteDocument,
  ExportPDF,
  GetDocument,
  GetSettings,
  NewDocumentDraft,
  SaveDocument,
  showError,
} from '../api';
import type { Document } from '../api';
import { diferenta, incarcaDescarca, totals, valoare } from '../calc';
import { formatNumber, parseNumber } from '../format';
import { navigate } from '../router';
import { escapeHtml } from '../sidebar';

// documentInputAbort holds the AbortController for the currently-attached
// `input` listener. renderDocumentView runs afresh on every hashchange *and*
// on every save (navigate() re-invokes the router even when the hash is
// unchanged), but `outlet` is a single long-lived element created once in
// main.ts. Without this, each run would attach another listener that is
// never removed — leaking handlers, and (worse) leaving a stale document
// handler attached when the user navigates to an unrelated view such as
// Setări, where it throws on the first keystroke.
let documentInputAbort: AbortController | undefined;

/** Renders the document form. Pass no id for a new document. */
export async function renderDocumentView(
  outlet: HTMLElement,
  id: string | undefined,
  refreshSidebar: () => Promise<void>,
): Promise<void> {
  let doc: Document;
  let unitate = '';

  // Sticky override flag: once true, nothing in this view rewrites the footer
  // (Diferență / Suma cu care se încarcă-descarcă) again for the life of the
  // view. Those two fields are pre-filled from a computation but are
  // deliberately editable — the paper form is hand-corrected there — so a
  // saved correction must survive every later edit, not just the initial
  // paint. Do not "simplify" this into a per-event check (e.g. "is this
  // input event's target a footer control?"): that only protects the footer
  // from the keystroke that lands directly on it, and any subsequent edit to
  // an unrelated field (Gestionar, a row...) would silently recompute and
  // overwrite the saved correction.
  let footerOverridden = false;

  try {
    const [loaded, settings] = await Promise.all([
      id === undefined ? NewDocumentDraft() : GetDocument(Number(id)),
      GetSettings(),
    ]);
    doc = loaded;
    unitate = settings.unitateNume;
  } catch (err) {
    showError('Nu s-a putut încărca documentul', err);
    outlet.innerHTML = '<p class="empty">Documentul nu a putut fi încărcat.</p>';
    return;
  }

  // A saved document whose stored footer values differ from what the current
  // rows would compute means the user hand-corrected them; start overridden
  // so that correction is never silently recomputed away. A brand-new draft
  // starts un-overridden, since its footer is legitimately still "pre-filled".
  footerOverridden = doc.id !== 0 && footerDiffersFromComputed();

  // Replace any listener left by a previous renderDocumentView invocation on
  // this same outlet (see the comment on documentInputAbort above).
  documentInputAbort?.abort();
  documentInputAbort = new AbortController();
  outlet.addEventListener('input', onInput, { signal: documentInputAbort.signal });

  // A saved document's stored footer values must survive a reload untouched; only a
  // brand-new draft should have its footer computed from (empty) rows on first render.
  renderAll(doc.id === 0);

  function footerDiffersFromComputed(): boolean {
    const intrare = totals(doc.intrare);
    const iesire = totals(doc.iesire);
    const dif = diferenta(iesire.valoareCuTva, intrare.valoareCuTva);
    const inc = incarcaDescarca(iesire.valoareCuTva, intrare.valoareCuTva);
    return (
      doc.diferentaTip !== dif.tip ||
      doc.diferentaValoare !== dif.valoare ||
      doc.incarcaDescarcaTip !== inc.tip ||
      doc.incarcaDescarcaValoare !== inc.valoare
    );
  }

  function renderAll(refreshFooter = true): void {
    outlet.innerHTML = `
      <h1>${doc.id === 0 ? 'Document nou' : `Proces verbal NR ${doc.nr}`}</h1>

      <div class="header-grid">
        <div class="field">
          <label for="f-unitate">Unitate</label>
          <input id="f-unitate" value="${escapeHtml(unitate)}" readonly />
        </div>
        <div class="field">
          <label for="f-gestiune">Gestiune</label>
          <input id="f-gestiune" value="${escapeHtml(doc.gestiune)}" />
        </div>
        <div class="field">
          <label for="f-nr">NR</label>
          <input id="f-nr" class="num" type="number" min="1" step="1" value="${doc.nr}" />
        </div>
        <div class="field">
          <label for="f-data">Data</label>
          <input id="f-data" type="date" value="${escapeHtml(doc.data)}" />
        </div>
        <div class="field">
          <label for="f-referinta">Document referință</label>
          <input id="f-referinta" value="${escapeHtml(doc.documentReferinta)}" />
        </div>
      </div>

      <h2>Ce intră</h2>
      ${intrareTable()}
      <div class="table-actions">
        <button class="btn" id="add-intrare">+ Adaugă rând</button>
      </div>

      <h2>Ce iese</h2>
      ${iesireTable()}

      <div class="footer-grid">
        <div class="field">
          <label for="f-dif-tip">Diferență</label>
          <div class="inline-field">
            <select id="f-dif-tip">
              <option value=""></option>
              <option value="plus">plus</option>
              <option value="minus">minus</option>
            </select>
            <input id="f-dif-val" class="num" value="${formatNumber(doc.diferentaValoare)}" />
          </div>
        </div>
        <div class="field">
          <label for="f-id-tip">Suma cu care se încarcă/descarcă gestiunea</label>
          <div class="inline-field">
            <select id="f-id-tip">
              <option value=""></option>
              <option value="incarca">încarcă</option>
              <option value="descarca">descarcă</option>
            </select>
            <input id="f-id-val" class="num" value="${formatNumber(doc.incarcaDescarcaValoare)}" />
          </div>
        </div>
        <div class="field">
          <label for="f-gestionar">Gestionar</label>
          <input id="f-gestionar" value="${escapeHtml(doc.gestionar)}" />
        </div>
        <div class="field">
          <label for="f-calculator">Calculator</label>
          <input id="f-calculator" value="${escapeHtml(doc.calculator)}" />
        </div>
        <div class="field">
          <label for="f-vizat">Vizat compartiment producție</label>
          <input id="f-vizat" value="${escapeHtml(doc.vizatCompartimentProductie)}" />
        </div>
      </div>

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
        <button class="btn" id="print" ${doc.id === 0 ? 'disabled title="Salvează documentul întâi"' : ''}>Printează (PDF)</button>
        ${doc.id === 0 ? '' : '<button class="btn btn-danger" id="delete">Șterge</button>'}
      </div>
    `;

    (outlet.querySelector('#f-dif-tip') as HTMLSelectElement).value = doc.diferentaTip;
    (outlet.querySelector('#f-id-tip') as HTMLSelectElement).value = doc.incarcaDescarcaTip;

    wireEvents();
    recompute(refreshFooter);
  }

  function intrareTable(): string {
    const rows = doc.intrare
      .map(
        (row, i) => `
        <tr data-table="intrare" data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(row.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(row.um)}" /></td>
          <td class="num"><input class="num" data-field="cantitate" value="${row.cantitate || ''}" /></td>
          <td class="num"><input class="num" data-field="pretFaraTva" value="${row.pretFaraTva || ''}" /></td>
          <td class="num" data-out="valoareFaraTva"></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${row.pretCuTva || ''}" /></td>
          <td class="num" data-out="valoareCuTva"></td>
          <td><button class="btn-icon" data-remove="${i}" title="Șterge rândul">✕</button></td>
        </tr>`,
      )
      .join('');

    return `
      <table id="t-intrare">
        <thead>
          <tr>
            <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
            <th class="num">Cantitate</th><th class="num">Preț fără TVA</th><th class="num">Valoare</th>
            <th class="num">Preț cu TVA</th><th class="num">Valoare</th><th></th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
        <tfoot>
          <tr>
            <td colspan="3">Total</td>
            <td class="num" data-total="intrare-cantitate"></td>
            <td></td>
            <td class="num" data-total="intrare-faraTva"></td>
            <td></td>
            <td class="num" data-total="intrare-cuTva"></td>
            <td></td>
          </tr>
        </tfoot>
      </table>`;
  }

  function iesireTable(): string {
    const rows = doc.iesire
      .map(
        (row, i) => `
        <tr data-table="iesire" data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(row.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(row.um)}" /></td>
          <td class="num"><input class="num" data-field="cantitate" value="${row.cantitate || ''}" /></td>
          <td class="num"><input class="num" data-field="pretFaraTva" value="${row.pretFaraTva || ''}" /></td>
          <td class="num" data-out="valoareFaraTva"></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${row.pretCuTva || ''}" /></td>
          <td class="num" data-out="valoareCuTva"></td>
        </tr>`,
      )
      .join('');

    return `
      <table id="t-iesire">
        <thead>
          <tr>
            <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
            <th class="num">Cantitate</th><th class="num">Preț fără TVA</th><th class="num">Valoare</th>
            <th class="num">Preț cu TVA</th><th class="num">Valoare</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
        <tfoot>
          <tr>
            <td colspan="3">Total</td>
            <td class="num" data-total="iesire-cantitate"></td>
            <td></td>
            <td class="num" data-total="iesire-faraTva"></td>
            <td></td>
            <td class="num" data-total="iesire-cuTva"></td>
          </tr>
        </tfoot>
      </table>`;
  }

  function wireEvents(): void {
    outlet.querySelector('#add-intrare')!.addEventListener('click', () => {
      readForm();
      doc.intrare.push({
        id: 0,
        pozitie: doc.intrare.length,
        denumire: '',
        um: 'Kg',
        cantitate: 0,
        pretFaraTva: 0,
        pretCuTva: 0,
      });
      renderAll();
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-remove]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        doc.intrare.splice(Number(btn.dataset.remove), 1);
        renderAll();
      });
    });

    outlet.querySelector('#save')!.addEventListener('click', () => void onSave());

    const printBtn = outlet.querySelector('#print') as HTMLButtonElement | null;
    if (printBtn && !printBtn.disabled) {
      printBtn.addEventListener('click', () => void onPrint());
    }

    const deleteBtn = outlet.querySelector('#delete');
    if (deleteBtn) deleteBtn.addEventListener('click', () => void onDelete());
  }

  function onInput(event: Event): void {
    const target = event.target as HTMLElement;

    // Typing into any of the four footer controls marks the override sticky
    // (see the footerOverridden comment above) for the rest of the view's
    // lifetime, not just for this one event.
    if (isFooterControl(target)) {
      footerOverridden = true;
    }

    readForm();
    recompute(true);
  }

  function isFooterControl(el: HTMLElement): boolean {
    return (
      el.id === 'f-dif-val' || el.id === 'f-dif-tip' || el.id === 'f-id-val' || el.id === 'f-id-tip'
    );
  }

  /** Copies every input's current value back into doc. */
  function readForm(): void {
    doc.gestiune = value('#f-gestiune');
    doc.nr = Number(value('#f-nr')) || 0;
    doc.data = value('#f-data');
    doc.documentReferinta = value('#f-referinta');
    doc.diferentaTip = value('#f-dif-tip');
    doc.diferentaValoare = parseNumber(value('#f-dif-val'));
    doc.incarcaDescarcaTip = value('#f-id-tip');
    doc.incarcaDescarcaValoare = parseNumber(value('#f-id-val'));
    doc.gestionar = value('#f-gestionar');
    doc.calculator = value('#f-calculator');
    doc.vizatCompartimentProductie = value('#f-vizat');

    outlet.querySelectorAll<HTMLTableRowElement>('tr[data-table]').forEach((tr) => {
      const index = Number(tr.dataset.index);
      const row = tr.dataset.table === 'intrare' ? doc.intrare[index] : doc.iesire[index];
      if (!row) return;
      row.pozitie = index;
      row.denumire = fieldValue(tr, 'denumire');
      row.um = fieldValue(tr, 'um');
      row.cantitate = parseNumber(fieldValue(tr, 'cantitate'));
      row.pretFaraTva = parseNumber(fieldValue(tr, 'pretFaraTva'));
      row.pretCuTva = parseNumber(fieldValue(tr, 'pretCuTva'));
    });
  }

  /** Refreshes the computed cells, and the footer unless it has been overridden. */
  function recompute(refreshFooter = true): void {
    outlet.querySelectorAll<HTMLTableRowElement>('tr[data-table]').forEach((tr) => {
      const index = Number(tr.dataset.index);
      const row = tr.dataset.table === 'intrare' ? doc.intrare[index] : doc.iesire[index];
      if (!row) return;
      setOut(tr, 'valoareFaraTva', valoare(row.cantitate, row.pretFaraTva));
      setOut(tr, 'valoareCuTva', valoare(row.cantitate, row.pretCuTva));
    });

    const intrare = totals(doc.intrare);
    const iesire = totals(doc.iesire);
    setTotal('intrare-cantitate', intrare.cantitate);
    setTotal('intrare-faraTva', intrare.valoareFaraTva);
    setTotal('intrare-cuTva', intrare.valoareCuTva);
    setTotal('iesire-cantitate', iesire.cantitate);
    setTotal('iesire-faraTva', iesire.valoareFaraTva);
    setTotal('iesire-cuTva', iesire.valoareCuTva);

    // footerOverridden is checked here (not just at the onInput call site) so
    // that every caller of recompute — the initial render, add/remove row,
    // and onInput — is covered by one gate instead of each needing to know
    // about the override.
    if (!refreshFooter || footerOverridden) return;

    const dif = diferenta(iesire.valoareCuTva, intrare.valoareCuTva);
    const inc = incarcaDescarca(iesire.valoareCuTva, intrare.valoareCuTva);
    doc.diferentaTip = dif.tip;
    doc.diferentaValoare = dif.valoare;
    doc.incarcaDescarcaTip = inc.tip;
    doc.incarcaDescarcaValoare = inc.valoare;
    (outlet.querySelector('#f-dif-tip') as HTMLSelectElement).value = dif.tip;
    (outlet.querySelector('#f-dif-val') as HTMLInputElement).value = formatNumber(dif.valoare);
    (outlet.querySelector('#f-id-tip') as HTMLSelectElement).value = inc.tip;
    (outlet.querySelector('#f-id-val') as HTMLInputElement).value = formatNumber(inc.valoare);
  }

  /**
   * Validates and saves the current form, updating `doc` to the stored
   * version on success. Shared by onSave and onPrint so that printing always
   * exports what is on screen rather than whatever was last persisted (see
   * onPrint).
   */
  async function saveCurrentForm(): Promise<boolean> {
    readForm();
    if (doc.nr <= 0) {
      window.alert('Completați numărul documentului (NR).');
      return false;
    }
    if (!doc.data) {
      window.alert('Completați data documentului.');
      return false;
    }
    try {
      doc = await SaveDocument(doc);
      await refreshSidebar();
      return true;
    } catch (err) {
      showError('Documentul nu a putut fi salvat', err);
      return false;
    }
  }

  async function onSave(): Promise<void> {
    if (await saveCurrentForm()) {
      navigate(`#/document/${doc.id}`);
    }
  }

  // ExportPDF re-reads the document from SQLite by id, so printing without
  // saving first would silently export stale data for any edit made since
  // the last save. Rather than warn and make the user click twice, we save
  // automatically before exporting — the print button is only enabled once
  // the document already exists (doc.id !== 0), so this is always an update
  // to an already-saved document, never a surprise first save. This keeps
  // "Printează" a single click, which matters for a form whose whole purpose
  // is to be printed and signed.
  async function onPrint(): Promise<void> {
    if (!(await saveCurrentForm())) return;
    try {
      const path = await ExportPDF(doc.id);
      if (path === '') return; // dialog cancelled
    } catch (err) {
      showError('PDF-ul nu a putut fi generat', err);
    }
  }

  async function onDelete(): Promise<void> {
    if (!window.confirm(`Ștergeți documentul NR ${doc.nr}?`)) return;
    try {
      await DeleteDocument(doc.id);
      await refreshSidebar();
      navigate('#/document/new');
    } catch (err) {
      showError('Documentul nu a putut fi șters', err);
    }
  }

  function value(selector: string): string {
    // Null-safe as cheap insurance: with the AbortController fix above this
    // view's own elements should always be present when its own listener
    // fires, but a stale/mismatched handler must degrade to '' rather than
    // throw and break an unrelated view.
    const el = outlet.querySelector(selector) as HTMLInputElement | HTMLSelectElement | null;
    return el ? el.value : '';
  }

  function fieldValue(tr: HTMLTableRowElement, field: string): string {
    const input = tr.querySelector<HTMLInputElement>(`[data-field="${field}"]`);
    return input ? input.value : '';
  }

  function setOut(tr: HTMLTableRowElement, name: string, amount: number): void {
    const cell = tr.querySelector<HTMLTableCellElement>(`[data-out="${name}"]`);
    if (cell) cell.textContent = amount === 0 ? '' : formatNumber(amount);
  }

  function setTotal(name: string, amount: number): void {
    const cell = outlet.querySelector<HTMLTableCellElement>(`[data-total="${name}"]`);
    if (cell) cell.textContent = formatNumber(amount);
  }
}
