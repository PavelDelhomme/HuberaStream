import {
  loadAuth,
  persistAuth,
  clearAuth,
  syncSuiteCookieFromStorage,
  onHuberaHost,
} from './hubera-suite-auth.js';

const $ = (sel, el = document) => el.querySelector(sel);
const app = $('#app');

const STREAM_OWNER_EMAILS = ['paul@delhomme.ovh'];
const PERSONAL_SOURCES_KEY = 'hubera.stream.personalSources.v1';

let source = 'all';
let query = '';
let items = [];
let err = '';
let loading = true;
let open = null;
let player = null;
let me = null;
let pendingEmail = '';
let pending2fa = false;
let authSheet = false;
let loginErr = '';

function esc(s) {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/"/g, '&quot;');
}

function isStreamOwner(email) {
  const e = String(email || '').trim().toLowerCase();
  return e && STREAM_OWNER_EMAILS.some((o) => o.toLowerCase() === e);
}

function readPersonalSourcesLocal() {
  try {
    const raw = localStorage.getItem(PERSONAL_SOURCES_KEY);
    const list = raw ? JSON.parse(raw) : [];
    return Array.isArray(list) ? list : [];
  } catch {
    return [];
  }
}

function writePersonalSourcesLocal(list) {
  localStorage.setItem(PERSONAL_SOURCES_KEY, JSON.stringify(list));
}

function personalSourcesToItems(list) {
  return list
    .filter((s) => s && s.label && (s.url || s.magnet))
    .map((s) => {
      const id = s.id || uuid();
      const isMagnet = s.kind === 'magnet' || String(s.magnet || s.url || '').startsWith('magnet:');
      const target = isMagnet ? (s.magnet || s.url) : (s.url || '');
      return {
        key: `personal:${id}`,
        source: 'personal',
        id,
        title: s.label,
        subtitle: isMagnet ? 'Source personnelle · magnet' : 'Source personnelle · lien',
        summary: s.note || 'Catalogue privé (owner). Pas d’index public Hubera.',
        image: '',
        officialUrl: isMagnet ? target : target,
        isMagnet,
        kind: 'link',
      };
    });
}

async function loadPersonalSourcesRemote() {
  if (!me?.accessToken) return readPersonalSourcesLocal();
  try {
    const r = await fetch('/auth/me/preferences', {
      headers: { Authorization: `Bearer ${me.accessToken}` },
    });
    if (!r.ok) return readPersonalSourcesLocal();
    const j = await r.json();
    const remote = j?.preferences?.stream?.personalSources;
    if (Array.isArray(remote)) {
      writePersonalSourcesLocal(remote);
      return remote;
    }
  } catch { /* ignore */ }
  return readPersonalSourcesLocal();
}

function finishLogin(j, email) {
  persistAuth({
    accessToken: j.access_token,
    refreshToken: j.refresh_token || null,
    tenantId: Number(j.tenant_id) || 1,
    email: j.email || email || pendingEmail,
  });
  me = loadAuth();
  authSheet = false;
  loginErr = '';
  pending2fa = false;
  loadAll();
}

function loginSheetHtml(errMsg = '', twofa = false) {
  pending2fa = twofa;
  return `<div class="sheet-bg" data-auth-close>
    <div class="sheet login-sheet" role="dialog" aria-label="Hubera ID">
      <h2>Hubera ID</h2>
      <p class="muted">Même compte que Mail, Drive et Calendar. Le cookie <code>hubera_suite_auth</code> est posé sur <code>.hubera.cloud</code> après connexion.</p>
      <form id="login">
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
        <div class="row">
          <button class="primary" type="submit">${twofa ? 'Valider' : 'Entrer'}</button>
          <button type="button" class="ghost" data-auth-close>Annuler</button>
        </div>
      </form>
    </div>
  </div>`;
}

function bindLoginSheet() {
  $('#login')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    try {
      if (pending2fa) {
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
        pending2fa = true;
        authSheet = true;
        loginErr = '';
        render();
        return;
      }
      finishLogin(j, pendingEmail);
    } catch (ex) {
      authSheet = true;
      loginErr = ex.message || 'Connexion impossible';
      render();
    }
  });
  document.querySelectorAll('[data-auth-close]').forEach((b) => {
    b.onclick = (ev) => {
      if (ev.currentTarget.classList.contains('sheet-bg') && ev.target !== ev.currentTarget) return;
      authSheet = false;
      render();
    };
  });
}

function imgUrl(raw, fallbackSize = '440x248') {
  if (!raw) return '';
  if (typeof raw === 'string') return raw.replace('__SIZE__', fallbackSize);
  const u = raw.url || raw.path || raw.src || '';
  return String(u).replace('__SIZE__', fallbackSize);
}

function uuid() {
  if (crypto.randomUUID) return crypto.randomUUID();
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
}

async function loadPluto() {
  const r = await fetch('/api/catalog/pluto/channels');
  if (!r.ok) throw new Error(`Pluto TV ${r.status}`);
  const list = await r.json();
  if (!Array.isArray(list)) return [];
  return list
    .filter((c) => c && !c.plutoOfficeOnly && c.isStitched !== false)
    .map((c) => {
      const id = c._id || c.id;
      const slug = c.slug || id;
      const hls = c.stitched?.urls?.find((u) => u.type === 'hls')?.url || '';
      return {
        key: `pluto:${id}`,
        source: 'pluto',
        id,
        slug,
        title: c.name || 'Sans titre',
        subtitle: c.category || 'Chaîne live',
        summary: c.summary || c.onDemandDescription || '',
        image: imgUrl(c.featuredImage) || imgUrl(c.thumbnail) || imgUrl(c.colorLogoPNG),
        officialUrl: `https://pluto.tv/fr/live-tv/${encodeURIComponent(slug)}`,
        hlsHint: hls,
        kind: 'live',
      };
    });
}

function arteImage(item) {
  const img = item.mainImage || item.images?.[0] || item.graphicalTitle;
  return imgUrl(img, '440x248');
}

const ARTE_ID = /^(?:\d{6}-\d{3}-[A-Z]|RC-\d+|PCF\d+)$/;

function walkArte(node, acc) {
  if (!node) return;
  if (Array.isArray(node)) {
    node.forEach((n) => walkArte(n, acc));
    return;
  }
  if (typeof node !== 'object') return;
  const id = node.programId || node.id;
  if (node.title && ARTE_ID.test(String(id || ''))) {
    const kindCode = node.kind?.code || node.type || '';
    const official = node.deeplink || node.url || `https://www.arte.tv/fr/videos/${encodeURIComponent(id)}/`;
    acc.push({
      key: `arte:${id}`,
      source: 'arte',
      id,
      title: node.title,
      subtitle: node.subtitle || node.kind?.label || 'Arte.tv',
      summary: node.shortDescription || node.description || '',
      image: arteImage(node),
      officialUrl: official,
      embedUrl: /^\d{6}-\d{3}-[A-Z]$/.test(id)
        ? `https://www.arte.tv/embeds/fr/${encodeURIComponent(id)}`
        : '',
      kind: kindCode === 'TV_SERIES' ? 'serie' : 'vod',
    });
  }
  const skip = new Set(['parent', 'trackingPixel', 'stats', 'alternativeLanguages']);
  Object.entries(node).forEach(([k, v]) => {
    if (skip.has(k) || !v || typeof v !== 'object') return;
    walkArte(v, acc);
  });
}

async function loadArte() {
  const r = await fetch('/api/catalog/arte/home');
  if (!r.ok) throw new Error(`Arte.tv ${r.status}`);
  const data = await r.json();
  const acc = [];
  walkArte(data, acc);
  const seen = new Set();
  return acc.filter((it) => {
    if (!it.id || seen.has(it.id)) return false;
    seen.add(it.id);
    return true;
  });
}

async function loadAll() {
  loading = true;
  err = '';
  render();
  const personalRaw = isStreamOwner(me?.email) ? await loadPersonalSourcesRemote() : [];
  const personal = isStreamOwner(me?.email) ? personalSourcesToItems(personalRaw) : [];
  const results = await Promise.allSettled([loadPluto(), loadArte()]);
  const out = [...personal];
  const parts = [];
  if (personal.length) parts.push(`Perso ${personal.length}`);
  results.forEach((r, i) => {
    const name = i === 0 ? 'Pluto TV' : 'Arte.tv';
    if (r.status === 'fulfilled') {
      out.push(...r.value);
      parts.push(`${name} ${r.value.length}`);
    } else {
      parts.push(`${name} indisponible`);
      err = [err, r.reason?.message || String(r.reason)].filter(Boolean).join(' · ');
    }
  });
  items = out;
  loading = false;
  if (!out.length && !err) err = 'Catalogue vide';
  render();
  return parts;
}

function filtered() {
  const q = query.trim().toLowerCase();
  return items.filter((it) => {
    if (source !== 'all' && it.source !== source) return false;
    if (!q) return true;
    return `${it.title} ${it.subtitle} ${it.summary}`.toLowerCase().includes(q);
  });
}

function accountHtml() {
  const bits = [];
  if (me?.email) {
    bits.push(`<span class="who" title="Hubera ID">${esc(me.email)}</span>`);
    if (isStreamOwner(me.email)) {
      bits.push(`<a class="id-link" href="/admin/sources/">Sources perso</a>`);
    }
    bits.push(`<button type="button" class="ghost tiny" id="logout">Déconnexion</button>`);
  } else if (onHuberaHost()) {
    bits.push(`<button type="button" class="id-link" id="open-login">Hubera ID</button>`);
  } else {
    bits.push(`<a class="id-link" href="https://id.hubera.cloud">Hubera ID</a>`);
  }
  return bits.join('');
}

function cardHtml(it) {
  const img = it.image
    ? `<img src="${esc(it.image)}" alt="" loading="lazy" />`
    : `<div class="ph">${it.source === 'pluto' ? 'Pluto' : it.source === 'personal' ? 'Perso' : 'Arte'}</div>`;
  const srcLabel =
    it.source === 'pluto' ? 'Pluto TV · live' : it.source === 'personal' ? 'Perso' : 'Arte.tv';
  return `<button class="card" data-open="${esc(it.key)}">${img}<div class="meta"><b>${esc(it.title)}</b><small>${esc(srcLabel)} · ${esc(it.subtitle)}</small></div></button>`;
}

function sheetHtml(it) {
  if (!it) return '';
  const playable = it.source === 'arte' && it.embedUrl;
  const hero = it.image ? `<img class="hero" src="${esc(it.image)}" alt="" />` : '';
  let playerBlock = '';
  if (player === it.key && playable) {
    playerBlock = `<iframe class="player" src="${esc(it.embedUrl)}" allow="fullscreen; autoplay; encrypted-media" allowfullscreen title="${esc(it.title)}"></iframe>
      <p class="player-note">Lecteur officiel Arte.tv (embed autorisé). Géoblocage et pubs selon les ToS Arte.</p>`;
  } else if (player === it.key && it.source === 'pluto') {
    playerBlock = `<iframe class="player" src="${esc(it.officialUrl)}" allow="fullscreen; autoplay; encrypted-media" allowfullscreen title="${esc(it.title)}"></iframe>
      <p class="player-note">Chaîne Pluto TV officielle (FAST, publicités). Si l’iframe est bloquée, ouvre le site Pluto.</p>`;
  }
  if (it.source === 'personal') {
    const href = esc(it.officialUrl);
    const openBtn = it.isMagnet
      ? `<a class="primary" href="${href}">Ouvrir le magnet (client local)</a>`
      : `<a class="primary" href="${href}" target="_blank" rel="noopener">Ouvrir le site</a>`;
    return `<div class="sheet-bg" data-close>
      <div class="sheet" role="dialog" aria-label="${esc(it.title)}">
        <h2>${esc(it.title)}</h2>
        <p><b>Source personnelle</b> · ${esc(it.subtitle)}</p>
        <p>${esc(it.summary)}</p>
        <p class="player-note">Hubera ne diffuse pas ce flux : lien privé owner uniquement, pas d’index torrent public.</p>
        <div class="row">${openBtn}<button type="button" class="ghost" data-close>Fermer</button></div>
      </div>
    </div>`;
  }
  return `<div class="sheet-bg" data-close>
    <div class="sheet" role="dialog" aria-label="${esc(it.title)}">
      ${hero}
      <h2>${esc(it.title)}</h2>
      <p><b>${it.source === 'pluto' ? 'Pluto TV' : 'Arte.tv'}</b> · ${esc(it.subtitle)}</p>
      <p>${esc(it.summary) || 'Pas de synopsis.'}</p>
      ${playerBlock}
      <div class="row">
        ${playable || it.source === 'pluto' ? `<button type="button" class="primary" data-play="${esc(it.key)}">${player === it.key ? 'Recharger' : 'Regarder ici'}</button>` : ''}
        <a class="ghost" href="${esc(it.officialUrl)}" target="_blank" rel="noopener">Ouvrir la source officielle</a>
        <button type="button" class="ghost" data-close>Fermer</button>
      </div>
    </div>
  </div>`;
}

function render() {
  const list = filtered();
  app.innerHTML = `
    <header class="top">
      <h1>Hubera Stream</h1>
      ${accountHtml()}
    </header>
    <input class="search" id="q" type="search" placeholder="Titre, chaîne, documentaire…" value="${esc(query)}" />
    <div class="tabs">
      <button type="button" data-src="all" class="${source === 'all' ? 'on' : ''}">Tout</button>
      <button type="button" data-src="pluto" class="${source === 'pluto' ? 'on' : ''}">Pluto TV</button>
      <button type="button" data-src="arte" class="${source === 'arte' ? 'on' : ''}">Arte.tv</button>
      ${isStreamOwner(me?.email) ? `<button type="button" data-src="personal" class="${source === 'personal' ? 'on' : ''}">Perso</button>` : ''}
    </div>
    <p class="legal">Catalogue légal uniquement : APIs publiques Pluto TV et Arte.tv. Lecture via les lecteurs / sites officiels. Pas de torrent, pas de miroir, pas de contournement DRM.</p>
    ${err ? `<p class="err">${esc(err)}</p>` : ''}
    ${loading ? `<p class="muted">Chargement du catalogue officiel…</p>` : ''}
    ${!loading && !list.length ? `<p class="muted">Aucun titre.</p>` : ''}
    <div class="grid">${list.map(cardHtml).join('')}</div>
    ${sheetHtml(open)}
    ${authSheet ? loginSheetHtml(loginErr, pending2fa) : ''}
  `;
  $('#q')?.addEventListener('input', (e) => {
    query = e.target.value;
    render();
    const n = $('#q');
    if (n) {
      n.focus();
      n.setSelectionRange(query.length, query.length);
    }
  });
  document.querySelectorAll('[data-src]').forEach((b) => {
    b.onclick = () => {
      source = b.dataset.src;
      render();
    };
  });
  document.querySelectorAll('[data-open]').forEach((b) => {
    b.onclick = () => {
      open = items.find((it) => it.key === b.dataset.open) || null;
      player = null;
      render();
    };
  });
  document.querySelectorAll('[data-close]').forEach((b) => {
    b.onclick = (e) => {
      if (e.currentTarget.classList.contains('sheet-bg') && e.target !== e.currentTarget) return;
      open = null;
      player = null;
      render();
    };
  });
  document.querySelectorAll('[data-play]').forEach((b) => {
    b.onclick = (e) => {
      e.stopPropagation();
      player = b.dataset.play;
      render();
    };
  });
  $('#open-login')?.addEventListener('click', () => {
    authSheet = true;
    loginErr = '';
    render();
  });
  $('#logout')?.addEventListener('click', () => {
    me = null;
    clearAuth();
    loadAll();
  });
  if (authSheet) bindLoginSheet();
}

me = onHuberaHost() ? syncSuiteCookieFromStorage() || loadAuth() : loadAuth();
render();
loadAll();
