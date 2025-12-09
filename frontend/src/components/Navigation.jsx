import React, { useState, useRef, useEffect } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import './Navigation.css'

const Navigation = () => {
  const { user, logout, isAdmin } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchQuery, setSearchQuery] = useState('')
  const [showAdminMenu, setShowAdminMenu] = useState(false)
  const [showUserMenu, setShowUserMenu] = useState(false)
  const [showMobileMenu, setShowMobileMenu] = useState(false)
  const [showSearchBar, setShowSearchBar] = useState(false)
  const adminMenuRef = useRef(null)
  const userMenuRef = useRef(null)
  const mobileMenuRef = useRef(null)
  const searchInputRef = useRef(null)

  // Fermer les menus quand on clique ailleurs
  useEffect(() => {
    const handleClickOutside = (event) => {
      // Vérifier si le clic est sur un lien dans le dropdown
      const isLinkClick = event.target.closest('a.dropdown-item') || 
                          event.target.closest('.dropdown-item')
      
      // Ne pas fermer si on clique sur un lien dans le menu
      if (isLinkClick) {
        return
      }
      
      if (adminMenuRef.current && !adminMenuRef.current.contains(event.target)) {
        setShowAdminMenu(false)
      }
      if (userMenuRef.current && !userMenuRef.current.contains(event.target)) {
        setShowUserMenu(false)
      }
      if (mobileMenuRef.current && !mobileMenuRef.current.contains(event.target)) {
        setShowMobileMenu(false)
      }
      // Fermer la barre de recherche si on clique ailleurs (mais pas sur le toggle)
      if (showSearchBar && 
          !event.target.closest('.search-container') && 
          !event.target.closest('.search-toggle-button')) {
        setShowSearchBar(false)
      }
    }
    // Utiliser 'click' au lieu de 'mousedown' pour permettre aux liens de fonctionner
    document.addEventListener('click', handleClickOutside, true)
    return () => document.removeEventListener('click', handleClickOutside, true)
  }, [showSearchBar])

  // Fermer le menu mobile quand on change de page
  useEffect(() => {
    setShowMobileMenu(false)
  }, [location.pathname])

  // Focus sur l'input de recherche quand la barre s'ouvre
  useEffect(() => {
    if (showSearchBar && searchInputRef.current) {
      setTimeout(() => {
        searchInputRef.current?.focus()
      }, 100)
    }
  }, [showSearchBar])

  const handleSearch = (e) => {
    e.preventDefault()
    e.stopPropagation()
    if (searchQuery.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchQuery)}`)
      setShowSearchBar(false)
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
              {(user?.role === 'admin' || isAdmin()) && (
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

        {/* Barre de recherche animée */}
        <div className={`search-container ${showSearchBar ? 'active' : ''}`}>
          <form className="nav-search" onSubmit={(e) => {
            handleSearch(e)
            setShowSearchBar(false)
          }}>
            <input
              ref={searchInputRef}
              type="text"
              placeholder="Rechercher des vidéos..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="search-input"
              onBlur={(e) => {
                // Ne pas fermer si on clique sur le bouton ou le toggle
                if (!e.relatedTarget?.closest('.search-button') && 
                    !e.relatedTarget?.closest('.search-toggle-button')) {
                  // Petit délai pour permettre le clic sur le bouton
                  setTimeout(() => {
                    if (!document.activeElement?.closest('.search-container') &&
                        !document.activeElement?.closest('.search-toggle-button')) {
                      setShowSearchBar(false)
                    }
                  }, 200)
                }
              }}
            />
            <button type="submit" className="search-button" aria-label="Rechercher">
              🔍
            </button>
          </form>
        </div>

        {/* Bouton recherche qui ouvre la barre */}
        <button 
          className={`search-toggle-button ${showSearchBar ? 'active' : ''}`}
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            setShowSearchBar(prev => {
              const newState = !prev
              console.log('Toggle search bar:', newState)
              return newState
            })
          }}
          type="button"
          aria-label="Rechercher"
        >
          🔍
        </button>

        {/* Bouton Upload visible pour les admins */}
        {user && (user.role === 'admin' || isAdmin()) && (
          <Link 
            to="/upload" 
            className="nav-link nav-link-upload"
            title="Uploader une vidéo"
          >
            ⬆️ Upload
          </Link>
        )}

        <div className="nav-links">
          {user ? (
            <>
              {(user?.role === 'admin' || isAdmin()) && (
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
                      <Link 
                        to="/admin" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin');
                        }}
                      >
                        📊 Tableau de bord
                      </Link>
                      <Link 
                        to="/admin/users" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/users');
                        }}
                      >
                        👥 Utilisateurs
                      </Link>
                      <Link 
                        to="/admin/torrents" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/torrents');
                        }}
                      >
                        📥 Torrents
                      </Link>
                      <Link 
                        to="/admin/torrent-configs" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/torrent-configs');
                        }}
                      >
                        ⚙️ Configurations Torrent
                      </Link>
                      <Link 
                        to="/admin/categories" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/categories');
                        }}
                      >
                        📂 Catégories
                      </Link>
                      <div className="dropdown-divider"></div>
                      <span className="dropdown-section-title">🎬 Gestion Vidéos</span>
                      <Link 
                        to="/admin/videos" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/videos');
                        }}
                      >
                        📹 Gérer Vidéos
                      </Link>
                      <Link 
                        to="/upload" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/upload');
                        }}
                      >
                        ⬆️ Upload Vidéo
                      </Link>
                      <Link 
                        to="/dvd-extract" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/dvd-extract');
                        }}
                      >
                        📀 Extraire DVD
                      </Link>
                      <div className="dropdown-divider"></div>
                      <span className="dropdown-section-title">📥 Torrents</span>
                      <Link 
                        to="/admin/torrents" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/torrents');
                        }}
                      >
                        📥 Gérer Torrents
                      </Link>
                      <Link 
                        to="/admin/torrent-configs" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/torrent-configs');
                        }}
                      >
                        ⚙️ Configurations
                      </Link>
                      <Link 
                        to="/admin/torrent-clients" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/torrent-clients');
                        }}
                      >
                        ⚙️ Clients Torrent
                      </Link>
                      <div className="dropdown-divider"></div>
                      <Link 
                        to="/admin/media-settings" 
                        className="dropdown-item" 
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowAdminMenu(false);
                          navigate('/admin/media-settings');
                        }}
                      >
                        🎬 Paramètres Média
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
                    <span className="dropdown-username">
                      {user.username}
                      {(() => {
                        const userIsAdmin = user?.role === 'admin' || (isAdmin && typeof isAdmin === 'function' && isAdmin());
                        return userIsAdmin;
                      })() && (
                        <span className="admin-badge">👑 Admin</span>
                      )}
                    </span>
                    {/* Menu Administration - Afficher si admin */}
                    {(() => {
                      // Vérifier le rôle de plusieurs façons pour une détection robuste
                      const userRole = user?.role;
                      const isAdminResult = isAdmin && typeof isAdmin === 'function' ? isAdmin() : false;
                      const userRoleString = String(userRole || '').toLowerCase();
                      const emailIsAdmin = user?.email === 'dev@delhomme.ovh'; // Email admin connu
                      
                      // Vérifications multiples pour garantir l'affichage
                      return userRole === 'admin' || 
                             userRoleString === 'admin' || 
                             isAdminResult ||
                             (user && user.role && String(user.role).toLowerCase() === 'admin') ||
                             emailIsAdmin; // Fallback: si c'est l'email admin, afficher le menu
                    })() && (
                      <>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">⚙️ Administration</span>
                        <Link 
                          to="/admin" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin');
                          }}
                        >
                          📊 Tableau de bord
                        </Link>
                        <Link 
                          to="/admin/users" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/users');
                          }}
                        >
                          👥 Utilisateurs
                        </Link>
                        <Link 
                          to="/admin/categories" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/categories');
                          }}
                        >
                          📂 Catégories
                        </Link>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">🎬 Gestion Vidéos</span>
                        <Link 
                          to="/admin/videos" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/videos');
                          }}
                        >
                          📹 Gérer Vidéos
                        </Link>
                        <Link 
                          to="/upload" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/upload');
                          }}
                        >
                          ⬆️ Upload Vidéo
                        </Link>
                        <Link 
                          to="/dvd-extract" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/dvd-extract');
                          }}
                        >
                          📀 Extraire DVD
                        </Link>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">📥 Torrents</span>
                        <Link 
                          to="/admin/torrents" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/torrents');
                          }}
                        >
                          📥 Gérer Torrents
                        </Link>
                        <Link 
                          to="/admin/torrent-clients" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/torrent-clients');
                          }}
                        >
                          ⚙️ Clients Torrent
                        </Link>
                        <Link 
                          to="/admin/torrent-configs" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/torrent-configs');
                          }}
                        >
                          ⚙️ Configurations
                        </Link>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">🎛️ Paramètres</span>
                        <Link 
                          to="/admin/media-settings" 
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/media-settings');
                          }}
                        >
                          🎬 Paramètres Média
                        </Link>
                        <div className="dropdown-divider"></div>
                      </>
                    )}
                    <Link 
                      to="/profile" 
                      className="dropdown-item" 
                      onClick={(e) => {
                        e.stopPropagation();
                        setShowUserMenu(false);
                        navigate('/profile');
                      }}
                    >
                      👤 Mon Profil
                    </Link>
                    <Link 
                      to="/favorites" 
                      className="dropdown-item" 
                      onClick={(e) => {
                        e.stopPropagation();
                        setShowUserMenu(false);
                        navigate('/favorites');
                      }}
                    >
                      ⭐ Favoris
                    </Link>
                    <Link 
                      to="/history" 
                      className="dropdown-item" 
                      onClick={(e) => {
                        e.stopPropagation();
                        setShowUserMenu(false);
                        navigate('/history');
                      }}
                    >
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

