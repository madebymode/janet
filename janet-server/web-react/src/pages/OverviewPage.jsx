import React from 'react'
import { useApi } from '../hooks/useApi'
import StatCard from '../components/StatCard'
import PieChart from '../components/charts/PieChart'
import LineChart from '../components/charts/LineChart'

function OverviewPage({ selectedYear }) {
  const { data: stats, loading: statsLoading, error: statsError } = useApi(
    selectedYear === 0 ? '/api/stats' : `/api/stats?year=${selectedYear}`
  )
  const { data: status, loading: statusLoading } = useApi('/api/status')
  const { data: karmaDistribution, loading: karmaDistLoading } = useApi(
    selectedYear === 0 
      ? '/api/stats/karma-distribution' 
      : `/api/stats/karma-distribution?year=${selectedYear}`
  )
  const { data: pointsOverTime, loading: pointsTimeLoading } = useApi(
    selectedYear === 0 
      ? '/api/stats/points-over-time' 
      : `/api/stats/points-over-time?year=${selectedYear}`
  )

  const getYearLabel = () => {
    if (selectedYear === 0) return 'All Time'
    if (selectedYear === new Date().getFullYear()) return 'This Year'
    return selectedYear.toString()
  }

  return (
    <div className="overview-page">
      {/* Hero Section */}
      <div style={{
        background: 'linear-gradient(135deg, #f8f9ff 0%, #e8f0ff 100%)',
        borderRadius: '16px',
        padding: '2rem',
        marginBottom: '2rem',
        border: '1px solid #e1e8f7'
      }}>
        <div style={{ textAlign: 'center', marginBottom: '1.5rem' }}>
          <h1 style={{ 
            fontSize: '2.5rem', 
            margin: '0 0 0.5rem 0',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            fontWeight: '700'
          }}>
            Welcome to Karma Dashboard
          </h1>
          <p style={{ 
            fontSize: '1.1rem', 
            color: '#666', 
            margin: 0,
            maxWidth: '600px',
            marginLeft: 'auto',
            marginRight: 'auto'
          }}>
            Track team karma, celebrate achievements, and build a positive community culture.
            Currently viewing <strong>{getYearLabel()}</strong> data.
          </p>
        </div>
      </div>

      {/* Key Statistics */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
        gap: '1.5rem',
        marginBottom: '2rem'
      }}>
        <StatCard
          icon="👥"
          title="Community Members"
          value={stats?.totalUsers || 0}
          loading={statsLoading}
          error={statsError}
          subtitle={`Active ${getYearLabel().toLowerCase()}`}
          color="#3498db"
        />
        <StatCard
          icon="⭐"
          title="Karma Points"
          value={stats?.totalPoints || 0}
          loading={statsLoading}
          error={statsError}
          subtitle={`Earned ${getYearLabel().toLowerCase()}`}
          color="#e74c3c"
        />
        <StatCard
          icon="🔄"
          title="Interactions"
          value={stats?.totalTransactions || 0}
          loading={statsLoading}
          error={statsError}
          subtitle={`Total ${getYearLabel().toLowerCase()}`}
          color="#9b59b6"
        />
        <StatCard
          icon="🤖"
          title="Janet Bot"
          value={status?.botOnline ? 'Online' : 'Offline'}
          loading={statusLoading}
          isStatus={true}
          subtitle="Current status"
          color={status?.botOnline ? '#27ae60' : '#e74c3c'}
        />
      </div>

      {/* Charts Section */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
        gap: '2rem',
        marginBottom: '2rem'
      }}>
        <div className="card" style={{
          background: 'white',
          borderRadius: '12px',
          padding: '1.5rem',
          boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
          border: '1px solid #f0f0f0'
        }}>
          <h3 style={{
            margin: '0 0 1rem 0',
            fontSize: '1.2rem',
            fontWeight: '600',
            color: '#2c3e50',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <span>📊</span>
            Karma Distribution
          </h3>
          {karmaDistLoading ? (
            <div style={{ 
              height: '300px', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              color: '#999'
            }}>
              Loading chart...
            </div>
          ) : karmaDistribution ? (
            <PieChart data={karmaDistribution} />
          ) : (
            <div style={{ 
              height: '300px', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              color: '#999'
            }}>
              No data available
            </div>
          )}
        </div>

        <div className="card" style={{
          background: 'white',
          borderRadius: '12px',
          padding: '1.5rem',
          boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
          border: '1px solid #f0f0f0'
        }}>
          <h3 style={{
            margin: '0 0 1rem 0',
            fontSize: '1.2rem',
            fontWeight: '600',
            color: '#2c3e50',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <span>📈</span>
            Points Over Time
          </h3>
          {pointsTimeLoading ? (
            <div style={{ 
              height: '300px', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              color: '#999'
            }}>
              Loading chart...
            </div>
          ) : pointsOverTime ? (
            <LineChart 
              data={pointsOverTime} 
              xKey="month" 
              yKey="totalPoints"
            />
          ) : (
            <div style={{ 
              height: '300px', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              color: '#999'
            }}>
              No data available
            </div>
          )}
        </div>
      </div>

      {/* About Section */}
      <div className="card" style={{
        background: 'white',
        borderRadius: '12px',
        padding: '2rem',
        boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
        border: '1px solid #f0f0f0'
      }}>
        <h2 style={{
          margin: '0 0 1rem 0',
          fontSize: '1.5rem',
          fontWeight: '600',
          color: '#2c3e50'
        }}>
          About Janet Karma System
        </h2>
        <p style={{
          fontSize: '1rem',
          lineHeight: '1.6',
          color: '#666',
          marginBottom: '1.5rem'
        }}>
          Janet is a Slack bot that tracks karma points for team members, helping build a positive 
          and collaborative workplace culture. Team members can recognize each other's contributions 
          and achievements using simple commands.
        </p>
        
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
          gap: '1rem'
        }}>
          {[
            { command: '@username++', description: 'Give points', icon: '⬆️', color: '#27ae60' },
            { command: '@username--', description: 'Take points', icon: '⬇️', color: '#e74c3c' },
            { command: '@username?', description: 'Check points', icon: '❓', color: '#3498db' },
            { command: 'goodplace leaderboard', description: 'Show rankings', icon: '🏆', color: '#f39c12' }
          ].map((item, index) => (
            <div
              key={index}
              style={{
                padding: '1rem',
                background: `${item.color}10`,
                border: `1px solid ${item.color}30`,
                borderRadius: '8px',
                textAlign: 'center'
              }}
            >
              <div style={{ fontSize: '1.5rem', marginBottom: '0.5rem' }}>
                {item.icon}
              </div>
              <code style={{
                background: `${item.color}20`,
                padding: '0.25rem 0.5rem',
                borderRadius: '4px',
                fontSize: '0.875rem',
                fontWeight: '500',
                color: item.color,
                display: 'block',
                marginBottom: '0.5rem'
              }}>
                {item.command}
              </code>
              <div style={{
                fontSize: '0.875rem',
                color: '#666'
              }}>
                {item.description}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default OverviewPage
