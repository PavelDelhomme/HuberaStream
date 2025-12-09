import React, { createContext, useState, useContext, useEffect } from 'react'
import api from '../services/api'

const AuthContext = createContext(null)

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    // En mode développement avec hot reload, retourner des valeurs par défaut
    // au lieu de lancer une erreur pour éviter les crashes
    if (import.meta.env.DEV) {
      console.warn('useAuth appelé en dehors d\'un AuthProvider - valeurs par défaut utilisées')
      return {
        user: null,
        token: null,
        loading: true,
        login: async () => ({ success: false, error: 'AuthProvider non disponible' }),
        register: async () => ({ success: false, error: 'AuthProvider non disponible' }),
        logout: () => {},
        isAdmin: () => false
      }
    }
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
          // Utiliser un timeout plus long et plusieurs tentatives
          let response
          let lastError
          
          // Essayer jusqu'à 3 fois avec des timeouts progressifs
          for (let attempt = 0; attempt < 3; attempt++) {
            try {
              response = await api.get('/users/me', {
                timeout: (attempt + 1) * 10000, // 10s, 20s, 30s
                validateStatus: (status) => status < 500
              })
              
              if (response.status === 200 && response.data) {
                break // Succès, sortir de la boucle
              }
            } catch (err) {
              lastError = err
              // Si c'est une erreur d'authentification (401/403), arrêter immédiatement
              if (err.response?.status === 401 || err.response?.status === 403) {
                throw err
              }
              // Sinon, attendre un peu avant de réessayer
              if (attempt < 2) {
                await new Promise(resolve => setTimeout(resolve, 1000 * (attempt + 1)))
              }
            }
          }
          
          if (response && response.status === 200 && response.data) {
            setUser(response.data)
            setToken(storedToken)
          } else {
            // Token invalide ou expiré
            throw new Error('Token invalide')
          }
        } catch (error) {
          // Seulement nettoyer le token si c'est vraiment une erreur d'authentification
          if (error.response?.status === 401 || error.response?.status === 403) {
            console.error('Token invalide ou expiré:', error)
            localStorage.removeItem('token')
            delete api.defaults.headers.common['Authorization']
            setToken(null)
            setUser(null)
          } else {
            // Pour les autres erreurs (réseau, timeout), garder le token ET définir un utilisateur temporaire
            // Cela évite les redirections intempestives
            console.warn('Erreur réseau lors de la vérification du token, conservation du token:', error)
            // Garder le token - l'utilisateur sera vérifié lors de la prochaine requête
            setToken(storedToken)
            // Ne pas définir user pour l'instant, mais ne pas non plus nettoyer le token
            // Cela permettra aux requêtes suivantes de fonctionner si le serveur est disponible
          }
        }
      }
      
      setLoading(false)
    }

    initAuth()
  }, []) // Exécuter une seule fois au montage

  const isAdmin = () => {
    if (!user) return false
    // Vérifier le rôle dans user.role
    if (user.role === 'admin') return true
    // Fallback : si l'email est dev@delhomme.ovh, considérer comme admin
    if (user.email === 'dev@delhomme.ovh') return true
    // Fallback : si le username est 'admin', considérer comme admin
    if (user.username === 'admin') return true
    return false
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
