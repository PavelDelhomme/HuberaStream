'use client'

import { useEffect, useState } from 'react'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import Link from 'next/link'
import './Home.css'

export default function HomeContent() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [videos, setVideos] = useState<any[]>([])
  const [categories, setCategories] = useState<any[]>([])

  useEffect(() => {
    const loadData = async () => {
      if (authService.isAuthenticated()) {
        const currentUser = await authService.getCurrentUser()
        setUser(currentUser)
      }
      
      try {
        const [videosRes, categoriesRes] = await Promise.all([
          api.get('/videos', { params: { limit: 20 } }),
          api.get('/categories')
        ])
        setVideos(videosRes.data.videos || [])
        setCategories(categoriesRes.data || [])
      } catch (err) {
        console.error('Erreur lors du chargement:', err)
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [])

  if (loading) {
    return (
      <div className="home-container">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="home-container">
      <h1>Bienvenue sur StreamMake</h1>
      {user && (
        <p className="welcome-message">
          Bonjour, {user.email} {authService.isAdmin(user) && '(Admin)'}
        </p>
      )}
      
      {categories.length > 0 && (
        <section className="categories-section">
          <h2>Catégories</h2>
          <div className="categories-grid">
            {categories.map((cat) => (
              <Link key={cat.id} href={`/category/${cat.id}`} className="category-card">
                <h3>{cat.name}</h3>
              </Link>
            ))}
          </div>
        </section>
      )}

      {videos.length > 0 && (
        <section className="videos-section">
          <h2>Vidéos récentes</h2>
          <div className="videos-grid">
            {videos.map((video) => (
              <Link key={video.id} href={`/video/${video.id}`} className="video-card">
                <h3>{video.title}</h3>
                {video.description && <p>{video.description.substring(0, 100)}...</p>}
              </Link>
            ))}
          </div>
        </section>
      )}

      {!loading && videos.length === 0 && (
        <div className="empty-state">
          <p>Aucune vidéo disponible pour le moment</p>
        </div>
      )}
    </div>
  )
}

