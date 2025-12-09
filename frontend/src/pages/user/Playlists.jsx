import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './Playlists.css'

const Playlists = () => {
  const { user, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [playlists, setPlaylists] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newPlaylistName, setNewPlaylistName] = useState('')
  const [newPlaylistDescription, setNewPlaylistDescription] = useState('')
  
  // Note: Le terme "playlist" est utilisé dans l'API, mais l'interface affiche "Liste"

  useEffect(() => {
    // Attendre que le chargement de l'authentification soit terminé
    if (authLoading) return
    
    if (user) {
      fetchPlaylists()
    } else {
      navigate('/login')
    }
  }, [user, authLoading, navigate])

  const fetchPlaylists = async () => {
    try {
      setLoading(true)
      const response = await api.get('/playlists')
      setPlaylists(response.data || [])
      setError(null)
    } catch (err) {
      console.error('Erreur lors de la récupération des playlists:', err)
      setError('Impossible de charger les listes')
    } finally {
      setLoading(false)
    }
  }

  const handleCreatePlaylist = async (e) => {
    e.preventDefault()
    try {
      const response = await api.post('/playlists', {
        name: newPlaylistName,
        description: newPlaylistDescription,
        is_public: false,
      })
      setPlaylists([...playlists, response.data])
      setNewPlaylistName('')
      setNewPlaylistDescription('')
      setShowCreateForm(false)
    } catch (err) {
      console.error('Erreur lors de la création de la playlist:', err)
      alert('Erreur lors de la création de la playlist')
    }
  }

  if (loading) {
    return (
      <div className="playlists-container">
        <LoadingSpinner message="Chargement des listes..." />
      </div>
    )
  }

  if (error) {
    return (
      <div className="playlists-container">
        <div className="error">{error}</div>
      </div>
    )
  }

  return (
    <div className="playlists-container">
      <div className="playlists-header">
        <h1>Mes Listes</h1>
        <button
          className="create-playlist-button"
          onClick={() => setShowCreateForm(!showCreateForm)}
        >
          {showCreateForm ? 'Annuler' : '+ Créer une liste'}
        </button>
      </div>

      {showCreateForm && (
        <form className="create-playlist-form" onSubmit={handleCreatePlaylist}>
          <input
            type="text"
            placeholder="Nom de la playlist"
            value={newPlaylistName}
            onChange={(e) => setNewPlaylistName(e.target.value)}
            required
            className="form-input"
          />
          <textarea
            placeholder="Description (optionnel)"
            value={newPlaylistDescription}
            onChange={(e) => setNewPlaylistDescription(e.target.value)}
            className="form-textarea"
          />
          <button type="submit" className="submit-button">
            Créer
          </button>
        </form>
      )}

      {playlists.length === 0 ? (
        <div className="no-playlists">
          <p>Vous n'avez pas encore de playlists</p>
          <p className="hint">Créez votre première playlist pour commencer</p>
        </div>
      ) : (
        <div className="playlists-grid">
          {playlists.map((playlist) => (
            <div
              key={playlist.id}
              className="playlist-card"
              onClick={() => navigate(`/playlist/${playlist.id}`)}
            >
              <div className="playlist-thumbnail">
                {playlist.videos && playlist.videos.length > 0 ? (
                  <img
                    src={`/thumbnails/${playlist.videos[0].thumbnail_path?.split('/').pop()}`}
                    alt={playlist.name}
                    onError={(e) => {
                      e.target.src = 'https://via.placeholder.com/320x180?text=Playlist'
                    }}
                  />
                ) : (
                  <div className="playlist-placeholder">
                    <span>📋</span>
                  </div>
                )}
                <div className="playlist-video-count">
                  {playlist.videos?.length || 0} vidéo(s)
                </div>
              </div>
              <div className="playlist-info">
                <h3 className="playlist-name">{playlist.name}</h3>
                {playlist.description && (
                  <p className="playlist-description">{playlist.description}</p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default Playlists

