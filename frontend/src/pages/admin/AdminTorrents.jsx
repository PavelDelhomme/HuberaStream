import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './AdminTorrents.css'

const AdminTorrents = () => {
  const { user, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [torrents, setTorrents] = useState([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    magnet_url: '',
    torrent_url: '',
    source: 'custom',
    category_id: '',
    sub_category_id: ''
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
    fetchTorrents()
    fetchCategories()
  }, [user, isAdmin, navigate])

  const fetchTorrents = async () => {
    try {
      setLoading(true)
      const response = await api.get('/admin/torrents')
      setTorrents(response.data.torrents || [])
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
      await api.post('/admin/torrents', formData)
      setShowAddForm(false)
      setFormData({
        name: '',
        magnet_url: '',
        torrent_url: '',
        source: 'custom',
        category_id: '',
        sub_category_id: ''
      })
      fetchTorrents()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de l\'ajout du torrent')
    }
  }

  const formatBytes = (bytes) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  const getStatusColor = (status) => {
    const colors = {
      pending: '#aaaaaa',
      downloading: '#3498db',
      seeding: '#2ecc71',
      completed: '#27ae60',
      error: '#e74c3c',
      paused: '#f39c12'
    }
    return colors[status] || '#aaaaaa'
  }

  if (loading) {
    return <div className="admin-torrents">Chargement...</div>
  }

  return (
    <div className="admin-torrents">
      <div className="admin-header">
        <h1>Gestion des torrents</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
          {showAddForm ? 'Annuler' : '+ Ajouter un torrent'}
        </button>
      </div>

      {showAddForm && (
        <form className="add-torrent-form" onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Nom du torrent"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <input
            type="text"
            placeholder="Magnet URL"
            value={formData.magnet_url}
            onChange={(e) => setFormData({ ...formData, magnet_url: e.target.value })}
          />
          <input
            type="text"
            placeholder="Torrent URL"
            value={formData.torrent_url}
            onChange={(e) => setFormData({ ...formData, torrent_url: e.target.value })}
          />
          <select
            value={formData.source}
            onChange={(e) => setFormData({ ...formData, source: e.target.value })}
          >
            <option value="custom">Custom</option>
            <option value="1337x">1337x</option>
            <option value="rarbg">RARBG</option>
            <option value="yts">YTS</option>
            <option value="nyaa">Nyaa</option>
          </select>
          <button type="submit">Ajouter</button>
        </form>
      )}

      <div className="torrents-list">
        {torrents.map((torrent) => (
          <div key={torrent.id} className="torrent-card">
            <div className="torrent-header">
              <h3>{torrent.name}</h3>
              <span
                className="torrent-status"
                style={{ backgroundColor: getStatusColor(torrent.status) }}
              >
                {torrent.status}
              </span>
            </div>
            <div className="torrent-info">
              <div className="torrent-progress">
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{ width: `${torrent.progress}%` }}
                  />
                </div>
                <span>{torrent.progress.toFixed(1)}%</span>
              </div>
              <div className="torrent-details">
                <span>Taille: {formatBytes(torrent.size)}</span>
                <span>Téléchargé: {formatBytes(torrent.downloaded)}</span>
                <span>Source: {torrent.source}</span>
                {torrent.download_speed > 0 && (
                  <span>Vitesse: {formatBytes(torrent.download_speed)}/s</span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default AdminTorrents

