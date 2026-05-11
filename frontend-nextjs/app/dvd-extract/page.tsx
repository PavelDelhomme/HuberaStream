'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter } from 'next/navigation'
import { authService, User } from '@/lib/auth'
import api from '@/lib/api'
import './DVDExtract.css'

export default function DVDExtractPage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)
  const [step, setStep] = useState(1)
  const [device, setDevice] = useState<string | null>(null)
  const [availableDevices, setAvailableDevices] = useState<string[]>([])
  const [manualDevice, setManualDevice] = useState('')
  const [dvdInfo, setDvdInfo] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [subCategoryId, setSubCategoryId] = useState('')
  const [isPublic, setIsPublic] = useState(false)
  const [thumbnailFile, setThumbnailFile] = useState<File | null>(null)
  const [thumbnailPreview, setThumbnailPreview] = useState<string | null>(null)
  const [metadataResults, setMetadataResults] = useState<any[]>([])
  const [showMetadataResults, setShowMetadataResults] = useState(false)
  const [selectedMetadata, setSelectedMetadata] = useState<any>(null)
  const [searchingMetadata, setSearchingMetadata] = useState(false)
  const [selectedVideoTrack, setSelectedVideoTrack] = useState(0)
  const [selectedAudioTracks, setSelectedAudioTracks] = useState<number[]>([])
  const [selectedSubtitleTracks, setSelectedSubtitleTracks] = useState<number[]>([])
  const [qualities, setQualities] = useState<string[]>(['1080p', '720p', '480p'])
  const [availableQualities, setAvailableQualities] = useState<string[]>(['2160p', '1440p', '1080p', '720p', '480p', '360p'])
  const [categories, setCategories] = useState<any[]>([])
  const [subCategories, setSubCategories] = useState<any[]>([])
  const [extracting, setExtracting] = useState(false)
  const [extractionProgress, setExtractionProgress] = useState(0)
  const [extractionStage, setExtractionStage] = useState('')
  const [extractionDetails, setExtractionDetails] = useState('')
  const [videoId, setVideoId] = useState<string | null>(null)
  const [analyzingProgress, setAnalyzingProgress] = useState(0)
  const [analyzingStage, setAnalyzingStage] = useState('')
  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const [cancelling, setCancelling] = useState(false)
  const [authLoading, setAuthLoading] = useState(true)
  const authCheckTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const hasCheckedAuthRef = useRef(false)

  useEffect(() => {
    // Éviter les vérifications multiples
    if (hasCheckedAuthRef.current) {
      console.log('[DVD-Extract] ⚠️ Vérification déjà en cours, ignoré')
      return
    }

    let isMounted = true

    const checkAuth = async () => {
      hasCheckedAuthRef.current = true
      console.log('[DVD-Extract] 🔍 Début vérification authentification')
      
      // Attendre un peu pour s'assurer que les cookies sont chargés (problème Next.js SSR)
      await new Promise(resolve => setTimeout(resolve, 100))
      
      // Vérifier d'abord si le token existe
      const hasToken = authService.isAuthenticated()
      const tokenValue = authService.getToken()
      console.log('[DVD-Extract] Token présent:', hasToken, 'Token value:', tokenValue ? `${tokenValue.substring(0, 20)}...` : 'null')
      
      if (!hasToken) {
        console.log('[DVD-Extract] ❌ Pas de token, redirection vers login')
        if (isMounted) {
          hasCheckedAuthRef.current = false
          router.push('/login?redirect=/dvd-extract')
        }
        return
      }

      console.log('[DVD-Extract] 📡 Appel getCurrentUser...')
      // Récupérer l'utilisateur actuel
      const currentUser = await authService.getCurrentUser()
      console.log('[DVD-Extract] Utilisateur récupéré:', currentUser ? `ID: ${currentUser.id}, Role: ${currentUser.role}` : 'null')
      
      // Si le composant a été démonté, ne pas continuer
      if (!isMounted) {
        console.log('[DVD-Extract] ⚠️ Composant démonté, arrêt')
        return
      }
      
      // Si getCurrentUser retourne null, cela peut être dû à :
      // 1. Token invalide/expiré (401) -> getCurrentUser supprime déjà le token
      // 2. Erreur réseau temporaire -> ne pas rediriger immédiatement
      if (!currentUser) {
        console.log('[DVD-Extract] ⚠️ Utilisateur null, vérification token...')
        // Vérifier si le token existe encore (si non, c'est qu'il a été supprimé par getCurrentUser)
        const stillHasToken = authService.isAuthenticated()
        console.log('[DVD-Extract] Token toujours présent:', stillHasToken)
        
        if (!stillHasToken) {
          console.log('[DVD-Extract] ❌ Token supprimé, redirection vers login')
          hasCheckedAuthRef.current = false
          router.push('/login?redirect=/dvd-extract')
          return
        }
        // Sinon, c'est probablement une erreur réseau, réessayer après un délai
        console.log('[DVD-Extract] 🔄 Réessai dans 1 seconde...')
        hasCheckedAuthRef.current = false // Permettre le réessai
        if (authCheckTimeoutRef.current) {
          clearTimeout(authCheckTimeoutRef.current)
        }
        authCheckTimeoutRef.current = setTimeout(() => {
          if (isMounted) {
            checkAuth()
          }
        }, 1000)
        return
      }

      // Vérifier que l'utilisateur est admin (requis pour l'extraction DVD)
      const isAdmin = authService.isAdmin(currentUser)
      console.log('[DVD-Extract] Vérification admin:', isAdmin, 'Role:', currentUser.role)
      
      if (!isAdmin) {
        console.log('[DVD-Extract] ❌ Pas admin, redirection vers accueil')
        hasCheckedAuthRef.current = false
        router.push('/')
        return
      }

      console.log('[DVD-Extract] ✅ Authentification OK, chargement page')
      setUser(currentUser)
      setAuthLoading(false)
      fetchCategories()
      fetchMediaSettings()
      checkForActiveExtraction()
    }
    
    checkAuth()

    return () => {
      console.log('[DVD-Extract] 🧹 Nettoyage useEffect')
      isMounted = false
      hasCheckedAuthRef.current = false
      if (authCheckTimeoutRef.current) {
        clearTimeout(authCheckTimeoutRef.current)
        authCheckTimeoutRef.current = null
      }
    }
  }, [router])

  useEffect(() => {
    if (categoryId) {
      fetchSubCategories(categoryId)
    } else {
      setSubCategories([])
    }
  }, [categoryId, categories])

  useEffect(() => {
    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current)
      }
    }
  }, [])

  const fetchCategories = async () => {
    try {
      const response = await api.get('/categories')
      setCategories(response.data || [])
    } catch (error) {
      console.error('Erreur:', error)
    }
  }

  const fetchSubCategories = async (categoryId: string) => {
    try {
      const category = categories.find((c: any) => c.id === categoryId)
      if (category && category.sub_categories) {
        setSubCategories(category.sub_categories)
      } else {
        setSubCategories([])
      }
    } catch (error) {
      console.error('Erreur:', error)
      setSubCategories([])
    }
  }

  const fetchMediaSettings = async () => {
    try {
      const response = await api.get('/admin/media-settings')
      const settings = response.data
      
      // Récupérer les qualités disponibles depuis les paramètres
      const configuredQualities = settings.transcoding_qualities || []
      if (configuredQualities.length > 0) {
        setAvailableQualities(configuredQualities)
        // Définir les qualités par défaut parmi celles disponibles
        const defaultQualities = ['1080p', '720p', '480p'].filter(q => configuredQualities.includes(q))
        if (defaultQualities.length > 0) {
          setQualities(defaultQualities)
        } else {
          // Si aucune qualité par défaut n'est disponible, prendre la première
          setQualities([configuredQualities[0]])
        }
      } else {
        // Par défaut, toutes les qualités sont disponibles
        const allQualities = ['2160p', '1440p', '1080p', '720p', '480p', '360p']
        setAvailableQualities(allQualities)
        setQualities(['1080p', '720p', '480p'])
      }
    } catch (error) {
      console.error('Erreur lors de la récupération des paramètres média:', error)
      // En cas d'erreur, utiliser les valeurs par défaut
      const allQualities = ['2160p', '1440p', '1080p', '720p', '480p', '360p']
      setAvailableQualities(allQualities)
      setQualities(['1080p', '720p', '480p'])
    }
  }

  const checkForActiveExtraction = async () => {
    try {
      const response = await api.get('/videos?status=processing')
      const processingVideos = response.data.videos || []
      const dvdVideos = processingVideos.filter((video: any) => {
        const createdAt = new Date(video.created_at)
        const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000)
        return createdAt > twoHoursAgo && video.status === 'processing'
      })

      if (dvdVideos.length > 0) {
        const latestVideo = dvdVideos.sort((a: any, b: any) =>
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )[0]

        setVideoId(latestVideo.id)
        setExtracting(true)
        setStep(4)
        setExtractionProgress(5)
        setExtractionStage('Extraction en cours...')
        startPolling(latestVideo.id)
      }
    } catch (error) {
      console.error('Erreur lors de la vérification:', error)
    }
  }

  const startPolling = (videoIdToPoll: string) => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current)
    }

    const pollStatus = setInterval(async () => {
      try {
        const videoResponse = await api.get(`/videos/${videoIdToPoll}/status`)
        const videoStatus = videoResponse.data.status
        const videoProgress = videoResponse.data.progress || 0

        setExtractionProgress(videoProgress)

        if (videoStatus === 'processing') {
          if (videoProgress < 30) {
            setExtractionStage('Extraction du DVD...')
            setExtractionDetails(`Lecture des pistes (${Math.round(videoProgress)}%)`)
          } else if (videoProgress < 50) {
            setExtractionStage('Compression optimale...')
            setExtractionDetails(`Compression H.265/HEVC (${Math.round(videoProgress)}%)`)
          } else if (videoProgress < 90) {
            setExtractionStage('Transcodage multi-qualités...')
            setExtractionDetails(`Génération des formats (${Math.round(videoProgress)}%)`)
          } else if (videoProgress >= 100) {
            // Si le pourcentage est à 100% mais que le statut est encore "processing"
            setExtractionStage('⏳ Finalisation en cours...')
            setExtractionDetails('Finalisation des fichiers, optimisation et génération des formats...')
          } else {
            setExtractionStage('Finalisation...')
            setExtractionDetails(`Finalisation (${Math.round(videoProgress)}%)`)
          }
        } else if (videoStatus === 'ready') {
          clearInterval(pollStatus)
          pollIntervalRef.current = null
          setExtractionProgress(100)
          setExtractionStage('Extraction terminée !')
          
          // Uploader la miniature si elle a été sélectionnée (fichier ou URL depuis métadonnées)
          if (videoIdToPoll) {
            try {
              setExtractionDetails('Upload de la miniature...')
              
              if (thumbnailFile) {
                // Upload depuis un fichier
                const thumbnailData = new FormData()
                thumbnailData.append('thumbnail', thumbnailFile)
                await api.post(`/videos/${videoIdToPoll}/thumbnail`, thumbnailData, {
                  headers: {
                    'Content-Type': 'multipart/form-data',
                  },
                })
              } else if (selectedMetadata?.poster_url) {
                // Upload depuis une URL (métadonnées)
                const thumbnailData = new FormData()
                thumbnailData.append('thumbnail_url', selectedMetadata.poster_url)
                await api.post(`/videos/${videoIdToPoll}/thumbnail`, thumbnailData, {
                  headers: {
                    'Content-Type': 'multipart/form-data',
                  },
                })
              }
            } catch (error) {
              console.error('Erreur lors de l\'upload de la miniature:', error)
            }
          }
          
          setTimeout(() => {
            router.push(`/video/${videoIdToPoll}`)
          }, 2000)
        } else if (videoStatus === 'error' || videoStatus === 'cancelled') {
          clearInterval(pollStatus)
          pollIntervalRef.current = null
          setError('Erreur lors de l\'extraction')
          setExtracting(false)
          setStep(3)
        }
      } catch (err) {
        console.error('Erreur lors du polling:', err)
      }
    }, 2000)

    pollIntervalRef.current = pollStatus
  }

  const handleDetectDVD = async () => {
    setLoading(true)
    setError('')
    setAvailableDevices([])
    setManualDevice('')
    try {
      const response = await api.get('/dvd/detect')
      console.log('📀 Réponse détection DVD:', response.data)
      
      if (response.data.success && response.data.device) {
        setDevice(response.data.device)
        setStep(2)
        setError('')
      } else {
        // Même si la détection automatique échoue, proposer les devices disponibles
        if (response.data.available_devices && response.data.available_devices.length > 0) {
          setAvailableDevices(response.data.available_devices)
          setManualDevice(response.data.available_devices[0])
          setError(response.data.message || response.data.error || 'Aucun lecteur DVD détecté automatiquement, mais des lecteurs sont disponibles. Sélectionnez-en un manuellement.')
        } else {
          setError(response.data.message || response.data.error || 'Aucun lecteur DVD détecté. Assurez-vous qu\'un DVD/CD est inséré dans le lecteur et que le conteneur Docker a accès aux devices.')
        }
      }
    } catch (error: any) {
      console.error('❌ Erreur détection DVD:', error)
      const errorMessage = error.response?.data?.message ||
        error.response?.data?.error ||
        'Erreur lors de la détection du lecteur DVD. Vérifiez que le conteneur Docker a accès aux devices (/dev/sr*, /dev/dvd, etc.).'
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  const handleManualDeviceSelect = () => {
    if (manualDevice) {
      setDevice(manualDevice)
      setStep(2)
    } else {
      setError('Veuillez sélectionner un lecteur DVD')
    }
  }

  const handleGetDVDInfo = async () => {
    setLoading(true)
    setError('')
    setAnalyzingProgress(0)
    setAnalyzingStage('Initialisation de l\'analyse...')

    try {
      const progressInterval = setInterval(() => {
        setAnalyzingProgress((prev) => {
          if (prev >= 90) return prev
          return prev + 5
        })
      }, 500)

      const stageMessages = [
        'Lecture du device DVD...',
        'Analyse des pistes vidéo...',
        'Détection des pistes audio...',
        'Recherche des sous-titres...',
        'Extraction des métadonnées...',
        'Finalisation...'
      ]
      let stageIndex = 0
      const stageInterval = setInterval(() => {
        if (stageIndex < stageMessages.length - 1) {
          stageIndex++
          setAnalyzingStage(stageMessages[stageIndex])
        }
      }, 2000)

      const response = await api.get(`/dvd/info?device=${encodeURIComponent(device!)}`, {
        timeout: 60000
      })

      clearInterval(progressInterval)
      clearInterval(stageInterval)
      setAnalyzingProgress(90)
      setAnalyzingStage('Analyse terminée, recherche de métadonnées...')

      if (response.data) {
        const dvdData = {
          ...response.data,
          video_tracks: response.data.video_tracks || [],
          audio_tracks: response.data.audio_tracks || [],
          subtitle_tracks: response.data.subtitle_tracks || [],
          title: response.data.title || 'DVD Extraite',
        }

        setDvdInfo(dvdData)
        setTitle(dvdData.title)
        setSelectedVideoTrack(0)
        setSelectedAudioTracks(dvdData.audio_tracks.length > 0 ? [0] : [])
        setSelectedSubtitleTracks([])
        
        // Rechercher automatiquement les métadonnées depuis le titre du DVD
        if (dvdData.title && dvdData.title.trim() !== '' && dvdData.title !== 'DVD Extraite') {
          // Passer le titre, le réalisateur et l'année si disponibles
          // Le backend priorisera automatiquement les résultats qui correspondent
          setAnalyzingStage('Recherche de métadonnées en cours...')
          await searchMetadataFromTitle(
            dvdData.title,
            dvdData.director || undefined,
            dvdData.year || undefined
          )
        }
        
        setAnalyzingProgress(100)
        setAnalyzingStage('Analyse terminée !')
        setStep(3)
      } else {
        setError('Aucune information trouvée sur le DVD.')
        setAnalyzingProgress(0)
        setAnalyzingStage('')
      }
    } catch (error: any) {
      const errorMessage = error.response?.data?.message ||
        error.response?.data?.error ||
        'Erreur lors de la lecture du DVD.'
      setError(errorMessage)
    } finally {
      setLoading(false)
      setAnalyzingProgress(0)
      setAnalyzingStage('')
    }
  }

  const toggleAudioTrack = (index: number) => {
    setSelectedAudioTracks((prev) =>
      prev.includes(index)
        ? prev.filter((i) => i !== index)
        : [...prev, index]
    )
  }

  const toggleSubtitleTrack = (index: number) => {
    setSelectedSubtitleTracks((prev) =>
      prev.includes(index)
        ? prev.filter((i) => i !== index)
        : [...prev, index]
    )
  }

  const toggleQuality = (quality: string) => {
    setQualities((prev) =>
      prev.includes(quality)
        ? prev.filter((q) => q !== quality)
        : [...prev, quality]
    )
  }

  const decodeHtmlEntities = (text: string) => {
    if (!text || typeof text !== 'string') return text
    const textarea = document.createElement('textarea')
    textarea.innerHTML = text
    return textarea.value
  }

  const searchMetadataFromTitle = async (title: string, director?: string, year?: number, preserveExistingResults: boolean = false) => {
    if (!title || title.trim() === '') return

    setSearchingMetadata(true)
    // Ne pas réinitialiser les résultats si on veut les préserver (pour recherche en cascade)
    if (!preserveExistingResults) {
      setMetadataResults([])
      setShowMetadataResults(false)
    }
    setAnalyzingStage('Recherche de métadonnées en cours...')

    try {
      // Construire la requête avec le titre, le réalisateur et l'année si disponibles
      const searchPayload: any = {
        filename: title
      }
      
      // Ajouter le réalisateur si fourni en paramètre ou disponible dans les infos du DVD
      const directorToUse = director || dvdInfo?.director
      if (directorToUse) {
        searchPayload.director = directorToUse
      }
      
      // Ajouter l'année si fournie en paramètre ou disponible dans les infos du DVD
      const yearToUse = year || dvdInfo?.year
      if (yearToUse) {
        searchPayload.year = yearToUse
      }

      const response = await api.post('/admin/videos/search-metadata', searchPayload)

      const results = response.data.results || []
      if (results.length > 0) {
        // Si on préserve les résultats existants et qu'on a déjà des résultats, les fusionner
        if (preserveExistingResults && metadataResults.length > 0) {
          // Fusionner les résultats en évitant les doublons (basé sur imdb_id)
          const existingIds = new Set(metadataResults.map((r: any) => r.imdb_id).filter(Boolean))
          const newResults = results.filter((r: any) => !r.imdb_id || !existingIds.has(r.imdb_id))
          setMetadataResults([...metadataResults, ...newResults])
        } else {
          setMetadataResults(results)
        }
        setShowMetadataResults(true) // Afficher automatiquement les résultats
        setAnalyzingStage('Métadonnées trouvées !')
        
        // Ne pas sélectionner automatiquement un résultat si plusieurs sont disponibles
        // Laisser l'utilisateur choisir parmi tous les résultats
        // Seulement sélectionner automatiquement si un seul résultat exact est trouvé
        const currentResults = preserveExistingResults && metadataResults.length > 0 
          ? [...metadataResults, ...results] 
          : results
        
        // Si on a des infos du DVD (director/year), essayer de trouver une correspondance exacte
        // Le backend a déjà priorisé les résultats, donc le premier devrait être le meilleur
        if (dvdInfo?.director || dvdInfo?.year) {
          // Chercher une correspondance exacte (titre + réalisateur + année)
          const exactMatch = currentResults.find((r: any) => {
            const titleMatch = r.title?.toLowerCase().trim() === title.toLowerCase().trim() || 
                              title.toLowerCase().trim() === r.title?.toLowerCase().trim().split('(')[0].trim()
            const directorMatch = !dvdInfo.director || 
                                 r.director?.toLowerCase().includes(dvdInfo.director.toLowerCase()) || 
                                 dvdInfo.director.toLowerCase().includes(r.director?.toLowerCase() || '')
            const yearMatch = !dvdInfo.year || 
                             r.year === dvdInfo.year || 
                             r.production_year === dvdInfo.year ||
                             Math.abs((r.year || r.production_year || 0) - dvdInfo.year) <= 2
            return titleMatch && directorMatch && yearMatch
          })
          
          // Si on trouve une correspondance exacte, la sélectionner automatiquement
          if (exactMatch) {
            console.log('[DVD-Extract] ✅ Correspondance exacte trouvée, sélection automatique:', exactMatch.title)
            selectMetadata(exactMatch)
          } else if (currentResults.length > 0) {
            // Sinon, sélectionner le premier résultat (déjà priorisé par le backend)
            // Le backend a déjà trié les résultats : director+year en premier
            console.log('[DVD-Extract] 📊 Sélection automatique du premier résultat priorisé:', currentResults[0].title)
            selectMetadata(currentResults[0])
          }
        } else if (currentResults.length === 1) {
          // Si un seul résultat sans réalisateur/année, le sélectionner automatiquement
          console.log('[DVD-Extract] 📊 Un seul résultat trouvé, sélection automatique:', currentResults[0].title)
          selectMetadata(currentResults[0])
        } else if (currentResults.length > 0) {
          // Si plusieurs résultats sans réalisateur/année, sélectionner le premier (déjà priorisé)
          console.log('[DVD-Extract] 📊 Plusieurs résultats, sélection automatique du premier:', currentResults[0].title)
          selectMetadata(currentResults[0])
        }
      } else {
        setAnalyzingStage('Aucune métadonnée trouvée')
      }
    } catch (error) {
      console.error('Erreur lors de la recherche de métadonnées:', error)
      setAnalyzingStage('Erreur lors de la recherche de métadonnées')
    } finally {
      setSearchingMetadata(false)
    }
  }

  const selectMetadata = (metadata: any) => {
    setSelectedMetadata(metadata)
    setShowMetadataResults(false)

    if (metadata.title) {
      setTitle(decodeHtmlEntities(metadata.title))
    }

    if (metadata.synopsis || metadata.description) {
      setDescription(decodeHtmlEntities(metadata.synopsis || metadata.description || ''))
    }

    if (metadata.poster_url) {
      setThumbnailPreview(metadata.poster_url)
      // Réinitialiser thumbnailFile pour utiliser l'affiche depuis les métadonnées
      setThumbnailFile(null)
    }

    // Mapper les genres aux catégories si disponibles
    if (metadata.genres && metadata.genres.length > 0 && categories.length > 0) {
      // Mapping des genres IMDb vers les catégories de l'application
      const genreMapping: { [key: string]: string[] } = {
        'action': ['action', 'aventure', 'combat'],
        'adventure': ['aventure', 'action'],
        'comedy': ['comédie', 'humour'],
        'drama': ['drame', 'dramatique'],
        'horror': ['horreur', 'épouvante', 'thriller'],
        'thriller': ['thriller', 'suspense', 'horreur'],
        'sci-fi': ['science-fiction', 'sf', 'futuriste'],
        'science fiction': ['science-fiction', 'sf'],
        'fantasy': ['fantastique', 'fantasy'],
        'romance': ['romance', 'romantique', 'amour'],
        'crime': ['crime', 'policier', 'thriller'],
        'mystery': ['mystère', 'policier', 'thriller'],
        'animation': ['animation', 'dessin animé'],
        'family': ['famille', 'jeunesse'],
        'war': ['guerre', 'militaire'],
        'western': ['western', 'cowboy'],
        'documentary': ['documentaire'],
        'biography': ['biographie', 'biopic'],
        'history': ['historique', 'histoire'],
        'sport': ['sport', 'sportif'],
        'music': ['musical', 'musique'],
        'musical': ['musical', 'musique'],
      }

      // Chercher une catégorie correspondante pour chaque genre
      let categoryFound = false
      for (const genre of metadata.genres) {
        const genreLower = genre.toLowerCase().trim()
        const possibleNames = genreMapping[genreLower] || [genreLower]
        
        // Chercher une catégorie qui correspond à l'un des noms possibles
        const matchingCategory = categories.find((cat: any) => {
          if (!cat.name) return false
          const catNameLower = cat.name.toLowerCase().trim()
          // Vérifier correspondance exacte ou partielle
          return possibleNames.some(name => {
            const nameLower = name.toLowerCase().trim()
            return catNameLower === nameLower || 
                   catNameLower.includes(nameLower) || 
                   nameLower.includes(catNameLower)
          })
        })
        
        if (matchingCategory) {
          setCategoryId(matchingCategory.id)
          // Réinitialiser la sous-catégorie
          setSubCategoryId('')
          categoryFound = true
          console.log(`✅ Catégorie automatiquement sélectionnée: ${matchingCategory.name} (genre: ${genre})`)
          break // Utiliser la première correspondance trouvée
        }
      }
      
      if (!categoryFound) {
        console.log('⚠️ Aucune catégorie trouvée pour les genres:', metadata.genres)
      }
    }
  }

  const handleThumbnailChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setThumbnailFile(file)
      const reader = new FileReader()
      reader.onloadend = () => {
        setThumbnailPreview(reader.result as string)
      }
      reader.readAsDataURL(file)
    }
  }

  const handleCancelExtraction = async () => {
    if (!videoId) return

    if (!window.confirm('Êtes-vous sûr de vouloir annuler l\'extraction ?')) {
      return
    }

    setCancelling(true)
    try {
      await api.post(`/dvd/extract/${videoId}/cancel`)
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current)
        pollIntervalRef.current = null
      }
      setExtracting(false)
      setExtractionProgress(0)
      setExtractionStage('')
      setExtractionDetails('')
      setError('Extraction annulée par l\'utilisateur')
      setStep(3)
    } catch (error: any) {
      setError(error.response?.data?.error || 'Erreur lors de l\'annulation')
    } finally {
      setCancelling(false)
    }
  }

  const handleExtract = async () => {
    if (!title.trim()) {
      setError('Le titre est requis')
      return
    }
    if (selectedAudioTracks.length === 0) {
      const confirmNoAudio = window.confirm(
        '⚠️ Aucune piste audio sélectionnée. La vidéo sera extraite sans son. Voulez-vous continuer ?'
      )
      if (!confirmNoAudio) {
        return
      }
    }
    if (qualities.length === 0) {
      setError('Sélectionnez au moins une qualité')
      return
    }

    setLoading(true)
    setError('')
    setExtracting(true)
    setExtractionProgress(0)
    setExtractionStage('Initialisation de l\'extraction...')
    setExtractionDetails('Préparation du processus d\'extraction')
    setStep(4)

    try {
      const response = await api.post('/dvd/extract', {
        device_path: device,
        title,
        description,
        category_id: categoryId || null,
        sub_category_id: subCategoryId || null,
        is_public: isPublic,
        video_track_index: selectedVideoTrack,
        audio_track_indices: selectedAudioTracks,
        subtitle_track_indices: selectedSubtitleTracks,
        qualities,
        trailer_url: selectedMetadata?.trailer_url || null,
        platform: selectedMetadata?.platform || null
      })

      setVideoId(response.data.video_id)
      setExtractionProgress(5)
      setExtractionStage('Extraction du DVD en cours...')
      startPolling(response.data.video_id)
    } catch (error: any) {
      setError(error.response?.data?.error || 'Erreur lors du démarrage de l\'extraction')
      setExtracting(false)
      setStep(3)
      setExtractionProgress(0)
      setExtractionStage('')
      setExtractionDetails('')
    } finally {
      setLoading(false)
    }
  }

  if (authLoading || !user) {
    return (
      <div className="dvd-extract-page">
        <div className="loading">Vérification des permissions...</div>
      </div>
    )
  }

  return (
    <div className="dvd-extract-page">
      <div className="dvd-extract-container">
        <h1>📀 Extraction DVD</h1>
        <p className="subtitle">Extrayez vos DVD directement dans votre bibliothèque</p>

        {error && (
          <div className="error-message">
            <span>❌ {error}</span>
            <button onClick={() => setError('')}>×</button>
          </div>
        )}

        {step === 1 && (
          <div className="step-container">
            <div className="step-header">
              <h2>📀 Étape 1: Détection du lecteur DVD</h2>
              <p className="step-description">
                Insérez votre DVD dans le lecteur, puis cliquez sur "Détecter automatiquement"
              </p>
            </div>

            <div className="detection-section">
              <button
                onClick={handleDetectDVD}
                disabled={loading}
                className="btn-primary btn-large"
              >
                {loading ? '⏳ Détection en cours...' : '🔍 Détecter automatiquement'}
              </button>

              {availableDevices.length > 0 && (
                <div className="manual-device-selection">
                  <h3>📋 Lecteurs disponibles détectés :</h3>
                  <p className="device-hint">
                    Aucun DVD n'a été détecté automatiquement, mais des lecteurs sont disponibles. 
                    Sélectionnez-en un manuellement et assurez-vous qu'un DVD est inséré.
                  </p>
                  <div className="device-selector">
                    <select
                      value={manualDevice}
                      onChange={(e) => setManualDevice(e.target.value)}
                      className="device-select"
                    >
                      {availableDevices.map((dev) => (
                        <option key={dev} value={dev}>{dev}</option>
                      ))}
                    </select>
                    <button onClick={handleManualDeviceSelect} className="btn-secondary">
                      Utiliser ce lecteur
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="step-container">
            <div className="step-header">
              <h2>📖 Étape 2: Lecture des informations du DVD</h2>
              <p className="step-description">
                Cliquez sur le bouton ci-dessous pour analyser le contenu du DVD
              </p>
            </div>
            <div className="device-info">
              <p><strong>📀 Lecteur détecté:</strong> <code>{device}</code></p>
            </div>

            {loading && (analyzingProgress > 0 || analyzingStage) && (
              <div className="analyzing-progress-container">
                <div className="analyzing-stage">{analyzingStage || 'Analyse en cours...'}</div>
                <div className="analyzing-progress-bar">
                  <div
                    className="analyzing-progress-fill"
                    style={{ width: `${analyzingProgress}%` }}
                  ></div>
                </div>
                <div className="analyzing-percentage">{analyzingProgress}%</div>
              </div>
            )}

            {!loading && (
              <button
                onClick={handleGetDVDInfo}
                disabled={loading}
                className="btn-primary btn-large"
              >
                📖 Lire les informations du DVD
              </button>
            )}

            <button onClick={() => setStep(1)} className="btn-secondary">
              ← Retour
            </button>
          </div>
        )}

        {step === 3 && dvdInfo && (
          <div className="step-container">
            <div className="step-header">
              <h2>Étape 3: Configuration de l'extraction</h2>
            </div>

            <div className="config-section">
              <h3>📝 Informations de la vidéo</h3>
              <div className="form-group">
                <label>Titre *</label>
                <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
                  <input
                    type="text"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    placeholder="Titre de la vidéo"
                    className="form-input"
                    style={{ flex: 1 }}
                  />
                  <button
                    type="button"
                    onClick={() => searchMetadataFromTitle(title)}
                    disabled={!title.trim() || searchingMetadata}
                    className="btn-secondary"
                    style={{ whiteSpace: 'nowrap' }}
                  >
                    {searchingMetadata ? '🔍 Recherche...' : '🔍 Rechercher métadonnées'}
                  </button>
                </div>
              </div>
              
              {showMetadataResults && metadataResults.length > 0 && (
                <div className="metadata-results" style={{ marginBottom: '20px', marginTop: '10px' }}>
                  <h4 style={{ marginBottom: '10px', fontSize: '14px', color: 'rgba(255, 255, 255, 0.8)' }}>
                    Résultats de recherche:
                  </h4>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '15px' }}>
                    {metadataResults.map((result: any, idx: number) => (
                      <div
                        key={idx}
                        onClick={() => selectMetadata(result)}
                        style={{
                          cursor: 'pointer',
                          padding: '12px',
                          background: selectedMetadata?.imdb_id === result.imdb_id 
                            ? 'rgba(255, 0, 0, 0.2)' 
                            : 'rgba(255, 255, 255, 0.05)',
                          border: selectedMetadata?.imdb_id === result.imdb_id 
                            ? '2px solid #ff0000' 
                            : '1px solid rgba(255, 255, 255, 0.1)',
                          borderRadius: '8px',
                          transition: 'all 0.3s'
                        }}
                      >
                        {result.poster_url && (
                          <img
                            src={result.poster_url}
                            alt={result.title}
                            style={{ width: '100%', height: 'auto', borderRadius: '4px', marginBottom: '8px' }}
                          />
                        )}
                        <div style={{ fontSize: '12px', fontWeight: 'bold', marginBottom: '4px' }}>
                          {result.title}
                        </div>
                        {result.year && (
                          <div style={{ fontSize: '11px', color: 'rgba(255, 255, 255, 0.6)', marginBottom: '2px' }}>
                            📅 {result.year}
                          </div>
                        )}
                        {result.duration && (
                          <div style={{ fontSize: '11px', color: 'rgba(255, 255, 255, 0.6)', marginBottom: '2px' }}>
                            ⏱️ {result.duration}
                          </div>
                        )}
                        {result.platform && (
                          <div style={{ fontSize: '11px', color: 'rgba(255, 255, 255, 0.6)', marginBottom: '2px' }}>
                            📺 {result.platform}
                          </div>
                        )}
                        {result.trailer_url && (
                          <div style={{ fontSize: '11px', color: 'rgba(255, 255, 255, 0.6)', marginBottom: '2px' }}>
                            🎬 Bande-annonce disponible
                          </div>
                        )}
                        {result.rating && (
                          <div style={{ fontSize: '11px', color: 'rgba(255, 255, 255, 0.6)', marginBottom: '2px' }}>
                            ⭐ {result.rating}/10
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
              
              <div className="form-group">
                <label>Description</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Description de la vidéo"
                  className="form-textarea"
                  rows={3}
                />
              </div>
              
              <div className="form-group">
                <label>Affiche (optionnel)</label>
                {thumbnailPreview && (
                  <div style={{ marginBottom: '10px' }}>
                    <img
                      src={thumbnailPreview}
                      alt="Aperçu"
                      style={{ maxWidth: '200px', maxHeight: '300px', borderRadius: '8px', border: '1px solid rgba(255, 255, 255, 0.2)' }}
                    />
                    {selectedMetadata?.poster_url && (
                      <p style={{ fontSize: '12px', color: 'rgba(255, 255, 255, 0.6)', marginTop: '8px' }}>
                        Affiche sélectionnée depuis les métadonnées
                      </p>
                    )}
                  </div>
                )}
                {!selectedMetadata?.poster_url && (
                  <input
                    type="file"
                    accept="image/*"
                    onChange={handleThumbnailChange}
                    className="form-input"
                  />
                )}
                {selectedMetadata?.poster_url && (
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedMetadata(null)
                      setThumbnailPreview(null)
                      setThumbnailFile(null)
                    }}
                    className="btn-secondary"
                    style={{ marginTop: '10px' }}
                  >
                    Choisir un autre fichier
                  </button>
                )}
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Catégorie</label>
                  <select
                    value={categoryId}
                    onChange={(e) => {
                      setCategoryId(e.target.value)
                      setSubCategoryId('')
                    }}
                    className="form-select"
                  >
                    <option value="">Aucune</option>
                    {categories.map((cat: any) => (
                      <option key={cat.id} value={cat.id}>{cat.name}</option>
                    ))}
                  </select>
                </div>
                {subCategories.length > 0 && (
                  <div className="form-group">
                    <label>Sous-catégorie</label>
                    <select
                      value={subCategoryId}
                      onChange={(e) => setSubCategoryId(e.target.value)}
                      className="form-select"
                    >
                      <option value="">Aucune</option>
                      {subCategories.map((sub: any) => (
                        <option key={sub.id} value={sub.id}>{sub.name}</option>
                      ))}
                    </select>
                  </div>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>🎬 Piste vidéo</h3>
              <div className="tracks-list">
                {dvdInfo.video_tracks && dvdInfo.video_tracks.length > 0 ? (
                  dvdInfo.video_tracks.map((track: any, idx: number) => (
                    <label key={idx} className="track-item">
                      <input
                        type="radio"
                        name="videoTrack"
                        checked={selectedVideoTrack === idx}
                        onChange={() => setSelectedVideoTrack(idx)}
                      />
                      <span>Piste {idx + 1} - {track.codec || 'Inconnu'}</span>
                    </label>
                  ))
                ) : (
                  <p className="no-tracks">Aucune piste vidéo disponible</p>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>🔊 Pistes audio</h3>
              <div className="tracks-list">
                {dvdInfo.audio_tracks && dvdInfo.audio_tracks.length > 0 ? (
                  dvdInfo.audio_tracks.map((track: any, idx: number) => (
                    <label key={idx} className="track-item">
                      <input
                        type="checkbox"
                        checked={selectedAudioTracks.includes(idx)}
                        onChange={() => toggleAudioTrack(idx)}
                      />
                      <span>
                        {track.title || `Piste ${idx + 1} - ${track.language || 'Inconnu'}`}
                      </span>
                    </label>
                  ))
                ) : (
                  <p className="no-tracks">Aucune piste audio disponible</p>
                )}
              </div>
            </div>

            <div className="config-section">
              <h3>🎥 Qualités de transcodage</h3>
              <div className="qualities-grid">
                {['2160p', '1440p', '1080p', '720p', '480p', '360p'].map((quality) => {
                  const isAvailable = availableQualities.includes(quality)
                  return (
                    <label 
                      key={quality} 
                      className={`quality-item ${!isAvailable ? 'disabled' : ''}`}
                      style={!isAvailable ? { opacity: 0.5, cursor: 'not-allowed' } : {}}
                      title={!isAvailable ? 'Cette qualité n\'est pas disponible dans les paramètres' : ''}
                    >
                      <input
                        type="checkbox"
                        checked={qualities.includes(quality)}
                        onChange={() => toggleQuality(quality)}
                        disabled={!isAvailable}
                      />
                      <span>{quality}</span>
                    </label>
                  )
                })}
              </div>
            </div>

            {selectedMetadata && (
              <div className="config-section" style={{ marginTop: '20px', padding: '15px', background: 'rgba(255, 0, 0, 0.1)', border: '1px solid rgba(255, 0, 0, 0.3)', borderRadius: '8px' }}>
                <h3 style={{ marginBottom: '10px', fontSize: '14px', color: 'rgba(255, 255, 255, 0.9)' }}>
                  📋 Métadonnées sélectionnées qui seront ajoutées à la vidéo:
                </h3>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '10px', fontSize: '12px' }}>
                  {selectedMetadata.poster_url && (
                    <div style={{ color: 'rgba(255, 255, 255, 0.8)' }}>
                      🖼️ <strong>Affiche:</strong> Oui
                    </div>
                  )}
                  {selectedMetadata.trailer_url && (
                    <div style={{ color: 'rgba(255, 255, 255, 0.8)' }}>
                      🎬 <strong>Bande-annonce:</strong> Oui
                    </div>
                  )}
                  {selectedMetadata.duration && (
                    <div style={{ color: 'rgba(255, 255, 255, 0.8)' }}>
                      ⏱️ <strong>Durée:</strong> {selectedMetadata.duration}
                    </div>
                  )}
                  {selectedMetadata.platform && (
                    <div style={{ color: 'rgba(255, 255, 255, 0.8)' }}>
                      📺 <strong>Plateforme:</strong> {selectedMetadata.platform}
                    </div>
                  )}
                  {selectedMetadata.rating && (
                    <div style={{ color: 'rgba(255, 255, 255, 0.8)' }}>
                      ⭐ <strong>Note:</strong> {selectedMetadata.rating}/10
                    </div>
                  )}
                  {selectedMetadata.year && (
                    <div style={{ color: 'rgba(255, 255, 255, 0.8)' }}>
                      📅 <strong>Année:</strong> {selectedMetadata.year}
                    </div>
                  )}
                </div>
              </div>
            )}

            <div className="actions">
              <button onClick={() => setStep(2)} className="btn-secondary">
                ← Retour
              </button>
              <button
                onClick={handleExtract}
                disabled={loading || !title.trim()}
                className="btn-primary btn-large"
              >
                🚀 Démarrer l'extraction
              </button>
            </div>
          </div>
        )}

        {step === 4 && extracting && (
          <div className="step-container">
            <div className="extraction-progress">
              <h2>⏳ {extractionStage || 'Extraction en cours...'}</h2>
              <div className="progress-bar-container">
                <div className="progress-bar" style={{ width: `${extractionProgress}%` }}></div>
              </div>
              <p className="progress-text">{Math.round(extractionProgress)}%</p>
              {extractionDetails && (
                <p className="progress-info">{extractionDetails}</p>
              )}
              <div className="extraction-actions">
                <button
                  onClick={handleCancelExtraction}
                  disabled={cancelling}
                  className="btn-secondary btn-cancel"
                >
                  {cancelling ? '⏳ Annulation...' : '❌ Annuler l\'extraction'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

