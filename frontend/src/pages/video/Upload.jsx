import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './Upload.css'

const Upload = () => {
  const { user, loading: authLoading, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    category_id: '',
    sub_category_id: '',
    is_public: true
  })
  const [videoFile, setVideoFile] = useState(null)
  const [thumbnailFile, setThumbnailFile] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [categories, setCategories] = useState([])
  const [subCategories, setSubCategories] = useState([])

  React.useEffect(() => {
    // Attendre que le chargement de l'authentification soit terminé
    if (authLoading) return
    
    if (!user) {
      navigate('/login')
      return
    }
    // Vérifier que l'utilisateur est admin
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchCategories()
  }, [user, authLoading, isAdmin, navigate])

  React.useEffect(() => {
    if (formData.category_id) {
      fetchSubCategories(formData.category_id)
    } else {
      setSubCategories([])
    }
  }, [formData.category_id])

  const fetchCategories = async () => {
    try {
      const response = await api.get('/categories')
      setCategories(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const fetchSubCategories = async (categoryId) => {
    try {
      const category = categories.find(c => c.id === categoryId)
      if (category && category.sub_categories) {
        setSubCategories(category.sub_categories)
      } else {
        setSubCategories([])
      }
    } catch (error) {
      console.error('Erreur:', error)
      setSubCategories([])
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!videoFile) {
      alert('Veuillez sélectionner un fichier vidéo')
      return
    }

    // Valider et nettoyer le titre
    let cleanTitle = formData.title ? formData.title.trim() : ''
    if (cleanTitle === '' || cleanTitle === 'null' || cleanTitle === 'Null' || cleanTitle === 'NULL') {
      // Utiliser le nom du fichier sans extension
      const fileName = videoFile.name.replace(/\.[^/.]+$/, '')
      cleanTitle = fileName && fileName.trim() !== '' ? fileName : 'Vidéo sans titre'
    }
    
    // Mettre à jour le titre nettoyé
    const finalFormData = { ...formData, title: cleanTitle }

    setUploading(true)
    setProgress(0)

    try {
      const uploadData = new FormData()
      uploadData.append('video', videoFile)
      uploadData.append('title', finalFormData.title)
      uploadData.append('description', finalFormData.description || '')
      if (finalFormData.category_id) {
        uploadData.append('category_id', finalFormData.category_id)
      }
      if (finalFormData.sub_category_id) {
        uploadData.append('sub_category_id', finalFormData.sub_category_id)
      }
      uploadData.append('is_public', finalFormData.is_public)

      const response = await api.post('/admin/videos/upload', uploadData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        onUploadProgress: (progressEvent) => {
          const percentCompleted = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          )
          setProgress(percentCompleted)
        },
      })

      // Upload thumbnail si fourni
      if (thumbnailFile && response.data.video_id) {
        const thumbnailData = new FormData()
        thumbnailData.append('thumbnail', thumbnailFile)
        await api.post(`/videos/${response.data.video_id}/thumbnail`, thumbnailData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        })
      }

      const videoId = response.data.video_id
      
      if (videoId) {
        alert('✅ Vidéo uploadée avec succès! Le traitement peut prendre quelques instants.')
        // Réinitialiser le formulaire
        setFormData({
          title: '',
          description: '',
          category_id: '',
          sub_category_id: '',
          is_public: true
        })
        setVideoFile(null)
        setThumbnailFile(null)
        // Rediriger directement vers la vidéo (elle sera accessible même en traitement pour le propriétaire)
        navigate(`/video/${videoId}`)
      } else {
        alert('✅ Vidéo uploadée avec succès!')
        setFormData({
          title: '',
          description: '',
          category_id: '',
          sub_category_id: '',
          is_public: true
        })
        setVideoFile(null)
        setThumbnailFile(null)
      }
    } catch (error) {
      console.error('Erreur:', error)
      const errorMessage = error.response?.data?.error || error.message || 'Erreur lors de l\'upload'
      alert(`Erreur: ${errorMessage}`)
    } finally {
      setUploading(false)
      setProgress(0)
    }
  }

  return (
    <div className="upload-page">
      <div className="upload-container">
        <h1>Uploader une vidéo</h1>
        <form onSubmit={handleSubmit} className="upload-form">
          <div className="form-group">
            <label htmlFor="video">Fichier vidéo *</label>
            <input
              type="file"
              id="video"
              accept="video/*"
              onChange={(e) => {
                const file = e.target.files[0]
                if (file) {
                  setVideoFile(file)
                  // Extraire le nom du fichier sans extension pour le titre
                  const fileName = file.name.replace(/\.[^/.]+$/, '')
                  // Nettoyer le titre pour éviter "null"
                  const cleanTitle = fileName && fileName.trim() !== '' && fileName !== 'null' 
                    ? fileName 
                    : 'Vidéo sans titre'
                  
                  // Auto-remplir le titre si vide
                  if (!formData.title || formData.title.trim() === '' || formData.title === 'null') {
                    setFormData({ ...formData, title: cleanTitle })
                  }
                }
              }}
              required
              disabled={uploading}
            />
            {videoFile && (
              <span className="file-info">
                📹 {videoFile.name} ({(videoFile.size / 1024 / 1024).toFixed(2)} MB)
              </span>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="thumbnail">Miniature (optionnel)</label>
            <input
              type="file"
              id="thumbnail"
              accept="image/*"
              onChange={(e) => setThumbnailFile(e.target.files[0])}
              disabled={uploading}
            />
          </div>

          <div className="form-group">
            <label htmlFor="title">Titre *</label>
            <input
              type="text"
              id="title"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              required
              disabled={uploading}
            />
          </div>

          <div className="form-group">
            <label htmlFor="description">Description</label>
            <textarea
              id="description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              disabled={uploading}
              rows="4"
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label htmlFor="category">Catégorie</label>
              <select
                id="category"
                value={formData.category_id}
                onChange={(e) => setFormData({ ...formData, category_id: e.target.value, sub_category_id: '' })}
                disabled={uploading}
              >
                <option value="">Aucune</option>
                {categories.map((cat) => (
                  <option key={cat.id} value={cat.id}>
                    {cat.icon} {cat.name}
                  </option>
                ))}
              </select>
            </div>

            {subCategories.length > 0 && (
              <div className="form-group">
                <label htmlFor="subcategory">Sous-catégorie</label>
                <select
                  id="subcategory"
                  value={formData.sub_category_id}
                  onChange={(e) => setFormData({ ...formData, sub_category_id: e.target.value })}
                  disabled={uploading}
                >
                  <option value="">Aucune</option>
                  {subCategories.map((sub) => (
                    <option key={sub.id} value={sub.id}>
                      {sub.name}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          <div className="form-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={formData.is_public}
                onChange={(e) => setFormData({ ...formData, is_public: e.target.checked })}
                disabled={uploading}
              />
              <span className="checkbox-text">Vidéo publique</span>
            </label>
          </div>

          {uploading && (
            <div className="upload-progress">
              <div className="progress-bar">
                <div className="progress-fill" style={{ width: `${progress}%` }} />
              </div>
              <span>{progress}%</span>
            </div>
          )}

          <div className="form-actions">
            <button 
              type="button" 
              onClick={() => navigate(-1)} 
              className="cancel-button"
              disabled={uploading}
            >
              ← Retour
            </button>
            <button type="submit" disabled={uploading} className="submit-button">
              {uploading ? 'Upload en cours...' : 'Uploader'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default Upload

