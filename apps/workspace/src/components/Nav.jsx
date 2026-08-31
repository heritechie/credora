import { h } from 'preact';

const links = [
  { href: '/workspace/', label: 'Home', key: 'home' },
  { href: '/workspace/policies', label: 'Policies', key: 'policies' },
  { href: '/workspace/assessments', label: 'Assessments', key: 'assessments' },
  { href: '/workspace/assessments/new', label: 'New Assessment', key: 'new-assessment' },
];

export function Nav({ current }) {
  return h('nav', { class: 'nav' },
    h('div', { class: 'nav-brand' },
      h('a', { href: '/workspace/', class: 'nav-logo' }, 'Credora'),
      h('span', { class: 'nav-subtitle' }, 'Decisioning Workspace')
    ),
    h('ul', { class: 'nav-links' },
      links.map(link =>
        h('li', { key: link.key },
          h('a', {
            href: link.href,
            class: current === link.key ? 'nav-link active' : 'nav-link'
          }, link.label)
        )
      )
    )
  );
}
