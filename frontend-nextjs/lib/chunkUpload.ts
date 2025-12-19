// Utilitaires pour l'upload par chunks avec upload parallèle

const CHUNK_SIZE = 5 * 1024 * 1024 // 5MB par chunk
const MAX_PARALLEL_UPLOADS = 4

export function calculateOptimalChunkSize(fileSize: number): number {
  const fileSizeMB = fileSize / (1024 * 1024)
  
  if (fileSizeMB < 100) {
    return 2 * 1024 * 1024 // 2MB
  }
  
  if (fileSizeMB < 1024) {
    return 5 * 1024 * 1024 // 5MB
  }
  
  if (fileSizeMB < 5120) {
    return 10 * 1024 * 1024 // 10MB
  }
  
  return 20 * 1024 * 1024 // 20MB
}

export function calculateOptimalParallelUploads(fileSize: number, chunkSize: number): number {
  const fileSizeMB = fileSize / (1024 * 1024)
  
  if (fileSizeMB < 100) {
    return 2
  }
  
  if (fileSizeMB < 1024) {
    return 4
  }
  
  if (fileSizeMB < 5120) {
    return 6
  }
  
  return 8
}

export async function calculateChunkHash(chunk: Blob): Promise<string> {
  const arrayBuffer = await chunk.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

export interface Chunk {
  index: number
  blob: Blob
  size: number
  hash: string | null
  uploading?: boolean
}

export function splitFileIntoChunks(file: File, chunkSize?: number): Chunk[] {
  if (!chunkSize) {
    chunkSize = calculateOptimalChunkSize(file.size)
  }
  
  const chunks: Chunk[] = []
  const totalChunks = Math.ceil(file.size / chunkSize)
  
  for (let i = 0; i < totalChunks; i++) {
    const start = i * chunkSize
    const end = Math.min(start + chunkSize, file.size)
    chunks.push({
      index: i,
      blob: file.slice(start, end),
      size: end - start,
      hash: null,
    })
  }
  
  return chunks
}

export async function calculateChunksHash(chunks: Chunk[]): Promise<Chunk[]> {
  const hashPromises = chunks.map(async (chunk) => {
    const hash = await calculateChunkHash(chunk.blob)
    return { ...chunk, hash }
  })
  
  return Promise.all(hashPromises)
}

interface UploadControl {
  paused?: boolean
  cancelled?: boolean
  cancelling?: boolean
}

interface ProgressData {
  progress: number
  uploadedBytes: number
  totalBytes: number
  uploadedChunks: number
  totalChunks: number
  currentChunk: number
}

async function uploadChunk(
  sessionId: string,
  chunk: Chunk,
  api: any
): Promise<{ success: boolean; chunk: Chunk; response?: any; error?: any }> {
  const formData = new FormData()
  formData.append('chunk', chunk.blob)
  formData.append('chunk_index', chunk.index.toString())
  formData.append('checksum', chunk.hash || '')
  
  try {
    const response = await api.post(`/admin/uploads/${sessionId}/chunk`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    return { success: true, chunk, response: response.data }
  } catch (error: any) {
    return { success: false, chunk, error }
  }
}

export async function uploadChunksParallel(
  sessionId: string,
  chunks: Chunk[],
  api: any,
  onProgress: (data: ProgressData) => void,
  control: UploadControl = {}
): Promise<void> {
  const uploadedChunks = new Set<number>()
  let uploadedBytes = 0
  const totalBytes = chunks.reduce((sum, chunk) => sum + chunk.size, 0)
  
  const maxParallel = calculateOptimalParallelUploads(totalBytes, chunks[0]?.size || CHUNK_SIZE)
  
  const uploadNext = async (): Promise<any> => {
    if (control.cancelled) {
      throw new Error('Upload annulé par l\'utilisateur')
    }
    
    while (control.paused && !control.cancelled) {
      await new Promise(resolve => setTimeout(resolve, 100))
    }
    
    if (control.cancelled) {
      throw new Error('Upload annulé par l\'utilisateur')
    }
    
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
        chunk.uploading = false
        await new Promise(resolve => setTimeout(resolve, 1000))
        return uploadNext()
      }
    } catch (error: any) {
      chunk.uploading = false
      if (error.message === 'Upload annulé par l\'utilisateur') {
        throw error
      }
      return { success: false, chunkIndex, error }
    }
  }
  
  const uploadPromises: Promise<any>[] = []
  for (let i = 0; i < Math.min(maxParallel, chunks.length); i++) {
    uploadPromises.push(uploadNext())
  }
  
  await Promise.all(uploadPromises)
  
  // Uploader les chunks restants
  while (uploadedChunks.size < chunks.length) {
    if (control.cancelled) {
      throw new Error('Upload annulé par l\'utilisateur')
    }
    
    const nextChunkIndex = chunks.findIndex(
      (chunk, idx) => !uploadedChunks.has(idx) && !chunks[idx].uploading
    )
    
    if (nextChunkIndex === -1) {
      break
    }
    
    await uploadNext()
  }
}

