import React, { useState, useRef, useEffect } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import './Navigation.css'

const Navigation = () => {
  // Protection contre les erreurs lors du hot reload
  let authContext
  try {
    authContext = useAuth()
  } catch (error) {
    console.warn('Navigation: Erreur lors de l\'accès au contexte d\'authentification:', error)
    // Valeurs par défaut si le contexte n'est pas disponible
    authContext = {
      user: null,
      token: null,
      loading: true,
      logout: () => {},
      isAdmin: () => false
    }
  }
  const { user, logout, isAdmin } = authContext
  const navigate = useNavigate()
  const location = useLocation()
  const [searchQuery, setSearchQuery] = useState('')
  const [showUserMenu, setShowUserMenu] = useState(false)
  const [showMobileMenu, setShowMobileMenu] = useState(false)
  const [showSearchBar, setShowSearchBar] = useState(false)
  const userMenuRef = useRef(null)
  const mobileMenuRef = useRef(null)
  const searchInputRef = useRef(null)

  // Fermer les menus quand on clique ailleurs
  useEffect(() => {
    const handleClickOutside = (event) => {
      // Vérifier si le clic est sur un élément du dropdown (lien, bouton, etc.)
      const isDropdownClick = event.target.closest('.dropdown-menu') || 
                             event.target.closest('.dropdown-item') ||
                             event.target.closest('a.dropdown-item') ||
                             event.target.closest('button.dropdown-item')
      
      // Ne pas fermer si on clique dans le menu
      if (isDropdownClick) {
        return
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
    // Utiliser un délai pour permettre aux clics sur les liens de fonctionner
    const timeoutId = setTimeout(() => {
      document.addEventListener('click', handleClickOutside, true)
    }, 100)
    return () => {
      clearTimeout(timeoutId)
      document.removeEventListener('click', handleClickOutside, true)
    }
  }, [showSearchBar, showUserMenu])

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

        {/* Barre de recherche animée - seulement si connecté */}
        {user && (
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
        )}

        {/* Bouton recherche qui ouvre la barre - seulement si connecté */}
        {user && (
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
        )}

        {/* Bouton Upload visible pour les admins */}
        {user && (user.role === 'admin' || isAdmin()) && (
          <Link 
            to="/upload" 
            className="nav-link nav-link-upload search-toggle-button"
            title="Uploader une vidéo"
          >
            ⬆️
          </Link>
        )}

        <div className="nav-links">
          {user ? (
            <>
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
                  <div className="dropdown-menu user-dropdown" onClick={(e) => e.stopPropagation()}>
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
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin');
                          }}
                        >
                          📊 Tableau de bord
                        </button>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/users');
                          }}
                        >
                          👥 Utilisateurs
                        </button>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/categories');
                          }}
                        >
                          📂 Catégories
                        </button>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">🎬 Gestion Vidéos</span>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/videos');
                          }}
                        >
                          📹 Gérer Vidéos
                        </button>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/upload');
                          }}
                        >
                          ⬆️ Upload Vidéo
                        </button>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/dvd-extract');
                          }}
                        >
                          📀 Extraire DVD
                        </button>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">📥 Torrents</span>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/torrents');
                          }}
                        >
                          📥 Gérer Torrents
                        </button>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/torrent-clients');
                          }}
                        >
                          ⚙️ Clients Torrent
                        </button>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/torrent-configs');
                          }}
                        >
                          ⚙️ Configurations
                        </button>
                        <div className="dropdown-divider"></div>
                        <span className="dropdown-section-title">🎛️ Paramètres</span>
                        <button
                          type="button"
                          className="dropdown-item" 
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setShowUserMenu(false);
                            navigate('/admin/media-settings');
                          }}
                        >
                          🎬 Paramètres Média
                        </button>
                        <div className="dropdown-divider"></div>
                      </>
                    )}
                    <button
                      type="button"
                      className="dropdown-item" 
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        setShowUserMenu(false);
                        navigate('/profile');
                      }}
                    >
                      👤 Mon Profil
                    </button>
                    <button
                      type="button"
                      className="dropdown-item" 
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        setShowUserMenu(false);
                        navigate('/favorites');
                      }}
                    >
                      ⭐ Favoris
                    </button>
                    <button
                      type="button"
                      className="dropdown-item" 
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        setShowUserMenu(false);
                        navigate('/history');
                      }}
                    >
                      📺 Historique
                    </button>
                    <div className="dropdown-divider"></div>
                    <button 
                      type="button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        handleLogout();
                      }} 
                      className="dropdown-item dropdown-item-danger"
                    >
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

