import axios from 'axios'

// Utiliser l'URL via Nginx (port 3300) pour éviter les problèmes CORS
// Si on est sur localhost:3300, utiliser le chemin relatif /api
// Sinon, utiliser l'URL complète via Nginx
const getApiBaseURL = () => {
  const envURL = import.meta.env.VITE_API_URL
  if (envURL) {
    return envURL
  }
  // Si on accède via Nginx (port 3300), utiliser le chemin relatif
  if (typeof window !== 'undefined') {
    const origin = window.location.origin
    if (origin === 'http://localhost:3300' || origin === 'http://127.0.0.1:3300') {
      return '/api'
    }
  }
  // Sinon, utiliser l'URL complète via Nginx
  return 'http://localhost:3300/api'
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
