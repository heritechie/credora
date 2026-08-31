import { describe, it, expect } from 'vitest';
import { formatCurrency, formatTime, outcomeBadgeClass, statusBadgeClass } from '../format.js';

describe('formatCurrency', () => {
  it('formats a number with default currency', () => {
    expect(formatCurrency(15000000)).toBe('Rp 15,000,000');
  });

  it('returns dash for null', () => {
    expect(formatCurrency(null)).toBe('\u2014');
  });

  it('returns dash for undefined', () => {
    expect(formatCurrency(undefined)).toBe('\u2014');
  });

  it('formats zero', () => {
    expect(formatCurrency(0)).toBe('Rp 0');
  });
});

describe('formatTime', () => {
  it('returns dash for null', () => {
    expect(formatTime(null)).toBe('\u2014');
  });

  it('formats an ISO string', () => {
    const result = formatTime('2025-01-15T10:30:00Z');
    expect(result).not.toBe('\u2014');
    expect(typeof result).toBe('string');
  });
});

describe('outcomeBadgeClass', () => {
  it('returns success for APPROVE', () => {
    expect(outcomeBadgeClass('APPROVE')).toBe('badge-success');
  });

  it('returns warning for REVIEW', () => {
    expect(outcomeBadgeClass('REVIEW')).toBe('badge-warning');
  });

  it('returns danger for REJECT', () => {
    expect(outcomeBadgeClass('REJECT')).toBe('badge-danger');
  });

  it('returns neutral for unknown', () => {
    expect(outcomeBadgeClass('UNKNOWN')).toBe('badge-neutral');
  });
});

describe('statusBadgeClass', () => {
  it('returns success for COMPLETED', () => {
    expect(statusBadgeClass('COMPLETED')).toBe('badge-success');
  });

  it('returns warning for RUNNING', () => {
    expect(statusBadgeClass('RUNNING')).toBe('badge-warning');
  });

  it('returns danger for FAILED', () => {
    expect(statusBadgeClass('FAILED')).toBe('badge-danger');
  });

  it('returns neutral for PENDING', () => {
    expect(statusBadgeClass('PENDING')).toBe('badge-neutral');
  });
});
