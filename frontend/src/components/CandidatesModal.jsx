import React, { useState, useEffect } from 'react'
import api from '../services/api'
import './CandidatesModal.css'

const CandidatesModal = ({ configId, configName, onClose }) => {
  const [candidates, setCandidates] = useState([])
  const [loading, setLoading] = useState(true)
  const [filters, setFilters] = useState({
    status: '',
    approved: '',
    rejected: ''
  })

  useEffect(() => {
    if (configId) {
      fetchCandidates()
    }
  }, [configId, filters])

  // Écouter les événements de recherche terminée pour rafraîchir automatiquement
  useEffect(() => {
    const handleSearchCompleted = (event) => {
      if (event.detail && event.detail.configId === configId) {
        // Attendre un peu pour que les candidats soient créés dans la DB
        setTimeout(() => {
          fetchCandidates()
        }, 1000)
      }
    }
    
    window.addEventListener('candidatesSearchCompleted', handleSearchCompleted)
    return () => {
      window.removeEventListener('candidatesSearchCompleted', handleSearchCompleted)
    }
  }, [configId])

  const fetchCandidates = async () => {
    try {
      setLoading(true)
      const params = new URLSearchParams({ config_id: configId })
      if (filters.status) params.append('status', filters.status)
      if (filters.approved) params.append('approved', filters.approved)
      if (filters.rejected) params.append('rejected', filters.rejected)
      
      const response = await api.get(`/admin/torrent-candidates?${params.toString()}`)
      setCandidates(response.data.candidates || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleApprove = async (candidateId) => {
    try {
      await api.post(`/admin/torrent-candidates/${candidateId}/approve`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de l\'approbation')
    }
  }

  const handleDisapprove = async (candidateId) => {
    try {
      await api.post(`/admin/torrent-candidates/${candidateId}/disapprove`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la désapprobation')
    }
  }

  const handleReject = async (candidateId) => {
    try {
      await api.post(`/admin/torrent-candidates/${candidateId}/reject`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors du rejet')
    }
  }

  const handleDownload = async (candidateId) => {
    try {
      if (!confirm('Télécharger ce torrent maintenant ?')) return
      const response = await api.post(`/admin/torrent-candidates/${candidateId}/download`)
      alert('✅ Torrent ajouté avec succès ! Il apparaîtra dans la gestion des torrents.')
      fetchCandidates()
      // Déclencher un événement personnalisé pour rafraîchir la liste des torrents
      window.dispatchEvent(new CustomEvent('torrentDownloaded', { detail: response.data }))
    } catch (error) {
      console.error('Erreur:', error)
      const errorMsg = error.response?.data?.error || 'Erreur lors du téléchargement'
      alert(`❌ Erreur: ${errorMsg}`)
    }
  }

  const handleDeleteAll = async () => {
    try {
      if (!window.confirm(`Supprimer TOUS les candidats de la configuration "${configName}" ?\n\nCette action est irréversible.`)) return
      const response = await api.delete(`/admin/torrent-candidates/config/${configId}/all`)
      alert(`✅ ${response.data.deleted_count || response.data.message || 'Candidats supprimés'} !`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur lors de la suppression:', error)
      const errorMsg = error.response?.data?.error || 'Erreur lors de la suppression'
      alert(`Erreur: ${errorMsg}`)
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

  if (!configId) return null

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content candidates-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>🎯 Candidats - {configName}</h2>
          <div className="header-actions">
            <button
              className="btn-delete-all"
              onClick={handleDeleteAll}
              title="Supprimer tous les candidats de cette configuration"
            >
              🗑️ Supprimer tout
            </button>
            <button className="modal-close" onClick={onClose}>✕</button>
          </div>
        </div>

        <div className="modal-filters">
          <select
            value={filters.status}
            onChange={(e) => setFilters({ ...filters, status: e.target.value })}
          >
            <option value="">Tous les statuts</option>
            <option value="pending">En attente</option>
            <option value="approved">Approuvé</option>
            <option value="rejected">Rejeté</option>
            <option value="downloaded">Téléchargé</option>
          </select>
          <select
            value={filters.approved}
            onChange={(e) => setFilters({ ...filters, approved: e.target.value })}
          >
            <option value="">Tous</option>
            <option value="true">Approuvés</option>
            <option value="false">Non approuvés</option>
          </select>
          <select
            value={filters.rejected}
            onChange={(e) => setFilters({ ...filters, rejected: e.target.value })}
          >
            <option value="">Tous</option>
            <option value="true">Rejetés</option>
            <option value="false">Non rejetés</option>
          </select>
        </div>

        <div className="modal-body">
          {loading ? (
            <div className="modal-loading">Chargement...</div>
          ) : candidates.length === 0 ? (
            <div className="modal-empty">Aucun candidat trouvé</div>
          ) : (
            <div className="candidates-list-modal">
              {candidates.map(candidate => (
                <div key={candidate.id} className="candidate-card-modal">
                  <div className="candidate-header-modal">
                    <h4>{candidate.name}</h4>
                    {getStatusBadge(candidate.status, candidate.approved_for_download, candidate.rejected)}
                  </div>
                  <div className="candidate-info-modal">
                    <div className="info-row-modal">
                      <span className="label">Taille:</span>
                      <span>{formatSize(candidate.size)}</span>
                    </div>
                    <div className="info-row-modal">
                      <span className="label">Seeders:</span>
                      <span className={candidate.seeders > 0 ? 'seeders-good' : 'seeders-bad'}>{candidate.seeders}</span>
                    </div>
                    <div className="info-row-modal">
                      <span className="label">Leechers:</span>
                      <span>{candidate.leechers}</span>
                    </div>
                    {candidate.quality && (
                      <div className="info-row-modal">
                        <span className="label">Qualité:</span>
                        <span className="quality-badge">{candidate.quality}</span>
                      </div>
                    )}
                    {candidate.language && (
                      <div className="info-row-modal">
                        <span className="label">Langue:</span>
                        <span>{candidate.language}</span>
                      </div>
                    )}
                  </div>
                  <div className="candidate-actions-modal">
                    {!candidate.rejected && candidate.status !== 'downloaded' && (
                      <>
                        {!candidate.approved_for_download ? (
                          <button
                            className="btn-approve-small"
                            onClick={() => handleApprove(candidate.id)}
                            title="Approuver pour téléchargement automatique"
                          >
                            ✅
                          </button>
                        ) : (
                          <>
                            <button
                              className="btn-disapprove-small"
                              onClick={() => handleDisapprove(candidate.id)}
                              title="Désapprouver (remettre en attente)"
                            >
                              ⏸️
                            </button>
                            <button className="btn-approved-small" disabled title="Approuvé pour téléchargement automatique">
                              ✅
                            </button>
                          </>
                        )}
                        <button
                          className="btn-download-small"
                          onClick={() => handleDownload(candidate.id)}
                          title="Télécharger maintenant"
                        >
                          ⬇️
                        </button>
                      </>
                    )}
                    {!candidate.approved_for_download && candidate.status !== 'downloaded' && (
                      <button
                        className="btn-reject-small"
                        onClick={() => handleReject(candidate.id)}
                        title="Rejeter"
                      >
                        ❌
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default CandidatesModal

