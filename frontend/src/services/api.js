import axios from 'axios'

// WORKAROUND TEMPORAIRE: Utiliser directement le backend (port 3301) au lieu de Nginx (port 3300)
// pour contourner le problème 403 Forbidden avec les requêtes POST via Nginx
// TODO: Résoudre le problème avec Gin v1.11.0 et SetTrustedProxies pour réactiver Nginx
const getApiBaseURL = () => {
  const envURL = import.meta.env.VITE_API_URL
  if (envURL) {
    return envURL
  }
  // Utiliser directement le backend (port 3301) au lieu de Nginx
  return 'http://localhost:3301/api'
}

const api = axios.create({
  baseURL: getApiBaseURL(),
  headers: {
    'Content-Type': 'application/json',
  },
})

// Intercepteur pour ajouter le token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Intercepteur pour gérer les erreurs
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api
