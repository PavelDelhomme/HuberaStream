'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './Profile.css'

export default function ProfilePage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [profile, setProfile] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [userStats, setUserStats] = useState({ videosWatched: 0, playlists: 0, favorites: 0 })
  const [formData, setFormData] = useState({
    username: '',
    email: ''
  })

  useEffect(() => {
    const loadUser = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login')
        return
      }
      
      const currentUser = await authService.getCurrentUser()
      if (!currentUser) {
        router.push('/login')
        return
      }
      
      setUser(currentUser)
      fetchProfile()
    }
    loadUser()
  }, [router])

  const fetchProfile = async () => {
    try {
      const response = await api.get('/users/me')
      setProfile(response.data)
      setFormData({
        username: response.data.username || '',
        email: response.data.email || ''
      })
      fetchStats().then(setUserStats)
    } catch (error) {
      console.error('Erreur lors de la récupération du profil:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchStats = async () => {
    try {
      const [historyRes, playlistsRes] = await Promise.allSettled([
        api.get('/history'),
        api.get('/playlists')
      ])
      
      const historyCount = historyRes.status === 'fulfilled' 
        ? (Array.isArray(historyRes.value.data) ? historyRes.value.data.length : historyRes.value.data?.videos?.length || 0)
        : 0
      const playlistsCount = playlistsRes.status === 'fulfilled' 
        ? (playlistsRes.value.data?.length || 0)
        : 0
      
      return {
        videosWatched: historyCount,
        playlists: playlistsCount,
        favorites: 0
      }
    } catch (error) {
      console.error('Erreur lors de la récupération des statistiques:', error)
      return { videosWatched: 0, playlists: 0, favorites: 0 }
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      try {
        await api.put('/users/me', formData)
        alert('Profil mis à jour avec succès !')
        fetchProfile()
        setEditing(false)
      } catch (updateError: any) {
        if (updateError.response?.status === 404 || updateError.response?.status === 405) {
          alert('La mise à jour du profil n\'est pas encore disponible.')
        } else {
          throw updateError
        }
      }
    } catch (error) {
      console.error('Erreur lors de la mise à jour:', error)
      alert('Erreur lors de la mise à jour du profil')
    }
  }

  if (loading) {
    return (
      <div className="profile-page">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="profile-page">
      <div className="profile-header">
        <h1>👤 Mon Profil</h1>
      </div>

      <div className="profile-content">
        <div className="profile-card">
          <h2>Informations personnelles</h2>
          {editing ? (
            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Nom d'utilisateur</label>
                <input
                  type="text"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label>Email</label>
                <input
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                />
              </div>
              <div className="form-actions">
                <button type="submit" className="btn-primary">Enregistrer</button>
                <button type="button" onClick={() => setEditing(false)} className="btn-secondary">
                  Annuler
                </button>
              </div>
            </form>
          ) : (
            <div className="profile-info">
              <p><strong>Email:</strong> {profile?.email || user?.email}</p>
              <p><strong>Rôle:</strong> {user?.role === 'admin' ? 'Administrateur' : 'Utilisateur'}</p>
              <button onClick={() => setEditing(true)} className="btn-primary">
                Modifier
              </button>
            </div>
          )}
        </div>

        <div className="stats-card">
          <h2>Statistiques</h2>
          <div className="stats-grid">
            <div className="stat-item">
              <div className="stat-value">{userStats.videosWatched}</div>
              <div className="stat-label">Vidéos regardées</div>
            </div>
            <div className="stat-item">
              <div className="stat-value">{userStats.playlists}</div>
              <div className="stat-label">Listes</div>
            </div>
            <div className="stat-item">
              <div className="stat-value">{userStats.favorites}</div>
              <div className="stat-label">Favoris</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

