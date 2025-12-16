import React from 'react'
import './LoadingSpinner.css'

const LoadingSpinner = ({ message = 'Chargement...', fullScreen = false, size = 'medium' }) => {
  return (
    <div className={`loading-wrapper ${fullScreen ? 'full-screen' : ''} size-${size}`}>
      <div className="loading-container">
        <div className="loading-spinner">
          <div className="spinner-ring"></div>
          <div className="spinner-ring"></div>
          <div className="spinner-ring"></div>
          <div className="spinner-center"></div>
        </div>
        {message && <p className="loading-message">{message}</p>}
      </div>
    </div>
  )
}

export default LoadingSpinner

