import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { createAssessment, getPolicies, getEvidenceByAssessmentId, ApiError } from '../api/client.js';
import { formatCurrency, outcomeBadgeClass, formatTime } from '../lib/format.js';

export function NewAssessment() {
  const [policies, setPolicies] = useState([]);
  const [policyId, setPolicyId] = useState('personal-loan');
  const [policyVersion, setPolicyVersion] = useState(1);

  const [applicantAge, setApplicantAge] = useState('30');

  const [monthlyIncome, setMonthlyIncome] = useState('10000000');
  const [monthlyObligations, setMonthlyObligations] = useState('3000000');

  const [creditScore, setCreditScore] = useState('720');
  const [scoreProvider, setScoreProvider] = useState('mock-credit-bureau');

  const [requestedAmount, setRequestedAmount] = useState('');
  const [appPurpose, setAppPurpose] = useState('');

  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);
  const [evidence, setEvidence] = useState(null);
  const [error, setError] = useState(null);
  const [validationErrors, setValidationErrors] = useState({});

  useEffect(() => {
    let cancelled = false;
    async function loadPolicies() {
      try {
        const data = await getPolicies();
        if (cancelled || !data.items || data.items.length === 0) return;
        setPolicies(data.items);
        const defaultPolicy = data.items.find(p => p.id === 'personal-loan') || data.items[0];
        setPolicyId(defaultPolicy.id);
        setPolicyVersion(defaultPolicy.version || 1);
      } catch (err) {
        console.warn('Failed to load policies:', err.message);
      }
    }
    loadPolicies();
    return () => { cancelled = true; };
  }, []);

  const parseOptionalAmount = (value) => {
    if (!value.trim()) return null;
    const n = Number(value);
    return isNaN(n) ? NaN : n;
  };

  const validateForm = () => {
    const errors = {};

    const age = Number(applicantAge);
    if (!applicantAge.trim()) {
      errors.applicantAge = 'Age is required';
    } else if (!Number.isInteger(age) || age < 0 || age > 150) {
      errors.applicantAge = 'Age must be a number between 0 and 150';
    }

    const income = Number(monthlyIncome);
    if (!monthlyIncome.trim()) {
      errors.monthlyIncome = 'Monthly income is required';
    } else if (!Number.isInteger(income) || income < 0) {
      errors.monthlyIncome = 'Monthly income must be a non-negative number';
    }

    const obligations = Number(monthlyObligations);
    if (!monthlyObligations.trim()) {
      errors.monthlyObligations = 'Monthly obligations is required';
    } else if (!Number.isInteger(obligations) || obligations < 0) {
      errors.monthlyObligations = 'Monthly obligations must be a non-negative number';
    }

    const score = Number(creditScore);
    if (!creditScore.trim()) {
      errors.creditScore = 'Credit score is required';
    } else if (!Number.isInteger(score) || score < 0 || score > 1000) {
      errors.creditScore = 'Credit score must be a number between 0 and 1000';
    }

    if (!scoreProvider.trim()) {
      errors.scoreProvider = 'Score provider is required';
    }

    if (requestedAmount.trim()) {
      const amount = parseOptionalAmount(requestedAmount);
      if (isNaN(amount) || !Number.isInteger(amount) || amount < 0) {
        errors.requestedAmount = 'Requested amount must be a non-negative number';
      }
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!validateForm()) return;

    setLoading(true);
    setError(null);
    setResult(null);
    setEvidence(null);

    const body = {
      applicant: {
        id: `applicant-${Date.now()}`,
        name: 'Simulator',
        age: parseInt(applicantAge, 10),
      },
      monthly_income: parseInt(monthlyIncome, 10),
      monthly_obligations: parseInt(monthlyObligations, 10),
      score: {
        value: parseInt(creditScore, 10),
        provider: scoreProvider.trim(),
      },
      policy: {
        id: policyId,
        version: parseInt(policyVersion, 10),
      },
    };

    const requestedAmountValue = parseOptionalAmount(requestedAmount);
    if (requestedAmountValue !== null || appPurpose.trim()) {
      body.application = {
        id: `app-${Date.now()}`,
        purpose: appPurpose.trim(),
      };
      if (requestedAmountValue !== null) {
        body.application.requested_amount = requestedAmountValue;
      }
    }

    try {
      const assessment = await createAssessment(body);
      setResult(assessment);
      const ev = await getEvidenceByAssessmentId(assessment.id);
      setEvidence(ev);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(`${err.code}: ${err.message}`);
      } else {
        setError(err.message || 'Failed to run assessment');
      }
    } finally {
      setLoading(false);
    }
  };

  const getSelectedPolicy = () => {
    return policies.find(p => p.id === policyId && p.version === parseInt(policyVersion, 10));
  };

  const renderPolicySelector = () => {
    const selectedPolicy = getSelectedPolicy();
    return h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Policy'),
      h('div', { class: 'form-row' },
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label form-label-required', for: 'policy-id' }, 'Policy'),
          h('select', {
            id: 'policy-id',
            class: 'form-input',
            value: policyId,
            onChange: (e) => {
              const newPolicyId = e.target.value;
              setPolicyId(newPolicyId);
              const policy = policies.find(p => p.id === newPolicyId);
              if (policy) {
                setPolicyVersion(policy.version || 1);
              }
            },
            required: true,
          },
            policies.map(p =>
              h('option', { key: `${p.id}-v${p.version}`, value: p.id }, `${p.id} (v${p.version})`)
            )
          )
        ),
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label form-label-required', for: 'policy-version' }, 'Version'),
          h('input', {
            id: 'policy-version',
            class: 'form-input',
            type: 'number',
            min: '1',
            value: policyVersion,
            onInput: (e) => setPolicyVersion(e.target.value),
            required: true,
            readOnly: true,
          })
        )
      ),
      selectedPolicy && h('div', { class: 'text-xs text-muted mt-1' },
        selectedPolicy.description || 'No description available'
      ),
      policies.length === 0 && h('div', { class: 'text-xs text-muted mt-1' },
        'Loading policies...'
      )
    );
  };

  const renderApplicant = () => {
    return h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Applicant'),
      h('div', { class: 'form-row' },
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label form-label-required', for: 'applicant-age' }, 'Age'),
          h('input', {
            id: 'applicant-age',
            class: 'form-input',
            type: 'number',
            min: '0',
            max: '150',
            placeholder: '30',
            value: applicantAge,
            onInput: (e) => setApplicantAge(e.target.value),
            required: true,
            style: 'max-width:120px',
          }),
          validationErrors.applicantAge && h('div', { class: 'text-xs text-danger mt-1' }, validationErrors.applicantAge)
        )
      )
    );
  };

  const renderFinancial = () => {
    return h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Financial'),
      h('div', { class: 'form-row' },
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label form-label-required', for: 'monthly-income' }, 'Monthly Income'),
          h('input', {
            id: 'monthly-income',
            class: 'form-input',
            type: 'number',
            min: '0',
            placeholder: '10000000',
            value: monthlyIncome,
            onInput: (e) => setMonthlyIncome(e.target.value),
            required: true,
          }),
          h('div', { class: 'form-hint' }, 'Smallest currency unit (e.g., sen, cents)'),
          validationErrors.monthlyIncome && h('div', { class: 'text-xs text-danger mt-1' }, validationErrors.monthlyIncome)
        ),
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label form-label-required', for: 'monthly-obligations' }, 'Monthly Obligations'),
          h('input', {
            id: 'monthly-obligations',
            class: 'form-input',
            type: 'number',
            min: '0',
            placeholder: '3000000',
            value: monthlyObligations,
            onInput: (e) => setMonthlyObligations(e.target.value),
            required: true,
          }),
          h('div', { class: 'form-hint' }, 'Smallest currency unit (e.g., sen, cents)'),
          validationErrors.monthlyObligations && h('div', { class: 'text-xs text-danger mt-1' }, validationErrors.monthlyObligations)
        )
      )
    );
  };

  const renderCredit = () => {
    return h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Credit'),
      h('div', { class: 'form-row' },
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label form-label-required', for: 'credit-score' }, 'Credit Score'),
          h('input', {
            id: 'credit-score',
            class: 'form-input',
            type: 'number',
            min: '0',
            max: '1000',
            placeholder: '720',
            value: creditScore,
            onInput: (e) => setCreditScore(e.target.value),
            required: true,
          }),
          validationErrors.creditScore && h('div', { class: 'text-xs text-danger mt-1' }, validationErrors.creditScore)
        ),
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label form-label-required', for: 'score-provider' }, 'Score Provider'),
          h('input', {
            id: 'score-provider',
            class: 'form-input',
            type: 'text',
            placeholder: 'mock-credit-bureau',
            value: scoreProvider,
            onInput: (e) => setScoreProvider(e.target.value),
            required: true,
          }),
          validationErrors.scoreProvider && h('div', { class: 'text-xs text-danger mt-1' }, validationErrors.scoreProvider)
        )
      )
    );
  };

  const renderApplication = () => {
    return h('div', { class: 'card mb-2' },
      h('div', { class: 'section-title' }, 'Application'),
      h('div', { class: 'form-row' },
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label', for: 'requested-amount' },
            'Requested Amount',
            h('span', { class: 'form-optional-tag' }, '(optional)')
          ),
          h('input', {
            id: 'requested-amount',
            class: 'form-input',
            type: 'number',
            min: '0',
            placeholder: 'Amount in smallest currency unit (e.g., 50000000)',
            value: requestedAmount,
            onInput: (e) => setRequestedAmount(e.target.value),
          }),
          validationErrors.requestedAmount && h('div', { class: 'text-xs text-danger mt-1' }, validationErrors.requestedAmount)
        ),
        h('div', { class: 'form-group' },
          h('label', { class: 'form-label', for: 'app-purpose' },
            'Purpose',
            h('span', { class: 'form-optional-tag' }, '(optional)')
          ),
          h('input', {
            id: 'app-purpose',
            class: 'form-input',
            type: 'text',
            placeholder: 'working_capital',
            value: appPurpose,
            onInput: (e) => setAppPurpose(e.target.value),
          })
        )
      ),
      h('div', { class: 'text-xs text-muted mt-1' },
        'Without a requested amount this runs a Limit Assessment (eligibility and credit limit). ',
        'With a requested amount this runs a Loan Application assessment (evaluate the requested amount against the policy).'
      )
    );
  };

  const renderReasons = (reasons) => {
    return reasons.map((r, i) =>
      h('div', { key: i, style: 'padding:0.5rem;background:var(--bg-input);border-radius:var(--radius);margin-bottom:0.375rem' },
        h('div', { class: 'flex-between' },
          h('code', { class: 'text-sm' }, r.code),
          r.value != null && h('span', { class: 'text-xs text-dim' },
            String(r.value), r.threshold != null ? ` / threshold: ${r.threshold}` : ''
          )
        ),
        r.description && h('div', { class: 'text-xs text-muted mt-1' }, r.description)
      )
    );
  };

  const renderEvidenceTable = () => {
    if (!evidence || evidence.length === 0) return null;
    return h('div', { class: 'mt-2' },
      h('div', { class: 'section-title' }, 'Evidence'),
      h('div', { class: 'table-wrap' },
        h('table', null,
          h('thead', null,
            h('tr', null,
              h('th', null, 'Source'),
              h('th', null, 'Field'),
              h('th', null, 'Value'),
              h('th', null, 'Reference'),
              h('th', null, 'Retrieved At')
            )
          ),
          h('tbody', null,
            evidence.map((e, i) =>
              h('tr', { key: i },
                h('td', null, e.source),
                h('td', null, h('code', null, e.field)),
                h('td', null, String(e.value)),
                h('td', null, h('code', null, e.reference)),
                h('td', null, formatTime(e.retrieved_at))
              )
            )
          )
        )
      )
    );
  };

  const renderResult = () => {
    if (!result) return null;

    const d = result.decision;

    return h('div', { class: 'card mt-2' },
      h('div', { class: 'section-title' }, 'Result'),
      h('div', { class: 'flex-between mb-1' },
        h('div', null,
          h('div', { class: 'text-xs text-dim' }, 'Assessment ID'),
          h('a', { href: `/workspace/assessments/${result.id}`, class: 'text-sm font-mono' }, result.id)
        ),
        d && h('span', { class: 'badge ' + outcomeBadgeClass(d.outcome) }, d.outcome)
      ),
      d && h('div', { class: 'mt-1' },
        h('div', { class: 'text-xs text-dim' }, 'Policy'),
        h('div', { class: 'text-sm font-mono' },
          d.policy?.id, ':v', d.policy?.version
        )
      ),
      d?.reasons?.length > 0 && h('div', { class: 'mt-2' },
        h('div', { class: 'text-xs text-dim mb-1' }, 'Reasons'),
        renderReasons(d.reasons)
      ),
      d?.outputs && h('div', { class: 'mt-2' },
        h('div', { class: 'text-xs text-dim mb-1' }, 'Outputs'),
        h('div', { class: 'grid grid-2' },
          h('div', null,
            h('div', { class: 'text-xs text-dim' }, 'Credit Limit'),
            h('div', { class: 'text-sm font-mono' },
              d.outputs.credit_limit != null
                ? formatCurrency(d.outputs.credit_limit)
                : '\u2014'
            )
          ),
          h('div', null,
            h('div', { class: 'text-xs text-dim' }, 'Approved Amount'),
            h('div', { class: 'text-sm font-mono' },
              d.outputs.approved_amount != null
                ? formatCurrency(d.outputs.approved_amount)
                : '\u2014'
            )
          )
        )
      ),
      h('div', { class: 'mt-2' },
        h('a', { href: `/workspace/assessments/${result.id}`, class: 'btn' }, 'View Full Details \u2192')
      ),
      renderEvidenceTable()
    );
  };

  return h('div', null,
    h('div', { class: 'page-header' },
      h('h1', { class: 'page-title' }, 'Assessment Simulator'),
      h('p', { class: 'page-desc' }, 'Run credit assessments against configured policies. All decisioning is executed by the backend engine.')
    ),
    h('form', { onSubmit: handleSubmit },
      renderPolicySelector(),
      renderApplicant(),
      renderFinancial(),
      renderCredit(),
      renderApplication(),
      h('button', {
        type: 'submit',
        class: 'btn btn-primary',
        disabled: loading,
        style: 'width:100%',
      }, loading ? 'Running assessment...' : 'Run Assessment')
    ),
    error && h('div', { class: 'error-msg mt-2' }, error),
    renderResult()
  );
}
