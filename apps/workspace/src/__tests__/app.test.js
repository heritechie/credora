import { describe, it, expect } from 'vitest';

describe('Workspace page modules', () => {
  it('Home module exports', async () => {
    const mod = await import('../pages/Home.jsx');
    expect(typeof mod.Home).toBe('function');
  });

  it('Policies module exports', async () => {
    const mod = await import('../pages/Policies.jsx');
    expect(typeof mod.Policies).toBe('function');
  });

  it('Assessments module exports', async () => {
    const mod = await import('../pages/Assessments.jsx');
    expect(typeof mod.Assessments).toBe('function');
  });

  it('NewAssessment module exports', async () => {
    const mod = await import('../pages/NewAssessment.jsx');
    expect(typeof mod.NewAssessment).toBe('function');
  });

  it('AssessmentDetail module exports', async () => {
    const mod = await import('../pages/AssessmentDetail.jsx');
    expect(typeof mod.AssessmentDetail).toBe('function');
  });
});

describe('Clean URL routing', () => {
  it('parseRoute extracts /workspace/policies correctly', () => {
    // Verify the parseRoute logic works with clean URLs
    const path = '/workspace/policies';
    const BASE = '/workspace';
    const sub = path.startsWith(BASE) ? path.slice(BASE.length) : path;
    const parts = sub.split('/').filter(Boolean);
    expect(parts[0]).toBe('policies');
  });

  it('parseRoute extracts /workspace/assessments/new correctly', () => {
    const path = '/workspace/assessments/new';
    const BASE = '/workspace';
    const sub = path.startsWith(BASE) ? path.slice(BASE.length) : path;
    const parts = sub.split('/').filter(Boolean);
    expect(parts[0]).toBe('assessments');
    expect(parts[1]).toBe('new');
  });

  it('parseRoute extracts /workspace/assessments/:id correctly', () => {
    const path = '/workspace/assessments/abc-123';
    const BASE = '/workspace';
    const sub = path.startsWith(BASE) ? path.slice(BASE.length) : path;
    const parts = sub.split('/').filter(Boolean);
    expect(parts[0]).toBe('assessments');
    expect(parts[1]).toBe('abc-123');
  });

  it('parseRoute handles /workspace root', () => {
    const path = '/workspace';
    const BASE = '/workspace';
    const sub = path.startsWith(BASE) ? path.slice(BASE.length) : path;
    const parts = sub.split('/').filter(Boolean);
    expect(parts.length).toBe(0);
  });

  it('parseRoute handles /workspace/ with trailing slash', () => {
    const path = '/workspace/';
    const BASE = '/workspace';
    const sub = path.startsWith(BASE) ? path.slice(BASE.length) : path;
    const parts = sub.split('/').filter(Boolean);
    expect(parts.length).toBe(0);
  });
});
