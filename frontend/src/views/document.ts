import {
  DeleteDocument,
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

/** Renders the document form. Pass no id for a new document. */
export async function renderDocumentView(
  outlet: HTMLElement,
  id: string | undefined,
  refreshSidebar: () => Promise<void>,
): Promise<void> {
  let doc: Document;
  let unitate = '';

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

  // Attached once per view (not per renderAll), since outlet survives innerHTML rewrites.
  outlet.addEventListener('input', onInput);

  // A saved document's stored footer values must survive a reload untouched; only a
  // brand-new draft should have its footer computed from (empty) rows on first render.
  renderAll(doc.id === 0);

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
          <label for="f-id-tip">Suma cu care se … gestiunea</label>
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

    const deleteBtn = outlet.querySelector('#delete');
    if (deleteBtn) deleteBtn.addEventListener('click', () => void onDelete());
  }

  function onInput(event: Event): void {
    const target = event.target as HTMLElement;
    readForm();

    // Retyping either total means the user is overriding the computed value, so
    // only refresh the footer when the edit came from somewhere else.
    const isFooterOverride =
      target.id === 'f-dif-val' ||
      target.id === 'f-dif-tip' ||
      target.id === 'f-id-val' ||
      target.id === 'f-id-tip';
    recompute(!isFooterOverride);
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

  /** Refreshes the computed cells, and the footer unless the user overrode it. */
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

    if (!refreshFooter) return;

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

  async function onSave(): Promise<void> {
    readForm();
    if (doc.nr <= 0) {
      window.alert('Completați numărul documentului (NR).');
      return;
    }
    if (!doc.data) {
      window.alert('Completați data documentului.');
      return;
    }
    try {
      const saved = await SaveDocument(doc);
      await refreshSidebar();
      navigate(`#/document/${saved.id}`);
    } catch (err) {
      showError('Documentul nu a putut fi salvat', err);
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
    return (outlet.querySelector(selector) as HTMLInputElement | HTMLSelectElement).value;
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
