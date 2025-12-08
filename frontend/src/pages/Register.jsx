import React, { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import './Auth.css'

const Register = () => {
  const navigate = useNavigate()
  const { register } = useAuth()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess(false)

    if (password !== confirmPassword) {
      setError('Les mots de passe ne correspondent pas')
      return
    }

    if (password.length < 6) {
      setError('Le mot de passe doit contenir au moins 6 caractères')
      return
    }

    setLoading(true)
    const result = await register(username, email, password)
    setLoading(false)

    if (result.success) {
      setSuccess(true)
      // Afficher le message de succès pendant 5 secondes avant de rediriger
      setTimeout(() => {
        navigate('/login')
      }, 5000)
    } else {
      setError(result.error)
    }
  }

  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1>Inscription</h1>
        {success ? (
          <div className="auth-success">
            <h2>✅ Demande d'inscription envoyée !</h2>
            <p>Votre compte sera activé après validation par un administrateur.</p>
            <p>Vous serez redirigé vers la page de connexion dans quelques secondes...</p>
            <Link to="/login" className="auth-button">Aller à la connexion</Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            {error && <div className="auth-error">{error}</div>}
            <div className="info-box">
              <p>ℹ️ Votre compte nécessitera une validation par un administrateur avant d'être activé.</p>
            </div>
          <div className="form-group">
            <label htmlFor="username">Nom d'utilisateur</label>
            <input
              type="text"
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              disabled={loading}
            />
          </div>
          <div className="form-group">
            <label htmlFor="email">Email</label>
            <input
              type="email"
              id="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              disabled={loading}
            />
          </div>
          <div className="form-group">
            <label htmlFor="password">Mot de passe</label>
            <input
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              disabled={loading}
            />
          </div>
          <div className="form-group">
            <label htmlFor="confirmPassword">Confirmer le mot de passe</label>
            <input
              type="password"
              id="confirmPassword"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              disabled={loading}
            />
          </div>
            <button type="submit" disabled={loading} className="auth-button">
              {loading ? 'Inscription...' : "S'inscrire"}
            </button>
          </form>
        )}
        {!success && (
          <p className="auth-link">
            Déjà un compte ? <Link to="/login">Se connecter</Link>
          </p>
        )}
      </div>
    </div>
  )
}

export default Register

