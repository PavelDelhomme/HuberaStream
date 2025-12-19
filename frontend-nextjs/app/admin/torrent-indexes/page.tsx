'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminTorrentIndexes.css'

export default function AdminTorrentIndexesPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [indexes, setIndexes] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [testingIndex, setTestingIndex] = useState<string | null>(null)
  const [testResults, setTestResults] = useState<any>(null)
  const [formData, setFormData] = useState({
    name: '',
    source: '1337x',
    base_url: '',
    search_url: '/lmsearch?q={query}&category=lmsearch',
    enabled: true,
    priority: 0,
    is_default: false,
    description: ''
  })

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/torrent-indexes')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchIndexes()
    }
    checkAuth()
  }, [router])

  const fetchIndexes = async () => {
    try {
      setLoading(true)
      const response = await api.get('/admin/torrent-indexes')
      setIndexes(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const dataToSend = {
        ...formData,
        search_path: formData.search_url
      }
      await api.post('/admin/torrent-indexes', dataToSend)
      setShowAddForm(false)
      setFormData({
        name: '',
        source: '1337x',
        base_url: '',
        search_url: '/lmsearch?q={query}&category=lmsearch',
        enabled: true,
        priority: 0,
        is_default: false,
        description: ''
      })
      fetchIndexes()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la création')
    }
  }

  const handleTest = async (id: string) => {
    setTestingIndex(id)
    try {
      const response = await api.post(`/admin/torrent-indexes/${id}/test`)
      setTestResults({ id, ...response.data })
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors du test')
    } finally {
      setTestingIndex(null)
    }
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer cet index ?')) {
      return
    }
    try {
      await api.delete(`/admin/torrent-indexes/${id}`)
      fetchIndexes()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la suppression')
    }
  }

  if (loading) {
    return (
      <div className="admin-torrent-indexes">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="admin-torrent-indexes">
      <div className="admin-header">
        <h1>🔍 Index Torrent</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
          {showAddForm ? 'Annuler' : '+ Ajouter'}
        </button>
      </div>

      {showAddForm && (
        <form className="add-index-form" onSubmit={handleSubmit}>
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
            placeholder="URL de base"
            value={formData.base_url}
            onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
          />
          <button type="submit">Créer</button>
        </form>
      )}

      {testResults && (
        <div className="test-results-modal">
          <div className="modal-content">
            <h3>Résultats du test</h3>
            <p>{testResults.message}</p>
            {testResults.results && testResults.results.length > 0 && (
              <div className="results-list">
                {testResults.results.slice(0, 10).map((result: any, idx: number) => (
                  <div key={idx} className="result-item">
                    <strong>{result.name}</strong>
                    <span>📦 {result.size}</span>
                    <span>🌱 {result.seeders}</span>
                  </div>
                ))}
              </div>
            )}
            <button onClick={() => setTestResults(null)}>Fermer</button>
          </div>
        </div>
      )}

      <div className="indexes-list">
        {indexes.map((index: any) => (
          <div key={index.id} className="index-card">
            <h3>{index.name}</h3>
            <p>Source: {index.source}</p>
            <p>URL: {index.base_url}</p>
            <div className="index-actions">
              <button
                onClick={() => handleTest(index.id)}
                disabled={testingIndex === index.id}
                className="btn-test"
              >
                {testingIndex === index.id ? '⏳ Test...' : '🧪 Tester'}
              </button>
              <button onClick={() => handleDelete(index.id)} className="btn-delete">
                🗑️ Supprimer
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

