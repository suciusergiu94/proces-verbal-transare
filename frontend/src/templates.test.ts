import { describe, expect, it } from 'vitest';
import type { Product, Template } from './api';
import {
  DRAFT_PREFIX,
  draftHash,
  draftTemplateId,
  duplicateTemplate,
  emptyTemplate,
  isDraftHash,
  templatesCuProcenteNoi,
  validateTemplates,
} from './templates';

function product(over: Partial<Product> = {}): Product {
  return {
    id: 0,
    templateId: 0,
    denumire: 'Produs',
    um: 'Kg',
    pretCuTva: 10,
    procentDinIntrare: 100,
    ordine: 0,
    ...over,
  } as Product;
}

function template(over: Partial<Template> = {}): Template {
  return { id: 1, nume: 'Carcasa Porc', ordine: 0, products: [product()], ...over } as Template;
}

describe('draft hashes', () => {
  it('round-trips a template id', () => {
    expect(draftHash(7)).toBe('#/document/new/7');
    expect(draftTemplateId(draftHash(7))).toBe(7);
  });

  it('recognises both the bare and the templated draft route', () => {
    expect(isDraftHash(DRAFT_PREFIX)).toBe(true);
    expect(isDraftHash('#/document/new/7')).toBe(true);
  });

  it('does not mistake a saved document or another page for a draft', () => {
    expect(isDraftHash('#/document/12')).toBe(false);
    expect(isDraftHash('#/setari')).toBe(false);
    expect(isDraftHash('')).toBe(false);
  });

  it('has no template id on the bare draft route', () => {
    expect(draftTemplateId(DRAFT_PREFIX)).toBeUndefined();
    expect(draftTemplateId('#/document/12')).toBeUndefined();
  });
});

describe('emptyTemplate', () => {
  it('is an unsaved template with nothing in it', () => {
    const t = emptyTemplate();
    expect(t.id).toBe(0);
    expect(t.nume).toBe('');
    expect(t.products).toEqual([]);
  });
});

describe('duplicateTemplate', () => {
  it('marks the copy and every product as unsaved', () => {
    const source = template({ id: 3, products: [product({ id: 9, denumire: 'Pulpa' })] });

    const copy = duplicateTemplate(source);

    expect(copy.id).toBe(0);
    expect(copy.nume).toBe('Carcasa Porc (copie)');
    expect(copy.products).toHaveLength(1);
    expect(copy.products[0].id).toBe(0);
    expect(copy.products[0].denumire).toBe('Pulpa');
  });

  it('leaves the source untouched', () => {
    const source = template({ id: 3, products: [product({ id: 9 })] });

    duplicateTemplate(source);

    expect(source.id).toBe(3);
    expect(source.nume).toBe('Carcasa Porc');
    expect(source.products[0].id).toBe(9);
  });
});

describe('validateTemplates', () => {
  it('accepts a template whose ratios account for the whole carcass', () => {
    expect(validateTemplates([template()])).toBeUndefined();
  });

  it('rejects a blank name, by position', () => {
    const problem = validateTemplates([template(), template({ id: 2, nume: '  ' })]);
    expect(problem?.templateIndex).toBe(1);
    expect(problem?.message).toContain('nume');
  });

  it('rejects a template with no products', () => {
    const problem = validateTemplates([template({ products: [] })]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('Carcasa Porc');
  });

  it('rejects a blank product denumire', () => {
    const problem = validateTemplates([template({ products: [product({ denumire: ' ' })] })]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('denumire');
  });

  it('rejects a negative ratio', () => {
    const problem = validateTemplates([
      template({ products: [product({ procentDinIntrare: -5 }), product({ procentDinIntrare: 105 })] }),
    ]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('negativ');
  });

  it('names the template and the shortfall when the ratios do not reach 100', () => {
    const problem = validateTemplates([template({ products: [product({ procentDinIntrare: 97.5 })] })]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('Carcasa Porc');
    expect(problem?.message).toContain('97,5');
    expect(problem?.message).toContain('2,5');
  });

  it('says what to subtract when the ratios overshoot 100', () => {
    const problem = validateTemplates([template({ products: [product({ procentDinIntrare: 101 })] })]);
    expect(problem?.message).toContain('scăzute');
    expect(problem?.message).toContain('1');
  });

  it('reports the first bad template, not the last', () => {
    const problem = validateTemplates([
      template({ id: 1, nume: 'Unu', products: [product({ procentDinIntrare: 50 })] }),
      template({ id: 2, nume: 'Doi', products: [product({ procentDinIntrare: 50 })] }),
    ]);
    expect(problem?.templateIndex).toBe(0);
    expect(problem?.message).toContain('Unu');
  });
});

describe('templatesCuProcenteNoi', () => {
  const porc = template({
    id: 1,
    nume: 'Carcasa Porc',
    products: [
      product({ id: 10, denumire: 'Pulpa', procentDinIntrare: 60 }),
      product({ id: 11, denumire: 'Deseu', procentDinIntrare: 40 }),
    ],
  });
  const vitel = template({
    id: 2,
    nume: 'Carcasa Vitel',
    products: [product({ id: 20, denumire: 'Antricot', procentDinIntrare: 100 })],
  });

  it('rewrites only the named template', () => {
    const out = templatesCuProcenteNoi([porc, vitel], 1, new Map([[10, 30], [11, 70]]));

    expect(out).toBeDefined();
    expect(out![0].products.map((p) => p.procentDinIntrare)).toEqual([30, 70]);
    expect(out![1]).toEqual(vitel);
  });

  it('zeroes a product this document did not yield', () => {
    const out = templatesCuProcenteNoi([porc], 1, new Map([[10, 100]]));

    expect(out![0].products[1].procentDinIntrare).toBe(0);
    expect(out![0].products[0].procentDinIntrare).toBe(100);
  });

  it('gives back nothing when there are no quantities to work from', () => {
    expect(templatesCuProcenteNoi([porc], 1, new Map([[10, 0], [11, 0]]))).toBeUndefined();
    expect(templatesCuProcenteNoi([porc], 1, new Map())).toBeUndefined();
  });

  it('gives back nothing for a template that is not in the list', () => {
    expect(templatesCuProcenteNoi([porc], 99, new Map([[10, 5]]))).toBeUndefined();
  });

  it('does not mutate its input', () => {
    templatesCuProcenteNoi([porc, vitel], 1, new Map([[10, 30], [11, 70]]));
    expect(porc.products[0].procentDinIntrare).toBe(60);
  });
});
