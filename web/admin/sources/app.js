import {
  loadAuth,
  persistAuth,
  clearAuth,
  syncSuiteCookieFromStorage,
  onHuberaHost,
} from '../../app/hubera-suite-auth.js';

const STREAM_OWNER_EMAILS = ['paul@delhomme.ovh'];
const PERSONAL_SOURCES_KEY = 'hubera.stream.personalSources.v1';

const $ = (sel, el = document) => el.querySelector(sel);
const app = $('#app');

let me = onHuberaHost() ? syncSuiteCookieFromStorage() || loadAuth() : loadAuth();
let sources = [];
let pendingEmail = '';
let pending2fa = false;
let loginErr = '';

function esc(s) {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/"/g, '&quot;');
}

function isOwner(email) {
  const e = String(email || '').trim().toLowerCase();
  return e && STREAM_OWNER_EMAILS.some((o) => o.toLowerCase() === e);
}

function readLocal() {
  try {
    const raw = localStorage.getItem(PERSONAL_SOURCES_KEY);
    const list = raw ? JSON.parse(raw) : [];
    return Array.isArray(list) ? list : [];
  } catch {
    return [];
  }
}

function writeLocal(list) {
  localStorage.setItem(PERSONAL_SOURCES_KEY, JSON.stringify(list));
}

async function loadRemote() {
  if (!me?.accessToken) {
    sources = readLocal();
    return;
  }
  try {
    const r = await fetch('/auth/me/preferences', {
      headers: { Authorization: `Bearer ${me.accessToken}` },
    });
    if (r.ok) {
      const j = await r.json();
      const remote = j?.preferences?.stream?.personalSources;
      if (Array.isArray(remote)) {
        sources = remote;
        writeLocal(remote);
        return;
      }
    }
  } catch { /* ignore */ }
  sources = readLocal();
}

async function saveRemote(list) {
  writeLocal(list);
  sources = list;
  if (!me?.accessToken) return;
  try {
    const r = await fetch('/auth/me/preferences', {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${me.accessToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        preferences: {
          stream: { personalSources: list },
        },
      }),
    });
    if (!r.ok) throw new Error(`Sync ${r.status}`);
  } catch (ex) {
    alert(`Sauvegardé localement. Sync cloud : ${ex.message}`);
  }
}

function uid() {
  if (crypto.randomUUID) return crypto.randomUUID();
  return `src-${Date.now().toString(36)}`;
}

function finishLogin(j, email) {
  persistAuth({
    accessToken: j.access_token,
    refreshToken: j.refresh_token || null,
    tenantId: Number(j.tenant_id) || 1,
    email: j.email || email || pendingEmail,
  });
  me = loadAuth();
  pending2fa = false;
  loginErr = '';
  boot();
}

function loginView(errMsg = '', twofa = false) {
  pending2fa = twofa;
  app.innerHTML = `
    <main>
      <h1>Sources personnelles</h1>
      <p class="muted">Réservé au propriétaire Hubera Stream. Connexion Hubera ID requise (cookie suite).</p>
      <form id="login" class="card">
        ${twofa ? `
          <label for="code">Code 2FA</label>
          <input id="code" name="code" inputmode="numeric" autocomplete="one-time-code" required autofocus />
        ` : `
          <label for="email">E-mail</label>
          <input id="email" name="email" type="email" autocomplete="username" required autofocus value="${esc(pendingEmail)}" />
          <label for="password">Mot de passe</label>
          <input id="password" name="password" type="password" autocomplete="current-password" required />
        `}
        ${errMsg ? `<p class="err">${esc(errMsg)}</p>` : ''}
        <button class="primary" type="submit">${twofa ? 'Valider' : 'Entrer'}</button>
      </form>
      <p><a href="/app/">← Catalogue Stream</a></p>
    </main>`;
  $('#login').onsubmit = async (e) => {
    e.preventDefault();
    const form = e.target;
    try {
      if (twofa) {
        const r = await fetch('/auth/2fa/verify', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: pendingEmail, code: form.code.value.trim(), tenant_id: '1' }),
        });
        const j = await r.json().catch(() => ({}));
        if (!r.ok) throw new Error(j.error || j.message || 'Code invalide');
        finishLogin(j);
        return;
      }
      pendingEmail = form.email.value.trim();
      const r = await fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: pendingEmail, password: form.password.value, tenant_id: '1' }),
      });
      const j = await r.json().catch(() => ({}));
      if (!r.ok) throw new Error(j.error || j.message || `Connexion impossible (${r.status})`);
      if (j.requires_2fa) {
        loginView('', true);
        return;
      }
      finishLogin(j, pendingEmail);
    } catch (ex) {
      loginView(ex.message, twofa);
    }
  };
}

function deniedView() {
  app.innerHTML = `
    <main>
      <h1>Accès refusé</h1>
      <p>Cette page admin est réservée au propriétaire (<code>${esc(STREAM_OWNER_EMAILS[0])}</code>).</p>
      <p><a href="/app/">← Catalogue Stream</a> · <a href="/admin/torrents/">Note légale torrent</a></p>
    </main>`;
}

function adminView() {
  app.innerHTML = `
    <main>
      <header class="top">
        <h1>Sources personnelles</h1>
        <span class="who">${esc(me.email)}</span>
      </header>
      <p class="legal">Ajoute des liens <strong>privés</strong> (magnet ou URL de ton site). Hubera Stream n’héberge pas de client torrent et ne publie pas d’index.</p>
      <form id="add" class="card">
        <label for="label">Libellé</label>
        <input id="label" required placeholder="Ex. Flux perso NAS" />
        <label for="kind">Type</label>
        <select id="kind">
          <option value="url">URL site / flux perso</option>
          <option value="magnet">Magnet (client local)</option>
        </select>
        <label for="target">URL ou magnet</label>
        <textarea id="target" required placeholder="https://… ou magnet:?xt=…"></textarea>
        <label for="note">Note (optionnel)</label>
        <input id="note" placeholder="Rappel usage perso" />
        <button class="primary" type="submit">Ajouter</button>
      </form>
      <section class="list">
        <h2>Liste (${sources.length})</h2>
        ${sources.length ? sources.map((s) => `
          <div class="card item">
            <div>
              <strong>${esc(s.label)}</strong>
              <div class="muted">${esc(s.kind)} · ${esc((s.url || s.magnet || '').slice(0, 80))}</div>
            </div>
            <button type="button" class="ghost danger" data-del="${esc(s.id)}">Supprimer</button>
          </div>
        `).join('') : '<p class="muted">Aucune source — le catalogue public reste Pluto + Arte.</p>'}
      </section>
      <p class="row">
        <a href="/app/">← Catalogue</a>
        <button type="button" class="ghost" id="logout">Déconnexion</button>
      </p>
    </main>`;

  $('#add').onsubmit = async (e) => {
    e.preventDefault();
    const kind = $('#kind').value;
    const target = $('#target').value.trim();
    const entry = {
      id: uid(),
      label: $('#label').value.trim(),
      kind,
      note: $('#note').value.trim(),
      url: kind === 'url' ? target : '',
      magnet: kind === 'magnet' ? target : '',
    };
    await saveRemote([...sources, entry]);
    adminView();
  };

  document.querySelectorAll('[data-del]').forEach((b) => {
    b.onclick = async () => {
      const id = b.dataset.del;
      await saveRemote(sources.filter((s) => s.id !== id));
      adminView();
    };
  });

  $('#logout').onclick = () => {
    clearAuth();
    me = null;
    loginView();
  };
}

async function boot() {
  if (!me?.accessToken) {
    loginView(loginErr);
    return;
  }
  if (!isOwner(me.email)) {
    deniedView();
    return;
  }
  await loadRemote();
  adminView();
}

boot();
