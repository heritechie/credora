import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { h } from 'preact';
import { render, screen, fireEvent, waitFor } from '@testing-library/preact';
import { NewAssessment } from '../NewAssessment.jsx';
import * as apiClient from '../../api/client.js';

vi.mock('../../api/client.js');

describe('NewAssessment (Assessment Simulator)', () => {
  const mockPolicies = {
    items: [
      { id: 'personal-loan', version: 1, description: 'Deterministic personal loan decision policy' },
      { id: 'business-loan', version: 2, description: 'Business loan policy' },
    ],
  };

  const mockAssessmentResult = {
    id: 'test-assessment-123',
    status: 'COMPLETED',
    applicant: { id: 'applicant-1', name: 'Simulator', age: 30 },
    policy: { id: 'personal-loan', version: 1 },
    decision: {
      outcome: 'APPROVE',
      policy: { id: 'personal-loan', version: 1 },
      reasons: [
        { code: 'HIGH_DSR', description: 'Debt service ratio exceeds policy threshold', value: 0.8, threshold: 0.7, evidence_ref: 'HIGH_DSR' },
      ],
      outputs: {
        credit_limit: 20000000,
        approved_amount: 15000000,
      },
    },
  };

  const mockEvidence = [
    { source: 'applicant', field: 'monthly_income', value: 10000000, retrieved_at: new Date().toISOString(), reference: 'applicant-1' },
    { source: 'applicant', field: 'monthly_obligations', value: 3000000, retrieved_at: new Date().toISOString(), reference: 'applicant-1' },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    apiClient.getPolicies.mockResolvedValue(mockPolicies);
    apiClient.createAssessment.mockResolvedValue(mockAssessmentResult);
    apiClient.getEvidenceByAssessmentId.mockResolvedValue(mockEvidence);
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

  const runAssessment = async () => {
    fireEvent.click(screen.getByRole('button', { name: 'Run Assessment' }));
  };

  // Submits the form directly. Clicking the submit button runs native
  // constraint validation first; with invalid required fields happy-dom
  // blocks the submit event, so validation-only tests dispatch submit
  // on the form element to exercise the component's client-side checks.
  const submitForm = () => {
    fireEvent.submit(document.querySelector('form'));
  };

  it('renders simulator with all sections', () => {
    render(h(NewAssessment));
    expect(screen.getAllByText('Assessment Simulator').length).toBeGreaterThan(0);
    expect(screen.getByText('Applicant')).toBeTruthy();
    expect(screen.getByText('Financial')).toBeTruthy();
    expect(screen.getByText('Credit')).toBeTruthy();
    expect(screen.getByText('Application')).toBeTruthy();
    expect(screen.getAllByText('Policy').length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: 'Run Assessment' })).toBeTruthy();
  });

  it('loads policies from API on mount', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    const policySelect = screen.getByLabelText('Policy');
    expect(policySelect.options.length).toBe(2);
  });

  it('defaults to personal-loan:v1', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    const policySelect = screen.getByLabelText('Policy');
    expect(policySelect.value).toBe('personal-loan');
    const versionInput = screen.getByLabelText('Version');
    expect(versionInput.value).toBe('1');
  });

  it('policy selector updates version when policy changes', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    const policySelect = screen.getByLabelText('Policy');
    fireEvent.change(policySelect, { target: { value: 'business-loan' } });
    await waitFor(() => {
      const versionInput = screen.getByLabelText('Version');
      expect(versionInput.value).toBe('2');
    });
  });

  it('shows validation errors for required fields', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });

    fireEvent.input(screen.getByLabelText('Age'), { target: { value: '' } });
    fireEvent.input(screen.getByLabelText('Monthly Income'), { target: { value: '' } });
    fireEvent.input(screen.getByLabelText('Monthly Obligations'), { target: { value: '' } });
    fireEvent.input(screen.getByLabelText('Credit Score'), { target: { value: '' } });
    fireEvent.input(screen.getByLabelText('Score Provider'), { target: { value: '' } });
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Age is required')).toBeTruthy();
      expect(screen.getByText('Monthly income is required')).toBeTruthy();
      expect(screen.getByText('Monthly obligations is required')).toBeTruthy();
      expect(screen.getByText('Credit score is required')).toBeTruthy();
      expect(screen.getByText('Score provider is required')).toBeTruthy();
      expect(apiClient.createAssessment).not.toHaveBeenCalled();
    });
  });

  it('validates numeric values and ranges', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });

    fireEvent.input(screen.getByLabelText('Age'), { target: { value: '200' } });
    fireEvent.input(screen.getByLabelText('Monthly Income'), { target: { value: '-100' } });
    fireEvent.input(screen.getByLabelText('Monthly Obligations'), { target: { value: '12.5' } });
    fireEvent.input(screen.getByLabelText('Credit Score'), { target: { value: '1500' } });
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Age must be a number between 0 and 150')).toBeTruthy();
      expect(screen.getByText('Monthly income must be a non-negative number')).toBeTruthy();
      expect(screen.getByText('Monthly obligations must be a non-negative number')).toBeTruthy();
      expect(screen.getByText('Credit score must be a number between 0 and 1000')).toBeTruthy();
    });
  });

  it('marks requested amount and purpose as optional', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    const amountInput = screen.getByLabelText(/Requested Amount/);
    expect(amountInput.required).toBe(false);
    const purposeInput = screen.getByLabelText(/Purpose/);
    expect(purposeInput.required).toBe(false);
  });

  it('runs a limit assessment without requested amount (application omitted)', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });

    await runAssessment();

    await waitFor(() => {
      expect(apiClient.createAssessment).toHaveBeenCalledWith(expect.objectContaining({
        applicant: expect.objectContaining({ age: 30 }),
        monthly_income: 10000000,
        monthly_obligations: 3000000,
        score: expect.objectContaining({ value: 720, provider: 'mock-credit-bureau' }),
        policy: expect.objectContaining({ id: 'personal-loan', version: 1 }),
      }));
      const body = apiClient.createAssessment.mock.calls[0][0];
      expect(body.application).toBeUndefined();
    });
  });

  it('runs a loan application assessment with requested amount', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });

    fireEvent.input(screen.getByLabelText(/Requested Amount/), { target: { value: '50000000' } });
    fireEvent.input(screen.getByLabelText(/Purpose/), { target: { value: 'working_capital' } });
    await runAssessment();

    await waitFor(() => {
      expect(apiClient.createAssessment).toHaveBeenCalledWith(expect.objectContaining({
        application: expect.objectContaining({
          purpose: 'working_capital',
          requested_amount: 50000000,
        }),
      }));
    });
  });

  it('includes an application without requested amount when purpose is provided', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });

    fireEvent.input(screen.getByLabelText(/Purpose/), { target: { value: 'working_capital' } });
    await runAssessment();

    await waitFor(() => {
      const body = apiClient.createAssessment.mock.calls[0][0];
      expect(body.application).toBeDefined();
      expect(body.application.requested_amount).toBeUndefined();
      expect(body.application.purpose).toBe('working_capital');
    });
  });

  it('validates requested amount when provided', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });

    fireEvent.input(screen.getByLabelText(/Requested Amount/), { target: { value: '-500' } });
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Requested amount must be a non-negative number')).toBeTruthy();
      expect(apiClient.createAssessment).not.toHaveBeenCalled();
    });
  });

  it('shows loading state during assessment and disables submit', async () => {
    let resolveCreate;
    apiClient.createAssessment.mockImplementation(() => new Promise(resolve => { resolveCreate = resolve; }));
    apiClient.getEvidenceByAssessmentId.mockResolvedValue(mockEvidence);

    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });

    await runAssessment();

    await waitFor(() => {
      const btn = screen.getByRole('button', { name: 'Running assessment...' });
      expect(btn.disabled).toBe(true);
    });

    resolveCreate(mockAssessmentResult);

    await waitFor(() => {
      const btn = screen.getByRole('button', { name: 'Run Assessment' });
      expect(btn.disabled).toBe(false);
    });
  });

  it('handles API 4xx error', async () => {
    apiClient.createAssessment.mockRejectedValue(
      new apiClient.ApiError(400, 'VALIDATION_ERROR', 'applicant.id is required')
    );

    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      expect(screen.getByText('VALIDATION_ERROR: applicant.id is required')).toBeTruthy();
    });
  });

  it('handles API 5xx error', async () => {
    apiClient.createAssessment.mockRejectedValue(
      new apiClient.ApiError(500, 'EVALUATION_ERROR', 'assessment evaluation failed')
    );

    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      expect(screen.getByText('EVALUATION_ERROR: assessment evaluation failed')).toBeTruthy();
    });
  });

  it('handles network failure', async () => {
    apiClient.createAssessment.mockRejectedValue(new Error('Network error'));

    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      expect(screen.getByText('Network error')).toBeTruthy();
    });
  });

  it('renders the decision result from the backend', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      expect(screen.getByText('APPROVE')).toBeTruthy();
      expect(screen.getByText('test-assessment-123')).toBeTruthy();
      expect(screen.getByText('personal-loan:v1')).toBeTruthy();
    });
  });

  it('renders decision reasons from the backend', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      expect(screen.getByText('HIGH_DSR')).toBeTruthy();
      expect(screen.getByText('Debt service ratio exceeds policy threshold')).toBeTruthy();
      expect(screen.getByText('0.8 / threshold: 0.7')).toBeTruthy();
    });
  });

  it('renders credit limit and approved amount from the backend', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      expect(screen.getByText('Credit Limit')).toBeTruthy();
      expect(screen.getByText('Approved Amount')).toBeTruthy();
      expect(screen.getByText('Rp 20,000,000')).toBeTruthy();
      expect(screen.getByText('Rp 15,000,000')).toBeTruthy();
    });
  });

  it('fetches and displays evidence after assessment', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      expect(apiClient.getEvidenceByAssessmentId).toHaveBeenCalledWith('test-assessment-123');
      expect(screen.getByText('Evidence')).toBeTruthy();
      expect(screen.getByText('monthly_income')).toBeTruthy();
      expect(screen.getByText('monthly_obligations')).toBeTruthy();
    });
  });

  it('shows assessment ID and links to the detail page', async () => {
    render(h(NewAssessment));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    await runAssessment();

    await waitFor(() => {
      const link = screen.getByRole('link', { name: 'test-assessment-123' });
      expect(link.getAttribute('href')).toBe('/workspace/assessments/test-assessment-123');
      const detailLink = screen.getByRole('link', { name: 'View Full Details →' });
      expect(detailLink.getAttribute('href')).toBe('/workspace/assessments/test-assessment-123');
    });
  });
});
