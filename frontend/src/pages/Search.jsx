import React, { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import VideoGrid from '../components/VideoGrid'
import api from '../services/api'
import './Search.css'

const Search = () => {
  const [searchParams] = useSearchParams()
  const query = searchParams.get('q') || ''
  const [videos, setVideos] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    if (query) {
      performSearch(query)
    }
  }, [query])

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
      setVideos(response.data.videos || [])
    } catch (err) {
      console.error('Erreur lors de la recherche:', err)
      setError('Erreur lors de la recherche')
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="search-container">
        <div className="loading">Recherche en cours...</div>
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
      {query && (
        <div className="search-results-header">
          <h2>Résultats pour "{query}"</h2>
          <p className="results-count">{videos.length} vidéo(s) trouvée(s)</p>
        </div>
      )}
      {!query && (
        <div className="search-empty">
          <p>Entrez un terme de recherche dans la barre de navigation</p>
        </div>
      )}
      <VideoGrid videos={videos} />
    </div>
  )
}

export default Search

