import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminTorrentCandidates.css'

const AdminTorrentCandidates = () => {
  const { user, isAdmin, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [candidates, setCandidates] = useState([])
  const [loading, setLoading] = useState(true)
  const [configs, setConfigs] = useState([])
  const [filters, setFilters] = useState({
    config_id: '',
    status: '',
    approved: '',
    rejected: ''
  })

  // Récupérer config_id depuis l'URL si présent
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search)
    const configId = urlParams.get('config_id')
    if (configId) {
      setFilters(prev => ({ ...prev, config_id: configId }))
    }
  }, [])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [limit] = useState(50)

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
    fetchConfigs()
    fetchCandidates()
  }, [user, authLoading, isAdmin, navigate, page, filters])

  const fetchConfigs = async () => {
    try {
      const response = await api.get('/admin/torrent-configs')
      setConfigs(response.data || [])
    } catch (error) {
      console.error('Erreur lors de la récupération des configurations:', error)
    }
  }

  const fetchCandidates = async () => {
    try {
      setLoading(true)
      const params = new URLSearchParams()
      if (filters.config_id) params.append('config_id', filters.config_id)
      if (filters.status) params.append('status', filters.status)
      if (filters.approved) params.append('approved', filters.approved)
      if (filters.rejected) params.append('rejected', filters.rejected)
      params.append('page', page)
      params.append('limit', limit)

      const response = await api.get(`/admin/torrent-candidates?${params.toString()}`)
      setCandidates(response.data.candidates || [])
      setTotal(response.data.total || 0)
    } catch (error) {
      console.error('Erreur lors de la récupération des candidats:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleApprove = async (candidateId) => {
    try {
      await api.post(`/admin/torrent-candidates/${candidateId}/approve`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur lors de l\'approbation:', error)
      alert('Erreur lors de l\'approbation')
    }
  }

  const handleDisapprove = async (candidateId) => {
    try {
      await api.post(`/admin/torrent-candidates/${candidateId}/disapprove`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur lors de la désapprobation:', error)
      alert('Erreur lors de la désapprobation')
    }
  }

  const handleReject = async (candidateId) => {
    try {
      await api.post(`/admin/torrent-candidates/${candidateId}/reject`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur lors du rejet:', error)
      alert('Erreur lors du rejet')
    }
  }

  const handleLearn = async (candidateId, action) => {
    try {
      const patternType = prompt('Type de pattern:\n- keyword (mot-clé dans le nom)\n- quality (qualité: 1080p, 720p, etc.)\n- language (langue: FR, EN, etc.)\n- seeders (nombre minimum)\n\nLaissez vide pour détection automatique:', '')
      
      const pattern = patternType ? prompt('Pattern à rechercher:', '') : ''
      
      await api.post(`/admin/torrent-candidates/${candidateId}/learn`, {
        action: action,
        pattern_type: patternType || undefined,
        pattern: pattern || undefined
      })
      
      alert(`Pattern d'apprentissage créé ! Le système apprendra à ${action === 'approve' ? 'approuver' : 'rejeter'} les torrents similaires.`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur lors de l\'apprentissage:', error)
      alert('Erreur lors de l\'apprentissage')
    }
  }

  const handleDownload = async (candidateId) => {
    try {
      if (!confirm('Télécharger ce torrent maintenant ?')) return
      await api.post(`/admin/torrent-candidates/${candidateId}/download`)
      alert('Torrent ajouté avec succès !')
      fetchCandidates()
    } catch (error) {
      console.error('Erreur lors du téléchargement:', error)
      const errorMsg = error.response?.data?.error || 'Erreur lors du téléchargement'
      alert(`Erreur: ${errorMsg}`)
    }
  }

  const handleDelete = async (candidateId) => {
    try {
      if (!confirm('Supprimer ce candidat ?')) return
      await api.delete(`/admin/torrent-candidates/${candidateId}`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur lors de la suppression:', error)
      alert('Erreur lors de la suppression')
    }
  }

  const formatSize = (bytes) => {
    if (!bytes || bytes === 0) return '0.00 MB'
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return `${gb.toFixed(2)} GB`
    const mb = bytes / (1024 * 1024)
    if (mb >= 1) return `${mb.toFixed(2)} MB`
    const kb = bytes / 1024
    if (kb >= 1) return `${kb.toFixed(2)} KB`
    return `${bytes} B`
  }

  const getStatusBadge = (status, approved, rejected) => {
    if (rejected) return <span className="badge rejected">Rejeté</span>
    if (status === 'downloaded') return <span className="badge downloaded">Téléchargé</span>
    if (approved) return <span className="badge approved">Approuvé</span>
    return <span className="badge pending">En attente</span>
  }

  const totalPages = Math.ceil(total / limit)

  if (loading && candidates.length === 0) {
    return (
      <div className="admin-torrent-candidates">
        <LoadingSpinner message="Chargement des candidats..." />
      </div>
    )
  }

  return (
    <div className="admin-torrent-candidates">
      <div className="admin-header">
        <h1>🎯 Candidats de Torrents</h1>
        <p className="subtitle">Résultats de recherche automatique - {total} candidat(s) trouvé(s)</p>
      </div>

      {/* Filtres */}
      <div className="filters-section">
        <div className="filters-grid">
          <div className="filter-group">
            <label>Configuration:</label>
            <select
              value={filters.config_id}
              onChange={(e) => {
                setFilters({ ...filters, config_id: e.target.value })
                setPage(1)
              }}
            >
              <option value="">Toutes</option>
              {configs.map(config => (
                <option key={config.id} value={config.id}>{config.name}</option>
              ))}
            </select>
          </div>

          <div className="filter-group">
            <label>Statut:</label>
            <select
              value={filters.status}
              onChange={(e) => {
                setFilters({ ...filters, status: e.target.value })
                setPage(1)
              }}
            >
              <option value="">Tous</option>
              <option value="pending">En attente</option>
              <option value="approved">Approuvé</option>
              <option value="rejected">Rejeté</option>
              <option value="downloaded">Téléchargé</option>
            </select>
          </div>

          <div className="filter-group">
            <label>Approuvé:</label>
            <select
              value={filters.approved}
              onChange={(e) => {
                setFilters({ ...filters, approved: e.target.value })
                setPage(1)
              }}
            >
              <option value="">Tous</option>
              <option value="true">Oui</option>
              <option value="false">Non</option>
            </select>
          </div>

          <div className="filter-group">
            <label>Rejeté:</label>
            <select
              value={filters.rejected}
              onChange={(e) => {
                setFilters({ ...filters, rejected: e.target.value })
                setPage(1)
              }}
            >
              <option value="">Tous</option>
              <option value="true">Oui</option>
              <option value="false">Non</option>
            </select>
          </div>
        </div>
      </div>

      {/* Liste des candidats */}
      <div className="candidates-list">
        {candidates.length === 0 ? (
          <div className="empty-state">
            <p>📭 Aucun candidat trouvé</p>
            <p className="hint">Les candidats apparaîtront ici après que les configurations aient effectué leurs recherches automatiques.</p>
          </div>
        ) : (
          candidates.map(candidate => (
            <div key={candidate.id} className="candidate-card">
              <div className="candidate-header">
                <h3>{candidate.name}</h3>
                {getStatusBadge(candidate.status, candidate.approved_for_download, candidate.rejected)}
              </div>

              <div className="candidate-info">
                <div className="info-row">
                  <span className="label">Configuration:</span>
                  <span>{candidate.config?.name || 'N/A'}</span>
                </div>
                <div className="info-row">
                  <span className="label">Source:</span>
                  <span className="source-badge">{candidate.source}</span>
                </div>
                <div className="info-row">
                  <span className="label">Taille:</span>
                  <span>{formatSize(candidate.size)}</span>
                </div>
                <div className="info-row">
                  <span className="label">Seeders:</span>
                  <span className={candidate.seeders > 0 ? 'seeders-good' : 'seeders-bad'}>{candidate.seeders}</span>
                </div>
                <div className="info-row">
                  <span className="label">Leechers:</span>
                  <span>{candidate.leechers}</span>
                </div>
                {candidate.quality && (
                  <div className="info-row">
                    <span className="label">Qualité:</span>
                    <span className="quality-badge">{candidate.quality}</span>
                  </div>
                )}
                {candidate.language && (
                  <div className="info-row">
                    <span className="label">Langue:</span>
                    <span>{candidate.language}</span>
                  </div>
                )}
                {candidate.category && (
                  <div className="info-row">
                    <span className="label">Catégorie:</span>
                    <span>{candidate.category}</span>
                  </div>
                )}
                <div className="info-row">
                  <span className="label">Trouvé le:</span>
                  <span>{new Date(candidate.created_at).toLocaleString('fr-FR')}</span>
                </div>
              </div>

              <div className="candidate-actions">
                {!candidate.rejected && candidate.status !== 'downloaded' && (
                  <>
                    {!candidate.approved_for_download ? (
                      <button
                        className="btn-approve"
                        onClick={() => handleApprove(candidate.id)}
                        title="Approuver pour téléchargement automatique"
                      >
                        ✅ Approuver
                      </button>
                    ) : (
                      <>
                        <button
                          className="btn-disapprove"
                          onClick={() => handleDisapprove(candidate.id)}
                          title="Désapprouver (remettre en attente)"
                        >
                          ⏸️ Désapprouver
                        </button>
                        <button
                          className="btn-approved"
                          disabled
                          title="Approuvé pour téléchargement automatique"
                        >
                          ✅ Approuvé
                        </button>
                      </>
                    )}
                    <button
                      className="btn-download"
                      onClick={() => handleDownload(candidate.id)}
                      title="Télécharger immédiatement"
                    >
                      ⬇️ Télécharger
                    </button>
                  </>
                )}
                {!candidate.approved_for_download && candidate.status !== 'downloaded' && (
                  <button
                    className="btn-reject"
                    onClick={() => handleReject(candidate.id)}
                    title="Rejeter ce candidat"
                  >
                    ❌ Rejeter
                  </button>
                )}
                {candidate.status !== 'downloaded' && (
                  <>
                    <button
                      className="btn-learn-approve"
                      onClick={() => handleLearn(candidate.id, 'approve')}
                      title="Apprendre à approuver les torrents similaires"
                    >
                      🧠 Apprendre (Approuver)
                    </button>
                    <button
                      className="btn-learn-reject"
                      onClick={() => handleLearn(candidate.id, 'reject')}
                      title="Apprendre à rejeter les torrents similaires"
                    >
                      🧠 Apprendre (Rejeter)
                    </button>
                  </>
                )}
                <button
                  className="btn-delete"
                  onClick={() => handleDelete(candidate.id)}
                  title="Supprimer ce candidat"
                >
                  🗑️ Supprimer
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="pagination">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
          >
            ← Précédent
          </button>
          <span>Page {page} sur {totalPages}</span>
          <button
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
          >
            Suivant →
          </button>
        </div>
      )}
    </div>
  )
}

export default AdminTorrentCandidates

