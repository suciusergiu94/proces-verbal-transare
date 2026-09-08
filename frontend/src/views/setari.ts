import { GetSettings, ListTemplates, SaveSettings, SaveTemplates, showError } from '../api';
import type { Product, Settings, Template } from '../api';
import { sumaProcente } from '../calc';
import { showAlert, showConfirm } from '../dialog';
import { formatNumber, parseNumber } from '../format';
import { escapeHtml } from '../sidebar';
import { duplicateTemplate, emptyTemplate, validateTemplates } from '../templates';

/** Renders the settings screen: global settings and one section per template. */
export async function renderSetariView(outlet: HTMLElement): Promise<void> {
  let settings: Settings;
  let templates: Template[];
  // Which sections are open. View state only, never persisted: a shop with
  // five templates should not open Setări on a wall of tables, so only the
  // first is expanded on load.
  let expanded = new Set<number>();

  try {
    [settings, templates] = await Promise.all([GetSettings(), ListTemplates()]);
  } catch (err) {
    showError('Nu s-au putut încărca setările', err);
    outlet.innerHTML = '<p class="empty">Setările nu au putut fi încărcate.</p>';
    return;
  }
  if (templates.length > 0) expanded.add(0);

  renderAll();

  /** One template: a header that is always visible and a table that folds. */
  function templateSection(t: Template, ti: number): string {
    const rows = t.products
      .map(
        (p, i) => `
        <tr data-template="${ti}" data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(p.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(p.um)}" /></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${formatNumber(p.pretCuTva)}" /></td>
          <td class="num"><input class="num" data-field="procentDinIntrare" value="${formatNumber(p.procentDinIntrare, 3)}" /></td>
          <td>
            <button class="btn-icon" data-up="${ti}:${i}" title="Mută în sus" ${i === 0 ? 'disabled' : ''}>↑</button>
            <button class="btn-icon" data-down="${ti}:${i}" title="Mută în jos" ${i === t.products.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="btn-icon" data-remove="${ti}:${i}" title="Șterge produsul">✕</button>
          </td>
        </tr>`,
      )
      .join('');

    return `
      <section class="template-section${expanded.has(ti) ? ' open' : ''}" data-template="${ti}">
        <div class="template-head">
          <button class="btn-icon template-toggle" data-toggle="${ti}"
                  title="${expanded.has(ti) ? 'Restrânge' : 'Extinde'}">${expanded.has(ti) ? '▾' : '▸'}</button>
          <input class="template-nume" data-template-nume="${ti}"
                 placeholder="Numele șablonului" value="${escapeHtml(t.nume)}" />
          <span class="template-total" data-total="${ti}"></span>
          <button class="btn" data-duplicate="${ti}">Duplică</button>
          ${templates.length > 1 ? `<button class="btn btn-danger" data-delete="${ti}">Șterge</button>` : ''}
        </div>
        <div class="template-body" ${expanded.has(ti) ? '' : 'hidden'}>
          <table>
            <thead>
              <tr>
                <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
                <th class="num">Preț cu TVA</th><th class="num">% din intrare</th><th></th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
          <div class="table-actions">
            <button class="btn" data-add-product="${ti}">+ Adaugă produs</button>
          </div>
        </div>
      </section>`;
  }

  function renderAll(): void {
    const sectiuni = templates.map((t, ti) => templateSection(t, ti)).join('');

    outlet.innerHTML = `
      <h1>Setări</h1>

      <div class="header-grid">
        <div class="field">
          <label for="s-unitate">Unitate</label>
          <input id="s-unitate" value="${escapeHtml(settings.unitateNume)}" />
        </div>
        <div class="field">
          <label for="s-gestiune">Gestiune implicită</label>
          <input id="s-gestiune" value="${escapeHtml(settings.gestiune)}" />
        </div>
        <div class="field">
          <label for="s-nextnr">Următorul NR</label>
          <input id="s-nextnr" class="num" type="number" min="1" step="1" value="${settings.nextNr}" />
        </div>
        <div class="field">
          <label for="s-cota">Cotă TVA implicită (%)</label>
          <input id="s-cota" class="num" value="${settings.cotaTva}" />
        </div>
      </div>

      <h2>Șabloane</h2>
      <p class="empty">
        Un șablon este o listă "ce iese" cu numele ei — de exemplu "Carcasa Porc"
        sau "Carcasa Vitel". Procentele spun cât din ce intră se transformă în
        fiecare produs, deci trebuie să însumeze exact 100% în fiecare șablon,
        inclusiv pierderile, trecute la deșeu.
      </p>

      <div id="templates">${sectiuni}</div>
      <div class="table-actions">
        <button class="btn" id="add-template">+ Șablon nou</button>
      </div>

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
      </div>
      <p class="empty" id="status"></p>
    `;

    wireEvents();
    refreshTotaluri();
  }

  /**
   * Shows what each template's ratio column accounts for, marked as wrong
   * unless it is exactly 100%. The figures are the ones validateTemplates
   * checks, so the screen can never disagree with the message the user gets
   * when saving is refused.
   */
  function refreshTotaluri(): void {
    templates.forEach((t, ti) => {
      const cell = outlet.querySelector<HTMLElement>(`[data-total="${ti}"]`);
      if (!cell) return;
      const total = sumaProcente(t.products.map((p) => p.procentDinIntrare));
      cell.textContent = `${formatNumber(total, 3)} %`;
      cell.classList.toggle('invalid', total !== 100);
    });
  }

  /** Copies every input's current value back into settings and templates. */
  function readForm(): void {
    settings.unitateNume = (outlet.querySelector('#s-unitate') as HTMLInputElement).value;
    settings.gestiune = (outlet.querySelector('#s-gestiune') as HTMLInputElement).value;
    // Number(...) || 1 alone would let a negative value (e.g. -3) through
    // unchanged, since -3 is truthy; only "" / NaN fall back to 1 that way.
    const rawNextNr = Number((outlet.querySelector('#s-nextnr') as HTMLInputElement).value);
    const truncatedNextNr = Math.trunc(rawNextNr);
    settings.nextNr = Number.isFinite(truncatedNextNr) && truncatedNextNr >= 1 ? truncatedNextNr : 1;
    settings.cotaTva = parseNumber((outlet.querySelector('#s-cota') as HTMLInputElement).value);

    outlet.querySelectorAll<HTMLInputElement>('[data-template-nume]').forEach((input) => {
      const t = templates[Number(input.dataset.templateNume)];
      if (t) t.nume = input.value;
    });

    outlet.querySelectorAll<HTMLTableRowElement>('tr[data-template][data-index]').forEach((tr) => {
      const t = templates[Number(tr.dataset.template)];
      const product = t?.products[Number(tr.dataset.index)];
      if (!product) return;
      product.denumire = tr.querySelector<HTMLInputElement>('[data-field="denumire"]')!.value;
      product.um = tr.querySelector<HTMLInputElement>('[data-field="um"]')!.value;
      product.pretCuTva = parseNumber(
        tr.querySelector<HTMLInputElement>('[data-field="pretCuTva"]')!.value,
      );
      product.procentDinIntrare = parseNumber(
        tr.querySelector<HTMLInputElement>('[data-field="procentDinIntrare"]')!.value,
      );
    });
  }

  function wireEvents(): void {
    // Keeps the totals in step as the user types, without re-rendering the
    // table under their caret.
    outlet.querySelectorAll<HTMLInputElement>('[data-field="procentDinIntrare"]').forEach((input) => {
      input.addEventListener('input', () => {
        readForm();
        refreshTotaluri();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-toggle]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const ti = Number(btn.dataset.toggle);
        if (expanded.has(ti)) expanded.delete(ti);
        else expanded.add(ti);
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-add-product]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const ti = Number(btn.dataset.addProduct);
        templates[ti].products.push({
          id: 0,
          templateId: templates[ti].id,
          denumire: '',
          um: 'Kg',
          pretCuTva: 0,
          procentDinIntrare: 0,
          ordine: templates[ti].products.length,
        } as Product);
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-up]').forEach((btn) => {
      btn.addEventListener('click', () => moveProduct(btn.dataset.up!, -1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-down]').forEach((btn) => {
      btn.addEventListener('click', () => moveProduct(btn.dataset.down!, 1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-remove]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const [ti, i] = btn.dataset.remove!.split(':').map(Number);
        templates[ti].products.splice(i, 1);
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-duplicate]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        const ti = Number(btn.dataset.duplicate);
        templates.splice(ti + 1, 0, duplicateTemplate(templates[ti]));
        // Every index past the insertion point has shifted, so the open set is
        // rebuilt rather than patched: the copy opens, and the rest follow the
        // sections they were attached to.
        expanded = new Set(
          [...expanded].map((n) => (n > ti ? n + 1 : n)).concat(ti + 1),
        );
        renderAll();
      });
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-delete]').forEach((btn) => {
      btn.addEventListener('click', () => void onDeleteTemplate(Number(btn.dataset.delete)));
    });

    outlet.querySelector('#add-template')!.addEventListener('click', () => {
      readForm();
      templates.push(emptyTemplate());
      expanded.add(templates.length - 1);
      renderAll();
      // A blank section is useless until it is named, so start there.
      outlet
        .querySelector<HTMLInputElement>(`[data-template-nume="${templates.length - 1}"]`)
        ?.focus();
    });

    outlet.querySelector('#save')!.addEventListener('click', () => void onSave());
  }

  function moveProduct(key: string, delta: number): void {
    const [ti, index] = key.split(':').map(Number);
    const products = templates[ti].products;
    const target = index + delta;
    if (target < 0 || target >= products.length) return;
    readForm();
    const [row] = products.splice(index, 1);
    products.splice(target, 0, row);
    renderAll();
  }

  /**
   * Removes a template from the list being edited. Nothing is written until
   * Salvează, so this is undone by leaving the page — which the confirmation
   * says, because deleting a profile someone spent an afternoon on is not a
   * click to take lightly.
   */
  async function onDeleteTemplate(ti: number): Promise<void> {
    readForm();
    const nume = templates[ti].nume.trim() || `Șablonul ${ti + 1}`;
    if (
      !(await showConfirm(
        `Ștergeți șablonul „${nume}”? Documentele făcute cu el rămân neatinse, ` +
          'dar nu vor mai putea completa cantitățile automat. Ștergerea are loc la Salvează.',
      ))
    ) {
      return;
    }
    templates.splice(ti, 1);
    expanded = new Set([...expanded].filter((n) => n !== ti).map((n) => (n > ti ? n - 1 : n)));
    renderAll();
  }

  async function onSave(): Promise<void> {
    readForm();
    if (settings.cotaTva < 0) {
      void showAlert('Cota TVA nu poate fi negativă.');
      return;
    }
    const problem = validateTemplates(templates);
    if (problem !== undefined) {
      // Open the offending section so the user is looking at what is wrong
      // when they read why the save was refused.
      expanded.add(problem.templateIndex);
      renderAll();
      outlet
        .querySelector(`.template-section[data-template="${problem.templateIndex}"]`)
        ?.scrollIntoView({ block: 'center' });
      void showAlert(problem.message);
      return;
    }
    try {
      await SaveSettings(settings);
      await SaveTemplates(templates);
      templates = await ListTemplates();
      renderAll();
      outlet.querySelector('#status')!.textContent = 'Setările au fost salvate.';
    } catch (err) {
      showError('Setările nu au putut fi salvate', err);
    }
  }
}
