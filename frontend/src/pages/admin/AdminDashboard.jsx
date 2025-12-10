import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminDashboard.css'

const AdminDashboard = () => {
  const { user, isAdmin, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Attendre que le chargement de l'authentification soit terminé
    if (authLoading) {
      console.log('AdminDashboard: En attente du chargement de l\'authentification...')
      return
    }
    
    console.log('AdminDashboard: authLoading terminé, user:', user, 'isAdmin:', isAdmin ? isAdmin() : 'non défini')
    
    if (!user) {
      console.log('AdminDashboard: Pas d\'utilisateur, redirection vers /login')
      navigate('/login')
      return
    }
    
    const adminCheck = isAdmin()
    console.log('AdminDashboard: Vérification admin - user.role:', user?.role, 'isAdmin():', adminCheck)
    
    if (!adminCheck) {
      console.log('AdminDashboard: Utilisateur n\'est pas admin, redirection vers /')
      navigate('/')
      return
    }
    
    console.log('AdminDashboard: Utilisateur est admin, chargement des stats...')
    fetchStats()
    
    // Écouter les événements de nettoyage de vidéos pour rafraîchir les stats
    const handleVideosCleaned = () => {
      console.log('AdminDashboard: Événement videosCleaned reçu, rafraîchissement des stats...')
      fetchStats()
    }
    
    window.addEventListener('videosCleaned', handleVideosCleaned)
    
    // Nettoyer l'écouteur au démontage
    return () => {
      window.removeEventListener('videosCleaned', handleVideosCleaned)
    }
  }, [user, authLoading, isAdmin, navigate])

  const fetchStats = async () => {
    try {
      setLoading(true)
      console.log('AdminDashboard: Début du chargement des stats...')
      
      // Essayer de récupérer les stats depuis l'API admin
      try {
        const response = await api.get('/admin/stats')
        console.log('AdminDashboard: Stats récupérées depuis /admin/stats:', response.data)
        setStats(response.data)
      } catch (statsError) {
        // Si l'endpoint n'existe pas, calculer les stats depuis les autres endpoints
        console.log('Endpoint /admin/stats non disponible, calcul des stats...')
        const [usersRes, videosRes, torrentsRes, categoriesRes] = await Promise.allSettled([
          api.get('/admin/users?page=1&limit=1'),
          api.get('/videos?page=1&limit=1'),
          api.get('/admin/torrents'),
          api.get('/categories')
        ])
        
        const stats = {
          total_users: usersRes.status === 'fulfilled' ? (usersRes.value.data?.total || usersRes.value.data?.users?.length || 0) : 0,
          total_videos: videosRes.status === 'fulfilled' ? (videosRes.value.data?.total || videosRes.value.data?.videos?.length || 0) : 0,
          total_torrents: torrentsRes.status === 'fulfilled' ? (torrentsRes.value.data?.torrents?.length || 0) : 0,
          active_torrents: torrentsRes.status === 'fulfilled' 
            ? (torrentsRes.value.data?.torrents?.filter(t => t.status === 'downloading' || t.status === 'seeding')?.length || 0) 
            : 0,
          total_categories: categoriesRes.status === 'fulfilled' ? (categoriesRes.value.data?.length || 0) : 0
        }
        setStats(stats)
      }
    } catch (error) {
      console.error('AdminDashboard: Erreur lors de la récupération des statistiques:', error)
      // Stats par défaut en cas d'erreur
      setStats({
        total_users: 0,
        total_videos: 0,
        total_torrents: 0,
        active_torrents: 0,
        total_categories: 0
      })
    } finally {
      console.log('AdminDashboard: Fin du chargement des stats, setLoading(false)')
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="admin-dashboard">
        <LoadingSpinner message="Chargement du tableau de bord..." />
      </div>
    )
  }

  return (
    <div className="admin-dashboard">
      <div className="admin-header">
        <div className="admin-header-title">
          <h1>📊 Tableau de bord administrateur</h1>
          <p className="admin-subtitle">Vue d'ensemble de votre plateforme StreamMake</p>
        </div>
        <div className="admin-header-actions">
          <Link to="/upload" className="admin-action-button primary">
            ⬆️ Upload Vidéo
          </Link>
          <Link to="/dvd-extract" className="admin-action-button secondary">
            📀 Extraire DVD
          </Link>
        </div>
      </div>

      {/* Statistiques principales */}
      <div className="admin-stats">
        <div className="stat-card stat-card-users">
          <div className="stat-icon">👥</div>
          <div className="stat-content">
            <h3>Utilisateurs</h3>
            <p className="stat-value">{stats?.total_users || 0}</p>
            <Link to="/admin/users" className="stat-link">Voir tous →</Link>
          </div>
        </div>
        <div className="stat-card stat-card-videos">
          <div className="stat-icon">🎬</div>
          <div className="stat-content">
            <h3>Vidéos</h3>
            <p className="stat-value">{stats?.total_videos || 0}</p>
            <Link to="/admin/videos" className="stat-link">Gérer →</Link>
          </div>
        </div>
        <div className="stat-card stat-card-torrents">
          <div className="stat-icon">📥</div>
          <div className="stat-content">
            <h3>Torrents</h3>
            <p className="stat-value">{stats?.total_torrents || 0}</p>
            <Link to="/admin/torrents" className="stat-link">Gérer →</Link>
          </div>
        </div>
        <div className="stat-card stat-card-active">
          <div className="stat-icon">⚡</div>
          <div className="stat-content">
            <h3>Torrents actifs</h3>
            <p className="stat-value">{stats?.active_torrents || 0}</p>
            <Link to="/admin/torrents" className="stat-link">Voir →</Link>
          </div>
        </div>
        <div className="stat-card stat-card-categories">
          <div className="stat-icon">📂</div>
          <div className="stat-content">
            <h3>Catégories</h3>
            <p className="stat-value">{stats?.total_categories || 0}</p>
            <Link to="/admin/categories" className="stat-link">Gérer →</Link>
          </div>
        </div>
      </div>

      {/* Actions rapides */}
      <div className="admin-quick-actions">
        <h2 className="section-title">⚡ Actions rapides</h2>
        <div className="quick-actions-grid">
          <Link to="/upload" className="quick-action-card upload-card">
            <div className="quick-action-icon">⬆️</div>
            <h3>Upload Vidéo</h3>
            <p>Téléverser une nouvelle vidéo sur la plateforme</p>
          </Link>
          <Link to="/dvd-extract" className="quick-action-card dvd-card">
            <div className="quick-action-icon">📀</div>
            <h3>Extraire DVD</h3>
            <p>Extraire le contenu d'un DVD physique</p>
          </Link>
        </div>
      </div>

      {/* Menu d'administration */}
      <div className="admin-menu">
        <h2 className="section-title">⚙️ Administration</h2>
        <div className="admin-menu-grid">
          <Link to="/admin/videos" className="admin-menu-card">
            <div className="menu-card-icon">📹</div>
            <h3>Gérer Vidéos</h3>
            <p>Gérer toutes les vidéos de la plateforme</p>
          </Link>
          <Link to="/admin/users" className="admin-menu-card">
            <div className="menu-card-icon">👥</div>
            <h3>Utilisateurs</h3>
            <p>Gérer les utilisateurs et leurs permissions</p>
          </Link>
          <Link to="/admin/categories" className="admin-menu-card">
            <div className="menu-card-icon">📂</div>
            <h3>Catégories</h3>
            <p>Gérer les catégories et sous-catégories</p>
          </Link>
          <Link to="/admin/torrents" className="admin-menu-card">
            <div className="menu-card-icon">📥</div>
            <h3>Torrents</h3>
            <p>Gérer les téléchargements torrent</p>
          </Link>
          <Link to="/admin/torrent-clients" className="admin-menu-card">
            <div className="menu-card-icon">⚙️</div>
            <h3>Clients Torrent</h3>
            <p>Configurer les clients torrent</p>
          </Link>
          <Link to="/admin/torrent-configs" className="admin-menu-card">
            <div className="menu-card-icon">🔧</div>
            <h3>Configurations</h3>
            <p>Gérer les configurations de téléchargement</p>
          </Link>
          <Link to="/admin/media-settings" className="admin-menu-card">
            <div className="menu-card-icon">🎬</div>
            <h3>Paramètres Média</h3>
            <p>Configurer les paramètres de traitement vidéo</p>
          </Link>
        </div>
      </div>
    </div>
  )
}

export default AdminDashboard

