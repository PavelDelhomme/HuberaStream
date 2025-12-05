import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './AdminTorrentConfigs.css'

const AdminTorrentConfigs = () => {
  const { user, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [configs, setConfigs] = useState([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    source: '1337x',
    enabled: true,
    auto_download: false,
    search_keywords: '',
    min_seeders: 1,
    check_interval: 3600
  })
  const [categories, setCategories] = useState([])

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchConfigs()
    fetchCategories()
  }, [user, isAdmin, navigate])

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

  const fetchCategories = async () => {
    try {
      const response = await api.get('/categories')
      setCategories(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const handleSubmit = async (e) => {
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
        min_seeders: 1,
        check_interval: 3600
      })
      fetchConfigs()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la création')
    }
  }

  const handleToggle = async (configId, field, value) => {
    try {
      await api.put(`/admin/torrent-configs/${configId}`, { [field]: value })
      fetchConfigs()
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  if (loading) {
    return <div className="admin-torrent-configs">Chargement...</div>
  }

  return (
    <div className="admin-torrent-configs">
      <div className="admin-header">
        <h1>Configurations de téléchargement automatique</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
          {showAddForm ? 'Annuler' : '+ Nouvelle configuration'}
        </button>
      </div>

      {showAddForm && (
        <form className="add-config-form" onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Nom de la configuration"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <select
            value={formData.source}
            onChange={(e) => setFormData({ ...formData, source: e.target.value })}
          >
            <option value="1337x">1337x</option>
            <option value="rarbg">RARBG</option>
            <option value="yts">YTS</option>
            <option value="nyaa">Nyaa</option>
            <option value="custom">Custom</option>
          </select>
          <input
            type="text"
            placeholder="Mots-clés de recherche (séparés par virgules)"
            value={formData.search_keywords}
            onChange={(e) => setFormData({ ...formData, search_keywords: e.target.value })}
          />
          <div className="form-row">
            <label>
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
              />
              Activé
            </label>
            <label>
              <input
                type="checkbox"
                checked={formData.auto_download}
                onChange={(e) => setFormData({ ...formData, auto_download: e.target.checked })}
              />
              Téléchargement automatique
            </label>
          </div>
          <div className="form-row">
            <input
              type="number"
              placeholder="Seeders minimum"
              value={formData.min_seeders}
              onChange={(e) => setFormData({ ...formData, min_seeders: parseInt(e.target.value) })}
              min="1"
            />
            <input
              type="number"
              placeholder="Intervalle de vérification (secondes)"
              value={formData.check_interval}
              onChange={(e) => setFormData({ ...formData, check_interval: parseInt(e.target.value) })}
              min="60"
            />
          </div>
          <button type="submit">Créer</button>
        </form>
      )}

      <div className="configs-list">
        {configs.map((config) => (
          <div key={config.id} className="config-card">
            <div className="config-header">
              <h3>{config.name}</h3>
              <span className="config-source">{config.source}</span>
            </div>
            <div className="config-details">
              <p><strong>Mots-clés:</strong> {config.search_keywords || 'Aucun'}</p>
              <p><strong>Seeders minimum:</strong> {config.min_seeders}</p>
              <p><strong>Vérification:</strong> Toutes les {config.check_interval} secondes</p>
            </div>
            <div className="config-actions">
              <label className="toggle-switch">
                <input
                  type="checkbox"
                  checked={config.enabled}
                  onChange={(e) => handleToggle(config.id, 'enabled', e.target.checked)}
                />
                <span className="toggle-slider"></span>
                <span>Activé</span>
              </label>
              <label className="toggle-switch">
                <input
                  type="checkbox"
                  checked={config.auto_download}
                  onChange={(e) => handleToggle(config.id, 'auto_download', e.target.checked)}
                />
                <span className="toggle-slider"></span>
                <span>Auto-download</span>
              </label>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default AdminTorrentConfigs

