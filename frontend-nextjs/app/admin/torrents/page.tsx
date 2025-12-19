'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminTorrents.css'

export default function AdminTorrentsPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [torrents, setTorrents] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true)
  const torrentsRef = useRef<any[]>([])

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/torrents')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchTorrents()
    }
    checkAuth()
  }, [router])

  useEffect(() => {
    torrentsRef.current = torrents
  }, [torrents])

  useEffect(() => {
    const interval = setInterval(() => {
      if (autoRefreshEnabled && user) {
        const hasActiveTorrents = torrentsRef.current.some((t: any) => 
          t.status === 'pending' || t.status === 'downloading' || t.status === 'seeding'
        )
        if (hasActiveTorrents) {
          fetchTorrentsBackground()
        }
      }
    }, 5000)
    return () => clearInterval(interval)
  }, [autoRefreshEnabled, user])

  const fetchTorrents = async () => {
    try {
      setLoading(true)
      const response = await api.get('/admin/torrents')
      setTorrents(response.data.torrents || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchTorrentsBackground = async () => {
    try {
      const response = await api.get('/admin/torrents')
      setTorrents(response.data.torrents || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const handlePause = async (id: string) => {
    try {
      await api.post(`/admin/torrents/${id}/pause`)
      fetchTorrentsBackground()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la mise en pause')
    }
  }

  const handleResume = async (id: string) => {
    try {
      await api.post(`/admin/torrents/${id}/resume`)
      fetchTorrentsBackground()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la reprise')
    }
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer ce torrent ?')) {
      return
    }
    try {
      await api.delete(`/admin/torrents/${id}`)
      fetchTorrentsBackground()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la suppression')
    }
  }

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  if (loading) {
    return (
      <div className="admin-torrents">
        <div className="loading">Chargement des torrents...</div>
      </div>
    )
  }

  return (
    <div className="admin-torrents">
      <div className="admin-header">
        <h1>📥 Gestion des Torrents</h1>
        <div className="header-controls">
          <button
            onClick={() => setAutoRefreshEnabled(!autoRefreshEnabled)}
            className={`toggle-refresh ${autoRefreshEnabled ? 'active' : ''}`}
          >
            {autoRefreshEnabled ? '⏸️ Auto-refresh ON' : '▶️ Auto-refresh OFF'}
          </button>
          <button onClick={fetchTorrents} className="refresh-button">
            🔄 Rafraîchir
          </button>
        </div>
      </div>

      {torrents.length === 0 ? (
        <div className="empty-state">
          <p>Aucun torrent téléchargé</p>
        </div>
      ) : (
        <div className="torrents-list">
          {torrents.map((torrent: any) => (
            <div key={torrent.id} className="torrent-card">
              <div className="torrent-header">
                <h3>{torrent.name}</h3>
                <span className={`status-badge status-${torrent.status}`}>
                  {torrent.status}
                </span>
              </div>
              <div className="torrent-info">
                <span>📦 {formatBytes(torrent.size)}</span>
                <span>⬇️ {formatBytes(torrent.downloaded)}</span>
                <span>📊 {torrent.progress?.toFixed(1) || 0}%</span>
              </div>
              <div className="torrent-actions">
                {torrent.status === 'paused' || torrent.status === 'error' ? (
                  <button onClick={() => handleResume(torrent.id)} className="btn-resume">
                    ▶️ Reprendre
                  </button>
                ) : (
                  <button onClick={() => handlePause(torrent.id)} className="btn-pause">
                    ⏸️ Pause
                  </button>
                )}
                <button onClick={() => handleDelete(torrent.id)} className="btn-delete">
                  🗑️ Supprimer
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

