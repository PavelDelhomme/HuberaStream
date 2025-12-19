'use client'

import { useEffect, useState, useRef } from 'react'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import Link from 'next/link'
import VideoGrid from '@/components/VideoGrid'
import './Home.css'

interface Video {
  id: string
  title: string
  description?: string
  thumbnail_path?: string
  thumbnail_url?: string
  duration: number
  views: number
  created_at: string
  status?: string
  user?: {
    username: string
  }
  watchPosition?: number
}

export default function HomeContent() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [continueWatching, setContinueWatching] = useState<Video[]>([])
  const [trendingVideos, setTrendingVideos] = useState<Video[]>([])
  const [recentVideos, setRecentVideos] = useState<Video[]>([])
  const [popularVideos, setPopularVideos] = useState<Video[]>([])
  const [userVideos, setUserVideos] = useState<Video[]>([])
  const [categories, setCategories] = useState<any[]>([])
  const [videosByCategory, setVideosByCategory] = useState<Record<string, Video[]>>({})
  const [error, setError] = useState<string | null>(null)
  const hasLoadedRef = useRef(false)
  const userIdRef = useRef<string | null>(null)

  // Charger l'utilisateur une seule fois au montage
  useEffect(() => {
    const loadUser = async () => {
      if (authService.isAuthenticated()) {
        const currentUser = await authService.getCurrentUser()
        setUser(currentUser)
        userIdRef.current = currentUser?.id || null
      } else {
        setUser(null)
        userIdRef.current = null
      }
    }
    loadUser()
  }, []) // Pas de dépendances - exécuté une seule fois

  // Charger les données une seule fois après le chargement de l'utilisateur
  useEffect(() => {
    if (hasLoadedRef.current) return
    
    const loadData = async () => {
      hasLoadedRef.current = true
      await fetchAllData()
    }
    
    // Attendre un peu pour que l'utilisateur soit chargé
    const timer = setTimeout(() => {
      loadData()
    }, 100)
    
    return () => clearTimeout(timer)
  }, []) // Pas de dépendances - exécuté une seule fois

  const filterValidVideos = (videos: Video[]): Video[] => {
    if (!videos || !Array.isArray(videos)) return []
    return videos.filter(video => {
      if (!video || !video.id) return false
      if (video.title === 'null' || video.title === 'Null' || video.title === 'NULL') return false
      if (video.title && typeof video.title === 'string' && video.title.trim() === '' && video.status !== 'processing') return false
      return true
    })
  }

  const fetchAllData = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Utiliser userIdRef pour éviter les dépendances sur l'objet user
      const currentUserId = userIdRef.current || user?.id
      
      // Récupérer les vidéos en cours de visionnage (si connecté)
      if (currentUserId) {
        try {
          const continueResponse = await api.get('/continue-watching')
          const historyItems = continueResponse.data.videos || []
          const validVideos = filterValidVideos(
            historyItems.map((item: any) => {
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
          console.debug('Aucune vidéo en cours de visionnage')
          setContinueWatching([])
        }

        // Récupérer les vidéos uploadées par l'utilisateur
        try {
          const userVideosResponse = await api.get('/videos', {
            params: { 
              page: 1, 
              limit: 12,
              user_id: currentUserId,
              status: 'all'
            }
          })
          setUserVideos(filterValidVideos(userVideosResponse.data.videos || []))
        } catch (err) {
          console.debug('Aucune vidéo uploadée par l\'utilisateur')
          setUserVideos([])
        }
      } else {
        setContinueWatching([])
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

      // Récupérer les catégories
      try {
        const categoriesResponse = await api.get('/categories')
        const cats = categoriesResponse.data || []
        setCategories(cats)

        // Récupérer les vidéos par catégorie
        const categoryVideos: Record<string, Video[]> = {}
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
        console.debug('Erreur lors de la récupération des catégories:', err)
        setCategories([])
        setVideosByCategory({})
      }
    } catch (err: any) {
      console.error('Erreur générale lors de la récupération des données:', err)
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
                <Link href="/register" className="btn-hero-primary" prefetch={false}>Commencer</Link>
                <Link href="/login" className="btn-hero-secondary" prefetch={false}>Se connecter</Link>
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

  const hasAnyVideos = 
    trendingVideos.length > 0 || 
    popularVideos.length > 0 || 
    recentVideos.length > 0 || 
    continueWatching.length > 0 ||
    userVideos.length > 0 ||
    Object.values(videosByCategory).some(videos => videos && videos.length > 0)

  return (
    <div className="home-container">
      {/* Hero Section */}
      <div className="hero-section">
        <div className="hero-content">
          <h1 className="hero-title">Bienvenue sur StreamMake</h1>
          <p className="hero-subtitle">Votre plateforme de streaming personnelle</p>
          {!user && (
            <div className="hero-actions">
              <Link href="/register" className="btn-hero-primary" prefetch={false}>Commencer</Link>
              <Link href="/login" className="btn-hero-secondary" prefetch={false}>Se connecter</Link>
            </div>
          )}
        </div>
      </div>

      {/* Section: Mes Vidéos (pour l'utilisateur connecté) */}
      {user && userVideos.length > 0 && (
        <div className="section-container">
          <div className="section-header">
            <h2 className="section-title">📤 Mes Vidéos</h2>
            {authService.isAdmin(user) && (
              <Link href="/admin/videos" className="section-link" prefetch={false}>Gérer mes vidéos</Link>
            )}
          </div>
          <VideoGrid videos={userVideos} />
        </div>
      )}

      {/* Section: Continuer à regarder */}
      {user && continueWatching.length > 0 && (
        <div className="section-container">
          <div className="section-header">
            <h2 className="section-title">▶️ Continuer à regarder</h2>
            <Link href="/user/history" className="section-link" prefetch={false}>Voir tout</Link>
          </div>
          <VideoGrid videos={continueWatching} />
        </div>
      )}

      {/* Section: Tendances */}
      <div className="section-container">
        <div className="section-header">
          <h2 className="section-title">🔥 Tendances</h2>
        </div>
        {trendingVideos.length > 0 ? (
          <VideoGrid videos={trendingVideos} />
        ) : (
          <div className="section-empty">
            <p>Aucune vidéo tendance pour le moment</p>
          </div>
        )}
      </div>

      {/* Section: Populaires */}
      <div className="section-container">
        <div className="section-header">
          <h2 className="section-title">⭐ Populaires</h2>
        </div>
        {popularVideos.length > 0 ? (
          <VideoGrid videos={popularVideos} />
        ) : (
          <div className="section-empty">
            <p>Aucune vidéo populaire pour le moment</p>
          </div>
        )}
      </div>

      {/* Section: Vidéos récentes */}
      <div className="section-container">
        <div className="section-header">
          <h2 className="section-title">📺 Nouveautés</h2>
        </div>
        {recentVideos.length > 0 ? (
          <VideoGrid videos={recentVideos} />
        ) : (
          <div className="section-empty">
            <p>Aucune nouvelle vidéo pour le moment</p>
          </div>
        )}
      </div>

      {/* Sections par catégorie */}
      {categories.map((category) => {
        const videos = videosByCategory[category.id] || []
        
        return (
          <div key={category.id} className="section-container">
            <div className="section-header">
              <h2 className="section-title">📂 {category.name}</h2>
              {videos.length > 0 && (
                <Link href={`/search?category=${category.id}`} className="section-link" prefetch={false}>Voir tout</Link>
              )}
            </div>
            {videos.length > 0 ? (
              <VideoGrid videos={videos} />
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

