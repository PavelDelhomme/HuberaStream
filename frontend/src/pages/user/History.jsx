import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './History.css'

const History = () => {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [history, setHistory] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    fetchHistory()
  }, [user, navigate])

  const fetchHistory = async () => {
    try {
      const response = await api.get('/history')
      // L'API peut retourner un tableau ou un objet avec une propriété videos
      const historyData = Array.isArray(response.data) 
        ? response.data 
        : (response.data?.videos || response.data?.history || [])
      
      // Mapper les données pour avoir un format cohérent
      const mappedHistory = historyData.map(item => {
        const video = item.Video || item.video || item
        return {
          id: item.id || video?.id,
          video_id: item.video_id || video?.id,
          title: video?.title || item.title,
          thumbnail: video?.thumbnail_path || item.thumbnail,
          watched_at: item.watched_at || item.updated_at || item.created_at,
          position: item.position || item.Position || 0
        }
      })
      
      setHistory(mappedHistory)
    } catch (error) {
      console.error('Erreur lors de la récupération de l\'historique:', error)
      setHistory([])
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="history-page">
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Chargement de l'historique...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="history-page">
      <div className="history-header">
        <h1>📺 Mon Historique</h1>
        <p>Vos vidéos récemment regardées</p>
      </div>

      {history.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">📺</div>
          <p>Vous n'avez pas encore regardé de vidéos</p>
          <p className="empty-hint">Commencez à regarder des vidéos pour les voir apparaître ici</p>
          <Link to="/" className="btn-primary">Découvrir des vidéos</Link>
        </div>
      ) : (
        <div className="history-grid">
          {history.map((item) => (
            <Link key={item.id || item.video_id} to={`/video/${item.video_id}`} className="history-item">
              {item.thumbnail ? (
                <img 
                  src={`/thumbnails/${item.thumbnail.split('/').pop()}`} 
                  alt={item.title || 'Vidéo'} 
                  className="history-thumbnail"
                  onError={(e) => {
                    e.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'
                  }}
                />
              ) : (
                <div className="history-thumbnail-placeholder">
                  <span>📹</span>
                </div>
              )}
              <div className="history-info">
                <h3>{item.title || 'Vidéo'}</h3>
                <p className="history-date">
                  Regardé le {item.watched_at ? new Date(item.watched_at).toLocaleDateString('fr-FR', {
                    day: 'numeric',
                    month: 'long',
                    year: 'numeric'
                  }) : 'Date inconnue'}
                </p>
                {item.position > 0 && (
                  <p className="history-progress">
                    Progression: {Math.round((item.position / 100) * 100)}%
                  </p>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

export default History

