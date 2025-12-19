'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminTorrentConfigs.css'

export default function AdminTorrentConfigsPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [configs, setConfigs] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    source: '1337x',
    enabled: true,
    auto_download: false,
    search_keywords: '',
    base_url: 'https://1337x.quest',
    min_seeders: 1,
    max_size: 0,
    quality: '',
    language: ''
  })

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/torrent-configs')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchConfigs()
    }
    checkAuth()
  }, [router])

  const fetchConfigs = async () => {
    try {
      setLoading(true)
      const response = await api.get('/admin/torrent-configs')
      setConfigs(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.post('/admin/torrent-configs', formData)
      setShowAddForm(false)
      setFormData({
        name: '',
        source: '1337x',
        enabled: true,
        auto_download: false,
        search_keywords: '',
        base_url: 'https://1337x.quest',
        min_seeders: 1,
        max_size: 0,
        quality: '',
        language: ''
      })
      fetchConfigs()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la création')
    }
  }

  const handleToggle = async (id: string, field: string, value: boolean) => {
    try {
      await api.put(`/admin/torrent-configs/${id}`, { [field]: value })
      fetchConfigs()
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer cette configuration ?')) {
      return
    }
    try {
      await api.delete(`/admin/torrent-configs/${id}`)
      fetchConfigs()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la suppression')
    }
  }

  if (loading) {
    return (
      <div className="admin-torrent-configs">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="admin-torrent-configs">
      <div className="admin-header">
        <h1>⚙️ Configurations Torrent</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
          {showAddForm ? 'Annuler' : '+ Ajouter'}
        </button>
      </div>

      {showAddForm && (
        <form className="add-config-form" onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Nom"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <select
            value={formData.source}
            onChange={(e) => setFormData({ ...formData, source: e.target.value })}
          >
            <option value="1337x">1337x</option>
            <option value="piratebay">The Pirate Bay</option>
            <option value="torrentdownload">TorrentDownload</option>
          </select>
          <input
            type="text"
            placeholder="Mots-clés de recherche"
            value={formData.search_keywords}
            onChange={(e) => setFormData({ ...formData, search_keywords: e.target.value })}
          />
          <button type="submit">Créer</button>
        </form>
      )}

      <div className="configs-list">
        {configs.map((config: any) => (
          <div key={config.id} className="config-card">
            <h3>{config.name}</h3>
            <p>Source: {config.source}</p>
            <p>Mots-clés: {config.search_keywords}</p>
            <div className="config-actions">
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.enabled}
                  onChange={(e) => handleToggle(config.id, 'enabled', e.target.checked)}
                />
                <span>Actif</span>
              </label>
              <button onClick={() => handleDelete(config.id)} className="btn-delete">
                🗑️ Supprimer
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

