import React, { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import api from '../services/api'
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
      setHistory(response.data || [])
    } catch (error) {
      console.error('Erreur lors de la récupération de l\'historique:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div className="history-page">Chargement...</div>
  }

  return (
    <div className="history-page">
      <div className="history-header">
        <h1>📺 Mon Historique</h1>
        <p>Vos vidéos récemment regardées</p>
      </div>

      {history.length === 0 ? (
        <div className="empty-state">
          <p>Vous n'avez pas encore regardé de vidéos</p>
          <Link to="/" className="btn-primary">Découvrir des vidéos</Link>
        </div>
      ) : (
        <div className="history-grid">
          {history.map((item) => (
            <Link key={item.id} to={`/video/${item.video_id}`} className="history-item">
              {item.thumbnail && (
                <img src={item.thumbnail} alt={item.title} className="history-thumbnail" />
              )}
              <div className="history-info">
                <h3>{item.title || 'Vidéo'}</h3>
                <p className="history-date">
                  Regardé le {new Date(item.watched_at).toLocaleDateString('fr-FR')}
                </p>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

export default History

