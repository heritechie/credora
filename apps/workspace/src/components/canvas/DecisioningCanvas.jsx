import { h } from 'preact';
import { useMemo } from 'preact/hooks';
import { mapAssessmentToStages, STAGE_TYPES } from '../../lib/assessmentGraph.js';

/**
 * DecisioningCanvas renders a static, horizontal explanation flow for an
 * executed assessment:
 *
 *   Applicant → Evaluated Checks → Policy → Decision
 *
 * This is intentionally a NON-interactive, diagram-like view. It is an
 * explanation of how the assessment reached its decision, not a workflow
 * builder or execution trace. It only reads the assessment response + evidence
 * the page already loaded and displays them truthfully.
 *
 * No backend data beyond what the API exposes is represented, and no
 * provider execution is fabricated.
 */

const TYPE_LABELS = {
  [STAGE_TYPES.INPUT]: 'Input',
  [STAGE_TYPES.CHECKS]: 'Checks',
  [STAGE_TYPES.POLICY]: 'Policy',
  [STAGE_TYPES.DECISION]: 'Decision',
};

function decisionModifier(outcome) {
  switch (outcome) {
    case 'APPROVE': return 'stage-decision-approve';
    case 'REVIEW': return 'stage-decision-review';
    case 'REJECT': return 'stage-decision-reject';
    default: return '';
  }
}

export function DecisioningCanvas({ assessment, evidence }) {
  const stages = useMemo(
    () => mapAssessmentToStages(assessment, evidence),
    [assessment, evidence]
  );

  if (stages.length === 0) {
    return h('div', { class: 'canvas-empty' }, 'Nothing to visualize for this assessment.');
  }

  return h('div', { class: 'decision-flow' },
    h('div', { class: 'decision-flow-scroll' },
      stages.map((stage, i) => {
        const isDecision = stage.type === STAGE_TYPES.DECISION;
        return h('div', { key: stage.id, class: 'decision-flow-item' },
          h('div', {
            class: [
              'flow-stage',
              'flow-stage-' + stage.type,
              isDecision ? decisionModifier(stage.outcome) : '',
            ].filter(Boolean).join(' '),
            'data-stage': stage.type,
            'data-testid': `flow-stage-${stage.type}`,
          },
            h('div', { class: 'flow-stage-type' }, TYPE_LABELS[stage.type] || stage.type),
            isDecision
              ? h('div', { class: 'flow-stage-decision-value', 'data-testid': 'flow-stage-decision-value' },
                  stage.outcome || '\u2014')
              : h('div', { class: 'flow-stage-title' }, stage.title),
            !isDecision && stage.lines && stage.lines.length > 0
              ? h('div', { class: 'flow-stage-lines' },
                  stage.lines.map((line, j) => h('div', { key: j, class: 'flow-stage-line' }, line))
                )
              : null
          ),
          i < stages.length - 1
            ? h('div', { class: 'flow-arrow', 'aria-hidden': 'true' }, '\u2192')
            : null
        );
      })
    ),
    h('div', { class: 'decision-flow-hint' },
      'Explanation view based on the assessment result \u2014 not an execution trace.'
    )
  );
}
