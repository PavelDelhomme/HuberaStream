import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './AdminDashboard.css'

const AdminDashboard = () => {
  const { user, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchStats()
  }, [user, isAdmin, navigate])

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
        <h1>Tableau de bord administrateur</h1>
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

      <div className="admin-menu">
        <Link to="/admin/users" className="admin-menu-card">
          <h2>👥 Utilisateurs</h2>
          <p>Gérer les utilisateurs et leurs permissions</p>
        </Link>
        <Link to="/admin/torrents" className="admin-menu-card">
          <h2>📥 Torrents</h2>
          <p>Gérer les téléchargements torrent</p>
        </Link>
        <Link to="/admin/torrent-configs" className="admin-menu-card">
          <h2>⚙️ Configurations Torrent</h2>
          <p>Configurer les téléchargements automatiques</p>
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

