'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter } from 'next/navigation'
import ReactPlayer from 'react-player'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import Link from 'next/link'
import './VideoPlayerPage.css'

interface Video {
  id: string
  title: string
  description?: string
  status: string
  thumbnail_path?: string
  formats?: any[]
  progress?: number
  formats_count?: number
  views?: number
  user?: any
  created_at?: string
}

export default function VideoPlayerContent({ videoId }: { videoId: string }) {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [video, setVideo] = useState<Video | null>(null)
  const [relatedVideos, setRelatedVideos] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [fetchingVideo, setFetchingVideo] = useState(false)
  const [currentFormat, setCurrentFormat] = useState('progressive')
  const [currentQuality, setCurrentQuality] = useState<string | null>(null)
  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    const loadUser = async () => {
      if (authService.isAuthenticated()) {
        const currentUser = await authService.getCurrentUser()
        setUser(currentUser)
      }
    }
    loadUser()
    
    if (videoId && !fetchingVideo) {
      fetchVideo()
      fetchRelatedVideos()
    }

    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current)
      }
    }
  }, [videoId])

  const fetchVideo = async () => {
    if (fetchingVideo) return

    try {
      setFetchingVideo(true)
      setLoading(true)
      setError(null)

      try {
        const response = await api.get(`/videos/${videoId}`)
        
        if (!response.data || !response.data.id) {
          throw new Error('Vidéo invalide')
        }
        
        setVideo(response.data)
        
        if (response.data.status === 'processing') {
          startStatusPolling(videoId)
        }
        
        if (response.data.status === 'error') {
          setError('Cette vidéo est invalide ou a rencontré une erreur lors du traitement.')
        }
      } catch (err: any) {
        if (err.response?.status === 404) {
          try {
            const statusResponse = await api.get(`/videos/${videoId}/status`)
            if (statusResponse.data) {
              const videoData: Video = {
                id: statusResponse.data.video_id || videoId,
                title: statusResponse.data.title || 'Vidéo en traitement',
                description: statusResponse.data.description || '',
                status: statusResponse.data.status,
                thumbnail_path: statusResponse.data.thumbnail_path,
                formats: [],
                progress: statusResponse.data.progress || 0,
                formats_count: statusResponse.data.formats_count || 0,
                views: 0,
                user: null,
                created_at: new Date().toISOString()
              }
              
              if (statusResponse.data.status === 'error') {
                setError('Cette vidéo est invalide ou a rencontré une erreur lors du traitement.')
              }
              
              setVideo(videoData)
              setLoading(false)
              
              if (statusResponse.data.status === 'processing') {
                startStatusPolling(videoId)
              }
              return
            }
          } catch (statusErr) {
            console.error('Erreur lors de la récupération du statut:', statusErr)
          }
        }
        throw err
      }
    } catch (err: any) {
      console.error('Erreur lors de la récupération de la vidéo:', err)
      const errorMessage = err.response?.data?.error || err.message || 'Vidéo non trouvée'
      setError(errorMessage)
    } finally {
      setLoading(false)
      setFetchingVideo(false)
    }
  }

  const startStatusPolling = (videoId: string) => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current)
    }

    let pollCount = 0
    pollIntervalRef.current = setInterval(async () => {
      pollCount++
      try {
        const statusResponse = await api.get(`/videos/${videoId}/status`)
        if (statusResponse.data) {
          setVideo((prev) => ({
            ...prev!,
            status: statusResponse.data.status,
            progress: statusResponse.data.progress,
            formats_count: statusResponse.data.formats_count,
            thumbnail_path: statusResponse.data.thumbnail_path || prev?.thumbnail_path,
            title: statusResponse.data.title || prev?.title,
            description: statusResponse.data.description || prev?.description
          }))

          if (statusResponse.data.status === 'ready') {
            if (pollIntervalRef.current) {
              clearInterval(pollIntervalRef.current)
            }
            setVideo((prev) => ({
              ...prev!,
              status: 'ready',
              progress: 100
            }))
            setLoading(false)
            
            let retryCount = 0
            const maxRetries = 3
            while (retryCount < maxRetries) {
              try {
                const fullResponse = await api.get(`/videos/${videoId}`)
                if (fullResponse.data) {
                  setVideo(fullResponse.data)
                  break
                }
              } catch (err) {
                retryCount++
                if (retryCount < maxRetries) {
                  await new Promise(resolve => setTimeout(resolve, 1000 * retryCount))
                }
              }
            }
          } else if (statusResponse.data.status === 'error') {
            if (pollIntervalRef.current) {
              clearInterval(pollIntervalRef.current)
            }
            setError('Erreur lors du traitement de la vidéo.')
          }
        }
      } catch (err) {
        console.error('Erreur lors du polling:', err)
      }
    }, 5000)
  }

  const fetchRelatedVideos = async () => {
    try {
      const response = await api.get('/videos', {
        params: { page: 1, limit: 12 }
      })
      const videos = response.data.videos || []
      const filteredVideos = videos.filter((v: any) => v.id !== videoId)
      setRelatedVideos(filteredVideos.slice(0, 12))
    } catch (err) {
      console.error('Erreur lors de la récupération des vidéos similaires:', err)
      setRelatedVideos([])
    }
  }

  const handleTimeUpdate = async (position: number) => {
    if (user && video) {
      try {
        await api.post(`/history/${video.id}`, { position })
      } catch (err) {
        console.error('Erreur lors de la mise à jour de l\'historique:', err)
      }
    }
  }

  const getVideoUrl = () => {
    if (!video?.formats || video.formats.length === 0) {
      return null
    }

    const format = video.formats.find(
      (f: any) => f.format === currentFormat && (!currentQuality || f.quality === currentQuality)
    ) || video.formats.find((f: any) => f.format === currentFormat) || video.formats[0]

    if (currentFormat === 'hls' && format.manifest_path) {
      return `/api/videos/${video.id}/stream?format=hls&quality=${format.quality || ''}`
    } else if (currentFormat === 'dash' && format.manifest_path) {
      return `/api/videos/${video.id}/stream?format=dash&quality=${format.quality || ''}`
    } else if (currentFormat === 'progressive') {
      return `/api/videos/${video.id}/stream?format=progressive&quality=${format.quality || ''}`
    }

    return null
  }

  if (loading) {
    return (
      <div className="video-player-page">
        <div className="loading">Chargement de la vidéo...</div>
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

  if (video.status === 'error' || video.status === 'cancelled') {
    return (
      <div className="video-player-page">
        <div className="error-container">
          <div className="error-icon">❌</div>
          <div className="error-message">
            {video.status === 'error' 
              ? 'Cette vidéo est invalide ou a rencontré une erreur lors du traitement.'
              : 'Le traitement de cette vidéo a été annulé.'}
          </div>
        </div>
      </div>
    )
  }

  if (video.status === 'processing') {
    return (
      <div className="video-player-page">
        <div className="processing-container">
          <div className="processing-icon">⏳</div>
          <h2>Vidéo en cours de traitement</h2>
          <div className="progress-bar-container">
            <div className="progress-bar" style={{ width: `${video.progress || 0}%` }}></div>
          </div>
          <p>{video.progress || 0}% - {video.formats_count || 0} format(s) généré(s)</p>
        </div>
      </div>
    )
  }

  const videoUrl = getVideoUrl()

  return (
    <div className="video-player-page">
      <div className="video-container">
        <div className="video-player-wrapper">
          {videoUrl ? (
            <ReactPlayer
              url={videoUrl}
              controls
              width="100%"
              height="100%"
              onProgress={(state) => handleTimeUpdate(Math.floor(state.playedSeconds))}
              config={{
                file: {
                  attributes: {
                    controlsList: 'nodownload',
                  },
                  hlsOptions: {
                    xhrSetup: (xhr: any) => {
                      const token = authService.getToken()
                      if (token) {
                        xhr.setRequestHeader('Authorization', `Bearer ${token}`)
                      }
                    },
                  },
                },
              }}
            />
          ) : (
            <div className="video-error">Vidéo non disponible</div>
          )}
        </div>

        <div className="video-info">
          <h1>{video.title}</h1>
          {video.description && <p className="video-description">{video.description}</p>}
          
          {video.formats && video.formats.length > 0 && (
            <div className="quality-selector">
              <label>Qualité:</label>
              <select 
                value={currentQuality || ''} 
                onChange={(e) => setCurrentQuality(e.target.value || null)}
              >
                {video.formats.map((format: any) => (
                  <option key={`${format.format}-${format.quality}`} value={format.quality}>
                    {format.quality} ({format.format})
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>
      </div>

      {relatedVideos.length > 0 && (
        <div className="related-videos">
          <h2>Vidéos similaires</h2>
          <div className="related-videos-grid">
            {relatedVideos.map((relatedVideo) => (
              <Link key={relatedVideo.id} href={`/video/${relatedVideo.id}`} className="related-video-item">
                {relatedVideo.thumbnail_path ? (
                  <img 
                    src={`/api/thumbnails/${relatedVideo.thumbnail_path.split('/').pop()}`} 
                    alt={relatedVideo.title}
                  />
                ) : (
                  <div className="thumbnail-placeholder">📺</div>
                )}
                <h3>{relatedVideo.title}</h3>
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

