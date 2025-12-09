import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import api from '../services/api'
import './Profile.css'

const Profile = () => {
  const { user, setUser } = useAuth()
  const navigate = useNavigate()
  const [profile, setProfile] = useState(null)
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState(false)
  const [formData, setFormData] = useState({
    username: '',
    email: ''
  })

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    fetchProfile()
  }, [user, navigate])

  const fetchProfile = async () => {
    try {
      const response = await api.get('/users/me')
      setProfile(response.data)
      setFormData({
        username: response.data.username || '',
        email: response.data.email || ''
      })
    } catch (error) {
      console.error('Erreur lors de la récupération du profil:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      // TODO: Implémenter l'API de mise à jour du profil
      // await api.put('/users/me', formData)
      alert('Mise à jour du profil à implémenter')
      setEditing(false)
    } catch (error) {
      console.error('Erreur lors de la mise à jour:', error)
    }
  }

  if (loading) {
    return <div className="profile-page">Chargement...</div>
  }

  return (
    <div className="profile-page">
      <div className="profile-header">
        <h1>👤 Mon Profil</h1>
      </div>

      <div className="profile-content">
        <div className="profile-card">
          <div className="profile-avatar">
            <span>{profile?.username?.charAt(0).toUpperCase() || 'U'}</span>
          </div>
          
          {!editing ? (
            <div className="profile-info">
              <h2>{profile?.username || 'Utilisateur'}</h2>
              <p className="profile-email">{profile?.email || ''}</p>
              <p className="profile-role">
                {profile?.role === 'admin' ? '👑 Administrateur' : '👤 Utilisateur'}
              </p>
              <button onClick={() => setEditing(true)} className="btn-edit">
                ✏️ Modifier le profil
              </button>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="profile-form">
              <div className="form-group">
                <label>Nom d'utilisateur</label>
                <input
                  type="text"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  className="form-input"
                />
              </div>
              <div className="form-group">
                <label>Email</label>
                <input
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  className="form-input"
                />
              </div>
              <div className="form-actions">
                <button type="submit" className="btn-primary">Enregistrer</button>
                <button type="button" onClick={() => setEditing(false)} className="btn-secondary">
                  Annuler
                </button>
              </div>
            </form>
          )}
        </div>

        <div className="profile-stats">
          <h3>Statistiques</h3>
          <div className="stats-grid">
            <div className="stat-item">
              <span className="stat-value">-</span>
              <span className="stat-label">Vidéos regardées</span>
            </div>
            <div className="stat-item">
              <span className="stat-value">-</span>
              <span className="stat-label">Playlists</span>
            </div>
            <div className="stat-item">
              <span className="stat-value">-</span>
              <span className="stat-label">Favoris</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Profile

