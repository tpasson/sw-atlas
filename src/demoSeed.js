// Seed data for the static (no-backend) demo build. Mirrors the server seed so
// the GitHub Pages demo shows the same content as a fresh ATLAS install.
const YEAR = new Date().getFullYear()
const d = (m, day) => `${YEAR}-${String(m).padStart(2, '0')}-${String(day).padStart(2, '0')}`

export function demoSeed() {
  const ms = (id, swimlaneId, subLaneId, month, title, what, why, how, who, when) => ({
    id, swimlaneId, subLaneId, year: YEAR, month, title, what, why, how, who, when,
    kind: 'milestone', marker: 'l:Flag', startDate: null, endDate: null,
    color: null, sourceSystem: null, externalId: null, externalUrl: null, lastSyncedAt: null,
  })

  // Event (bar) helper. marker defaults to null (no marker on the bar).
  const ev = (id, swimlaneId, subLaneId, title, startDate, endDate, who = '', marker = null) => ({
    id, swimlaneId, subLaneId, year: YEAR, month: Number(startDate.slice(5, 7)), title,
    what: '', why: '', how: '', who,
    kind: 'event', marker, startDate, endDate, when: startDate,
    color: null, sourceSystem: null, externalId: null, externalUrl: null, lastSyncedAt: null,
  })

  const swimlanes = [
    { id: 'ca', name: 'Customer Accounts', color: '#0A84FF', subLanes: [
      { id: 'ca-con', name: 'Contoso' }, { id: 'ca-nor', name: 'Northwind' }, { id: 'ca-glo', name: 'Globex' },
      { id: 'ca-acm', name: 'Acme' }, { id: 'ca-fab', name: 'Fabrikam' }, { id: 'ca-ini', name: 'Initech' } ] },
    { id: 'ev', name: 'Events', color: '#FFCC00', subLanes: [
      { id: 'ev-conf', name: 'Annual Conference' }, { id: 'ev-summit', name: 'Industry Summit' }, { id: 'ev-launch', name: 'Product Launch' } ] },
    { id: 'pr', name: 'Product', color: '#30D158', subLanes: [
      { id: 'pr-disc', name: 'Discovery' }, { id: 'pr-des', name: 'Design' },
      { id: 'pr-proto', name: 'Prototyping' }, { id: 'pr-beta', name: 'Beta Program' } ] },
    { id: 'pf', name: 'Platform', color: '#FF9F0A', subLanes: [
      { id: 'pf-api', name: 'API' }, { id: 'pf-web', name: 'Web App' }, { id: 'pf-mob', name: 'Mobile' },
      { id: 'pf-infra', name: 'Infrastructure' }, { id: 'pf-data', name: 'Data & Analytics' } ] },
    { id: 'fn', name: 'Features', color: '#FF375F', subLanes: [
      { id: 'fn-fd', name: 'Feature Development' }, { id: 'fn-qa', name: 'QA & Validation' },
      { id: 'fn-int', name: 'Integration' }, { id: 'fn-comp', name: 'Compliance' } ] },
    { id: 'tl', name: 'Tools', color: '#BF5AF2', subLanes: [
      { id: 'tl-a', name: 'Tool Alpha' }, { id: 'tl-b', name: 'Tool Beta' }, { id: 'tl-g', name: 'Tool Gamma' } ] },
    { id: 'rm', name: 'Release Management', color: '#32ADE6', subLanes: [
      { id: 'rm-sp', name: 'Sprint Planning' }, { id: 'rm-qa', name: 'QA & Testing' },
      { id: 'rm-so', name: 'Sign-off' }, { id: 'rm-de', name: 'Deployment' } ] },
  ]

  const milestones = [
    ms('m-con-rev', 'ca', 'ca-con', 6, 'Solution Review', 'Present the platform to Contoso', 'Strategic partnership potential', 'Contoso HQ', 'Sales Lead', d(6, 15)),
    ms('m-con-fw', 'ca', 'ca-con', 9, 'Framework Agreement', 'Sign framework agreement for next year', 'Lock in Contoso as anchor customer', 'Legal negotiation', 'Account Exec + Legal', d(9, 15)),
    ms('m-nor-poc', 'ca', 'ca-nor', 5, 'Proof of Concept', 'PoC for the Northwind use case', 'Gateway to a full rollout', 'Joint workshop', 'Key Account Manager', d(5, 29)),
    ms('m-glo-rev', 'ca', 'ca-glo', 7, 'Account Review', 'Annual review with Globex', 'Prepare the renewal', 'Business review deck', 'Customer Success', d(7, 9)),
    ms('m-acm-jda', 'ca', 'ca-acm', 7, 'Partnership Signed', 'Joint development agreement', 'Build a shared integration', 'Contract + NDA', 'Business Dev', d(7, 31)),
    ms('m-conf', 'ev', 'ev-conf', 1, 'Annual Conference', 'Company conference & keynote', 'Industry visibility', 'Booth + live demo', 'Marketing + Eng', d(1, 8)),
    ms('m-summit', 'ev', 'ev-summit', 9, 'Industry Summit', 'Exhibition at the industry summit', 'Reach new customers', 'Demo + keynote', 'CEO + Marketing', d(9, 10)),
    ms('m-mvp', 'pr', 'pr-proto', 5, 'MVP Ready', 'Minimum viable product complete', 'Foundation for the beta', 'Core build', 'Product Lead', d(5, 15)),
    ms('m-proto', 'pr', 'pr-proto', 3, 'Interactive Prototype', 'Clickable end-to-end prototype', 'Validate the concept early', 'Rapid prototyping', 'Design Engineer', d(3, 31)),
    ms('m-beta-launch', 'pr', 'pr-beta', 9, 'Beta Launch', 'Open beta to pilot customers', 'Real-world feedback', 'Controlled rollout', 'Product Team', d(9, 3)),
    ms('m-apiv2', 'pf', 'pf-api', 3, 'API v2', 'New public API architecture', 'Reduce integration friction', 'Design + implementation', 'Platform Architect', d(3, 31)),
    ms('m-webr2', 'pf', 'pf-web', 6, 'Web App R2', 'Redesigned web client', 'Foundation to scale', 'SW + QA verification', 'Web Team', d(6, 30)),
    ms('m-pipeline', 'pf', 'pf-infra', 7, 'CI/CD Pipeline', 'Automated build & deploy pipeline', 'Faster, safer releases', 'Infrastructure automation', 'DevOps Engineer', d(7, 31)),
    ms('m-search', 'fn', 'fn-fd', 2, 'Search & Filters', 'Advanced search and filtering', 'Core usability', 'Development sprints', 'Feature Team', d(2, 20)),
    ms('m-collab', 'fn', 'fn-fd', 5, 'Real-time Collaboration', 'Multi-user concurrent editing', 'Top customer requirement', 'Backend + UI', 'Feature Lead', d(5, 31)),
    ms('m-sso', 'fn', 'fn-fd', 9, 'SSO Integration', 'Single sign-on (SAML / OIDC)', 'Required for enterprise', 'Auth integration', 'Security Team', d(9, 30)),
    ms('m-qaq2', 'fn', 'fn-qa', 6, 'QA Campaign Q2', 'Full regression coverage for Q2', 'Quality gate for release', 'Test execution', 'QA Team', d(6, 30)),
    ms('m-audit', 'fn', 'fn-comp', 10, 'Security Audit', 'External security audit', 'Compliance requirement', 'External auditor', 'Security Lead', d(10, 31)),
    ms('m-beta', 'tl', 'tl-b', 3, 'Beta Pilot Rollout', 'Pilot with 3 teams', 'Real-world feedback', 'Controlled rollout', 'Product Owner', d(3, 31)),
    ms('m-gamma', 'tl', 'tl-g', 4, 'Gamma Prototype', 'Working Gamma prototype', 'Validate the concept', 'Rapid prototyping', 'Gamma Team', d(4, 30)),
    ms('m-systest', 'rm', 'rm-qa', 7, 'System Test Campaign', 'Full system test for H1', 'Release readiness', 'Test report', 'QA Team', d(7, 31)),
    ms('m-freeze', 'rm', 'rm-so', 9, 'Feature Freeze', 'No new features after this date', 'Enter stabilisation', 'Management decision', 'PM + CTO', d(9, 30)),
    ms('m-release', 'rm', 'rm-de', 12, 'Annual Release', 'Production release', 'Deliver the annual features', 'Deployment + hypercare', 'Release Manager', d(12, 31)),
    {
      id: 'm-summit-week', swimlaneId: 'ev', subLaneId: 'ev-summit', year: YEAR, month: 9,
      title: 'Summit Week', what: 'On-site exhibition week', why: 'Customer meetings', how: 'Conference venue', who: 'Whole team',
      kind: 'event', marker: 'bar', startDate: d(9, 8), endDate: d(9, 14), when: d(9, 8),
      color: null, sourceSystem: null, externalId: null, externalUrl: null, lastSyncedAt: null,
    },

    // ── Test-case events (edge cases) ─────────────────────────────────────
    // Spans two months
    ev('e-closed-beta', 'pr', 'pr-beta', 'Closed Beta', d(4, 15), d(6, 15), 'Product Team'),
    // Long, multi-month phase
    ev('e-int-phase', 'fn', 'fn-int', 'Integration Phase', d(9, 1), d(12, 20), 'Integration Team'),
    // Short event WITH a marker (tests the optional event marker)
    ev('e-launch-win', 'ev', 'ev-launch', 'Launch Window', d(11, 3), d(11, 9), 'Marketing', 'l:Rocket'),
    // Crosses the year-end → continues right (arrow ▶, clamped at Dec)
    ev('e-dc-migration', 'pf', 'pf-infra', 'Data Center Migration', d(12, 15), `${YEAR + 1}-01-25`, 'DevOps'),
    // Started last year → continues left (arrow ◀, clamped at Jan)
    ev('e-carryover', 'pf', 'pf-web', 'Carryover Rework', `${YEAR - 1}-12-10`, d(2, 5), 'Web Team'),
  ]

  const links = [
    { a: 'm-con-rev', b: 'm-apiv2' }, { a: 'm-con-rev', b: 'm-mvp' }, { a: 'm-con-rev', b: 'm-webr2' },
    { a: 'm-con-rev', b: 'm-search' }, { a: 'm-con-rev', b: 'm-collab' }, { a: 'm-con-fw', b: 'm-con-rev' },
    { a: 'm-nor-poc', b: 'm-search' }, { a: 'm-nor-poc', b: 'm-proto' }, { a: 'm-nor-poc', b: 'm-mvp' },
    { a: 'm-glo-rev', b: 'm-systest' }, { a: 'm-glo-rev', b: 'm-mvp' },
    { a: 'm-summit', b: 'm-beta-launch' }, { a: 'm-summit', b: 'm-con-rev' }, { a: 'm-summit', b: 'm-nor-poc' },
    { a: 'm-freeze', b: 'm-release' }, { a: 'm-sso', b: 'm-audit' },
  ]

  return { swimlanes, milestones, links }
}
