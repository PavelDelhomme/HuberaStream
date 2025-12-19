'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './AdminTorrentClients.css'

export default function AdminTorrentClientsPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [currentClient, setCurrentClient] = useState<any>(null)
  const [clients, setClients] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [testing, setTesting] = useState<Record<string, boolean>>({})
  const [testResults, setTestResults] = useState<Record<string, any>>({})

  useEffect(() => {
    const checkAuth = async () => {
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin/torrent-clients')
        return
      }

      const currentUser = await authService.getCurrentUser()
      if (!currentUser || !authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      fetchCurrentClient()
      fetchClientsStatus()
    }
    checkAuth()
  }, [router])

  const fetchCurrentClient = async () => {
    try {
      const response = await api.get('/admin/torrent-clients/current')
      setCurrentClient(response.data)
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchClientsStatus = async () => {
    try {
      const response = await api.get('/admin/torrent-clients/status')
      setClients(response.data.clients || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const handleTest = async (clientType: string) => {
    setTesting((prev) => ({ ...prev, [clientType]: true }))
    try {
      const response = await api.post('/admin/torrent-clients/test', {
        type: clientType
      })
      setTestResults((prev) => ({ ...prev, [clientType]: response.data }))
    } catch (error: any) {
      setTestResults((prev) => ({
        ...prev,
        [clientType]: { success: false, error: error.response?.data?.error || 'Erreur inconnue' }
      }))
    } finally {
      setTesting((prev) => ({ ...prev, [clientType]: false }))
    }
  }

  const handleSetCurrent = async (clientType: string) => {
    try {
      await api.post('/admin/torrent-clients/set-current', { type: clientType })
      alert('Client torrent défini avec succès')
      fetchCurrentClient()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la définition du client')
    }
  }

  if (loading) {
    return (
      <div className="admin-torrent-clients">
        <div className="loading">Chargement...</div>
      </div>
    )
  }

  return (
    <div className="admin-torrent-clients">
      <div className="admin-header">
        <h1>💻 Clients Torrent</h1>
      </div>

      {currentClient && (
        <div className="current-client">
          <h2>Client actuel: {currentClient.type}</h2>
          <p>Status: {currentClient.status || 'N/A'}</p>
        </div>
      )}

      <div className="clients-list">
        {clients.map((client: any) => (
          <div key={client.type} className="client-card">
            <h3>{client.type}</h3>
            <p>Status: {client.available ? '✅ Disponible' : '❌ Indisponible'}</p>
            <div className="client-actions">
              <button
                onClick={() => handleTest(client.type)}
                disabled={testing[client.type]}
                className="btn-test"
              >
                {testing[client.type] ? '⏳ Test...' : '🧪 Tester'}
              </button>
              {client.available && (
                <button
                  onClick={() => handleSetCurrent(client.type)}
                  className="btn-set-current"
                >
                  ✓ Définir comme actuel
                </button>
              )}
            </div>
            {testResults[client.type] && (
              <div className={`test-result ${testResults[client.type].success ? 'success' : 'error'}`}>
                {testResults[client.type].success ? '✅ Test réussi' : `❌ ${testResults[client.type].error}`}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

