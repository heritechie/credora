import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { getPolicies, ApiError } from '../api/client.js';

export function Policies() {
  const [policies, setPolicies] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setError(null);
      try {
        const data = await getPolicies();
        if (cancelled) return;
        setPolicies(data.items || []);
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError) {
          setError(`${err.code}: ${err.message}`);
        } else {
          setError(err.message || 'Failed to load policies');
        }
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  return h('div', null,
    h('div', { class: 'page-header' },
      h('h1', { class: 'page-title' }, 'Policies'),
      h('p', { class: 'page-desc' }, 'Available credit decisioning policies and their versions.')
    ),
    error && h('div', { class: 'error-msg mt-2' }, error),
    policies === null && !error && h('div', { class: 'loading' }, 'Loading policies...'),
    policies !== null && policies.length === 0 && h('div', { class: 'empty-state' },
      h('div', { class: 'empty-state-text' }, 'No policies are registered.')
    ),
    policies !== null && policies.length > 0 && h('div', { class: 'grid', style: 'gap:0.75rem' },
      policies.map(pol =>
        h('div', { key: `${pol.id}-v${pol.version}`, class: 'card' },
          h('div', { class: 'flex-between' },
            h('div', null,
              h('div', { class: 'card-title', style: 'margin-bottom:0.25rem' }, pol.id),
              h('div', { class: 'flex gap-1' },
                pol.version != null && h('span', { class: 'badge' }, `v${pol.version}`),
                pol.status && h('span', { class: 'badge badge-neutral' }, pol.status)
              )
            )
          ),
          pol.description && h('div', { class: 'text-xs text-muted mt-1' }, pol.description)
        )
      )
    )
  );
}
