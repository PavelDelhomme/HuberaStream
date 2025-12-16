import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './DVDExtract.css'

const DVDExtract = () => {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [step, setStep] = useState(1) // 1: Détection, 2: Info, 3: Configuration, 4: Extraction
  const [device, setDevice] = useState(null)
  const [availableDevices, setAvailableDevices] = useState([])
  const [manualDevice, setManualDevice] = useState('')
  const [dvdInfo, setDvdInfo] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  
  // Configuration d'extraction
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [subCategoryId, setSubCategoryId] = useState('')
  const [isPublic, setIsPublic] = useState(false)
  const [selectedVideoTrack, setSelectedVideoTrack] = useState(0)
  const [selectedAudioTracks, setSelectedAudioTracks] = useState([])
  const [selectedSubtitleTracks, setSelectedSubtitleTracks] = useState([])
  const [qualities, setQualities] = useState(['1080p', '720p', '480p'])
  const [suggestedQualities, setSuggestedQualities] = useState(['1080p', '720p', '480p'])
  const [categories, setCategories] = useState([])
  const [extracting, setExtracting] = useState(false)
  const [extractionProgress, setExtractionProgress] = useState(0)
  const [extractionStage, setExtractionStage] = useState('')
  const [extractionDetails, setExtractionDetails] = useState('')
  const [videoId, setVideoId] = useState(null)
  const [analyzingProgress, setAnalyzingProgress] = useState(0)
  const [analyzingStage, setAnalyzingStage] = useState('')
  const [pollInterval, setPollInterval] = useState(null)
  const [cancelling, setCancelling] = useState(false)

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    fetchCategories()
    checkForActiveExtraction()
  }, [user, navigate])

  // Vérifier s'il y a une extraction en cours
  const checkForActiveExtraction = async () => {
    try {
      // Récupérer les vidéos en cours de traitement de l'utilisateur
      const response = await api.get('/videos?status=processing')
      const processingVideos = response.data.videos || []
      
      // Filtrer pour trouver les vidéos DVD en cours (celles qui ont été créées récemment et sont en processing)
      // On peut aussi vérifier si le titre contient "DVD" ou si c'est dans le dossier dvd
      const dvdVideos = processingVideos.filter(video => {
        // Vérifier si c'est une vidéo DVD récente (créée dans les dernières 2 heures)
        const createdAt = new Date(video.created_at)
        const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000)
        return createdAt > twoHoursAgo && video.status === 'processing'
      })
      
      if (dvdVideos.length > 0) {
        // Prendre la plus récente
        const latestVideo = dvdVideos.sort((a, b) => 
          new Date(b.created_at) - new Date(a.created_at)
        )[0]
        
        console.log('📀 Extraction DVD en cours détectée:', latestVideo.id)
        
        // Reprendre le suivi de l'extraction
        setVideoId(latestVideo.id)
        setExtracting(true)
        setStep(4)
        setExtractionProgress(5)
        setExtractionStage('Extraction en cours...')
        setExtractionDetails(`Reprise du suivi de l'extraction - ${latestVideo.title || 'DVD'}`)
        
        // Afficher un message informatif
        setMessage({
          type: 'info',
          text: `📀 Une extraction DVD est en cours. Reprise du suivi...`
        })
        
        // Démarrer le polling
        startPolling(latestVideo.id)
      }
    } catch (error) {
      console.error('Erreur lors de la vérification des extractions en cours:', error)
    }
  }

  // Fonction pour démarrer le polling
  const startPolling = (videoIdToPoll) => {
    // Arrêter le polling existant si présent
    if (pollInterval) {
      clearInterval(pollInterval)
    }
    
    let lastProgress = 5
    let noProgressCount = 0
    
    const pollStatus = setInterval(async () => {
      try {
        // Essayer d'abord l'API de statut vidéo (plus détaillée)
        try {
          const videoResponse = await api.get(`/videos/${videoIdToPoll}/status`, {
            timeout: 5000
          })
          const videoStatus = videoResponse.data.status
          const videoProgress = videoResponse.data.progress || 0
          
          // Si la progression a changé, réinitialiser le compteur
          if (videoProgress > lastProgress) {
            noProgressCount = 0
            lastProgress = videoProgress
          } else {
            noProgressCount++
          }
          
          // Mapper le statut aux étapes d'extraction
          if (videoStatus === 'processing') {
            // Calculer la progression basée sur le statut
            if (videoProgress < 30) {
              setExtractionStage('Extraction du DVD...')
              setExtractionDetails(`Lecture des pistes sélectionnées (${Math.round(videoProgress)}%)`)
            } else if (videoProgress < 50) {
              setExtractionStage('Compression optimale...')
              setExtractionDetails(`Compression H.265/HEVC en cours (${Math.round(videoProgress)}%)`)
            } else if (videoProgress < 90) {
              setExtractionStage('Transcodage multi-qualités...')
              setExtractionDetails(`Génération des formats (${Math.round(videoProgress)}%)`)
            } else {
              setExtractionStage('Finalisation...')
              setExtractionDetails(`Génération des miniatures et métadonnées (${Math.round(videoProgress)}%)`)
            }
            
            setExtractionProgress(videoProgress)
            
            // Si pas de progression depuis 30 secondes (10 polls), afficher un avertissement
            if (noProgressCount > 10 && videoProgress < 30) {
              setExtractionDetails(`Extraction en cours... (${Math.round(videoProgress)}%) - Cela peut prendre du temps pour les gros DVDs`)
            }
          } else if (videoStatus === 'ready') {
            clearInterval(pollStatus)
            setPollInterval(null)
            setExtractionProgress(100)
            setExtractionStage('Extraction terminée !')
            setExtractionDetails('Votre vidéo est prête. Redirection en cours...')
            setTimeout(() => {
              navigate(`/video/${videoIdToPoll}`)
            }, 2000)
          } else if (videoStatus === 'error') {
            clearInterval(pollStatus)
            setPollInterval(null)
            setError('Erreur lors de l\'extraction. Vérifiez les logs pour plus de détails.')
            setExtracting(false)
            setStep(3)
          } else if (videoStatus === 'cancelled') {
            clearInterval(pollStatus)
            setPollInterval(null)
            setError('Extraction annulée')
            setExtracting(false)
            setStep(3)
          }
        } catch (videoErr) {
          // Fallback sur l'API DVD si l'API vidéo échoue
          try {
            const statusResponse = await api.get(`/dvd/extract/${videoIdToPoll}/status`, {
              timeout: 5000
            })
            const dvdStatus = statusResponse.data.status
            const dvdProgress = statusResponse.data.progress || 0
            
            if (dvdStatus === 'ready') {
              clearInterval(pollStatus)
              setPollInterval(null)
              setExtractionProgress(100)
              setExtractionStage('Extraction terminée !')
              setExtractionDetails('Votre vidéo est prête. Redirection en cours...')
              setTimeout(() => {
                navigate(`/video/${videoIdToPoll}`)
              }, 2000)
            } else if (dvdStatus === 'error') {
              clearInterval(pollStatus)
              setPollInterval(null)
              setError('Erreur lors de l\'extraction')
              setExtracting(false)
              setStep(3)
            } else if (dvdStatus === 'processing') {
              // Utiliser la progression de l'API DVD si disponible
              if (dvdProgress > 0) {
                setExtractionProgress(dvdProgress)
                lastProgress = dvdProgress
                noProgressCount = 0
              } else {
                // Incrémenter progressivement si pas de progression disponible
                if (lastProgress < 95) {
                  lastProgress = Math.min(lastProgress + 1, 95)
                  setExtractionProgress(lastProgress)
                }
              }
              
              // Mapper la progression aux étapes
              if (lastProgress < 30) {
                setExtractionStage('Extraction du DVD...')
                setExtractionDetails(`Lecture des pistes (${Math.round(lastProgress)}%)`)
              } else if (lastProgress < 50) {
                setExtractionStage('Compression optimale...')
                setExtractionDetails(`Compression H.265/HEVC (${Math.round(lastProgress)}%)`)
              } else if (lastProgress < 90) {
                setExtractionStage('Transcodage multi-qualités...')
                setExtractionDetails(`Génération des formats (${Math.round(lastProgress)}%)`)
              } else {
                setExtractionStage('Finalisation...')
                setExtractionDetails(`Finalisation (${Math.round(lastProgress)}%)`)
              }
            }
          } catch (statusErr) {
            console.error('Erreur lors de la vérification du statut:', statusErr)
            // Si les deux APIs échouent, incrémenter progressivement
            if (lastProgress < 95) {
              lastProgress = Math.min(lastProgress + 0.5, 95)
              setExtractionProgress(lastProgress)
            }
          }
        }
      } catch (err) {
        console.error('Erreur lors du polling:', err)
        // Continuer à poller même en cas d'erreur temporaire
      }
    }, 2000) // Poller toutes les 2 secondes
    
    setPollInterval(pollStatus)
    
    // Nettoyer après 30 minutes
    setTimeout(() => {
      if (pollStatus) {
        clearInterval(pollStatus)
        setPollInterval(null)
      }
    }, 1800000)
  }

  const fetchCategories = async () => {
    try {
      const response = await api.get('/categories')
      setCategories(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const handleDetectDVD = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.get('/dvd/detect')
      // Vérifier si la détection a réussi
      if (response.data.success && response.data.device) {
        setDevice(response.data.device)
        setStep(2)
      } else {
        // La détection a échoué mais la requête a réussi (200)
        // Afficher les devices disponibles pour sélection manuelle
        if (response.data.available_devices && response.data.available_devices.length > 0) {
          setAvailableDevices(response.data.available_devices)
          setManualDevice(response.data.available_devices[0])
        }
        const errorMessage = response.data.message || 
                            response.data.error || 
                            'Aucun lecteur DVD détecté automatiquement.'
        setError(errorMessage)
      }
    } catch (error) {
      // Erreur réseau ou authentification
      const errorMessage = error.response?.data?.message || 
                          error.response?.data?.error || 
                          (error.response?.status === 401 || error.response?.status === 403
                            ? 'Vous devez être connecté en tant qu\'administrateur pour accéder à cette fonctionnalité.'
                            : 'Aucun lecteur DVD détecté. Assurez-vous qu\'un DVD/CD est inséré dans le lecteur et que le conteneur Docker a accès aux devices.')
      setError(errorMessage)
      console.error('Erreur détection DVD:', error)
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
      // Simuler une progression pendant l'analyse
      const progressInterval = setInterval(() => {
        setAnalyzingProgress(prev => {
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

      const response = await api.get(`/dvd/info?device=${encodeURIComponent(device)}`, {
        timeout: 60000 // 60 secondes pour l'analyse
      })
      
      clearInterval(progressInterval)
      clearInterval(stageInterval)
      setAnalyzingProgress(100)
      setAnalyzingStage('Analyse terminée !')

      if (response.data) {
        // S'assurer que les tableaux existent et sont initialisés
        const dvdData = {
          ...response.data,
          video_tracks: response.data.video_tracks || [],
          audio_tracks: response.data.audio_tracks || [],
          subtitle_tracks: response.data.subtitle_tracks || [],
          title: response.data.title || 'DVD Extraite',
          duration: response.data.duration || 0,
          chapters: response.data.chapters || []
        }
        
        console.log('📀 Informations DVD récupérées:', dvdData)
        console.log(`  - Titre: ${dvdData.title}`)
        console.log(`  - Pistes vidéo: ${dvdData.video_tracks.length}`)
        console.log(`  - Pistes audio: ${dvdData.audio_tracks.length}`)
        console.log(`  - Sous-titres: ${dvdData.subtitle_tracks.length}`)
        
        setDvdInfo(dvdData)
        setTitle(dvdData.title)
        setSelectedVideoTrack(0)
        setSelectedAudioTracks(dvdData.audio_tracks.length > 0 ? [0] : [])
        setSelectedSubtitleTracks([])
        setStep(3)
      } else {
        setError('Aucune information trouvée sur le DVD. Vérifiez que le DVD est bien inséré et lisible.')
      }
    } catch (error) {
      const errorMessage = error.response?.data?.message || 
                          error.response?.data?.error || 
                          'Erreur lors de la lecture du DVD. Vérifiez que le DVD est bien inséré et lisible.'
      setError(errorMessage)
      console.error('Erreur lecture DVD:', error)
      if (error.response?.data) {
        console.error('Détails:', error.response.data)
      }
    } finally {
      setLoading(false)
      setAnalyzingProgress(0)
      setAnalyzingStage('')
    }
  }

  const toggleAudioTrack = (index) => {
    setSelectedAudioTracks(prev => 
      prev.includes(index) 
        ? prev.filter(i => i !== index)
        : [...prev, index]
    )
  }

  const toggleSubtitleTrack = (index) => {
    setSelectedSubtitleTracks(prev => 
      prev.includes(index) 
        ? prev.filter(i => i !== index)
        : [...prev, index]
    )
  }

  const toggleQuality = (quality) => {
    setQualities(prev => 
      prev.includes(quality)
        ? prev.filter(q => q !== quality)
        : [...prev, quality]
    )
  }

  const handleCancelExtraction = async () => {
    if (!videoId) return
    
    if (!window.confirm('Êtes-vous sûr de vouloir annuler l\'extraction ? Tous les fichiers partiels seront supprimés.')) {
      return
    }

    setCancelling(true)
    try {
      await api.post(`/dvd/extract/${videoId}/cancel`)
      
      // Arrêter le polling
      if (pollInterval) {
        clearInterval(pollInterval)
        setPollInterval(null)
      }
      
      setExtracting(false)
      setExtractionProgress(0)
      setExtractionStage('')
      setExtractionDetails('')
      setError('Extraction annulée par l\'utilisateur')
      setStep(3)
    } catch (error) {
      setError(error.response?.data?.error || 'Erreur lors de l\'annulation')
      console.error('Erreur annulation:', error)
    } finally {
      setCancelling(false)
    }
  }

  // Nettoyer le polling au démontage
  useEffect(() => {
    return () => {
      if (pollInterval) {
        clearInterval(pollInterval)
      }
    }
  }, [pollInterval])

  const handleExtract = async () => {
    if (!title.trim()) {
      setError('Le titre est requis')
      return
    }
    if (selectedAudioTracks.length === 0) {
      setError('Sélectionnez au moins une piste audio')
      return
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
      setExtractionDetails('Initialisation de l\'extraction...')
      
      // Démarrer le polling avec la fonction réutilisable
      startPolling(response.data.video_id)
    } catch (error) {
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

        {/* Étape 1: Détection */}
        {step === 1 && (
          <div className="step-container">
            <div className="step-header">
              <h2>📀 Étape 1: Détection du lecteur DVD</h2>
              <p className="step-description">
                Insérez votre DVD dans le lecteur (USB ou interne), puis cliquez sur "Détecter automatiquement"
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
                  <h3>📋 Ou sélectionnez manuellement un lecteur :</h3>
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
                    <button 
                      onClick={handleManualDeviceSelect}
                      className="btn-secondary"
                    >
                      Utiliser ce lecteur
                    </button>
                  </div>
                  <p className="device-hint">
                    💡 Si votre lecteur n'apparaît pas, vérifiez qu'il est bien connecté et qu'un DVD est inséré.
                  </p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Étape 2: Informations */}
        {step === 2 && (
          <div className="step-container">
            <div className="step-header">
              <h2>📖 Étape 2: Lecture des informations du DVD</h2>
              <p className="step-description">
                Cliquez sur le bouton ci-dessous pour analyser le contenu du DVD et détecter toutes les pistes disponibles.
              </p>
            </div>
            <div className="device-info">
              <p><strong>📀 Lecteur détecté:</strong> <code>{device}</code></p>
            </div>
            
            {/* Affichage de la progression de l'analyse */}
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
            
            <button onClick={() => {
              setStep(1)
              setAnalyzingProgress(0)
              setAnalyzingStage('')
            }} className="btn-secondary">
              ← Retour
            </button>
          </div>
        )}

        {/* Étape 3: Configuration */}
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
                  rows="3"
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Catégorie</label>
                  <select
                    value={categoryId}
                    onChange={(e) => setCategoryId(e.target.value)}
                    className="form-select"
                  >
                    <option value="">Aucune</option>
                    {categories.map(cat => (
                      <option key={cat.id} value={cat.id}>{cat.name}</option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label>
                    <input
                      type="checkbox"
                      checked={isPublic}
                      onChange={(e) => setIsPublic(e.target.checked)}
                    />
                    Vidéo publique
                  </label>
                </div>
              </div>
            </div>

            <div className="config-section">
              <h3>🎬 Piste vidéo</h3>
              <div className="tracks-list">
                {dvdInfo.video_tracks && Array.isArray(dvdInfo.video_tracks) && dvdInfo.video_tracks.length > 0 ? (
                  dvdInfo.video_tracks.map((track, idx) => (
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
              <p className="section-description">Sélectionnez les pistes audio à inclure</p>
              <div className="tracks-list">
                {dvdInfo.audio_tracks && Array.isArray(dvdInfo.audio_tracks) && dvdInfo.audio_tracks.length > 0 ? (
                  dvdInfo.audio_tracks.map((track, idx) => (
                    <label key={idx} className="track-item">
                      <input
                        type="checkbox"
                        checked={selectedAudioTracks.includes(idx)}
                        onChange={() => toggleAudioTrack(idx)}
                      />
                      <span>
                        Piste {idx + 1} - {track.language || 'Inconnu'} ({track.codec || 'Inconnu'})
                      </span>
                    </label>
                  ))
                ) : (
                  <p className="no-tracks">Aucune piste audio disponible</p>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>📝 Sous-titres</h3>
              <p className="section-description">Sélectionnez les pistes de sous-titres à inclure</p>
              <div className="tracks-list">
                {dvdInfo.subtitle_tracks && Array.isArray(dvdInfo.subtitle_tracks) && dvdInfo.subtitle_tracks.length > 0 ? (
                  dvdInfo.subtitle_tracks.map((track, idx) => (
                    <label key={idx} className="track-item">
                      <input
                        type="checkbox"
                        checked={selectedSubtitleTracks.includes(idx)}
                        onChange={() => toggleSubtitleTrack(idx)}
                      />
                      <span>
                        Piste {idx + 1} - {track.language || 'Inconnu'}
                      </span>
                    </label>
                  ))
                ) : (
                  <p className="no-tracks">Aucun sous-titre disponible</p>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>🎥 Qualités de transcodage</h3>
              <p className="section-description">Sélectionnez les qualités à générer</p>
              <div className="qualities-grid">
                {['2160p', '1440p', '1080p', '720p', '480p', '360p'].map(quality => {
                  const isSuggested = suggestedQualities.includes(quality)
                  const isAvailable = suggestedQualities.includes(quality)
                  return (
                    <label 
                      key={quality} 
                      className={`quality-item ${!isAvailable ? 'quality-disabled' : ''} ${isSuggested ? 'quality-suggested' : ''}`}
                      title={!isAvailable ? 'Qualité supérieure à la résolution du DVD' : ''}
                    >
                      <input
                        type="checkbox"
                        checked={qualities.includes(quality)}
                        onChange={() => toggleQuality(quality)}
                        disabled={!isAvailable}
                      />
                      <span>{quality} {isSuggested && !isAvailable ? '(non disponible)' : ''}</span>
                    </label>
                  )
                })}
              </div>
            </div>

            <div className="actions">
              <button onClick={() => setStep(2)} className="btn-secondary">
                ← Retour
              </button>
              <button 
                onClick={handleExtract} 
                disabled={loading || !title.trim() || selectedAudioTracks.length === 0}
                className="btn-primary btn-large"
              >
                🚀 Démarrer l'extraction
              </button>
            </div>
          </div>
        )}

        {/* Étape 4: Extraction en cours */}
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
              <div className="extraction-steps">
                <div className={`extraction-step ${extractionProgress >= 5 ? 'active' : ''} ${extractionProgress >= 30 ? 'completed' : ''}`}>
                  <span className="step-number">1</span>
                  <span className="step-label">Extraction DVD</span>
                </div>
                <div className={`extraction-step ${extractionProgress >= 30 ? 'active' : ''} ${extractionProgress >= 50 ? 'completed' : ''}`}>
                  <span className="step-number">2</span>
                  <span className="step-label">Compression</span>
                </div>
                <div className={`extraction-step ${extractionProgress >= 50 ? 'active' : ''} ${extractionProgress >= 90 ? 'completed' : ''}`}>
                  <span className="step-number">3</span>
                  <span className="step-label">Transcodage</span>
                </div>
                <div className={`extraction-step ${extractionProgress >= 90 ? 'active' : ''} ${extractionProgress >= 100 ? 'completed' : ''}`}>
                  <span className="step-number">4</span>
                  <span className="step-label">Finalisation</span>
                </div>
              </div>
              <p className="progress-warning">
                ⚠️ Cette opération peut prendre plusieurs minutes selon la taille du DVD. Ne fermez pas cette page.
              </p>
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

        {step === 4 && !extracting && extractionProgress === 100 && (
          <div className="step-container">
            <div className="success-message">
              <h2>✅ Extraction terminée !</h2>
              <p>Votre vidéo est prête. Redirection en cours...</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default DVDExtract

