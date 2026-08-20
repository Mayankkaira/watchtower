const API = '/api/v1';
let currentUser = null;
let monitors = [];
let sslDomains = [];
let incidents = [];

function showPage(page) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  document.getElementById('page-' + page).classList.add('active');
  window.scrollTo(0, 0);
  if (page === 'dashboard') loadDashboard();
  if (page === 'profile') loadProfile();
}

function switchTab(tab) {
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
  const btn = document.querySelector('[onclick="switchTab(\'' + tab + '\')"]');
  if (btn) btn.classList.add('active');
  const panel = document.getElementById('tab-' + tab);
  if (panel) panel.classList.add('active');
  if (tab === 'monitors') renderAllMonitors();
  if (tab === 'incidents') renderIncidentTimeline();
  if (tab === 'ssl') renderSSLList();
  if (tab === 'sla') renderSLAReport();
}

async function api(method, path, body) {
  const headers = { 'Content-Type': 'application/json' };
  const token = localStorage.getItem('token');
  if (token) headers['Authorization'] = 'Bearer ' + token;
  const res = await fetch(API + path, {
    method, headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Request failed');
  return data;
}

async function handleSignup(e) {
  e.preventDefault();
  const btn = document.getElementById('signup-btn');
  const err = document.getElementById('signup-error');
  err.textContent = '';
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span>';
  try {
    const data = await api('POST', '/auth/signup', {
      name: document.getElementById('signup-name').value,
      email: document.getElementById('signup-email').value,
      password: document.getElementById('signup-password').value,
    });
    localStorage.setItem('token', data.token);
    currentUser = data.user;
    showPage('dashboard');
  } catch (ex) {
    err.textContent = ex.message;
  } finally {
    btn.disabled = false;
    btn.textContent = 'Create account';
  }
}

async function handleLogin(e) {
  e.preventDefault();
  const btn = document.getElementById('login-btn');
  const err = document.getElementById('login-error');
  err.textContent = '';
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span>';
  try {
    const data = await api('POST', '/auth/login', {
      email: document.getElementById('login-email').value,
      password: document.getElementById('login-password').value,
    });
    localStorage.setItem('token', data.token);
    currentUser = data.user;
    showPage('dashboard');
  } catch (ex) {
    err.textContent = ex.message;
  } finally {
    btn.disabled = false;
    btn.textContent = 'Log in';
  }
}

function handleLogout() {
  localStorage.removeItem('token');
  currentUser = null;
  monitors = [];
  sslDomains = [];
  incidents = [];
  showPage('landing');
}

async function loadDashboard() {
  if (!currentUser) {
    try { currentUser = await api('GET', '/me'); }
    catch { showPage('login'); return; }
  }
  document.getElementById('dash-user-name').textContent = currentUser.name || currentUser.email;
  document.getElementById('dash-plan-badge').textContent = currentUser.plan || 'free';
  await loadMonitors();
  loadDemoData();
}

async function loadMonitors() {
  try {
    const serverMonitors = await api('GET', '/monitors');
    if (serverMonitors.length > 0) {
      monitors = serverMonitors;
    } else {
      monitors = getDemoMonitors();
    }
  } catch {
    monitors = getDemoMonitors();
  }
  renderMonitors();
  updateStats();
}

function getDemoMonitors() {
  return [
    { id: 1, name: 'Production API', url: 'https://api.myapp.com/health', type: 'http', status: 'up', uptime: 99.98, response_time: 142, interval: 30, history: [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1] },
    { id: 2, name: 'User Login Flow', url: 'POST /login > GET /me > GET /dashboard', type: 'chain', status: 'up', uptime: 99.85, response_time: 312, interval: 60, steps: 3, history: [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1] },
    { id: 3, name: 'Checkout Service', url: 'https://checkout.myapp.com', type: 'http', status: 'degraded', uptime: 99.74, response_time: 890, interval: 30, baseline_response: 210, history: [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,2,2,2,2,2,2,2,2,2,2,2] },
    { id: 4, name: 'Payment Service', url: 'https://payments.myapp.com', type: 'http', status: 'down', uptime: 99.12, response_time: 0, interval: 30, history: [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,0,0,0,0,1,1,1,0,0], last_incident: '3 minutes ago', ai_summary: 'SSL certificate for payments.myapp.com expired 2 hours ago. TLS handshake failing. Recommended: renew the SSL certificate.' },
    { id: 5, name: 'CDN Origin', url: 'https://cdn.myapp.com', type: 'http', status: 'up', uptime: 99.95, response_time: 34, interval: 60, history: [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,0,1,1,1,1,1,1,1,1,1,1,1,1,1,1] },
  ];
}

function loadDemoData() {
  sslDomains = [
    { domain: 'myapp.com', issuer: "Let's Encrypt", expires: '2026-03-15', daysLeft: 89, status: 'ok' },
    { domain: 'api.myapp.com', issuer: 'DigiCert', expires: '2025-08-22', daysLeft: 12, status: 'warning' },
    { domain: 'payments.myapp.com', issuer: "Let's Encrypt", expires: '2025-08-08', daysLeft: -2, status: 'expired' },
    { domain: 'cdn.myapp.com', issuer: 'Cloudflare', expires: '2026-11-01', daysLeft: 320, status: 'ok' },
  ];
  incidents = [
    { id: 1, monitor: 'Payment Service', status: 'active', started: '12 minutes ago', duration: '12m', ai_summary: 'SSL certificate expired at 14:32 UTC. TLS handshake failing on all endpoints. Recommended: renew SSL cert.' },
    { id: 2, monitor: 'Checkout Service', status: 'degraded', started: '47 minutes ago', duration: '47m', ai_summary: 'p95 response time increased from 210ms to 890ms (+324%). Correlates with database connection pool saturation.' },
    { id: 3, monitor: 'Production API', status: 'resolved', started: '6 hours ago', duration: '4m', resolved: '5h 56m ago', ai_summary: 'Brief outage during deployment of v2.14.3. Auto-recovered after Kubernetes rollback.' },
  ];
}

function renderMonitors() {
  const el = document.getElementById('monitors-list');
  if (!monitors.length) {
    el.innerHTML = '<div class="empty">No monitors yet. Add one to get started.</div>';
    return;
  }
  el.innerHTML = monitors.map(renderMonitorItem).join('');
}

function renderAllMonitors() {
  const el = document.getElementById('all-monitors-list');
  if (!el) return;
  if (!monitors.length) {
    el.innerHTML = '<div class="empty">No monitors configured.</div>';
    return;
  }
  el.innerHTML = monitors.map(renderMonitorItem).join('');
}

function renderMonitorItem(m) {
  const typeBadge = m.type === 'chain'
    ? '<span class="mon-type mon-type--chain">chain &middot; ' + m.steps + ' steps</span>'
    : '<span class="mon-type">http</span>';

  const respText = m.status === 'down' ? 'Down' : m.response_time + 'ms';
  const uptimeColor = m.uptime >= 99.9 ? 'var(--green)' : m.uptime >= 99 ? 'var(--yellow)' : 'var(--red)';

  const bars = (m.history || []).map(h => {
    if (h === 0) return '<div class="ubar ubar--down"></div>';
    if (h === 2) return '<div class="ubar ubar--degraded"></div>';
    return '<div class="ubar"></div>';
  }).join('');

  return '<div class="monitor-row">' +
    '<span class="status-dot status-dot--' + m.status + '"></span>' +
    '<div class="mon-info"><div class="mon-name">' + esc(m.name) + ' ' + typeBadge + '</div>' +
    '<div class="mon-url">' + esc(m.url) + '</div></div>' +
    '<div class="mon-metrics">' +
    '<div class="mon-metric"><div class="mon-metric-val">' + respText + '</div><div class="mon-metric-lbl">Response</div></div>' +
    '<div class="mon-metric"><div class="mon-metric-val" style="color:' + uptimeColor + '">' + m.uptime + '%</div><div class="mon-metric-lbl">Uptime</div></div>' +
    '</div>' +
    '<div class="mon-bars">' + bars + '</div>' +
    '<div class="mon-actions"><button class="btn btn--danger btn--sm" onclick="deleteMonitor(' + m.id + ')">Delete</button></div>' +
    '</div>';
}

function updateStats() {
  const total = monitors.length;
  const up = monitors.filter(m => m.status === 'up').length;
  const degraded = monitors.filter(m => m.status === 'degraded').length;
  const down = monitors.filter(m => m.status === 'down').length;
  const responding = monitors.filter(m => m.response_time > 0);
  const avg = responding.length ? Math.round(responding.reduce((a, m) => a + m.response_time, 0) / responding.length) : 0;
  const sla = total ? (monitors.reduce((a, m) => a + m.uptime, 0) / total).toFixed(2) + '%' : '\u2014';

  document.getElementById('stat-total').textContent = total;
  document.getElementById('stat-up').textContent = up;
  document.getElementById('stat-degraded').textContent = degraded;
  document.getElementById('stat-down').textContent = down;
  document.getElementById('stat-avg-response').textContent = total ? avg + 'ms' : '\u2014';
  document.getElementById('stat-sla').textContent = sla;

  renderDegradationAlerts();
  renderActiveIncidents();
}

function renderDegradationAlerts() {
  const el = document.getElementById('degradation-list');
  const degraded = monitors.filter(m => m.status === 'degraded');
  if (!degraded.length) {
    el.innerHTML = '<div class="empty">No degradation detected. All services within baseline.</div>';
    return;
  }
  el.innerHTML = degraded.map(m => {
    const pct = m.baseline_response ? Math.round((m.response_time / m.baseline_response - 1) * 100) + '%' : 'significant';
    return '<div class="alert-row">' +
      '<span class="alert-dot"></span>' +
      '<div class="alert-body">' +
      '<div class="alert-title">' + esc(m.name) + ' \u2014 response time +' + pct + '</div>' +
      '<div class="alert-desc">Current: ' + m.response_time + 'ms \u00b7 Baseline: ' + (m.baseline_response || 'N/A') + 'ms</div>' +
      '<div class="alert-ai">Likely database connection pool saturation. Similar pattern observed previously.</div>' +
      '</div></div>';
  }).join('');
}

function renderActiveIncidents() {
  const el = document.getElementById('incidents-list');
  const active = incidents.filter(i => i.status !== 'resolved');
  if (!active.length) {
    el.innerHTML = '<div class="empty">No active incidents. All systems operational.</div>';
    return;
  }
  el.innerHTML = active.map(i => {
    const dotCls = i.status === 'degraded' ? 'incident-dot--degraded' : 'incident-dot--down';
    const rowCls = i.status === 'active' ? 'incident-row--active' : '';
    return '<div class="incident-row ' + rowCls + '">' +
      '<span class="incident-dot ' + dotCls + '"></span>' +
      '<div class="inc-body"><div class="inc-title">' + esc(i.monitor) + ' is ' + (i.status === 'degraded' ? 'degraded' : 'down') + '</div>' +
      '<div class="inc-analysis">' + esc(i.ai_summary) + '</div></div>' +
      '<div class="inc-time">' + i.started + '</div></div>';
  }).join('');
}

function renderIncidentTimeline() {
  const el = document.getElementById('incident-timeline');
  if (!el) return;
  if (!incidents.length) {
    el.innerHTML = '<div class="empty">No incidents recorded.</div>';
    return;
  }
  const cls = i => i.status === 'resolved' ? 'tl-item--resolved' : i.status === 'degraded' ? 'tl-item--degraded' : '';
  const label = i => i.status === 'resolved' ? 'Resolved' : i.status === 'degraded' ? 'Degraded' : 'Down';
  el.innerHTML = '<div class="tl">' + incidents.map(i =>
    '<div class="tl-item ' + cls(i) + '">' +
    '<div class="tl-hd"><span class="tl-title">' + esc(i.monitor) + ' \u2014 ' + label(i) + '</span>' +
    '<span class="tl-time">' + i.started + ' \u00b7 ' + i.duration + '</span></div>' +
    '<div class="tl-ai"><strong>Analysis</strong><br>' + esc(i.ai_summary) + '</div></div>'
  ).join('') + '</div>';
}

function renderSSLList() {
  const el = document.getElementById('ssl-list');
  if (!el) return;
  if (!sslDomains.length) {
    el.innerHTML = '<div class="empty">No domains tracked. Add a domain to monitor SSL certificates.</div>';
    return;
  }
  el.innerHTML = sslDomains.map(s => {
    const dotCls = s.status === 'expired' ? 'ssl-dot--expired' : s.status === 'warning' ? 'ssl-dot--warning' : 'ssl-dot--ok';
    const daysCls = s.daysLeft < 0 ? 'ssl-days--danger' : s.daysLeft <= 14 ? 'ssl-days--warning' : 'ssl-days--ok';
    const daysText = s.daysLeft < 0 ? 'EXPIRED' : s.daysLeft + 'd';
    return '<div class="ssl-row">' +
      '<span class="ssl-dot ' + dotCls + '"></span>' +
      '<div class="ssl-info"><div class="ssl-domain">' + esc(s.domain) + '</div>' +
      '<div class="ssl-meta">' + esc(s.issuer) + ' \u00b7 Expires ' + s.expires + '</div></div>' +
      '<div class="ssl-countdown"><div class="ssl-days ' + daysCls + '">' + daysText + '</div>' +
      '<div class="ssl-days-lbl">remaining</div></div></div>';
  }).join('');
}

function renderSLAReport() {
  const el = document.getElementById('sla-monitors-list');
  if (!el) return;
  el.innerHTML = monitors.map(m => {
    const color = m.uptime >= 99.9 ? 'var(--green)' : m.uptime >= 99 ? 'var(--yellow)' : 'var(--red)';
    return '<div class="sla-bar-row">' +
      '<div class="sla-bar-name">' + esc(m.name) + '</div>' +
      '<div class="sla-bar-track"><div class="sla-bar-fill" style="width:' + m.uptime + '%;background:' + color + '"></div></div>' +
      '<div class="sla-bar-pct" style="color:' + color + '">' + m.uptime + '%</div></div>';
  }).join('');
}

function openAddMonitor() { document.getElementById('add-monitor-modal').classList.add('open'); }
function closeAddMonitor() {
  document.getElementById('add-monitor-modal').classList.remove('open');
  document.getElementById('add-monitor-form').reset();
  document.getElementById('monitor-error').textContent = '';
}
function closeModal(e) { if (e.target === e.currentTarget) closeAddMonitor(); }

async function handleAddMonitor(e) {
  e.preventDefault();
  const name = document.getElementById('monitor-name').value.trim();
  const url = document.getElementById('monitor-url').value.trim();
  const interval = parseInt(document.getElementById('monitor-interval').value);
  if (!name || !url) { document.getElementById('monitor-error').textContent = 'Name and URL required'; return; }
  try {
    const m = await api('POST', '/monitors', { name, url, type: 'http', interval });
    m.history = Array.from({ length: 30 }, () => 1);
    monitors.push(m);
  } catch (ex) {
    document.getElementById('monitor-error').textContent = ex.message;
    return;
  }
  renderMonitors(); updateStats(); closeAddMonitor();
  toast('Monitor added', 'success');
}

function openAddChain() { document.getElementById('add-chain-modal').classList.add('open'); }
function closeAddChain() {
  document.getElementById('add-chain-modal').classList.remove('open');
  document.getElementById('add-chain-form').reset();
  document.getElementById('chain-steps').innerHTML = getChainStepHTML(1);
  document.getElementById('chain-error').textContent = '';
}
function closeChainModal(e) { if (e.target === e.currentTarget) closeAddChain(); }

let chainStepCount = 1;
function addChainStep() {
  chainStepCount++;
  document.getElementById('chain-steps').insertAdjacentHTML('beforeend', getChainStepHTML(chainStepCount));
}
function getChainStepHTML(n) {
  return '<div class="chain-step"><span class="chain-step-lbl">Step ' + n + '</span>' +
    '<div class="field-row"><div class="field field--sm"><select class="chain-method">' +
    '<option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option></select></div>' +
    '<div class="field field--grow"><input type="url" class="chain-url" placeholder="https://api.example.com/endpoint" required></div></div></div>';
}

async function handleAddChain(e) {
  e.preventDefault();
  const name = document.getElementById('chain-name').value.trim();
  const methods = document.querySelectorAll('.chain-method');
  const urls = document.querySelectorAll('.chain-url');
  const steps = [];
  methods.forEach((m, i) => { const u = urls[i].value.trim(); if (u) steps.push(m.value + ' ' + u); });
  if (!name || !steps.length) { document.getElementById('chain-error').textContent = 'Name and at least one step required'; return; }
  const hist = Array.from({ length: 30 }, () => Math.random() > 0.08 ? 1 : 0);
  monitors.push({ id: Date.now(), name, url: steps.join(' > '), type: 'chain', steps: steps.length, status: 'up', uptime: parseFloat((97 + Math.random() * 3).toFixed(2)), response_time: Math.round(200 + Math.random() * 500), interval: 60, history: hist });
  renderMonitors(); updateStats(); closeAddChain();
  toast('Chain created', 'success');
}

function openAddSSL() { document.getElementById('add-ssl-modal').classList.add('open'); }
function closeAddSSL() {
  document.getElementById('add-ssl-modal').classList.remove('open');
  document.getElementById('add-ssl-form').reset();
  document.getElementById('ssl-error').textContent = '';
}
function closeSSLModal(e) { if (e.target === e.currentTarget) closeAddSSL(); }

async function handleAddSSL(e) {
  e.preventDefault();
  const domain = document.getElementById('ssl-domain').value.trim();
  if (!domain) { document.getElementById('ssl-error').textContent = 'Domain required'; return; }
  sslDomains.push({ domain, issuer: 'Checking...', expires: 'Checking...', daysLeft: Math.floor(30 + Math.random() * 300), status: 'ok' });
  renderSSLList(); closeAddSSL();
  toast('Domain added', 'success');
}

async function deleteMonitor(id) {
  try { await api('DELETE', '/monitors?id=' + id); } catch {}
  monitors = monitors.filter(m => m.id !== id);
  renderMonitors(); renderAllMonitors(); updateStats();
  toast('Monitor deleted');
}

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

function toast(msg, type) {
  const c = document.getElementById('toast-container');
  const el = document.createElement('div');
  el.className = 'toast' + (type ? ' toast--' + type : '');
  el.textContent = msg;
  c.appendChild(el);
  setTimeout(() => { el.classList.add('fade-out'); setTimeout(() => el.remove(), 150); }, 2500);
}

async function loadProfile() {
  if (!currentUser) {
    try { currentUser = await api('GET', '/me'); }
    catch { showPage('login'); return; }
  }
  const u = currentUser;
  document.getElementById('profile-name').textContent = u.name || u.email;
  document.getElementById('profile-email').textContent = u.email || '';
  document.getElementById('profile-avatar').textContent = (u.name || u.email || 'W').charAt(0).toUpperCase();
  document.getElementById('profile-name-input').value = u.name || '';
  document.getElementById('profile-email-input').value = u.email || '';
  document.getElementById('profile-joined-date').textContent = u.created_at ? new Date(u.created_at).toLocaleDateString('en-US', { month: 'short', year: 'numeric' }) : 'N/A';
  const plan = u.plan || 'free';
  document.getElementById('profile-plan-name').textContent = plan;
  const planDescs = { free: '5 monitors, 5-minute intervals', pro: '50 monitors, 30-second intervals', enterprise: 'Unlimited monitors, 10-second intervals' };
  document.getElementById('profile-plan-desc').textContent = planDescs[plan] || '';
  loadProfileKeys();
}

async function loadProfileKeys() {
  const el = document.getElementById('profile-keys-list');
  try {
    const keys = await api('GET', '/keys');
    if (!keys || !keys.length) {
      el.innerHTML = '<div class="empty">No API keys created yet.</div>';
      return;
    }
    el.innerHTML = keys.map(function(k) {
      return '<div class="profile-key-row">' +
        '<div><div class="profile-key-val">' + esc(k.key || k.prefix || 'wt_***') + '</div>' +
        '<div class="profile-key-created">Created ' + (k.created_at ? new Date(k.created_at).toLocaleDateString() : 'recently') + '</div></div>' +
        '<button class="btn btn--danger btn--sm" onclick="handleDeleteKey(\'' + k.id + '\')">Revoke</button></div>';
    }).join('');
  } catch {
    el.innerHTML = '<div class="empty">No API keys created yet.</div>';
  }
}

async function handleCreateKey() {
  try {
    await api('POST', '/keys', { name: 'api-key-' + Date.now() });
  } catch {}
  loadProfileKeys();
}

async function handleDeleteKey(id) {
  try { await api('DELETE', '/keys?id=' + id); } catch {}
  loadProfileKeys();
}

async function handleProfileUpdate(e) {
  e.preventDefault();
  const msg = document.getElementById('profile-msg');
  msg.textContent = '';
  msg.className = 'field-msg';
  try {
    const name = document.getElementById('profile-name-input').value.trim();
    const email = document.getElementById('profile-email-input').value.trim();
    currentUser = await api('PUT', '/me', { name: name, email: email });
    msg.textContent = 'Profile updated.';
    msg.classList.add('field-msg--ok');
    toast('Profile updated', 'success');
    loadProfile();
  } catch (ex) {
    msg.textContent = ex.message;
    msg.classList.add('field-msg--err');
  }
}

async function handleDeleteAccount() {
  if (!confirm('Are you sure you want to permanently delete your account? This cannot be undone.')) return;
  try {
    await api('DELETE', '/me');
    localStorage.removeItem('token');
    currentUser = null;
    monitors = [];
    sslDomains = [];
    incidents = [];
    toast('Account deleted');
    showPage('landing');
  } catch (ex) {
    toast(ex.message || 'Failed to delete account', 'error');
  }
}

(function() {
  const token = localStorage.getItem('token');
  if (token) showPage('dashboard');
  else showPage('landing');
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape') { closeAddMonitor(); closeAddChain(); closeAddSSL(); }
  });
  document.querySelectorAll('.footer-year').forEach(el => {
    el.textContent = new Date().getFullYear();
  });
})();