import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { getPolicies, getAssessments, ApiError } from '../api/client.js';
import { outcomeBadgeClass, statusBadgeClass } from '../lib/format.js';

function formatError(err) {
  if (err instanceof ApiError) return `${err.code}: ${err.message}`;
  return err.message || 'Failed to load workspace data';
}

const WORKFLOW_STEPS = [
  {
    n: '1',
    title: 'Configure a Policy',
    desc: 'Choose a policy and version from the engine registry.',
    href: '/workspace/policies',
  },
  {
    n: '2',
    title: 'Run an Assessment',
    desc: 'Simulate a limit assessment or a loan application.',
    href: '/workspace/assessments/new',
  },
  {
    n: '3',
    title: 'Explain the Decision',
    desc: 'Inspect the outcome, decision reasons, and supporting evidence.',
    href: '/workspace/assessments',
  },
];

export function Home() {
  const [policies, setPolicies] = useState(null);
  const [policiesError, setPoliciesError] = useState(null);
  const [assessments, setAssessments] = useState(null);
  const [assessmentsError, setAssessmentsError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    async function loadPolicies() {
      try {
        const data = await getPolicies();
        if (!cancelled) setPolicies(data.items || []);
      } catch (err) {
        if (!cancelled) setPoliciesError(formatError(err));
      }
    }
    async function loadAssessments() {
      try {
        const data = await getAssessments();
        if (!cancelled) setAssessments(data.items || []);
      } catch (err) {
        if (!cancelled) setAssessmentsError(formatError(err));
      }
    }
    loadPolicies();
    loadAssessments();
    return () => { cancelled = true; };
  }, []);

  const renderWorkflow = () => {
    return h('div', { class: 'grid grid-3 mt-2' },
      WORKFLOW_STEPS.map(step =>
        h('a', {
          key: step.n,
          href: step.href,
          class: 'card',
          style: 'cursor:pointer;text-decoration:none',
        },
          h('div', { class: 'flex gap-1', style: 'align-items:center' },
            h('span', { class: 'workflow-step-num' }, step.n),
            h('div', { class: 'card-title', style: 'margin-bottom:0' }, step.title)
          ),
          h('p', { class: 'text-sm text-muted mt-1' }, step.desc)
        )
      )
    );
  };

  const renderPolicies = () => {
    let body;
    if (policiesError) {
      body = h('div', { class: 'text-xs text-danger' }, policiesError);
    } else if (policies === null) {
      body = h('div', { class: 'text-xs text-muted' }, 'Loading policies...');
    } else if (policies.length === 0) {
      body = h('div', { class: 'text-xs text-muted' }, 'No policies registered.');
    } else {
      body = h('div', { class: 'flex gap-1', style: 'flex-wrap:wrap' },
        policies.map(p =>
          h('span', { key: `${p.id}-v${p.version}`, class: 'badge' }, `${p.id}:v${p.version}`)
        )
      );
    }
    return h('div', { class: 'card' },
      h('div', { class: 'flex-between' },
        h('div', { class: 'card-title', style: 'margin-bottom:0' }, 'Registered Policies'),
        h('a', { href: '/workspace/policies', class: 'text-xs' }, 'View all')
      ),
      h('div', { class: 'mt-1' }, body)
    );
  };

  const renderAssessments = () => {
    let body;
    if (assessmentsError) {
      body = h('div', { class: 'text-xs text-danger' }, assessmentsError);
    } else if (assessments === null) {
      body = h('div', { class: 'text-xs text-muted' }, 'Loading assessments...');
    } else if (assessments.length === 0) {
      body = h('div', { class: 'text-xs text-muted' },
        'No assessments yet. ',
        h('a', { href: '/workspace/assessments/new' }, 'Run one from the simulator.')
      );
    } else {
      body = h('div', { class: 'mt-1' },
        assessments.slice(0, 5).map(a =>
          h('a', {
            key: a.id,
            href: `/workspace/assessments/${a.id}`,
            class: 'flex-between',
            style: 'padding:0.5rem 0;border-bottom:1px solid var(--border);text-decoration:none',
          },
            h('div', null,
              h('code', { class: 'text-xs' }, a.id),
              h('div', { class: 'text-xs text-dim' },
                a.policy?.id, ':v', a.policy?.version
              )
            ),
            h('span', {
              class: 'badge ' + (a.decision ? outcomeBadgeClass(a.decision.outcome) : statusBadgeClass(a.status))
            }, a.decision ? a.decision.outcome : a.status)
          )
        )
      );
    }
    return h('div', { class: 'card' },
      h('div', { class: 'flex-between' },
        h('div', { class: 'card-title', style: 'margin-bottom:0' }, 'Recent Assessments'),
        h('a', { href: '/workspace/assessments', class: 'text-xs' }, 'View all')
      ),
      body
    );
  };

  return h('div', null,
    h('div', { class: 'page-header flex-between' },
      h('div', null,
        h('h1', { class: 'page-title' }, 'Credora Decisioning Workspace'),
        h('p', { class: 'page-desc' }, 'Define assessments, run them against policies, and understand why every decision was produced. All evaluation executes in the Credora engine.')
      ),
      h('a', { href: '/workspace/assessments/new', class: 'btn btn-primary' }, 'Run an Assessment')
    ),
    h('div', { class: 'section-title mt-2' }, 'Decisioning Workflow'),
    renderWorkflow(),
    h('div', { class: 'grid grid-2 mt-2' },
      renderPolicies(),
      renderAssessments()
    ),
    h('div', { class: 'section mt-3' },
      h('div', { class: 'card' },
        h('div', { class: 'section-title' }, 'About'),
        h('p', { class: 'text-sm text-muted' },
          'The Credora Decisioning Workspace is a workbench for building and understanding credit decisions. ',
          'It connects to the ',
          h('code', null, 'Credora Engine API'),
          ' — all decisioning logic (knockouts, rules, score thresholds) runs in the backend. ',
          'This interface is for definition, inspection, and explanation only.'
        )
      )
    )
  );
}
