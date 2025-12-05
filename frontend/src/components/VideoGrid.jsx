import React from 'react'
import { Link } from 'react-router-dom'
import './VideoGrid.css'

const VideoGrid = ({ videos, title }) => {
  if (!videos || videos.length === 0) {
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
        {videos.map((video) => (
          <Link key={video.id} to={`/video/${video.id}`} className="video-card">
            <div className="video-thumbnail">
              {video.thumbnail_path ? (
                <img
                  src={`/thumbnails/${video.thumbnail_path.split('/').pop()}`}
                  alt={video.title}
                  onError={(e) => {
                    e.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'
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
        ))}
      </div>
    </div>
  )
}

const formatDuration = (seconds) => {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

const formatViews = (views) => {
  if (views >= 1000000) {
    return `${(views / 1000000).toFixed(1)}M`
  }
  if (views >= 1000) {
    return `${(views / 1000).toFixed(1)}K`
  }
  return views.toString()
}

export default VideoGrid

