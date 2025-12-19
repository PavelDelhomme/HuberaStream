import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Routes admin nécessitant une authentification
  const adminRoutes = ['/admin']
  const isAdminRoute = adminRoutes.some(route => pathname.startsWith(route))

  // Routes utilisateur nécessitant une authentification
  const userRoutes = ['/user']
  const isUserRoute = userRoutes.some(route => pathname.startsWith(route))

  // Vérifier le token dans les cookies
  const token = request.cookies.get('token')?.value

  // Rediriger vers login si accès à une route protégée sans token
  if ((isAdminRoute || isUserRoute) && !token) {
    const loginUrl = new URL('/login', request.url)
    loginUrl.searchParams.set('redirect', pathname)
    return NextResponse.redirect(loginUrl)
  }

  // Pour les routes admin, vérifier aussi que le token est valide et que l'utilisateur est admin
  if (isAdminRoute && token) {
    try {
      // Utiliser la même URL que dans api.ts
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3300/api'
      const response = await fetch(`${apiUrl}/users/me`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        // Ne pas suivre les redirections automatiquement
        redirect: 'manual',
      })

      // Seulement rediriger si c'est vraiment une erreur d'authentification
      if (response.status === 401 || response.status === 403) {
        // Token invalide ou expiré, rediriger vers login
        const loginUrl = new URL('/login', request.url)
        loginUrl.searchParams.set('redirect', pathname)
        const redirectResponse = NextResponse.redirect(loginUrl)
        // Supprimer le cookie invalide
        redirectResponse.cookies.delete('token')
        return redirectResponse
      }

      // Si la réponse est OK, vérifier le rôle
      if (response.ok) {
        const user = await response.json()
        if (user.role !== 'admin') {
          // Utilisateur n'est pas admin, rediriger vers la page d'accueil
          return NextResponse.redirect(new URL('/', request.url))
        }
      }
      // Si la réponse n'est pas OK mais pas 401/403, laisser passer (erreur réseau temporaire)
      // Le composant client vérifiera l'authentification de toute façon
    } catch (error) {
      // Erreur réseau ou autre : ne pas rediriger automatiquement
      // Laisser le composant client gérer l'authentification
      // Cela évite les redirections en boucle en cas d'erreur réseau temporaire
      console.error('[Middleware] Erreur lors de la vérification du token:', error)
      // Ne pas rediriger, laisser passer la requête
    }
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    '/admin/:path*',
    '/user/:path*',
  ],
}

