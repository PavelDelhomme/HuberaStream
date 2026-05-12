'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminWebhooks.css'

type WebhookSetup = {
  secret_configured: boolean
  secret_suffix: string
  public_base_url: string
  paths: { radarr: string; sonarr: string }
}

type WebhookEvent = {
  id: string
  source: string
  event_type: string
  payload: string
  created_at: string
}

function apiOrigin(): string {
  const u = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3300/api'
  return u.replace(/\/api\/?$/, '')
}

export default function AdminWebhooksPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [setup, setSetup] = useState<WebhookSetup | null>(null)
  const [events, setEvents] = useState<WebhookEvent[]>([])
  const [manualToken, setManualToken] = useState('')

  const baseForUrls = useMemo(() => {
    if (!setup) return apiOrigin()
    return setup.public_base_url?.trim() || apiOrigin()
  }, [setup])

  const radarrUrl = useMemo(() => {
    const path = setup?.paths.radarr || '/api/webhooks/radarr'
    if (!manualToken.trim()) return `${baseForUrls}${path}?token=`
    return `${baseForUrls}${path}?token=${encodeURIComponent(manualToken.trim())}`
  }, [baseForUrls, setup, manualToken])

  const sonarrUrl = useMemo(() => {
    const path = setup?.paths.sonarr || '/api/webhooks/sonarr'
    if (!manualToken.trim()) return `${baseForUrls}${path}?token=`
    return `${baseForUrls}${path}?token=${encodeURIComponent(manualToken.trim())}`
  }, [baseForUrls, setup, manualToken])

  const loadData = useCallback(async () => {
    const [s, e] = await Promise.all([
      api.get('/admin/webhooks/setup'),
      api.get('/admin/webhooks/events?limit=40'),
    ])
    setSetup(s.data)
    setEvents(e.data?.events || [])
  }, [])

  useEffect(() => {
    const run = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/webhooks')
        return
      }
      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }
      setUser(currentUser)
      try {
        await loadData()
      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    run()
  }, [router, loadData])

  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      alert('Copié dans le presse-papiers')
    } catch {
      alert('Impossible de copier (navigateur)')
    }
  }

  if (loading) {
    return (
      <div className="admin-webhooks">
        <div className="loading">Chargement…</div>
      </div>
    )
  }

  return (
    <div className="admin-webhooks">
      <div className="admin-header">
        <h1>Webhooks Radarr / Sonarr</h1>
        <p className="subtitle">
          Radarr et Sonarr appellent ces URL après import ou téléchargement. Le token doit être identique à{' '}
          <code>WEBHOOK_SECRET</code> côté backend (Portainer / .env). Le secret n’est jamais renvoyé par l’API : saisis-le
          ci-dessous uniquement pour générer l’URL à coller dans Radarr/Sonarr (reste dans ton navigateur).
        </p>
      </div>

      <div className="panel">
        <h2>Perf admin : Next.js ou autre ?</h2>
        <div className="perf-note">
          Pour une <strong>zone admin</strong> (tableaux, formulaires), Next.js 14 en mode client est déjà très proche d’une
          SPA Vite/React : le gain d’un autre framework (Astro, Remix, Qwik…) serait marginal sur le temps de
          chargement réel. Les alternatives deviennent intéressantes surtout pour du <strong>site marketing</strong> très
          statique (Astro) ou des contraintes extrêmes de TTI sur mobile bas de gamme. Remplacer Next ici doublerait la
          maintenance sans bénéfice mesurable pour StreamMake.
          <ul>
            <li>Optimiser plutôt : <code>dynamic import</code> des gros modules, images <code>next/image</code>, limiter les bundles admin.</li>
          </ul>
        </div>
      </div>

      <div className="panel">
        <h2>Configuration</h2>
        <p className="subtitle">
          Backend : <code>WEBHOOK_SECRET</code> obligatoire pour accepter les webhooks. Optionnel :{' '}
          <code>PUBLIC_WEBHOOK_BASE_URL</code> (ex. <code>https://stream.delhomme.ovh</code>) si l’URL publique diffère de
          celle déduite de <code>NEXT_PUBLIC_API_URL</code>.
        </p>
        {setup && (
          <p className="subtitle" style={{ marginTop: 10 }}>
            Secret configuré côté serveur : {setup.secret_configured ? `oui (${setup.secret_suffix || '****'})` : 'non — les webhooks seront refusés (401).'}
          </p>
        )}
        <label htmlFor="wh-token">Token webhook (saisi localement pour prévisualiser l’URL)</label>
        <input
          id="wh-token"
          type="password"
          autoComplete="off"
          placeholder="Collez ici la même valeur que WEBHOOK_SECRET"
          value={manualToken}
          onChange={(e) => setManualToken(e.target.value)}
        />
        <div className="warn">
          Dans Radarr / Sonarr : Paramètres → Connect → Webhook → URL complète ci-dessous, méthode POST, JSON. Déclencheurs
          conseillés : On Import, On Download, Grab (selon ton flux).
        </div>
      </div>

      <div className="panel">
        <h2>URL Radarr</h2>
        <div className="url-row">
          <code>{radarrUrl}</code>
          <button type="button" onClick={() => copy(radarrUrl)} disabled={!manualToken.trim()}>
            Copier
          </button>
        </div>
        <h2>URL Sonarr</h2>
        <div className="url-row">
          <code>{sonarrUrl}</code>
          <button type="button" onClick={() => copy(sonarrUrl)} disabled={!manualToken.trim()}>
            Copier
          </button>
        </div>
      </div>

      <div className="panel">
        <h2>Derniers événements</h2>
        <button type="button" style={{ marginBottom: 14 }} onClick={() => loadData()}>
          Rafraîchir
        </button>
        {events.length === 0 ? (
          <p className="subtitle">Aucun webhook enregistré pour l’instant.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Date</th>
                <th>Source</th>
                <th>EventType</th>
                <th>Payload (extrait)</th>
              </tr>
            </thead>
            <tbody>
              {events.map((ev) => (
                <tr key={ev.id}>
                  <td>{new Date(ev.created_at).toLocaleString()}</td>
                  <td>{ev.source}</td>
                  <td>{ev.event_type}</td>
                  <td>
                    <pre>{ev.payload?.slice(0, 600) || ''}{ev.payload?.length > 600 ? '…' : ''}</pre>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {user && (
        <p className="subtitle" style={{ marginTop: 24 }}>
          Connecté en tant que {user.email}
        </p>
      )}
    </div>
  )
}
