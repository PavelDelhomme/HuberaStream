import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import { splitFileIntoChunks, calculateChunksHash, uploadChunksParallel } from '../../utils/chunkUpload'
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
  const [thumbnailPreview, setThumbnailPreview] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [categories, setCategories] = useState([])
  const [subCategories, setSubCategories] = useState([])
  const [extractingMetadata, setExtractingMetadata] = useState(false)

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

  // Générer automatiquement une miniature quand le titre change
  React.useEffect(() => {
    if (formData.title && formData.title.trim() !== '' && !videoFile) {
      generateThumbnailFromTitle(formData.title)
    }
  }, [formData.title])

  const generateThumbnailFromTitle = async (title) => {
    if (!title || title.trim() === '') return
    
    try {
      const response = await api.get('/admin/thumbnails/generate', {
        params: { title: title.trim() }
      })
      if (response.data.thumbnail_url) {
        setThumbnailPreview(response.data.thumbnail_url)
      }
    } catch (error) {
      console.error('Erreur lors de la génération de la miniature:', error)
      // Ne pas afficher d'erreur à l'utilisateur, c'est optionnel
    }
  }

  const extractVideoMetadata = async (file) => {
    if (!file) return
    
    setExtractingMetadata(true)
    try {
      const formData = new FormData()
      formData.append('video', file)
      
      const response = await api.post('/admin/videos/extract-metadata', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      })
      
      const metadata = response.data
      
      // Pré-remplir les champs avec les métadonnées extraites
      const newFormData = { ...formData }
      
      if (metadata.title && metadata.title.trim() !== '' && (!newFormData.title || newFormData.title.trim() === '')) {
        newFormData.title = metadata.title.trim()
      }
      
      if (metadata.description && metadata.description.trim() !== '' && (!newFormData.description || newFormData.description.trim() === '')) {
        newFormData.description = metadata.description.trim()
      }
      
      // Si on a un auteur, on peut l'ajouter à la description
      if (metadata.author && metadata.author.trim() !== '') {
        if (newFormData.description) {
          newFormData.description = `Par ${metadata.author}\n\n${newFormData.description}`
        } else {
          newFormData.description = `Par ${metadata.author}`
        }
      }
      
      setFormData(newFormData)
      
      // Générer aussi une miniature depuis le titre si disponible
      if (metadata.title) {
        await generateThumbnailFromTitle(metadata.title)
      }
      
    } catch (error) {
      console.error('Erreur lors de l\'extraction des métadonnées:', error)
      // Ne pas bloquer l'upload si l'extraction échoue
    } finally {
      setExtractingMetadata(false)
    }
  }

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
              onChange={async (e) => {
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
                  
                  // Extraire les métadonnées du fichier vidéo
                  await extractVideoMetadata(file)
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
              onChange={(e) => {
                const file = e.target.files[0]
                setThumbnailFile(file)
                if (file) {
                  // Afficher un aperçu de la miniature uploadée
                  const reader = new FileReader()
                  reader.onloadend = () => {
                    setThumbnailPreview(reader.result)
                  }
                  reader.readAsDataURL(file)
                } else {
                  setThumbnailPreview(null)
                }
              }}
              disabled={uploading}
            />
            {thumbnailPreview && (
              <div className="thumbnail-preview">
                <img src={thumbnailPreview} alt="Aperçu miniature" />
                <p className="thumbnail-info">✨ Miniature générée automatiquement</p>
              </div>
            )}
            {extractingMetadata && (
              <p className="metadata-extracting">🔍 Extraction des métadonnées en cours...</p>
            )}
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

