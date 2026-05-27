import { dateFilterComparator, dateSortComparator } from './date.util';

describe('dateFilterComparator', () => {
  it('should return 0 when cellValue equals filterLocalDateAtMidnight', () => {
    const result = dateFilterComparator(
      new Date('2024-09-30T00:00:00.000Z'),
      '2024-09-30T00:00:00.000Z',
    );
    expect(result).toBe(0);
  });

  it('should return -1 when cellValue is before filterLocalDateAtMidnight', () => {
    const result = dateFilterComparator(
      new Date('2024-09-30T00:00:00.000z'),
      '2024-09-29T00:00:00.000Z',
    );
    expect(result).toBe(-1);
  });

  it('should return 1 when cellValue is after filterLocalDateAtMidnight', () => {
    const result = dateFilterComparator(
      new Date('2024-09-30T00:00:00.000Z'),
      '2024-10-01T00:00:00.000Z',
    );
    expect(result).toBe(1);
  });
});

describe('dateSortComparator', () => {
  it('should return a negative number when the first date is earlier than the second', () => {
    const result = dateSortComparator(
      '2024-09-29T00:00:00.000Z',
      '2024-09-30T00:00:00.000Z',
    );
    expect(result).toBeLessThan(0);
  });

  it('should return a positive number when the first date is later than the second', () => {
    const result = dateSortComparator(
      '2024-10-01T00:00:00.000Z',
      '2024-09-30T00:00:00.000Z',
    );
    expect(result).toBeGreaterThan(0);
  });

  it('should return 0 when both dates are the same', () => {
    const result = dateSortComparator(
      '2024-09-30T00:00:00.000Z',
      '2024-09-30T00:00:00.000Z',
    );
    expect(result).toBe(0);
  });
});
