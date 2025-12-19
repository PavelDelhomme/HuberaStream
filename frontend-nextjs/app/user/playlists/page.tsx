'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { authService } from '@/lib/auth'
import api from '@/lib/api'
import './Playlists.css'

export default function PlaylistsPage() {
  const router = useRouter()
  const [playlists, setPlaylists] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newPlaylistName, setNewPlaylistName] = useState('')
  const [newPlaylistDescription, setNewPlaylistDescription] = useState('')

  useEffect(() => {
    const loadPlaylists = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login')
        return
      }
      fetchPlaylists()
    }
    loadPlaylists()
  }, [router])

  const fetchPlaylists = async () => {
    try {
      setLoading(true)
      const response = await api.get('/playlists')
      setPlaylists(response.data || [])
      setError(null)
    } catch (err: any) {
      console.error('Erreur lors de la récupération des playlists:', err)
      setError('Impossible de charger les listes')
    } finally {
      setLoading(false)
    }
  }

  const handleCreatePlaylist = async (e: React.FormEvent) => {
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
    } catch (err: any) {
      console.error('Erreur lors de la création de la playlist:', err)
      alert('Erreur lors de la création de la playlist')
    }
  }

  if (loading) {
    return (
      <div className="playlists-container">
        <div className="loading">Chargement des listes...</div>
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
        <h1>📚 Mes Listes</h1>
        <button 
          onClick={() => setShowCreateForm(!showCreateForm)} 
          className="btn-primary"
        >
          {showCreateForm ? 'Annuler' : '+ Nouvelle liste'}
        </button>
      </div>

      {showCreateForm && (
        <div className="create-playlist-form">
          <h2>Créer une nouvelle liste</h2>
          <form onSubmit={handleCreatePlaylist}>
            <div className="form-group">
              <label>Nom de la liste</label>
              <input
                type="text"
                value={newPlaylistName}
                onChange={(e) => setNewPlaylistName(e.target.value)}
                required
                placeholder="Ma liste de films"
              />
            </div>
            <div className="form-group">
              <label>Description (optionnel)</label>
              <textarea
                value={newPlaylistDescription}
                onChange={(e) => setNewPlaylistDescription(e.target.value)}
                placeholder="Description de la liste..."
                rows={3}
              />
            </div>
            <button type="submit" className="btn-primary">
              Créer la liste
            </button>
          </form>
        </div>
      )}

      {playlists.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">📚</div>
          <p>Vous n'avez pas encore de listes</p>
          <p className="empty-hint">Créez votre première liste pour organiser vos vidéos</p>
        </div>
      ) : (
        <div className="playlists-grid">
          {playlists.map((playlist) => (
            <Link 
              key={playlist.id} 
              href={`/playlist/${playlist.id}`} 
              className="playlist-item"
            >
              <div className="playlist-icon">📚</div>
              <div className="playlist-info">
                <h3>{playlist.name}</h3>
                {playlist.description && (
                  <p className="playlist-description">{playlist.description}</p>
                )}
                <p className="playlist-count">
                  {playlist.videos?.length || 0} vidéo(s)
                </p>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

