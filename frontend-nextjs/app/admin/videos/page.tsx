'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminVideos.css'

export default function AdminVideosPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [videos, setVideos] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [filterStatus, setFilterStatus] = useState('all')
  const [cleaning, setCleaning] = useState(false)
  const [invalidVideos, setInvalidVideos] = useState<any[]>([])
  const [showInvalidList, setShowInvalidList] = useState(false)

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/videos')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchVideos()
      fetchInvalidVideos()
    }
    checkAuth()
  }, [router])

  useEffect(() => {
    if (user) {
      fetchVideos()
    }
  }, [searchQuery, filterStatus, user])

  const fetchVideos = async () => {
    try {
      setLoading(true)
      setError(null)
      const params: any = {
        page: 1,
        limit: 1000,
        status: 'all',
      }
      if (searchQuery) {
        params.search = searchQuery
      }
      const response = await api.get('/videos', { params })
      
      let allVideos: any[] = []
      if (Array.isArray(response.data)) {
        allVideos = response.data
      } else if (response.data?.videos) {
        allVideos = response.data.videos
      } else if (response.data?.data?.videos) {
        allVideos = response.data.data.videos
      }
      
      let validVideos = allVideos.filter((v: any) => v && v.id)
      let filteredVideos = validVideos.filter((v: any) => v.status !== 'cancelled')
      if (filterStatus !== 'all') {
        filteredVideos = filteredVideos.filter((v: any) => v.status === filterStatus)
      }
      
      setVideos(filteredVideos)
    } catch (err: any) {
      console.error('Erreur lors de la récupération des vidéos:', err)
      setError('Erreur lors de la récupération des vidéos: ' + (err.response?.data?.error || err.message))
    } finally {
      setLoading(false)
    }
  }

  const fetchInvalidVideos = async (): Promise<any[]> => {
    try {
      const response = await api.get('/admin/videos/invalid')
      const invalid = response.data.invalid_videos || []
      setInvalidVideos(invalid)
      return invalid
    } catch (err) {
      console.error('Erreur lors de la récupération des vidéos invalides:', err)
      setInvalidVideos([])
      return []
    }
  }

  const handleDelete = async (videoId: string) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer cette vidéo ?')) {
      return
    }

    try {
      await api.delete(`/videos/${videoId}`)
      fetchVideos()
    } catch (err: any) {
      console.error('Erreur lors de la suppression:', err)
      alert('Erreur lors de la suppression de la vidéo')
    }
  }

  const handleCleanInvalidVideos = async () => {
    const invalid = await fetchInvalidVideos()
    
    if (invalid.length === 0) {
      alert('✅ Aucune vidéo invalide trouvée. Tout est propre !')
      return
    }

    const videoList = invalid.map((v: any) => {
      const reasons = Array.isArray(v.reasons) ? v.reasons.join(', ') : v.reasons
      return `  • ${v.title || '(sans titre)'} (ID: ${v.id})\n    Raisons: ${reasons}`
    }).join('\n')

    const confirmMessage = `⚠️ ATTENTION: Vous êtes sur le point de supprimer ${invalid.length} vidéo(s) invalide(s):\n\n${videoList}\n\nÊtes-vous sûr de vouloir continuer ?`
    
    if (!window.confirm(confirmMessage)) {
      return
    }

    try {
      setCleaning(true)
      setError(null)
      
      const response = await api.post('/admin/videos/clean-invalid')
      const deletedCount = response.data.deleted_count || 0
      
      alert(`✅ Nettoyage terminé: ${deletedCount} vidéo(s) supprimée(s)`)
      setInvalidVideos([])
      fetchVideos()
      
      window.dispatchEvent(new CustomEvent('videosCleaned', { detail: { deletedCount } }))
    } catch (err: any) {
      console.error('Erreur lors du nettoyage:', err)
      const errorMessage = err.response?.data?.error || err.message || 'Erreur inconnue'
      alert('❌ Erreur lors du nettoyage des vidéos invalides: ' + errorMessage)
      setError('Erreur lors du nettoyage: ' + errorMessage)
    } finally {
      setCleaning(false)
    }
  }

  if (loading && videos.length === 0) {
    return (
      <div className="admin-videos">
        <div className="loading">Chargement des vidéos...</div>
      </div>
    )
  }

  return (
    <div className="admin-videos">
      <div className="admin-videos-header">
        <h1>🎬 Gestion des Vidéos</h1>
        <div className="admin-videos-actions">
          {invalidVideos.length > 0 && (
            <>
              <button 
                onClick={async () => {
                  await fetchInvalidVideos()
                  setShowInvalidList(!showInvalidList)
                }}
                className="admin-action-button info"
                disabled={cleaning}
              >
                {showInvalidList ? '👁️ Masquer' : '👁️ Voir'} les vidéos invalides
              </button>
              <button 
                onClick={handleCleanInvalidVideos} 
                className="admin-action-button clean"
                disabled={cleaning}
              >
                {cleaning ? '🧹 Nettoyage...' : '🧹 Nettoyer les vidéos invalides'}
              </button>
            </>
          )}
          <Link href="/upload" className="admin-action-button primary">
            ⬆️ Upload Vidéo
          </Link>
        </div>
      </div>

      {showInvalidList && (
        <div className="invalid-videos-preview">
          <h3>📋 Vidéos invalides ({invalidVideos.length})</h3>
          {invalidVideos.length === 0 ? (
            <p className="no-invalid">✅ Aucune vidéo invalide trouvée</p>
          ) : (
            <div className="invalid-videos-list">
              {invalidVideos.map((video: any) => (
                <div key={video.id} className="invalid-video-item">
                  <div className="invalid-video-info">
                    <strong>{video.title || '(sans titre)'}</strong>
                    <span className="invalid-video-id">ID: {video.id}</span>
                    <span className={`invalid-video-status status-${video.status}`}>
                      {video.status}
                    </span>
                  </div>
                  <div className="invalid-video-reasons">
                    <strong>Raisons:</strong>
                    <ul>
                      {Array.isArray(video.reasons) ? (
                        video.reasons.map((reason: string, idx: number) => (
                          <li key={idx}>{reason}</li>
                        ))
                      ) : (
                        <li>{video.reasons}</li>
                      )}
                    </ul>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="admin-videos-filters">
        <div className="search-box">
          <input
            type="text"
            placeholder="Rechercher une vidéo..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="search-input"
          />
          <button onClick={fetchVideos} className="search-button">🔍</button>
        </div>
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value)}
          className="status-filter"
        >
          <option value="all">Tous les statuts</option>
          <option value="pending">En attente</option>
          <option value="processing">En traitement</option>
          <option value="ready">Prêtes</option>
          <option value="error">Erreur</option>
        </select>
      </div>

      {error && (
        <div className="error-message">{error}</div>
      )}

      <div className="videos-table-container">
        <table className="videos-table">
          <thead>
            <tr>
              <th>Miniature</th>
              <th>Titre</th>
              <th>Auteur</th>
              <th>Statut</th>
              <th>Vues</th>
              <th>Date</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {videos.length === 0 ? (
              <tr>
                <td colSpan={7} className="no-videos">
                  Aucune vidéo trouvée
                </td>
              </tr>
            ) : (
              videos.map((video: any) => (
                <tr key={video.id}>
                  <td>
                    {video.thumbnail_path ? (
                      <img
                        src={`/api/thumbnails/${video.thumbnail_path.split('/').pop()}`}
                        alt={video.title}
                        className="video-thumbnail-small"
                        onError={(e: any) => {
                          e.target.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iODAiIGhlaWdodD0iNDUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PHJlY3Qgd2lkdGg9IjgwIiBoZWlnaHQ9IjQ1IiBmaWxsPSIjMzMzIi8+PHRleHQgeD0iNTAwIiB5PSI1MDAiIGZvbnQtc2l6ZT0iMTQiIGZpbGw9IiM5OTkiIHRleHQtYW5jaG9yPSJtaWRkbGUiIGR5PSIuM2VtIj7wn5KlPC90ZXh0Pjwvc3ZnPg=='
                          e.target.onerror = null
                        }}
                      />
                    ) : (
                      <div className="video-thumbnail-placeholder">📹</div>
                    )}
                  </td>
                  <td>
                    <Link href={`/video/${video.id}`} className="video-title-link">
                      {video.title || '(sans titre)'}
                    </Link>
                  </td>
                  <td>{video.user?.username || 'N/A'}</td>
                  <td>
                    <span className={`status-badge status-${video.status || 'pending'}`}>
                      {video.status === 'processing' && '⏳ En traitement'}
                      {video.status === 'ready' && '✅ Prête'}
                      {video.status === 'pending' && '⏸️ En attente'}
                      {video.status === 'error' && '❌ Erreur'}
                      {!video.status && '❓ Inconnu'}
                    </span>
                  </td>
                  <td>{video.views || 0}</td>
                  <td>{new Date(video.created_at).toLocaleDateString('fr-FR')}</td>
                  <td>
                    <div className="action-buttons">
                      <Link
                        href={`/video/${video.id}`}
                        className="action-button view"
                        title="Voir"
                      >
                        👁️
                      </Link>
                      <button
                        onClick={() => handleDelete(video.id)}
                        className="action-button delete"
                        title="Supprimer"
                      >
                        🗑️
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="admin-videos-stats">
        <div className="stat-item">
          <span className="stat-label">Total:</span>
          <span className="stat-value">{videos.length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Prêtes:</span>
          <span className="stat-value">{videos.filter((v: any) => v.status === 'ready').length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">En traitement:</span>
          <span className="stat-value">{videos.filter((v: any) => v.status === 'processing').length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">En attente:</span>
          <span className="stat-value">{videos.filter((v: any) => v.status === 'pending').length}</span>
        </div>
      </div>
    </div>
  )
}

