import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminTorrents.css'

const AdminTorrents = () => {
  const { user, isAdmin, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('list') // 'list', 'search', 'browse'
  const [torrents, setTorrents] = useState([])
  const [loading, setLoading] = useState(true)
  const [initialLoading, setInitialLoading] = useState(true) // Chargement initial uniquement
  const [refreshing, setRefreshing] = useState({})
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true)
  
  // Recherche
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState([])
  const [searching, setSearching] = useState(false)
  
  // Parcours catégories
  const [browseCategory, setBrowseCategory] = useState('TV')
  const [browsePage, setBrowsePage] = useState(1)
  const [browseResults, setBrowseResults] = useState([])
  const [browsing, setBrowsing] = useState(false)
  const [browseLanguage, setBrowseLanguage] = useState('')
  
  // Form ajout
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    magnet_url: '',
    torrent_url: '',
    source: 'custom',
    category_id: '',
    sub_category_id: ''
  })
  const [categories, setCategories] = useState([])

  useEffect(() => {
    if (authLoading) return
    if (!user) {
      navigate('/login')
      return
    }
    if (!isAdmin()) {
      navigate('/')
      return
    }
    fetchTorrents(true) // Chargement initial
    fetchCategories()
    // Auto-refresh toutes les 5 secondes UNIQUEMENT s'il y a des torrents actifs
    const interval = setInterval(() => {
      if (autoRefreshEnabled) {
        // Vérifier s'il y a des torrents actifs (pending, downloading, seeding)
        const hasActiveTorrents = torrents.some(t => 
          t.status === 'pending' || t.status === 'downloading' || t.status === 'seeding'
        )
        // Ne rafraîchir que s'il y a des torrents actifs
        if (hasActiveTorrents) {
          fetchTorrentsBackground()
        }
      }
    }, 5000)
    return () => clearInterval(interval)
  }, [user, authLoading, isAdmin, navigate, autoRefreshEnabled, torrents])

  const fetchTorrents = async (isInitial = false) => {
    try {
      // Ne mettre loading à true que lors du chargement initial
      if (isInitial || torrents.length === 0) {
        setLoading(true)
      }
      const response = await api.get('/admin/torrents')
      setTorrents(response.data.torrents || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
      setInitialLoading(false)
    }
  }

  // Rafraîchissement en arrière-plan sans bloquer l'UI
  const fetchTorrentsBackground = async () => {
    try {
      const response = await api.get('/admin/torrents')
      setTorrents(response.data.torrents || [])
    } catch (error) {
      // Ignorer les erreurs en arrière-plan pour ne pas perturber l'utilisateur
      console.error('Erreur lors du rafraîchissement en arrière-plan:', error)
    }
  }

  const fetchCategories = async () => {
    try {
      const response = await api.get('/categories')
      setCategories(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const handleRefreshTorrent = async (torrentId) => {
    setRefreshing(prev => ({ ...prev, [torrentId]: true }))
    try {
      const response = await api.get(`/admin/torrents/${torrentId}/status`)
      setTorrents(prev => prev.map(t => t.id === torrentId ? response.data : t))
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setRefreshing(prev => ({ ...prev, [torrentId]: false }))
    }
  }

  const handlePauseTorrent = async (torrentId) => {
    try {
      await api.post(`/admin/torrents/${torrentId}/pause`)
      // Mettre à jour localement sans recharger toute la liste
      setTorrents(prev => prev.map(t => 
        t.id === torrentId ? { ...t, status: 'paused' } : t
      ))
      // Rafraîchir en arrière-plan
      fetchTorrentsBackground()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la mise en pause')
    }
  }

  const handleResumeTorrent = async (torrentId) => {
    try {
      await api.post(`/admin/torrents/${torrentId}/resume`)
      // Mettre à jour localement sans recharger toute la liste
      setTorrents(prev => prev.map(t => 
        t.id === torrentId ? { ...t, status: 'downloading' } : t
      ))
      // Rafraîchir en arrière-plan
      fetchTorrentsBackground()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la reprise')
    }
  }

  const handleDeleteTorrent = async (torrentId) => {
    if (!window.confirm('Êtes-vous sûr de vouloir supprimer ce torrent ?')) {
      return
    }
    try {
      await api.delete(`/admin/torrents/${torrentId}`)
      // Mettre à jour localement sans recharger toute la liste
      setTorrents(prev => prev.filter(t => t.id !== torrentId))
      // Rafraîchir en arrière-plan pour synchroniser
      fetchTorrentsBackground()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la suppression')
    }
  }

  const handleSearch = async () => {
    if (!searchQuery.trim()) return
    setSearching(true)
    try {
      const response = await api.post('/admin/scraper/search', {
        query: searchQuery,
        source: '1337x'
      })
      setSearchResults(response.data.results || [])
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la recherche')
    } finally {
      setSearching(false)
    }
  }

  const handleBrowseCategory = async () => {
    setBrowsing(true)
    try {
      const response = await api.get('/admin/scraper/browse-1337x', {
        params: {
          category: browseCategory,
          page: browsePage,
          language: browseLanguage || undefined
        }
      })
      setBrowseResults(response.data.results || [])
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors du parcours de la catégorie')
    } finally {
      setBrowsing(false)
    }
  }

  const handleDownloadFrom1337x = async (torrentURL, name, addToLibrary = false) => {
    const action = addToLibrary ? 'Ajouter à la bibliothèque' : 'Télécharger'
    if (!window.confirm(`${action} "${name}" ?\n\n${addToLibrary ? '✅ Le torrent sera téléchargé et la vidéo sera automatiquement ajoutée à la bibliothèque (visible sur l\'accueil)' : 'Le torrent sera téléchargé mais ne sera pas automatiquement ajouté à la bibliothèque'}`)) {
      return
    }
    try {
      await api.post('/admin/torrents/download-1337x', {
        torrent_url: torrentURL,
        category_id: null,
        sub_category_id: null,
        add_to_library: addToLibrary
      })
      alert(addToLibrary ? '✅ Torrent ajouté à la bibliothèque ! Il sera téléchargé et la vidéo sera visible sur l\'accueil une fois le traitement terminé.' : 'Torrent ajouté avec succès !')
      fetchTorrentsBackground() // Rafraîchir en arrière-plan
      setActiveTab('list')
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors du téléchargement: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleToggleAddToLibrary = async (torrentId) => {
    try {
      const response = await api.post(`/admin/torrents/${torrentId}/toggle-library`)
      alert(response.data.message || 'Flag "Ajouter à la bibliothèque" mis à jour')
      // Mettre à jour localement
      setTorrents(prev => prev.map(t => 
        t.id === torrentId ? { ...t, add_to_library: !t.add_to_library } : t
      ))
      fetchTorrentsBackground() // Rafraîchir en arrière-plan
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur: ' + (error.response?.data?.error || error.message))
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      await api.post('/admin/torrents', formData)
      setShowAddForm(false)
      setFormData({
        name: '',
        magnet_url: '',
        torrent_url: '',
        source: 'custom',
        category_id: '',
        sub_category_id: ''
      })
      fetchTorrentsBackground() // Rafraîchir en arrière-plan
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de l\'ajout du torrent')
    }
  }

  const formatBytes = (bytes) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  const getStatusColor = (status) => {
    const colors = {
      pending: '#aaaaaa',
      downloading: '#3498db',
      seeding: '#2ecc71',
      completed: '#27ae60',
      error: '#e74c3c',
      paused: '#f39c12'
    }
    return colors[status] || '#aaaaaa'
  }

  const getStatusLabel = (status) => {
    const labels = {
      pending: 'En attente',
      downloading: 'Téléchargement',
      seeding: 'Partage',
      completed: 'Terminé',
      error: 'Erreur',
      paused: 'En pause'
    }
    return labels[status] || status
  }

  useEffect(() => {
    if (activeTab === 'browse') {
      handleBrowseCategory()
    }
  }, [browseCategory, browsePage, browseLanguage])

  // Afficher le spinner uniquement lors du chargement initial
  if (initialLoading && torrents.length === 0) {
    return (
      <div className="admin-torrents">
        <LoadingSpinner message="Chargement des torrents..." />
      </div>
    )
  }

  return (
    <div className="admin-torrents">
      <div className="admin-header">
        <h1>📥 Gestion des Torrents</h1>
        <div style={{ display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
          <div className="header-controls">
            <button
              onClick={() => setAutoRefreshEnabled(!autoRefreshEnabled)}
              className={`toggle-refresh ${autoRefreshEnabled ? 'active' : ''}`}
              title={autoRefreshEnabled ? 'Désactiver le rafraîchissement automatique' : 'Activer le rafraîchissement automatique'}
            >
              {autoRefreshEnabled ? '⏸️ Auto-refresh ON' : '▶️ Auto-refresh OFF'}
            </button>
            <button onClick={fetchTorrents} className="refresh-button" title="Rafraîchir maintenant">
              🔄 Rafraîchir
            </button>
          </div>
          <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
            {showAddForm ? 'Annuler' : '+ Ajouter manuellement'}
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="torrent-tabs">
        <button
          className={activeTab === 'list' ? 'active' : ''}
          onClick={() => setActiveTab('list')}
        >
          📋 Mes Torrents ({torrents.length})
        </button>
        <button
          className={activeTab === 'search' ? 'active' : ''}
          onClick={() => setActiveTab('search')}
        >
          🔍 Rechercher
        </button>
        <button
          className={activeTab === 'browse' ? 'active' : ''}
          onClick={() => setActiveTab('browse')}
        >
          📚 Parcourir Catégories
        </button>
      </div>

      {/* Form ajout manuel */}
      {showAddForm && (
        <form className="add-torrent-form" onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Nom du torrent"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <input
            type="text"
            placeholder="Magnet URL"
            value={formData.magnet_url}
            onChange={(e) => setFormData({ ...formData, magnet_url: e.target.value })}
          />
          <input
            type="text"
            placeholder="Torrent URL"
            value={formData.torrent_url}
            onChange={(e) => setFormData({ ...formData, torrent_url: e.target.value })}
          />
          <select
            value={formData.source}
            onChange={(e) => setFormData({ ...formData, source: e.target.value })}
          >
            <option value="custom">Custom</option>
            <option value="1337x">1337x</option>
            <option value="rarbg">RARBG</option>
            <option value="yts">YTS</option>
            <option value="nyaa">Nyaa</option>
          </select>
          <button type="submit">Ajouter</button>
        </form>
      )}

      {/* Liste des torrents */}
      {activeTab === 'list' && (
        <div className="torrents-list">
          {torrents.length === 0 ? (
            <div className="empty-state">
              <p>Aucun torrent téléchargé</p>
              <p>Utilisez l'onglet "Rechercher" ou "Parcourir Catégories" pour trouver des torrents</p>
            </div>
          ) : (
            torrents.map((torrent) => (
              <div key={torrent.id} className="torrent-card">
                <div className="torrent-header">
                  <div className="torrent-title">
                    <h3>{torrent.name}</h3>
                    <span
                      className="torrent-status"
                      style={{ backgroundColor: getStatusColor(torrent.status) }}
                    >
                      {getStatusLabel(torrent.status)}
                    </span>
                  </div>
                  <div className="torrent-actions">
                    <button
                      onClick={() => handleRefreshTorrent(torrent.id)}
                      disabled={refreshing[torrent.id]}
                      className="action-btn refresh"
                      title="Actualiser"
                    >
                      {refreshing[torrent.id] ? '⏳' : '🔄'}
                    </button>
                    {torrent.status === 'paused' || torrent.status === 'error' ? (
                      <button
                        onClick={() => handleResumeTorrent(torrent.id)}
                        className="action-btn resume"
                        title="Reprendre"
                      >
                        ▶️
                      </button>
                    ) : (
                      <button
                        onClick={() => handlePauseTorrent(torrent.id)}
                        disabled={torrent.status === 'completed' || torrent.status === 'seeding'}
                        className="action-btn pause"
                        title="Mettre en pause"
                      >
                        ⏸️
                      </button>
                    )}
                    <button
                      onClick={() => handleToggleAddToLibrary(torrent.id)}
                      className={`action-btn library ${torrent.add_to_library ? 'active' : ''}`}
                      title={torrent.add_to_library ? 'Retirer de la bibliothèque' : 'Ajouter à la bibliothèque'}
                    >
                      {torrent.add_to_library ? '📚✓' : '📚'}
                    </button>
                    <button
                      onClick={() => handleDeleteTorrent(torrent.id)}
                      className="action-btn delete"
                      title="Supprimer"
                    >
                      🗑️
                    </button>
                  </div>
                </div>
                <div className="torrent-info">
                  {torrent.add_to_library && (
                    <div className="library-badge">
                      📚 Ajouté à la bibliothèque - La vidéo sera visible sur l'accueil une fois le traitement terminé
                    </div>
                  )}
                  <div className="torrent-progress">
                    <div className="progress-bar">
                      <div
                        className="progress-fill"
                        style={{ width: `${torrent.progress || 0}%` }}
                      />
                    </div>
                    <span>{torrent.progress ? torrent.progress.toFixed(1) : 0}%</span>
                  </div>
                  <div className="torrent-details">
                    <span>📦 Taille: {formatBytes(torrent.size)}</span>
                    <span>⬇️ Téléchargé: {formatBytes(torrent.downloaded)}</span>
                    {torrent.download_speed > 0 && (
                      <span>⚡ Vitesse: {formatBytes(torrent.download_speed)}/s</span>
                    )}
                    {torrent.seeders !== undefined && (
                      <span>🌱 Seeders: {torrent.seeders}</span>
                    )}
                    {torrent.leechers !== undefined && (
                      <span>👥 Leechers: {torrent.leechers}</span>
                    )}
                    <span>🔗 Source: {torrent.source}</span>
                  </div>
                  {torrent.error && (
                    <div className="torrent-error">
                      <strong>Erreur:</strong> {torrent.error}
                    </div>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {/* Recherche */}
      {activeTab === 'search' && (
        <div className="search-section">
          <div className="search-form">
            <input
              type="text"
              placeholder="Rechercher un torrent (ex: Game of Thrones S01E01)"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            />
            <button onClick={handleSearch} disabled={searching}>
              {searching ? '⏳ Recherche...' : '🔍 Rechercher'}
            </button>
          </div>
          <div className="search-results">
            {searchResults.map((result, idx) => (
              <div key={idx} className="result-card">
                <div className="result-header">
                  <h4>{result.name}</h4>
                  <div className="result-meta">
                    {result.seeders !== undefined && (
                      <span className="seeders">🌱 {result.seeders}</span>
                    )}
                    {result.size && (
                      <span className="size">📦 {result.size}</span>
                    )}
                    {result.quality && (
                      <span className="quality">🎬 {result.quality}</span>
                    )}
                  </div>
                </div>
                <div className="result-actions">
                  <button
                    onClick={() => handleDownloadFrom1337x(result.torrent_url, result.name, false)}
                    className="download-btn"
                    title="Télécharger le torrent"
                  >
                    ⬇️ Télécharger
                  </button>
                  <button
                    onClick={() => handleDownloadFrom1337x(result.torrent_url, result.name, true)}
                    className="download-btn library-btn"
                    title="Ajouter à la bibliothèque (téléchargement + ajout automatique à l'accueil)"
                  >
                    📚 Ajouter à la bibliothèque
                  </button>
                </div>
              </div>
            ))}
            {searchResults.length === 0 && !searching && (
              <div className="empty-state">
                <p>Entrez un terme de recherche et cliquez sur "Rechercher"</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Parcourir catégories */}
      {activeTab === 'browse' && (
        <div className="browse-section">
          <div className="browse-controls">
            <select
              value={browseCategory}
              onChange={(e) => {
                setBrowseCategory(e.target.value)
                setBrowsePage(1)
              }}
            >
              <option value="TV">📺 TV Shows</option>
              <option value="movie-library">🎬 Movie Library</option>
              <option value="Anime">🎌 Anime</option>
              <option value="Movies">🎥 Movies</option>
              <option value="Documentaries">📚 Documentaries</option>
              <option value="popular-movies">⭐ Popular Movies</option>
            </select>
            <select
              value={browseLanguage}
              onChange={(e) => {
                setBrowseLanguage(e.target.value)
                setBrowsePage(1)
              }}
            >
              <option value="">Toutes les langues</option>
              <option value="FR">🇫🇷 Français</option>
              <option value="EN">🇬🇧 English</option>
              <option value="ES">🇪🇸 Español</option>
            </select>
            <div className="pagination">
              <button
                onClick={() => setBrowsePage(prev => Math.max(1, prev - 1))}
                disabled={browsePage === 1 || browsing}
              >
                ← Précédent
              </button>
              <span>Page {browsePage}</span>
              <button
                onClick={() => setBrowsePage(prev => prev + 1)}
                disabled={browsing}
              >
                Suivant →
              </button>
            </div>
          </div>
          <div className="browse-results">
            {browsing ? (
              <LoadingSpinner message="Chargement..." />
            ) : (
              browseResults.map((result, idx) => (
                <div key={idx} className="result-card">
                  <div className="result-header">
                    <h4>{result.name}</h4>
                    <div className="result-meta">
                      {result.seeders !== undefined && (
                        <span className="seeders">🌱 {result.seeders}</span>
                      )}
                      {result.size && (
                        <span className="size">📦 {result.size}</span>
                      )}
                      {result.quality && (
                        <span className="quality">🎬 {result.quality}</span>
                      )}
                      {result.language && (
                        <span className="language">🌍 {result.language}</span>
                      )}
                    </div>
                  </div>
                  <div className="result-actions">
                    <button
                      onClick={() => handleDownloadFrom1337x(result.torrent_url, result.name, false)}
                      className="download-btn"
                      title="Télécharger le torrent"
                    >
                      ⬇️ Télécharger
                    </button>
                    <button
                      onClick={() => handleDownloadFrom1337x(result.torrent_url, result.name, true)}
                      className="download-btn library-btn"
                      title="Ajouter à la bibliothèque (téléchargement + ajout automatique à l'accueil)"
                    >
                      📚 Ajouter à la bibliothèque
                    </button>
                  </div>
                </div>
              ))
            )}
            {browseResults.length === 0 && !browsing && (
              <div className="empty-state">
                <p>Aucun résultat pour cette catégorie</p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default AdminTorrents
