import { describe, expect, it } from 'vitest';
import { escapeHtml } from './sidebar';

describe('escapeHtml', () => {
  it('escapes double quotes (attribute-breakout case)', () => {
    expect(escapeHtml('"')).toBe('&quot;');
  });

  it('escapes single quotes', () => {
    expect(escapeHtml("'")).toBe('&#39;');
  });

  it('escapes ampersand', () => {
    expect(escapeHtml('&')).toBe('&amp;');
  });

  it('escapes less-than', () => {
    expect(escapeHtml('<')).toBe('&lt;');
  });

  it('escapes greater-than', () => {
    expect(escapeHtml('>')).toBe('&gt;');
  });

  it('escapes ampersand first so entities are not double-escaped', () => {
    expect(escapeHtml('<')).not.toBe('&amp;lt;');
    expect(escapeHtml('&lt;')).toBe('&amp;lt;');
  });

  it('escapes a realistic product name with embedded quotes', () => {
    expect(escapeHtml('Costita "afumata"')).toBe('Costita &quot;afumata&quot;');
  });

  it('escapes an injection attempt aimed at an attribute value', () => {
    expect(escapeHtml('x" onfocus="alert(1)')).toBe('x&quot; onfocus=&quot;alert(1)');
  });
});
