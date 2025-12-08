import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './AdminDashboard.css'

const AdminDashboard = () => {
  const { user, isAdmin, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Attendre que le chargement de l'authentification soit terminé
    if (authLoading) return
    
    if (!user) {
      navigate('/login')
      return
    }
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchStats()
  }, [user, authLoading, isAdmin, navigate])

  const fetchStats = async () => {
    try {
      const response = await api.get('/admin/stats')
      setStats(response.data)
    } catch (error) {
      console.error('Erreur lors de la récupération des statistiques:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div className="admin-dashboard">Chargement...</div>
  }

  return (
    <div className="admin-dashboard">
      <div className="admin-header">
        <h1>📊 Tableau de bord administrateur</h1>
        <div className="admin-header-actions">
          <Link to="/upload" className="admin-action-button primary">
            ⬆️ Upload Vidéo
          </Link>
          <Link to="/dvd-extract" className="admin-action-button secondary">
            📀 Extraire DVD
          </Link>
        </div>
      </div>

      <div className="admin-stats">
        <div className="stat-card">
          <h3>Utilisateurs</h3>
          <p className="stat-value">{stats?.total_users || 0}</p>
        </div>
        <div className="stat-card">
          <h3>Vidéos</h3>
          <p className="stat-value">{stats?.total_videos || 0}</p>
        </div>
        <div className="stat-card">
          <h3>Torrents</h3>
          <p className="stat-value">{stats?.total_torrents || 0}</p>
        </div>
        <div className="stat-card">
          <h3>Torrents actifs</h3>
          <p className="stat-value">{stats?.active_torrents || 0}</p>
        </div>
        <div className="stat-card">
          <h3>Catégories</h3>
          <p className="stat-value">{stats?.total_categories || 0}</p>
        </div>
      </div>

      {/* Actions rapides */}
      <div className="admin-quick-actions">
        <Link to="/upload" className="quick-action-card upload-card">
          <div className="quick-action-icon">⬆️</div>
          <h2>Upload Vidéo</h2>
          <p>Téléverser une nouvelle vidéo sur la plateforme</p>
        </Link>
        <Link to="/dvd-extract" className="quick-action-card dvd-card">
          <div className="quick-action-icon">📀</div>
          <h2>Extraire DVD</h2>
          <p>Extraire le contenu d'un DVD physique</p>
        </Link>
      </div>

      <div className="admin-menu">
        <Link to="/admin/users" className="admin-menu-card">
          <h2>👥 Utilisateurs</h2>
          <p>Gérer les utilisateurs et leurs permissions</p>
        </Link>
        <Link to="/admin/torrents" className="admin-menu-card">
          <h2>📥 Torrents</h2>
          <p>Gérer les téléchargements torrent</p>
        </Link>
        <Link to="/admin/torrent-clients" className="admin-menu-card">
          <h2>⚙️ Clients Torrent</h2>
          <p>Configurer les clients torrent (Transmission, qBittorrent, etc.)</p>
        </Link>
        <Link to="/admin/torrent-configs" className="admin-menu-card">
          <h2>⚙️ Configurations Torrent</h2>
          <p>Configurer les téléchargements automatiques</p>
        </Link>
        <Link to="/admin/media-settings" className="admin-menu-card">
          <h2>🎬 Paramètres Média</h2>
          <p>Configurer le transcodage et le stockage</p>
        </Link>
        <Link to="/admin/categories" className="admin-menu-card">
          <h2>📂 Catégories</h2>
          <p>Gérer les catégories et sous-catégories</p>
        </Link>
      </div>
    </div>
  )
}

export default AdminDashboard

