'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import Link from 'next/link'
import './AdminLayout.css'

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const checkAuth = async () => {
      // Vérifier d'abord si le token existe
      if (!authService.isAuthenticated()) {
        router.push('/login?redirect=/admin')
        return
      }

      // Récupérer l'utilisateur actuel
      const currentUser = await authService.getCurrentUser()
      
      // Si getCurrentUser retourne null, cela peut être dû à :
      // 1. Token invalide/expiré (401) -> getCurrentUser supprime déjà le token
      // 2. Erreur réseau temporaire -> ne pas rediriger immédiatement
      if (!currentUser) {
        // Vérifier si le token existe encore (si non, c'est qu'il a été supprimé par getCurrentUser)
        if (!authService.isAuthenticated()) {
          router.push('/login?redirect=/admin')
          return
        }
        // Sinon, c'est probablement une erreur réseau, réessayer après un délai
        setTimeout(() => {
          checkAuth()
        }, 1000)
        return
      }

      // Vérifier que l'utilisateur est admin
      if (!authService.isAdmin(currentUser)) {
        router.push('/')
        return
      }

      setUser(currentUser)
      setLoading(false)
    }
    checkAuth()
  }, [router])

  if (loading) {
    return (
      <div className="admin-layout">
        <div className="loading">Vérification des permissions...</div>
      </div>
    )
  }

  if (!user || !authService.isAdmin(user)) {
    return null
  }

  return (
    <div className="admin-layout">
      <div className="admin-sidebar">
        <h2>🔧 Administration</h2>
        <nav className="admin-nav">
          <Link href="/admin/dashboard">📊 Tableau de bord</Link>
          <Link href="/admin/users">👥 Utilisateurs</Link>
          <Link href="/admin/videos">🎬 Vidéos</Link>
          <Link href="/admin/categories">📁 Catégories</Link>
          <Link href="/admin/media-settings">🎛️ Paramètres</Link>
        </nav>
      </div>
      <div className="admin-content">
        {children}
      </div>
    </div>
  )
}

