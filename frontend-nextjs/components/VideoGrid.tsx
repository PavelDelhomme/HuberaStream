'use client'

import Link from 'next/link'
import './VideoGrid.css'

interface Video {
  id: string
  title: string
  description?: string
  thumbnail_path?: string
  thumbnail_url?: string
  duration: number
  views: number
  created_at: string
  user?: {
    username: string
  }
  watchPosition?: number
}

interface VideoGridProps {
  videos: Video[]
  title?: string
}

export default function VideoGrid({ videos, title }: VideoGridProps) {
  // Filtrer les vidéos invalides
  const validVideos = (videos || []).filter(video => {
    if (!video || !video.id) return false
    if (video.title === 'null' || video.title === 'Null' || video.title === 'NULL') return false
    if (video.title && typeof video.title === 'string' && video.title.trim() === '' && video.status !== 'processing') return false
    return true
  })

  const formatDuration = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`
  }

  const formatViews = (views: number): string => {
    if (views >= 1000000) {
      return `${(views / 1000000).toFixed(1)}M`
    }
    if (views >= 1000) {
      return `${(views / 1000).toFixed(1)}K`
    }
    return views.toString()
  }

  if (validVideos.length === 0) {
    return (
      <div className="video-grid-container">
        {title && <h2 className="section-title">{title}</h2>}
        <p className="no-videos">Aucune vidéo disponible</p>
      </div>
    )
  }

  return (
    <div className="video-grid-container">
      {title && <h2 className="section-title">{title}</h2>}
      <div className="video-grid">
        {validVideos.map((video) => {
          const thumbnailUrl = video.thumbnail_url || 
            (video.thumbnail_path ? `/thumbnails/${video.thumbnail_path.split('/').pop()}` : null)
          
          return (
            <Link key={video.id} href={`/video/${video.id}`} className="video-card" prefetch={false}>
              <div className="video-thumbnail">
                {thumbnailUrl ? (
                  <img
                    src={thumbnailUrl}
                    alt={video.title}
                    onError={(e) => {
                      e.currentTarget.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'
                    }}
                  />
                ) : (
                  <div className="thumbnail-placeholder">
                    <span>📹</span>
                  </div>
                )}
                {video.duration > 0 && (
                  <span className="video-duration">
                    {formatDuration(video.duration)}
                  </span>
                )}
                {video.watchPosition && video.watchPosition > 0 && (
                  <div className="watch-progress">
                    <div 
                      className="watch-progress-bar" 
                      style={{ width: `${(video.watchPosition / video.duration) * 100}%` }}
                    ></div>
                  </div>
                )}
              </div>
              <div className="video-info">
                <h3 className="video-title">{video.title}</h3>
                <div className="video-meta">
                  <span className="video-views">{formatViews(video.views)} vues</span>
                  {video.user && (
                    <span className="video-author">{video.user.username}</span>
                  )}
                </div>
              </div>
            </Link>
          )
        })}
      </div>
    </div>
  )
}

