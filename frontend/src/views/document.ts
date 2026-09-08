import {
  DeleteDocument,
  ExportPDF,
  GetDocument,
  GetSettings,
  ListTemplates,
  NewDocumentDraft,
  SaveDocument,
  SaveTemplates,
  showError,
} from '../api';
import type { Document, Template } from '../api';
import {
  ajusteazaMarja,
  cantitatiDinProcente,
  diferenta,
  incarcaDescarca,
  marjaProfit,
  pretCuTvaDin,
  pretFaraTvaDin,
  totals,
  valoare,
} from '../calc';
import { showConfirm } from '../dialog';
import { formatDateRO, formatNumber, parseDateRO, parseNumber } from '../format';
import { navigate } from '../router';
import { showToast } from '../toast';
import { escapeHtml } from '../sidebar';
import { DRAFT_PREFIX, draftHash, templatesCuProcenteNoi } from '../templates';

// documentInputAbort holds the AbortController for the currently-attached
// `input` listener. renderDocumentView runs afresh on every hashchange *and*
// on every save (navigate() re-invokes the router even when the hash is
// unchanged), but `outlet` is a single long-lived element created once in
// main.ts. Without this, each run would attach another listener that is
// never removed — leaking handlers, and (worse) leaving a stale document
// handler attached when the user navigates to an unrelated view such as
// Setări, where it throws on the first keystroke.
let documentInputAbort: AbortController | undefined;

// How much one click of the +/- buttons beside "Marja de profit" is worth, in
// percentage points. Small on purpose: the buttons nudge the split between
// products, they do not redesign the butchering.
const PAS_MARJA = 0.25;

/** Renders the document form. Pass no id for a new document. */
export async function renderDocumentView(
  outlet: HTMLElement,
  id: string | undefined,
  refreshSidebar: () => Promise<void>,
  templateId: number | undefined,
): Promise<void> {
  let doc: Document;
  let unitate = '';
  // The rate new rows start from. Rows already on the document keep their own.
  let cotaImplicita = 0;
  // The template this document was made from, and its name for the header. A
  // document whose template has since been deleted has neither.
  let templates: Template[] = [];
  let templateNume = '';
  // Each product's share of the carcass, by product id. The "ce iese"
  // quantities are filled from these whenever the input quantity changes. It
  // is empty for a document whose template is gone, which is what turns the
  // automatic fill off.
  let procente = new Map<number, number>();
  // Set once the +/- buttons have actually moved the split, which is what puts
  // "Salvează procentele noi" on screen. Nothing else offers to rewrite the
  // stored ratios, so the button stays out of the way until the user has a
  // split of their own to keep.
  let marjaNudged = false;

  try {
    const [settings, loadedTemplates] = await Promise.all([GetSettings(), ListTemplates()]);
    unitate = settings.unitateNume;
    cotaImplicita = settings.cotaTva;
    templates = loadedTemplates;

    if (id !== undefined) {
      doc = await GetDocument(Number(id));
    } else {
      // A draft route with no template on it is a stale hash or the fresh-launch
      // fallback. There is always at least one template, so send it to the
      // first rather than failing at the user.
      const wanted = templateId ?? templates[0]?.id;
      if (wanted === undefined) {
        outlet.innerHTML = '<p class="empty">Nu există niciun șablon. Creați unul din Setări.</p>';
        return;
      }
      if (templateId === undefined) {
        navigate(draftHash(wanted));
        return;
      }
      doc = await NewDocumentDraft(wanted);
    }

    const template = templates.find((t) => t.id === doc.templateId);
    if (template !== undefined) {
      templateNume = template.nume;
      procente = new Map(template.products.map((p) => [p.id, p.procentDinIntrare]));
    }
  } catch (err) {
    showError('Nu s-a putut încărca documentul', err);
    outlet.innerHTML = '<p class="empty">Documentul nu a putut fi încărcat.</p>';
    return;
  }

  // Replace any listener left by a previous renderDocumentView invocation on
  // this same outlet (see the comment on documentInputAbort above).
  documentInputAbort?.abort();
  documentInputAbort = new AbortController();
  outlet.addEventListener('input', onInput, { signal: documentInputAbort.signal });

  // No ratios are applied up front: a new document carries the previous one's
  // "ce intră" rows but not their quantities, so there is nothing to split
  // until the user weighs the carcass. Typing that quantity fills the whole
  // "ce iese" table from the ratios (see onInput).
  renderAll();

  function renderAll(): void {
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
          <label for="f-sablon">Șablon</label>
          <input id="f-sablon" value="${escapeHtml(templateNume || 'șters')}" readonly />
        </div>
        <div class="field">
          <label for="f-nr">NR</label>
          <input id="f-nr" class="num" type="number" min="1" step="1" value="${doc.nr}" />
        </div>
        <div class="field">
          <label for="f-data">Data</label>
          <input
            id="f-data"
            type="text"
            inputmode="numeric"
            maxlength="10"
            placeholder="ZZ/LL/AAAA"
            value="${escapeHtml(formatDateRO(doc.data))}"
          />
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
      ${
        procente.size === 0
          ? `<p class="empty">Șablonul acestui document a fost șters; cantitățile nu se
             mai completează automat din cantitatea de la "ce intră".</p>`
          : ''
      }
      ${procente.size === 0 ? '' : marjaRow()}
      ${iesireTable()}

      <div class="footer-grid footer-computed">
        <div class="field">
          <label for="f-dif-val">Diferență</label>
          <div class="inline-field">
            <input id="f-dif-tip" class="tip" readonly tabindex="-1" />
            <input id="f-dif-val" class="num" readonly tabindex="-1" />
          </div>
        </div>
        <div class="field">
          <label for="f-id-val">Suma cu care se încarcă/descarcă gestiunea</label>
          <div class="inline-field">
            <input id="f-id-tip" class="tip" readonly tabindex="-1" />
            <input id="f-id-val" class="num" readonly tabindex="-1" />
          </div>
        </div>
      </div>

      <div class="footer-grid footer-semnaturi">
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
        ${doc.id === 0 ? '' : '<button class="btn" id="print">Printează (PDF)</button>'}
        ${doc.id === 0 ? '' : '<button class="btn btn-danger" id="delete">Șterge</button>'}
      </div>
    `;

    wireEvents();
    recompute();
  }

  /** The margin controls: nudging the split, and saving it back to the template. */
  function marjaRow(): string {
    return `
      <div class="marja">
        <label for="f-marja">Marja de profit</label>
        <input id="f-marja" class="num" readonly tabindex="-1" />
        <div class="marja-pas">
          <button class="btn-icon" id="marja-minus" type="button" title="Scade marja cu ${PAS_MARJA} % redistribuind cantitățile">−</button>
          <button class="btn-icon" id="marja-plus" type="button" title="Crește marja cu ${PAS_MARJA} % redistribuind cantitățile">+</button>
        </div>
        <button
          class="btn marja-salveaza"
          id="save-procente"
          type="button"
          ${marjaNudged ? '' : 'hidden'}
        >Salvează procentele noi</button>
      </div>`;
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
          <td class="num cota"><input class="num" data-field="cotaTva" value="${row.cotaTva}" /></td>
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
            <th class="num cota">TVA %</th><th class="num">Preț cu TVA</th><th class="num">Valoare</th><th></th>
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
          <td class="num cota"><input class="num" data-field="cotaTva" value="${row.cotaTva}" /></td>
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
            <th class="num cota">TVA %</th><th class="num">Preț cu TVA</th><th class="num">Valoare</th>
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
        cotaTva: cotaImplicita,
      });
      renderAll();
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-remove]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        doc.intrare.splice(Number(btn.dataset.remove), 1);
        // One fewer input row is less carcass, so the yields follow.
        aplicaProcente();
        renderAll();
      });
    });

    const marjaPlus = outlet.querySelector('#marja-plus');
    if (marjaPlus) marjaPlus.addEventListener('click', () => stepMarja(PAS_MARJA));
    const marjaMinus = outlet.querySelector('#marja-minus');
    if (marjaMinus) marjaMinus.addEventListener('click', () => stepMarja(-PAS_MARJA));

    // Tidies "3.9.2026" into "03/09/2026" once the user leaves the field, so
    // the form always shows the date in the shape the printed document uses.
    const dataInput = outlet.querySelector<HTMLInputElement>('#f-data');
    dataInput?.addEventListener('blur', () => {
      const iso = parseDateRO(dataInput.value);
      if (iso) dataInput.value = formatDateRO(iso);
    });

    outlet.querySelector('#save')!.addEventListener('click', () => void onSave());

    const printBtn = outlet.querySelector('#print');
    if (printBtn) printBtn.addEventListener('click', () => void onPrint());

    const saveProcenteBtn = outlet.querySelector('#save-procente');
    if (saveProcenteBtn) saveProcenteBtn.addEventListener('click', () => void onSaveProcente());

    const deleteBtn = outlet.querySelector('#delete');
    if (deleteBtn) deleteBtn.addEventListener('click', () => void onDelete());
  }

  /**
   * Nudges the margin by `delta` percentage points by redistributing the "ce
   * iese" quantities — see ajusteazaMarja for how the shift is spread. The
   * total quantity is unchanged; only the split between products moves.
   *
   * The quantity inputs are written in place rather than through renderAll(),
   * so the user keeps their scroll position and can click the button several
   * times in a row while watching the same rows change.
   */
  function stepMarja(delta: number): void {
    readForm();
    const propunere = ajusteazaMarja(doc.iesire, totals(doc.intrare).valoareCuTva, delta);
    if (propunere === undefined) {
      showToast('Marja nu poate fi ajustată din cantități.');
      return;
    }
    propunere.forEach((cantitate, i) => {
      doc.iesire[i].cantitate = cantitate;
      const tr = outlet.querySelector<HTMLTableRowElement>(
        `tr[data-table="iesire"][data-index="${i}"]`,
      );
      if (tr) setRowField(tr, 'cantitate', cantitate);
    });
    marjaNudged = true;
    syncSalvareProcente();
    recompute();
  }

  /** Puts the "save the new ratios" button in step with marjaNudged. */
  function syncSalvareProcente(): void {
    outlet.querySelector('#save-procente')?.toggleAttribute('hidden', !marjaNudged);
  }

  /**
   * Stores the split the user nudged into as this document's template's carcass
   * ratios, so every later document from that template opens on it — the same
   * column Setări shows and edits by hand. Only this document's template is
   * touched; the others go back exactly as they came.
   */
  async function onSaveProcente(): Promise<void> {
    readForm();
    if (doc.templateId === undefined || doc.templateId === null) return;

    let stored: Template[];
    try {
      stored = await ListTemplates();
    } catch (err) {
      showError('Lista de șabloane nu a putut fi citită', err);
      return;
    }

    const cantitatiPerProdus = new Map<number, number>();
    doc.iesire.forEach((row) => {
      if (row.productId == null) return;
      cantitatiPerProdus.set(
        row.productId,
        (cantitatiPerProdus.get(row.productId) ?? 0) + row.cantitate,
      );
    });

    const actualizate = templatesCuProcenteNoi(stored, doc.templateId, cantitatiPerProdus);
    if (actualizate === undefined) {
      showToast('Nu există cantități din care să se calculeze procentele.');
      return;
    }

    const nume = stored.find((t) => t.id === doc.templateId)?.nume ?? '';
    if (
      !(await showConfirm(
        `Salvați procentele noi? Procentele șablonului „${nume}” din Setări vor fi ` +
          'înlocuite cu cele rezultate din cantitățile de pe acest document.',
      ))
    ) {
      return;
    }

    try {
      await SaveTemplates(actualizate);
      // The form fills "ce iese" from these whenever the carcass weight
      // changes, so the in-memory copy has to move with the stored one or the
      // next keystroke would undo what was just saved.
      const salvat = actualizate.find((t) => t.id === doc.templateId);
      procente = new Map((salvat?.products ?? []).map((p) => [p.id, p.procentDinIntrare]));
      marjaNudged = false;
      syncSalvareProcente();
      showToast('Procentele au fost salvate!');
    } catch (err) {
      showError('Procentele nu au putut fi salvate', err);
    }
  }

  function onInput(event: Event): void {
    readForm();
    const target = event.target;
    if (target instanceof HTMLInputElement) {
      const tr = target.closest<HTMLTableRowElement>('tr[data-table]');
      if (tr && target.dataset.field) syncPrices(tr, target.dataset.field);
      // What goes in decides what comes out, so editing an input quantity
      // refills the whole "ce iese" table from the ratios.
      if (tr?.dataset.table === 'intrare' && target.dataset.field === 'cantitate') {
        aplicaProcente();
        writeCantitatiIesire();
      }
    }
    recompute();
  }

  /**
   * Fills every "ce iese" quantity from its product's share of what went in.
   *
   * This overwrites quantities the user has typed, and undoes a margin nudge:
   * the carcass is what the butchering starts from, so a correction to it
   * settles the table. Rows whose product has since been deleted from Setări
   * have no ratio to go on and keep the quantity they already carry.
   *
   * Writes onto doc only. The inputs are refreshed separately, because on the
   * first run the table has not been rendered yet.
   */
  function aplicaProcente(): void {
    const indici: number[] = [];
    const cote: number[] = [];
    doc.iesire.forEach((row, i) => {
      const procent = row.productId == null ? undefined : procente.get(row.productId);
      if (procent === undefined) return;
      indici.push(i);
      cote.push(procent);
    });
    if (indici.length === 0) return;

    const cantitati = cantitatiDinProcente(cote, totals(doc.intrare).cantitate);
    indici.forEach((i, n) => {
      doc.iesire[i].cantitate = cantitati[n];
    });
  }

  /**
   * Copies the "ce iese" quantities from doc back into their inputs, in place
   * rather than through renderAll(), so the user keeps their caret and scroll
   * position while typing the carcass weight and watching the table fill in.
   */
  function writeCantitatiIesire(): void {
    doc.iesire.forEach((row, i) => {
      const tr = outlet.querySelector<HTMLTableRowElement>(
        `tr[data-table="iesire"][data-index="${i}"]`,
      );
      if (tr) setRowField(tr, 'cantitate', row.cantitate);
    });
  }

  /**
   * Keeps a row's two prices in step through its TVA rate: typing one price
   * fills in the other. Changing the rate re-derives the price without TVA,
   * because the price with TVA is the anchor — it is what the product list
   * stores and what the shop actually charges.
   *
   * Only the *other* input is written, never the one being typed in, so the
   * caret never jumps and assigning .value raises no further `input` event.
   * A rate with no usable inverse (-100% or below) leaves both prices alone
   * rather than writing a zero the user did not ask for.
   */
  function syncPrices(tr: HTMLTableRowElement, changed: string): void {
    const index = Number(tr.dataset.index);
    const row = tr.dataset.table === 'intrare' ? doc.intrare[index] : doc.iesire[index];
    if (!row) return;

    if (changed === 'pretFaraTva') {
      const derived = pretCuTvaDin(row.pretFaraTva, row.cotaTva);
      if (derived === undefined) return;
      row.pretCuTva = derived;
      setRowField(tr, 'pretCuTva', derived);
      return;
    }
    if (changed === 'pretCuTva' || changed === 'cotaTva') {
      const derived = pretFaraTvaDin(row.pretCuTva, row.cotaTva);
      if (derived === undefined) return;
      row.pretFaraTva = derived;
      setRowField(tr, 'pretFaraTva', derived);
    }
  }

  /** Copies every input's current value back into doc. */
  function readForm(): void {
    doc.gestiune = value('#f-gestiune');
    doc.nr = Number(value('#f-nr')) || 0;
    // An unparseable date (blank, or half-typed) lands as an empty string;
    // saveCurrentForm is what tells the two apart for the error message.
    doc.data = parseDateRO(value('#f-data')) ?? '';
    doc.documentReferinta = value('#f-referinta');
    // Diferență and "Suma cu care se încarcă/descarcă" are derived, never
    // typed: recompute() is what writes them onto doc (see recompute).
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
      row.cotaTva = parseNumber(fieldValue(tr, 'cotaTva'));
    });
  }

  /** Refreshes every derived cell: row values, table totals and the footer. */
  function recompute(): void {
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

    // Adaos pe cost: what the butchering gained, as a percentage of what it
    // consumed. Blank while nothing has gone in yet, rather than a division
    // by zero (see marjaProfit).
    const marja = marjaProfit(iesire.valoareCuTva, intrare.valoareCuTva);
    setField('#f-marja', marja === undefined ? '' : `${formatNumber(marja)} %`);

    // The footer follows the two tables: a butchering that yields more than
    // it consumed is a "plus" that loads (încarcă) the gestiune, the reverse
    // is a "minus" that unloads (descarcă) it. Both the sign and the amount
    // are derived here — they are never entered by hand — so a saved
    // document's footer always agrees with the rows it was saved with, and
    // with what internal/calc re-derives for the PDF.
    const dif = diferenta(iesire.valoareCuTva, intrare.valoareCuTva);
    const inc = incarcaDescarca(iesire.valoareCuTva, intrare.valoareCuTva);
    doc.diferentaTip = dif.tip;
    doc.diferentaValoare = dif.valoare;
    doc.incarcaDescarcaTip = inc.tip;
    doc.incarcaDescarcaValoare = inc.valoare;
    setField('#f-dif-tip', tipLabel(dif.tip));
    setField('#f-dif-val', formatNumber(dif.valoare));
    setField('#f-id-tip', tipLabel(inc.tip));
    setField('#f-id-val', formatNumber(inc.valoare));
  }

  /** The Romanian wording the printed form uses for a computed tip. */
  function tipLabel(tip: string): string {
    switch (tip) {
      case 'plus':
        return 'plus';
      case 'minus':
        return 'minus';
      case 'incarca':
        return 'încarcă';
      case 'descarca':
        return 'descarcă';
      default:
        return '';
    }
  }

  function setField(selector: string, text: string): void {
    const el = outlet.querySelector<HTMLInputElement>(selector);
    if (el) el.value = text;
  }

  /**
   * Validates and saves the current form, updating `doc` to the stored
   * version on success. Shared by onSave and onPrint so that printing always
   * exports what is on screen rather than whatever was last persisted (see
   * onPrint).
   */
  async function saveCurrentForm(): Promise<boolean> {
    readForm();
    // readForm no longer touches the derived footer, so refresh it here:
    // what gets persisted must match the rows being persisted with it.
    recompute();
    if (doc.nr <= 0) {
      window.alert('Completați numărul documentului (NR).');
      return false;
    }
    if (!doc.data) {
      window.alert(
        value('#f-data').trim() === ''
          ? 'Completați data documentului.'
          : 'Data documentului nu este validă. Folosiți formatul ZZ/LL/AAAA.',
      );
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
    if (!(await saveCurrentForm())) return;
    // navigate() re-runs the router and rebuilds this view from the stored
    // document, so the toast is raised after it — the toast lives on
    // document.body, outside the outlet that renderAll() overwrites.
    navigate(`#/document/${doc.id}`);
    showToast('Salvat!');
  }

  // ExportPDF re-reads the document from SQLite by id, so printing without
  // saving first would silently export stale data for any edit made since
  // the last save. Rather than warn and make the user click twice, we save
  // automatically before exporting — the print button is only rendered once
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
    if (!(await showConfirm('Sunteți sigur că doriți să ștergeți documentul?'))) return;
    try {
      await DeleteDocument(doc.id);
      await refreshSidebar();
      navigate(doc.templateId == null ? DRAFT_PREFIX : draftHash(doc.templateId));
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

  /**
   * Writes a derived price into a row input. Zero shows as blank, matching how
   * the table renders an empty price, so clearing one price clears the other.
   */
  function setRowField(tr: HTMLTableRowElement, field: string, amount: number): void {
    const input = tr.querySelector<HTMLInputElement>(`[data-field="${field}"]`);
    if (input) input.value = amount === 0 ? '' : String(amount);
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
