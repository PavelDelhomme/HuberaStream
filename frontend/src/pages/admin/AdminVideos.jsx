import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import { filterValidVideos } from '../../utils/videoUtils'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminVideos.css'

const AdminVideos = () => {
  const { user, isAdmin, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [videos, setVideos] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [filterStatus, setFilterStatus] = useState('all')
  const [cleaning, setCleaning] = useState(false)
  const [invalidVideos, setInvalidVideos] = useState([])
  const [showInvalidList, setShowInvalidList] = useState(false)

  useEffect(() => {
    if (authLoading) return
    
    if (!user) {
      navigate('/login')
      return
    }
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchVideos()
  }, [user, authLoading, isAdmin, navigate])

  const fetchVideos = async () => {
    try {
      setLoading(true)
      setError(null)
      const params = {
        page: 1,
        limit: 1000, // Augmenter la limite pour voir toutes les vidéos
        status: 'all', // Demander toutes les vidéos, pas seulement celles prêtes
      }
      if (searchQuery) {
        params.search = searchQuery
      }
      const response = await api.get('/videos', { params })
      
      // Extraire les vidéos de la réponse
      let allVideos = []
      if (Array.isArray(response.data)) {
        allVideos = response.data
      } else if (response.data && Array.isArray(response.data.videos)) {
        allVideos = response.data.videos
      } else if (response.data && response.data.data && Array.isArray(response.data.data.videos)) {
        allVideos = response.data.data.videos
      }
      
      // Pour les admins, voir TOUTES les vidéos (même celles en cours de traitement ou avec titre vide)
      // Filtrer seulement les vidéos vraiment invalides (sans ID)
      let validVideos = allVideos.filter(v => {
        if (!v || !v.id) {
          return false
        }
        // Pour les admins, accepter toutes les vidéos même avec titre "null" ou vide
        // Ils doivent pouvoir voir et gérer toutes les vidéos
        return true
      })
      
      // Filtrer par statut si nécessaire
      let filteredVideos = validVideos
      if (filterStatus !== 'all') {
        filteredVideos = validVideos.filter(v => v.status === filterStatus)
      }
      
      setVideos(filteredVideos)
    } catch (err) {
      console.error('AdminVideos: Erreur lors de la récupération des vidéos:', err)
      setError('Erreur lors de la récupération des vidéos: ' + (err.response?.data?.error || err.message))
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (videoId) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer cette vidéo ?')) {
      return
    }

    try {
      await api.delete(`/videos/${videoId}`)
      fetchVideos()
    } catch (err) {
      console.error('Erreur lors de la suppression:', err)
      alert('Erreur lors de la suppression de la vidéo')
    }
  }

  const handleStatusChange = async (videoId, newStatus) => {
    // DÉSACTIVÉ: Le statut est géré automatiquement par le système
    // Les vidéos passent automatiquement de "processing" à "ready" quand le transcodage est terminé
    alert('Le statut des vidéos est géré automatiquement par le système.\n\n' +
          '• "processing" = Vidéo en cours de transcodage\n' +
          '• "ready" = Vidéo prête (transcodage terminé avec succès)\n' +
          '• "error" = Erreur lors du traitement\n\n' +
          'Vous ne pouvez pas modifier manuellement le statut.')
  }

  const fetchInvalidVideos = async () => {
    try {
      console.log('🔍 AdminVideos: Récupération de la liste des vidéos invalides...')
      const response = await api.get('/admin/videos/invalid')
      const invalid = response.data.invalid_videos || []
      console.log(`📋 AdminVideos: ${invalid.length} vidéo(s) invalide(s) trouvée(s):`, invalid)
      setInvalidVideos(invalid)
      return invalid
    } catch (err) {
      console.error('❌ AdminVideos: Erreur lors de la récupération des vidéos invalides:', err)
      setInvalidVideos([])
      return []
    }
  }

  const handleCleanInvalidVideos = async () => {
    // D'abord, récupérer la liste des vidéos invalides
    console.log('🧹 AdminVideos: Début du processus de nettoyage...')
    const invalid = await fetchInvalidVideos()
    
    if (invalid.length === 0) {
      console.log('✅ AdminVideos: Aucune vidéo invalide à nettoyer')
      alert('✅ Aucune vidéo invalide trouvée. Tout est propre !')
      return
    }

    // Afficher la liste des vidéos à supprimer
    const videoList = invalid.map(v => {
      const reasons = Array.isArray(v.reasons) ? v.reasons.join(', ') : v.reasons
      return `  • ${v.title || '(sans titre)'} (ID: ${v.id})\n    Raisons: ${reasons}`
    }).join('\n')

    const confirmMessage = `⚠️ ATTENTION: Vous êtes sur le point de supprimer ${invalid.length} vidéo(s) invalide(s):\n\n${videoList}\n\nÊtes-vous sûr de vouloir continuer ?`
    
    if (!window.confirm(confirmMessage)) {
      console.log('❌ AdminVideos: Nettoyage annulé par l\'utilisateur')
      return
    }

    try {
      console.log('🚀 AdminVideos: Démarrage du nettoyage des vidéos invalides...')
      setCleaning(true)
      setError(null)
      
      const response = await api.post('/admin/videos/clean-invalid')
      
      const deletedCount = response.data.deleted_count || 0
      const deletedVideos = response.data.deleted_videos || []
      const errors = response.data.errors || []
      
      console.log(`✅ AdminVideos: Nettoyage terminé - ${deletedCount} vidéo(s) supprimée(s)`)
      console.log('📋 AdminVideos: Liste des vidéos supprimées:', deletedVideos)
      
      if (errors.length > 0) {
        console.error('⚠️ AdminVideos: Erreurs lors du nettoyage:', errors)
      }
      
      // Afficher un message détaillé
      let message = `✅ Nettoyage terminé: ${deletedCount} vidéo(s) supprimée(s)\n\n`
      
      if (deletedVideos.length > 0) {
        message += 'Vidéos supprimées:\n'
        deletedVideos.forEach(v => {
          const reasons = Array.isArray(v.reasons) ? v.reasons.join(', ') : v.reasons
          message += `  • ${v.title || '(sans titre)'} (${v.id})\n    Raisons: ${reasons}\n`
        })
      }
      
      if (errors.length > 0) {
        message += `\n⚠️ Erreurs: ${errors.length} erreur(s) lors de la suppression`
        console.error('Erreurs détaillées:', errors)
      }
      
      alert(message)
      setInvalidVideos([])
      
      // Recharger la liste des vidéos
      fetchVideos()
      
      // Émettre un événement pour notifier le dashboard de rafraîchir les stats
      window.dispatchEvent(new CustomEvent('videosCleaned', { detail: { deletedCount } }))
    } catch (err) {
      console.error('❌ AdminVideos: Erreur lors du nettoyage:', err)
      const errorMessage = err.response?.data?.error || err.message || 'Erreur inconnue'
      console.error('Détails de l\'erreur:', err.response?.data)
      alert('❌ Erreur lors du nettoyage des vidéos invalides: ' + errorMessage)
      setError('Erreur lors du nettoyage: ' + errorMessage)
    } finally {
      setCleaning(false)
    }
  }

  useEffect(() => {
    fetchVideos()
  }, [searchQuery, filterStatus])

  if (loading) {
    return (
      <div className="admin-videos">
        <LoadingSpinner message="Chargement des vidéos..." />
      </div>
    )
  }

  return (
    <div className="admin-videos">
      <div className="admin-videos-header">
        <h1>🎬 Gestion des Vidéos</h1>
        <div className="admin-videos-actions">
          <button 
            onClick={async () => {
              const invalid = await fetchInvalidVideos()
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
          <Link to="/upload" className="admin-action-button primary">
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
              {invalidVideos.map(video => (
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
                        video.reasons.map((reason, idx) => (
                          <li key={idx}>{reason}</li>
                        ))
                      ) : (
                        <li>{video.reasons}</li>
                      )}
                    </ul>
                  </div>
                  {video.format_count !== undefined && (
                    <div className="invalid-video-meta">
                      Formats: {video.format_count} | 
                      Créée: {new Date(video.created_at).toLocaleDateString('fr-FR')} |
                      Par: {video.user || 'N/A'}
                    </div>
                  )}
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
                <td colSpan="7" className="no-videos">
                  Aucune vidéo trouvée
                </td>
              </tr>
            ) : (
              videos.map((video) => (
                <tr key={video.id}>
                  <td>
                    {video.thumbnail_path ? (
                      <img
                        src={`/thumbnails/${video.thumbnail_path.split('/').pop()}`}
                        alt={video.title}
                        className="video-thumbnail-small"
                        onError={(e) => {
                          e.target.src = 'https://via.placeholder.com/80x45?text=No+Thumb'
                        }}
                      />
                    ) : (
                      <div className="video-thumbnail-placeholder">📹</div>
                    )}
                  </td>
                  <td>
                    <Link to={`/video/${video.id}`} className="video-title-link">
                      {video.title}
                    </Link>
                  </td>
                  <td>{video.user?.username || 'N/A'}</td>
                  <td>
                    <span
                      className={`status-badge status-${video.status || 'pending'}`}
                      title="Statut automatique (non modifiable manuellement)"
                    >
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
                        to={`/video/${video.id}`}
                        className="action-button view"
                        title="Voir"
                      >
                        👁️
                      </Link>
                      <Link
                        to={`/video/${video.id}/edit`}
                        className="action-button edit"
                        title="Modifier"
                      >
                        ✏️
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
          <span className="stat-value">{videos.filter(v => v.status === 'ready').length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">En traitement:</span>
          <span className="stat-value">{videos.filter(v => v.status === 'processing').length}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">En attente:</span>
          <span className="stat-value">{videos.filter(v => v.status === 'pending').length}</span>
        </div>
      </div>
    </div>
  )
}

export default AdminVideos

