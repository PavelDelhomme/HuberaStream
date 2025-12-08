import React from 'react'
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Navigation from './components/Navigation'
import Home from './pages/Home'
import VideoPlayer from './pages/VideoPlayer'
import Search from './pages/Search'
import Playlists from './pages/Playlists'
import Login from './pages/Login'
import Register from './pages/Register'
import History from './pages/History'
import Profile from './pages/Profile'
import Favorites from './pages/Favorites'
import DVDExtract from './pages/DVDExtract'
import AdminDashboard from './pages/admin/AdminDashboard'
import AdminUsers from './pages/admin/AdminUsers'
import AdminTorrents from './pages/admin/AdminTorrents'
import AdminCategories from './pages/admin/AdminCategories'
import AdminTorrentConfigs from './pages/admin/AdminTorrentConfigs'
import AdminTorrentClients from './pages/admin/AdminTorrentClients'
import AdminMediaSettings from './pages/admin/AdminMediaSettings'
import Upload from './pages/Upload'
import { AuthProvider } from './context/AuthContext'
import './App.css'

function App() {
  return (
    <AuthProvider>
      <Router future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <div className="App">
          <Navigation />
          <main className="main-content">
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/video/:id" element={<VideoPlayer />} />
              <Route path="/search" element={<Search />} />
              <Route path="/playlists" element={<Playlists />} />
              <Route path="/history" element={<History />} />
              <Route path="/profile" element={<Profile />} />
              <Route path="/favorites" element={<Favorites />} />
              <Route path="/dvd-extract" element={<DVDExtract />} />
              <Route path="/login" element={<Login />} />
              <Route path="/register" element={<Register />} />
              <Route path="/admin" element={<AdminDashboard />} />
              <Route path="/admin/users" element={<AdminUsers />} />
              <Route path="/admin/torrents" element={<AdminTorrents />} />
              <Route path="/admin/categories" element={<AdminCategories />} />
              <Route path="/admin/torrent-configs" element={<AdminTorrentConfigs />} />
              <Route path="/admin/torrent-clients" element={<AdminTorrentClients />} />
              <Route path="/admin/media-settings" element={<AdminMediaSettings />} />
              <Route path="/upload" element={<Upload />} />
            </Routes>
          </main>
        </div>
      </Router>
    </AuthProvider>
  )
}

export default App

