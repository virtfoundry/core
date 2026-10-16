import { describe, expect, it } from 'vitest';
import { mergeTags } from './TagsInput';

describe('mergeTags', () => {
  it('appends the pending draft to the committed tags', () => {
    expect(mergeTags(['prod'], 'staging')).toEqual(['prod', 'staging']);
  });

  it('splits the draft on commas and semicolons', () => {
    expect(mergeTags(['prod'], 'staging, dev; qa')).toEqual(['prod', 'staging', 'dev', 'qa']);
  });

  it('trims whitespace and drops empty parts', () => {
    expect(mergeTags(['prod'], '  staging ,  , dev  ')).toEqual(['prod', 'staging', 'dev']);
  });

  it('does not duplicate an existing tag', () => {
    expect(mergeTags(['prod', 'dev'], 'dev, prod')).toEqual(['prod', 'dev']);
  });

  it('returns the committed tags when the draft is empty', () => {
    expect(mergeTags(['prod', 'dev'], '')).toEqual(['prod', 'dev']);
  });
});
