import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './Favorites.css'

const Favorites = () => {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [favorites, setFavorites] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    fetchFavorites()
  }, [user, navigate])

  const fetchFavorites = async () => {
    try {
      // Essayer l'endpoint /favorites, sinon utiliser les playlists ou une autre source
      try {
        const response = await api.get('/favorites')
        setFavorites(response.data || [])
      } catch (favoritesError) {
        // Si l'endpoint n'existe pas, essayer de récupérer depuis les playlists favorites
        try {
          const playlistsResponse = await api.get('/playlists')
          const favoritePlaylist = playlistsResponse.data?.find(p => p.name?.toLowerCase().includes('favori'))
          if (favoritePlaylist?.videos) {
            setFavorites(favoritePlaylist.videos)
          } else {
            setFavorites([])
          }
        } catch (playlistError) {
          console.log('Aucun système de favoris disponible pour le moment')
          setFavorites([])
        }
      }
    } catch (error) {
      console.error('Erreur lors de la récupération des favoris:', error)
      setFavorites([])
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="favorites-page">
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Chargement des favoris...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="favorites-page">
      <div className="favorites-header">
        <h1>⭐ Mes Favoris</h1>
        <p>Vos vidéos favorites</p>
      </div>

      {favorites.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">⭐</div>
          <p>Vous n'avez pas encore de favoris</p>
          <p className="empty-hint">Ajoutez des vidéos à vos favoris pour les retrouver facilement</p>
          <Link to="/" className="btn-primary">Découvrir des vidéos</Link>
        </div>
      ) : (
        <div className="favorites-grid">
          {favorites.map((video) => {
            const videoData = video.Video || video.video || video
            return (
              <Link key={videoData.id || video.id} to={`/video/${videoData.id || video.id}`} className="favorite-item">
                {videoData.thumbnail_path || video.thumbnail ? (
                  <img 
                    src={videoData.thumbnail_path ? `/thumbnails/${videoData.thumbnail_path.split('/').pop()}` : video.thumbnail} 
                    alt={videoData.title || video.title || 'Vidéo'} 
                    className="favorite-thumbnail"
                    onError={(e) => {
                      e.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'
                    }}
                  />
                ) : (
                  <div className="favorite-thumbnail-placeholder">
                    <span>📹</span>
                  </div>
                )}
                <div className="favorite-info">
                  <h3>{videoData.title || video.title || 'Vidéo'}</h3>
                  <p className="favorite-description">{videoData.description || video.description || ''}</p>
                  {videoData.views !== undefined && (
                    <p className="favorite-views">{videoData.views} vues</p>
                  )}
                </div>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}

export default Favorites

