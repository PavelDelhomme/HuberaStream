import { Suspense } from 'react'
import CategoryContent from './CategoryContent'

export const revalidate = 60 // ISR: revalider toutes les 60 secondes

export default function CategoryPage() {
  return (
    <Suspense fallback={<div className="loading">Chargement de la catégorie...</div>}>
      <CategoryContent />
    </Suspense>
  )
}

