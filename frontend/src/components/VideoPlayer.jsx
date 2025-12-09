import React, { useRef, useEffect, useState } from 'react'
import ReactPlayer from 'react-player'
import './VideoPlayer.css'

const VideoPlayer = ({ video, formats, subtitles, onTimeUpdate }) => {
  const playerRef = useRef(null)
  const [currentFormat, setCurrentFormat] = useState('progressive')
  const [currentQuality, setCurrentQuality] = useState(null)
  const [currentSubtitle, setCurrentSubtitle] = useState(null)
  const [showSubtitles, setShowSubtitles] = useState(false)

  useEffect(() => {
    // Déterminer le meilleur format disponible
    if (formats && formats.length > 0) {
      const hlsFormat = formats.find(f => f.format === 'hls')
      const dashFormat = formats.find(f => f.format === 'dash')
      const progressiveFormat = formats.find(f => f.format === 'progressive')

      if (hlsFormat) {
        setCurrentFormat('hls')
        setCurrentQuality(hlsFormat.quality)
      } else if (dashFormat) {
        setCurrentFormat('dash')
        setCurrentQuality(dashFormat.quality)
      } else if (progressiveFormat) {
        setCurrentFormat('progressive')
        setCurrentQuality(progressiveFormat.quality)
      }
    }
  }, [formats])

  const getVideoUrl = () => {
    if (!formats || formats.length === 0) {
      return null
    }

    const format = formats.find(
      f => f.format === currentFormat && (!currentQuality || f.quality === currentQuality)
    ) || formats.find(f => f.format === currentFormat) || formats[0]

    if (currentFormat === 'hls' && format.manifest_path) {
      return `/api/videos/${video.id}/stream?format=hls&quality=${format.quality || ''}`
    } else if (currentFormat === 'dash' && format.manifest_path) {
      return `/api/videos/${video.id}/stream?format=dash&quality=${format.quality || ''}`
    } else if (currentFormat === 'progressive') {
      return `/api/videos/${video.id}/stream?format=progressive&quality=${format.quality || ''}`
    }

    return null
  }

  const handleProgress = (state) => {
    if (onTimeUpdate) {
      onTimeUpdate(Math.floor(state.playedSeconds))
    }
  }

  const videoUrl = getVideoUrl()

  if (!videoUrl) {
    return (
      <div className="video-player-container">
        <div className="video-player-error">
          <p>Vidéo non disponible</p>
          <p className="error-details">Aucun format de vidéo n'est disponible pour cette vidéo.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="video-player-container">
      <div className="video-player-wrapper">
        <ReactPlayer
          ref={playerRef}
          url={videoUrl}
          controls
          width="100%"
          height="100%"
          onProgress={handleProgress}
          config={{
            file: {
              attributes: {
                controlsList: 'nodownload',
              },
              tracks: subtitles && subtitles.length > 0 ? subtitles.map((sub, index) => ({
                kind: 'subtitles',
                src: `/api/videos/${video.id}/subtitles/${sub.id}`,
                srcLang: sub.language,
                label: sub.language_name || sub.language,
                default: sub.is_default || index === 0,
              })) : undefined,
              hlsOptions: {
                xhrSetup: (xhr, url) => {
                  const token = localStorage.getItem('token')
                  if (token) {
                    xhr.setRequestHeader('Authorization', `Bearer ${token}`)
                  }
                },
              },
            },
          }}
        />
      </div>

      <div className="video-controls-bar">
        {formats && formats.length > 1 && (
          <div className="video-quality-selector">
            <label htmlFor="quality-select">Qualité: </label>
            <select
              id="quality-select"
              value={currentQuality || ''}
              onChange={(e) => {
                const quality = e.target.value
                setCurrentQuality(quality)
                const format = formats.find(f => f.quality === quality && f.format === currentFormat)
                if (format) {
                  setCurrentFormat(format.format)
                }
              }}
            >
              {formats
                .filter(f => f.format === currentFormat)
                .map((format) => (
                  <option key={format.id} value={format.quality || ''}>
                    {format.quality || 'Auto'}
                  </option>
                ))}
            </select>
          </div>
        )}
        
        {subtitles && subtitles.length > 0 && (
          <div className="video-subtitle-selector">
            <label htmlFor="subtitle-select">Sous-titres: </label>
            <select
              id="subtitle-select"
              value={currentSubtitle || 'none'}
              onChange={(e) => {
                const subId = e.target.value
                setCurrentSubtitle(subId === 'none' ? null : subId)
                setShowSubtitles(subId !== 'none')
                // Mettre à jour les tracks du lecteur
                if (playerRef.current && playerRef.current.getInternalPlayer) {
                  const player = playerRef.current.getInternalPlayer()
                  if (player && player.textTracks) {
                    for (let i = 0; i < player.textTracks.length; i++) {
                      const track = player.textTracks[i]
                      track.mode = subId === 'none' ? 'hidden' : (track.id === subId ? 'showing' : 'hidden')
                    }
                  }
                }
              }}
            >
              <option value="none">Aucun</option>
              {subtitles.map((sub) => (
                <option key={sub.id} value={sub.id}>
                  {sub.language_name || sub.language}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>
    </div>
  )
}

export default VideoPlayer

