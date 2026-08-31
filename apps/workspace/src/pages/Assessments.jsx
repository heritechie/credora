import { h } from 'preact';
import { useState, useEffect, useCallback } from 'preact/hooks';
import { getAssessments } from '../api/client.js';
import { formatTime, outcomeBadgeClass, statusBadgeClass } from '../lib/format.js';

const PAGE_SIZE = 20;

function truncateId(id) {
  if (!id) return '';
  return id.length > 12 ? id.slice(0, 12) + '...' : id;
}

function navigateTo(path) {
  window.history.pushState(null, '', path);
  window.dispatchEvent(new PopStateEvent('popstate'));
}

export function Assessments() {
  const [items, setItems] = useState(null);
  const [error, setError] = useState(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [lookupId, setLookupId] = useState('');

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const data = await getAssessments();
        if (!cancelled) setItems(data.items || []);
      } catch (err) {
        if (!cancelled) setError(err);
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  const handleLookup = useCallback((e) => {
    e.preventDefault();
    const id = lookupId.trim();
    if (!id) return;
    navigateTo('/workspace/assessments/' + encodeURIComponent(id));
  }, [lookupId]);

  const totalPages = items ? Math.max(1, Math.ceil(items.length / PAGE_SIZE)) : 1;
  const pageItems = items
    ? items.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
    : [];

  const renderHistory = () => {
    if (error) {
      return h('div', { class: 'error-msg mt-2' }, error?.message || 'Failed to load assessments.');
    }
    if (items === null) {
      return h('div', { class: 'loading' }, 'Loading assessments...');
    }
    if (items.length === 0) {
      return h('div', { class: 'empty-state' },
        h('div', { class: 'empty-state-icon' }, '\u{1F4CB}'),
        h('div', { class: 'empty-state-text' }, 'No assessments have been executed yet.'),
        h('a', { href: '/workspace/assessments/new', class: 'btn btn-primary mt-1' }, 'New Assessment')
      );
    }
    return h('div', null,
      h('div', { class: 'table-wrap' },
        h('table', null,
          h('thead', null,
            h('tr', null,
              h('th', null, 'Assessment ID'),
              h('th', null, 'Policy'),
              h('th', null, 'Decision'),
              h('th', null, 'Status'),
              h('th', null, 'Created'),
              h('th', { class: 'hide-mobile' }, 'Completed'),
              h('th', { class: 'hide-mobile' }, '')
            )
          ),
          h('tbody', null,
            pageItems.map(a =>
              h('tr', { key: a.id },
                h('td', null,
                  h('code', { class: 'text-xs', title: a.id }, truncateId(a.id))
                ),
                h('td', null,
                  h('code', { class: 'text-xs' }, a.policy?.id, ':v', a.policy?.version)
                ),
                h('td', null,
                  a.decision
                    ? h('span', { class: 'badge ' + outcomeBadgeClass(a.decision.outcome) }, a.decision.outcome)
                    : h('span', { class: 'badge badge-neutral' }, '\u2014')
                ),
                h('td', null,
                  h('span', { class: 'badge ' + statusBadgeClass(a.status) }, a.status)
                ),
                h('td', null, formatTime(a.created_at)),
                h('td', { class: 'hide-mobile' }, formatTime(a.completed_at)),
                h('td', { class: 'hide-mobile' },
                  h('a', { href: '/workspace/assessments/' + a.id }, 'View')
                )
              )
            )
          )
        )
      ),
      totalPages > 1 && h('div', { class: 'flex-between mt-2' },
        h('button', {
          class: 'btn',
          disabled: currentPage <= 1,
          onClick: () => setCurrentPage(p => p - 1),
        }, 'Previous'),
        h('span', { class: 'text-xs text-muted' }, 'Page ', currentPage, ' of ', totalPages),
        h('button', {
          class: 'btn',
          disabled: currentPage >= totalPages,
          onClick: () => setCurrentPage(p => p + 1),
        }, 'Next')
      )
    );
  };

  return h('div', null,
    h('div', { class: 'page-header flex-between' },
      h('div', null,
        h('h1', { class: 'page-title' }, 'Assessments'),
        h('p', { class: 'page-desc' }, 'Inspect executed assessments, decisions, and evidence.')
      ),
      h('a', { href: '/workspace/assessments/new', class: 'btn btn-primary' }, 'New Assessment')
    ),
    h('div', { class: 'card mt-2' },
      h('div', { class: 'section-title' }, 'Executed Assessments'),
      renderHistory()
    ),
    h('hr', { style: 'border:none;border-top:1px solid var(--border);margin:2rem 0' }),
    h('div', { class: 'card' },
      h('div', { class: 'card-title' }, 'Find a Specific Assessment'),
      h('p', { class: 'text-xs text-muted mb-1' }, 'Enter a known assessment ID to open its decision explanation.'),
      h('form', { onSubmit: handleLookup, class: 'flex gap-1', style: 'align-items:flex-end' },
        h('div', { class: 'form-group', style: 'flex:1;margin-bottom:0' },
          h('label', { class: 'form-label', for: 'assessment-id' }, 'Assessment ID'),
          h('input', {
            id: 'assessment-id',
            class: 'form-input',
            type: 'text',
            placeholder: 'e.g. 9f35cddd0cba4d99b86229d9e83e5e34',
            value: lookupId,
            onInput: (e) => setLookupId(e.target.value),
          })
        ),
        h('button', { type: 'submit', class: 'btn', disabled: !lookupId.trim() }, 'Look Up')
      )
    )
  );
}
