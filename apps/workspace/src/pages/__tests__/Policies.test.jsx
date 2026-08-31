import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { h } from 'preact';
import { render, screen, waitFor } from '@testing-library/preact';
import { Policies } from '../Policies.jsx';
import * as apiClient from '../../api/client.js';

vi.mock('../../api/client.js');

describe('Policies', () => {
  const mockPolicies = {
    items: [
      { id: 'personal-loan', version: 1, description: 'Deterministic personal loan decision policy', status: 'active' },
      { id: 'business-loan', version: 2, description: 'Business loan policy', status: 'active' },
    ],
  };

  beforeEach(() => {
    vi.clearAllMocks();
    apiClient.getPolicies.mockResolvedValue(mockPolicies);
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

  it('loads policies from the API on mount', async () => {
    render(h(Policies));
    await waitFor(() => {
      expect(apiClient.getPolicies).toHaveBeenCalled();
    });
  });

  it('renders policy ID, version, description, and status from API data', async () => {
    render(h(Policies));
    await waitFor(() => {
      expect(screen.getByText('personal-loan')).toBeTruthy();
      expect(screen.getByText('business-loan')).toBeTruthy();
      expect(screen.getByText('v1')).toBeTruthy();
      expect(screen.getByText('v2')).toBeTruthy();
    });
    expect(screen.getByText('Deterministic personal loan decision policy')).toBeTruthy();
    expect(screen.getByText('Business loan policy')).toBeTruthy();
    expect(screen.getAllByText('active').length).toBe(2);
  });

  it('shows a loading state while the request is pending', async () => {
    let resolveLoad;
    apiClient.getPolicies.mockImplementation(() => new Promise(resolve => { resolveLoad = resolve; }));

    render(h(Policies));
    expect(screen.getByText('Loading policies...')).toBeTruthy();

    resolveLoad(mockPolicies);
    await waitFor(() => {
      expect(screen.getByText('personal-loan')).toBeTruthy();
    });
  });

  it('handles an empty policy list', async () => {
    apiClient.getPolicies.mockResolvedValue({ items: [] });

    render(h(Policies));
    await waitFor(() => {
      expect(screen.getByText('No policies are registered.')).toBeTruthy();
    });
  });

  it('handles a response without an items list', async () => {
    apiClient.getPolicies.mockResolvedValue({});

    render(h(Policies));
    await waitFor(() => {
      expect(screen.getByText('No policies are registered.')).toBeTruthy();
    });
  });

  it('handles API errors without exposing stack traces', async () => {
    apiClient.getPolicies.mockRejectedValue(
      new apiClient.ApiError(500, 'REPOSITORY_ERROR', 'failed to retrieve policies')
    );

    render(h(Policies));
    await waitFor(() => {
      expect(screen.getByText('REPOSITORY_ERROR: failed to retrieve policies')).toBeTruthy();
    });
  });

  it('handles network failures without exposing stack traces', async () => {
    apiClient.getPolicies.mockRejectedValue(new Error('Network error'));

    render(h(Policies));
    await waitFor(() => {
      expect(screen.getByText('Network error')).toBeTruthy();
    });
  });
});
