import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import './AdminTorrentClients.css'

const AdminTorrentClients = () => {
  const { user, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [clients, setClients] = useState([])
  const [currentClient, setCurrentClient] = useState(null)
  const [loading, setLoading] = useState(true)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState(null)
  const [formData, setFormData] = useState({
    client_type: 'transmission',
    transmission_url: 'http://transmission:9091/transmission/rpc',
    transmission_user: 'transmission',
    transmission_pass: 'transmission',
    qbittorrent_url: 'http://qbittorrent:8080',
    qbittorrent_user: 'admin',
    qbittorrent_pass: 'adminadmin',
    deluge_url: 'http://deluge:8112/json',
    deluge_pass: 'deluge',
    rtorrent_url: 'http://rtorrent:5000/RPC2'
  })

  useEffect(() => {
    if (authLoading) return
    if (!user || user.role !== 'admin') {
      navigate('/login')
      return
    }
    fetchCurrentClient()
    fetchClientsStatus()
  }, [user, authLoading, navigate])

  const fetchCurrentClient = async () => {
    try {
      const response = await api.get('/admin/torrent-clients/current')
      setCurrentClient(response.data)
      if (response.data) {
        setFormData(prev => ({
          ...prev,
          client_type: response.data.type || 'transmission',
          ...response.data.config
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
    setTesting(true)
    setTestResult(null)
    try {
      const response = await api.post('/admin/torrent-clients/test', {
        client_type: clientType,
        config: getConfigForClient(clientType)
      })
      setTestResult({
        success: true,
        message: `Connexion réussie à ${clientType}`,
        data: response.data
      })
    } catch (error) {
      setTestResult({
        success: false,
        message: error.response?.data?.error || `Erreur de connexion à ${clientType}`,
        error: error.response?.data
      })
    } finally {
      setTesting(false)
    }
  }

  const getConfigForClient = (clientType) => {
    switch (clientType) {
      case 'transmission':
        return {
          url: formData.transmission_url,
          user: formData.transmission_user,
          pass: formData.transmission_pass
        }
      case 'qbittorrent':
        return {
          url: formData.qbittorrent_url,
          user: formData.qbittorrent_user,
          pass: formData.qbittorrent_pass
        }
      case 'deluge':
        return {
          url: formData.deluge_url,
          pass: formData.deluge_pass
        }
      case 'rtorrent':
        return {
          url: formData.rtorrent_url
        }
      default:
        return {}
    }
  }

  const handleSave = async () => {
    try {
      await api.put('/admin/torrent-clients/current', {
        client_type: formData.client_type,
        config: getConfigForClient(formData.client_type)
      })
      alert('Configuration sauvegardée avec succès !')
      fetchCurrentClient()
    } catch (error) {
      console.error('Erreur:', error)
      alert('Erreur lors de la sauvegarde')
    }
  }

  if (loading) {
    return <div className="admin-torrent-clients">Chargement...</div>
  }

  return (
    <div className="admin-torrent-clients">
      <div className="admin-header">
        <h1>⚙️ Gestion des Clients Torrent</h1>
        <p className="subtitle">Configurez et testez vos clients torrent</p>
      </div>

      {/* Statut des clients */}
      <div className="clients-status-section">
        <h2>📊 Statut des Clients</h2>
        <div className="clients-grid">
          {clients.map((client) => (
            <div key={client.type} className="client-status-card">
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
              <button
                onClick={() => handleTestClient(client.type)}
                disabled={testing}
                className="test-button"
              >
                {testing ? '⏳ Test...' : '🔍 Tester'}
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Résultat du test */}
      {testResult && (
        <div className={`test-result ${testResult.success ? 'success' : 'error'}`}>
          <p><strong>{testResult.success ? '✅' : '❌'}</strong> {testResult.message}</p>
          {testResult.data && (
            <pre>{JSON.stringify(testResult.data, null, 2)}</pre>
          )}
        </div>
      )}

      {/* Configuration du client actif */}
      <div className="config-section">
        <h2>🔧 Configuration du Client Actif</h2>
        <div className="config-form">
          <div className="form-group">
            <label>Client Torrent</label>
            <select
              value={formData.client_type}
              onChange={(e) => setFormData({ ...formData, client_type: e.target.value })}
            >
              <option value="transmission">Transmission (Port 3305)</option>
              <option value="qbittorrent">qBittorrent (Port 3306)</option>
              <option value="deluge">Deluge (Port 3307)</option>
              <option value="rtorrent">rTorrent (Port 3308)</option>
            </select>
          </div>

          {/* Configuration Transmission */}
          {formData.client_type === 'transmission' && (
            <>
              <div className="form-group">
                <label>URL Transmission</label>
                <input
                  type="text"
                  value={formData.transmission_url}
                  onChange={(e) => setFormData({ ...formData, transmission_url: e.target.value })}
                  placeholder="http://transmission:9091/transmission/rpc"
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Utilisateur</label>
                  <input
                    type="text"
                    value={formData.transmission_user}
                    onChange={(e) => setFormData({ ...formData, transmission_user: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label>Mot de passe</label>
                  <input
                    type="password"
                    value={formData.transmission_pass}
                    onChange={(e) => setFormData({ ...formData, transmission_pass: e.target.value })}
                  />
                </div>
              </div>
            </>
          )}

          {/* Configuration qBittorrent */}
          {formData.client_type === 'qbittorrent' && (
            <>
              <div className="form-group">
                <label>URL qBittorrent</label>
                <input
                  type="text"
                  value={formData.qbittorrent_url}
                  onChange={(e) => setFormData({ ...formData, qbittorrent_url: e.target.value })}
                  placeholder="http://qbittorrent:8080"
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Utilisateur</label>
                  <input
                    type="text"
                    value={formData.qbittorrent_user}
                    onChange={(e) => setFormData({ ...formData, qbittorrent_user: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label>Mot de passe</label>
                  <input
                    type="password"
                    value={formData.qbittorrent_pass}
                    onChange={(e) => setFormData({ ...formData, qbittorrent_pass: e.target.value })}
                  />
                </div>
              </div>
            </>
          )}

          {/* Configuration Deluge */}
          {formData.client_type === 'deluge' && (
            <>
              <div className="form-group">
                <label>URL Deluge</label>
                <input
                  type="text"
                  value={formData.deluge_url}
                  onChange={(e) => setFormData({ ...formData, deluge_url: e.target.value })}
                  placeholder="http://deluge:8112/json"
                />
              </div>
              <div className="form-group">
                <label>Mot de passe</label>
                <input
                  type="password"
                  value={formData.deluge_pass}
                  onChange={(e) => setFormData({ ...formData, deluge_pass: e.target.value })}
                />
              </div>
            </>
          )}

          {/* Configuration rTorrent */}
          {formData.client_type === 'rtorrent' && (
            <div className="form-group">
              <label>URL rTorrent XMLRPC</label>
              <input
                type="text"
                value={formData.rtorrent_url}
                onChange={(e) => setFormData({ ...formData, rtorrent_url: e.target.value })}
                placeholder="http://rtorrent:5000/RPC2"
              />
            </div>
          )}

          <div className="form-actions">
            <button onClick={() => handleTestClient(formData.client_type)} disabled={testing} className="test-button">
              {testing ? '⏳ Test en cours...' : '🔍 Tester la connexion'}
            </button>
            <button onClick={handleSave} className="save-button">
              💾 Sauvegarder
            </button>
          </div>
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
