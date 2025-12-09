import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import VideoGrid from '../components/VideoGrid'
import api from '../services/api'
import { useAuth } from '../context/AuthContext'
import { filterValidVideos } from '../utils/videoUtils'
import LoadingSpinner from '../components/LoadingSpinner'
import './Home.css'

const Home = () => {
  const { user, isAdmin, loading: authLoading } = useAuth()
  const [continueWatching, setContinueWatching] = useState([])
  const [trendingVideos, setTrendingVideos] = useState([])
  const [recentVideos, setRecentVideos] = useState([])
  const [popularVideos, setPopularVideos] = useState([])
  const [userVideos, setUserVideos] = useState([]) // Vidéos uploadées par l'utilisateur
  const [categories, setCategories] = useState([])
  const [videosByCategory, setVideosByCategory] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    // Attendre que l'authentification soit chargée avant de faire les requêtes
    if (authLoading) {
      return
    }
    fetchAllData()
  }, [user, authLoading])

  const fetchAllData = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Récupérer les vidéos en cours de visionnage (si connecté)
      if (user) {
        try {
          const continueResponse = await api.get('/continue-watching')
          const historyItems = continueResponse.data.videos || []
          const validVideos = filterValidVideos(
            historyItems.map(item => {
              if (item.Video) {
                return { ...item.Video, watchPosition: item.Position }
              }
              if (item.video) {
                return { ...item.video, watchPosition: item.position }
              }
              return item
            })
          )
          setContinueWatching(validVideos)
        } catch (err) {
          console.debug('Aucune vidéo en cours de visionnage:', err.response?.status)
          setContinueWatching([])
        }
      } else {
        setContinueWatching([])
      }

      // Récupérer les vidéos uploadées par l'utilisateur (si connecté)
      if (user) {
        try {
          const userVideosResponse = await api.get('/videos', {
            params: { 
              page: 1, 
              limit: 12,
              user_id: user.id,
              status: 'all' // Voir toutes les vidéos de l'utilisateur
            }
          })
          const userVids = filterValidVideos(userVideosResponse.data.videos || [])
          setUserVideos(userVids)
        } catch (err) {
          console.debug('Aucune vidéo uploadée par l\'utilisateur:', err.response?.status)
          setUserVideos([])
        }
      } else {
        setUserVideos([])
      }

      // Récupérer les vidéos récentes (publiques seulement)
      try {
        const recentResponse = await api.get('/videos', {
          params: { page: 1, limit: 12, is_public: true }
        })
        setRecentVideos(filterValidVideos(recentResponse.data.videos || []))
      } catch (err) {
        console.error('Erreur lors de la récupération des vidéos récentes:', err)
        setRecentVideos([])
      }

      // Récupérer les vidéos populaires
      try {
        const popularResponse = await api.get('/videos', {
          params: { page: 1, limit: 12, sort: 'views', is_public: true }
        })
        const validPopularVideos = filterValidVideos(popularResponse.data.videos || [])
        setPopularVideos(validPopularVideos)
        setTrendingVideos(validPopularVideos.slice(0, 8))
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

        // Récupérer les vidéos par catégorie (pour tous les utilisateurs)
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
            categoryVideos[category.id] = filterValidVideos(response.data.videos || [])
          } catch (err) {
            categoryVideos[category.id] = []
          }
        }
        setVideosByCategory(categoryVideos)
      } catch (err) {
        // Gérer les erreurs silencieusement - les catégories sont maintenant publiques
        console.debug('Erreur lors de la récupération des catégories:', err)
        setCategories([])
        setVideosByCategory({})
      }
    } catch (err) {
      // Gestion des erreurs générales
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
        <LoadingSpinner message="Chargement..." />
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

  // Vérifier s'il y a des vidéos à afficher
  const hasAnyVideos = 
    trendingVideos.length > 0 || 
    popularVideos.length > 0 || 
    recentVideos.length > 0 || 
    continueWatching.length > 0 ||
    userVideos.length > 0 ||
    Object.values(videosByCategory).some(videos => videos && videos.length > 0)

  return (
    <div className="home-container">
      {/* Hero Section - Toujours affichée */}
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

      {/* Section: Mes Vidéos (pour l'utilisateur connecté) */}
      {user && userVideos.length > 0 && (
        <div className="section-container">
          <div className="section-header">
            <h2 className="section-title">📤 Mes Vidéos</h2>
            <Link to="/admin/videos" className="section-link">Gérer mes vidéos</Link>
          </div>
          <VideoGrid videos={userVideos} title="" />
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

      {/* Section: Tendances - Toujours affichée style Netflix */}
      <div className="section-container">
        <div className="section-header">
          <h2 className="section-title">🔥 Tendances</h2>
        </div>
        {trendingVideos.length > 0 ? (
          <VideoGrid videos={trendingVideos} title="" />
        ) : (
          <div className="section-empty">
            <p>Aucune vidéo tendance pour le moment</p>
          </div>
        )}
      </div>

      {/* Section: Populaires - Toujours affichée style Netflix */}
      <div className="section-container">
        <div className="section-header">
          <h2 className="section-title">⭐ Populaires</h2>
        </div>
        {popularVideos.length > 0 ? (
          <VideoGrid videos={popularVideos} title="" />
        ) : (
          <div className="section-empty">
            <p>Aucune vidéo populaire pour le moment</p>
          </div>
        )}
      </div>

      {/* Section: Vidéos récentes - Toujours affichée style Netflix */}
      <div className="section-container">
        <div className="section-header">
          <h2 className="section-title">📺 Nouveautés</h2>
        </div>
        {recentVideos.length > 0 ? (
          <VideoGrid videos={recentVideos} title="" />
        ) : (
          <div className="section-empty">
            <p>Aucune nouvelle vidéo pour le moment</p>
          </div>
        )}
      </div>

      {/* Sections par catégorie - Toujours affichées style Netflix */}
      {categories.map((category) => {
        const videos = videosByCategory[category.id] || []
        
        return (
          <div key={category.id} className="section-container">
            <div className="section-header">
              <h2 className="section-title">📂 {category.name}</h2>
              {videos.length > 0 && (
                <Link to={`/search?category=${category.id}`} className="section-link">Voir tout</Link>
              )}
            </div>
            {videos.length > 0 ? (
              <VideoGrid videos={videos} title="" />
            ) : (
              <div className="section-empty">
                <p>Aucune vidéo dans cette catégorie pour le moment</p>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

export default Home

