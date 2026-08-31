import { describe, it, expect } from 'vitest';
import { createAssessment, getAssessment, getAssessments, ApiError } from '../client.js';

describe('API client', () => {
  it('ApiError carries status and code', () => {
    const err = new ApiError(404, 'NOT_FOUND', 'not found');
    expect(err.status).toBe(404);
    expect(err.code).toBe('NOT_FOUND');
    expect(err.message).toBe('not found');
    expect(err.name).toBe('ApiError');
  });

  it('createAssessment is a function', () => {
    expect(typeof createAssessment).toBe('function');
  });

  it('getAssessment is a function', () => {
    expect(typeof getAssessment).toBe('function');
  });

  it('getAssessments is a function', () => {
    expect(typeof getAssessments).toBe('function');
  });
});
