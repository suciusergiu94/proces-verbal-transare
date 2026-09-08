import { describe, expect, it } from 'vitest';
import { DRAFT_PREFIX, currentHash } from './sidebar';

describe('currentHash', () => {
  it('falls back to the draft route for the empty hash of a freshly launched app', () => {
    expect(currentHash('')).toBe(DRAFT_PREFIX);
  });

  it('leaves a real route untouched', () => {
    expect(currentHash('#/document/193')).toBe('#/document/193');
    expect(currentHash('#/setari')).toBe('#/setari');
  });

  it('leaves the draft route untouched', () => {
    expect(currentHash(DRAFT_PREFIX)).toBe(DRAFT_PREFIX);
  });

  it('recognises a templated draft route as a real route', () => {
    expect(currentHash('#/document/new/3')).toBe('#/document/new/3');
  });
});
