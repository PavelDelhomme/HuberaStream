'use client'

import { useEffect } from 'react'
import Link from 'next/link'
import './Error.css'

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error('Erreur Next.js:', error)
  }, [error])

  return (
    <div className="error-page">
      <div className="error-container">
        <div className="error-icon">⚠️</div>
        <h1 className="error-title">Une erreur est survenue</h1>
        <p className="error-message">
          {error.message || 'Une erreur inattendue s\'est produite'}
        </p>
        {error.digest && (
          <p className="error-digest">
            Code d'erreur: {error.digest}
          </p>
        )}
        <div className="error-actions">
          <button onClick={reset} className="btn-retry">
            🔄 Réessayer
          </button>
          <Link href="/" className="btn-home">
            🏠 Retour à l'accueil
          </Link>
        </div>
        <details className="error-details">
          <summary>Détails techniques</summary>
          <pre className="error-stack">{error.stack}</pre>
        </details>
      </div>
    </div>
  )
}

