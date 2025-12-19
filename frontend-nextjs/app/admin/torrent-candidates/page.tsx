'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminTorrentCandidates.css'

export default function AdminTorrentCandidatesPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [user, setUser] = useState<User | null>(null)
  const [candidates, setCandidates] = useState<any[]>([])
  const [configs, setConfigs] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [filters, setFilters] = useState({
    config_id: searchParams.get('config_id') || '',
    status: '',
    approved: '',
    rejected: ''
  })

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/torrent-candidates')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchConfigs()
      fetchCandidates()
    }
    checkAuth()
  }, [router])

  useEffect(() => {
    if (user) {
      fetchCandidates()
    }
  }, [filters, user])

  const fetchConfigs = async () => {
    try {
      const response = await api.get('/admin/torrent-configs')
      setConfigs(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const fetchCandidates = async () => {
    try {
      setLoading(true)
      const params: any = {}
      if (filters.config_id) params.config_id = filters.config_id
      if (filters.status) params.status = filters.status
      if (filters.approved) params.approved = filters.approved
      if (filters.rejected) params.rejected = filters.rejected

      const response = await api.get('/admin/torrent-candidates', { params })
      setCandidates(response.data.candidates || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleApprove = async (id: string) => {
    try {
      await api.post(`/admin/torrent-candidates/${id}/approve`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de l\'approbation')
    }
  }

  const handleReject = async (id: string) => {
    try {
      await api.post(`/admin/torrent-candidates/${id}/reject`)
      fetchCandidates()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors du rejet')
    }
  }

  const formatBytes = (bytes: number) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  if (loading) {
    return (
      <div className="admin-torrent-candidates">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="admin-torrent-candidates">
      <div className="admin-header">
        <h1>🎯 Candidats Torrent</h1>
      </div>

      <div className="filters">
        <select
          value={filters.config_id}
          onChange={(e) => setFilters({ ...filters, config_id: e.target.value })}
        >
          <option value="">Toutes les configurations</option>
          {configs.map((config: any) => (
            <option key={config.id} value={config.id}>{config.name}</option>
          ))}
        </select>
        <select
          value={filters.status}
          onChange={(e) => setFilters({ ...filters, status: e.target.value })}
        >
          <option value="">Tous les statuts</option>
          <option value="pending">En attente</option>
          <option value="approved">Approuvé</option>
          <option value="rejected">Rejeté</option>
        </select>
      </div>

      <div className="candidates-list">
        {candidates.map((candidate: any) => (
          <div key={candidate.id} className="candidate-card">
            <h3>{candidate.name}</h3>
            <div className="candidate-info">
              <span>📦 {formatBytes(candidate.size)}</span>
              <span>🌱 {candidate.seeders} seeders</span>
              <span>👥 {candidate.leechers} leechers</span>
            </div>
            <div className="candidate-actions">
              {candidate.status === 'pending' && (
                <>
                  <button onClick={() => handleApprove(candidate.id)} className="btn-approve">
                    ✅ Approuver
                  </button>
                  <button onClick={() => handleReject(candidate.id)} className="btn-reject">
                    ❌ Rejeter
                  </button>
                </>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

