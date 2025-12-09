import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminMediaSettings.css'

const AdminMediaSettings = () => {
  const { user, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [settings, setSettings] = useState({
    video_storage_path: '/app/storage/videos',
    thumbnail_storage_path: '/app/storage/thumbnails',
    torrent_download_path: '/app/storage/torrents/downloads',
    enable_auto_download: false,
    default_video_quality: '720p',
    enable_hls: true,
    enable_dash: false,
    hls_segment_duration: 10,
    max_concurrent_transcodes: 2,
    ffmpeg_preset: 'medium',
    ffmpeg_crf: 23
  })

  useEffect(() => {
    if (authLoading) return
    if (!user || user.role !== 'admin') {
      navigate('/login')
      return
    }
    fetchSettings()
  }, [user, authLoading, navigate])

  const fetchSettings = async () => {
    try {
      const response = await api.get('/admin/media-settings')
      if (response.data) {
        setSettings(prev => ({ ...prev, ...response.data }))
      }
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    try {
      await api.put('/admin/media-settings', settings)
      alert('Paramètres sauvegardés avec succès !')
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la sauvegarde')
    }
  }

  if (loading) {
    return (
      <div className="admin-media-settings">
        <LoadingSpinner message="Chargement des paramètres..." />
      </div>
    )
  }

  return (
    <div className="admin-media-settings">
      <div className="admin-header">
        <h1>🎬 Paramètres Média</h1>
        <p className="subtitle">Configurez les paramètres de traitement vidéo et de stockage</p>
      </div>

      <div className="settings-sections">
        {/* Stockage */}
        <div className="settings-section">
          <h2>📁 Stockage</h2>
          <div className="form-group">
            <label>Chemin de stockage des vidéos</label>
            <input
              type="text"
              value={settings.video_storage_path}
              onChange={(e) => setSettings({ ...settings, video_storage_path: e.target.value })}
            />
          </div>
          <div className="form-group">
            <label>Chemin de stockage des miniatures</label>
            <input
              type="text"
              value={settings.thumbnail_storage_path}
              onChange={(e) => setSettings({ ...settings, thumbnail_storage_path: e.target.value })}
            />
          </div>
          <div className="form-group">
            <label>Chemin de téléchargement des torrents</label>
            <input
              type="text"
              value={settings.torrent_download_path}
              onChange={(e) => setSettings({ ...settings, torrent_download_path: e.target.value })}
            />
          </div>
        </div>

        {/* Transcodage */}
        <div className="settings-section">
          <h2>🎥 Transcodage Vidéo</h2>
          <div className="form-group">
            <label>Qualité vidéo par défaut</label>
            <select
              value={settings.default_video_quality}
              onChange={(e) => setSettings({ ...settings, default_video_quality: e.target.value })}
            >
              <option value="1080p">1080p</option>
              <option value="720p">720p</option>
              <option value="480p">480p</option>
              <option value="360p">360p</option>
            </select>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>FFmpeg Preset</label>
              <select
                value={settings.ffmpeg_preset}
                onChange={(e) => setSettings({ ...settings, ffmpeg_preset: e.target.value })}
              >
                <option value="ultrafast">Ultrafast</option>
                <option value="superfast">Superfast</option>
                <option value="veryfast">Veryfast</option>
                <option value="faster">Faster</option>
                <option value="fast">Fast</option>
                <option value="medium">Medium</option>
                <option value="slow">Slow</option>
                <option value="slower">Slower</option>
                <option value="veryslow">Veryslow</option>
              </select>
            </div>
            <div className="form-group">
              <label>FFmpeg CRF (18-28, plus bas = meilleure qualité)</label>
              <input
                type="number"
                min="18"
                max="28"
                value={settings.ffmpeg_crf}
                onChange={(e) => setSettings({ ...settings, ffmpeg_crf: parseInt(e.target.value) })}
              />
            </div>
          </div>
          <div className="form-group">
            <label>Transcodages simultanés maximum</label>
            <input
              type="number"
              min="1"
              max="8"
              value={settings.max_concurrent_transcodes}
              onChange={(e) => setSettings({ ...settings, max_concurrent_transcodes: parseInt(e.target.value) })}
            />
          </div>
        </div>

        {/* Formats de streaming */}
        <div className="settings-section">
          <h2>📺 Formats de Streaming</h2>
          <div className="form-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={settings.enable_hls}
                onChange={(e) => setSettings({ ...settings, enable_hls: e.target.checked })}
              />
              Activer HLS (HTTP Live Streaming)
            </label>
          </div>
          <div className="form-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={settings.enable_dash}
                onChange={(e) => setSettings({ ...settings, enable_dash: e.target.checked })}
              />
              Activer DASH (Dynamic Adaptive Streaming)
            </label>
          </div>
          {settings.enable_hls && (
            <div className="form-group">
              <label>Durée des segments HLS (secondes)</label>
              <input
                type="number"
                min="5"
                max="30"
                value={settings.hls_segment_duration}
                onChange={(e) => setSettings({ ...settings, hls_segment_duration: parseInt(e.target.value) })}
              />
            </div>
          )}
        </div>

        {/* Téléchargements automatiques */}
        <div className="settings-section">
          <h2>📥 Téléchargements</h2>
          <div className="form-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={settings.enable_auto_download}
                onChange={(e) => setSettings({ ...settings, enable_auto_download: e.target.checked })}
              />
              Activer les téléchargements automatiques depuis les configurations torrent
            </label>
          </div>
        </div>

        <div className="form-actions">
          <button onClick={handleSave} className="save-button">
            💾 Sauvegarder les paramètres
          </button>
        </div>
      </div>
    </div>
  )
}

export default AdminMediaSettings
