'use client'

import { useState, useEffect } from 'react'
import { useSearchParams } from 'next/navigation'
import api from '@/lib/api'
import './Search.css'

export default function SearchContent() {
  const searchParams = useSearchParams()
  const query = searchParams.get('q') || ''
  const [searchQuery, setSearchQuery] = useState(query)
  const [results, setResults] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (query) {
      performSearch(query)
    }
  }, [query])

  const performSearch = async (q: string) => {
    if (!q.trim()) {
      setResults([])
      return
    }

    setLoading(true)
    setError('')

    try {
      const response = await api.get('/videos', {
        params: { search: q, limit: 50 }
      })
      setResults(response.data.videos || [])
    } catch (err: any) {
      setError(err.response?.data?.error || 'Erreur lors de la recherche')
      setResults([])
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      window.location.href = `/search?q=${encodeURIComponent(searchQuery)}`
    }
  }

  return (
    <div className="search-page">
      <div className="search-container">
        <h1>Recherche</h1>
        <form onSubmit={handleSubmit} className="search-form">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Rechercher des vidéos..."
            className="search-input"
          />
          <button type="submit" className="search-button">
            🔍 Rechercher
          </button>
        </form>

        {loading && <div className="loading">Recherche en cours...</div>}
        {error && <div className="error">{error}</div>}

        {!loading && !error && results.length > 0 && (
          <div className="results">
            <h2>{results.length} résultat(s) trouvé(s)</h2>
            <div className="results-grid">
              {results.map((video) => (
                <div key={video.id} className="result-item">
                  <h3>{video.title}</h3>
                  <p>{video.description}</p>
                </div>
              ))}
            </div>
          </div>
        )}

        {!loading && !error && query && results.length === 0 && (
          <div className="no-results">
            <p>Aucun résultat trouvé pour "{query}"</p>
          </div>
        )}
      </div>
    </div>
  )
}

