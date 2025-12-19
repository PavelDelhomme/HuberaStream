'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './DVDExtract.css'

export default function DVDExtractPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [step, setStep] = useState(1)
  const [device, setDevice] = useState<string | null>(null)
  const [availableDevices, setAvailableDevices] = useState<string[]>([])
  const [manualDevice, setManualDevice] = useState('')
  const [dvdInfo, setDvdInfo] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [subCategoryId, setSubCategoryId] = useState('')
  const [isPublic, setIsPublic] = useState(false)
  const [selectedVideoTrack, setSelectedVideoTrack] = useState(0)
  const [selectedAudioTracks, setSelectedAudioTracks] = useState<number[]>([])
  const [selectedSubtitleTracks, setSelectedSubtitleTracks] = useState<number[]>([])
  const [qualities, setQualities] = useState<string[]>(['1080p', '720p', '480p'])
  const [categories, setCategories] = useState<any[]>([])
  const [subCategories, setSubCategories] = useState<any[]>([])
  const [extracting, setExtracting] = useState(false)
  const [extractionProgress, setExtractionProgress] = useState(0)
  const [extractionStage, setExtractionStage] = useState('')
  const [extractionDetails, setExtractionDetails] = useState('')
  const [videoId, setVideoId] = useState<string | null>(null)
  const [analyzingProgress, setAnalyzingProgress] = useState(0)
  const [analyzingStage, setAnalyzingStage] = useState('')
  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const [cancelling, setCancelling] = useState(false)

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/dvd-extract')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser) {
        router.push('/login')
        return
      }

      setUser(currentUser)
      fetchCategories()
      checkForActiveExtraction()
    }
    checkAuth()
  }, [router])

  useEffect(() => {
    if (categoryId) {
      fetchSubCategories(categoryId)
    } else {
      setSubCategories([])
    }
  }, [categoryId, categories])

  useEffect(() => {
    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current)
      }
    }
  }, [])

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

  const checkForActiveExtraction = async () => {
    try {
      const response = await api.get('/videos?status=processing')
      const processingVideos = response.data.videos || []
      const dvdVideos = processingVideos.filter((video: any) => {
        const createdAt = new Date(video.created_at)
        const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000)
        return createdAt > twoHoursAgo && video.status === 'processing'
      })

      if (dvdVideos.length > 0) {
        const latestVideo = dvdVideos.sort((a: any, b: any) =>
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )[0]

        setVideoId(latestVideo.id)
        setExtracting(true)
        setStep(4)
        setExtractionProgress(5)
        setExtractionStage('Extraction en cours...')
        startPolling(latestVideo.id)
      }
    } catch (error) {
      console.error('Erreur lors de la vérification:', error)
    }
  }

  const startPolling = (videoIdToPoll: string) => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current)
    }

    const pollStatus = setInterval(async () => {
      try {
        const videoResponse = await api.get(`/videos/${videoIdToPoll}/status`)
        const videoStatus = videoResponse.data.status
        const videoProgress = videoResponse.data.progress || 0

        setExtractionProgress(videoProgress)

        if (videoStatus === 'processing') {
          if (videoProgress < 30) {
            setExtractionStage('Extraction du DVD...')
            setExtractionDetails(`Lecture des pistes (${Math.round(videoProgress)}%)`)
          } else if (videoProgress < 50) {
            setExtractionStage('Compression optimale...')
            setExtractionDetails(`Compression H.265/HEVC (${Math.round(videoProgress)}%)`)
          } else if (videoProgress < 90) {
            setExtractionStage('Transcodage multi-qualités...')
            setExtractionDetails(`Génération des formats (${Math.round(videoProgress)}%)`)
          } else {
            setExtractionStage('Finalisation...')
            setExtractionDetails(`Finalisation (${Math.round(videoProgress)}%)`)
          }
        } else if (videoStatus === 'ready') {
          clearInterval(pollStatus)
          pollIntervalRef.current = null
          setExtractionProgress(100)
          setExtractionStage('Extraction terminée !')
          setTimeout(() => {
            router.push(`/video/${videoIdToPoll}`)
          }, 2000)
        } else if (videoStatus === 'error' || videoStatus === 'cancelled') {
          clearInterval(pollStatus)
          pollIntervalRef.current = null
          setError('Erreur lors de l\'extraction')
          setExtracting(false)
          setStep(3)
        }
      } catch (err) {
        console.error('Erreur lors du polling:', err)
      }
    }, 2000)

    pollIntervalRef.current = pollStatus
  }

  const handleDetectDVD = async () => {
    setLoading(true)
    setError('')
    setAvailableDevices([])
    setManualDevice('')
    try {
      const response = await api.get('/dvd/detect')
      console.log('📀 Réponse détection DVD:', response.data)
      
      if (response.data.success && response.data.device) {
        setDevice(response.data.device)
        setStep(2)
        setError('')
      } else {
        // Même si la détection automatique échoue, proposer les devices disponibles
        if (response.data.available_devices && response.data.available_devices.length > 0) {
          setAvailableDevices(response.data.available_devices)
          setManualDevice(response.data.available_devices[0])
          setError(response.data.message || response.data.error || 'Aucun lecteur DVD détecté automatiquement, mais des lecteurs sont disponibles. Sélectionnez-en un manuellement.')
        } else {
          setError(response.data.message || response.data.error || 'Aucun lecteur DVD détecté. Assurez-vous qu\'un DVD/CD est inséré dans le lecteur et que le conteneur Docker a accès aux devices.')
        }
      }
    } catch (error: any) {
      console.error('❌ Erreur détection DVD:', error)
      const errorMessage = error.response?.data?.message ||
        error.response?.data?.error ||
        'Erreur lors de la détection du lecteur DVD. Vérifiez que le conteneur Docker a accès aux devices (/dev/sr*, /dev/dvd, etc.).'
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  const handleManualDeviceSelect = () => {
    if (manualDevice) {
      setDevice(manualDevice)
      setStep(2)
    } else {
      setError('Veuillez sélectionner un lecteur DVD')
    }
  }

  const handleGetDVDInfo = async () => {
    setLoading(true)
    setError('')
    setAnalyzingProgress(0)
    setAnalyzingStage('Initialisation de l\'analyse...')

    try {
      const progressInterval = setInterval(() => {
        setAnalyzingProgress((prev) => {
          if (prev >= 90) return prev
          return prev + 5
        })
      }, 500)

      const stageMessages = [
        'Lecture du device DVD...',
        'Analyse des pistes vidéo...',
        'Détection des pistes audio...',
        'Recherche des sous-titres...',
        'Extraction des métadonnées...',
        'Finalisation...'
      ]
      let stageIndex = 0
      const stageInterval = setInterval(() => {
        if (stageIndex < stageMessages.length - 1) {
          stageIndex++
          setAnalyzingStage(stageMessages[stageIndex])
        }
      }, 2000)

      const response = await api.get(`/dvd/info?device=${encodeURIComponent(device!)}`, {
        timeout: 60000
      })

      clearInterval(progressInterval)
      clearInterval(stageInterval)
      setAnalyzingProgress(100)
      setAnalyzingStage('Analyse terminée !')

      if (response.data) {
        const dvdData = {
          ...response.data,
          video_tracks: response.data.video_tracks || [],
          audio_tracks: response.data.audio_tracks || [],
          subtitle_tracks: response.data.subtitle_tracks || [],
          title: response.data.title || 'DVD Extraite',
        }

        setDvdInfo(dvdData)
        setTitle(dvdData.title)
        setSelectedVideoTrack(0)
        setSelectedAudioTracks(dvdData.audio_tracks.length > 0 ? [0] : [])
        setSelectedSubtitleTracks([])
        setStep(3)
      } else {
        setError('Aucune information trouvée sur le DVD.')
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.message ||
        error.response?.data?.error ||
        'Erreur lors de la lecture du DVD.'
      setError(errorMessage)
    } finally {
      setLoading(false)
      setAnalyzingProgress(0)
      setAnalyzingStage('')
    }
  }

  const toggleAudioTrack = (index: number) => {
    setSelectedAudioTracks((prev) =>
      prev.includes(index)
        ? prev.filter((i) => i !== index)
        : [...prev, index]
    )
  }

  const toggleSubtitleTrack = (index: number) => {
    setSelectedSubtitleTracks((prev) =>
      prev.includes(index)
        ? prev.filter((i) => i !== index)
        : [...prev, index]
    )
  }

  const toggleQuality = (quality: string) => {
    setQualities((prev) =>
      prev.includes(quality)
        ? prev.filter((q) => q !== quality)
        : [...prev, quality]
    )
  }

  const handleCancelExtraction = async () => {
    if (!videoId) return

    if (!window.confirm('Êtes-vous sûr de vouloir annuler l\'extraction ?')) {
      return
    }

    setCancelling(true)
    try {
      await api.post(`/dvd/extract/${videoId}/cancel`)
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current)
        pollIntervalRef.current = null
      }
      setExtracting(false)
      setExtractionProgress(0)
      setExtractionStage('')
      setExtractionDetails('')
      setError('Extraction annulée par l\'utilisateur')
      setStep(3)
    } catch (error: any) {
      setError(error.response?.data?.error || 'Erreur lors de l\'annulation')
    } finally {
      setCancelling(false)
    }
  }

  const handleExtract = async () => {
    if (!title.trim()) {
      setError('Le titre est requis')
      return
    }
    if (selectedAudioTracks.length === 0) {
      const confirmNoAudio = window.confirm(
        '⚠️ Aucune piste audio sélectionnée. La vidéo sera extraite sans son. Voulez-vous continuer ?'
      )
      if (!confirmNoAudio) {
        return
      }
    }
    if (qualities.length === 0) {
      setError('Sélectionnez au moins une qualité')
      return
    }

    setLoading(true)
    setError('')
    setExtracting(true)
    setExtractionProgress(0)
    setExtractionStage('Initialisation de l\'extraction...')
    setExtractionDetails('Préparation du processus d\'extraction')
    setStep(4)

    try {
      const response = await api.post('/dvd/extract', {
        device_path: device,
        title,
        description,
        category_id: categoryId || null,
        sub_category_id: subCategoryId || null,
        is_public: isPublic,
        video_track_index: selectedVideoTrack,
        audio_track_indices: selectedAudioTracks,
        subtitle_track_indices: selectedSubtitleTracks,
        qualities
      })

      setVideoId(response.data.video_id)
      setExtractionProgress(5)
      setExtractionStage('Extraction du DVD en cours...')
      startPolling(response.data.video_id)
    } catch (error: any) {
      setError(error.response?.data?.error || 'Erreur lors du démarrage de l\'extraction')
      setExtracting(false)
      setStep(3)
      setExtractionProgress(0)
      setExtractionStage('')
      setExtractionDetails('')
    } finally {
      setLoading(false)
    }
  }

  if (!user) {
    return (
      <div className="dvd-extract-page">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="dvd-extract-page">
      <div className="dvd-extract-container">
        <h1>📀 Extraction DVD</h1>
        <p className="subtitle">Extrayez vos DVD directement dans votre bibliothèque</p>

        {error && (
          <div className="error-message">
            <span>❌ {error}</span>
            <button onClick={() => setError('')}>×</button>
          </div>
        )}

        {step === 1 && (
          <div className="step-container">
            <div className="step-header">
              <h2>📀 Étape 1: Détection du lecteur DVD</h2>
              <p className="step-description">
                Insérez votre DVD dans le lecteur, puis cliquez sur "Détecter automatiquement"
              </p>
            </div>

            <div className="detection-section">
              <button
                onClick={handleDetectDVD}
                disabled={loading}
                className="btn-primary btn-large"
              >
                {loading ? '⏳ Détection en cours...' : '🔍 Détecter automatiquement'}
              </button>

              {availableDevices.length > 0 && (
                <div className="manual-device-selection">
                  <h3>📋 Lecteurs disponibles détectés :</h3>
                  <p className="device-hint">
                    Aucun DVD n'a été détecté automatiquement, mais des lecteurs sont disponibles. 
                    Sélectionnez-en un manuellement et assurez-vous qu'un DVD est inséré.
                  </p>
                  <div className="device-selector">
                    <select
                      value={manualDevice}
                      onChange={(e) => setManualDevice(e.target.value)}
                      className="device-select"
                    >
                      {availableDevices.map((dev) => (
                        <option key={dev} value={dev}>{dev}</option>
                      ))}
                    </select>
                    <button onClick={handleManualDeviceSelect} className="btn-secondary">
                      Utiliser ce lecteur
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="step-container">
            <div className="step-header">
              <h2>📖 Étape 2: Lecture des informations du DVD</h2>
              <p className="step-description">
                Cliquez sur le bouton ci-dessous pour analyser le contenu du DVD
              </p>
            </div>
            <div className="device-info">
              <p><strong>📀 Lecteur détecté:</strong> <code>{device}</code></p>
            </div>

            {loading && (analyzingProgress > 0 || analyzingStage) && (
              <div className="analyzing-progress-container">
                <div className="analyzing-stage">{analyzingStage || 'Analyse en cours...'}</div>
                <div className="analyzing-progress-bar">
                  <div
                    className="analyzing-progress-fill"
                    style={{ width: `${analyzingProgress}%` }}
                  ></div>
                </div>
                <div className="analyzing-percentage">{analyzingProgress}%</div>
              </div>
            )}

            {!loading && (
              <button
                onClick={handleGetDVDInfo}
                disabled={loading}
                className="btn-primary btn-large"
              >
                📖 Lire les informations du DVD
              </button>
            )}

            <button onClick={() => setStep(1)} className="btn-secondary">
              ← Retour
            </button>
          </div>
        )}

        {step === 3 && dvdInfo && (
          <div className="step-container">
            <div className="step-header">
              <h2>Étape 3: Configuration de l'extraction</h2>
            </div>

            <div className="config-section">
              <h3>📝 Informations de la vidéo</h3>
              <div className="form-group">
                <label>Titre *</label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Titre de la vidéo"
                  className="form-input"
                />
              </div>
              <div className="form-group">
                <label>Description</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Description de la vidéo"
                  className="form-textarea"
                  rows={3}
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Catégorie</label>
                  <select
                    value={categoryId}
                    onChange={(e) => {
                      setCategoryId(e.target.value)
                      setSubCategoryId('')
                    }}
                    className="form-select"
                  >
                    <option value="">Aucune</option>
                    {categories.map((cat: any) => (
                      <option key={cat.id} value={cat.id}>{cat.name}</option>
                    ))}
                  </select>
                </div>
                {subCategories.length > 0 && (
                  <div className="form-group">
                    <label>Sous-catégorie</label>
                    <select
                      value={subCategoryId}
                      onChange={(e) => setSubCategoryId(e.target.value)}
                      className="form-select"
                    >
                      <option value="">Aucune</option>
                      {subCategories.map((sub: any) => (
                        <option key={sub.id} value={sub.id}>{sub.name}</option>
                      ))}
                    </select>
                  </div>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>🎬 Piste vidéo</h3>
              <div className="tracks-list">
                {dvdInfo.video_tracks && dvdInfo.video_tracks.length > 0 ? (
                  dvdInfo.video_tracks.map((track: any, idx: number) => (
                    <label key={idx} className="track-item">
                      <input
                        type="radio"
                        name="videoTrack"
                        checked={selectedVideoTrack === idx}
                        onChange={() => setSelectedVideoTrack(idx)}
                      />
                      <span>Piste {idx + 1} - {track.codec || 'Inconnu'}</span>
                    </label>
                  ))
                ) : (
                  <p className="no-tracks">Aucune piste vidéo disponible</p>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>🔊 Pistes audio</h3>
              <div className="tracks-list">
                {dvdInfo.audio_tracks && dvdInfo.audio_tracks.length > 0 ? (
                  dvdInfo.audio_tracks.map((track: any, idx: number) => (
                    <label key={idx} className="track-item">
                      <input
                        type="checkbox"
                        checked={selectedAudioTracks.includes(idx)}
                        onChange={() => toggleAudioTrack(idx)}
                      />
                      <span>
                        {track.title || `Piste ${idx + 1} - ${track.language || 'Inconnu'}`}
                      </span>
                    </label>
                  ))
                ) : (
                  <p className="no-tracks">Aucune piste audio disponible</p>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>🎥 Qualités de transcodage</h3>
              <div className="qualities-grid">
                {['2160p', '1440p', '1080p', '720p', '480p', '360p'].map((quality) => (
                  <label key={quality} className="quality-item">
                    <input
                      type="checkbox"
                      checked={qualities.includes(quality)}
                      onChange={() => toggleQuality(quality)}
                    />
                    <span>{quality}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="actions">
              <button onClick={() => setStep(2)} className="btn-secondary">
                ← Retour
              </button>
              <button
                onClick={handleExtract}
                disabled={loading || !title.trim()}
                className="btn-primary btn-large"
              >
                🚀 Démarrer l'extraction
              </button>
            </div>
          </div>
        )}

        {step === 4 && extracting && (
          <div className="step-container">
            <div className="extraction-progress">
              <h2>⏳ {extractionStage || 'Extraction en cours...'}</h2>
              <div className="progress-bar-container">
                <div className="progress-bar" style={{ width: `${extractionProgress}%` }}></div>
              </div>
              <p className="progress-text">{Math.round(extractionProgress)}%</p>
              {extractionDetails && (
                <p className="progress-info">{extractionDetails}</p>
              )}
              <div className="extraction-actions">
                <button
                  onClick={handleCancelExtraction}
                  disabled={cancelling}
                  className="btn-secondary btn-cancel"
                >
                  {cancelling ? '⏳ Annulation...' : '❌ Annuler l\'extraction'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

