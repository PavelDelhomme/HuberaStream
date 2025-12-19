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
    
    // Stocker le token dans un cookie HTTP-only via le backend
    // Pour l'instant, on utilise js-cookie côté client
    Cookies.set('token', token, { expires: 7 }) // 7 jours
    
    return { token, user }
  },

  async register(email: string, password: string, name?: string): Promise<AuthResponse> {
    const response = await api.post('/auth/register', { email, password, name })
    const { token, user } = response.data
    
    Cookies.set('token', token, { expires: 7 })
    
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
      const response = await api.get('/users/me')
      return response.data
    } catch (error) {
      return null
    }
  },

  isAdmin(user: User | null): boolean {
    return user?.role === 'admin' || false
  },
}

