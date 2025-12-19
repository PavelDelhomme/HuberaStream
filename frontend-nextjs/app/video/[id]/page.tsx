import { Suspense } from 'react'
import VideoPlayerContent from './VideoPlayerContent'
import type { Metadata } from 'next'

// SSR pour chaque requête (données dynamiques)
export const dynamic = 'force-dynamic'
export const revalidate = 0

// Fonction pour générer les métadonnées dynamiquement
export async function generateMetadata({ params }: { params: { id: string } }): Promise<Metadata> {
  try {
    const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3300/api'
    const response = await fetch(`${API_URL}/videos/${params.id}`, {
      next: { revalidate: 60 } // Cache 60 secondes
    })
    
    if (response.ok) {
      const video = await response.json()
      return {
        title: `${video.title} - StreamMake`,
        description: video.description || `Regardez ${video.title} sur StreamMake`,
        openGraph: {
          title: video.title,
          description: video.description,
          images: video.thumbnail_path ? [`${API_URL}/thumbnails/${video.thumbnail_path.split('/').pop()}`] : [],
        },
      }
    }
  } catch (error) {
    console.error('Erreur lors de la récupération des métadonnées:', error)
  }

  return {
    title: 'Vidéo - StreamMake',
    description: 'Regardez des vidéos sur StreamMake',
  }
}

export default function VideoPlayerPage({ params }: { params: { id: string } }) {
  return (
    <Suspense fallback={<div className="video-player-page"><div className="loading">Chargement...</div></div>}>
      <VideoPlayerContent videoId={params.id} />
    </Suspense>
  )
}
