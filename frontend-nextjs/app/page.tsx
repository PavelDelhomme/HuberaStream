import { Suspense } from 'react'
import HomeContent from './HomeContent'

// ISR avec revalidation toutes les 60 secondes
export const revalidate = 60

export default function Home() {
  return (
    <Suspense fallback={<div className="home-container"><div className="loading">Chargement...</div></div>}>
      <HomeContent />
    </Suspense>
  )
}
