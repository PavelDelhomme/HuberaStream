import React, { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import api from '../../services/api'
import LoadingSpinner from '../../components/LoadingSpinner'
import CandidatesModal from '../../components/CandidatesModal'
import './AdminTorrentConfigs.css'

const AdminTorrentConfigs = () => {
  const { user, isAdmin, loading: authLoading } = useAuth()
  const navigate = useNavigate()
  const [configs, setConfigs] = useState([])
  const [loading, setLoading] = useState(true)
  const [showAddForm, setShowAddForm] = useState(false)
  const [editingConfig, setEditingConfig] = useState(null)
  const [editFormData, setEditFormData] = useState(null)
  const [triggeringSearch, setTriggeringSearch] = useState({})
  const [formData, setFormData] = useState({
    name: '',
    source: '1337x',
    enabled: true,
    auto_download: false,
    search_keywords: '',
    base_url: 'https://1337x.quest',
    search_url: '',
    category_id: '',
    sub_category_id: '',
    min_seeders: 1,
    max_size: 0,
    quality: '',
    language: '',
    check_interval: 3600
  })
  const [categories, setCategories] = useState([])
  const [selectedConfigs, setSelectedConfigs] = useState(new Set())
  const [showCandidatesModal, setShowCandidatesModal] = useState(false)
  const [selectedConfigForModal, setSelectedConfigForModal] = useState(null)

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
    fetchCategories()
  }, [user, authLoading, isAdmin, navigate])

  const fetchConfigs = async () => {
    try {
      setLoading(true)
      const response = await api.get('/admin/torrent-configs')
      setConfigs(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    } finally {
      setLoading(false)
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

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      // Préparer les données en convertissant les chaînes vides en null pour les UUID
      const dataToSend = {
        ...formData,
        category_id: formData.category_id && formData.category_id !== '' ? formData.category_id : null,
        sub_category_id: formData.sub_category_id && formData.sub_category_id !== '' ? formData.sub_category_id : null,
        // Convertir max_size de GB en bytes si nécessaire (déjà fait dans le formulaire)
        max_size: formData.max_size || 0
      }
      
      await api.post('/admin/torrent-configs', dataToSend)
      setShowAddForm(false)
      setFormData({
        name: '',
        source: '1337x',
        enabled: true,
        auto_download: false,
        search_keywords: '',
        base_url: 'https://1337x.quest',
        search_url: '',
        category_id: '',
        sub_category_id: '',
        min_seeders: 1,
        max_size: 0,
        quality: '',
        language: '',
        check_interval: 3600
      })
      fetchConfigs()
    } catch (error) {
      console.error('Erreur:', error)
      // Afficher le message d'erreur détaillé si disponible
      const errorMessage = error.response?.data?.error || error.message || 'Erreur lors de la création'
      alert(`Erreur: ${errorMessage}`)
    }
  }

  const handleToggle = async (configId, field, value) => {
    try {
      await api.put(`/admin/torrent-configs/${configId}`, { [field]: value })
      fetchConfigs()
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const handleTriggerSearch = async (configId) => {
    try {
      setTriggeringSearch(prev => ({ ...prev, [configId]: true }))
      await api.post(`/admin/torrent-configs/${configId}/trigger-search`)
      
      // Rafraîchir plusieurs fois pour voir les résultats apparaître progressivement
      // La recherche peut prendre quelques secondes
      setTimeout(() => {
        fetchConfigs()
        // Déclencher un événement pour rafraîchir le modal s'il est ouvert
        window.dispatchEvent(new CustomEvent('candidatesSearchCompleted', { detail: { configId } }))
      }, 2000)
      setTimeout(() => {
        fetchConfigs()
        window.dispatchEvent(new CustomEvent('candidatesSearchCompleted', { detail: { configId } }))
      }, 5000)
      setTimeout(() => {
        fetchConfigs()
        window.dispatchEvent(new CustomEvent('candidatesSearchCompleted', { detail: { configId } }))
      }, 10000)
      
      // Message avec lien vers les candidats
      const config = configs.find(c => c.id === configId)
      const configName = config ? config.name : 'cette configuration'
      alert(`Recherche déclenchée pour "${configName}" !\n\nLes résultats seront disponibles dans quelques instants.\n\nVous pouvez cliquer sur "Voir tous les candidats" pour voir les résultats trouvés.`)
    } catch (error) {
      console.error('Erreur lors du déclenchement de la recherche:', error)
      const errorMsg = error.response?.data?.error || 'Erreur lors du déclenchement de la recherche'
      alert(`Erreur: ${errorMsg}`)
    } finally {
      setTriggeringSearch(prev => ({ ...prev, [configId]: false }))
    }
  }

  const handleSelectConfig = (configId) => {
    setSelectedConfigs(prev => {
      const newSet = new Set(prev)
      if (newSet.has(configId)) {
        newSet.delete(configId)
      } else {
        newSet.add(configId)
      }
      return newSet
    })
  }

  const handleSelectAll = () => {
    if (selectedConfigs.size === configs.length) {
      setSelectedConfigs(new Set())
    } else {
      setSelectedConfigs(new Set(configs.map(c => c.id)))
    }
  }

  const handleBulkDisable = async () => {
    if (selectedConfigs.size === 0) {
      alert('Veuillez sélectionner au moins une configuration')
      return
    }
    if (!confirm(`Désactiver ${selectedConfigs.size} configuration(s) ?`)) return

    try {
      const promises = Array.from(selectedConfigs).map(configId =>
        api.put(`/admin/torrent-configs/${configId}`, { enabled: false })
      )
      await Promise.all(promises)
      setSelectedConfigs(new Set())
      fetchConfigs()
      alert(`${selectedConfigs.size} configuration(s) désactivée(s)`)
    } catch (error) {
      console.error('Erreur lors de la désactivation:', error)
      alert('Erreur lors de la désactivation')
    }
  }

  const handleBulkEnable = async () => {
    if (selectedConfigs.size === 0) {
      alert('Veuillez sélectionner au moins une configuration')
      return
    }
    if (!confirm(`Activer ${selectedConfigs.size} configuration(s) ?`)) return

    try {
      const promises = Array.from(selectedConfigs).map(configId =>
        api.put(`/admin/torrent-configs/${configId}`, { enabled: true })
      )
      await Promise.all(promises)
      setSelectedConfigs(new Set())
      fetchConfigs()
      alert(`${selectedConfigs.size} configuration(s) activée(s)`)
    } catch (error) {
      console.error('Erreur lors de l\'activation:', error)
      alert('Erreur lors de l\'activation')
    }
  }

  const handleBulkDelete = async () => {
    if (selectedConfigs.size === 0) {
      alert('Veuillez sélectionner au moins une configuration')
      return
    }
    if (!confirm(`Supprimer définitivement ${selectedConfigs.size} configuration(s) ? Cette action est irréversible.`)) return

    try {
      const promises = Array.from(selectedConfigs).map(configId =>
        api.delete(`/admin/torrent-configs/${configId}`)
      )
      await Promise.all(promises)
      setSelectedConfigs(new Set())
      fetchConfigs()
      alert(`${selectedConfigs.size} configuration(s) supprimée(s)`)
    } catch (error) {
      console.error('Erreur lors de la suppression:', error)
      alert('Erreur lors de la suppression')
    }
  }

  const handleEdit = (config) => {
    setEditingConfig(config.id)
    setEditFormData({
      name: config.name,
      source: config.source,
      enabled: config.enabled,
      auto_download: config.auto_download,
      search_keywords: config.search_keywords || '',
      base_url: config.base_url || '',
      search_url: config.search_url || '',
      category_id: config.category_id || '',
      sub_category_id: config.sub_category_id || '',
      min_seeders: config.min_seeders || 1,
      max_size: config.max_size || 0,
      quality: config.quality || '',
      language: config.language || '',
      check_interval: config.check_interval || 3600
    })
  }

  const handleCancelEdit = () => {
    setEditingConfig(null)
    setEditFormData(null)
  }

  const handleUpdate = async (e) => {
    e.preventDefault()
    try {
      const dataToSend = {
        ...editFormData,
        category_id: editFormData.category_id && editFormData.category_id !== '' ? editFormData.category_id : null,
        sub_category_id: editFormData.sub_category_id && editFormData.sub_category_id !== '' ? editFormData.sub_category_id : null,
        enabled: editFormData.enabled,
        auto_download: editFormData.auto_download
      }
      
      await api.put(`/admin/torrent-configs/${editingConfig}`, dataToSend)
      setEditingConfig(null)
      setEditFormData(null)
      fetchConfigs()
    } catch (error) {
      console.error('Erreur lors de la mise à jour:', error)
      const errorMessage = error.response?.data?.error || error.message || 'Erreur lors de la mise à jour'
      alert(`Erreur: ${errorMessage}`)
    }
  }

  if (loading) {
    return (
      <div className="admin-torrent-configs">
        <LoadingSpinner message="Chargement des configurations..." />
      </div>
    )
  }

  return (
    <div className="admin-torrent-configs">
      <div className="admin-header">
        <h1>Configurations de téléchargement automatique</h1>
        <button onClick={() => setShowAddForm(!showAddForm)} className="add-button">
          {showAddForm ? 'Annuler' : '+ Nouvelle configuration'}
        </button>
      </div>

      {configs.length > 0 && (
        <div className="selection-toolbar">
          <label className="select-all-checkbox">
            <input
              type="checkbox"
              checked={selectedConfigs.size === configs.length && configs.length > 0}
              onChange={handleSelectAll}
            />
            <span className="checkbox-custom"></span>
            <span>Sélectionner tout ({configs.length})</span>
          </label>
          {selectedConfigs.size > 0 && (
            <div className="bulk-actions">
              <span className="selected-count">{selectedConfigs.size} sélectionné(s)</span>
              <button onClick={handleBulkEnable} className="btn-bulk btn-bulk-enable">
                ✅ Activer
              </button>
              <button onClick={handleBulkDisable} className="btn-bulk btn-bulk-disable">
                ⏸️ Désactiver
              </button>
              <button onClick={handleBulkDelete} className="btn-bulk btn-bulk-delete">
                🗑️ Supprimer
              </button>
            </div>
          )}
        </div>
      )}

      {showAddForm && (
        <form className="add-config-form" onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Nom de la configuration"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            required
          />
          <select
            value={formData.source}
            onChange={(e) => setFormData({ ...formData, source: e.target.value })}
          >
            <option value="1337x">1337x</option>
            <option value="piratebay">The Pirate Bay</option>
            <option value="torrentdownload">TorrentDownload</option>
            <option value="rarbg">RARBG</option>
            <option value="yts">YTS</option>
            <option value="nyaa">Nyaa</option>
            <option value="custom">Custom</option>
          </select>
          <div className="form-group">
            <label>URL de base du site torrent (optionnel)</label>
            <input
              type="text"
              placeholder="https://1337x.quest ou https://1337x.proxyninja.net"
              value={formData.base_url}
              onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
            />
            <small>Exemples: https://1337x.quest, https://1337x.proxyninja.net, https://thepiratebay.org, https://torrentdownload.proxyninja.net</small>
          </div>
          <div className="form-group">
            <label>URL de recherche/liste (optionnel)</label>
            <input
              type="text"
              placeholder="/watch-movie/ ou /cat/Movies/1/"
              value={formData.search_url}
              onChange={(e) => setFormData({ ...formData, search_url: e.target.value })}
            />
            <small>Si rempli, cette URL sera parsée directement au lieu d'utiliser les mots-clés. Exemple: /watch-movie/ pour parser la page de films</small>
          </div>
          <input
            type="text"
            placeholder="Mots-clés de recherche (séparés par virgules) - ignoré si URL de recherche remplie"
            value={formData.search_keywords}
            onChange={(e) => setFormData({ ...formData, search_keywords: e.target.value })}
          />
          <div className="form-row">
            <select
              value={formData.category_id}
              onChange={(e) => setFormData({ ...formData, category_id: e.target.value || null })}
            >
              <option value="">Aucune catégorie</option>
              {categories.map(cat => (
                <option key={cat.id} value={cat.id}>{cat.name}</option>
              ))}
            </select>
            {formData.category_id && (
              <select
                value={formData.sub_category_id}
                onChange={(e) => setFormData({ ...formData, sub_category_id: e.target.value || null })}
              >
                <option value="">Aucune sous-catégorie</option>
                {categories.find(c => c.id === formData.category_id)?.sub_categories?.map(sub => (
                  <option key={sub.id} value={sub.id}>{sub.name}</option>
                ))}
              </select>
            )}
          </div>
          <div className="form-row">
            <input
              type="text"
              placeholder="Qualité (1080p, 720p, etc.)"
              value={formData.quality}
              onChange={(e) => setFormData({ ...formData, quality: e.target.value })}
            />
            <input
              type="text"
              placeholder="Langue (FR, EN, etc.)"
              value={formData.language}
              onChange={(e) => setFormData({ ...formData, language: e.target.value })}
            />
          </div>
          <input
            type="number"
            placeholder="Taille maximum (en GB, 0 = illimité)"
            value={formData.max_size ? formData.max_size / (1024 * 1024 * 1024) : ''}
            onChange={(e) => setFormData({ ...formData, max_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 * 1024 : 0 })}
            min="0"
          />
          <div className="form-row">
            <label>
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
              />
              Activé
            </label>
            <label>
              <input
                type="checkbox"
                checked={formData.auto_download}
                onChange={(e) => setFormData({ ...formData, auto_download: e.target.checked })}
              />
              Téléchargement automatique
            </label>
          </div>
          <div className="form-row">
            <input
              type="number"
              placeholder="Seeders minimum"
              value={formData.min_seeders}
              onChange={(e) => setFormData({ ...formData, min_seeders: parseInt(e.target.value) })}
              min="1"
            />
            <input
              type="number"
              placeholder="Intervalle de vérification (secondes)"
              value={formData.check_interval}
              onChange={(e) => setFormData({ ...formData, check_interval: parseInt(e.target.value) })}
              min="60"
            />
          </div>
          <button type="submit">Créer</button>
        </form>
      )}

      <div className="configs-list">
        {configs.map((config) => (
          <div key={config.id} className={`config-card ${selectedConfigs.has(config.id) ? 'selected' : ''}`}>
            <label className="config-checkbox-label">
              <input
                type="checkbox"
                checked={selectedConfigs.has(config.id)}
                onChange={() => handleSelectConfig(config.id)}
                className="config-checkbox"
              />
              <span className="checkbox-custom"></span>
            </label>
            {editingConfig === config.id && editFormData ? (
              <form className="edit-config-form" onSubmit={handleUpdate}>
                <h3>Modifier la configuration</h3>
                <input
                  type="text"
                  placeholder="Nom de la configuration"
                  value={editFormData.name}
                  onChange={(e) => setEditFormData({ ...editFormData, name: e.target.value })}
                  required
                />
                <select
                  value={editFormData.source}
                  onChange={(e) => setEditFormData({ ...editFormData, source: e.target.value })}
                >
                  <option value="1337x">1337x</option>
                  <option value="piratebay">The Pirate Bay</option>
                  <option value="torrentdownload">TorrentDownload</option>
                  <option value="rarbg">RARBG</option>
                  <option value="yts">YTS</option>
                  <option value="nyaa">Nyaa</option>
                  <option value="custom">Custom</option>
                </select>
                <div className="form-group">
                  <label>URL de base du site torrent (optionnel)</label>
                  <input
                    type="text"
                    placeholder="https://1337x.quest"
                    value={editFormData.base_url}
                    onChange={(e) => setEditFormData({ ...editFormData, base_url: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label>URL de recherche/liste (optionnel)</label>
                  <input
                    type="text"
                    placeholder="/watch-movie/ ou /cat/Movies/1/"
                    value={editFormData.search_url}
                    onChange={(e) => setEditFormData({ ...editFormData, search_url: e.target.value })}
                  />
                </div>
                <input
                  type="text"
                  placeholder="Mots-clés de recherche (séparés par virgules)"
                  value={editFormData.search_keywords}
                  onChange={(e) => setEditFormData({ ...editFormData, search_keywords: e.target.value })}
                />
                <div className="form-row">
                  <select
                    value={editFormData.category_id}
                    onChange={(e) => setEditFormData({ ...editFormData, category_id: e.target.value || null })}
                  >
                    <option value="">Aucune catégorie</option>
                    {categories.map(cat => (
                      <option key={cat.id} value={cat.id}>{cat.name}</option>
                    ))}
                  </select>
                  {editFormData.category_id && (
                    <select
                      value={editFormData.sub_category_id}
                      onChange={(e) => setEditFormData({ ...editFormData, sub_category_id: e.target.value || null })}
                    >
                      <option value="">Aucune sous-catégorie</option>
                      {categories.find(c => c.id === editFormData.category_id)?.sub_categories?.map(sub => (
                        <option key={sub.id} value={sub.id}>{sub.name}</option>
                      ))}
                    </select>
                  )}
                </div>
                <div className="form-row">
                  <input
                    type="text"
                    placeholder="Qualité (1080p, 720p, etc.)"
                    value={editFormData.quality}
                    onChange={(e) => setEditFormData({ ...editFormData, quality: e.target.value })}
                  />
                  <input
                    type="text"
                    placeholder="Langue (FR, EN, etc.)"
                    value={editFormData.language}
                    onChange={(e) => setEditFormData({ ...editFormData, language: e.target.value })}
                  />
                </div>
                <input
                  type="number"
                  placeholder="Taille maximum (en GB, 0 = illimité)"
                  value={editFormData.max_size ? editFormData.max_size / (1024 * 1024 * 1024) : ''}
                  onChange={(e) => setEditFormData({ ...editFormData, max_size: e.target.value ? parseInt(e.target.value) * 1024 * 1024 * 1024 : 0 })}
                  min="0"
                />
                <div className="form-row">
                  <label>
                    <input
                      type="checkbox"
                      checked={editFormData.enabled}
                      onChange={(e) => setEditFormData({ ...editFormData, enabled: e.target.checked })}
                    />
                    Activé
                  </label>
                  <label>
                    <input
                      type="checkbox"
                      checked={editFormData.auto_download}
                      onChange={(e) => setEditFormData({ ...editFormData, auto_download: e.target.checked })}
                    />
                    Téléchargement automatique
                  </label>
                </div>
                <div className="form-row">
                  <input
                    type="number"
                    placeholder="Seeders minimum"
                    value={editFormData.min_seeders}
                    onChange={(e) => setEditFormData({ ...editFormData, min_seeders: parseInt(e.target.value) })}
                    min="1"
                  />
                  <input
                    type="number"
                    placeholder="Intervalle de vérification (secondes)"
                    value={editFormData.check_interval}
                    onChange={(e) => setEditFormData({ ...editFormData, check_interval: parseInt(e.target.value) })}
                    min="60"
                  />
                </div>
                <div className="form-actions">
                  <button type="submit" className="btn-save">💾 Enregistrer</button>
                  <button type="button" onClick={handleCancelEdit} className="btn-cancel">❌ Annuler</button>
                </div>
              </form>
            ) : (
              <>
                <div className="config-header">
                  <h3>{config.name}</h3>
                  <span className="config-source">{config.source}</span>
                </div>
                <div className="config-details">
                  {config.search_url ? (
                    <p><strong>URL de recherche:</strong> {config.base_url || 'https://1337x.quest'}{config.search_url}</p>
                  ) : (
                    <p><strong>Mots-clés:</strong> {config.search_keywords || 'Aucun'}</p>
                  )}
                  {config.base_url && <p><strong>URL de base:</strong> {config.base_url}</p>}
                  {config.category && <p><strong>Catégorie:</strong> {config.category.name}</p>}
                  {config.sub_category && <p><strong>Sous-catégorie:</strong> {config.sub_category.name}</p>}
                  {config.quality && <p><strong>Qualité:</strong> {config.quality}</p>}
                  {config.language && <p><strong>Langue:</strong> {config.language}</p>}
                  <p><strong>Seeders minimum:</strong> {config.min_seeders}</p>
                  {config.max_size > 0 && <p><strong>Taille max:</strong> {(config.max_size / (1024 * 1024 * 1024)).toFixed(2)} GB</p>}
                  <p><strong>Vérification:</strong> Toutes les {config.check_interval} secondes</p>
                  {config.last_check && <p><strong>Dernière vérification:</strong> {new Date(config.last_check).toLocaleString('fr-FR')}</p>}
                  
                  {/* Statistiques des candidats */}
                  <div className="candidates-stats">
                    <p className="stats-title"><strong>📊 Résultats de recherche:</strong></p>
                    {config.candidates_stats ? (
                      <>
                        <div className="stats-grid">
                          <div className="stat-item">
                            <span className="stat-label">Total:</span>
                            <span className="stat-value">{config.candidates_stats.total || 0}</span>
                          </div>
                          <div className="stat-item pending">
                            <span className="stat-label">En attente:</span>
                            <span className="stat-value">{config.candidates_stats.pending || 0}</span>
                          </div>
                          <div className="stat-item approved">
                            <span className="stat-label">Approuvés:</span>
                            <span className="stat-value">{config.candidates_stats.approved || 0}</span>
                          </div>
                          <div className="stat-item rejected">
                            <span className="stat-label">Rejetés:</span>
                            <span className="stat-value">{config.candidates_stats.rejected || 0}</span>
                          </div>
                          <div className="stat-item downloaded">
                            <span className="stat-label">Téléchargés:</span>
                            <span className="stat-value">{config.candidates_stats.downloaded || 0}</span>
                          </div>
                        </div>
                        {config.candidates_stats.total > 0 ? (
                          <button
                            onClick={() => {
                              setSelectedConfigForModal({ id: config.id, name: config.name })
                              setShowCandidatesModal(true)
                            }}
                            className="view-candidates-link"
                          >
                            👁️ Voir tous les candidats ({config.candidates_stats.total})
                          </button>
                        ) : (
                          <div className="no-candidates-message">
                            <span>ℹ️ Aucun candidat trouvé pour le moment. Lancez une recherche pour commencer.</span>
                          </div>
                        )}
                      </>
                    ) : (
                      <div className="no-candidates-message">
                        <span>ℹ️ Aucun candidat trouvé pour le moment. Lancez une recherche pour commencer.</span>
                      </div>
                    )}
                  </div>
                </div>
                <div className="config-actions">
                  <button
                    className={`action-button btn-trigger-search ${triggeringSearch[config.id] ? 'loading' : ''}`}
                    onClick={() => handleTriggerSearch(config.id)}
                    disabled={triggeringSearch[config.id] || !config.enabled}
                    title="Déclencher une recherche maintenant"
                  >
                    <span className="action-icon">
                      {triggeringSearch[config.id] ? '⏳' : '🔍'}
                    </span>
                    <span className="action-text">
                      {triggeringSearch[config.id] ? 'Recherche...' : 'Lancer recherche'}
                    </span>
                  </button>
                  <button
                    className="action-button btn-edit"
                    onClick={() => handleEdit(config)}
                    title="Modifier la configuration"
                  >
                    <span className="action-icon">✏️</span>
                    <span className="action-text">Modifier</span>
                  </button>
                  <label className={`toggle-button ${config.enabled ? 'enabled' : 'disabled'}`}>
                    <input
                      type="checkbox"
                      checked={config.enabled}
                      onChange={(e) => handleToggle(config.id, 'enabled', e.target.checked)}
                    />
                    <span className="toggle-checkbox-custom"></span>
                    <span className="toggle-text">{config.enabled ? 'Activé' : 'Désactivé'}</span>
                  </label>
                  <label className={`toggle-button ${config.auto_download ? 'enabled' : 'disabled'}`}>
                    <input
                      type="checkbox"
                      checked={config.auto_download}
                      onChange={(e) => handleToggle(config.id, 'auto_download', e.target.checked)}
                    />
                    <span className="toggle-checkbox-custom"></span>
                    <span className="toggle-text">{config.auto_download ? 'Auto-download activé' : 'Auto-download désactivé'}</span>
                  </label>
                </div>
              </>
            )}
          </div>
        ))}
      </div>

      {/* Modal des candidats */}
      {showCandidatesModal && selectedConfigForModal && (
        <CandidatesModal
          configId={selectedConfigForModal.id}
          configName={selectedConfigForModal.name}
          onClose={() => {
            setShowCandidatesModal(false)
            setSelectedConfigForModal(null)
            fetchConfigs() // Rafraîchir pour mettre à jour les stats
          }}
        />
      )}
    </div>
  )
}

export default AdminTorrentConfigs

