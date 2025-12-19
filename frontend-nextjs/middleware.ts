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

  // Pour les routes admin, on laisse le layout admin faire la vérification détaillée
  // Le middleware vérifie seulement la présence du token pour éviter les redirections en boucle
  // Le layout admin (client-side) vérifiera le rôle et la validité du token

  return NextResponse.next()
}

export const config = {
  matcher: [
    '/admin/:path*',
    '/user/:path*',
  ],
}

