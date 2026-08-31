import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { h } from 'preact';
import { render, screen, fireEvent, waitFor } from '@testing-library/preact';
import { Assessments } from '../Assessments.jsx';
import * as apiClient from '../../api/client.js';

vi.mock('../../api/client.js');

function makeAssessment(i) {
  const outcomes = ['APPROVE', 'REVIEW', 'REJECT'];
  const outcome = outcomes[i % 3];
  return {
    id: 'asm-' + String(i).padStart(2, '0'),
    status: 'COMPLETED',
    policy: { id: 'personal-loan', version: 1 },
    decision: { outcome },
    created_at: `2025-01-${String((i % 28) + 1).padStart(2, '0')}T10:30:00Z`,
    completed_at: `2025-01-${String((i % 28) + 1).padStart(2, '0')}T10:30:05Z`,
  };
}

function makeMany(n) {
  return { items: Array.from({ length: n }, (_, i) => makeAssessment(i)) };
}

describe('Assessments (Assessment History)', () => {
  const smallList = {
    items: [
      makeAssessment(0),
      makeAssessment(1),
      makeAssessment(2),
    ],
  };

  beforeEach(() => {
    vi.clearAllMocks();
    apiClient.getAssessments.mockResolvedValue(smallList);
    apiClient.ApiError.mockImplementation(function ApiError(status, code, message) {
      this.status = status;
      this.code = code;
      this.message = message;
      this.name = 'ApiError';
    });
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('renders the page header and developer-oriented description', async () => {
    render(h(Assessments));
    expect(screen.getByText('Assessments')).toBeTruthy();
    expect(screen.getByText('Inspect executed assessments, decisions, and evidence.')).toBeTruthy();
  });

  it('renders New Assessment as a primary action linking to the simulator', async () => {
    render(h(Assessments));
    const cta = screen.getByRole('link', { name: 'New Assessment' });
    expect(cta.getAttribute('href')).toBe('/workspace/assessments/new');
  });

  it('loads assessments from the API on mount', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(apiClient.getAssessments).toHaveBeenCalled(); });
    await waitFor(() => { expect(screen.getByText('asm-00')).toBeTruthy(); });
  });

  it('displays the assessment ID, truncated, with the full ID accessible', async () => {
    apiClient.getAssessments.mockResolvedValue({
      items: [{ id: '1234567890abcdef1234567890abcdef', status: 'COMPLETED', policy: { id: 'p', version: 1 }, decision: { outcome: 'APPROVE' }, created_at: '2025-01-01T00:00:00Z' }],
    });
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('1234567890ab...')).toBeTruthy(); });
    const el = screen.getByText('1234567890ab...');
    expect(el.getAttribute('title')).toBe('1234567890abcdef1234567890abcdef');
  });

  it('displays the policy ID and version', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getAllByText(/personal-loan:v1/).length).toBeGreaterThan(0); });
  });

  it('displays an APPROVE decision outcome', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('APPROVE')).toBeTruthy(); });
  });

  it('displays a REVIEW decision outcome', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('REVIEW')).toBeTruthy(); });
  });

  it('displays a REJECT decision outcome', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('REJECT')).toBeTruthy(); });
  });

  it('renders a placeholder when decision is absent (e.g. FAILED)', async () => {
    apiClient.getAssessments.mockResolvedValue({
      items: [{ id: 'asm-f', status: 'FAILED', policy: { id: 'p', version: 1 }, decision: null, created_at: '2025-01-02T00:00:00Z' }],
    });
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('asm-f')).toBeTruthy(); });
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
  });

  it('displays the assessment status', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getAllByText('COMPLETED').length).toBeGreaterThan(0); });
  });

  it('displays the created_at timestamp', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('asm-00')).toBeTruthy(); });
    expect(screen.getAllByText(/2025/).length).toBeGreaterThan(0);
  });

  it('displays the completed_at timestamp when present', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('asm-00')).toBeTruthy(); });
    // Each of the 3 rows has created_at and completed_at timestamps (6 year-bearing cells).
    expect(screen.getAllByText(/2025/).length).toBeGreaterThanOrEqual(6);
  });

  it('links each assessment to its decision explanation page', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('asm-00')).toBeTruthy(); });
    const links = screen.getAllByRole('link', { name: 'View' });
    const target = links.find(l => l.getAttribute('href') === '/workspace/assessments/asm-00');
    expect(target).toBeTruthy();
  });

  it('shows a useful empty state when no assessments exist', async () => {
    apiClient.getAssessments.mockResolvedValue({ items: [] });
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText(/No assessments have been executed yet/)).toBeTruthy(); });
    const ctas = screen.getAllByRole('link', { name: 'New Assessment' });
    expect(ctas.some(c => c.getAttribute('href') === '/workspace/assessments/new')).toBe(true);
  });

  it('renders an API error and keeps the lookup form available', async () => {
    apiClient.getAssessments.mockRejectedValue(new apiClient.ApiError(500, 'REPOSITORY_ERROR', 'failed to retrieve assessments'));
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('failed to retrieve assessments')).toBeTruthy(); });
    expect(screen.getByRole('button', { name: 'Look Up' })).toBeTruthy();
    expect(screen.getAllByRole('link', { name: 'New Assessment' }).length).toBeGreaterThan(0);
  });

  it('renders a network error without stack traces and keeps lookup available', async () => {
    apiClient.getAssessments.mockRejectedValue(new Error('Network error'));
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('Network error')).toBeTruthy(); });
    expect(screen.getByRole('button', { name: 'Look Up' })).toBeTruthy();
  });

  it('renders the direct lookup form as a secondary utility', async () => {
    render(h(Assessments));
    expect(screen.getByText('Find a Specific Assessment')).toBeTruthy();
    expect(screen.getByLabelText('Assessment ID')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Look Up' })).toBeTruthy();
  });

  it('navigates to the detail page when lookup is submitted', async () => {
    const pushState = vi.spyOn(window.history, 'pushState').mockImplementation(() => {});
    const dispatch = vi.spyOn(window, 'dispatchEvent');
    render(h(Assessments));
    fireEvent.input(screen.getByLabelText('Assessment ID'), { target: { value: 'abc123' } });
    fireEvent.submit(screen.getByRole('button', { name: 'Look Up' }).closest('form'));
    await waitFor(() => {
      expect(pushState).toHaveBeenCalledWith(null, '', '/workspace/assessments/abc123');
    });
    expect(dispatch).toHaveBeenCalled();
    pushState.mockRestore();
    dispatch.mockRestore();
  });

  it('does not navigate when the lookup input is empty', async () => {
    const pushState = vi.spyOn(window.history, 'pushState').mockImplementation(() => {});
    render(h(Assessments));
    const form = screen.getByRole('button', { name: 'Look Up' }).closest('form');
    fireEvent.submit(form);
    expect(pushState).not.toHaveBeenCalled();
    pushState.mockRestore();
  });

  it('disables the lookup button when the input is empty', () => {
    render(h(Assessments));
    expect(screen.getByRole('button', { name: 'Look Up' }).disabled).toBe(true);
  });

  it('shows pagination controls when there are more than one page of items', async () => {
    apiClient.getAssessments.mockResolvedValue(makeMany(25));
    render(h(Assessments));
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Previous' })).toBeTruthy();
    });
    expect(screen.getByRole('button', { name: 'Next' })).toBeTruthy();
    expect(screen.getByText(/Page 1 of 2/)).toBeTruthy();
  });

  it('disables Previous on the first page', async () => {
    apiClient.getAssessments.mockResolvedValue(makeMany(25));
    render(h(Assessments));
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Previous' }).disabled).toBe(true);
    });
    expect(screen.getByRole('button', { name: 'Next' }).disabled).toBe(false);
  });

  it('navigates between pages with Previous and Next', async () => {
    apiClient.getAssessments.mockResolvedValue(makeMany(25));
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('Page 1 of 2')).toBeTruthy(); });
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(screen.getByText('Page 2 of 2')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Previous' }).disabled).toBe(false);
    expect(screen.getByRole('button', { name: 'Next' }).disabled).toBe(true);
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }));
    expect(screen.getByText('Page 1 of 2')).toBeTruthy();
  });

  it('renders the second page items when moving next', async () => {
    apiClient.getAssessments.mockResolvedValue(makeMany(25));
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('Page 1 of 2')).toBeTruthy(); });
    expect(screen.getByText('asm-00')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(screen.getByText('asm-20')).toBeTruthy();
  });

  it('does not show pagination when there is a single page', async () => {
    render(h(Assessments));
    await waitFor(() => { expect(screen.getByText('asm-00')).toBeTruthy(); });
    expect(screen.queryByRole('button', { name: 'Previous' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Next' })).toBeNull();
  });
});
