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
      } catch (err) {
        // Si 404, essayer de récupérer le statut (pour les vidéos en traitement)
        if (err.response?.status === 404 && user) {
          try {
            const statusResponse = await api.get(`/videos/${id}/status`)
            if (statusResponse.data) {
              // Créer un objet vidéo minimal avec les infos du statut
              setVideo({
                id: statusResponse.data.video_id,
                title: statusResponse.data.title || 'Vidéo en traitement',
                description: statusResponse.data.description || '',
                status: statusResponse.data.status,
                thumbnail_path: statusResponse.data.thumbnail_path,
                formats: [],
                progress: statusResponse.data.progress,
                formats_count: statusResponse.data.formats_count
              })
              // Démarrer le polling
              startStatusPolling(id)
            } else {
              throw err
            }
          } catch (statusErr) {
            // Si le statut n'est pas accessible non plus, afficher l'erreur
            throw err
          }
        } else {
          throw err
        }
      }
    } catch (err) {
      console.error('Erreur lors de la récupération de la vidéo:', err)
      const errorMessage = err.response?.data?.error || err.message || 'Vidéo non trouvée'
      setError(errorMessage)
      
      // Si la vidéo est en cours de traitement, afficher un message spécifique
      if (err.response?.status === 404) {
        setError('Cette vidéo est en cours de traitement ou n\'existe pas encore. Veuillez réessayer dans quelques instants.')
      }
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
              {video.views.toLocaleString()} vues
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

