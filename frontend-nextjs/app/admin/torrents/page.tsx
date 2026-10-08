'use client'

import Link from 'next/link'

export default function AdminTorrentsPage() {
  return (
    <div style={{ maxWidth: '40rem', margin: '12vh auto', padding: '0 1.25rem', fontFamily: 'system-ui, sans-serif', lineHeight: 1.5 }}>
      <h1>Hubera Stream</h1>
      <p>
        Cette adresse n’est <strong>pas</strong> un client torrent. Catalogue légal uniquement :
        Pluto TV et Arte.tv (APIs publiques, flux officiels). Pas de magnets, pas de miroir, pas de contournement DRM.
      </p>
      <p>
        <Link href="/">Retour</Link>
      </p>
    </div>
  )
}
