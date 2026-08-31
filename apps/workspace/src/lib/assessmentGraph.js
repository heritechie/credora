/**
 * assessmentGraph
 *
 * Maps an executed assessment (assessment response + evidence) into a small,
 * deterministic, horizontal flow of stages for the Decisioning Canvas.
 *
 * IMPORTANT: This module is a PRESENTATION layer only. It does not perform or
 * reproduce any backend decisioning logic. It reads the decision outcome,
 * policy, and evidence directly from the API response and arranges them into a
 * readable, explanatory layout.
 *
 * The result is an EXPLANATION VIEW, not an execution trace. The backend does
 * not currently expose true execution ordering, parallelism, retries, or
 * provider execution, so the flow is static and must not be read as a record
 * of how the assessment actually executed.
 *
 * Truthfulness:
 *  - Only stages for which the API provides data are produced.
 *  - The applicant response exposes id/name/age (no income is available), so
 *    no income is shown.
 *  - Evidence only reports knockout/rule check results, so the middle stage is
 *    "Evaluated Checks"; no provider nodes (KYC/FDC/PEFINDO/Fraud) are invented.
 */

export const STAGE_TYPES = {
  INPUT: 'input',
  CHECKS: 'checks',
  POLICY: 'policy',
  DECISION: 'decision',
};

/**
 * Build the compact display lines for a stage, sourced only from backend data.
 * Keep each stage to a small number of lines so the canvas stays readable.
 */
function buildLines(stageType, assessment, evidence) {
  const lines = [];

  switch (stageType) {
    case STAGE_TYPES.INPUT: {
      const applicant = assessment.applicant || {};
      if (applicant.name) lines.push(applicant.name);
      if (applicant.age != null) lines.push(`Age ${applicant.age}`);
      // The assessment response may carry an externally provided score.
      if (assessment.score && assessment.score.value != null) {
        lines.push(`Score ${assessment.score.value}`);
      }
      break;
    }
    case STAGE_TYPES.CHECKS: {
      const n = Array.isArray(evidence) ? evidence.length : 0;
      lines.push(`${n} condition${n === 1 ? '' : 's'} evaluated`);
      break;
    }
    case STAGE_TYPES.POLICY: {
      const p = assessment.policy || {};
      const id = p.id || '\u2014';
      const version = p.version != null ? `v${p.version}` : '';
      lines.push(`${id}${version ? ':' + version : ''}`);
      break;
    }
    case STAGE_TYPES.DECISION: {
      const d = assessment.decision;
      if (d && d.outcome) lines.push(d.outcome);
      break;
    }
    default:
      break;
  }

  return lines;
}

/**
 * Map an assessment + evidence list into a horizontal flow of stages.
 * Returns { stages }. Deterministic ordering; returns an empty array when
 * there is no meaningful data to display.
 */
export function mapAssessmentToStages(assessment, evidence) {
  if (!assessment) return [];

  const applicant = assessment.applicant;
  const hasApplicant = !!(applicant && (applicant.id || applicant.name || applicant.age != null));
  const hasScore = !!(assessment.score && assessment.score.value != null);
  const hasEvidence = Array.isArray(evidence) && evidence.length > 0;
  const hasPolicy = !!(assessment.policy && (assessment.policy.id || assessment.policy.version));
  const decision = assessment.decision;
  const hasDecision = !!decision;

  const stages = [];

  if (hasApplicant || hasScore) {
    stages.push({
      id: STAGE_TYPES.INPUT,
      type: STAGE_TYPES.INPUT,
      title: applicant && applicant.name ? 'Applicant' : 'Inputs',
      lines: buildLines(STAGE_TYPES.INPUT, assessment, evidence),
    });
  }

  if (hasEvidence) {
    stages.push({
      id: STAGE_TYPES.CHECKS,
      type: STAGE_TYPES.CHECKS,
      title: 'Evaluated Checks',
      lines: buildLines(STAGE_TYPES.CHECKS, assessment, evidence),
    });
  }

  if (hasPolicy || hasDecision) {
    stages.push({
      id: STAGE_TYPES.POLICY,
      type: STAGE_TYPES.POLICY,
      title: 'Policy',
      lines: buildLines(STAGE_TYPES.POLICY, assessment, evidence),
    });
  }

  if (hasDecision) {
    const outcome = (decision.outcome || '').toUpperCase();
    stages.push({
      id: STAGE_TYPES.DECISION,
      type: STAGE_TYPES.DECISION,
      title: 'Decision',
      outcome,
      lines: buildLines(STAGE_TYPES.DECISION, assessment, evidence),
    });
  }

  return stages;
}
