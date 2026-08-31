import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { h } from 'preact';
import { render, screen, waitFor } from '@testing-library/preact';
import { Home } from '../Home.jsx';
import * as apiClient from '../../api/client.js';

vi.mock('../../api/client.js');

describe('Home (Decisioning Workspace)', () => {
  const mockPolicies = {
    items: [
      { id: 'personal-loan', version: 1, description: 'Deterministic personal loan decision policy', status: 'active' },
    ],
  };

  const mockAssessments = {
    items: [
      { id: 'assessment-1', status: 'COMPLETED', policy: { id: 'personal-loan', version: 1 }, decision: { outcome: 'APPROVE' }, created_at: '2025-01-15T10:30:00Z' },
      { id: 'assessment-2', status: 'COMPLETED', policy: { id: 'personal-loan', version: 1 }, decision: { outcome: 'REJECT' }, created_at: '2025-01-15T11:30:00Z' },
      { id: 'assessment-3', status: 'FAILED', policy: { id: 'personal-loan', version: 1 }, decision: null, created_at: '2025-01-15T12:30:00Z' },
    ],
  };

  beforeEach(() => {
    vi.clearAllMocks();
    apiClient.getPolicies.mockResolvedValue(mockPolicies);
    apiClient.getAssessments.mockResolvedValue(mockAssessments);
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

  it('renders the workspace header and decisioning workflow steps', async () => {
    render(h(Home));
    expect(screen.getByText('Credora Decisioning Workspace')).toBeTruthy();
    expect(screen.getByRole('link', { name: /Configure a Policy/ })).toBeTruthy();
    expect(screen.getAllByRole('link', { name: /Run an Assessment/ }).length).toBeGreaterThan(0);
    expect(screen.getByRole('link', { name: /Explain the Decision/ })).toBeTruthy();
  });

  it('links workflow steps to the corresponding pages', async () => {
    render(h(Home));
    expect(screen.getByRole('link', { name: /Configure a Policy/ }).getAttribute('href')).toBe('/workspace/policies');
    expect(screen.getByRole('link', { name: /Explain the Decision/ }).getAttribute('href')).toBe('/workspace/assessments');
  });

  it('provides a primary action that opens the simulator', async () => {
    render(h(Home));
    const cta = screen.getByRole('link', { name: 'Run an Assessment' });
    expect(cta.getAttribute('href')).toBe('/workspace/assessments/new');
  });

  it('loads registered policies from the API and renders them', async () => {
    render(h(Home));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
    expect(screen.getByText('Registered Policies')).toBeTruthy();
    expect(screen.getAllByText('personal-loan:v1').length).toBeGreaterThan(0);
  });

  it('loads recent assessments from the API with decision outcomes', async () => {
    render(h(Home));
    await waitFor(() => {
      expect(apiClient.getAssessments).toHaveBeenCalled();
    });
    expect(screen.getByText('Recent Assessments')).toBeTruthy();
    expect(screen.getByText('assessment-1')).toBeTruthy();
    expect(screen.getByText('assessment-2')).toBeTruthy();
    expect(screen.getByText('APPROVE')).toBeTruthy();
    expect(screen.getByText('REJECT')).toBeTruthy();
    expect(screen.getByText('FAILED')).toBeTruthy();
  });

  it('links each recent assessment to its decision explanation page', async () => {
    render(h(Home));
    await waitFor(() => {
      expect(screen.getByText('assessment-1')).toBeTruthy();
    });
    const link = screen.getByRole('link', { name: /assessment-1/ });
    expect(link.getAttribute('href')).toBe('/workspace/assessments/assessment-1');
  });

  it('handles an empty policy registry', async () => {
    apiClient.getPolicies.mockResolvedValue({ items: [] });
    render(h(Home));
    await waitFor(() => {
      expect(screen.getByText('No policies registered.')).toBeTruthy();
    });
  });

  it('handles an empty assessments list', async () => {
    apiClient.getAssessments.mockResolvedValue({ items: [] });
    render(h(Home));
    await waitFor(() => {
      expect(screen.getByText(/No assessments yet/)).toBeTruthy();
    });
  });

  it('keeps the workflow visible when the policy API fails', async () => {
    apiClient.getPolicies.mockRejectedValue(
      new apiClient.ApiError(500, 'REPOSITORY_ERROR', 'failed to retrieve policies')
    );
    render(h(Home));
    await waitFor(() => {
      expect(screen.getByText('REPOSITORY_ERROR: failed to retrieve policies')).toBeTruthy();
    });
    expect(screen.getByRole('link', { name: /Configure a Policy/ })).toBeTruthy();
  });

  it('keeps the workflow visible when the assessments API fails', async () => {
    apiClient.getAssessments.mockRejectedValue(
      new apiClient.ApiError(500, 'REPOSITORY_ERROR', 'failed to retrieve assessments')
    );
    render(h(Home));
    await waitFor(() => {
      expect(screen.getByText('REPOSITORY_ERROR: failed to retrieve assessments')).toBeTruthy();
    });
    expect(screen.getByRole('link', { name: /Explain the Decision/ })).toBeTruthy();
  });

  it('handles network failures without stack traces', async () => {
    apiClient.getPolicies.mockRejectedValue(new Error('Network error'));
    apiClient.getAssessments.mockRejectedValue(new Error('Network error'));
    render(h(Home));
    await waitFor(() => {
      expect(screen.getAllByText('Network error').length).toBeGreaterThan(0);
    });
    expect(screen.getAllByRole('link', { name: /Run an Assessment/ }).length).toBeGreaterThan(0);
  });
});
