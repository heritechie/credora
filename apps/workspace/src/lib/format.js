/**
 * Format a monetary value from smallest currency unit to display string.
 * e.g., 15000000 -> "Rp 15,000,000"
 */
export function formatCurrency(amount, currency = 'Rp') {
  if (amount == null) return '—';
  return `${currency} ${amount.toLocaleString()}`;
}

/**
 * Format an ISO timestamp to a human-readable string.
 */
export function formatTime(ts) {
  if (!ts) return '—';
  return new Date(ts).toLocaleString();
}

/**
 * Map a decision outcome to a CSS badge class.
 */
export function outcomeBadgeClass(outcome) {
  switch (outcome) {
    case 'APPROVE': return 'badge-success';
    case 'REVIEW': return 'badge-warning';
    case 'REJECT': return 'badge-danger';
    default: return 'badge-neutral';
  }
}

/**
 * Map an assessment status to a CSS badge class.
 */
export function statusBadgeClass(status) {
  switch (status) {
    case 'COMPLETED': return 'badge-success';
    case 'RUNNING': return 'badge-warning';
    case 'FAILED': return 'badge-danger';
    default: return 'badge-neutral';
  }
}
