import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminTorrentIndexes.css'

const AdminTorrentIndexes = () => {
  const { user, loading: authLoading, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [indexes, setIndexes] = useState([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [editingIndex, setEditingIndex] = useState(null)
  const [testingIndex, setTestingIndex] = useState(null)
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
    if (authLoading) return
    if (!user) {
      navigate('/login')
      return
    }
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchIndexes()
  }, [user, authLoading, isAdmin, navigate])

  const fetchIndexes = async () => {
    try {
      setLoading(true)
      const response = await api.get('/admin/torrent-indexes')
      setIndexes(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors du chargement des index')
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      // Convertir search_url en search_path pour le backend
      const dataToSend = {
        ...formData,
        search_path: formData.search_url, // Envoyer search_url comme search_path
        search_url: formData.search_url   // Garder aussi pour compatibilité
      }
      
      if (editingIndex) {
        await api.put(`/admin/torrent-indexes/${editingIndex}`, dataToSend)
        alert('Index mis à jour avec succès')
      } else {
        await api.post('/admin/torrent-indexes', dataToSend)
        alert('Index créé avec succès')
      }
      setShowAddForm(false)
      setEditingIndex(null)
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
      const errorMsg = error.response?.data?.error || 'Erreur lors de la sauvegarde'
      alert(`Erreur: ${errorMsg}`)
    }
  }

  const handleEdit = (index) => {
    setEditingIndex(index.id)
    setFormData({
      name: index.name,
      source: index.source,
      base_url: index.base_url,
      search_url: index.search_path || index.search_url || '/lmsearch?q={query}&category=lmsearch',
      enabled: index.enabled,
      priority: index.priority,
      is_default: index.is_default,
      description: index.description || ''
    })
    setShowAddForm(true)
  }

  const handleDelete = async (indexId) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer cet index ?')) {
      return
    }
    try {
      await api.delete(`/admin/torrent-indexes/${indexId}`)
      alert('Index supprimé avec succès')
      fetchIndexes()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la suppression')
    }
  }

  const handleTest = async (indexId) => {
    setTestingIndex(indexId)
    try {
      const response = await api.post(`/admin/torrent-indexes/${indexId}/test`)
      if (response.data.success) {
        alert(`✅ Test réussi ! ${response.data.message}`)
      } else {
        alert(`❌ Test échoué : ${response.data.message}`)
      }
    } catch (error) {
      console.error('Erreur:', error)
      const errorMsg = error.response?.data?.error || error.response?.data?.message || 'Erreur lors du test'
      alert(`❌ Erreur lors du test : ${errorMsg}`)
    } finally {
      setTestingIndex(null)
    }
  }

  const handleCancel = () => {
    setShowAddForm(false)
    setEditingIndex(null)
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
  }

  if (loading) {
    return (
      <div className="admin-torrent-indexes">
        <LoadingSpinner message="Chargement des index..." />
      </div>
    )
  }

  return (
    <div className="admin-torrent-indexes">
      <div className="admin-header">
        <div className="admin-header-content">
          <h1>🔍 Index de Torrents</h1>
          <p className="subtitle">Configurez les sites de torrents pour la recherche</p>
        </div>
        {!showAddForm && (
          <button
            className="action-button btn-add"
            onClick={() => setShowAddForm(true)}
          >
            <span className="action-icon">➕</span>
            <span className="action-text">Nouvel index</span>
          </button>
        )}
      </div>

      {showAddForm && (
        <div className="add-form">
          <h2>{editingIndex ? 'Modifier l\'index' : 'Nouvel index'}</h2>
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label>Nom de l'index *</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="ex: 1337x.quest (Par défaut)"
                required
              />
            </div>

            <div className="form-group">
              <label>Source *</label>
              <select
                value={formData.source}
                onChange={(e) => setFormData({ ...formData, source: e.target.value })}
                required
              >
                <option value="1337x">1337x</option>
                <option value="piratebay">The Pirate Bay</option>
                <option value="torrentdownload">TorrentDownload</option>
                <option value="rarbg">RARBG</option>
                <option value="yts">YTS</option>
                <option value="nyaa">Nyaa</option>
              </select>
            </div>

            <div className="form-group">
              <label>URL de base *</label>
              <input
                type="url"
                value={formData.base_url}
                onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                placeholder="https://1337x.quest"
                required
              />
            </div>

            <div className="form-group">
              <label>Chemin de recherche (search_path)</label>
              <input
                type="text"
                value={formData.search_url}
                onChange={(e) => setFormData({ ...formData, search_url: e.target.value })}
                placeholder="/lmsearch?q={query}&category=lmsearch"
              />
              <small>Utilisez {"{query}"} comme placeholder pour la requête de recherche. Ce champ est envoyé comme "search_path" au backend.</small>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Priorité</label>
                <input
                  type="number"
                  value={formData.priority}
                  onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })}
                  min="0"
                />
                <small>Plus élevé = utilisé en premier</small>
              </div>

              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                  />
                  <span>Activé</span>
                </label>
              </div>

              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.is_default}
                    onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                  />
                  <span>Index par défaut</span>
                </label>
              </div>
            </div>

            <div className="form-group">
              <label>Description</label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="Description de l'index..."
                rows="3"
              />
            </div>

            <div className="form-actions">
              <button type="submit" className="action-button btn-save">
                <span className="action-icon">💾</span>
                <span className="action-text">{editingIndex ? 'Mettre à jour' : 'Créer'}</span>
              </button>
              <button type="button" onClick={handleCancel} className="action-button btn-cancel">
                <span className="action-icon">❌</span>
                <span className="action-text">Annuler</span>
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="indexes-list">
        {indexes.length === 0 ? (
          <div className="empty-state">
            <p className="empty-state-icon">🔍</p>
            <p className="empty-state-title">Aucun index configuré</p>
            <p className="empty-state-message">
              Créez votre premier index de torrents pour commencer à rechercher.
            </p>
          </div>
        ) : (
          indexes.map((index) => (
            <div key={index.id} className={`index-card ${!index.enabled ? 'disabled' : ''}`}>
              <div className="index-header">
                <div className="index-title">
                  <h3>{index.name}</h3>
                  <div className="index-badges">
                    {index.is_default && <span className="badge badge-default">Par défaut</span>}
                    {index.enabled ? (
                      <span className="badge badge-enabled">Activé</span>
                    ) : (
                      <span className="badge badge-disabled">Désactivé</span>
                    )}
                    <span className="badge badge-source">{index.source}</span>
                  </div>
                </div>
                <div className="index-actions">
                  <button
                    className="action-button btn-test"
                    onClick={() => handleTest(index.id)}
                    disabled={testingIndex === index.id}
                    title="Tester l'index"
                  >
                    <span className="action-icon">{testingIndex === index.id ? '⏳' : '🧪'}</span>
                    <span className="action-text">{testingIndex === index.id ? 'Test...' : 'Tester'}</span>
                  </button>
                  <button
                    className="action-button btn-edit"
                    onClick={() => handleEdit(index)}
                    title="Modifier l'index"
                  >
                    <span className="action-icon">✏️</span>
                    <span className="action-text">Modifier</span>
                  </button>
                  <button
                    className="action-button btn-delete"
                    onClick={() => handleDelete(index.id)}
                    title="Supprimer l'index"
                  >
                    <span className="action-icon">🗑️</span>
                    <span className="action-text">Supprimer</span>
                  </button>
                </div>
              </div>
              <div className="index-details">
                <div className="detail-item">
                  <strong>URL de base :</strong>
                  <span>{index.base_url}</span>
                </div>
                {index.search_url && (
                  <div className="detail-item">
                    <strong>URL de recherche :</strong>
                    <span>{index.search_url}</span>
                  </div>
                )}
                <div className="detail-item">
                  <strong>Priorité :</strong>
                  <span>{index.priority}</span>
                </div>
                {index.description && (
                  <div className="detail-item">
                    <strong>Description :</strong>
                    <span>{index.description}</span>
                  </div>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

export default AdminTorrentIndexes

