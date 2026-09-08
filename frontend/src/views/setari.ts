import { GetSettings, ListProducts, SaveProducts, SaveSettings, showError } from '../api';
import type { Product, Settings } from '../api';
import { round3, sumaProcente } from '../calc';
import { formatNumber, parseNumber } from '../format';
import { escapeHtml } from '../sidebar';

/** Renders the settings screen: unit name and the "ce iese" product list. */
export async function renderSetariView(outlet: HTMLElement): Promise<void> {
  let settings: Settings;
  let products: Product[];

  try {
    [settings, products] = await Promise.all([GetSettings(), ListProducts()]);
  } catch (err) {
    showError('Nu s-au putut încărca setările', err);
    outlet.innerHTML = '<p class="empty">Setările nu au putut fi încărcate.</p>';
    return;
  }

  renderAll();

  function renderAll(): void {
    const rows = products
      .map(
        (p, i) => `
        <tr data-index="${i}">
          <td>${i + 1}</td>
          <td><input data-field="denumire" value="${escapeHtml(p.denumire)}" /></td>
          <td><input data-field="um" value="${escapeHtml(p.um)}" /></td>
          <td class="num"><input class="num" data-field="pretCuTva" value="${formatNumber(p.pretCuTva)}" /></td>
          <td class="num"><input class="num" data-field="procentDinIntrare" value="${formatNumber(p.procentDinIntrare, 3)}" /></td>
          <td>
            <button class="btn-icon" data-up="${i}" title="Mută în sus" ${i === 0 ? 'disabled' : ''}>↑</button>
            <button class="btn-icon" data-down="${i}" title="Mută în jos" ${i === products.length - 1 ? 'disabled' : ''}>↓</button>
            <button class="btn-icon" data-remove="${i}" title="Șterge produsul">✕</button>
          </td>
        </tr>`,
      )
      .join('');

    outlet.innerHTML = `
      <h1>Setări</h1>

      <div class="header-grid">
        <div class="field">
          <label for="s-unitate">Unitate</label>
          <input id="s-unitate" value="${escapeHtml(settings.unitateNume)}" />
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

      <h2>Produse (tabelul "ce iese")</h2>
      <p class="empty">
        Procentele spun cât din ce intră se transformă în fiecare produs. Formularul
        completează cantitățile din tabelul "ce iese" pe baza lor, deci trebuie să
        însumeze exact 100% — inclusiv pierderile, trecute la deșeu.
      </p>
      <table>
        <thead>
          <tr>
            <th>Nr. crt.</th><th>Denumire produs</th><th>U/M</th>
            <th class="num">Preț cu TVA</th><th class="num">% din intrare</th><th></th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
        <tfoot>
          <tr>
            <td colspan="4">Total</td>
            <td class="num" id="total-procent"></td>
            <td></td>
          </tr>
        </tfoot>
      </table>
      <div class="table-actions">
        <button class="btn" id="add-product">+ Adaugă produs</button>
      </div>

      <div class="btn-row">
        <button class="btn btn-primary" id="save">Salvează</button>
      </div>
      <p class="empty" id="status"></p>
    `;

    wireEvents();
    refreshTotalProcent();
  }

  /**
   * Shows what the ratio column currently accounts for, marked as wrong unless
   * it is exactly 100%. The figure is the one onSave checks, so the screen can
   * never disagree with the message the user gets when saving is refused.
   */
  function refreshTotalProcent(): void {
    const cell = outlet.querySelector<HTMLTableCellElement>('#total-procent');
    if (!cell) return;
    const total = sumaProcente(products.map((p) => p.procentDinIntrare));
    cell.textContent = `${formatNumber(total, 3)} %`;
    cell.classList.toggle('invalid', total !== 100);
  }

  function wireEvents(): void {
    // Keeps the total in step as the user types, without re-rendering the
    // table under their caret.
    outlet.querySelectorAll<HTMLInputElement>('[data-field="procentDinIntrare"]').forEach((input) => {
      input.addEventListener('input', () => {
        readForm();
        refreshTotalProcent();
      });
    });

    outlet.querySelector('#add-product')!.addEventListener('click', () => {
      readForm();
      products.push({
        id: 0,
        denumire: '',
        um: 'Kg',
        pretCuTva: 0,
        procentDinIntrare: 0,
        ordine: products.length,
      });
      renderAll();
    });

    outlet.querySelectorAll<HTMLButtonElement>('[data-up]').forEach((btn) => {
      btn.addEventListener('click', () => move(Number(btn.dataset.up), -1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-down]').forEach((btn) => {
      btn.addEventListener('click', () => move(Number(btn.dataset.down), 1));
    });
    outlet.querySelectorAll<HTMLButtonElement>('[data-remove]').forEach((btn) => {
      btn.addEventListener('click', () => {
        readForm();
        products.splice(Number(btn.dataset.remove), 1);
        renderAll();
      });
    });

    outlet.querySelector('#save')!.addEventListener('click', () => void onSave());
  }

  function move(index: number, delta: number): void {
    const target = index + delta;
    if (target < 0 || target >= products.length) return;
    readForm();
    const [row] = products.splice(index, 1);
    products.splice(target, 0, row);
    renderAll();
  }

  /** Copies every input's current value back into settings and products. */
  function readForm(): void {
    settings.unitateNume = (outlet.querySelector('#s-unitate') as HTMLInputElement).value;
    // Number(...) || 1 alone would let a negative value (e.g. -3) through
    // unchanged, since -3 is truthy; only "" / NaN fall back to 1 that way.
    const rawNextNr = Number((outlet.querySelector('#s-nextnr') as HTMLInputElement).value);
    const truncatedNextNr = Math.trunc(rawNextNr);
    settings.nextNr = Number.isFinite(truncatedNextNr) && truncatedNextNr >= 1 ? truncatedNextNr : 1;
    settings.cotaTva = parseNumber((outlet.querySelector('#s-cota') as HTMLInputElement).value);

    outlet.querySelectorAll<HTMLTableRowElement>('tbody tr[data-index]').forEach((tr) => {
      const product = products[Number(tr.dataset.index)];
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

  async function onSave(): Promise<void> {
    readForm();
    if (products.some((p) => p.denumire.trim() === '')) {
      window.alert('Fiecare produs trebuie să aibă o denumire.');
      return;
    }
    if (settings.cotaTva < 0) {
      window.alert('Cota TVA nu poate fi negativă.');
      return;
    }
    if (products.some((p) => p.procentDinIntrare < 0)) {
      window.alert('Procentul din intrare nu poate fi negativ.');
      return;
    }
    // The form fills the "ce iese" quantities from these ratios, so a column
    // that does not account for the whole carcass would quietly produce a
    // document that is short. Saving is refused rather than normalised: the
    // figures are the user's to decide, and the gap is named so it can be put
    // where it belongs — usually on deșeu.
    const total = sumaProcente(products.map((p) => p.procentDinIntrare));
    if (total !== 100) {
      const lipsa = round3(100 - total);
      window.alert(
        `Procentele din intrare însumează ${formatNumber(total, 3)} %, nu 100 %.\n` +
          (lipsa > 0
            ? `Mai trebuie repartizate ${formatNumber(lipsa, 3)} % (de obicei la deșeu).`
            : `Trebuie scăzute ${formatNumber(-lipsa, 3)} %.`),
      );
      return;
    }
    try {
      await SaveSettings(settings);
      await SaveProducts(products);
      products = await ListProducts();
      renderAll();
      outlet.querySelector('#status')!.textContent = 'Setările au fost salvate.';
    } catch (err) {
      showError('Setările nu au putut fi salvate', err);
    }
  }
}
