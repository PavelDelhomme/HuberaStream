import axios from 'axios'

// Configuration de l'URL de l'API
// En production, utiliser Nginx (port 3300) comme reverse proxy
// Le problème 403 Forbidden a été résolu en configurant SetTrustedProxies dans Gin
const getApiBaseURL = () => {
  const envURL = import.meta.env.VITE_API_URL
  if (envURL) {
    return envURL
  }
  // Utiliser Nginx comme reverse proxy (port 3300)
  // Si accès direct depuis le navigateur, utiliser le chemin relatif
  if (window.location.port === '3300' || window.location.hostname === 'localhost') {
    return '/api'
  }
  // Sinon, utiliser l'URL complète
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
    // Ne rediriger que si ce n'est pas la route de login et si c'est vraiment une erreur 401
    if (error.response?.status === 401 && !window.location.pathname.includes('/login')) {
      localStorage.removeItem('token')
      // Utiliser navigate au lieu de window.location pour éviter les rechargements
      // Mais ici on doit utiliser window.location car l'intercepteur n'a pas accès à navigate
      // On va laisser AuthContext gérer la redirection
    }
    return Promise.reject(error)
  }
)

export default api
