import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminUsers.css'

const AdminUsers = () => {
  const { user, loading: authLoading, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [filter, setFilter] = useState('all') // all, active, pending
  const [searchQuery, setSearchQuery] = useState('')

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
    fetchUsers()
  }, [user, authLoading, isAdmin, navigate, page, filter, searchQuery])

  const fetchUsers = async () => {
    try {
      setLoading(true)
      const params = { page, limit: 20 }
      if (searchQuery) {
        params.search = searchQuery
      }
      const response = await api.get('/admin/users', { params })
      let filteredUsers = response.data.users || []
      
      // Filtrer par statut
      if (filter === 'pending') {
        filteredUsers = filteredUsers.filter(u => !u.is_active)
      } else if (filter === 'active') {
        filteredUsers = filteredUsers.filter(u => u.is_active)
      }
      
      setUsers(filteredUsers)
      setTotal(response.data.total || 0)
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleUpdateUser = async (userId, updates) => {
    try {
      await api.put(`/admin/users/${userId}`, updates)
      fetchUsers()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la mise à jour')
    }
  }

  const handleDeleteUser = async (userId) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer cet utilisateur ?')) {
      return
    }
    try {
      await api.delete(`/admin/users/${userId}`)
      fetchUsers()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la suppression')
    }
  }

  if (loading) {
    return (
      <div className="admin-users">
        <LoadingSpinner message="Chargement des utilisateurs..." />
      </div>
    )
  }

  const pendingCount = users.filter(u => !u.is_active).length

  return (
    <div className="admin-users">
      <div className="admin-header">
        <h1>👥 Gestion des Utilisateurs</h1>
        <p className="subtitle">Validez les demandes d'inscription et gérez les comptes</p>
      </div>

      {/* Filtres et recherche */}
      <div className="users-filters">
        <div className="filter-tabs">
          <button
            className={`filter-tab ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
          >
            Tous ({users.length})
          </button>
          <button
            className={`filter-tab ${filter === 'pending' ? 'active' : ''}`}
            onClick={() => setFilter('pending')}
          >
            ⏳ En attente ({pendingCount})
          </button>
          <button
            className={`filter-tab ${filter === 'active' ? 'active' : ''}`}
            onClick={() => setFilter('active')}
          >
            ✅ Actifs ({users.filter(u => u.is_active).length})
          </button>
        </div>
        <div className="search-box">
          <input
            type="text"
            placeholder="Rechercher un utilisateur..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="search-input"
          />
        </div>
      </div>

      {/* Alerte pour les utilisateurs en attente */}
      {filter === 'all' && pendingCount > 0 && (
        <div className="pending-alert">
          <span>⚠️</span>
          <span>{pendingCount} demande(s) d'inscription en attente de validation</span>
        </div>
      )}

      <div className="users-table">
        <table>
          <thead>
            <tr>
              <th>Username</th>
              <th>Email</th>
              <th>Inscription</th>
              <th>Rôle</th>
              <th>Statut</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} className={!u.is_active ? 'pending-user' : ''}>
                <td>
                  <strong>{u.username}</strong>
                  {!u.is_active && <span className="pending-badge">⏳ En attente</span>}
                </td>
                <td>{u.email}</td>
                <td>{new Date(u.created_at).toLocaleDateString('fr-FR')}</td>
                <td>
                  <select
                    value={u.role}
                    onChange={(e) => handleUpdateUser(u.id, { role: e.target.value })}
                    className="role-select"
                  >
                    <option value="user">User</option>
                    <option value="admin">Admin</option>
                  </select>
                </td>
                <td>
                  <label className="toggle-switch">
                    <input
                      type="checkbox"
                      checked={u.is_active}
                      onChange={(e) => handleUpdateUser(u.id, { is_active: e.target.checked })}
                    />
                    <span className="toggle-slider"></span>
                    <span className="toggle-label">{u.is_active ? 'Actif' : 'Inactif'}</span>
                  </label>
                </td>
                <td>
                  <div className="action-buttons">
                    {!u.is_active && (
                      <button
                        onClick={() => handleUpdateUser(u.id, { is_active: true })}
                        className="approve-button"
                        title="Valider l'inscription"
                      >
                        ✅ Valider
                      </button>
                    )}
                    <button
                      onClick={() => handleDeleteUser(u.id)}
                      className="delete-button"
                      disabled={u.id === user.id}
                      title="Supprimer l'utilisateur"
                    >
                      🗑️ Supprimer
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="pagination">
        <button
          onClick={() => setPage(p => Math.max(1, p - 1))}
          disabled={page === 1}
        >
          Précédent
        </button>
        <span>Page {page} sur {Math.ceil(total / 20)}</span>
        <button
          onClick={() => setPage(p => p + 1)}
          disabled={page >= Math.ceil(total / 20)}
        >
          Suivant
        </button>
      </div>
    </div>
  )
}

export default AdminUsers

