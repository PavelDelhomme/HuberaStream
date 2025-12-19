import Cookies from 'js-cookie'
import api from './api'

export interface User {
  id: string
  email: string
  role: 'admin' | 'user'
  is_active: boolean
}

export interface AuthResponse {
  token: string
  user: User
}

export const authService = {
  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await api.post('/auth/login', { email, password })
    const { token, user } = response.data
    
    // Stocker le token dans un cookie avec persistance sécurisée
    // Expiration : 30 jours pour admin, 7 jours pour user (selon backend)
    const expires = user.role === 'admin' ? 30 : 7
    Cookies.set('token', token, { 
      expires,
      sameSite: 'lax', // Permet l'envoi du cookie en cross-site pour les liens
      secure: false, // En développement localhost, secure doit être false
      path: '/', // Accessible sur tout le site
    })
    
    return { token, user }
  },

  async register(email: string, password: string, name?: string): Promise<AuthResponse> {
    const response = await api.post('/auth/register', { email, password, name })
    const { token, user } = response.data
    
    const expires = user.role === 'admin' ? 30 : 7
    Cookies.set('token', token, { 
      expires,
      sameSite: 'lax',
      secure: false, // En développement localhost, secure doit être false
      path: '/',
    })
    
    return { token, user }
  },

  logout(): void {
    Cookies.remove('token')
    if (typeof window !== 'undefined') {
      window.location.href = '/login'
    }
  },

  getToken(): string | undefined {
    return Cookies.get('token')
  },

  isAuthenticated(): boolean {
    return !!Cookies.get('token')
  },

  async getCurrentUser(): Promise<User | null> {
    try {
      const token = this.getToken()
      if (!token) {
        return null
      }
      const response = await api.get('/users/me')
      return response.data
    } catch (error: any) {
      // Si erreur 401 ou 403, supprimer le token (token invalide/expiré)
      if (error.response?.status === 401 || error.response?.status === 403) {
        Cookies.remove('token', { path: '/' })
        return null
      }
      // Pour les autres erreurs (réseau, 500, etc.), retourner null mais garder le token
      // L'appelant pourra réessayer
      console.error('[Auth] Erreur lors de la récupération de l\'utilisateur:', error.response?.status || error.message)
      return null
    }
  },

  isAdmin(user: User | null): boolean {
    return user?.role === 'admin' || false
  },
}

