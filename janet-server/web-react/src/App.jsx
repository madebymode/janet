import React from 'react'
import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'

function App() {
  return (
    <Layout>
      <Routes>
        {/* Main app routes with deep-linking support */}
        <Route path="/" element={<Dashboard />} />
        <Route path="/overview" element={<Dashboard />} />
        <Route path="/overview/:year" element={<Dashboard />} />
        <Route path="/leaderboard" element={<Dashboard />} />
        <Route path="/leaderboard/:year" element={<Dashboard />} />
        <Route path="/popular" element={<Dashboard />} />
        <Route path="/popular/:year" element={<Dashboard />} />
        <Route path="/analytics" element={<Dashboard />} />
        <Route path="/analytics/:year" element={<Dashboard />} />
        <Route path="/activity" element={<Dashboard />} />
        <Route path="/activity/:year" element={<Dashboard />} />
        <Route path="/users" element={<Dashboard />} />
        <Route path="/users/:year" element={<Dashboard />} />
        <Route path="/users/:year/:username" element={<Dashboard />} />
      </Routes>
    </Layout>
  )
}

export default App