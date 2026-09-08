import { describe, expect, it } from 'vitest';
import css from './style.css?raw';

/* Butoanele trebuie sa ramana lizibile si in starea :hover.

   Regula generica `.btn:hover` are specificitate mai mare (0,2,0) decat
   `.btn-primary` (0,1,0), asa ca poate suprascrie fundalul variantei fara sa
   atinga si culoarea textului - de aici rezulta text alb pe fundal aproape alb.
   Testul rezolva cascada pe fisierul real de stiluri si verifica contrastul. */

interface Rule {
  selector: string;
  order: number;
  decls: Record<string, string>;
}

function parseRules(source: string): Rule[] {
  const withoutComments = source.replace(/\/\*[\s\S]*?\*\//g, '');
  // Blocurile at-rule (@media) nu vizeaza butoane; le eliminam cu tot cu continut.
  const flat = withoutComments.replace(/@[\w-]+[^{]*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}/g, '');

  const rules: Rule[] = [];
  let order = 0;
  for (const match of flat.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const decls: Record<string, string> = {};
    for (const line of match[2].split(';')) {
      const idx = line.indexOf(':');
      if (idx === -1) continue;
      decls[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
    }
    for (const selector of match[1].split(',')) {
      rules.push({ selector: selector.trim(), order: order++, decls });
    }
  }
  return rules;
}

const rules = parseRules(css);

function rootVars(): Record<string, string> {
  const vars: Record<string, string> = {};
  for (const rule of rules) {
    if (rule.selector !== ':root') continue;
    for (const [prop, value] of Object.entries(rule.decls)) {
      if (prop.startsWith('--')) vars[prop] = value;
    }
  }
  return vars;
}

const vars = rootVars();

function specificity(selector: string): number {
  const ids = selector.match(/#[\w-]+/g)?.length ?? 0;
  const classes = selector.match(/[.:\[][\w-]+/g)?.length ?? 0;
  return ids * 100 + classes * 10;
}

/** Selectoare compuse dintr-un singur element (fara combinatori) care se
 *  potrivesc pe un buton cu setul de clase dat, in starea ceruta. */
function matches(selector: string, classes: string[], hover: boolean): boolean {
  if (/[\s>+~]/.test(selector.trim())) return false;
  const tokens = selector.trim().match(/[.:][\w-]+/g);
  if (!tokens || tokens.join('') !== selector.trim()) return false;
  return tokens.every((token) => {
    if (token === ':hover') return hover;
    if (token.startsWith('.')) return classes.includes(token.slice(1));
    return false;
  });
}

function resolve(prop: string, classes: string[], hover: boolean): string | undefined {
  let best: { value: string; spec: number; order: number } | undefined;
  for (const rule of rules) {
    if (!matches(rule.selector, classes, hover)) continue;
    const value = rule.decls[prop] ?? (prop === 'background' ? rule.decls['background-color'] : undefined);
    if (value === undefined) continue;
    const spec = specificity(rule.selector);
    if (!best || spec > best.spec || (spec === best.spec && rule.order > best.order)) {
      best = { value, spec, order: rule.order };
    }
  }
  return best?.value;
}

/** Rezolva var(--x) si var(--x, fallback), cautand intai proprietatile
 *  personalizate declarate pe buton si abia apoi pe :root. */
function toRgb(raw: string, classes: string[], hover: boolean, depth = 0): [number, number, number] {
  if (depth > 10) throw new Error(`var() imbricat prea adanc: ${raw}`);
  const value = raw.trim();

  const varMatch = value.match(/^var\(\s*(--[\w-]+)\s*(?:,([\s\S]+))?\)$/);
  if (varMatch) {
    const name = varMatch[1];
    const declared = resolve(name, classes, hover) ?? vars[name];
    if (declared !== undefined) return toRgb(declared, classes, hover, depth + 1);
    if (varMatch[2] !== undefined) return toRgb(varMatch[2], classes, hover, depth + 1);
    throw new Error(`variabila nedefinita si fara alternativa: ${name}`);
  }

  const hex = value.match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i);
  if (!hex) throw new Error(`nu este o culoare simpla: ${raw}`);
  const digits = hex[1].length === 3 ? hex[1].replace(/./g, (c) => c + c) : hex[1];
  return [0, 2, 4].map((i) => parseInt(digits.slice(i, i + 2), 16)) as [number, number, number];
}

function luminance(rgb: [number, number, number]): number {
  const [r, g, b] = rgb.map((channel) => {
    const c = channel / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function buttonContrast(classes: string[], hover: boolean): number {
  const color = resolve('color', classes, hover);
  const background = resolve('background', classes, hover);
  if (!color) throw new Error(`fara culoare de text pentru ${classes.join('.')}`);
  if (!background) throw new Error(`fara fundal pentru ${classes.join('.')}`);
  const a = luminance(toRgb(color, classes, hover));
  const b = luminance(toRgb(background, classes, hover));
  const [hi, lo] = a > b ? [a, b] : [b, a];
  return (hi + 0.05) / (lo + 0.05);
}

const variants: Array<[string, string[]]> = [
  ['butonul standard', ['btn']],
  ['butonul principal', ['btn', 'btn-primary']],
  ['butonul de stergere', ['btn', 'btn-danger']],
];

describe('contrastul butoanelor', () => {
  for (const [name, classes] of variants) {
    it(`${name} ramane lizibil in repaus`, () => {
      expect(buttonContrast(classes, false)).toBeGreaterThanOrEqual(4.5);
    });

    it(`${name} ramane lizibil la hover`, () => {
      expect(buttonContrast(classes, true)).toBeGreaterThanOrEqual(4.5);
    });
  }
});
