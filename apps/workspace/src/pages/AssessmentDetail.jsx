import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { getAssessment, getEvidence, ApiError } from '../api/client.js';
import { formatCurrency, formatTime, outcomeBadgeClass, statusBadgeClass } from '../lib/format.js';
import { DecisioningCanvas } from '../components/canvas/DecisioningCanvas.jsx';

function formatError(err) {
  if (err instanceof ApiError) return `${err.code}: ${err.message}`;
  return err.message || 'Failed to load assessment';
}

// State maps an evidence value to a readable label. Backend evidence values
// are booleans recording whether a condition matched.
function stateLabel(value) {
  if (value === true) return 'matched';
  if (value === false) return 'not matched';
  return String(value);
}

export function AssessmentDetail({ id }) {
  const [assessment, setAssessment] = useState(null);
  const [evidence, setEvidence] = useState(null);
  const [evidenceError, setEvidenceError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [notFound, setNotFound] = useState(false);
  const [traceOpen, setTraceOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setError(null);
      setNotFound(false);
      setEvidence(null);
      setEvidenceError(null);
      try {
        const a = await getAssessment(id);
        if (cancelled) return;
        setAssessment(a);
        if (a.status === 'COMPLETED') {
          try {
            const ev = await getEvidence(a.id);
            if (!cancelled) setEvidence(ev);
          } catch (err) {
            if (!cancelled) setEvidenceError(formatError(err));
          }
        }
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 404) {
          setNotFound(true);
        } else {
          setError(formatError(err));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [id]);

  if (loading) return h('div', { class: 'loading' }, 'Loading assessment...');
  if (notFound) return h('div', { class: 'empty-state' }, 'Assessment not found.');
  if (error) return h('div', { class: 'error-msg mt-2' }, error);
  if (!assessment) return h('div', { class: 'empty-state' }, 'Assessment not found.');

  const d = assessment.decision;
  const reasons = d?.reasons || [];
  const traceKnockouts = (evidence || []).filter(e => e.source === 'knockout');
  const traceRules = (evidence || []).filter(e => e.source === 'rule');
  const hasTrace = traceKnockouts.length > 0 || traceRules.length > 0;

  // Reason-to-evidence relationship: a reason's evidence_ref matches the
  // evidence entry's condition field/reference.
  const reasonRefsFor = (e) => {
    if (!reasons.length) return [];
    return reasons
      .filter(r => r.evidence_ref && (r.evidence_ref === e.field || r.evidence_ref === e.reference))
      .map(r => r.code);
  };

  const renderReasons = () => {
    if (reasons.length === 0) {
      return h('div', { class: 'text-xs text-muted' }, 'No decision reasons were reported.');
    }
    return reasons.map((r, i) =>
      h('div', { key: i, class: 'card', style: 'padding:0.75rem;margin-bottom:0.5rem' },
        h('div', { class: 'flex-between' },
          h('code', { class: 'text-sm' }, r.code),
          r.value != null && h('span', { class: 'text-xs text-dim' },
            String(r.value), r.threshold != null ? ` / threshold: ${r.threshold}` : ''
          )
        ),
        r.description && h('div', { class: 'text-xs text-muted mt-1' }, r.description),
        r.evidence_ref && h('div', { class: 'text-xs text-dim mt-1' },
          'Evidence: ',
          h('code', null, r.evidence_ref)
        )
      )
    );
  };

  const renderOutputs = (outputs) => {
    if (!outputs) return null;
    return h('div', { class: 'mt-2' },
      h('div', { class: 'text-xs text-dim mb-1' }, 'Outputs'),
      h('div', { class: 'grid grid-2' },
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Credit Limit'),
          h('div', { class: 'text-sm font-mono' }, formatCurrency(outputs.credit_limit))
        ),
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Approved Amount'),
          h('div', { class: 'text-sm font-mono' }, formatCurrency(outputs.approved_amount))
        )
      )
    );
  };

  const renderEvidence = () => {
    if (!evidence || evidence.length === 0) return null;
    return h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Evidence'),
      h('div', { class: 'table-wrap' },
        h('table', null,
          h('thead', null,
            h('tr', null,
              h('th', null, 'Source'),
              h('th', null, 'Condition'),
              h('th', null, 'State'),
              h('th', null, 'Retrieved At'),
              h('th', null, 'Evidence For')
            )
          ),
          h('tbody', null,
            evidence.map((e, i) => {
              const refs = reasonRefsFor(e);
              return h('tr', { key: i },
                h('td', null, e.source),
                h('td', null, h('code', null, e.field)),
                h('td', null, stateLabel(e.value)),
                h('td', null, formatTime(e.retrieved_at)),
                h('td', null,
                  refs.length > 0
                    ? h('span', { class: 'flex gap-1', style: 'flex-wrap:wrap' },
                        refs.map(code => h('code', { key: code, class: 'text-xs' }, code))
                      )
                    : '\u2014'
                )
              );
            })
          )
        )
      )
    );
  };

  const renderTrace = () => {
    if (!hasTrace) return null;
    const renderEntries = (entries) => entries.map((e, i) =>
      h('div', { key: i, class: 'flex-between', style: 'padding:0.375rem 0;border-bottom:1px solid var(--border)' },
        h('code', { class: 'text-xs' }, e.field),
        h('span', { class: 'badge ' + (e.value === true ? 'badge-warning' : 'badge-neutral') }, stateLabel(e.value))
      )
    );
    return h('div', { class: 'card mb-2' },
      h('button', {
        type: 'button',
        class: 'btn',
        onClick: () => setTraceOpen(!traceOpen),
        style: 'width:100%;justify-content:space-between',
        'aria-expanded': traceOpen,
      },
        h('span', { class: 'text-sm' }, 'Evaluation Trace'),
        h('span', null, traceOpen ? '\u2212' : '+')
      ),
      traceOpen && h('div', { class: 'mt-2' },
        traceKnockouts.length > 0 && h('div', null,
          h('div', { class: 'text-xs text-dim mb-1' }, 'Knockouts'),
          renderEntries(traceKnockouts)
        ),
        traceRules.length > 0 && h('div', { class: 'mt-2' },
          h('div', { class: 'text-xs text-dim mb-1' }, 'Rules'),
          renderEntries(traceRules)
        ),
        h('div', { class: 'text-xs text-muted mt-2' },
          'Trace is derived from backend evidence. Evaluation values and thresholds for non-triggered conditions, and score-threshold evaluation, are not returned by the current API.'
        )
      )
    );
  };

  return h('div', null,
    h('div', { class: 'page-header' },
      h('h1', { class: 'page-title' }, 'Decision Explanation'),
      h('div', { class: 'text-xs text-dim mt-1' }, h('code', null, id))
    ),
    h('div', { class: 'mb-2' },
      h('div', { class: 'section-title' }, 'Decisioning Canvas'),
      h(DecisioningCanvas, { assessment, evidence })
    ),
    h('div', { class: 'card mb-2' },
      h('div', { class: 'flex-between mb-1' },
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Status'),
          h('span', { class: 'badge ' + statusBadgeClass(assessment.status) }, assessment.status)
        ),
        d && h('span', { class: 'badge ' + outcomeBadgeClass(d.outcome) }, d.outcome)
      ),
      h('div', { class: 'grid grid-3 mt-1' },
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Assessment ID'),
          h('div', { class: 'text-sm font-mono' }, assessment.id)
        ),
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Policy'),
          h('div', { class: 'text-sm font-mono' },
            assessment.policy?.id, ':v', assessment.policy?.version
          )
        ),
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Created'),
          h('div', { class: 'text-sm' }, formatTime(assessment.created_at))
        )
      ),
      h('div', { class: 'grid grid-3 mt-2' },
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Applicant'),
          h('div', { class: 'text-sm' },
            assessment.applicant?.name || '\u2014',
            assessment.applicant?.age != null && h('span', { class: 'text-dim' }, ` (age ${assessment.applicant.age})`)
          )
        ),
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Started'),
          h('div', { class: 'text-sm' }, formatTime(assessment.started_at))
        ),
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Completed'),
          h('div', { class: 'text-sm' }, formatTime(assessment.completed_at))
        )
      ),
      assessment.application && h('div', { class: 'mt-2' },
        h('div', { class: 'text-xs text-dim mb-1' }, 'Application'),
        h('div', { class: 'grid grid-3' },
          h('div', null,
            h('div', { class: 'text-xs text-dim' }, 'ID'),
            h('div', { class: 'text-sm font-mono' }, assessment.application.id)
          ),
          h('div', null,
            h('div', { class: 'text-xs text-dim' }, 'Requested Amount'),
            h('div', { class: 'text-sm font-mono' },
              assessment.application.requested_amount != null
                ? formatCurrency(assessment.application.requested_amount)
                : '\u2014'
            )
          ),
          h('div', null,
            h('div', { class: 'text-xs text-dim' }, 'Purpose'),
            h('div', { class: 'text-sm' }, assessment.application.purpose || '\u2014')
          )
        )
      ),
      assessment.score && h('div', { class: 'mt-2' },
        h('div', { class: 'text-xs text-dim mb-1' }, 'Credit Score'),
        h('div', { class: 'grid grid-2' },
          h('div', null,
            h('div', { class: 'text-xs text-dim' }, 'Value'),
            h('div', { class: 'text-sm font-mono' }, assessment.score.value)
          ),
          h('div', null,
            h('div', { class: 'text-xs text-dim' }, 'Provider'),
            h('div', { class: 'text-sm' }, assessment.score.provider)
          )
        )
      )
    ),
    d && h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Decision'),
      h('div', { class: 'decision-outcome decision-outcome-' + d.outcome }, d.outcome),
      h('div', { class: 'text-xs text-dim mt-1' },
        'Policy: ',
        h('code', null, d.policy?.id, ':v', d.policy?.version)
      ),
      renderOutputs(d.outputs),
      h('div', { class: 'mt-2' },
        h('div', { class: 'text-xs text-dim mb-1' }, 'Reasons'),
        renderReasons()
      )
    ),
    !d && h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Decision'),
      h('div', { class: 'text-xs text-muted' }, 'No decision is available for this assessment yet.')
    ),
    renderEvidence(),
    evidenceError && h('div', { class: 'error-msg mt-2' }, 'Evidence: ' + evidenceError),
    renderTrace(),
    assessment.error && h('div', { class: 'error-msg mt-2' },
      'Error: ', assessment.error
    )
  );
}
