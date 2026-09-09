import type { IesireRow, Template } from './api';
import { procenteDinCantitati, round3, sumaProcente } from './calc';
import { formatNumber } from './format';

/**
 * Pure logic behind the template screens. It lives outside the views so it can
 * be tested: vitest runs in node here, with no DOM, and the views are wiring.
 */

/** The route an unsaved document lives at, before its template is named. */
export const DRAFT_PREFIX = '#/document/new';

/** The route of an unsaved document created from a given template. */
export function draftHash(templateId: number): string {
  return `${DRAFT_PREFIX}/${templateId}`;
}

/** Whether a hash is the draft route, with or without a template on it. */
export function isDraftHash(hash: string): boolean {
  return hash === DRAFT_PREFIX || hash.startsWith(`${DRAFT_PREFIX}/`);
}

/** The template a draft route names, if it names one. */
export function draftTemplateId(hash: string): number | undefined {
  const match = /^#\/document\/new\/(\d+)$/.exec(hash);
  return match === null ? undefined : Number(match[1]);
}

/** A blank, unsaved template — what "+ Șablon nou" appends. */
export function emptyTemplate(): Template {
  // `as Template` alone fails here under strict mode (TS2352): the generated
  // Template type carries `convertValues`, which this object literal has no
  // way to supply, and `products: []` is inferred as `never[]`, which blocks
  // assignability the other way too. Going through `unknown` is the only way
  // to state "this is a plain-data Template" without adding a fake method.
  return { id: 0, nume: '', ordine: 0, products: [] } as unknown as Template;
}

/**
 * A copy of `t` that has never been stored: the template and every one of its
 * products carry id 0, so saving inserts them rather than moving the originals.
 * This is how a second profile that mostly resembles the first gets started.
 */
export function duplicateTemplate(t: Template): Template {
  return {
    id: 0,
    nume: `${t.nume} (copie)`,
    ordine: 0,
    products: t.products.map((p) => ({ ...p, id: 0, templateId: 0 })),
  } as Template;
}

/** Where a problem is and what to tell the user about it. */
export type TemplateProblem = { templateIndex: number; message: string };

/**
 * Renders a ratio the Romanian way: formatNumber's three-decimal fixed
 * precision (the same convention Setări's tables use), with the decimal
 * separator swapped from a dot to a comma for prose.
 *
 * Exported so Setări's section header uses the exact same rendering as the
 * save-refusal message that quotes the same figure — the two are on screen
 * together, and must never disagree on comma vs. dot.
 */
export function formatProcent(value: number): string {
  return formatNumber(value, 3).replace('.', ',');
}

/**
 * The first thing wrong with the template list, or undefined if nothing is.
 *
 * The 100% rule is the substantial one: the form fills the "ce iese"
 * quantities from these ratios, so a column that does not account for the whole
 * carcass would quietly produce a document that is short. It is refused rather
 * than normalised — the figures are the user's to decide, and the gap is named
 * so it can be put where it belongs, usually on deșeu.
 */
export function validateTemplates(templates: Template[]): TemplateProblem | undefined {
  for (const [templateIndex, t] of templates.entries()) {
    const nume = t.nume.trim();
    if (nume === '') {
      return { templateIndex, message: `Șablonul ${templateIndex + 1} nu are nume.` };
    }
    if (t.products.length === 0) {
      return { templateIndex, message: `Șablonul „${nume}” nu are niciun produs.` };
    }
    if (t.products.some((p) => p.denumire.trim() === '')) {
      return {
        templateIndex,
        message: `Șablonul „${nume}”: fiecare produs trebuie să aibă o denumire.`,
      };
    }
    if (t.products.some((p) => p.procentDinIntrare < 0)) {
      return {
        templateIndex,
        message: `Șablonul „${nume}”: procentul din intrare nu poate fi negativ.`,
      };
    }
    const total = sumaProcente(t.products.map((p) => p.procentDinIntrare));
    if (total !== 100) {
      const lipsa = round3(100 - total);
      return {
        templateIndex,
        message:
          `Șablonul „${nume}”: procentele din intrare însumează ` +
          `${formatProcent(total)} %, nu 100 %.\n` +
          (lipsa > 0
            ? `Mai trebuie repartizate ${formatProcent(lipsa)} % (de obicei la deșeu).`
            : `Trebuie scăzute ${formatProcent(-lipsa)} %.`),
      };
    }
  }
  return undefined;
}

/**
 * The template list with one template's ratios replaced by the split a document
 * actually yielded. Every other template comes back untouched.
 *
 * The ratios are worked out per product rather than per row: a row whose
 * product has since been deleted cannot carry a share, and must not take one
 * with it and leave the column short of the 100% the form insists on. A product
 * this document did not yield at all goes to zero for the same reason — the
 * column being written is the whole of it, not a patch over what is stored.
 *
 * Returns undefined when the template is not in the list, or when the document
 * yielded nothing to derive a split from.
 */
export function templatesCuProcenteNoi(
  templates: Template[],
  templateId: number,
  cantitatiPerProdus: Map<number, number>,
): Template[] | undefined {
  const target = templates.find((t) => t.id === templateId);
  if (target === undefined) return undefined;

  const noi = procenteDinCantitati(target.products.map((p) => cantitatiPerProdus.get(p.id) ?? 0));
  if (noi === undefined) return undefined;

  // As with emptyTemplate above, `{ ...t, ... }` is a plain object literal:
  // spread does not copy convertValues, which lives on Template's prototype,
  // not as an own property of `t`. So this needs the same unknown-first cast
  // as emptyTemplate — contrary to this being a plain reuse of `t`'s shape.
  return templates.map((t) =>
    t.id === templateId
      ? ({ ...t, products: t.products.map((p, i) => ({ ...p, procentDinIntrare: noi[i] })) } as unknown as Template)
      : t,
  );
}

/**
 * Sums each "ce iese" row's quantity by the product it came from, ignoring
 * rows with no product (their share was deleted along with the product, and
 * they have nowhere to add to). Two rows on the same product — which the form
 * never creates today, but nothing stops a document from someday listing a
 * product twice — accumulate into one total rather than overwriting.
 */
export function cantitatiPerProdus(rows: IesireRow[]): Map<number, number> {
  const out = new Map<number, number>();
  rows.forEach((row) => {
    if (row.productId == null) return;
    out.set(row.productId, (out.get(row.productId) ?? 0) + row.cantitate);
  });
  return out;
}

/**
 * Remaps a set of expanded section indices after a section is inserted at
 * `at`: every index at or after the insertion point shifts up by one to keep
 * pointing at the same section, and `at` itself is added so the new/copied
 * section opens.
 */
export function expandedAfterInsert(expanded: Set<number>, at: number): Set<number> {
  return new Set([...expanded].map((n) => (n >= at ? n + 1 : n)).concat(at));
}

/**
 * Remaps a set of expanded section indices after the section at `at` is
 * removed: it is dropped from the set, and every index after it shifts down
 * by one to keep pointing at the same section.
 */
export function expandedAfterRemove(expanded: Set<number>, at: number): Set<number> {
  return new Set([...expanded].filter((n) => n !== at).map((n) => (n > at ? n - 1 : n)));
}
