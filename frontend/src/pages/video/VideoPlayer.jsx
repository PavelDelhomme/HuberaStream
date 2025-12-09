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
  }, [id])

  const fetchVideo = async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await api.get(`/videos/${id}`)
      
      // Vérifier que la vidéo existe et a un ID valide
      if (!response.data || !response.data.id) {
        throw new Error('Vidéo invalide')
      }
      
      setVideo(response.data)
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
        <div className="error">{error || 'Vidéo non trouvée'}</div>
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

