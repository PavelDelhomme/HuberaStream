import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import VideoGrid from '../components/VideoGrid'
import api from '../services/api'
import { useAuth } from '../context/AuthContext'
import './Home.css'

const Home = () => {
  const { user } = useAuth()
  const [continueWatching, setContinueWatching] = useState([])
  const [trendingVideos, setTrendingVideos] = useState([])
  const [recentVideos, setRecentVideos] = useState([])
  const [popularVideos, setPopularVideos] = useState([])
  const [categories, setCategories] = useState([])
  const [videosByCategory, setVideosByCategory] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchAllData()
  }, [user])

  const fetchAllData = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Récupérer les vidéos en cours de visionnage (si connecté)
      if (user) {
        try {
          const continueResponse = await api.get('/continue-watching')
          const historyItems = continueResponse.data.videos || []
          setContinueWatching(historyItems.map(item => {
            if (item.Video) {
              return { ...item.Video, watchPosition: item.Position }
            }
            if (item.video) {
              return { ...item.video, watchPosition: item.position }
            }
            return item
          }))
        } catch (err) {
          console.debug('Aucune vidéo en cours de visionnage:', err.response?.status)
          setContinueWatching([])
        }
      } else {
        setContinueWatching([])
      }

      // Récupérer les vidéos récentes (publiques seulement)
      try {
        const recentResponse = await api.get('/videos', {
          params: { page: 1, limit: 12, is_public: true }
        })
        setRecentVideos(recentResponse.data.videos || [])
      } catch (err) {
        console.error('Erreur lors de la récupération des vidéos récentes:', err)
        setRecentVideos([])
      }

      // Récupérer les vidéos populaires
      try {
        const popularResponse = await api.get('/videos', {
          params: { page: 1, limit: 12, sort: 'views', is_public: true }
        })
        setPopularVideos(popularResponse.data.videos || [])
        setTrendingVideos(popularResponse.data.videos?.slice(0, 8) || [])
      } catch (err) {
        console.error('Erreur lors de la récupération des vidéos populaires:', err)
        setPopularVideos([])
        setTrendingVideos([])
      }

      // Récupérer les catégories (essayer même si pas connecté, mais gérer l'erreur)
      try {
        const categoriesResponse = await api.get('/categories')
        const cats = categoriesResponse.data || []
        setCategories(cats)

        // Récupérer les vidéos par catégorie (seulement si connecté)
        if (user) {
          const categoryVideos = {}
          for (const category of cats.slice(0, 6)) {
            try {
              const response = await api.get('/videos', {
                params: { 
                  page: 1, 
                  limit: 12,
                  category_id: category.id,
                  is_public: true
                }
              })
              categoryVideos[category.id] = response.data.videos || []
            } catch (err) {
              categoryVideos[category.id] = []
            }
          }
          setVideosByCategory(categoryVideos)
        } else {
          setVideosByCategory({})
        }
      } catch (err) {
        // Si erreur 401, c'est normal si pas connecté - on continue sans catégories
        if (err.response?.status !== 401) {
          console.error('Erreur lors de la récupération des catégories:', err)
        }
        setCategories([])
        setVideosByCategory({})
      }
      
    } catch (err) {
      console.error('Erreur générale lors de la récupération des données:', err)
      // Ne pas afficher d'erreur si c'est juste qu'il n'y a pas de vidéos
      if (err.response?.status !== 404 && err.response?.status !== 401) {
        setError('Impossible de charger les vidéos. Veuillez réessayer plus tard.')
      }
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="home-container">
        <div className="loading-spinner">
          <div className="spinner"></div>
          <p>Chargement...</p>
        </div>
      </div>
    )
  }

  if (error && recentVideos.length === 0 && trendingVideos.length === 0) {
    return (
      <div className="home-container">
        <div className="hero-section">
          <div className="hero-content">
            <h1 className="hero-title">Bienvenue sur StreamMake</h1>
            <p className="hero-subtitle">Votre plateforme de streaming personnelle</p>
            {!user && (
              <div className="hero-actions">
                <Link to="/register" className="btn-hero-primary">Commencer</Link>
                <Link to="/login" className="btn-hero-secondary">Se connecter</Link>
              </div>
            )}
          </div>
        </div>
        <div className="error-message">
          <p>⚠️ {error}</p>
          <button onClick={fetchAllData} className="retry-button">Réessayer</button>
        </div>
      </div>
    )
  }

  return (
    <div className="home-container">
      {/* Hero Section avec vidéo en vedette */}
      {trendingVideos.length > 0 && (
        <div className="hero-section">
          <div className="hero-content">
            <h1 className="hero-title">Bienvenue sur StreamMake</h1>
            <p className="hero-subtitle">Votre plateforme de streaming personnelle</p>
            {!user && (
              <div className="hero-actions">
                <Link to="/register" className="btn-hero-primary">Commencer</Link>
                <Link to="/login" className="btn-hero-secondary">Se connecter</Link>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Section: Continuer à regarder */}
      {user && continueWatching.length > 0 && (
        <div className="section-container">
          <div className="section-header">
            <h2 className="section-title">▶️ Continuer à regarder</h2>
            <Link to="/history" className="section-link">Voir tout</Link>
          </div>
          <VideoGrid videos={continueWatching} title="" />
        </div>
      )}

      {/* Section: Tendances */}
      {trendingVideos.length > 0 && (
        <div className="section-container">
          <div className="section-header">
            <h2 className="section-title">🔥 Tendances</h2>
          </div>
          <VideoGrid videos={trendingVideos} title="" />
        </div>
      )}

      {/* Section: Populaires */}
      {popularVideos.length > 0 && (
        <div className="section-container">
          <div className="section-header">
            <h2 className="section-title">⭐ Populaires</h2>
          </div>
          <VideoGrid videos={popularVideos} title="" />
        </div>
      )}

      {/* Section: Vidéos récentes */}
      {recentVideos.length > 0 && (
        <div className="section-container">
          <div className="section-header">
            <h2 className="section-title">📺 Nouveautés</h2>
          </div>
          <VideoGrid videos={recentVideos} title="" />
        </div>
      )}

      {/* Sections par catégorie */}
      {categories.map((category) => {
        const videos = videosByCategory[category.id] || []
        if (videos.length === 0) return null
        
        return (
          <div key={category.id} className="section-container">
            <div className="section-header">
              <h2 className="section-title">📂 {category.name}</h2>
              <Link to={`/search?category=${category.id}`} className="section-link">Voir tout</Link>
            </div>
            <VideoGrid videos={videos} title="" />
          </div>
        )
      })}

      {/* Message si aucune vidéo */}
      {recentVideos.length === 0 && categories.length === 0 && (
        <div className="empty-state">
          <p>🎬 Aucune vidéo disponible pour le moment</p>
          {user && user.role === 'admin' && (
            <Link to="/upload" className="btn-primary">Uploader une vidéo</Link>
          )}
        </div>
      )}
    </div>
  )
}

export default Home

