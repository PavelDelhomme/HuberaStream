'use client'

import { useState, useEffect } from 'react'
import api from '@/lib/api'
import './AdminDashboard.css'

export default function AdminDashboard() {
  const [stats, setStats] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchStats()
  }, [])

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
    return (
      <div className="admin-dashboard">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="admin-dashboard">
      <h1>📊 Tableau de Bord</h1>
      
      {stats && (
        <div className="stats-grid">
          <div className="stat-card">
            <div className="stat-icon">👥</div>
            <div className="stat-value">{stats.users_count || 0}</div>
            <div className="stat-label">Utilisateurs</div>
          </div>
          
          <div className="stat-card">
            <div className="stat-icon">🎬</div>
            <div className="stat-value">{stats.videos_count || 0}</div>
            <div className="stat-label">Vidéos</div>
          </div>
          
          <div className="stat-card">
            <div className="stat-icon">📁</div>
            <div className="stat-value">{stats.categories_count || 0}</div>
            <div className="stat-label">Catégories</div>
          </div>
        </div>
      )}
    </div>
  )
}

