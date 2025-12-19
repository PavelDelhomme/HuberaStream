'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminUsers.css'

export default function AdminUsersPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [filter, setFilter] = useState('all')
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/users')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchUsers()
    }
    checkAuth()
  }, [router, page, filter, searchQuery])

  const fetchUsers = async () => {
    try {
      setLoading(true)
      const params: any = { page, limit: 20 }
      if (searchQuery) {
        params.search = searchQuery
      }
      const response = await api.get('/admin/users', { params })
      let filteredUsers = response.data.users || []
      
      if (filter === 'pending') {
        filteredUsers = filteredUsers.filter((u: any) => !u.is_active)
      } else if (filter === 'active') {
        filteredUsers = filteredUsers.filter((u: any) => u.is_active)
      }
      
      setUsers(filteredUsers)
      setTotal(response.data.total || 0)
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleUpdateUser = async (userId: string, updates: any) => {
    try {
      await api.put(`/admin/users/${userId}`, updates)
      fetchUsers()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la mise à jour')
    }
  }

  const handleDeleteUser = async (userId: string) => {
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

  if (loading && users.length === 0) {
    return (
      <div className="admin-users">
        <div className="loading">Chargement des utilisateurs...</div>
      </div>
    )
  }

  const pendingCount = users.filter((u: any) => !u.is_active).length
  const activeCount = users.filter((u: any) => u.is_active).length

  return (
    <div className="admin-users">
      <div className="admin-header">
        <div className="admin-header-content">
          <h1>👥 Gestion des Utilisateurs</h1>
          <p className="subtitle">Validez les demandes d'inscription et gérez les comptes</p>
        </div>
        <div className="admin-header-stats">
          <div className="stat-badge">
            <span className="stat-badge-label">Total</span>
            <span className="stat-badge-value">{total}</span>
          </div>
          {pendingCount > 0 && (
            <div className="stat-badge pending">
              <span className="stat-badge-label">En attente</span>
              <span className="stat-badge-value">{pendingCount}</span>
            </div>
          )}
        </div>
      </div>

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
            ✅ Actifs ({activeCount})
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
            {users.map((u: any) => (
              <tr key={u.id} className={!u.is_active ? 'pending-user' : ''}>
                <td>
                  <strong>{u.username || u.email}</strong>
                  {!u.is_active && <span className="pending-badge">⏳ En attente</span>}
                </td>
                <td>{u.email}</td>
                <td>{new Date(u.created_at).toLocaleDateString('fr-FR')}</td>
                <td>
                  <select
                    value={u.role || 'user'}
                    onChange={(e) => handleUpdateUser(u.id, { role: e.target.value })}
                    className="role-select"
                  >
                    <option value="user">User</option>
                    <option value="admin">Admin</option>
                  </select>
                </td>
                <td>
                  <label className={`toggle-button ${u.is_active ? 'enabled' : 'disabled'}`}>
                    <input
                      type="checkbox"
                      checked={u.is_active}
                      onChange={(e) => handleUpdateUser(u.id, { is_active: e.target.checked })}
                    />
                    <span className="toggle-checkbox-custom"></span>
                    <span className="toggle-text">{u.is_active ? 'Actif' : 'Inactif'}</span>
                  </label>
                </td>
                <td>
                  <div className="action-buttons">
                    {!u.is_active && (
                      <button
                        onClick={() => handleUpdateUser(u.id, { is_active: true })}
                        className="action-button btn-approve"
                        title="Valider l'inscription"
                      >
                        <span className="action-icon">✅</span>
                        <span className="action-text">Valider</span>
                      </button>
                    )}
                    <button
                      onClick={() => handleDeleteUser(u.id)}
                      className="action-button btn-delete"
                      disabled={u.id === user?.id}
                      title="Supprimer l'utilisateur"
                    >
                      <span className="action-icon">🗑️</span>
                      <span className="action-text">Supprimer</span>
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {total > 20 && (
        <div className="pagination">
          <button
            disabled={page === 1}
            onClick={() => setPage(page - 1)}
          >
            Précédent
          </button>
          <span>Page {page} sur {Math.ceil(total / 20)}</span>
          <button
            disabled={page >= Math.ceil(total / 20)}
            onClick={() => setPage(page + 1)}
          >
            Suivant
          </button>
        </div>
      )}
    </div>
  )
}

