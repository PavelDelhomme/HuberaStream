import Link from 'next/link'
import './NotFound.css'

export default function NotFound() {
  return (
    <div className="not-found-page">
      <div className="not-found-container">
        <div className="not-found-icon">🔍</div>
        <h1 className="not-found-title">404</h1>
        <h2 className="not-found-subtitle">Page non trouvée</h2>
        <p className="not-found-message">
          La page que vous recherchez n'existe pas ou a été déplacée.
        </p>
        <div className="not-found-actions">
          <Link href="/" className="btn-home">
            🏠 Retour à l'accueil
          </Link>
          <Link href="/search" className="btn-search">
            🔍 Rechercher
          </Link>
        </div>
      </div>
    </div>
  )
}

