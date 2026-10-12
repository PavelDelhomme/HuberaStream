import React from 'react'
import { Link } from 'react-router-dom'

const AdminTorrents = () => (
  <div style={{ maxWidth: '40rem', margin: '12vh auto', padding: '0 1.25rem', lineHeight: 1.5 }}>
    <h1>Hubera Stream</h1>
    <p>
      Cette adresse n’est <strong>pas</strong> un client torrent. Catalogue public légal : Pluto TV et Arte.tv.
      Sources personnelles owner (prod nginx) :{' '}
      <a href="https://stream.hubera.cloud/admin/sources/" rel="noopener noreferrer">
        /admin/sources/
      </a>
      .
    </p>
    <p>
      <Link to="/">Retour</Link>
    </p>
  </div>
)

export default AdminTorrents
