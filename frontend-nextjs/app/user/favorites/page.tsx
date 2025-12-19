'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { authService } from '@/lib/auth'
import api from '@/lib/api'
import './Favorites.css'

export default function FavoritesPage() {
  const router = useRouter()
  const [favorites, setFavorites] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const loadFavorites = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login')
        return
      }
      fetchFavorites()
    }
    loadFavorites()
  }, [router])

  const fetchFavorites = async () => {
    try {
      try {
        const response = await api.get('/favorites')
        setFavorites(response.data || [])
      } catch (favoritesError: any) {
        try {
          const playlistsResponse = await api.get('/playlists')
          const favoritePlaylist = playlistsResponse.data?.find((p: any) => p.name?.toLowerCase().includes('favori'))
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
        <div className="loading">Chargement des favoris...</div>
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
          <Link href="/" className="btn-primary">Découvrir des vidéos</Link>
        </div>
      ) : (
        <div className="favorites-grid">
          {favorites.map((video) => {
            const videoData = video.Video || video.video || video
            return (
              <Link 
                key={videoData.id || video.id} 
                href={`/video/${videoData.id || video.id}`} 
                className="favorite-item"
              >
                {videoData.thumbnail_path || video.thumbnail ? (
                  <img 
                    src={`/api/thumbnails/${(videoData.thumbnail_path || video.thumbnail).split('/').pop()}`} 
                    alt={videoData.title || video.title}
                    className="favorite-thumbnail"
                  />
                ) : (
                  <div className="favorite-thumbnail-placeholder">⭐</div>
                )}
                <div className="favorite-info">
                  <h3>{videoData.title || video.title}</h3>
                  {videoData.description && (
                    <p className="favorite-description">
                      {videoData.description.substring(0, 100)}...
                    </p>
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

