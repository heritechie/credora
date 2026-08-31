import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { h } from 'preact';
import { render, screen, fireEvent, waitFor } from '@testing-library/preact';
import { AssessmentDetail } from '../AssessmentDetail.jsx';
import * as apiClient from '../../api/client.js';

vi.mock('../../api/client.js');

describe('AssessmentDetail (Decision Explanation)', () => {
  const baseAssessment = {
    id: 'assessment-123',
    status: 'COMPLETED',
    applicant: { id: 'a-1', name: 'Jane Doe', age: 30 },
    policy: { id: 'personal-loan', version: 1 },
    created_at: '2025-01-15T10:30:00Z',
    started_at: '2025-01-15T10:30:01Z',
    completed_at: '2025-01-15T10:30:02Z',
    decision: {
      outcome: 'APPROVE',
      policy: { id: 'personal-loan', version: 1 },
      reasons: [],
      outputs: { credit_limit: 20000000, approved_amount: 15000000 },
    },
  };

  const mockEvidence = [
    { source: 'knockout', field: 'AGE_BELOW_MINIMUM', value: false, retrieved_at: '2025-01-15T10:30:01Z', reference: 'AGE_BELOW_MINIMUM' },
    { source: 'knockout', field: 'HIGH_DSR', value: false, retrieved_at: '2025-01-15T10:30:01Z', reference: 'HIGH_DSR' },
    { source: 'rule', field: 'CREDIT_SCORE_REVIEW', value: false, retrieved_at: '2025-01-15T10:30:01Z', reference: 'CREDIT_SCORE_REVIEW' },
    { source: 'rule', field: 'LOW_CREDIT_SCORE', value: false, retrieved_at: '2025-01-15T10:30:01Z', reference: 'LOW_CREDIT_SCORE' },
  ];

  const rejectAssessment = {
    ...baseAssessment,
    decision: {
      outcome: 'REJECT',
      policy: { id: 'personal-loan', version: 1 },
      reasons: [
        { code: 'HIGH_DSR', description: 'Debt service ratio exceeds policy threshold', value: 800000000, threshold: 700000000, evidence_ref: 'HIGH_DSR' },
      ],
      outputs: null,
    },
  };

  const rejectEvidence = [
    { source: 'knockout', field: 'AGE_BELOW_MINIMUM', value: false, retrieved_at: '2025-01-15T10:30:01Z', reference: 'AGE_BELOW_MINIMUM' },
    { source: 'knockout', field: 'HIGH_DSR', value: true, retrieved_at: '2025-01-15T10:30:01Z', reference: 'HIGH_DSR' },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    apiClient.getAssessment.mockResolvedValue(baseAssessment);
    apiClient.getEvidence.mockResolvedValue(mockEvidence);
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

  const renderDetail = () => render(h(AssessmentDetail, { id: 'assessment-123' }));

  it('renders assessment metadata (ID, status, timestamps, applicant)', async () => {
    renderDetail();
    await waitFor(() => {
      expect(apiClient.getAssessment).toHaveBeenCalledWith('assessment-123');
    });
    expect(screen.getByText('Decision Explanation')).toBeTruthy();
    expect(screen.getAllByText('assessment-123').length).toBeGreaterThan(0);
    expect(screen.getByText('COMPLETED')).toBeTruthy();
    expect(screen.getByText('Created')).toBeTruthy();
    expect(screen.getByText('Started')).toBeTruthy();
    expect(screen.getByText('Completed')).toBeTruthy();
    // Applicant name appears in the applicant card and the canvas node.
    expect(screen.getAllByText('Jane Doe').length).toBeGreaterThan(0);
  });

  it('renders policy ID and version', async () => {
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('personal-loan:v1').length).toBeGreaterThan(0);
    });
  });

  it('renders APPROVE prominently', async () => {
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('APPROVE').length).toBeGreaterThanOrEqual(2);
    });
  });

  it('renders REVIEW decision', async () => {
    apiClient.getAssessment.mockResolvedValue({
      ...baseAssessment,
      decision: { ...baseAssessment.decision, outcome: 'REVIEW' },
    });
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('REVIEW').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('renders REJECT decision', async () => {
    apiClient.getAssessment.mockResolvedValue(rejectAssessment);
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('REJECT').length).toBeGreaterThanOrEqual(1);
    });
  });

  it('renders backend-provided decision outputs', async () => {
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('Credit Limit')).toBeTruthy();
      expect(screen.getByText('Approved Amount')).toBeTruthy();
      expect(screen.getByText('Rp 20,000,000')).toBeTruthy();
      expect(screen.getByText('Rp 15,000,000')).toBeTruthy();
    });
  });

  it('does not render outputs when the backend provides none', async () => {
    apiClient.getAssessment.mockResolvedValue(rejectAssessment);
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('REJECT').length).toBeGreaterThan(0);
    });
    expect(screen.queryByText('Credit Limit')).toBeNull();
    expect(screen.queryByText('Approved Amount')).toBeNull();
  });

  it('renders reason code and description', async () => {
    apiClient.getAssessment.mockResolvedValue(rejectAssessment);
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('HIGH_DSR').length).toBeGreaterThan(0);
      expect(screen.getByText('Debt service ratio exceeds policy threshold')).toBeTruthy();
    });
  });

  it('renders reason value and threshold when present', async () => {
    apiClient.getAssessment.mockResolvedValue(rejectAssessment);
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('800000000 / threshold: 700000000')).toBeTruthy();
    });
  });

  it('shows an empty reasons state when no reasons are reported', async () => {
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('No decision reasons were reported.')).toBeTruthy();
    });
  });

  it('renders evidence as a readable table', async () => {
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('Evidence')).toBeTruthy();
      expect(screen.getAllByText('AGE_BELOW_MINIMUM').length).toBeGreaterThan(0);
      expect(screen.getAllByText('not matched').length).toBeGreaterThan(0);
      expect(screen.getAllByText('knockout').length).toBe(2);
      expect(screen.getAllByText('rule').length).toBe(2);
    });
  });

  it('links a reason to its supporting evidence', async () => {
    apiClient.getAssessment.mockResolvedValue(rejectAssessment);
    apiClient.getEvidence.mockResolvedValue(rejectEvidence);
    renderDetail();
    await waitFor(() => {
      // Reason card shows code + its evidence_ref, and the evidence row's
      // "Evidence For" column surfaces the referencing reason code.
      expect(screen.getAllByText('HIGH_DSR').length).toBeGreaterThanOrEqual(2);
    });
    const matchedCells = screen.getAllByText('matched');
    expect(matchedCells.length).toBeGreaterThan(0);
  });

  it('shows the evaluation trace only when evidence provides condition evaluations', async () => {
    renderDetail();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Evaluation Trace/ })).toBeTruthy();
    });
  });

  it('hides trace content until expanded (collapsible behavior)', async () => {
    renderDetail();
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Evaluation Trace/ })).toBeTruthy();
    });
    expect(screen.queryByText('Knockouts')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /Evaluation Trace/ }));
    await waitFor(() => {
      expect(screen.getByText('Knockouts')).toBeTruthy();
      expect(screen.getByText('Rules')).toBeTruthy();
      expect(screen.getByText(/not returned by the current API/)).toBeTruthy();
    });
  });

  it('does not show a trace when there is no evidence', async () => {
    apiClient.getEvidence.mockResolvedValue([]);
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('APPROVE').length).toBeGreaterThan(0);
    });
    expect(screen.queryByText('Evidence')).toBeNull();
    expect(screen.queryByText(/Evaluation Trace/)).toBeNull();
  });

  it('shows a clear state when the assessment has no decision', async () => {
    apiClient.getAssessment.mockResolvedValue({
      ...baseAssessment,
      status: 'FAILED',
      decision: null,
      error: 'provider unavailable',
    });
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('No decision is available for this assessment yet.')).toBeTruthy();
    });
    expect(apiClient.getEvidence).not.toHaveBeenCalled();
  });

  it('keeps the decision visible when evidence loading fails', async () => {
    apiClient.getEvidence.mockRejectedValue(
      new apiClient.ApiError(500, 'INTERNAL_ERROR', 'failed to retrieve evidence')
    );
    renderDetail();
    await waitFor(() => {
      expect(screen.getAllByText('APPROVE').length).toBeGreaterThan(0);
      expect(screen.getByText('Evidence: INTERNAL_ERROR: failed to retrieve evidence')).toBeTruthy();
    });
  });

  it('shows a not-found state for a 404', async () => {
    apiClient.getAssessment.mockRejectedValue(
      new apiClient.ApiError(404, 'NOT_FOUND', 'assessment not found')
    );
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('Assessment not found.')).toBeTruthy();
    });
  });

  it('shows API errors without stack traces', async () => {
    apiClient.getAssessment.mockRejectedValue(
      new apiClient.ApiError(500, 'INTERNAL_ERROR', 'failed to retrieve assessment')
    );
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('INTERNAL_ERROR: failed to retrieve assessment')).toBeTruthy();
    });
  });

  it('shows network errors without stack traces', async () => {
    apiClient.getAssessment.mockRejectedValue(new Error('Network error'));
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText('Network error')).toBeTruthy();
    });
  });

  describe('Decisioning Canvas', () => {
    it('renders the canvas section heading', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByText('Decisioning Canvas')).toBeTruthy();
      });
    });

    it('renders the horizontal flow as a sequence of stages', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-input')).toBeTruthy();
      });
      const input = screen.getByTestId('flow-stage-input');
      const checks = screen.getByTestId('flow-stage-checks');
      const policy = screen.getByTestId('flow-stage-policy');
      const decision = screen.getByTestId('flow-stage-decision');
      expect(input.compareDocumentPosition(checks) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
      expect(checks.compareDocumentPosition(policy) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
      expect(policy.compareDocumentPosition(decision) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    });

    it('renders the applicant/input stage from backend data', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-input')).toBeTruthy();
      });
      expect(screen.getByTestId('flow-stage-input').textContent).toContain('Applicant');
    });

    it('renders the Evaluated Checks stage counting backend evidence', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-checks')).toBeTruthy();
      });
      expect(screen.getByTestId('flow-stage-checks').textContent).toContain('Evaluated Checks');
      expect(screen.getByTestId('flow-stage-checks').textContent).toContain('4 conditions evaluated');
    });

    it('renders the Policy stage with id:version', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-policy')).toBeTruthy();
      });
      expect(screen.getByTestId('flow-stage-policy').textContent).toContain('personal-loan:v1');
    });

    it('renders the Decision stage directly from the backend outcome', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-decision-value')).toBeTruthy();
      });
      expect(screen.getByTestId('flow-stage-decision-value').textContent).toBe('APPROVE');
    });

    it('does not fabricate provider stages', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-decision-value')).toBeTruthy();
      });
      const flow = screen.getByTestId('flow-stage-input').closest('.decision-flow');
      const text = flow.textContent;
      expect(text).not.toMatch(/KYC/i);
      expect(text).not.toMatch(/Fraud/i);
      expect(text).not.toMatch(/PEFINDO/i);
      expect(text).not.toMatch(/FDC/i);
      expect(screen.queryByTestId('flow-stage-provider')).toBeNull();
    });

    it('omits the Evaluated Checks stage when the API returns no evidence', async () => {
      apiClient.getEvidence.mockResolvedValue([]);
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-decision-value')).toBeTruthy();
      });
      expect(screen.queryByTestId('flow-stage-checks')).toBeNull();
    });

    it('shows a caption clarifying it is an explanation, not an execution trace', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-input')).toBeTruthy();
      });
      expect(screen.getByText(/not an execution trace/)).toBeTruthy();
    });

    it('does not show an interactive node-detail panel', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-input')).toBeTruthy();
      });
      expect(screen.queryByText(/Select a node to inspect its details/)).toBeNull();
      // Stages are not clickable/selectable surfaces.
      expect(screen.queryByTestId('canvas-flow')).toBeNull();
    });

    it('does not crash the canvas when evidence is missing/optional', async () => {
      apiClient.getAssessment.mockResolvedValue({
        ...baseAssessment,
        decision: null,
        status: 'RUNNING',
      });
      apiClient.getEvidence.mockResolvedValue(null);
      renderDetail();
      await waitFor(() => {
        // No-decision assessment still renders the canvas with the stages it can.
        expect(screen.getByTestId('flow-stage-input')).toBeTruthy();
      });
    });

    it('still renders the existing explanation sections', async () => {
      renderDetail();
      await waitFor(() => {
        expect(screen.getByTestId('flow-stage-decision-value')).toBeTruthy();
      });
      expect(screen.getByText('Evidence')).toBeTruthy();
      expect(screen.getByRole('button', { name: /Evaluation Trace/ })).toBeTruthy();
      expect(screen.getByText('Credit Limit')).toBeTruthy();
    });
  });
});
