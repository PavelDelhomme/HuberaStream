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
      // TODO: Implémenter l'API des favoris
      // const response = await api.get('/favorites')
      // setFavorites(response.data || [])
      setFavorites([]) // Temporaire
    } catch (error) {
      console.error('Erreur lors de la récupération des favoris:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div className="favorites-page">Chargement...</div>
  }

  return (
    <div className="favorites-page">
      <div className="favorites-header">
        <h1>⭐ Mes Favoris</h1>
        <p>Vos vidéos favorites</p>
      </div>

      {favorites.length === 0 ? (
        <div className="empty-state">
          <p>Vous n'avez pas encore de favoris</p>
          <Link to="/" className="btn-primary">Découvrir des vidéos</Link>
        </div>
      ) : (
        <div className="favorites-grid">
          {favorites.map((video) => (
            <Link key={video.id} to={`/video/${video.id}`} className="favorite-item">
              {video.thumbnail && (
                <img src={video.thumbnail} alt={video.title} className="favorite-thumbnail" />
              )}
              <div className="favorite-info">
                <h3>{video.title || 'Vidéo'}</h3>
                <p className="favorite-description">{video.description}</p>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

export default Favorites

