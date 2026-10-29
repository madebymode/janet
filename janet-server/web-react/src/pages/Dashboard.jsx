import React, { useState, useEffect } from 'react'
import { useParams, useLocation, useNavigate } from 'react-router-dom'
import { useApi } from '../hooks/useApi'
import OverviewPage from './OverviewPage'
import LeaderboardPage from './LeaderboardPage'
import AnalyticsPage from './AnalyticsPage'
import ActivityPage from './ActivityPage'
import UsersPage from './UsersPage'
import PopularMessagesPage from './PopularMessagesPage'

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
    if (path.includes('/popular')) return 'popular'
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
  const [selectedMonth, setSelectedMonth] = useState(() => {
    const params = new URLSearchParams(window.location.search)
    const m = parseInt(params.get('month') || '0', 10)
    return m > 0 && m <= 12 ? m : 0
  })
  const [selectedUser, setSelectedUser] = useState(urlUsername || '')

  // Sync state → URL
useEffect(() => {
  const base = activeTab === 'overview' ? '/overview' : `/${activeTab}`;
  const yrSeg = selectedYear === 0 ? 'all' : selectedYear;
  const path = activeTab === 'users' && selectedUser
                 ? `${base}/${yrSeg}/${selectedUser}`
                 : `${base}/${yrSeg}`;
  // Preserve popular page's own search params; for others encode month if set
  let searchSuffix = '';
  if (activeTab === 'popular') {
    searchSuffix = location.search;
  } else if (selectedMonth > 0) {
    searchSuffix = `?month=${selectedMonth}`;
  }
  const fullPath = `${path}${searchSuffix}`;
  if (location.pathname + location.search !== fullPath) {
    navigate(fullPath, { replace: true });
  }
}, [activeTab, selectedYear, selectedMonth, selectedUser, navigate, location.pathname, location.search]);

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

  // Restore month from search params (only for non-popular tabs)
  if (activeTab !== 'popular') {
    const params = new URLSearchParams(location.search);
    const monthParam = parseInt(params.get('month') || '0', 10);
    if (monthParam !== selectedMonth) setSelectedMonth(monthParam > 0 && monthParam <= 12 ? monthParam : 0);
  }

  if (location.pathname.includes('/users')) {
    if (urlUsername !== selectedUser) setSelectedUser(urlUsername || '');
  }

  const newTab = getCurrentTab();
  if (newTab !== activeTab) setActiveTab(newTab);
}, [urlYear, urlUsername, location.pathname, location.search]);

  const { data: availableYears, loading: yearsLoading, error: yearsError } = useApi('/api/stats/years');

  const NOW = new Date()
  const CURRENT_YEAR = NOW.getFullYear()
  const CURRENT_MONTH = NOW.getMonth() + 1

  const monthSuffix = selectedMonth > 0 ? `&month=${selectedMonth}` : ''

  const { data: leaderboardData, loading: leaderboardLoading, error: leaderboardError } = useApi(
    activeTab === 'leaderboard' && selectedYear !== null
      ? (selectedYear === 0
        ? '/api/leaderboard?limit=100'
        : `/api/leaderboard/${selectedYear}?limit=100${monthSuffix}`)
      : null,
    [activeTab, selectedYear, selectedMonth]
  );

  const { data: topGiversData, loading: topGiversLoading, error: topGiversError } = useApi(
    activeTab === 'leaderboard' && selectedYear !== null
      ? (selectedYear === 0
        ? '/api/stats/top-givers?limit=10'
        : `/api/stats/top-givers/${selectedYear}?limit=10${monthSuffix}`)
      : null,
    [activeTab, selectedYear, selectedMonth]
  );

  const handleTabChange = (tab) => {
    setActiveTab(tab)
    setSelectedMonth(0)
    if (tab !== 'users') {
      setSelectedUser('')
    }
  }

  const handleYearChange = (value) => {
    const newYear = value === 'all' ? 0 : parseInt(value, 10)
    setSelectedYear(newYear)
    setSelectedMonth(0)
  }

  const handleMonthChange = (value) => {
    setSelectedMonth(value === 'all' ? 0 : parseInt(value, 10))
  }

  const handleUserChange = (username) => {
    setSelectedUser(username)
    if (username && activeTab !== 'users' && activeTab !== 'popular') {
      setActiveTab('users')
    }
  }

  const renderContent = () => {
    const commonProps = {
      selectedYear,
      selectedMonth,
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
      case 'popular':
        return <PopularMessagesPage {...commonProps} />
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
            
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '0.5rem' }}>
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '0.75rem',
                background: 'rgba(255,255,255,0.1)',
                padding: '0.5rem 1rem',
                borderRadius: '25px',
                backdropFilter: 'blur(10px)',
                flexWrap: 'wrap'
              }}>
                <span style={{ fontSize: '0.9rem', fontWeight: '500' }}>📅 Viewing:</span>
                <select
                  value={selectedYear === 0 ? 'all' : selectedYear}
                  onChange={(e) => handleYearChange(e.target.value)}
                  style={{
                    background: selectedYear !== CURRENT_YEAR ? 'rgba(255,230,80,0.35)' : 'rgba(255,255,255,0.2)',
                    border: selectedYear !== CURRENT_YEAR ? '1.5px solid rgba(255,230,80,0.8)' : '1px solid rgba(255,255,255,0.3)',
                    borderRadius: '15px',
                    padding: '0.4rem 0.8rem',
                    color: 'white',
                    fontSize: '0.9rem',
                    fontWeight: selectedYear !== CURRENT_YEAR ? '700' : '500',
                    backdropFilter: 'blur(5px)',
                    cursor: leaderboardLoading ? 'wait' : 'pointer'
                  }}
                >
                  <option value={CURRENT_YEAR} style={{ background: '#333', color: 'white' }}>This Year</option>
                  <option value="all" style={{ background: '#333', color: 'white' }}>All Time</option>
                  {yearsLoading ? (
                    <option disabled style={{ background: '#333', color: 'white' }}>Loading years...</option>
                  ) : yearsError ? (
                    <option disabled style={{ background: '#333', color: 'white' }}>Error loading years</option>
                  ) : (
                    availableYears?.filter(y => y !== CURRENT_YEAR).map(year => (
                      <option key={year} value={year} style={{ background: '#333', color: 'white' }}>
                        {year}
                      </option>
                    ))
                  )}
                </select>
                {selectedYear > 0 && (
                  <select
                    value={selectedMonth === 0 ? 'all' : selectedMonth}
                    onChange={(e) => handleMonthChange(e.target.value)}
                    style={{
                      background: selectedMonth > 0 ? 'rgba(255,230,80,0.35)' : 'rgba(255,255,255,0.2)',
                      border: selectedMonth > 0 ? '1.5px solid rgba(255,230,80,0.8)' : '1px solid rgba(255,255,255,0.3)',
                      borderRadius: '15px',
                      padding: '0.4rem 0.8rem',
                      color: 'white',
                      fontSize: '0.9rem',
                      fontWeight: selectedMonth > 0 ? '700' : '500',
                      backdropFilter: 'blur(5px)',
                      cursor: 'pointer'
                    }}
                  >
                    <option value="all" style={{ background: '#333', color: 'white' }}>All Months</option>
                    {['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'].map((m, i) => (
                      <option key={i + 1} value={i + 1} style={{ background: '#333', color: 'white' }}>{m}</option>
                    ))}
                  </select>
                )}
              </div>

              {/* Active filter chips */}
              {(selectedYear !== CURRENT_YEAR || selectedMonth > 0) && (
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', flexWrap: 'wrap', justifyContent: 'flex-end' }}>
                  <span style={{ fontSize: '0.75rem', color: 'rgba(255,255,255,0.7)', fontWeight: '500' }}>Active filters:</span>
                  {selectedYear !== CURRENT_YEAR && (
                    <span style={{
                      background: 'rgba(255,230,80,0.25)',
                      border: '1px solid rgba(255,230,80,0.6)',
                      borderRadius: '12px',
                      padding: '0.2rem 0.6rem',
                      fontSize: '0.78rem',
                      fontWeight: '600',
                      color: 'rgba(255,255,200,1)',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.3rem'
                    }}>
                      {selectedYear === 0 ? 'All Time' : selectedYear}
                      <button
                        onClick={() => handleYearChange(CURRENT_YEAR)}
                        style={{ background: 'none', border: 'none', color: 'inherit', cursor: 'pointer', padding: 0, lineHeight: 1, fontSize: '0.9rem', opacity: 0.8 }}
                        title="Reset to this year"
                      >×</button>
                    </span>
                  )}
                  {selectedMonth > 0 && (
                    <span style={{
                      background: 'rgba(255,230,80,0.25)',
                      border: '1px solid rgba(255,230,80,0.6)',
                      borderRadius: '12px',
                      padding: '0.2rem 0.6rem',
                      fontSize: '0.78rem',
                      fontWeight: '600',
                      color: 'rgba(255,255,200,1)',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.3rem'
                    }}>
                      {['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'][selectedMonth - 1]}
                      <button
                        onClick={() => handleMonthChange('all')}
                        style={{ background: 'none', border: 'none', color: 'inherit', cursor: 'pointer', padding: 0, lineHeight: 1, fontSize: '0.9rem', opacity: 0.8 }}
                        title="Clear month filter"
                      >×</button>
                    </span>
                  )}
                </div>
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
                { id: 'popular', label: 'Popular', icon: '🔥' },
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
