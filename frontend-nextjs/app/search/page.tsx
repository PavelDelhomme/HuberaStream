import { Suspense } from 'react'
import SearchContent from './SearchContent'

// ISR avec revalidation toutes les 60 secondes
export const revalidate = 60

export default function SearchPage() {
  return (
    <Suspense fallback={<div className="search-page"><div className="loading">Chargement...</div></div>}>
      <SearchContent />
    </Suspense>
  )
}
