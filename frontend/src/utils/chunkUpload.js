// Utilitaires pour l'upload par chunks avec upload parallèle

const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB par chunk (par défaut)
const MAX_PARALLEL_UPLOADS = 4 // Nombre maximum de chunks uploadés en parallèle (par défaut)

/**
 * Calcule la taille optimale des chunks en fonction de la taille du fichier
 * @param {number} fileSize - Taille du fichier en octets
 * @returns {number} Taille optimale des chunks en octets
 */
export function calculateOptimalChunkSize(fileSize) {
  const fileSizeMB = fileSize / (1024 * 1024)
  
  // Petits fichiers (< 100MB) : chunks de 2MB pour plus de granularité
  if (fileSizeMB < 100) {
    return 2 * 1024 * 1024 // 2MB
  }
  
  // Fichiers moyens (100MB - 1GB) : chunks de 5MB (optimal)
  if (fileSizeMB < 1024) {
    return 5 * 1024 * 1024 // 5MB
  }
  
  // Gros fichiers (1GB - 5GB) : chunks de 10MB pour réduire le nombre de chunks
  if (fileSizeMB < 5120) {
    return 10 * 1024 * 1024 // 10MB
  }
  
  // Très gros fichiers (> 5GB) : chunks de 20MB pour optimiser l'upload
  return 20 * 1024 * 1024 // 20MB
}

/**
 * Calcule le nombre optimal de chunks uploadés en parallèle
 * @param {number} fileSize - Taille du fichier en octets
 * @param {number} chunkSize - Taille des chunks en octets
 * @returns {number} Nombre optimal de chunks en parallèle
 */
export function calculateOptimalParallelUploads(fileSize, chunkSize) {
  const fileSizeMB = fileSize / (1024 * 1024)
  const totalChunks = Math.ceil(fileSize / chunkSize)
  
  // Pour les petits fichiers, moins de parallélisme
  if (fileSizeMB < 100) {
    return Math.min(2, totalChunks)
  }
  
  // Pour les fichiers moyens, parallélisme standard
  if (fileSizeMB < 1024) {
    return Math.min(4, totalChunks)
  }
  
  // Pour les gros fichiers, plus de parallélisme pour accélérer
  if (fileSizeMB < 5120) {
    return Math.min(6, totalChunks)
  }
  
  // Pour les très gros fichiers, parallélisme maximum
  return Math.min(8, totalChunks)
}

/**
 * Calcule le hash SHA256 d'un Blob en utilisant l'API Web Crypto native
 */
export async function calculateChunkHash(chunk) {
  const arrayBuffer = await chunk.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

/**
 * Divise un fichier en chunks avec taille adaptative
 * @param {File} file - Le fichier à diviser
 * @param {number} chunkSize - Taille des chunks (optionnel, calculée automatiquement si non fourni)
 * @returns {Array} Tableau de chunks
 */
export function splitFileIntoChunks(file, chunkSize = null) {
  // Calculer la taille optimale des chunks si non fournie
  if (!chunkSize) {
    chunkSize = calculateOptimalChunkSize(file.size)
  }
  
  const chunks = []
  const totalChunks = Math.ceil(file.size / chunkSize)
  
  console.log(`📦 Division du fichier (${(file.size / (1024 * 1024)).toFixed(1)} MB) en ${totalChunks} chunks de ${(chunkSize / (1024 * 1024)).toFixed(1)} MB chacun`)
  
  for (let i = 0; i < totalChunks; i++) {
    const start = i * chunkSize
    const end = Math.min(start + chunkSize, file.size)
    chunks.push({
      index: i,
      blob: file.slice(start, end),
      size: end - start,
      hash: null, // Sera calculé plus tard
    })
  }
  
  return chunks
}

/**
 * Calcule les hash de tous les chunks en parallèle
 */
export async function calculateChunksHash(chunks) {
  const hashPromises = chunks.map(async (chunk) => {
    const hash = await calculateChunkHash(chunk.blob)
    return { ...chunk, hash }
  })
  
  return Promise.all(hashPromises)
}

/**
 * Upload un chunk vers le serveur
 */
async function uploadChunk(sessionId, chunk, api) {
  const formData = new FormData()
  formData.append('chunk', chunk.blob)
  formData.append('chunk_index', chunk.index.toString())
  formData.append('checksum', chunk.hash)
  
  try {
    const response = await api.post(`/admin/uploads/${sessionId}/chunk`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        // Progression individuelle du chunk
        const chunkProgress = (progressEvent.loaded / progressEvent.total) * 100
        if (chunk.onProgress) {
          chunk.onProgress(chunkProgress)
        }
      },
    })
    return { success: true, chunk, response: response.data }
  } catch (error) {
    return { success: false, chunk, error }
  }
}

/**
 * Upload tous les chunks en parallèle avec limite de concurrence adaptative
 * Supporte la pause et l'annulation
 * @param {string} sessionId - ID de la session d'upload
 * @param {Array} chunks - Tableau de chunks à uploader
 * @param {Object} api - Instance API axios
 * @param {Function} onProgress - Callback de progression
 * @param {Object} control - Objet de contrôle (pause/cancel)
 * @param {number} maxParallel - Nombre maximum de chunks en parallèle (optionnel, calculé automatiquement)
 */
export async function uploadChunksParallel(sessionId, chunks, api, onProgress, control = {}, maxParallel = null) {
  const results = []
  const uploadedChunks = new Set()
  let uploadedBytes = 0
  const totalBytes = chunks.reduce((sum, chunk) => sum + chunk.size, 0)
  
  // Calculer le nombre optimal de chunks en parallèle si non fourni
  if (maxParallel === null) {
    maxParallel = calculateOptimalParallelUploads(totalBytes, chunks[0]?.size || CHUNK_SIZE)
  }
  
  console.log(`🚀 Upload avec ${maxParallel} chunks en parallèle (${chunks.length} chunks au total)`)
  
  // Fonction pour uploader un chunk
  const uploadNext = async () => {
    // Vérifier si l'upload est annulé
    if (control.cancelled) {
      throw new Error('Upload annulé par l\'utilisateur')
    }
    
    // Attendre si l'upload est en pause
    while (control.paused && !control.cancelled) {
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    
    if (control.cancelled) {
      throw new Error('Upload annulé par l\'utilisateur')
    }
    
    // Trouver le prochain chunk non uploadé
    const chunkIndex = chunks.findIndex(
      (chunk, idx) => !uploadedChunks.has(idx) && !chunks[idx].uploading
    )
    
    if (chunkIndex === -1) {
      return null
    }
    
    const chunk = chunks[chunkIndex]
    chunk.uploading = true
    
    try {
      const result = await uploadChunk(sessionId, chunk, api)
      
      if (result.success) {
        uploadedChunks.add(chunkIndex)
        uploadedBytes += chunk.size
        
        // Mettre à jour la progression globale
        const progress = (uploadedBytes / totalBytes) * 100
        if (onProgress) {
          onProgress({
            progress,
            uploadedBytes,
            totalBytes,
            uploadedChunks: uploadedChunks.size,
            totalChunks: chunks.length,
            currentChunk: chunkIndex + 1,
          })
        }
        
        return { success: true, chunkIndex, result: result.response }
      } else {
        // Retry automatique en cas d'erreur
        console.warn(`Erreur upload chunk ${chunkIndex}, retry...`, result.error)
        chunk.uploading = false
        await new Promise(resolve => setTimeout(resolve, 1000)) // Attendre 1s avant retry
        return uploadNext()
      }
    } catch (error) {
      chunk.uploading = false
      if (error.message === 'Upload annulé par l\'utilisateur') {
        throw error
      }
      return { success: false, chunkIndex, error }
    }
  }
  
  // Lancer les uploads en parallèle (limité à maxParallel)
  const uploadPromises = []
  for (let i = 0; i < Math.min(maxParallel, chunks.length); i++) {
    uploadPromises.push(
      (async () => {
        const results = []
        while (uploadedChunks.size < chunks.length && !control.cancelled) {
          try {
            const result = await uploadNext()
            if (result) {
              results.push(result)
            } else {
              break
            }
          } catch (error) {
            if (error.message === 'Upload annulé par l\'utilisateur') {
              throw error
            }
            break
          }
        }
        return results
      })()
    )
  }
  
  try {
    const allResults = await Promise.all(uploadPromises)
    return allResults.flat()
  } catch (error) {
    if (error.message === 'Upload annulé par l\'utilisateur') {
      throw error
    }
    return []
  }
}

