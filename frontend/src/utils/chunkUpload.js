// Utilitaires pour l'upload par chunks avec upload parallèle

const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB par chunk
const MAX_PARALLEL_UPLOADS = 4 // Nombre maximum de chunks uploadés en parallèle

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
 * Divise un fichier en chunks
 */
export function splitFileIntoChunks(file, chunkSize = CHUNK_SIZE) {
  const chunks = []
  const totalChunks = Math.ceil(file.size / chunkSize)
  
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
 * Upload tous les chunks en parallèle avec limite de concurrence
 */
export async function uploadChunksParallel(sessionId, chunks, api, onProgress) {
  const results = []
  const uploadedChunks = new Set()
  let uploadedBytes = 0
  const totalBytes = chunks.reduce((sum, chunk) => sum + chunk.size, 0)
  
  // Fonction pour uploader un chunk
  const uploadNext = async () => {
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
      return { success: false, chunkIndex, error }
    }
  }
  
  // Lancer les uploads en parallèle (limité à MAX_PARALLEL_UPLOADS)
  const uploadPromises = []
  for (let i = 0; i < Math.min(MAX_PARALLEL_UPLOADS, chunks.length); i++) {
    uploadPromises.push(
      (async () => {
        const results = []
        while (uploadedChunks.size < chunks.length) {
          const result = await uploadNext()
          if (result) {
            results.push(result)
          } else {
            break
          }
        }
        return results
      })()
    )
  }
  
  const allResults = await Promise.all(uploadPromises)
  return allResults.flat()
}

