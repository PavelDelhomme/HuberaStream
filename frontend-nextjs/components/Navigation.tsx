'use client'

import Link from 'next/link'
import { useEffect, useState, useRef } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import './Navigation.css'

export default function Navigation() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [showSearchBar, setShowSearchBar] = useState(false)
  const pathname = usePathname()
  const router = useRouter()
  const searchInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const loadUser = async () => {
      if (authService.isAuthenticated()) {
        try {
          const currentUser = await authService.getCurrentUser()
          if (currentUser) {
            setUser(currentUser)
          } else {
            // Token invalide, supprimer le cookie
            authService.logout()
            setUser(null)
          }
        } catch (error) {
          // Erreur lors de la récupération de l'utilisateur
          authService.logout()
          setUser(null)
        }
      } else {
        setUser(null)
      }
      setLoading(false)
    }
    loadUser()

    // Réécouter les changements de cookie (pour détecter les déconnexions)
    const checkAuth = () => {
      if (!authService.isAuthenticated()) {
        setUser(null)
      }
    }

    // Vérifier l'authentification toutes les secondes (pour détecter les changements)
    const interval = setInterval(checkAuth, 1000)

    return () => clearInterval(interval)
  }, [pathname]) // Recharger quand la route change

  // Focus sur l'input de recherche quand la barre s'ouvre
  useEffect(() => {
    if (showSearchBar && searchInputRef.current) {
      setTimeout(() => {
        searchInputRef.current?.focus()
      }, 100)
    }
  }, [showSearchBar])

  // Fermer la barre de recherche si on clique ailleurs
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (showSearchBar && 
          !(event.target as Element)?.closest('.search-container') && 
          !(event.target as Element)?.closest('.search-toggle-button')) {
        setShowSearchBar(false)
      }
    }
    document.addEventListener('click', handleClickOutside, true)
    return () => document.removeEventListener('click', handleClickOutside, true)
  }, [showSearchBar])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchQuery.trim()) {
      router.push(`/search?q=${encodeURIComponent(searchQuery)}`)
      setShowSearchBar(false)
      setSearchQuery('')
    }
  }

  const handleLogout = () => {
    authService.logout()
    setUser(null)
  }

  if (loading) {
    return <nav className="navigation">Chargement...</nav>
  }

  return (
    <nav className="navigation">
      <div className="nav-container">
        <Link href="/" className="nav-logo" prefetch={false}>
          🎬 StreamMake
        </Link>
        
        {user && (
          <div className="nav-main-links">
            <Link href="/" prefetch={false} className={`nav-link ${pathname === '/' ? 'active' : ''}`}>
              🏠 Accueil
            </Link>
            <Link href="/user/playlists" prefetch={false} className={`nav-link ${pathname.startsWith('/user/playlists') ? 'active' : ''}`}>
              📋 Mes Listes
            </Link>
            <Link href="/user/history" prefetch={false} className={`nav-link ${pathname.startsWith('/user/history') ? 'active' : ''}`}>
              📺 Historique
            </Link>
          </div>
        )}

        {/* Barre de recherche animée */}
        {user && (
          <div className={`search-container ${showSearchBar ? 'active' : ''}`}>
            <form className="nav-search" onSubmit={handleSearch}>
              <input
                ref={searchInputRef}
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
          </div>
        )}

        {/* Bouton recherche qui ouvre la barre */}
        {user && (
          <button 
            className={`search-toggle-button ${showSearchBar ? 'active' : ''}`}
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              setShowSearchBar(prev => !prev)
            }}
            type="button"
            aria-label="Rechercher"
          >
            🔍
          </button>
        )}

        <div className="nav-links">
          {user ? (
            <>
              {authService.isAdmin(user) && (
                <Link href="/video/upload" prefetch={false} className="nav-link nav-link-upload" title="Uploader une vidéo">
                  ⬆️
                </Link>
              )}
              {authService.isAdmin(user) && (
                <div className="nav-dropdown">
                  <button className="user-menu-button">
                    <span className="user-avatar">{user.email.charAt(0).toUpperCase()}</span>
                    <span className="username">{user.email.split('@')[0]}</span>
                    <span className="dropdown-arrow">▼</span>
                  </button>
                  <div className="dropdown-menu user-dropdown">
                    <span className="dropdown-username">
                      {user.email.split('@')[0]}
                      <span className="admin-badge">👑 Admin</span>
                    </span>
                    <div className="dropdown-divider"></div>
                    <span className="dropdown-section-title">⚙️ Administration</span>
                    <Link href="/admin/dashboard" prefetch={false} className="dropdown-item">
                      📊 Tableau de bord
                    </Link>
                    <Link href="/admin/users" prefetch={false} className="dropdown-item">
                      👥 Utilisateurs
                    </Link>
                    <Link href="/admin/categories" prefetch={false} className="dropdown-item">
                      📂 Catégories
                    </Link>
                    <div className="dropdown-divider"></div>
                    <span className="dropdown-section-title">🎬 Gestion Vidéos</span>
                    <Link href="/admin/videos" prefetch={false} className="dropdown-item">
                      📹 Gérer Vidéos
                    </Link>
                    <Link href="/video/upload" prefetch={false} className="dropdown-item">
                      ⬆️ Upload Vidéo
                    </Link>
                    <Link href="/dvd-extract" prefetch={false} className="dropdown-item">
                      📀 Extraire DVD
                    </Link>
                    <div className="dropdown-divider"></div>
                    <span className="dropdown-section-title">📥 Torrents</span>
                    <Link href="/admin/torrents" prefetch={false} className="dropdown-item">
                      📥 Gérer Torrents
                    </Link>
                    <Link href="/admin/torrent-clients" prefetch={false} className="dropdown-item">
                      ⚙️ Clients Torrent
                    </Link>
                    <Link href="/admin/torrent-configs" prefetch={false} className="dropdown-item">
                      ⚙️ Configurations
                    </Link>
                    <Link href="/admin/torrent-indexes" prefetch={false} className="dropdown-item">
                      🔍 Index de Torrents
                    </Link>
                    <div className="dropdown-divider"></div>
                    <span className="dropdown-section-title">🎛️ Paramètres</span>
                    <Link href="/admin/media-settings" prefetch={false} className="dropdown-item">
                      🎬 Paramètres Média
                    </Link>
                    <div className="dropdown-divider"></div>
                    <Link href="/user/profile" prefetch={false} className="dropdown-item">
                      👤 Mon Profil
                    </Link>
                    <Link href="/user/favorites" prefetch={false} className="dropdown-item">
                      ⭐ Favoris
                    </Link>
                    <Link href="/user/history" prefetch={false} className="dropdown-item">
                      📺 Historique
                    </Link>
                    <div className="dropdown-divider"></div>
                    <button 
                      onClick={handleLogout} 
                      className="dropdown-item dropdown-item-danger"
                    >
                      🚪 Déconnexion
                    </button>
                  </div>
                </div>
              )}
              {!authService.isAdmin(user) && (
                <>
                  <Link href="/user/profile" prefetch={false} className="nav-link">Profil</Link>
                  <button onClick={handleLogout} className="nav-button">
                    Déconnexion
                  </button>
                </>
              )}
            </>
          ) : (
            <>
              <Link href="/login" prefetch={false} className="nav-link">Connexion</Link>
              <Link href="/register" prefetch={false} className="nav-link nav-link-primary">Inscription</Link>
            </>
          )}
        </div>
      </div>
    </nav>
  )
}

