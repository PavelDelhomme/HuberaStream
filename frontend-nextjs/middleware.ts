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
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3301'
      const response = await fetch(`${apiUrl}/api/users/me`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (!response.ok || response.status === 401 || response.status === 403) {
        // Token invalide ou expiré, rediriger vers login
        const loginUrl = new URL('/login', request.url)
        loginUrl.searchParams.set('redirect', pathname)
        const response = NextResponse.redirect(loginUrl)
        // Supprimer le cookie invalide
        response.cookies.delete('token')
        return response
      }

      const user = await response.json()
      if (user.role !== 'admin') {
        // Utilisateur n'est pas admin, rediriger vers la page d'accueil
        return NextResponse.redirect(new URL('/', request.url))
      }
    } catch (error) {
      // Erreur lors de la vérification, rediriger vers login
      const loginUrl = new URL('/login', request.url)
      loginUrl.searchParams.set('redirect', pathname)
      const response = NextResponse.redirect(loginUrl)
      response.cookies.delete('token')
      return response
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

