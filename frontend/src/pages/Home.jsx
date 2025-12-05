import React, { useState, useEffect } from 'react'
import VideoGrid from '../components/VideoGrid'
import api from '../services/api'
import './Home.css'

const Home = () => {
  const [videos, setVideos] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchVideos()
  }, [])

  const fetchVideos = async () => {
    try {
      setLoading(true)
      const response = await api.get('/videos', {
        params: {
          page: 1,
          limit: 24,
        },
      })
      setVideos(response.data.videos || [])
      setError(null)
    } catch (err) {
      console.error('Erreur lors de la récupération des vidéos:', err)
      setError('Impossible de charger les vidéos')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="home-container">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="home-container">
        <div className="error">{error}</div>
      </div>
    )
  }

  return (
    <div className="home-container">
      <VideoGrid videos={videos} title="Vidéos récentes" />
    </div>
  )
}

export default Home

