import React, { useState, useRef, useEffect } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import './Navigation.css'

const Navigation = () => {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchQuery, setSearchQuery] = useState('')
  const [showAdminMenu, setShowAdminMenu] = useState(false)
  const [showUserMenu, setShowUserMenu] = useState(false)
  const [showMobileMenu, setShowMobileMenu] = useState(false)
  const [showMobileSearch, setShowMobileSearch] = useState(false)
  const adminMenuRef = useRef(null)
  const userMenuRef = useRef(null)
  const mobileMenuRef = useRef(null)

  // Fermer les menus quand on clique ailleurs
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (adminMenuRef.current && !adminMenuRef.current.contains(event.target)) {
        setShowAdminMenu(false)
      }
      if (userMenuRef.current && !userMenuRef.current.contains(event.target)) {
        setShowUserMenu(false)
      }
      if (mobileMenuRef.current && !mobileMenuRef.current.contains(event.target)) {
        setShowMobileMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Fermer le menu mobile quand on change de page
  useEffect(() => {
    setShowMobileMenu(false)
  }, [location.pathname])

  const handleSearch = (e) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchQuery)}`)
    }
  }

  const handleLogout = () => {
    logout()
    navigate('/')
    setShowUserMenu(false)
  }

  const isActive = (path) => {
    return location.pathname === path || location.pathname.startsWith(path + '/')
  }

  return (
    <nav className="navigation">
      <div className="nav-container">
        <Link to="/" className="nav-logo">
          <span className="logo-text">🎬 StreamMake</span>
        </Link>

        {/* Menu hamburger pour mobile */}
        {user && (
          <button 
            className="mobile-menu-toggle"
            onClick={() => setShowMobileMenu(!showMobileMenu)}
            aria-label="Toggle menu"
          >
            <span className={`hamburger ${showMobileMenu ? 'active' : ''}`}>
              <span></span>
              <span></span>
              <span></span>
            </span>
          </button>
        )}

        {user && (
          <div className="nav-main-links">
            <Link to="/" className={`nav-link ${isActive('/') && !location.pathname.startsWith('/admin') ? 'active' : ''}`}>
              🏠 Accueil
            </Link>
            <Link to="/playlists" className={`nav-link ${isActive('/playlists') ? 'active' : ''}`}>
              📋 Mes Listes
            </Link>
            <Link to="/history" className={`nav-link ${isActive('/history') ? 'active' : ''}`}>
              📺 Historique
            </Link>
          </div>
        )}

        {/* Menu mobile */}
        {user && showMobileMenu && (
          <div className="mobile-menu" ref={mobileMenuRef}>
            <div className="mobile-menu-content">
              <Link 
                to="/" 
                className={`mobile-menu-item ${isActive('/') && !location.pathname.startsWith('/admin') ? 'active' : ''}`}
                onClick={() => setShowMobileMenu(false)}
              >
                🏠 Accueil
              </Link>
              <Link 
                to="/playlists" 
                className={`mobile-menu-item ${isActive('/playlists') ? 'active' : ''}`}
                onClick={() => setShowMobileMenu(false)}
              >
                📋 Mes Listes
              </Link>
              <Link 
                to="/history" 
                className={`mobile-menu-item ${isActive('/history') ? 'active' : ''}`}
                onClick={() => setShowMobileMenu(false)}
              >
                📺 Historique
              </Link>
              <Link 
                to="/favorites" 
                className={`mobile-menu-item ${isActive('/favorites') ? 'active' : ''}`}
                onClick={() => setShowMobileMenu(false)}
              >
                ⭐ Favoris
              </Link>
              <Link 
                to="/profile" 
                className={`mobile-menu-item ${isActive('/profile') ? 'active' : ''}`}
                onClick={() => setShowMobileMenu(false)}
              >
                👤 Mon Profil
              </Link>
              {user.role === 'admin' && (
                <>
                  <div className="mobile-menu-divider"></div>
                  <Link 
                    to="/admin" 
                    className={`mobile-menu-item ${isActive('/admin') ? 'active' : ''}`}
                    onClick={() => setShowMobileMenu(false)}
                  >
                    ⚙️ Administration
                  </Link>
                </>
              )}
              <div className="mobile-menu-divider"></div>
              <button 
                onClick={() => {
                  handleLogout()
                  setShowMobileMenu(false)
                }}
                className="mobile-menu-item mobile-menu-item-danger"
              >
                🚪 Déconnexion
              </button>
            </div>
          </div>
        )}

        <form className={`nav-search ${showMobileSearch ? 'show' : ''}`} onSubmit={(e) => {
          handleSearch(e)
          setShowMobileSearch(false)
        }}>
          <input
            type="text"
            placeholder="Rechercher des vidéos..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="search-input"
          />
          <button type="submit" className="search-button" aria-label="Rechercher">
            🔍
          </button>
        </form>

        {/* Bouton recherche mobile */}
        <button 
          className="mobile-search-toggle"
          onClick={() => setShowMobileSearch(!showMobileSearch)}
          aria-label="Rechercher"
        >
          🔍
        </button>

        <div className="nav-links">
          {user ? (
            <>
              {user.role === 'admin' && (
                <div className="nav-dropdown" ref={adminMenuRef}>
                  <button
                    className={`nav-link nav-link-admin ${isActive('/admin') ? 'active' : ''}`}
                    onClick={() => setShowAdminMenu(!showAdminMenu)}
                    style={{ border: 'none', cursor: 'pointer' }}
                  >
                    ⚙️ Admin
                    <span className="dropdown-arrow">▼</span>
                  </button>
                  {showAdminMenu && (
                    <div className="dropdown-menu admin-dropdown">
                      <Link to="/admin" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        📊 Tableau de bord
                      </Link>
                      <Link to="/admin/users" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        👥 Utilisateurs
                      </Link>
                      <Link to="/admin/torrents" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        📥 Torrents
                      </Link>
                      <Link to="/admin/torrent-configs" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        ⚙️ Configurations Torrent
                      </Link>
                      <Link to="/admin/categories" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        📂 Catégories
                      </Link>
                      <Link to="/admin/torrent-clients" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        ⚙️ Clients Torrent
                      </Link>
                      <Link to="/admin/media-settings" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        🎬 Paramètres Média
                      </Link>
                      <div className="dropdown-divider"></div>
                      <Link to="/upload" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        ⬆️ Upload Vidéo
                      </Link>
                      <Link to="/dvd-extract" className="dropdown-item" onClick={() => setShowAdminMenu(false)}>
                        📀 Extraire DVD
                      </Link>
                    </div>
                  )}
                </div>
              )}
              <div className="nav-dropdown" ref={userMenuRef}>
                <button
                  className="user-menu-button"
                  onClick={() => setShowUserMenu(!showUserMenu)}
                >
                  <span className="user-avatar">{user.username.charAt(0).toUpperCase()}</span>
                  <span className="username">{user.username}</span>
                  <span className="dropdown-arrow">▼</span>
                </button>
                {showUserMenu && (
                  <div className="dropdown-menu user-dropdown">
                    <span className="dropdown-username">{user.username}</span>
                    {user.role === 'admin' && (
                      <>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">⚙️ Administration</span>
                        <Link to="/admin" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                          📊 Tableau de bord
                        </Link>
                        <Link to="/admin/users" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                          👥 Utilisateurs
                        </Link>
                        <Link to="/admin/torrents" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                          📥 Torrents
                        </Link>
                        <Link to="/admin/torrent-clients" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                          ⚙️ Clients Torrent
                        </Link>
                        <Link to="/admin/torrent-configs" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                          📥 Configurations Torrent
                        </Link>
                        <Link to="/admin/media-settings" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                          🎬 Paramètres Média
                        </Link>
                        <Link to="/admin/categories" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                          📂 Catégories
                        </Link>
                        <div className="dropdown-divider"></div>
                      </>
                    )}
                    <Link to="/profile" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                      👤 Mon Profil
                    </Link>
                    <Link to="/favorites" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                      ⭐ Favoris
                    </Link>
                    <Link to="/history" className="dropdown-item" onClick={() => setShowUserMenu(false)}>
                      📺 Historique
                    </Link>
                    <div className="dropdown-divider"></div>
                    <button onClick={handleLogout} className="dropdown-item dropdown-item-danger">
                      🚪 Déconnexion
                    </button>
                  </div>
                )}
              </div>
            </>
          ) : (
            <>
              <Link to="/login" className="nav-link">
                🔐 Connexion
              </Link>
              <Link to="/register" className="nav-link nav-link-primary">
                ✨ Inscription
              </Link>
            </>
          )}
        </div>
      </div>
    </nav>
  )
}

export default Navigation

