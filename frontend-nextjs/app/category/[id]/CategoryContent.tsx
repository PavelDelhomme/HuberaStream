'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import api from '@/lib/api'
import Link from 'next/link'
import '../Category.css'

interface Video {
  id: string
  title: string
  description: string
  thumbnail_url: string
  duration: number
  views: number
  created_at: string
}

interface Category {
  id: string
  name: string
  description: string
  icon: string
}

export default function CategoryContent() {
  const params = useParams()
  const categoryId = params.id as string
  const [videos, setVideos] = useState<Video[]>([])
  const [category, setCategory] = useState<Category | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [hasMore, setHasMore] = useState(true)

  useEffect(() => {
    if (categoryId) {
      fetchCategory()
      fetchVideos()
    }
  }, [categoryId, page])

  const fetchCategory = async () => {
    try {
      const response = await api.get('/categories')
      const categories = response.data || []
      const foundCategory = categories.find((cat: Category) => cat.id === categoryId)
      if (foundCategory) {
        setCategory(foundCategory)
      } else {
        setError('Catégorie non trouvée')
      }
    } catch (err: any) {
      console.error('Erreur lors de la récupération de la catégorie:', err)
      setError('Erreur lors du chargement de la catégorie')
    }
  }

  const fetchVideos = async () => {
    try {
      setLoading(true)
      const response = await api.get('/videos', {
        params: {
          page,
          limit: 24,
          category_id: categoryId,
          is_public: true,
        },
      })

      const newVideos = response.data.videos || []
      if (page === 1) {
        setVideos(newVideos)
      } else {
        setVideos((prev) => [...prev, ...newVideos])
      }

      setHasMore(newVideos.length === 24)
    } catch (err: any) {
      console.error('Erreur lors de la récupération des vidéos:', err)
      setError('Erreur lors du chargement des vidéos')
    } finally {
      setLoading(false)
    }
  }

  const formatDuration = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`
  }

  if (error && !category) {
    return (
      <div className="category-page">
        <div className="error-message">
          <p>{error}</p>
          <Link href="/">Retour à l'accueil</Link>
        </div>
      </div>
    )
  }

  return (
    <div className="category-page">
      {category && (
        <div className="category-header">
          <h1>
            {category.icon} {category.name}
          </h1>
          {category.description && <p className="category-description">{category.description}</p>}
        </div>
      )}

      {loading && page === 1 ? (
        <div className="loading">Chargement des vidéos...</div>
      ) : videos.length > 0 ? (
        <>
          <div className="videos-grid">
            {videos.map((video) => (
              <Link key={video.id} href={`/video/${video.id}`} className="video-card">
                <div className="video-thumbnail">
                  {video.thumbnail_url ? (
                    <img src={video.thumbnail_url} alt={video.title} />
                  ) : (
                    <div className="video-placeholder">🎬</div>
                  )}
                  <div className="video-duration">{formatDuration(video.duration)}</div>
                </div>
                <div className="video-info">
                  <h3 className="video-title">{video.title}</h3>
                  <div className="video-meta">
                    <span>👁️ {video.views}</span>
                    <span>📅 {new Date(video.created_at).toLocaleDateString('fr-FR')}</span>
                  </div>
                  {video.description && (
                    <p className="video-description">{video.description.substring(0, 100)}...</p>
                  )}
                </div>
              </Link>
            ))}
          </div>
          {hasMore && (
            <div className="load-more-container">
              <button
                onClick={() => setPage((prev) => prev + 1)}
                disabled={loading}
                className="load-more-button"
              >
                {loading ? 'Chargement...' : 'Charger plus'}
              </button>
            </div>
          )}
        </>
      ) : (
        <div className="empty-state">
          <p>Aucune vidéo dans cette catégorie pour le moment</p>
          <Link href="/">Retour à l'accueil</Link>
        </div>
      )}
    </div>
  )
}

