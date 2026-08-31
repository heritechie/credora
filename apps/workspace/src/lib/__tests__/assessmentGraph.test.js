import { describe, it, expect } from 'vitest';
import { mapAssessmentToStages, STAGE_TYPES } from '../assessmentGraph.js';

describe('assessmentGraph mapAssessmentToStages', () => {
  const baseAssessment = {
    id: 'assessment-1',
    status: 'COMPLETED',
    applicant: { id: 'a-1', name: 'Jane Doe', age: 30 },
    policy: { id: 'personal-loan', version: 1 },
    created_at: '2025-01-15T10:30:00Z',
    decision: { outcome: 'APPROVE', reasons: [], outputs: { credit_limit: 20000000 } },
  };

  const evidence = [
    { source: 'knockout', field: 'HIGH_DSR', value: false, reference: 'HIGH_DSR', retrieved_at: '2025-01-15T10:30:01Z' },
    { source: 'rule', field: 'LOW_CREDIT_SCORE', value: false, reference: 'LOW_CREDIT_SCORE', retrieved_at: '2025-01-15T10:30:01Z' },
  ];

  it('returns an empty array for a null assessment', () => {
    expect(mapAssessmentToStages(null, [])).toEqual([]);
  });

  it('produces the horizontal flow order: input -> checks -> policy -> decision', () => {
    const stages = mapAssessmentToStages(baseAssessment, evidence);
    const order = stages.map((s) => s.type);
    expect(order).toEqual([STAGE_TYPES.INPUT, STAGE_TYPES.CHECKS, STAGE_TYPES.POLICY, STAGE_TYPES.DECISION]);
  });

  it('labels the input stage Applicant when a name is present', () => {
    const stages = mapAssessmentToStages(baseAssessment, []);
    const input = stages.find((s) => s.type === STAGE_TYPES.INPUT);
    expect(input.title).toBe('Applicant');
    expect(input.lines).toContain('Jane Doe');
    expect(input.lines).toContain('Age 30');
  });

  it('does not fabricate an income line (not exposed by the API)', () => {
    const stages = mapAssessmentToStages(baseAssessment, []);
    const input = stages.find((s) => s.type === STAGE_TYPES.INPUT);
    expect(input.lines.some((l) => /income/i.test(l))).toBe(false);
    expect(input.lines.some((l) => /Rp/i.test(l))).toBe(false);
  });

  it('includes the backend score value as a line when present', () => {
    const withScore = { ...baseAssessment, score: { value: 720, provider: 'mock-credit-bureau' } };
    const stages = mapAssessmentToStages(withScore, []);
    const input = stages.find((s) => s.type === STAGE_TYPES.INPUT);
    expect(input.lines).toContain('Score 720');
  });

  it('counts conditions evaluated from the evidence list', () => {
    const stages = mapAssessmentToStages(baseAssessment, evidence);
    const checks = stages.find((s) => s.type === STAGE_TYPES.CHECKS);
    expect(checks.title).toBe('Evaluated Checks');
    expect(checks.lines).toContain('2 conditions evaluated');
  });

  it('produces a single "Evaluated Checks" stage, not provider stages', () => {
    const stages = mapAssessmentToStages(baseAssessment, evidence);
    const types = stages.map((s) => s.type);
    expect(types.filter((t) => t === STAGE_TYPES.CHECKS).length).toBe(1);
    expect(types).not.toContain('provider');
    expect(types).not.toContain('kyc');
    expect(types).not.toContain('pefindo');
    expect(types).not.toContain('fdc');
    expect(types).not.toContain('fraud');
  });

  it('omits the checks stage when there is no evidence', () => {
    const stages = mapAssessmentToStages(baseAssessment, []);
    expect(stages.some((s) => s.type === STAGE_TYPES.CHECKS)).toBe(false);
    // Flow collapses to input -> policy -> decision.
    expect(stages.map((s) => s.type)).toEqual([STAGE_TYPES.INPUT, STAGE_TYPES.POLICY, STAGE_TYPES.DECISION]);
  });

  it('renders policy as id:version', () => {
    const stages = mapAssessmentToStages(baseAssessment, []);
    const policy = stages.find((s) => s.type === STAGE_TYPES.POLICY);
    expect(policy.lines).toContain('personal-loan:v1');
  });

  it('renders the decision outcome directly from the backend', () => {
    const stages = mapAssessmentToStages(baseAssessment, []);
    const decision = stages.find((s) => s.type === STAGE_TYPES.DECISION);
    expect(decision.outcome).toBe('APPROVE');
    expect(decision.lines).toContain('APPROVE');
  });

  it('does not create a decision stage when there is no decision', () => {
    const noDecision = { ...baseAssessment, decision: null };
    const stages = mapAssessmentToStages(noDecision, []);
    expect(stages.some((s) => s.type === STAGE_TYPES.DECISION)).toBe(false);
  });

  it('is tolerant of missing/optional evidence (null or undefined)', () => {
    const stages = mapAssessmentToStages(baseAssessment, null);
    expect(stages.some((s) => s.type === STAGE_TYPES.CHECKS)).toBe(false);
    expect(stages.map((s) => s.type)).toEqual([STAGE_TYPES.INPUT, STAGE_TYPES.POLICY, STAGE_TYPES.DECISION]);
  });

  it('handles an absent applicant gracefully', () => {
    const stages = mapAssessmentToStages({ ...baseAssessment, applicant: {} }, []);
    // Without applicant or score, the input stage is omitted.
    expect(stages.some((s) => s.type === STAGE_TYPES.INPUT)).toBe(false);
  });
});
