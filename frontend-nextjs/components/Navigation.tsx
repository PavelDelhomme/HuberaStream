'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { authService, User } from '@/lib/auth'
import './Navigation.css'

export default function Navigation() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const loadUser = async () => {
      if (authService.isAuthenticated()) {
        const currentUser = await authService.getCurrentUser()
        setUser(currentUser)
      }
      setLoading(false)
    }
    loadUser()
  }, [])

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
        <Link href="/" className="nav-logo">
          🎬 StreamMake
        </Link>
        <div className="nav-links">
          <Link href="/">Accueil</Link>
          <Link href="/search">Recherche</Link>
          {user ? (
            <>
              {authService.isAdmin(user) && (
                <Link href="/admin/dashboard">Admin</Link>
              )}
              <Link href="/user/profile">Profil</Link>
              <Link href="/user/history">Historique</Link>
              <Link href="/user/favorites">Favoris</Link>
              <Link href="/user/playlists">Listes</Link>
              <button onClick={handleLogout} className="nav-button">
                Déconnexion
              </button>
            </>
          ) : (
            <>
              <Link href="/login">Connexion</Link>
              <Link href="/register">Inscription</Link>
            </>
          )}
        </div>
      </div>
    </nav>
  )
}

