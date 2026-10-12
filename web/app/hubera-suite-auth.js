/**
 * Cookie SSO suite Hubera — copie déployée (Docker web/ only).
 * Source canonique : platform/suite-cookie/hubera-suite-auth.js
 */
export const HUBERA_SUITE_COOKIE = 'hubera_suite_auth';
export const AUTH_STORAGE_KEY = 'cloudity_admin_auth';

export function onHuberaHost() {
  const h = location.hostname.toLowerCase();
  return h === 'hubera.cloud' || h.endsWith('.hubera.cloud');
}

export function parseAuth(raw) {
  if (!raw) return null;
  try {
    const d = JSON.parse(raw);
    if (d.accessToken && d.tenantId != null) {
      return {
        accessToken: d.accessToken,
        refreshToken: d.refreshToken || null,
        tenantId: d.tenantId,
        email: d.email || null,
      };
    }
    if (d.accessToken || d.email) {
      return {
        accessToken: d.accessToken || null,
        refreshToken: d.refreshToken || null,
        tenantId: d.tenantId != null ? Number(d.tenantId) : null,
        email: d.email || null,
      };
    }
  } catch {
    /* ignore */
  }
  return null;
}

export function readSuiteCookie() {
  for (const part of document.cookie.split(';')) {
    const [k, ...rest] = part.trim().split('=');
    if (k !== HUBERA_SUITE_COOKIE) continue;
    try {
      return parseAuth(decodeURIComponent(rest.join('=')));
    } catch {
      return null;
    }
  }
  return null;
}

export function writeSuiteCookie(state) {
  if (!onHuberaHost()) return;
  if (state?.accessToken && state.tenantId != null) {
    const payload = encodeURIComponent(JSON.stringify(state));
    if (payload.length > 3500) return;
    document.cookie = `${HUBERA_SUITE_COOKIE}=${payload}; Domain=.hubera.cloud; Path=/; Secure; SameSite=Lax; Max-Age=1209600`;
  } else {
    document.cookie = `${HUBERA_SUITE_COOKIE}=; Domain=.hubera.cloud; Path=/; Secure; SameSite=Lax; Max-Age=0`;
  }
}

export function loadAuth() {
  return parseAuth(localStorage.getItem(AUTH_STORAGE_KEY)) || readSuiteCookie();
}

export function persistAuth(state) {
  if (state?.accessToken && state.tenantId != null) {
    localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(state));
    writeSuiteCookie(state);
  } else if (state?.accessToken || state?.email) {
    localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(state));
    writeSuiteCookie(state);
  } else {
    localStorage.removeItem(AUTH_STORAGE_KEY);
    writeSuiteCookie(null);
  }
}

export function clearAuth() {
  localStorage.removeItem(AUTH_STORAGE_KEY);
  writeSuiteCookie(null);
}

export function syncSuiteCookieFromStorage() {
  const fromLs = parseAuth(localStorage.getItem(AUTH_STORAGE_KEY));
  if (!fromLs?.accessToken) return fromLs;
  const fromCookie = readSuiteCookie();
  if (!fromCookie?.accessToken) {
    writeSuiteCookie(fromLs);
  }
  return fromLs || fromCookie;
}
