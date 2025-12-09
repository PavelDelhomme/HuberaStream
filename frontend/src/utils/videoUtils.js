/**
 * Utilitaires pour la gestion des vidéos
 */

/**
 * Filtre les vidéos invalides (null, sans ID, titre "null", etc.)
 * @param {Array} videos - Tableau de vidéos à filtrer
 * @returns {Array} - Tableau de vidéos valides
 */
export const filterValidVideos = (videos) => {
  if (!videos || !Array.isArray(videos)) {
    return []
  }

  return videos.filter(video => {
    // Vérifier que la vidéo existe
    if (!video) {
      return false
    }

    // Vérifier que la vidéo a un ID
    if (!video.id) {
      return false
    }

    // Vérifier que le titre existe et n'est pas "null" (string)
    // Accepter les titres même s'ils sont vides temporairement (en cours de traitement)
    if (video.title === 'null' || video.title === 'Null' || video.title === 'NULL') {
      return false
    }

    // Vérifier que le titre n'est pas seulement des espaces (mais accepter les titres vides si status est 'processing')
    if (video.title && typeof video.title === 'string' && video.title.trim() === '' && video.status !== 'processing') {
      return false
    }

    return true
  })
}

/**
 * Vérifie si une vidéo est valide
 * @param {Object} video - Vidéo à vérifier
 * @returns {boolean} - true si la vidéo est valide
 */
export const isValidVideo = (video) => {
  if (!video || !video.id) {
    return false
  }

  if (!video.title || video.title === 'null' || video.title === 'Null' || video.title === 'NULL') {
    return false
  }

  if (video.title.trim() === '') {
    return false
  }

  return true
}

