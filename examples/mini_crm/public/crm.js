/* LIAF Enterprise CRM - Frontend SPA Logic */

let ws;
let leads = [];
let products = [];
let currentTab = 'pipeline';
let draggedDealId = null;
let currentDealId = null;

const formatCurrency = (val) => {
  return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 }).format(val || 0);
};

const escapeHtml = (str) => {
  if (!str) return '';
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
};

/* Notificações Toast */
function showToast(msg, icon = '💰') {
  const c = document.getElementById('toast-container');
  if (!c) return;
  const t = document.createElement('div');
  t.className = 'toast';
  t.innerHTML = `<span>${icon}</span><span>${msg}</span>`;
  c.appendChild(t);
  setTimeout(() => {
    t.style.opacity = '0';
    t.style.transform = 'translateY(10px)';
    setTimeout(() => t.remove(), 300);
  }, 3500);
}

/* Navegação entre Abas (SPA) */
function switchTab(tab) {
  currentTab = tab;
  document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('.tab-view').forEach(el => el.classList.remove('active'));

  const navBtn = document.getElementById(`nav-${tab}`);
  const tabView = document.getElementById(`view-${tab}`);
  if (navBtn) navBtn.classList.add('active');
  if (tabView) tabView.classList.add('active');

  const titleEl = document.getElementById('header-title-text');
  const subEl = document.getElementById('header-title-sub');

  if (tab === 'pipeline') {
    titleEl.textContent = 'Funil de Vendas';
    subEl.textContent = 'Acompanhe e movimente suas oportunidades comerciais em tempo real';
    renderPipeline();
  } else if (tab === 'analytics') {
    titleEl.textContent = 'Analytics & Desempenho';
    subEl.textContent = 'Métricas agregadas, taxa de conversão e ticket médio do SQLite';
    renderAnalytics();
  } else if (tab === 'leads') {
    titleEl.textContent = 'Contatos & Oportunidades';
    subEl.textContent = 'Visão tabular completa de leads e histórico cadastral';
    renderLeadsTable();
  } else if (tab === 'products') {
    titleEl.textContent = 'Catálogo de Produtos & Serviços';
    subEl.textContent = 'Tabela de preços e itens gerenciados nativamente';
    loadProducts();
  }
}

/* Carregamento de Dados da API */
async function loadLeads() {
  try {
    const res = await fetch('/api/leads');
    const data = await res.json();
    leads = data.leads || [];
    updateKPIs();
    if (currentTab === 'pipeline') renderPipeline();
    if (currentTab === 'analytics') renderAnalytics();
    if (currentTab === 'leads') renderLeadsTable();
  } catch (err) {
    console.error('Falha ao carregar leads:', err);
  }
}

async function loadProducts() {
  try {
    const res = await fetch('/api/products');
    const data = await res.json();
    products = data.products || [];
    renderProductsTable();
  } catch (err) {
    console.error('Falha ao carregar produtos:', err);
  }
}

async function updateOnlineCount() {
  try {
    const res = await fetch('/api/online');
    const data = await res.json();
    const el = document.getElementById('kpi-online-count');
    if (el) el.textContent = `${data.online || 1} online`;
  } catch (e) {}
}

/* Atualização dos KPIs */
function updateKPIs() {
  let activeTotal = 0;
  let activeCount = 0;
  let wonTotal = 0;
  let wonCount = 0;

  leads.forEach(l => {
    if (l.stage === 'won') {
      wonTotal += l.value;
      wonCount++;
    } else {
      activeTotal += l.value;
      activeCount++;
    }
  });

  const totalDeals = leads.length;
  const avgTicket = totalDeals > 0 ? (activeTotal + wonTotal) / totalDeals : 0;

  const kpiPipe = document.getElementById('kpi-pipeline');
  const kpiPipeSub = document.getElementById('kpi-active-sub');
  const kpiWon = document.getElementById('kpi-won');
  const kpiWonSub = document.getElementById('kpi-won-sub');
  const kpiAvg = document.getElementById('kpi-avg');
  const kpiTotal = document.getElementById('kpi-total');

  if (kpiPipe) kpiPipe.textContent = formatCurrency(activeTotal);
  if (kpiPipeSub) kpiPipeSub.textContent = `${activeCount} oportunidades em aberto`;
  if (kpiWon) kpiWon.textContent = formatCurrency(wonTotal);
  if (kpiWonSub) kpiWonSub.textContent = `${wonCount} negócios fechados`;
  if (kpiAvg) kpiAvg.textContent = formatCurrency(avgTicket);
  if (kpiTotal) kpiTotal.textContent = totalDeals;
}

/* Renderização: Pipeline Kanban */
function renderPipeline() {
  const cols = { prospect: [], proposal: [], negotiation: [], won: [] };
  const sums = { prospect: 0, proposal: 0, negotiation: 0, won: 0 };

  leads.forEach(l => {
    const stage = cols[l.stage] ? l.stage : 'prospect';
    cols[stage].push(l);
    sums[stage] += l.value;
  });

  ['prospect', 'proposal', 'negotiation', 'won'].forEach(stage => {
    const cont = document.getElementById(`cards-${stage}`);
    const count = document.getElementById(`count-${stage}`);
    const sumEl = document.getElementById(`sum-${stage}`);
    if (!cont) return;

    cont.innerHTML = '';
    if (count) count.textContent = cols[stage].length;
    if (sumEl) sumEl.textContent = formatCurrency(sums[stage]);

    cols[stage].forEach(deal => {
      const el = document.createElement('div');
      el.className = 'deal-card';
      el.draggable = true;
      el.id = `deal-${deal.id}`;
      el.onclick = (e) => {
        if (!e.target.closest('.btn-icon')) openDrawer(deal.id);
      };

      const priority = deal.priority || 'medium';
      el.innerHTML = `
        <div class="deal-header-row">
          <div class="deal-company">${escapeHtml(deal.company || 'Empresa')}</div>
          <span class="priority-tag ${priority}">${priority}</span>
        </div>
        <div class="deal-name">${escapeHtml(deal.name)}</div>
        <div class="deal-value">${formatCurrency(deal.value)}</div>
        ${deal.notes ? `<div class="deal-notes">${escapeHtml(deal.notes)}</div>` : ''}
        <div class="deal-footer">
          <div class="deal-contact">📧 ${escapeHtml(deal.email)}</div>
          <div class="deal-actions">
            <button class="btn-icon" onclick="openDrawer(${deal.id}); event.stopPropagation();" title="Histórico">📝</button>
            <button class="btn-icon delete" onclick="deleteDeal(${deal.id}); event.stopPropagation();" title="Excluir">✕</button>
          </div>
        </div>
      `;

      el.addEventListener('dragstart', (e) => {
        draggedDealId = deal.id;
        el.classList.add('dragging');
        e.dataTransfer.setData('text/plain', deal.id);
      });
      el.addEventListener('dragend', () => {
        el.classList.remove('dragging');
        draggedDealId = null;
      });

      cont.appendChild(el);
    });
  });
}

function onDragOver(e) { e.preventDefault(); e.currentTarget.classList.add('dragover'); }
function onDragLeave(e) { e.currentTarget.classList.remove('dragover'); }

async function onDrop(e, stage) {
  e.preventDefault();
  e.currentTarget.classList.remove('dragover');
  if (!draggedDealId) return;

  try {
    const res = await fetch(`/api/leads/${draggedDealId}/stage`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stage })
    });
    if (res.ok) {
      loadLeads();
      if (stage === 'won') {
        showToast('Parabéns! Negócio marcado como GANHO!', '🎉');
      }
    }
  } catch (err) {
    console.error('Falha ao mover estágio:', err);
  }
}

/* Renderização: Tabela de Leads */
function renderLeadsTable() {
  const tbody = document.getElementById('leads-tbody');
  if (!tbody) return;
  tbody.innerHTML = '';

  leads.forEach(l => {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td><strong>${escapeHtml(l.name)}</strong></td>
      <td>${escapeHtml(l.company)}</td>
      <td>${escapeHtml(l.email)}</td>
      <td>${escapeHtml(l.phone || '-')}</td>
      <td><span class="priority-tag ${l.priority}">${l.stage.toUpperCase()}</span></td>
      <td><strong>${formatCurrency(l.value)}</strong></td>
      <td>
        <button class="btn-secondary" style="padding: 4px 8px; font-size: 0.75rem;" onclick="openDrawer(${l.id})">Ver 360°</button>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

/* Renderização: Tabela de Produtos */
function renderProductsTable() {
  const tbody = document.getElementById('products-tbody');
  if (!tbody) return;
  tbody.innerHTML = '';

  products.forEach(p => {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td><strong>${escapeHtml(p.name)}</strong></td>
      <td><code>${escapeHtml(p.sku)}</code></td>
      <td>${escapeHtml(p.category)}</td>
      <td><strong>${formatCurrency(p.price)}</strong></td>
    `;
    tbody.appendChild(tr);
  });
}

/* Renderização: Analytics */
function renderAnalytics() {
  const counts = { prospect: 0, proposal: 0, negotiation: 0, won: 0 };
  const values = { prospect: 0, proposal: 0, negotiation: 0, won: 0 };
  let total = leads.length || 1;

  leads.forEach(l => {
    if (counts[l.stage] !== undefined) {
      counts[l.stage]++;
      values[l.stage] += l.value;
    }
  });

  const stages = [
    { key: 'prospect', label: 'Prospecção', color: '#388bfd' },
    { key: 'proposal', label: 'Proposta', color: '#d29922' },
    { key: 'negotiation', label: 'Negociação', color: '#a371f7' },
    { key: 'won', label: 'Fechado / Ganho', color: '#2ea043' }
  ];

  const cont = document.getElementById('funnel-bars-container');
  if (!cont) return;
  cont.innerHTML = '';

  stages.forEach(st => {
    const pct = Math.round((counts[st.key] / total) * 100);
    const row = document.createElement('div');
    row.className = 'funnel-stage-row';
    row.innerHTML = `
      <div class="funnel-stage-label">
        <span><strong>${st.label}</strong> (${counts[st.key]} leads)</span>
        <span>${formatCurrency(values[st.key])} • ${pct}%</span>
      </div>
      <div class="funnel-bar-bg">
        <div class="funnel-bar-fill" style="width: ${Math.max(pct, 6)}%; background: ${st.color};">
          ${pct}%
        </div>
      </div>
    `;
    cont.appendChild(row);
  });

  // Métricas adicionais
  const winRate = total > 0 ? Math.round((counts.won / total) * 100) : 0;
  const wrEl = document.getElementById('metric-win-rate');
  if (wrEl) wrEl.textContent = `${winRate}%`;
}

/* Drawer / Visão 360° do Lead com Atividades */
async function openDrawer(dealId) {
  currentDealId = dealId;
  const deal = leads.find(l => l.id === dealId);
  if (!deal) return;

  document.getElementById('drawer-deal-name').textContent = deal.name;
  document.getElementById('drawer-deal-company').textContent = deal.company;
  document.getElementById('drawer-deal-value').textContent = formatCurrency(deal.value);
  document.getElementById('drawer-deal-stage').textContent = deal.stage.toUpperCase();
  document.getElementById('drawer-deal-email').textContent = deal.email;
  document.getElementById('drawer-deal-phone').textContent = deal.phone || 'Não informado';
  document.getElementById('drawer-deal-notes').textContent = deal.notes || 'Sem notas adicionais.';

  await loadActivities(dealId);

  document.getElementById('drawer-overlay').classList.add('active');
}

function closeDrawer() {
  document.getElementById('drawer-overlay').classList.remove('active');
  currentDealId = null;
}

async function loadActivities(dealId) {
  const timeline = document.getElementById('drawer-timeline');
  if (!timeline) return;
  timeline.innerHTML = '<div style="color: var(--text-muted); font-size: 0.8rem;">Carregando histórico...</div>';

  try {
    const res = await fetch(`/api/leads/${dealId}/activities`);
    const data = await res.json();
    const acts = data.activities || [];

    timeline.innerHTML = '';
    if (acts.length === 0) {
      timeline.innerHTML = '<div style="color: var(--text-muted); font-size: 0.8rem;">Nenhuma atividade registrada ainda.</div>';
      return;
    }

    acts.forEach(a => {
      const item = document.createElement('div');
      item.className = 'timeline-item';
      const icon = a.kind === 'call' ? '📞' : a.kind === 'meeting' ? '👥' : a.kind === 'won' ? '🏆' : '📝';
      item.innerHTML = `
        <div class="timeline-dot"></div>
        <div class="timeline-content">
          <div><strong>${icon} ${escapeHtml(a.description)}</strong></div>
          <div class="timeline-meta">
            <span>👤 ${escapeHtml(a.author)}</span>
            <span>🕒 ${escapeHtml(a.created_at)}</span>
          </div>
        </div>
      `;
      timeline.appendChild(item);
    });
  } catch (err) {
    console.error('Falha ao carregar atividades:', err);
  }
}

async function submitNewActivity(e) {
  e.preventDefault();
  if (!currentDealId) return;

  const desc = document.getElementById('activity-desc').value.trim();
  const kind = document.getElementById('activity-kind').value;
  const author = 'Gabriel';
  const created_at = new Date().toLocaleString([], { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });

  if (!desc) return;

  try {
    const res = await fetch('/api/activities', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        deal_id: currentDealId,
        kind,
        description: desc,
        author,
        created_at
      })
    });
    if (res.ok) {
      document.getElementById('activity-desc').value = '';
      loadActivities(currentDealId);
      showToast('Atividade registrada com sucesso');
    }
  } catch (err) {
    console.error('Falha ao registrar atividade:', err);
  }
}

/* Modais de Cadastro */
function openNewDealModal() {
  document.getElementById('modal-overlay').classList.add('active');
  document.getElementById('deal-name').focus();
}

function closeNewDealModal() {
  document.getElementById('modal-overlay').classList.remove('active');
  document.getElementById('new-deal-form').reset();
}

async function submitNewDeal(e) {
  e.preventDefault();
  const name = document.getElementById('deal-name').value.trim();
  const company = document.getElementById('deal-company').value.trim();
  const email = document.getElementById('deal-email').value.trim();
  const phone = document.getElementById('deal-phone').value.trim();
  const stage = document.getElementById('deal-stage').value;
  const value = parseInt(document.getElementById('deal-value').value, 10) || 0;
  const priority = document.getElementById('deal-priority').value;
  const notes = document.getElementById('deal-notes').value.trim();
  const created_at = new Date().toLocaleDateString('pt-BR');

  if (!name) return;

  try {
    const res = await fetch('/api/leads', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, company, email, phone, stage, value, priority, notes, created_at })
    });
    if (res.ok) {
      closeNewDealModal();
      loadLeads();
      showToast('Novo lead cadastrado com sucesso');
    }
  } catch (err) {
    console.error('Falha ao cadastrar lead:', err);
  }
}

async function deleteDeal(id) {
  if (!confirm('Deseja realmente excluir esta oportunidade?')) return;
  try {
    const res = await fetch(`/api/leads/${id}`, { method: 'DELETE' });
    if (res.ok) {
      loadLeads();
      if (currentDealId === id) closeDrawer();
      showToast('Oportunidade excluída');
    }
  } catch (err) {
    console.error('Falha ao excluir:', err);
  }
}

/* Conexão WebSocket em Tempo Real */
function connectWebSocket() {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
  const wsUrl = `${scheme}://${location.host}/ws/crm`;
  const badge = document.getElementById('status-badge');
  const text = document.getElementById('status-text');

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    if (badge) badge.classList.add('connected');
    if (text) text.textContent = 'CRM Tempo Real Conectado';
    updateOnlineCount();
  };

  ws.onclose = () => {
    if (badge) badge.classList.remove('connected');
    if (text) text.textContent = 'Desconectado (Reconectando...)';
    setTimeout(connectWebSocket, 2000);
  };

  ws.onerror = () => {
    if (badge) badge.classList.remove('connected');
    if (text) text.textContent = 'Erro de Conexão WS';
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data.kind === 'sync') {
        loadLeads();
        showToast(data.message || 'Atualização no CRM em tempo real');
      } else if (data.kind === 'activity_sync') {
        if (currentDealId) loadActivities(currentDealId);
        showToast(data.message || 'Nova atividade registrada');
      } else if (data.kind === 'presence') {
        updateOnlineCount();
      }
    } catch (e) {
      loadLeads();
    }
  };
}

/* Inicialização */
window.addEventListener('DOMContentLoaded', () => {
  loadLeads();
  connectWebSocket();
  setInterval(updateOnlineCount, 8000);
});
