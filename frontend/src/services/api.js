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
  // Si accès depuis le navigateur (localhost ou IP locale), utiliser le chemin relatif
  const hostname = window.location.hostname
  const port = window.location.port
  
  // Détecter si on est sur localhost ou une IP locale (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
  const isLocalNetwork = hostname === 'localhost' || 
                         hostname === '127.0.0.1' ||
                         hostname.startsWith('192.168.') ||
                         hostname.startsWith('10.') ||
                         /^172\.(1[6-9]|2[0-9]|3[0-1])\./.test(hostname)
  
  if (port === '3300' || isLocalNetwork) {
    return '/api'
  }
  // Sinon, utiliser l'URL complète avec le hostname actuel
  return `${window.location.protocol}//${hostname}:${port || '3300'}/api`
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
