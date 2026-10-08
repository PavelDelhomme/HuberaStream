const $ = (sel, el = document) => el.querySelector(sel);
const app = $('#app');
const COOKIE = 'hubera_suite_auth';

let source = 'all';
let query = '';
let items = [];
let err = '';
let loading = true;
let open = null;
let player = null;
let me = null;

function esc(s) {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/"/g, '&quot;');
}

function onHuberaHost() {
  const h = location.hostname.toLowerCase();
  return h === 'hubera.cloud' || h.endsWith('.hubera.cloud');
}

function parseAuth(raw) {
  if (!raw) return null;
  try {
    const d = JSON.parse(raw);
    if (d.accessToken || d.email) {
      return { email: d.email || null, accessToken: d.accessToken || null };
    }
  } catch { /* ignore */ }
  return null;
}

function readSuiteCookie() {
  for (const part of document.cookie.split(';')) {
    const [k, ...rest] = part.trim().split('=');
    if (k !== COOKIE) continue;
    try {
      return parseAuth(decodeURIComponent(rest.join('=')));
    } catch {
      return null;
    }
  }
  return null;
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
  const results = await Promise.allSettled([loadPluto(), loadArte()]);
  const out = [];
  const parts = [];
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
  if (me?.email) {
    return `<span class="who" title="Hubera ID">${esc(me.email)}</span>`;
  }
  return `<a class="id-link" href="https://id.hubera.cloud">Hubera ID</a>`;
}

function cardHtml(it) {
  const img = it.image
    ? `<img src="${esc(it.image)}" alt="" loading="lazy" />`
    : `<div class="ph">${it.source === 'pluto' ? 'Pluto' : 'Arte'}</div>`;
  return `<button class="card" data-open="${esc(it.key)}">${img}<div class="meta"><b>${esc(it.title)}</b><small>${esc(it.source === 'pluto' ? 'Pluto TV · live' : 'Arte.tv')} · ${esc(it.subtitle)}</small></div></button>`;
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
    </div>
    <p class="legal">Catalogue légal uniquement : APIs publiques Pluto TV et Arte.tv. Lecture via les lecteurs / sites officiels. Pas de torrent, pas de miroir, pas de contournement DRM.</p>
    ${err ? `<p class="err">${esc(err)}</p>` : ''}
    ${loading ? `<p class="muted">Chargement du catalogue officiel…</p>` : ''}
    ${!loading && !list.length ? `<p class="muted">Aucun titre.</p>` : ''}
    <div class="grid">${list.map(cardHtml).join('')}</div>
    ${sheetHtml(open)}
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
}

me = onHuberaHost() ? readSuiteCookie() : parseAuth(localStorage.getItem('cloudity_admin_auth'));
render();
loadAll();
