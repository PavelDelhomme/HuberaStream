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
      
      // Si pas de token ou token vide, ne rien faire (utilisateur non connecté)
      if (!storedToken || storedToken.trim() === '' || storedToken === 'null' || storedToken === 'undefined') {
        setLoading(false)
        setUser(null)
        setToken(null)
        return
      }
      
      // Vérifier que le token a l'air valide (format JWT basique)
      const trimmedToken = storedToken.trim()
      if (trimmedToken.length < 20 || !trimmedToken.includes('.')) {
        // Token invalide, nettoyer silencieusement
        localStorage.removeItem('token')
        setLoading(false)
        setUser(null)
        setToken(null)
        return
      }
      
      // Si on arrive ici, il y a un token qui semble valide
      if (trimmedToken) {
        try {
          // Configurer l'API avec le token
          api.defaults.headers.common['Authorization'] = `Bearer ${trimmedToken}`
          
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
              // Si on obtient un 404, cela signifie que la route n'existe pas encore
              // Attendre un peu et réessayer (le backend est peut-être en train de démarrer)
              // Ne pas logger d'avertissement - c'est peut-être temporaire
              if (response.status === 404) {
                if (attempt < 2) {
                  await new Promise(resolve => setTimeout(resolve, 2000 * (attempt + 1)))
                  continue
                }
                // Si après 3 tentatives on obtient toujours 404, considérer comme erreur réseau
                throw new Error('Route non trouvée (404) - le backend est peut-être en cours de démarrage')
              }
            } catch (err) {
              lastError = err
              // Si c'est une erreur d'authentification (401/403), arrêter immédiatement
              if (err.response?.status === 401 || err.response?.status === 403) {
                throw err
              }
              // Si c'est un 404, traiter comme une erreur réseau (backend peut être en cours de démarrage)
              // Ne pas logger d'avertissement - c'est peut-être temporaire
              if (err.response?.status === 404) {
                if (attempt < 2) {
                  await new Promise(resolve => setTimeout(resolve, 2000 * (attempt + 1)))
                  continue
                }
                throw new Error('Route non trouvée (404) - le backend est peut-être en cours de démarrage')
              }
              // Sinon, attendre un peu avant de réessayer
              if (attempt < 2) {
                await new Promise(resolve => setTimeout(resolve, 1000 * (attempt + 1)))
              }
            }
          }
          
          if (response && response.status === 200 && response.data) {
            setUser(response.data)
            setToken(trimmedToken)
            // Mettre à jour le token dans localStorage au cas où il aurait été modifié
            try {
              localStorage.setItem('token', trimmedToken)
            } catch (e) {
              // Ignorer les erreurs de stockage
            }
          } else {
            // Token invalide ou expiré
            throw new Error('Token invalide')
          }
        } catch (error) {
          // Seulement nettoyer le token si c'est vraiment une erreur d'authentification
          if (error.response?.status === 401 || error.response?.status === 403) {
            // Vérifier si c'est vraiment une erreur d'authentification ou juste le backend qui démarre
            // Ne nettoyer le token que si on est sûr que c'est invalide
            const errorMessage = error.response?.data?.error || ''
            if (errorMessage.includes('Token invalide') || errorMessage.includes('expiré') || errorMessage.includes('expired')) {
              // Token vraiment invalide ou expiré - nettoyer
              localStorage.removeItem('token')
              delete api.defaults.headers.common['Authorization']
              setToken(null)
              setUser(null)
            } else {
              // Peut-être juste le backend qui démarre - garder le token
              setToken(storedToken)
            }
          } else if (error.response?.status === 404) {
            // Si on obtient un 404, le backend est peut-être en cours de démarrage
            // Ne pas nettoyer le token, mais ne pas définir l'utilisateur non plus
            // Ne pas logger d'erreur - c'est peut-être temporaire
            setToken(storedToken)
            // Ne pas définir user pour l'instant
          } else {
            // Pour les autres erreurs (réseau, timeout), garder le token
            // Ne pas logger d'erreur - c'est peut-être temporaire
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
      
      // Sauvegarder le token de manière persistante
      // Utiliser try-catch pour s'assurer que le stockage fonctionne
      try {
        localStorage.setItem('token', newToken)
        // Stocker aussi la date de connexion pour debug et persistance
        localStorage.setItem('token_created', new Date().toISOString())
        if (userData?.role === 'admin') {
          localStorage.setItem('is_admin', 'true')
        } else {
          localStorage.removeItem('is_admin')
        }
      } catch (storageError) {
        console.error('Erreur lors du stockage du token:', storageError)
        // Continuer quand même, le token est dans la mémoire
      }
      
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
    // Nettoyer tous les éléments liés à l'authentification
    localStorage.removeItem('token')
    localStorage.removeItem('token_created')
    localStorage.removeItem('is_admin')
    delete api.defaults.headers.common['Authorization']
  }

  return (
    <AuthContext.Provider value={{ user, token, loading, login, register, logout, isAdmin }}>
      {children}
    </AuthContext.Provider>
  )
}
