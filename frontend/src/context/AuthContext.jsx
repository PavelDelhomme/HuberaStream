import React, { createContext, useState, useContext, useEffect } from 'react'
import api from '../services/api'

const AuthContext = createContext(null)

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth doit être utilisé dans un AuthProvider')
  }
  return context
}

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null)
  const [token, setToken] = useState(null)
  const [loading, setLoading] = useState(true)

  // Vérifier le token au chargement initial
  useEffect(() => {
    const initAuth = async () => {
      const storedToken = localStorage.getItem('token')
      
      if (storedToken) {
        try {
          // Configurer l'API avec le token
          api.defaults.headers.common['Authorization'] = `Bearer ${storedToken}`
          
          // Vérifier que le token est valide en récupérant l'utilisateur
          const response = await api.get('/users/me')
          setUser(response.data)
          setToken(storedToken)
        } catch (error) {
          console.error('Token invalide ou expiré:', error)
          // Token invalide, nettoyer
          localStorage.removeItem('token')
          delete api.defaults.headers.common['Authorization']
          setToken(null)
          setUser(null)
        }
      }
      
      setLoading(false)
    }

    initAuth()
  }, []) // Exécuter une seule fois au montage

  const isAdmin = () => {
    return user && user.role === 'admin'
  }

  const login = async (email, password) => {
    try {
      const response = await api.post('/auth/login', { email, password })
      const { token: newToken, user: userData } = response.data
      
      // Sauvegarder le token
      localStorage.setItem('token', newToken)
      api.defaults.headers.common['Authorization'] = `Bearer ${newToken}`
      
      // Mettre à jour l'état
      setToken(newToken)
      setUser(userData)
      
      return { success: true }
    } catch (error) {
      const errorMessage = error.response?.data?.error || 'Erreur lors de la connexion'
      return {
        success: false,
        error: errorMessage,
        pendingApproval: error.response?.data?.pending_approval || false
      }
    }
  }

  const register = async (username, email, password) => {
    try {
      await api.post('/auth/register', { username, email, password })
      return { success: true }
    } catch (error) {
      return {
        success: false,
        error: error.response?.data?.error || 'Erreur lors de l\'inscription'
      }
    }
  }

  const logout = () => {
    setToken(null)
    setUser(null)
    localStorage.removeItem('token')
    delete api.defaults.headers.common['Authorization']
  }

  return (
    <AuthContext.Provider value={{ user, token, loading, login, register, logout, isAdmin }}>
      {children}
    </AuthContext.Provider>
  )
}
