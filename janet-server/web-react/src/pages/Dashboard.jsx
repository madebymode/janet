import React, { useState, useEffect } from 'react'
import { useParams, useLocation, useNavigate } from 'react-router-dom'
import { useApi } from '../hooks/useApi'
import OverviewPage from './OverviewPage'
import LeaderboardPage from './LeaderboardPage'
import AnalyticsPage from './AnalyticsPage'
import ActivityPage from './ActivityPage'
import UsersPage from './UsersPage'

function Dashboard() {
  const navigate = useNavigate()
  const location = useLocation()
  const { year: urlYear, username: urlUsername } = useParams()
  
  // Determine current tab from URL path
  const getCurrentTab = () => {
    const path = location.pathname
    if (path.includes('/leaderboard')) return 'leaderboard'
    if (path.includes('/analytics')) return 'analytics'
    if (path.includes('/activity')) return 'activity'
    if (path.includes('/users')) return 'users'
    return 'overview'
  }

  const [activeTab, setActiveTab] = useState(getCurrentTab())
  const [selectedYear, setSelectedYear] = useState(() => {
    if (urlYear) {
      if (urlYear === 'all') return 0
      const yearNum = parseInt(urlYear)
      return isNaN(yearNum) ? new Date().getFullYear() : yearNum
    }
    return new Date().getFullYear()
  })
  const [selectedUser, setSelectedUser] = useState(urlUsername || '')

  // Sync state → URL
useEffect(() => {
  const base = activeTab === 'overview' ? '/overview' : `/${activeTab}`;
  const yrSeg = selectedYear === 0 ? 'all' : selectedYear;
  const path = activeTab === 'users' && selectedUser
                 ? `${base}/${yrSeg}/${selectedUser}`
                 : `${base}/${yrSeg}`;
  if (location.pathname !== path) navigate(path, { replace: true });
}, [activeTab, selectedYear, selectedUser, navigate, location.pathname]);

// Sync URL → state (handles browser navigation)
useEffect(() => {
  let yearVal;
  if (urlYear === 'all') yearVal = 0;
  else if (!urlYear) yearVal = new Date().getFullYear();
  else {
    const parsed = parseInt(urlYear, 10);
    yearVal = isNaN(parsed) ? new Date().getFullYear() : parsed;
  }
  if (yearVal !== selectedYear) setSelectedYear(yearVal);
  if (urlUsername !== selectedUser) setSelectedUser(urlUsername || '');

  const newTab = getCurrentTab();
  if (newTab !== activeTab) setActiveTab(newTab);
}, [urlYear, urlUsername, location.pathname]);

  const { data: availableYears, loading: yearsLoading, error: yearsError } = useApi('/api/stats/years');

  const { data: leaderboardData, loading: leaderboardLoading, error: leaderboardError } = useApi(
    activeTab === 'leaderboard' && selectedYear !== null
      ? (selectedYear === 0
        ? '/api/leaderboard?limit=100'
        : `/api/leaderboard/${selectedYear}?limit=100`)
      : null,
    [activeTab, selectedYear]
  );

  const { data: topGiversData, loading: topGiversLoading, error: topGiversError } = useApi(
    activeTab === 'leaderboard' && selectedYear !== null
      ? (selectedYear === 0
        ? '/api/stats/top-givers?limit=10'
        : `/api/stats/top-givers/${selectedYear}?limit=10`)
      : null,
    [activeTab, selectedYear]
  );

  const handleTabChange = (tab) => {
    setActiveTab(tab)
    if (tab !== 'users') {
      setSelectedUser('')
    }
  }

  const handleYearChange = (year) => {
    const newYear = year === 'all' ? 0 : parseInt(year, 10)
    setSelectedYear(newYear)
  }

  const handleUserChange = (username) => {
    setSelectedUser(username)
    if (username && activeTab !== 'users') {
      setActiveTab('users')
    }
  }

  const renderContent = () => {
    const commonProps = {
      selectedYear,
      selectedUser,
      onYearChange: handleYearChange,
      onUserChange: handleUserChange
    }

    switch (activeTab) {
      case 'leaderboard':
        return <LeaderboardPage {...commonProps} leaderboardData={leaderboardData} leaderboardLoading={leaderboardLoading} leaderboardError={leaderboardError} topGiversData={topGiversData} topGiversLoading={topGiversLoading} topGiversError={topGiversError} />
      case 'analytics':
        return <AnalyticsPage {...commonProps} />
      case 'activity':
        return <ActivityPage {...commonProps} />
      case 'users':
        return <UsersPage {...commonProps} />
      default:
        return <OverviewPage {...commonProps} />
    }
  }

  return (
    <div className="dashboard">
      {/* Global Header with Navigation and Year Filter */}
      <div className="dashboard-header" style={{
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        color: 'white',
        padding: '1.5rem 0',
        marginBottom: '2rem',
        boxShadow: '0 4px 20px rgba(0,0,0,0.1)'
      }}>
        <div className="container" style={{ maxWidth: '1200px', margin: '0 auto', padding: '0 1rem' }}>
          {/* Year Filter Bar */}
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '1.5rem',
            flexWrap: 'wrap',
            gap: '1rem'
          }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: '1rem'
            }}>
              <h1 style={{ margin: 0, fontSize: '1.8rem', fontWeight: '700' }}>
                🏆 Karma Dashboard
              </h1>
            </div>
            
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: '1rem',
              background: 'rgba(255,255,255,0.1)',
              padding: '0.5rem 1rem',
              borderRadius: '25px',
              backdropFilter: 'blur(10px)'
            }}>
              <span style={{ fontSize: '0.9rem', fontWeight: '500' }}>📅 Viewing:</span>
              <select
                value={selectedYear === 0 ? 'all' : selectedYear}
                onChange={(e) => handleYearChange(e.target.value)}
                disabled={leaderboardLoading}
                style={{
                  background: 'rgba(255,255,255,0.2)',
                  border: '1px solid rgba(255,255,255,0.3)',
                  borderRadius: '15px',
                  padding: '0.4rem 0.8rem',
                  color: 'white',
                  fontSize: '0.9rem',
                  fontWeight: '500',
                  backdropFilter: 'blur(5px)',
                  cursor: leaderboardLoading ? 'wait' : 'pointer'
                }}
              >
                <option value="all" style={{ background: '#333', color: 'white' }}>All Time</option>
                {yearsLoading ? (
                  <option disabled style={{ background: '#333', color: 'white' }}>Loading years...</option>
                ) : yearsError ? (
                  <option disabled style={{ background: '#333', color: 'white' }}>Error loading years</option>
                ) : (
                  availableYears?.map(year => (
                    <option key={year} value={year} style={{ background: '#333', color: 'white' }}>
                      {year}
                    </option>
                  ))
                )}
              </select>
              {selectedYear === 0 && (
                <span style={{ fontSize: '0.8rem', opacity: 0.8 }}>📊 Complete History</span>
              )}
              {selectedYear > 0 && selectedYear === new Date().getFullYear() && (
                <span style={{ fontSize: '0.8rem', opacity: 0.8 }}>🕐 Current Year</span>
              )}
              {selectedYear > 0 && selectedYear !== new Date().getFullYear() && (
                <span style={{ fontSize: '0.8rem', opacity: 0.8 }}>📅 Historical Data</span>
              )}
            </div>
          </div>

          {/* Navigation Tabs */}
          <nav>
            <ul style={{
              display: 'flex',
              gap: '0.5rem',
              listStyle: 'none',
              margin: 0,
              padding: 0,
              flexWrap: 'wrap'
            }}>
              {[
                { id: 'overview', label: 'Overview', icon: '📊' },
                { id: 'leaderboard', label: 'Leaderboard', icon: '🏆' },
                { id: 'analytics', label: 'Analytics', icon: '📈' },
                { id: 'activity', label: 'Activity', icon: '🔄' },
                { id: 'users', label: 'Users', icon: '👥' }
              ].map(tab => (
                <li key={tab.id}>
                  <button
                    onClick={() => handleTabChange(tab.id)}
                    style={{
                      background: activeTab === tab.id ? 'rgba(255,255,255,0.25)' : 'rgba(255,255,255,0.1)',
                      border: activeTab === tab.id ? '1px solid rgba(255,255,255,0.4)' : '1px solid transparent',
                      borderRadius: '20px',
                      padding: '0.6rem 1.2rem',
                      color: 'white',
                      fontSize: '0.9rem',
                      fontWeight: activeTab === tab.id ? '600' : '500',
                      cursor: 'pointer',
                      transition: 'all 0.3s ease',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.5rem',
                      backdropFilter: 'blur(5px)',
                      transform: activeTab === tab.id ? 'scale(1.05)' : 'scale(1)',
                      boxShadow: activeTab === tab.id ? '0 4px 15px rgba(0,0,0,0.2)' : 'none'
                    }}
                    onMouseEnter={(e) => {
                      if (activeTab !== tab.id) {
                        e.target.style.background = 'rgba(255,255,255,0.15)'
                        e.target.style.transform = 'scale(1.02)'
                      }
                    }}
                    onMouseLeave={(e) => {
                      if (activeTab !== tab.id) {
                        e.target.style.background = 'rgba(255,255,255,0.1)'
                        e.target.style.transform = 'scale(1)'
                      }
                    }}
                  >
                    <span style={{ fontSize: '1.1rem' }}>{tab.icon}</span>
                    {tab.label}
                  </button>
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </div>

      {/* Page Content */}
      <div className="dashboard-content" style={{
        maxWidth: '1200px',
        margin: '0 auto',
        padding: '0 1rem'
      }}>
        {renderContent()}
      </div>
    </div>
  )
}

export default Dashboard
