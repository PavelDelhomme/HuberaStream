'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminCategories.css'

export default function AdminCategoriesPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [categories, setCategories] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    type: 'other',
    description: '',
    icon: ''
  })

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/categories')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchCategories()
    }
    checkAuth()
  }, [router])

  const fetchCategories = async () => {
    try {
      setLoading(true)
      const response = await api.get('/categories')
      setCategories(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await api.post('/admin/categories', formData)
      setShowAddForm(false)
      setFormData({ name: '', type: 'other', description: '', icon: '' })
      fetchCategories()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la création')
    }
  }

  if (loading) {
    return (
      <div className="admin-categories">
        <div className="loading">Chargement des catégories...</div>
      </div>
    )
  }

  return (
    <div className="admin-categories">
      <div className="admin-header">
        <div className="admin-header-content">
          <h1>📂 Gestion des Catégories</h1>
          <p className="subtitle">Organisez vos contenus avec des catégories et sous-catégories</p>
        </div>
        <div className="admin-header-actions">
          <div className="stat-badge">
            <span className="stat-badge-label">Total</span>
            <span className="stat-badge-value">{categories.length}</span>
          </div>
          <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
            {showAddForm ? '❌ Annuler' : '➕ Ajouter une catégorie'}
          </button>
        </div>
      </div>

      {showAddForm && (
        <form className="add-category-form" onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Nom de la catégorie"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <select
            value={formData.type}
            onChange={(e) => setFormData({ ...formData, type: e.target.value })}
          >
            <option value="anime">Anime</option>
            <option value="film">Film</option>
            <option value="series">Série</option>
            <option value="documentary">Documentaire</option>
            <option value="other">Autre</option>
          </select>
          <textarea
            placeholder="Description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          />
          <input
            type="text"
            placeholder="Icône (emoji)"
            value={formData.icon}
            onChange={(e) => setFormData({ ...formData, icon: e.target.value })}
          />
          <button type="submit">Créer</button>
        </form>
      )}

      <div className="categories-grid">
        {categories.map((category) => (
          <div key={category.id} className="category-card">
            <div className="category-icon">{category.icon || '📁'}</div>
            <h3>{category.name}</h3>
            <p className="category-type">{category.type}</p>
            {category.description && <p className="category-description">{category.description}</p>}
          </div>
        ))}
      </div>
    </div>
  )
}

