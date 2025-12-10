import React, { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import VideoPlayer from '../../components/VideoPlayer'
import VideoGrid from '../../components/VideoGrid'
import api from '../../services/api'
import { useAuth } from '../../context/AuthContext'
import { filterValidVideos } from '../../utils/videoUtils'
import LoadingSpinner from '../../components/LoadingSpinner'
import './VideoPlayerPage.css'

const VideoPlayerPage = () => {
  const { id } = useParams()
  const { user } = useAuth()
  const [video, setVideo] = useState(null)
  const [relatedVideos, setRelatedVideos] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    if (id) {
      fetchVideo()
      fetchRelatedVideos()
    }
    
    // Nettoyer les intervalles au démontage
    return () => {
      // Le cleanup sera géré dans startStatusPolling
    }
  }, [id])

  const fetchVideo = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Essayer d'abord de récupérer la vidéo
      try {
        const response = await api.get(`/videos/${id}`)
        
        // Vérifier que la vidéo existe et a un ID valide
        if (!response.data || !response.data.id) {
          throw new Error('Vidéo invalide')
        }
        
        setVideo(response.data)
        
        // Si la vidéo est en traitement, démarrer le polling du statut
        if (response.data.status === 'processing') {
          startStatusPolling(id)
        }
        
        // Si la vidéo est en erreur, afficher un message approprié
        if (response.data.status === 'error') {
          setError('Cette vidéo est invalide ou a rencontré une erreur lors du traitement. Elle ne peut pas être visionnée.')
        }
      } catch (err) {
        // Si 404, essayer de récupérer le statut (pour les vidéos en traitement ou en erreur)
        // Essayer même sans user car le statut peut être accessible publiquement
        if (err.response?.status === 404) {
          try {
            console.log('Vidéo non trouvée (404), tentative de récupération du statut...')
            const statusResponse = await api.get(`/videos/${id}/status`)
            if (statusResponse.data) {
              console.log('Statut récupéré:', statusResponse.data)
              // Créer un objet vidéo minimal avec les infos du statut
              const videoData = {
                id: statusResponse.data.video_id || id,
                title: statusResponse.data.title || 'Vidéo en traitement',
                description: statusResponse.data.description || '',
                status: statusResponse.data.status,
                thumbnail_path: statusResponse.data.thumbnail_path,
                formats: [],
                progress: statusResponse.data.progress || 0,
                formats_count: statusResponse.data.formats_count || 0,
                views: 0, // Valeur par défaut pour éviter les erreurs
                user: null,
                created_at: new Date().toISOString() // Valeur par défaut
              }
              
              // Si la vidéo est en erreur, afficher un message approprié
              if (statusResponse.data.status === 'error') {
                setError('Cette vidéo est invalide ou a rencontré une erreur lors du traitement. Elle ne peut pas être visionnée.')
              }
              
              setVideo(videoData)
              setLoading(false) // Arrêter le chargement car on a les infos
              
              // Démarrer le polling seulement si en traitement
              if (statusResponse.data.status === 'processing') {
                console.log('Démarrage du polling pour la vidéo en traitement...')
                startStatusPolling(id)
              }
              return // Sortir de la fonction car on a géré le cas
            } else {
              throw err
            }
          } catch (statusErr) {
            console.error('Erreur lors de la récupération du statut:', statusErr)
            // Si le statut n'est pas accessible non plus, afficher l'erreur
            throw err
          }
        } else {
          throw err
        }
      }
    } catch (err) {
      console.error('Erreur lors de la récupération de la vidéo:', err)
      
      // Si 404, essayer une dernière fois de récupérer le statut
      // (au cas où la première tentative aurait échoué)
      if (err.response?.status === 404) {
        try {
          console.log('Dernière tentative de récupération du statut...')
          const statusResponse = await api.get(`/videos/${id}/status`)
          if (statusResponse.data && statusResponse.data.status === 'processing') {
            // Créer un objet vidéo minimal pour afficher l'écran de traitement
            const videoData = {
              id: statusResponse.data.video_id || id,
              title: statusResponse.data.title || 'Vidéo en traitement',
              description: statusResponse.data.description || '',
              status: 'processing',
              thumbnail_path: statusResponse.data.thumbnail_path,
              formats: [],
              progress: statusResponse.data.progress || 0,
              formats_count: statusResponse.data.formats_count || 0,
              views: 0,
              user: null,
              created_at: new Date().toISOString()
            }
            setVideo(videoData)
            setLoading(false)
            startStatusPolling(id)
            return
          }
        } catch (statusErr) {
          console.error('Impossible de récupérer le statut:', statusErr)
        }
      }
      
      const errorMessage = err.response?.data?.error || err.message || 'Vidéo non trouvée'
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }
  
  // Polling du statut pour les vidéos en traitement
  const startStatusPolling = (videoId) => {
    const pollInterval = setInterval(async () => {
      try {
        const statusResponse = await api.get(`/videos/${videoId}/status`)
        if (statusResponse.data) {
          // Mettre à jour le statut et la progression
          setVideo(prev => ({
            ...prev,
            status: statusResponse.data.status,
            progress: statusResponse.data.progress,
            formats_count: statusResponse.data.formats_count,
            thumbnail_path: statusResponse.data.thumbnail_path || prev?.thumbnail_path
          }))
          
          // Si la vidéo est prête, arrêter le polling et recharger la vidéo complète
          if (statusResponse.data.status === 'ready') {
            clearInterval(pollInterval)
            // Recharger la vidéo complète
            try {
              const fullResponse = await api.get(`/videos/${videoId}`)
              if (fullResponse.data) {
                setVideo(fullResponse.data)
              }
            } catch (err) {
              console.error('Erreur lors du rechargement de la vidéo:', err)
            }
          } else if (statusResponse.data.status === 'error') {
            clearInterval(pollInterval)
            setError('Erreur lors du traitement de la vidéo. Veuillez réessayer l\'upload.')
          }
        }
      } catch (err) {
        console.error('Erreur lors du polling du statut:', err)
        // Arrêter le polling en cas d'erreur
        clearInterval(pollInterval)
      }
    }, 2000) // Polling toutes les 2 secondes
    
    // Nettoyer l'intervalle quand le composant est démonté
    return () => clearInterval(pollInterval)
  }

  const fetchRelatedVideos = async () => {
    try {
      const response = await api.get('/videos', {
        params: {
          page: 1,
          limit: 12,
        },
      })
      setRelatedVideos(filterValidVideos(response.data.videos || []))
    } catch (err) {
      console.error('Erreur lors de la récupération des vidéos similaires:', err)
    }
  }

  const handleTimeUpdate = async (position) => {
    if (user && video) {
      try {
        await api.post(`/history/${video.id}`, { position })
      } catch (err) {
        console.error('Erreur lors de la mise à jour de l\'historique:', err)
      }
    }
  }

  if (loading) {
    return (
      <div className="video-player-page">
        <LoadingSpinner message="Chargement de la vidéo..." />
      </div>
    )
  }

  if (error || !video) {
    return (
      <div className="video-player-page">
        <div className="error-container">
          <div className="error-icon">⚠️</div>
          <div className="error-message">{error || 'Vidéo non trouvée'}</div>
          {error && error.includes('traitement') && (
            <button 
              onClick={() => {
                setError(null)
                setLoading(true)
                fetchVideo()
              }} 
              className="retry-button"
            >
              🔄 Réessayer
            </button>
          )}
        </div>
      </div>
    )
  }
  
  // Afficher un écran d'erreur si la vidéo est en erreur
  if (video && video.status === 'error') {
    return (
      <div className="video-player-page">
        <div className="error-container">
          <div className="error-icon">❌</div>
          <h2>Vidéo invalide</h2>
          <div className="error-message">
            Cette vidéo est invalide ou a rencontré une erreur lors du traitement. Elle ne peut pas être visionnée.
          </div>
          <div className="video-error-details" style={{ marginTop: '20px', textAlign: 'left', maxWidth: '600px' }}>
            <p><strong>Titre:</strong> {video.title || 'Sans titre'}</p>
            <p><strong>Statut:</strong> {video.status}</p>
            {video.description && <p><strong>Description:</strong> {video.description}</p>}
          </div>
          <div style={{ marginTop: '20px' }}>
            <button onClick={() => window.history.back()} className="retry-button">
              ← Retour
            </button>
            {user && user.role === 'admin' && (
              <button 
                onClick={() => window.location.href = '/admin/videos'} 
                className="retry-button"
                style={{ marginLeft: '10px' }}
              >
                Gérer les vidéos
              </button>
            )}
          </div>
        </div>
      </div>
    )
  }
  
  // Afficher un message si la vidéo est en cours de traitement
  if (video && video.status === 'processing') {
    return (
      <div className="video-player-page">
        <div className="processing-container">
          <div className="processing-icon">⏳</div>
          <h2>Vidéo en cours de traitement</h2>
          <p>Votre vidéo est en cours de traitement. Elle sera disponible dans quelques instants.</p>
          <div className="video-info">
            <h3>{video.title}</h3>
            {video.description && <p>{video.description}</p>}
          </div>
          <button 
            onClick={() => {
              setLoading(true)
              fetchVideo()
            }} 
            className="refresh-button"
          >
            🔄 Actualiser
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="video-player-page">
      <div className="video-main">
        <VideoPlayer
          video={video}
          formats={video.formats}
          subtitles={video.subtitles}
          onTimeUpdate={handleTimeUpdate}
        />
        <div className="video-details">
          <h1 className="video-title">{video.title}</h1>
          <div className="video-meta-info">
            <span className="video-views-count">
              {(video.views || 0).toLocaleString()} vues
            </span>
            {video.user && (
              <span className="video-author-name">{video.user.username}</span>
            )}
            <span className="video-date">
              {new Date(video.created_at).toLocaleDateString('fr-FR')}
            </span>
          </div>
          {video.description && (
            <div className="video-description">
              <p>{video.description}</p>
            </div>
          )}
        </div>
      </div>

      {relatedVideos.length > 0 && (
        <div className="related-videos">
          <VideoGrid videos={relatedVideos} title="Vidéos similaires" />
        </div>
      )}
    </div>
  )
}

export default VideoPlayerPage

