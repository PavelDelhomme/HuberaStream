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
  const [categories, setCategories] = useState([])
  const [extracting, setExtracting] = useState(false)
  const [extractionProgress, setExtractionProgress] = useState(0)
  const [videoId, setVideoId] = useState(null)

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    fetchCategories()
  }, [user, navigate])

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
      setDevice(response.data.device)
      setStep(2)
    } catch (error) {
      setError('Aucun lecteur DVD détecté. Assurez-vous qu\'un DVD est inséré dans le lecteur.')
    } finally {
      setLoading(false)
    }
  }

  const handleGetDVDInfo = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.get(`/dvd/info?device=${device}`)
      setDvdInfo(response.data)
      setTitle(response.data.title || 'DVD Extraite')
      setSelectedVideoTrack(0)
      setSelectedAudioTracks(response.data.audio_tracks.length > 0 ? [0] : [])
      setSelectedSubtitleTracks([])
      setStep(3)
    } catch (error) {
      setError('Erreur lors de la lecture du DVD. Vérifiez que le DVD est bien inséré.')
    } finally {
      setLoading(false)
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
      
      // Poller le statut
      const pollStatus = setInterval(async () => {
        try {
          const statusResponse = await api.get(`/dvd/extract/${response.data.video_id}/status`)
          if (statusResponse.data.status === 'ready') {
            clearInterval(pollStatus)
            setExtractionProgress(100)
            setTimeout(() => {
              navigate(`/video/${response.data.video_id}`)
            }, 2000)
          } else if (statusResponse.data.status === 'error') {
            clearInterval(pollStatus)
            setError('Erreur lors de l\'extraction')
            setExtracting(false)
          }
        } catch (err) {
          console.error('Erreur lors de la vérification du statut:', err)
        }
      }, 2000)

      // Nettoyer après 10 minutes
      setTimeout(() => clearInterval(pollStatus), 600000)
    } catch (error) {
      setError(error.response?.data?.error || 'Erreur lors du démarrage de l\'extraction')
      setExtracting(false)
      setStep(3)
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
              <h2>Étape 1: Détection du lecteur DVD</h2>
              <p>Insérez votre DVD dans le lecteur, puis cliquez sur "Détecter"</p>
            </div>
            <button 
              onClick={handleDetectDVD} 
              disabled={loading}
              className="btn-primary btn-large"
            >
              {loading ? '⏳ Détection en cours...' : '🔍 Détecter le lecteur DVD'}
            </button>
          </div>
        )}

        {/* Étape 2: Informations */}
        {step === 2 && (
          <div className="step-container">
            <div className="step-header">
              <h2>Étape 2: Lecture des informations du DVD</h2>
              <p>Analyse du contenu du DVD...</p>
            </div>
            <div className="device-info">
              <p><strong>Lecteur détecté:</strong> {device}</p>
            </div>
            <button 
              onClick={handleGetDVDInfo} 
              disabled={loading}
              className="btn-primary btn-large"
            >
              {loading ? '⏳ Analyse en cours...' : '📖 Lire les informations du DVD'}
            </button>
            <button onClick={() => setStep(1)} className="btn-secondary">
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
                {dvdInfo.video_tracks.map((track, idx) => (
                  <label key={idx} className="track-item">
                    <input
                      type="radio"
                      name="videoTrack"
                      checked={selectedVideoTrack === idx}
                      onChange={() => setSelectedVideoTrack(idx)}
                    />
                    <span>Piste {idx + 1} - {track.codec}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="config-section">
              <h3>🔊 Pistes audio</h3>
              <p className="section-description">Sélectionnez les pistes audio à inclure</p>
              <div className="tracks-list">
                {dvdInfo.audio_tracks.map((track, idx) => (
                  <label key={idx} className="track-item">
                    <input
                      type="checkbox"
                      checked={selectedAudioTracks.includes(idx)}
                      onChange={() => toggleAudioTrack(idx)}
                    />
                    <span>
                      Piste {idx + 1} - {track.language || 'Inconnu'} ({track.codec})
                    </span>
                  </label>
                ))}
              </div>
            </div>

            <div className="config-section">
              <h3>📝 Sous-titres</h3>
              <p className="section-description">Sélectionnez les pistes de sous-titres à inclure</p>
              <div className="tracks-list">
                {dvdInfo.subtitle_tracks.length > 0 ? (
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
                {['1080p', '720p', '480p', '360p'].map(quality => (
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
              <h2>⏳ Extraction en cours...</h2>
              <div className="progress-bar-container">
                <div className="progress-bar" style={{ width: `${extractionProgress}%` }}></div>
              </div>
              <p className="progress-text">{extractionProgress}%</p>
              <p className="progress-info">
                L'extraction peut prendre plusieurs minutes selon la taille du DVD...
              </p>
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

