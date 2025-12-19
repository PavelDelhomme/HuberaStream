'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import { splitFileIntoChunks, calculateChunksHash, uploadChunksParallel } from '@/lib/chunkUpload'
import './Upload.css'

const decodeHtmlEntities = (text: string) => {
  if (!text || typeof text !== 'string') return text
  const textarea = document.createElement('textarea')
  textarea.innerHTML = text
  return textarea.value
}

export default function UploadPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    category_id: '',
    sub_category_id: '',
    is_public: true
  })
  const [videoFile, setVideoFile] = useState<File | null>(null)
  const [thumbnailFile, setThumbnailFile] = useState<File | null>(null)
  const [thumbnailPreview, setThumbnailPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadPaused, setUploadPaused] = useState(false)
  const [uploadCancelled, setUploadCancelled] = useState(false)
  const [uploadSessionId, setUploadSessionId] = useState<string | null>(null)
  const [progress, setProgress] = useState(0)
  const [uploadStage, setUploadStage] = useState('')
  const [uploadDetails, setUploadDetails] = useState('')
  const [message, setMessage] = useState({ type: '', text: '' })
  const [categories, setCategories] = useState<any[]>([])
  const [subCategories, setSubCategories] = useState<any[]>([])
  const [searchingMetadata, setSearchingMetadata] = useState(false)
  const [metadataResults, setMetadataResults] = useState<any[]>([])
  const [showMetadataResults, setShowMetadataResults] = useState(false)
  const [selectedMetadata, setSelectedMetadata] = useState<any>(null)
  const uploadControlRef = useRef({ paused: false, cancelled: false, cancelling: false })

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/video/upload')
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

  useEffect(() => {
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

  const fetchSubCategories = async (categoryId: string) => {
    try {
      const category = categories.find((c: any) => c.id === categoryId)
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

  const searchMetadataFromFilename = async (filename: string) => {
    if (!filename) return

    setSearchingMetadata(true)
    setMetadataResults([])
    setShowMetadataResults(false)

    try {
      const response = await api.post('/admin/videos/search-metadata', {
        filename: filename
      })

      const results = response.data.results || []
      if (results.length > 0) {
        setMetadataResults(results)
        setShowMetadataResults(true)
      }
    } catch (error) {
      console.error('Erreur lors de la recherche de métadonnées:', error)
    } finally {
      setSearchingMetadata(false)
    }
  }

  const selectMetadata = (metadata: any) => {
    setSelectedMetadata(metadata)
    setShowMetadataResults(false)

    if (metadata.title) {
      setFormData({ ...formData, title: decodeHtmlEntities(metadata.title) })
    }

    if (metadata.synopsis || metadata.description) {
      setFormData({
        ...formData,
        description: decodeHtmlEntities(metadata.synopsis || metadata.description || '')
      })
    }

    if (metadata.poster_url) {
      setThumbnailPreview(metadata.poster_url)
    }
  }

  const pauseUpload = () => {
    setUploadPaused(true)
    uploadControlRef.current.paused = true
    setUploadStage('⏸️ En pause...')
  }

  const resumeUpload = () => {
    setUploadPaused(false)
    uploadControlRef.current.paused = false
    setUploadStage('Upload en cours...')
  }

  const cancelUpload = async () => {
    if (uploadControlRef.current.cancelling) {
      setUploadCancelled(true)
      uploadControlRef.current.cancelled = true
      if (uploadSessionId) {
        try {
          await api.post(`/admin/uploads/${uploadSessionId}/cancel`)
        } catch (error) {
          console.warn('Impossible d\'annuler la session:', error)
        }
      }
      setTimeout(() => {
        setUploading(false)
        setProgress(0)
        setUploadStage('')
        setUploadDetails('')
        setUploadSessionId(null)
      }, 1000)
    } else {
      uploadControlRef.current.cancelling = true
      setTimeout(() => {
        uploadControlRef.current.cancelling = false
      }, 3000)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!videoFile) {
      setMessage({ type: 'error', text: 'Veuillez sélectionner un fichier vidéo' })
      return
    }

    let cleanTitle = formData.title ? formData.title.trim() : ''
    if (cleanTitle === '' || cleanTitle === 'null') {
      const fileName = videoFile.name.replace(/\.[^/.]+$/, '')
      cleanTitle = fileName && fileName.trim() !== '' ? fileName : 'Vidéo sans titre'
    }

    const finalFormData = { ...formData, title: cleanTitle }

    setUploading(true)
    setUploadPaused(false)
    setUploadCancelled(false)
    uploadControlRef.current.paused = false
    uploadControlRef.current.cancelled = false
    uploadControlRef.current.cancelling = false
    setProgress(0)
    setUploadStage('Initialisation de la session...')
    setUploadDetails('')
    setMessage({ type: '', text: '' })

    try {
      const metadata = {
        title: finalFormData.title,
        description: finalFormData.description || '',
        category_id: finalFormData.category_id || null,
        sub_category_id: finalFormData.sub_category_id || null,
        is_public: finalFormData.is_public,
        ...(selectedMetadata && {
          director: selectedMetadata.director || '',
          actors: selectedMetadata.actors ? selectedMetadata.actors.join(', ') : '',
          release_date: selectedMetadata.release_date || '',
          production_year: selectedMetadata.production_year || null,
          synopsis: selectedMetadata.synopsis || '',
          genre: selectedMetadata.genre ? selectedMetadata.genre.join(', ') : '',
          rating: selectedMetadata.rating || null,
          imdb_id: selectedMetadata.imdb_id || '',
          tmdb_id: selectedMetadata.tmdb_id || null,
          poster_url: selectedMetadata.poster_url || '',
        }),
      }

      const initResponse = await api.post('/admin/uploads/init', {
        file_name: videoFile.name,
        file_size: videoFile.size,
        metadata: metadata,
      })

      const sessionId = initResponse.data.session_id
      setUploadSessionId(sessionId)
      setProgress(5)

      setUploadStage('Préparation du fichier...')
      const chunks = splitFileIntoChunks(videoFile)
      setProgress(7)

      setUploadStage('Calcul des vérifications...')
      const chunksWithHash = await calculateChunksHash(chunks)
      setProgress(10)

      setUploadStage('Upload en cours...')
      await uploadChunksParallel(sessionId, chunksWithHash, api, (progressData) => {
        const uploadProgress = 10 + (progressData.progress * 0.8)
        setProgress(Math.round(uploadProgress))
        setUploadDetails(`${progressData.uploadedChunks}/${progressData.totalChunks} chunks`)
        uploadControlRef.current.paused = uploadPaused
        uploadControlRef.current.cancelled = uploadCancelled
      }, uploadControlRef.current)

      setProgress(90)
      setUploadStage('Finalisation de l\'upload...')

      const finalizeResponse = await api.post(`/admin/uploads/${sessionId}/finalize`)
      setProgress(95)

      if (thumbnailFile && finalizeResponse.data.video_id) {
        setUploadStage('Upload de la miniature...')
        const thumbnailData = new FormData()
        thumbnailData.append('thumbnail', thumbnailFile)
        await api.post(`/videos/${finalizeResponse.data.video_id}/thumbnail`, thumbnailData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        })
      }

      setProgress(100)
      setUploadStage('Terminé!')
      const videoId = finalizeResponse.data.video_id

      if (videoId) {
        setMessage({ type: 'success', text: '✅ Vidéo uploadée avec succès! Redirection...' })
        setTimeout(() => {
          router.push(`/video/${videoId}`)
        }, 2000)
      }
    } catch (error: any) {
      if (error.message === 'Upload annulé par l\'utilisateur' || uploadCancelled) {
        return
      }
      const errorMessage = error.response?.data?.error || error.message || 'Erreur lors de l\'upload'
      setMessage({ type: 'error', text: `❌ Erreur: ${errorMessage}` })
    } finally {
      if (!uploadCancelled) {
        setUploading(false)
        setProgress(0)
        setUploadStage('')
        setUploadDetails('')
        setUploadPaused(false)
        setUploadSessionId(null)
      }
    }
  }

  if (!user) {
    return (
      <div className="upload-page">
        <div className="loading">Chargement...</div>
      </div>
    )
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
                const file = e.target.files?.[0]
                if (file) {
                  setVideoFile(file)
                  setSelectedMetadata(null)
                  setMetadataResults([])
                  setShowMetadataResults(false)

                  const fileName = file.name.replace(/\.[^/.]+$/, '')
                  const cleanTitle = fileName && fileName.trim() !== '' && fileName !== 'null'
                    ? fileName
                    : 'Vidéo sans titre'

                  if (!formData.title || formData.title.trim() === '' || formData.title === 'null') {
                    setFormData({ ...formData, title: cleanTitle })
                  }

                  await searchMetadataFromFilename(file.name)
                }
              }}
              required
              disabled={uploading}
            />
            {videoFile && (
              <div className="file-info-container">
                <span className="file-info">
                  📹 {videoFile.name} ({(videoFile.size / 1024 / 1024).toFixed(2)} MB)
                </span>
              </div>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="thumbnail">Miniature (optionnel)</label>
            <input
              type="file"
              id="thumbnail"
              accept="image/*"
              onChange={(e) => {
                const file = e.target.files?.[0]
                setThumbnailFile(file || null)
                if (file) {
                  const reader = new FileReader()
                  reader.onloadend = () => {
                    setThumbnailPreview(reader.result as string)
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
              </div>
            )}
            {searchingMetadata && (
              <div className="metadata-searching">
                <p>🔍 Recherche automatique de métadonnées en cours...</p>
              </div>
            )}
            {showMetadataResults && metadataResults.length > 0 && (
              <div className="metadata-results">
                <h3>🎬 Résultats de recherche ({metadataResults.length})</h3>
                <div className="metadata-results-grid">
                  {metadataResults.slice(0, 5).map((result: any, index: number) => (
                    <div
                      key={index}
                      className={`metadata-result-card ${selectedMetadata?.tmdb_id === result.tmdb_id ? 'selected' : ''}`}
                      onClick={() => selectMetadata(result)}
                    >
                      {result.poster_url && (
                        <img src={result.poster_url} alt={result.title} className="metadata-poster" />
                      )}
                      <div className="metadata-result-info">
                        <h4>{decodeHtmlEntities(result.title)}</h4>
                        {result.release_date && <p>📅 {result.release_date}</p>}
                        <button type="button" onClick={() => selectMetadata(result)}>
                          ✓ Sélectionner
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {selectedMetadata && (
              <div className="selected-metadata-info">
                <p>✅ Métadonnées sélectionnées: <strong>{selectedMetadata.title}</strong></p>
                <button type="button" onClick={() => setSelectedMetadata(null)}>
                  Changer
                </button>
              </div>
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
              rows={4}
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
                {categories.map((cat: any) => (
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
                  {subCategories.map((sub: any) => (
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
              <span>🌐 Vidéo publique</span>
            </label>
          </div>

          {message.text && (
            <div className={`upload-message ${message.type}`}>
              <span>{message.text}</span>
              <button type="button" onClick={() => setMessage({ type: '', text: '' })}>×</button>
            </div>
          )}

          {uploading && (
            <div className="upload-progress">
              <div className="progress-header">
                <span className="progress-stage">{uploadStage || 'Upload en cours...'}</span>
                <span className="progress-percentage">{progress}%</span>
              </div>
              <div className="progress-bar">
                <div className="progress-fill" style={{ width: `${progress}%` }} />
              </div>
              {uploadDetails && <div className="progress-details">{uploadDetails}</div>}
              <div className="upload-controls">
                {!uploadPaused ? (
                  <button type="button" onClick={pauseUpload} disabled={uploadCancelled}>
                    ⏸️ Pause
                  </button>
                ) : (
                  <button type="button" onClick={resumeUpload}>
                    ▶️ Reprendre
                  </button>
                )}
                <button type="button" onClick={cancelUpload} disabled={uploadCancelled}>
                  ❌ Annuler
                </button>
              </div>
            </div>
          )}

          <div className="form-actions">
            <button type="button" onClick={() => router.back()} className="cancel-button" disabled={uploading}>
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

