'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminMediaSettings.css'

export default function AdminMediaSettingsPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [settings, setSettings] = useState({
    video_storage_path: '/app/storage/videos',
    thumbnail_storage_path: '/app/storage/thumbnails',
    default_video_quality: '720p',
    enable_hls: true,
    enable_dash: false,
    hls_segment_duration: 10,
    max_concurrent_transcodes: 2,
    ffmpeg_preset: 'medium',
    ffmpeg_crf: 23
  })

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/media-settings')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchSettings()
    }
    checkAuth()
  }, [router])

  const fetchSettings = async () => {
    try {
      const response = await api.get('/admin/media-settings')
      if (response.data) {
        setSettings((prev) => ({ ...prev, ...response.data }))
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
        <div className="loading">Chargement des paramètres...</div>
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
        </div>

        <div className="settings-section">
          <h2>⚙️ Transcodage</h2>
          <div className="form-group">
            <label>Qualité vidéo par défaut</label>
            <select
              value={settings.default_video_quality}
              onChange={(e) => setSettings({ ...settings, default_video_quality: e.target.value })}
            >
              <option value="480p">480p</option>
              <option value="720p">720p</option>
              <option value="1080p">1080p</option>
              <option value="2160p">2160p (4K)</option>
            </select>
          </div>
          <div className="form-group">
            <label>
              <input
                type="checkbox"
                checked={settings.enable_hls}
                onChange={(e) => setSettings({ ...settings, enable_hls: e.target.checked })}
              />
              Activer HLS
            </label>
          </div>
          <div className="form-group">
            <label>
              <input
                type="checkbox"
                checked={settings.enable_dash}
                onChange={(e) => setSettings({ ...settings, enable_dash: e.target.checked })}
              />
              Activer DASH
            </label>
          </div>
          <div className="form-group">
            <label>Durée des segments HLS (secondes)</label>
            <input
              type="number"
              value={settings.hls_segment_duration}
              onChange={(e) => setSettings({ ...settings, hls_segment_duration: parseInt(e.target.value) })}
              min="5"
              max="30"
            />
          </div>
          <div className="form-group">
            <label>Transcodages simultanés maximum</label>
            <input
              type="number"
              value={settings.max_concurrent_transcodes}
              onChange={(e) => setSettings({ ...settings, max_concurrent_transcodes: parseInt(e.target.value) })}
              min="1"
              max="10"
            />
          </div>
          <div className="form-group">
            <label>Prését FFmpeg</label>
            <select
              value={settings.ffmpeg_preset}
              onChange={(e) => setSettings({ ...settings, ffmpeg_preset: e.target.value })}
            >
              <option value="ultrafast">Ultrafast</option>
              <option value="veryfast">Veryfast</option>
              <option value="faster">Faster</option>
              <option value="fast">Fast</option>
              <option value="medium">Medium</option>
              <option value="slow">Slow</option>
              <option value="slower">Slower</option>
            </select>
          </div>
          <div className="form-group">
            <label>CRF FFmpeg (qualité)</label>
            <input
              type="number"
              value={settings.ffmpeg_crf}
              onChange={(e) => setSettings({ ...settings, ffmpeg_crf: parseInt(e.target.value) })}
              min="18"
              max="28"
            />
          </div>
        </div>
      </div>

      <div className="settings-actions">
        <button onClick={handleSave} className="save-button">
          💾 Sauvegarder
        </button>
      </div>
    </div>
  )
}

