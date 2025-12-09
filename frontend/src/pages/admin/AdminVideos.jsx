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
      console.log('AdminVideos: Récupération des vidéos avec params:', params)
      const response = await api.get('/videos', { params })
      console.log('AdminVideos: Réponse API:', response.data)
      
      // Ne pas filtrer les vidéos "null" pour les admins - ils doivent voir toutes les vidéos
      let allVideos = response.data.videos || response.data || []
      
      // Si c'est un tableau, utiliser directement, sinon extraire videos
      if (!Array.isArray(allVideos)) {
        allVideos = allVideos.videos || []
      }
      
      console.log('AdminVideos: Vidéos récupérées:', allVideos.length)
      
      // Filtrer par statut si nécessaire
      let filteredVideos = allVideos
      if (filterStatus !== 'all') {
        filteredVideos = allVideos.filter(v => v.status === filterStatus)
      }
      
      console.log('AdminVideos: Vidéos après filtrage:', filteredVideos.length)
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
    try {
      // Note: Vous devrez peut-être créer une route spécifique pour changer le statut
      await api.put(`/videos/${videoId}`, { status: newStatus })
      fetchVideos()
    } catch (err) {
      console.error('Erreur lors de la mise à jour:', err)
      alert('Erreur lors de la mise à jour du statut')
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
          <Link to="/upload" className="admin-action-button primary">
            ⬆️ Upload Vidéo
          </Link>
        </div>
      </div>

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
                    <select
                      value={video.status || 'pending'}
                      onChange={(e) => handleStatusChange(video.id, e.target.value)}
                      className={`status-select status-${video.status || 'pending'}`}
                    >
                      <option value="pending">En attente</option>
                      <option value="processing">En traitement</option>
                      <option value="ready">Prête</option>
                      <option value="error">Erreur</option>
                    </select>
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

