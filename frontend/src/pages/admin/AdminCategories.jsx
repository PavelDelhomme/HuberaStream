import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './AdminCategories.css'

const AdminCategories = () => {
  const { user, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [categories, setCategories] = useState([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    type: 'other',
    description: '',
    icon: ''
  })

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchCategories()
  }, [user, isAdmin, navigate])

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

  const handleSubmit = async (e) => {
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
    return <div className="admin-categories">Chargement...</div>
  }

  return (
    <div className="admin-categories">
      <div className="admin-header">
        <h1>Gestion des catégories</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
          {showAddForm ? 'Annuler' : '+ Ajouter une catégorie'}
        </button>
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
            <option value="other">Autre</option>
          </select>
          <textarea
            placeholder="Description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          />
          <input
            type="text"
            placeholder="Icône (emoji ou URL)"
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
            <p>{category.description}</p>
            <span className="category-type">{category.type}</span>
            {category.sub_categories && category.sub_categories.length > 0 && (
              <div className="sub-categories">
                <strong>Sous-catégories:</strong>
                <ul>
                  {category.sub_categories.map((sub) => (
                    <li key={sub.id}>{sub.name}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

export default AdminCategories

