import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import { splitFileIntoChunks, calculateChunksHash, uploadChunksParallel } from '../../utils/chunkUpload'
import './Upload.css'

// Fonction pour décoder les entités HTML
const decodeHtmlEntities = (text) => {
  if (!text || typeof text !== 'string') return text

  const textarea = document.createElement('textarea')
  textarea.innerHTML = text
  return textarea.value
}

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
  const [uploadPaused, setUploadPaused] = useState(false)
  const [uploadCancelled, setUploadCancelled] = useState(false)
  const [uploadSessionId, setUploadSessionId] = useState(null)
  const [progress, setProgress] = useState(0)
  const [uploadStage, setUploadStage] = useState('') // Étape actuelle de l'upload
  const [uploadDetails, setUploadDetails] = useState('') // Détails supplémentaires
  const [message, setMessage] = useState({ type: '', text: '' }) // Message de succès/erreur
  const [categories, setCategories] = useState([])
  const [subCategories, setSubCategories] = useState([])
  const [extractingMetadata, setExtractingMetadata] = useState(false)
  const [searchingMetadata, setSearchingMetadata] = useState(false)
  const [metadataResults, setMetadataResults] = useState([])
  const [showMetadataResults, setShowMetadataResults] = useState(false)
  const [selectedMetadata, setSelectedMetadata] = useState(null)

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

  // Rechercher les métadonnées basées sur le nom de fichier (sans upload)
  const searchMetadataFromFilename = async (filename) => {
    if (!filename) return

    setSearchingMetadata(true)
    setMetadataResults([])
    setShowMetadataResults(false)

    try {
      console.log(`🔍 Recherche automatique de métadonnées pour: "${filename}"`)
      const response = await api.post('/admin/videos/search-metadata', {
        filename: filename
      })

      const results = response.data.results || []

      if (results.length > 0) {
        setMetadataResults(results)
        setShowMetadataResults(true)
        console.log(`✅ ${results.length} résultat(s) trouvé(s) pour "${filename}"`)

        // Si un seul résultat et que le titre correspond exactement, sélectionner automatiquement
        if (results.length === 1) {
          const cleanFilename = filename.replace(/\.[^/.]+$/, '').replace(/[_-]/g, ' ').toLowerCase()
          const resultTitle = results[0].title.toLowerCase().replace(/[^a-z0-9]/g, ' ')
          if (cleanFilename.includes(resultTitle.substring(0, 10)) || resultTitle.includes(cleanFilename.substring(0, 10))) {
            console.log('✅ Correspondance exacte détectée, sélection automatique...')
            setTimeout(() => {
              selectMetadata(results[0])
            }, 500)
          }
        }
      } else {
        console.log(`ℹ️ Aucun résultat trouvé pour "${filename}"`)
        // Ne pas essayer l'extraction classique automatiquement car elle nécessite un upload complet
        // L'utilisateur peut continuer sans métadonnées ou uploader le fichier pour extraction
      }
    } catch (error) {
      console.error('Erreur lors de la recherche de métadonnées:', error)
      // Ne pas essayer l'extraction classique automatiquement en cas d'erreur
      // L'utilisateur peut continuer sans métadonnées
    } finally {
      setSearchingMetadata(false)
    }
  }

  // Mapper les genres aux catégories
  const mapGenreToCategory = (genres) => {
    if (!genres || genres.length === 0) return null

    const genreMap = {
      'Action': ['Action', 'Aventure'],
      'Aventure': ['Action', 'Aventure'],
      'Animation': ['Animation'],
      'Comédie': ['Comédie', 'Humour'],
      'Crime': ['Thriller', 'Policier'],
      'Documentaire': ['Documentaire'],
      'Drame': ['Drame'],
      'Famille': ['Famille'],
      'Fantastique': ['Fantastique', 'Science-Fiction'],
      'Horreur': ['Horreur', 'Thriller'],
      'Musique': ['Musique'],
      'Mystère': ['Thriller', 'Mystère'],
      'Romance': ['Romance', 'Drame'],
      'Science-Fiction': ['Science-Fiction', 'Fantastique'],
      'Thriller': ['Thriller'],
      'Guerre': ['Action', 'Guerre'],
      'Western': ['Western'],
    }

    // Chercher la première correspondance
    for (const genre of genres) {
      const genreLower = genre.toLowerCase()
      for (const [key, categoryNames] of Object.entries(genreMap)) {
        if (genreLower.includes(key.toLowerCase()) || key.toLowerCase().includes(genreLower)) {
          // Chercher la catégorie correspondante
          const category = categories.find(cat =>
            categoryNames.some(name =>
              cat.name.toLowerCase().includes(name.toLowerCase()) ||
              name.toLowerCase().includes(cat.name.toLowerCase())
            )
          )
          if (category) {
            return category.id
          }
        }
      }
    }

    return null
  }

  // Sélectionner une métadonnée et pré-remplir le formulaire
  const selectMetadata = (metadata) => {
    setSelectedMetadata(metadata)
    setShowMetadataResults(false)

    const newFormData = { ...formData }

    // Titre (toujours remplacer si vide ou si c'est juste le nom de fichier)
    if (metadata.title) {
      const currentTitle = newFormData.title || ''
      const fileName = videoFile ? videoFile.name.replace(/\.[^/.]+$/, '') : ''
      // Remplacer si le titre actuel est vide, identique au nom de fichier, ou contient des underscores/hyphens (probablement le nom de fichier brut)
      const isLikelyFilename = currentTitle && (
        currentTitle === fileName ||
        currentTitle.includes('_') ||
        currentTitle.includes('-') ||
        currentTitle.toUpperCase() === currentTitle
      )
      if (!currentTitle || currentTitle.trim() === '' || isLikelyFilename) {
        newFormData.title = decodeHtmlEntities(metadata.title)
        console.log('✅ Titre mis à jour:', newFormData.title)
      }
    }

    // Description/Synopsis (construire une description complète)
    let descriptionParts = []

    // Synopsis principal (décoder les entités HTML)
    if (metadata.synopsis) {
      descriptionParts.push(decodeHtmlEntities(metadata.synopsis))
    } else if (metadata.description) {
      descriptionParts.push(decodeHtmlEntities(metadata.description))
    }

    // Ajouter les informations supplémentaires (décoder les entités HTML)
    let additionalInfo = []
    if (metadata.director) {
      additionalInfo.push(`Réalisateur: ${decodeHtmlEntities(metadata.director)}`)
    }
    if (metadata.actors && metadata.actors.length > 0) {
      const decodedActors = metadata.actors.map(actor => decodeHtmlEntities(actor))
      additionalInfo.push(`Acteurs: ${decodedActors.join(', ')}`)
    }
    if (metadata.release_date) {
      const year = metadata.release_date.split('-')[0]
      if (year) {
        additionalInfo.push(`Année: ${year}`)
      }
    }
    if (metadata.genre && metadata.genre.length > 0) {
      additionalInfo.push(`Genre: ${metadata.genre.join(', ')}`)
    }
    if (metadata.rating && metadata.rating > 0) {
      additionalInfo.push(`Note: ${metadata.rating.toFixed(1)}/10`)
    }

    // Construire la description complète
    if (descriptionParts.length > 0 || additionalInfo.length > 0) {
      let fullDescription = descriptionParts.join('\n\n')
      if (additionalInfo.length > 0) {
        if (fullDescription) {
          fullDescription += '\n\n' + additionalInfo.join(' | ')
        } else {
          fullDescription = additionalInfo.join(' | ')
        }
      }

      // Remplacer la description si elle est vide ou très courte
      const currentDesc = newFormData.description || ''
      if (!currentDesc || currentDesc.trim().length < 20) {
        newFormData.description = fullDescription
      } else {
        // Ajouter les infos supplémentaires à la description existante
        newFormData.description = currentDesc + '\n\n' + additionalInfo.join(' | ')
      }
    }

    // Sélectionner automatiquement la catégorie basée sur le genre
    if (metadata.genre && metadata.genre.length > 0 && categories.length > 0) {
      const categoryId = mapGenreToCategory(metadata.genre)
      if (categoryId && !newFormData.category_id) {
        newFormData.category_id = categoryId
        console.log('✅ Catégorie sélectionnée automatiquement:', categoryId)
      }
    }

    setFormData(newFormData)

    // Générer une miniature depuis le poster si disponible
    if (metadata.poster_url) {
      setThumbnailPreview(metadata.poster_url)
    } else if (metadata.title) {
      generateThumbnailFromTitle(metadata.title)
    }

    console.log('✅ Métadonnées sélectionnées et appliquées:', metadata)
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
        newFormData.title = decodeHtmlEntities(metadata.title.trim())
      }

      if (metadata.description && metadata.description.trim() !== '' && (!newFormData.description || newFormData.description.trim() === '')) {
        newFormData.description = decodeHtmlEntities(metadata.description.trim())
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

  // Ref pour le contrôle de l'upload (accessible dans les callbacks)
  const uploadControlRef = React.useRef({ paused: false, cancelled: false })

  // Fonctions de contrôle de l'upload
  const pauseUpload = () => {
    setUploadPaused(true)
    uploadControlRef.current.paused = true
    setUploadStage('⏸️ En pause...')
    setMessage({ type: 'info', text: 'Upload mis en pause. Cliquez sur "Reprendre" pour continuer.' })
  }

  const resumeUpload = () => {
    setUploadPaused(false)
    uploadControlRef.current.paused = false
    setUploadStage('Upload en cours...')
    setMessage({ type: '', text: '' })
  }

  const cancelUpload = async () => {
    // Afficher un message de confirmation dans l'interface au lieu d'une popup
    setMessage({
      type: 'warning',
      text: '⚠️ Voulez-vous vraiment annuler l\'upload ? La vidéo partielle sera supprimée. Cliquez à nouveau sur "Annuler" pour confirmer.'
    })

    // Si l'utilisateur clique une deuxième fois rapidement, annuler vraiment
    if (uploadControlRef.current.cancelling) {
      setUploadCancelled(true)
      uploadControlRef.current.cancelled = true
      uploadControlRef.current.cancelling = false
      setUploadPaused(false)
      setUploadStage('Annulation...')
      setMessage({ type: 'warning', text: 'Annulation de l\'upload en cours...' })

      // Si une session existe, on peut essayer de l'annuler côté serveur
      if (uploadSessionId) {
        try {
          await api.post(`/admin/uploads/${uploadSessionId}/cancel`)
        } catch (error) {
          console.warn('Impossible d\'annuler la session côté serveur:', error)
        }
      }

      // Réinitialiser après un court délai
      setTimeout(() => {
        setUploading(false)
        setUploadPaused(false)
        setUploadCancelled(false)
        uploadControlRef.current.paused = false
        uploadControlRef.current.cancelled = false
        uploadControlRef.current.cancelling = false
        setProgress(0)
        setUploadStage('')
        setUploadDetails('')
        setUploadSessionId(null)
        setMessage({ type: 'error', text: 'Upload annulé. Vous pouvez recommencer.' })
        setTimeout(() => setMessage({ type: '', text: '' }), 5000)
      }, 1000)
    } else {
      // Première fois : marquer comme "en cours d'annulation"
      uploadControlRef.current.cancelling = true
      setTimeout(() => {
        uploadControlRef.current.cancelling = false
        setMessage({ type: '', text: '' })
      }, 3000)
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!videoFile) {
      setMessage({ type: 'error', text: 'Veuillez sélectionner un fichier vidéo' })
      setTimeout(() => setMessage({ type: '', text: '' }), 5000)
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
      // Étape 1: Initialiser la session d'upload avec métadonnées enrichies
      const metadata = {
        title: finalFormData.title,
        description: finalFormData.description || '',
        category_id: finalFormData.category_id || null,
        sub_category_id: finalFormData.sub_category_id || null,
        is_public: finalFormData.is_public,
        // Métadonnées enrichies si sélectionnées
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
          backdrop_url: selectedMetadata.backdrop_url || '',
          trailer_url: selectedMetadata.trailer_url || '',
          runtime: selectedMetadata.runtime || null,
          writer: selectedMetadata.writer || '',
          platform: selectedMetadata.platform || '',
        }),
      }

      console.log('📤 Initialisation de la session d\'upload...')
      const initResponse = await api.post('/admin/uploads/init', {
        file_name: videoFile.name,
        file_size: videoFile.size,
        metadata: metadata,
      })

      const sessionId = initResponse.data.session_id
      console.log('✅ Session d\'upload initialisée:', sessionId)
      setUploadSessionId(sessionId)
      setProgress(5)

      // Étape 2: Diviser le fichier en chunks
      setUploadStage('Préparation du fichier...')
      setUploadDetails('Division en chunks')
      const chunks = splitFileIntoChunks(videoFile)
      console.log(`📦 Fichier divisé en ${chunks.length} chunks`)
      setProgress(7)

      // Étape 3: Calculer les hash de tous les chunks
      setUploadStage('Calcul des vérifications...')
      setUploadDetails('Calcul des hash SHA256')
      const chunksWithHash = await calculateChunksHash(chunks)
      console.log('✅ Hash calculés pour tous les chunks')
      setProgress(10)

      // Étape 4: Uploader les chunks en parallèle avec progression
      setUploadStage('Upload en cours...')
      let lastLoggedProgress = 0

      // Variables pour l'estimation du temps
      const uploadStartTime = Date.now()
      const speedHistory = [] // Historique des vitesses (MB/s)
      const MAX_HISTORY = 10 // Garder les 10 dernières mesures

      await uploadChunksParallel(sessionId, chunksWithHash, api, (progressData) => {
        // Progression de l'upload: 10% à 90% (10% pour init + hash, 80% pour upload, 10% pour finalisation)
        const uploadProgress = 10 + (progressData.progress * 0.8)
        const roundedProgress = Math.round(uploadProgress)
        setProgress(roundedProgress)

        // Calculer la vitesse actuelle et estimer le temps restant
        const elapsedTime = (Date.now() - uploadStartTime) / 1000 // en secondes
        const uploadedMB = progressData.uploadedBytes / (1024 * 1024)
        const totalMB = progressData.totalBytes / (1024 * 1024)
        const remainingMB = totalMB - uploadedMB

        if (elapsedTime > 0 && uploadedMB > 0) {
          const currentSpeed = uploadedMB / elapsedTime // MB/s
          speedHistory.push(currentSpeed)

          // Garder seulement les dernières mesures
          if (speedHistory.length > MAX_HISTORY) {
            speedHistory.shift()
          }

          // Calculer la vitesse moyenne sur les dernières mesures
          const avgSpeed = speedHistory.reduce((a, b) => a + b, 0) / speedHistory.length

          // Estimer le temps restant
          let timeRemaining = 0
          let timeRemainingStr = ''

          if (avgSpeed > 0 && remainingMB > 0) {
            timeRemaining = remainingMB / avgSpeed // en secondes

            if (timeRemaining < 60) {
              timeRemainingStr = `${Math.ceil(timeRemaining)}s`
            } else if (timeRemaining < 3600) {
              const minutes = Math.floor(timeRemaining / 60)
              const seconds = Math.ceil(timeRemaining % 60)
              timeRemainingStr = `${minutes}m ${seconds}s`
            } else {
              const hours = Math.floor(timeRemaining / 3600)
              const minutes = Math.floor((timeRemaining % 3600) / 60)
              timeRemainingStr = `${hours}h ${minutes}m`
            }
          }

          // Logger seulement tous les 5% ou tous les 50 chunks pour éviter le spam
          const shouldLog = (roundedProgress - lastLoggedProgress >= 5) ||
            (progressData.uploadedChunks % 50 === 0 && progressData.uploadedChunks > 0)

          if (shouldLog) {
            const uploadedMBStr = uploadedMB.toFixed(1)
            const totalMBStr = totalMB.toFixed(1)
          const speedStr = avgSpeed > 0 ? avgSpeed.toFixed(2) : '0.00'
          console.log(`📤 Upload: ${roundedProgress}% (${progressData.uploadedChunks}/${progressData.totalChunks} chunks, ${uploadedMBStr}/${totalMBStr} MB, ${speedStr} MB/s, ${timeRemainingStr} restant)`)
          lastLoggedProgress = roundedProgress
        }

        // Mettre à jour les détails avec estimation du temps
        const uploadedMBStr = uploadedMB.toFixed(1)
        const totalMBStr = totalMB.toFixed(1)
        const speedStr = avgSpeed > 0 ? avgSpeed.toFixed(2) : '0.00'
        let detailsStr = `${progressData.uploadedChunks}/${progressData.totalChunks} chunks • ${uploadedMBStr}/${totalMBStr} MB`
        if (timeRemainingStr) {
          detailsStr += ` • ${speedStr} MB/s • ~${timeRemainingStr} restant`
        }
        setUploadDetails(detailsStr)
      }

        // Mettre à jour le contrôle avec les valeurs actuelles
        uploadControlRef.current.paused = uploadPaused
        uploadControlRef.current.cancelled = uploadCancelled
      }, uploadControlRef.current)

      setProgress(90)
      setUploadStage('Finalisation de l\'upload...')
      setUploadDetails('Assemblage des chunks')

      // Étape 5: Finaliser l'upload
      const finalizeResponse = await api.post(`/admin/uploads/${sessionId}/finalize`)
      console.log('✅ Upload finalisé:', finalizeResponse.data)
      setProgress(95)
      setUploadStage('Upload terminé!')
      setUploadDetails('Traitement de la vidéo en cours...')

      // Étape 6: Uploader la miniature si fournie
      if (thumbnailFile && finalizeResponse.data.video_id) {
        setUploadStage('Upload de la miniature...')
        setUploadDetails('')
        const thumbnailData = new FormData()
        thumbnailData.append('thumbnail', thumbnailFile)
        await api.post(`/videos/${finalizeResponse.data.video_id}/thumbnail`, thumbnailData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        })
        console.log('✅ Miniature uploadée')
      }

      setProgress(100)
      setUploadStage('Terminé!')
      setUploadDetails('')
      const videoId = finalizeResponse.data.video_id

      if (videoId) {
        setMessage({ type: 'success', text: '✅ Vidéo uploadée avec succès! Le traitement peut prendre quelques instants. Redirection...' })
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
        setUploadSessionId(null)
        // Rediriger directement vers la vidéo après 2 secondes
        setTimeout(() => {
          navigate(`/video/${videoId}`)
        }, 2000)
      } else {
        setMessage({ type: 'success', text: '✅ Vidéo uploadée avec succès!' })
        setFormData({
          title: '',
          description: '',
          category_id: '',
          sub_category_id: '',
          is_public: true
        })
        setVideoFile(null)
        setThumbnailFile(null)
        setUploadSessionId(null)
        setTimeout(() => setMessage({ type: '', text: '' }), 5000)
      }
    } catch (error) {
      console.error('❌ Erreur lors de l\'upload:', error)

      // Si l'upload a été annulé, ne pas afficher d'erreur
      if (error.message === 'Upload annulé par l\'utilisateur' || uploadCancelled) {
        return
      }

      const errorData = error.response?.data
      let errorMessage = errorData?.error || error.message || 'Erreur lors de l\'upload'

      // Messages d'erreur plus détaillés
      if (error.response?.status === 400) {
        if (errorData?.uploaded_chunks !== undefined && errorData?.total_chunks !== undefined) {
          errorMessage = `Upload incomplet: ${errorData.uploaded_chunks}/${errorData.total_chunks} chunks uploadés. Veuillez réessayer.`
        } else if (errorMessage.includes('chunks')) {
          errorMessage = `Erreur lors de la finalisation: ${errorMessage}. Veuillez réessayer.`
        }
      }

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
                  setSelectedMetadata(null)
                  setMetadataResults([])
                  setShowMetadataResults(false)

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

                  // Rechercher automatiquement les métadonnées basées sur le nom de fichier
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
                {!searchingMetadata && !selectedMetadata && metadataResults.length === 0 && (
                  <button
                    type="button"
                    className="search-metadata-btn"
                    onClick={() => searchMetadataFromFilename(videoFile.name)}
                  >
                    🔍 Rechercher les métadonnées
                  </button>
                )}
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
            {searchingMetadata && (
              <div className="metadata-searching">
                <div className="searching-spinner"></div>
                <p className="metadata-extracting">
                  🔍 Recherche automatique de métadonnées en cours...
                  <br />
                  <span style={{ fontSize: '12px', opacity: 0.7 }}>
                    Recherche sur AlloCiné, IMDb...
                  </span>
                </p>
              </div>
            )}
            {extractingMetadata && (
              <p className="metadata-extracting">🔍 Extraction des métadonnées du fichier en cours...</p>
            )}

            {/* Résultats de recherche de métadonnées */}
            {showMetadataResults && metadataResults.length > 0 && (
              <div className="metadata-results">
                <h3>🎬 Résultats de recherche ({metadataResults.length})</h3>
                <p className="metadata-results-info">
                  Sélectionnez le film correspondant à votre vidéo :
                </p>
                <div className="metadata-results-grid">
                  {metadataResults.map((result, index) => (
                    <div
                      key={index}
                      className={`metadata-result-card ${selectedMetadata?.tmdb_id === result.tmdb_id || selectedMetadata?.imdb_id === result.imdb_id ? 'selected' : ''}`}
                      onClick={() => selectMetadata(result)}
                    >
                      <div className="metadata-poster-container">
                        {result.poster_url ? (
                          <img
                            src={result.poster_url}
                            alt={result.title}
                            className="metadata-poster"
                            onError={(e) => {
                              e.target.src = 'https://via.placeholder.com/200x300?text=No+Poster'
                            }}
                          />
                        ) : (
                          <div className="metadata-poster-placeholder">
                            <span>🎬</span>
                            <p>Aucune affiche</p>
                          </div>
                        )}
                        {result.rating > 0 && (
                          <div className="metadata-rating-badge">
                            ⭐ {result.rating.toFixed(1)}
                          </div>
                        )}
                      </div>
                      <div className="metadata-result-info">
                        <h4>{decodeHtmlEntities(result.title)}</h4>
                        {result.original_title && result.original_title !== result.title && (
                          <p className="original-title">{decodeHtmlEntities(result.original_title)}</p>
                        )}
                        <div className="metadata-details">
                          {result.release_date && (
                            <p className="release-date">📅 {result.release_date}</p>
                          )}
                          {result.production_year && (
                            <p className="production-year">📆 {result.production_year}</p>
                          )}
                          {result.runtime && result.runtime > 0 && (
                            <p className="runtime">⏱️ {Math.floor(result.runtime / 60)}h {result.runtime % 60}min</p>
                          )}
                          {result.platform && (
                            <p className="platform">📺 {result.platform}</p>
                          )}
                        </div>
                        {result.director && (
                          <p className="director">🎬 Réalisateur: <strong>{decodeHtmlEntities(result.director)}</strong></p>
                        )}
                        {result.writer && (
                          <p className="writer">✍️ Scénariste: <strong>{decodeHtmlEntities(result.writer)}</strong></p>
                        )}
                        {result.actors && result.actors.length > 0 && (
                          <p className="actors">👥 Acteurs: <strong>
                            {Array.isArray(result.actors)
                              ? result.actors.slice(0, 3).map(actor => decodeHtmlEntities(actor)).join(', ')
                              : decodeHtmlEntities(result.actors)
                            }
                          </strong></p>
                        )}
                        {result.genre && result.genre.length > 0 && (
                          <p className="genre">🏷️ {Array.isArray(result.genre) ? result.genre.join(', ') : result.genre}</p>
                        )}
                        {result.synopsis && (
                          <p className="synopsis" title={decodeHtmlEntities(result.synopsis)}>
                            {(() => {
                              const decoded = decodeHtmlEntities(result.synopsis)
                              return decoded.length > 120 ? decoded.substring(0, 120) + '...' : decoded
                            })()}
                          </p>
                        )}
                        {result.imdb_id && (
                          <p className="imdb-link">
                            <a
                              href={`https://www.imdb.com/title/${result.imdb_id}`}
                              target="_blank"
                              rel="noopener noreferrer"
                              onClick={(e) => e.stopPropagation()}
                            >
                              🔗 Voir sur IMDb
                            </a>
                          </p>
                        )}
                        {result.trailer_url && (
                          <p className="trailer-link">
                            <a
                              href={result.trailer_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              onClick={(e) => e.stopPropagation()}
                            >
                              ▶️ Voir la bande-annonce
                            </a>
                          </p>
                        )}
                        <button
                          type="button"
                          className="select-metadata-btn"
                          onClick={(e) => {
                            e.stopPropagation()
                            selectMetadata(result)
                          }}
                        >
                          ✓ Sélectionner
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
                <button
                  type="button"
                  className="skip-metadata-btn"
                  onClick={() => {
                    setShowMetadataResults(false)
                    // L'utilisateur peut continuer sans métadonnées
                    // L'extraction classique nécessite un upload complet et peut être lente
                  }}
                >
                  Ignorer et continuer sans métadonnées
                </button>

                {videoFile && !selectedMetadata && (
                  <button
                    type="button"
                    className="extract-metadata-btn"
                    onClick={async () => {
                      setShowMetadataResults(false)
                      // Optionnel: essayer l'extraction classique (peut être lente pour gros fichiers)
                      await extractVideoMetadata(videoFile)
                    }}
                  >
                    🔍 Extraire les métadonnées du fichier (peut être lent)
                  </button>
                )}
              </div>
            )}

            {selectedMetadata && (
              <div className="selected-metadata-info">
                <p>✅ Métadonnées sélectionnées: <strong>{selectedMetadata.title}</strong></p>
                <button
                  type="button"
                  className="change-metadata-btn"
                  onClick={() => {
                    setSelectedMetadata(null)
                    setShowMetadataResults(true)
                  }}
                >
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
            <label htmlFor="is_public" className="form-label">Visibilité</label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                id="is_public"
                checked={formData.is_public}
                onChange={(e) => setFormData({ ...formData, is_public: e.target.checked })}
                disabled={uploading}
              />
              <span className="checkbox-text">🌐 Vidéo publique</span>
            </label>
            <p className="form-hint">Les vidéos publiques sont visibles par tous les utilisateurs</p>
          </div>

          {/* Messages de succès/erreur */}
          {message.text && (
            <div className={`upload-message ${message.type}`}>
              <span>{message.text}</span>
              <button
                type="button"
                className="message-close"
                onClick={() => setMessage({ type: '', text: '' })}
              >
                ×
              </button>
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
              {uploadDetails && (
                <div className="progress-details">{uploadDetails}</div>
              )}
              <div className="upload-controls">
                {!uploadPaused ? (
                  <button
                    type="button"
                    className="pause-upload-btn"
                    onClick={pauseUpload}
                    disabled={uploadCancelled}
                  >
                    ⏸️ Pause
                  </button>
                ) : (
                  <button
                    type="button"
                    className="resume-upload-btn"
                    onClick={resumeUpload}
                  >
                    ▶️ Reprendre
                  </button>
                )}
                <button
                  type="button"
                  className="cancel-upload-btn"
                  onClick={cancelUpload}
                  disabled={uploadCancelled}
                >
                  ❌ Annuler
                </button>
              </div>
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

