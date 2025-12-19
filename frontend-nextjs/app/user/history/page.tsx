'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { authService } from '@/lib/auth'
import api from '@/lib/api'
import './History.css'

export default function HistoryPage() {
  const router = useRouter()
  const [history, setHistory] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const loadHistory = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login')
        return
      }
      fetchHistory()
    }
    loadHistory()
  }, [router])

  const fetchHistory = async () => {
    try {
      const response = await api.get('/history')
      const historyData = Array.isArray(response.data) 
        ? response.data 
        : (response.data?.videos || response.data?.history || [])
      
      const mappedHistory = historyData.map((item: any) => {
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
        <div className="loading">Chargement de l'historique...</div>
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
          <Link href="/" className="btn-primary">Découvrir des vidéos</Link>
        </div>
      ) : (
        <div className="history-grid">
          {history.map((item) => (
            <Link key={item.id || item.video_id} href={`/video/${item.video_id}`} className="history-item">
              {item.thumbnail ? (
                <img 
                  src={`/api/thumbnails/${item.thumbnail.split('/').pop()}`} 
                  alt={item.title}
                  className="history-thumbnail"
                />
              ) : (
                <div className="history-thumbnail-placeholder">📺</div>
              )}
              <div className="history-info">
                <h3>{item.title}</h3>
                <p className="history-date">
                  Regardé le {new Date(item.watched_at).toLocaleDateString('fr-FR')}
                </p>
                {item.position > 0 && (
                  <p className="history-position">
                    Position: {Math.floor(item.position / 60)}min {item.position % 60}s
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

