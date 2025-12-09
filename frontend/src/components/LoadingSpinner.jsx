import React from 'react'
import './LoadingSpinner.css'

const LoadingSpinner = ({ message = 'Chargement...', fullScreen = false }) => {
  return (
    <div className={`loading-wrapper ${fullScreen ? 'full-screen' : ''}`}>
      <div className="loading-container">
        <div className="loading-spinner"></div>
        {message && <p className="loading-message">{message}</p>}
      </div>
    </div>
  )
}

export default LoadingSpinner

