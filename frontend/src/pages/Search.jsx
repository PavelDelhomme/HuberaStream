import React, { useState, useEffect } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import VideoGrid from '../components/VideoGrid'
import api from '../services/api'
import { filterValidVideos } from '../utils/videoUtils'
import LoadingSpinner from '../components/LoadingSpinner'
import './Search.css'

const Search = () => {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const query = searchParams.get('q') || ''
  const [videos, setVideos] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    // Si pas de query, rediriger vers la page d'accueil
    if (!query) {
      navigate('/')
      return
    }
    performSearch(query)
  }, [query, navigate])

  const performSearch = async (searchQuery) => {
    try {
      setLoading(true)
      setError(null)
      const response = await api.get('/videos', {
        params: {
          search: searchQuery,
          page: 1,
          limit: 24,
        },
      })
      setVideos(filterValidVideos(response.data.videos || []))
    } catch (err) {
      console.error('Erreur lors de la recherche:', err)
      setError('Erreur lors de la recherche')
    } finally {
      setLoading(false)
    }
  }

  // Ne rien afficher si pas de query (redirection en cours)
  if (!query) {
    return null
  }

  if (loading) {
    return (
      <div className="search-container">
        <LoadingSpinner message="Recherche en cours..." />
      </div>
    )
  }

  if (error) {
    return (
      <div className="search-container">
        <div className="error">{error}</div>
      </div>
    )
  }

  return (
    <div className="search-container">
      <div className="search-results-header">
        <h2>Résultats pour "{query}"</h2>
        <p className="results-count">{videos.length} vidéo(s) trouvée(s)</p>
      </div>
      <VideoGrid videos={videos} />
    </div>
  )
}

export default Search

