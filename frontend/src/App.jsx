import React from 'react'
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'
import Navigation from './components/Navigation'
import Home from './pages/Home'
import Search from './pages/Search'
import Login from './pages/auth/Login'
import Register from './pages/auth/Register'
import History from './pages/user/History'
import Profile from './pages/user/Profile'
import Favorites from './pages/user/Favorites'
import Playlists from './pages/user/Playlists'
import DVDExtract from './pages/video/DVDExtract'
import VideoPlayerPage from './pages/video/VideoPlayer'
import Upload from './pages/video/Upload'
import AdminDashboard from './pages/admin/AdminDashboard'
import AdminUsers from './pages/admin/AdminUsers'
import AdminTorrents from './pages/admin/AdminTorrents'
import AdminCategories from './pages/admin/AdminCategories'
import AdminTorrentConfigs from './pages/admin/AdminTorrentConfigs'
import AdminTorrentClients from './pages/admin/AdminTorrentClients'
import AdminMediaSettings from './pages/admin/AdminMediaSettings'
import AdminVideos from './pages/admin/AdminVideos'
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
              <Route path="/video/:id" element={<VideoPlayerPage />} />
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
              <Route path="/admin/videos" element={<AdminVideos />} />
              <Route path="/upload" element={<Upload />} />
            </Routes>
          </main>
        </div>
      </Router>
    </AuthProvider>
  )
}

export default App

