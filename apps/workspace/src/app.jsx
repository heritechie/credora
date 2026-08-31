import { useState, useEffect, useCallback } from 'preact/hooks';
import { h } from 'preact';
import { Nav } from './components/Nav.jsx';
import { Home } from './pages/Home.jsx';
import { Policies } from './pages/Policies.jsx';
import { Assessments } from './pages/Assessments.jsx';
import { NewAssessment } from './pages/NewAssessment.jsx';
import { AssessmentDetail } from './pages/AssessmentDetail.jsx';

const BASE = '/workspace';

function parseRoute() {
  const path = window.location.pathname;
  // Strip the /workspace base prefix
  const sub = path.startsWith(BASE) ? path.slice(BASE.length) : path;
  const parts = sub.split('/').filter(Boolean);

  if (parts.length === 0) return { page: 'home' };
  if (parts[0] === 'policies') return { page: 'policies' };
  if (parts[0] === 'assessments') {
    if (parts[1] === 'new') return { page: 'new-assessment' };
    if (parts[1]) return { page: 'assessment-detail', id: parts[1] };
    return { page: 'assessments' };
  }
  return { page: 'home' };
}

export function App() {
  const [route, setRoute] = useState(parseRoute);

  useEffect(() => {
    const onPop = () => setRoute(parseRoute());
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);

  const navigate = useCallback((path) => {
    const url = BASE + path;
    if (window.location.pathname === url) return;
    window.history.pushState(null, '', url);
    setRoute(parseRoute());
  }, []);

  let page;
  switch (route.page) {
    case 'policies':
      page = <Policies />;
      break;
    case 'assessments':
      page = <Assessments />;
      break;
    case 'new-assessment':
      page = <NewAssessment />;
      break;
    case 'assessment-detail':
      page = <AssessmentDetail id={route.id} />;
      break;
    default:
      page = <Home />;
  }

  return h('div', { class: 'workspace' },
    h(Nav, { current: route.page }),
    h('main', { class: 'main' }, page)
  );
}
