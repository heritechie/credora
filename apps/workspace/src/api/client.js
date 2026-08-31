const BASE_URL = import.meta.env.VITE_API_URL || '';

async function request(path, options = {}) {
  const url = `${BASE_URL}${path}`;
  const res = await fetch(url, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => null);
    const message = body?.error || `HTTP ${res.status}`;
    throw new ApiError(res.status, body?.code, message);
  }

  return res.json();
}

export class ApiError extends Error {
  constructor(status, code, message) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

export async function createAssessment(data) {
  return request('/api/v1/assessments', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function getAssessment(id) {
  return request(`/api/v1/assessments/${encodeURIComponent(id)}`);
}

export async function getAssessments() {
  return request('/api/v1/assessments');
}

export async function getDecision(id) {
  return request(`/api/v1/assessments/${encodeURIComponent(id)}/decision`);
}

export async function getEvidence(id) {
  return request(`/api/v1/assessments/${encodeURIComponent(id)}/evidence`);
}

export async function healthCheck() {
  return request('/health');
}

export async function getPolicies() {
  return request('/api/v1/policies');
}

export async function getEvidenceByAssessmentId(id) {
  return request(`/api/v1/assessments/${encodeURIComponent(id)}/evidence`);
}
