import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import './AdminTorrentClients.css'

const AdminTorrentClients = () => {
  const { user, loading: authLoading, isAdmin } = useAuth()
  const navigate = useNavigate()
  const [clients, setClients] = useState([])
  const [currentClient, setCurrentClient] = useState(null)
  const [loading, setLoading] = useState(true)
  const [testing, setTesting] = useState({})
  const [testResults, setTestResults] = useState({})
  const [expandedClient, setExpandedClient] = useState(null)
  const [clientConfigs, setClientConfigs] = useState({
    transmission: {
      url: 'http://transmission:9091/transmission/rpc',
      user: 'transmission',
      pass: 'transmission'
    },
    qbittorrent: {
      url: 'http://qbittorrent:8080',
      user: 'admin',
      pass: 'fSBT53tSk' // Mot de passe temporaire actuel (voir logs Docker)
    },
    deluge: {
      url: 'http://deluge:8112/json',
      pass: 'deluge'
    },
    rtorrent: {
      url: 'http://rtorrent:5000/RPC2'
    }
  })

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
    fetchCurrentClient()
    fetchClientsStatus()
  }, [user, authLoading, isAdmin, navigate])

  const fetchCurrentClient = async () => {
    try {
      const response = await api.get('/admin/torrent-clients/current')
      setCurrentClient(response.data)
      if (response.data && response.data.config) {
        // Mettre à jour les configurations depuis le backend
        setClientConfigs(prev => ({
          transmission: {
            url: response.data.config.transmission_url || prev.transmission.url,
            user: response.data.config.transmission_user || prev.transmission.user,
            pass: response.data.config.transmission_pass || prev.transmission.pass
          },
          qbittorrent: {
            url: response.data.config.qbittorrent_url || prev.qbittorrent.url,
            user: response.data.config.qbittorrent_user || prev.qbittorrent.user,
            pass: response.data.config.qbittorrent_pass || prev.qbittorrent.pass
          },
          deluge: {
            url: response.data.config.deluge_url || prev.deluge.url,
            pass: response.data.config.deluge_pass || prev.deluge.pass
          },
          rtorrent: {
            url: response.data.config.rtorrent_url || prev.rtorrent.url
          }
        }))
      }
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

  const handleTestClient = async (clientType) => {
    setTesting(prev => ({ ...prev, [clientType]: true }))
    setTestResults(prev => ({ ...prev, [clientType]: null }))
    
    try {
      const config = getConfigForClient(clientType)
      const response = await api.post('/admin/torrent-clients/test', {
        client_type: clientType,
        config: config
      })
      setTestResults(prev => ({
        ...prev,
        [clientType]: {
          success: true,
          message: `Connexion réussie à ${clientType}`,
          data: response.data
        }
      }))
    } catch (error) {
      setTestResults(prev => ({
        ...prev,
        [clientType]: {
          success: false,
          message: error.response?.data?.error || `Erreur de connexion à ${clientType}`,
          error: error.response?.data,
          details: error.response?.data?.details || error.response?.data?.message
        }
      }))
    } finally {
      setTesting(prev => ({ ...prev, [clientType]: false }))
    }
  }

  const getConfigForClient = (clientType) => {
    const config = clientConfigs[clientType]
    if (!config) return {}
    
    switch (clientType) {
      case 'transmission':
        return {
          url: config.url,
          user: config.user,
          pass: config.pass
        }
      case 'qbittorrent':
        return {
          url: config.url,
          user: config.user,
          pass: config.pass
        }
      case 'deluge':
        return {
          url: config.url,
          pass: config.pass
        }
      case 'rtorrent':
        return {
          url: config.url
        }
      default:
        return {}
    }
  }

  const handleSaveClient = async (clientType) => {
    try {
      await api.put('/admin/torrent-clients/current', {
        client_type: clientType,
        config: getConfigForClient(clientType)
      })
      alert(`Configuration de ${clientType} sauvegardée avec succès !`)
      fetchCurrentClient()
    } catch (error) {
      console.error('Erreur:', error)
      alert(`Erreur lors de la sauvegarde de ${clientType}`)
    }
  }

  const updateClientConfig = (clientType, field, value) => {
    setClientConfigs(prev => ({
      ...prev,
      [clientType]: {
        ...prev[clientType],
        [field]: value
      }
    }))
  }

  const toggleClientConfig = (clientType) => {
    setExpandedClient(expandedClient === clientType ? null : clientType)
  }

  if (loading) {
    return (
      <div className="admin-torrent-clients">
        <LoadingSpinner message="Chargement des clients torrent..." />
      </div>
    )
  }

  return (
    <div className="admin-torrent-clients">
      <div className="admin-header">
        <h1>⚙️ Gestion des Clients Torrent</h1>
        <p className="subtitle">Configurez et testez vos clients torrent</p>
      </div>

      {/* Statut et configuration des clients */}
      <div className="clients-status-section">
        <h2>📊 Clients Torrent</h2>
        <div className="clients-grid">
          {clients.map((client) => {
            const clientType = client.type
            const config = clientConfigs[clientType] || {}
            const isExpanded = expandedClient === clientType
            const isTesting = testing[clientType]
            const testResult = testResults[clientType]

            return (
              <div key={clientType} className="client-status-card">
                <div className="client-header">
                  <h3>{client.name}</h3>
                  <span className={`status-badge ${client.available ? 'online' : 'offline'}`}>
                    {client.available ? '🟢 En ligne' : '🔴 Hors ligne'}
                  </span>
                </div>
                
                <div className="client-info">
                  <p><strong>Port Web UI:</strong> {client.web_port}</p>
                  <p><strong>URL:</strong> <a href={client.web_url} target="_blank" rel="noopener noreferrer">{client.web_url}</a></p>
                </div>

                {/* Bouton pour afficher/masquer la configuration */}
                <button
                  onClick={() => toggleClientConfig(clientType)}
                  className="config-toggle-button"
                >
                  {isExpanded ? '▼ Masquer la configuration' : '▶ Configurer'}
                </button>

                {/* Formulaire de configuration */}
                {isExpanded && (
                  <div className="client-config-form">
                    {/* Transmission */}
                    {clientType === 'transmission' && (
                      <>
                        <div className="config-form-group">
                          <label>URL Transmission</label>
                          <input
                            type="text"
                            value={config.url || ''}
                            onChange={(e) => updateClientConfig(clientType, 'url', e.target.value)}
                            placeholder="http://transmission:9091/transmission/rpc"
                          />
                        </div>
                        <div className="config-form-row">
                          <div className="config-form-group">
                            <label>Utilisateur</label>
                            <input
                              type="text"
                              value={config.user || ''}
                              onChange={(e) => updateClientConfig(clientType, 'user', e.target.value)}
                            />
                          </div>
                          <div className="config-form-group">
                            <label>Mot de passe</label>
                            <input
                              type="password"
                              value={config.pass || ''}
                              onChange={(e) => updateClientConfig(clientType, 'pass', e.target.value)}
                            />
                          </div>
                        </div>
                      </>
                    )}

                    {/* qBittorrent */}
                    {clientType === 'qbittorrent' && (
                      <>
                        <div className="config-form-group">
                          <label>URL qBittorrent</label>
                          <input
                            type="text"
                            value={config.url || ''}
                            onChange={(e) => updateClientConfig(clientType, 'url', e.target.value)}
                            placeholder="http://qbittorrent:8080"
                          />
                        </div>
                        <div className="config-form-row">
                          <div className="config-form-group">
                            <label>Utilisateur</label>
                            <input
                              type="text"
                              value={config.user || ''}
                              onChange={(e) => updateClientConfig(clientType, 'user', e.target.value)}
                            />
                          </div>
                          <div className="config-form-group">
                            <label>Mot de passe</label>
                            <input
                              type="password"
                              value={config.pass || ''}
                              onChange={(e) => updateClientConfig(clientType, 'pass', e.target.value)}
                              placeholder="Laissez vide pour utiliser le mot de passe temporaire"
                            />
                          </div>
                        </div>
                      </>
                    )}

                    {/* Deluge */}
                    {clientType === 'deluge' && (
                      <>
                        <div className="config-form-group">
                          <label>URL Deluge</label>
                          <input
                            type="text"
                            value={config.url || ''}
                            onChange={(e) => updateClientConfig(clientType, 'url', e.target.value)}
                            placeholder="http://deluge:8112/json"
                          />
                        </div>
                        <div className="config-form-group">
                          <label>Mot de passe</label>
                          <input
                            type="password"
                            value={config.pass || ''}
                            onChange={(e) => updateClientConfig(clientType, 'pass', e.target.value)}
                          />
                        </div>
                      </>
                    )}

                    {/* rTorrent */}
                    {clientType === 'rtorrent' && (
                      <>
                        <div className="config-form-group">
                          <label>URL rTorrent XMLRPC</label>
                          <input
                            type="text"
                            value={config.url || ''}
                            onChange={(e) => updateClientConfig(clientType, 'url', e.target.value)}
                            placeholder="http://rtorrent:5000/RPC2"
                          />
                        </div>
                        <div className="test-result-mini error" style={{ marginTop: '10px' }}>
                          <p><strong>⚠️</strong> rTorrent n'est pas encore implémenté</p>
                          <p className="test-details">Utilisez Transmission, qBittorrent ou Deluge pour l'instant.</p>
                        </div>
                      </>
                    )}

                    {/* Résultat du test */}
                    {testResult && (
                      <div className={`test-result-mini ${testResult.success ? 'success' : 'error'}`}>
                        <p><strong>{testResult.success ? '✅' : '❌'}</strong> {testResult.message}</p>
                        {testResult.details && (
                          <p className="test-details">{testResult.details}</p>
                        )}
                        {testResult.data && testResult.data.torrents_count !== undefined && (
                          <p className="test-details">Torrents trouvés: {testResult.data.torrents_count}</p>
                        )}
                      </div>
                    )}

                    {/* Actions */}
                    <div className="client-config-actions">
                      <button
                        onClick={() => handleTestClient(clientType)}
                        disabled={isTesting}
                        className="test-button-small"
                      >
                        {isTesting ? '⏳ Test...' : '🔍 Tester'}
                      </button>
                      <button
                        onClick={() => handleSaveClient(clientType)}
                        className="save-button-small"
                      >
                        💾 Sauvegarder
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {/* Configuration du client actif (pour compatibilité) */}
      <div className="config-section">
        <h2>🔧 Client Actif</h2>
        <div className="current-client-info">
          <p>Client actuellement sélectionné: <strong>{currentClient?.type || 'Aucun'}</strong></p>
          <p className="info-text">Configurez chaque client individuellement ci-dessus, puis testez et sauvegardez.</p>
        </div>
      </div>

      {/* Liens vers les interfaces Web */}
      <div className="web-interfaces-section">
        <h2>🌐 Interfaces Web des Clients</h2>
        <div className="web-interfaces-grid">
          <a href="http://localhost:3305" target="_blank" rel="noopener noreferrer" className="web-interface-card">
            <h3>📡 Transmission</h3>
            <p>Port 3305</p>
          </a>
          <a href="http://localhost:3306" target="_blank" rel="noopener noreferrer" className="web-interface-card">
            <h3>⚡ qBittorrent</h3>
            <p>Port 3306</p>
          </a>
          <a href="http://localhost:3307" target="_blank" rel="noopener noreferrer" className="web-interface-card">
            <h3>🌊 Deluge</h3>
            <p>Port 3307</p>
          </a>
          <a href="http://localhost:3308" target="_blank" rel="noopener noreferrer" className="web-interface-card">
            <h3>⚙️ ruTorrent</h3>
            <p>Port 3308</p>
          </a>
        </div>
      </div>
    </div>
  )
}

export default AdminTorrentClients
