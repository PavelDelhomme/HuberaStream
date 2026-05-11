'use client'

import { useState, useRef, useEffect } from 'react'
import Link from 'next/link'
import './VideoCardWithPreview.css'

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
  year?: number
  director?: string
  rating?: number
  genres?: string[]
  platform?: string
}

interface VideoCardWithPreviewProps {
  video: Video
  onHover?: (video: Video | null) => void
}

export default function VideoCardWithPreview({ video, onHover }: VideoCardWithPreviewProps) {
  const [isHovered, setIsHovered] = useState(false)
  const [showPreview, setShowPreview] = useState(false)
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const cardRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    return () => {
      if (hoverTimeoutRef.current) {
        clearTimeout(hoverTimeoutRef.current)
      }
    }
  }, [])

  const handleMouseEnter = () => {
    setIsHovered(true)
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
    }
    hoverTimeoutRef.current = setTimeout(() => {
      setShowPreview(true)
      if (onHover) {
        onHover(video)
      }
    }, 500) // Délai de 500ms avant d'afficher l'aperçu
  }

  const handleMouseLeave = () => {
    setIsHovered(false)
    setShowPreview(false)
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
    }
    if (onHover) {
      onHover(null)
    }
  }

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

  const thumbnailUrl = video.thumbnail_url || 
    (video.thumbnail_path ? `/api/thumbnails/${video.thumbnail_path.split('/').pop()}` : null)

  return (
    <div 
      ref={cardRef}
      className={`video-card-preview ${isHovered ? 'hovered' : ''}`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      <Link href={`/video/${video.id}`} className="video-card-link" prefetch={false}>
        <div className="video-thumbnail-container">
          {thumbnailUrl ? (
            <img
              src={thumbnailUrl}
              alt={video.title}
              className="video-thumbnail"
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
        <div className="video-info-compact">
          <h3 className="video-title">{video.title}</h3>
          <div className="video-meta-compact">
            <span className="video-views">{formatViews(video.views)} vues</span>
          </div>
        </div>
      </Link>

      {/* Aperçu au survol (style Netflix) */}
      {showPreview && isHovered && (
        <div className="video-preview-card">
          <div className="preview-thumbnail">
            {thumbnailUrl ? (
              <img
                src={thumbnailUrl}
                alt={video.title}
                onError={(e) => {
                  e.currentTarget.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'
                }}
              />
            ) : (
              <div className="preview-placeholder">📹</div>
            )}
            {video.duration > 0 && (
              <span className="preview-duration">{formatDuration(video.duration)}</span>
            )}
          </div>
          <div className="preview-info">
            <div className="preview-actions">
              <Link href={`/video/${video.id}`} className="preview-play-button">
                ▶️ Lecture
              </Link>
              <button className="preview-info-button" title="Plus d'infos">
                ℹ️
              </button>
            </div>
            <h3 className="preview-title">{video.title}</h3>
            {video.description && (
              <p className="preview-description">
                {video.description.length > 150 
                  ? `${video.description.substring(0, 150)}...` 
                  : video.description}
              </p>
            )}
            <div className="preview-meta">
              {video.year && <span className="preview-year">{video.year}</span>}
              {video.rating && (
                <span className="preview-rating">⭐ {video.rating}/10</span>
              )}
              {video.platform && (
                <span className="preview-platform">📺 {video.platform}</span>
              )}
              {video.genres && video.genres.length > 0 && (
                <span className="preview-genres">
                  {video.genres.slice(0, 2).join(', ')}
                </span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

