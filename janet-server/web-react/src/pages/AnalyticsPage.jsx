import React from 'react'
import { useApi } from '../hooks/useApi'
import PieChart from '../components/charts/PieChart'
import LineChart from '../components/charts/LineChart'
import BarChart from '../components/charts/BarChart'

function AnalyticsPage({ selectedYear }) {
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
  const { data: topEmojis, loading: emojisLoading } = useApi(
    selectedYear === 0 
      ? '/api/stats/emojis?limit=10' 
      : `/api/stats/emojis?limit=10&year=${selectedYear}`
  )

  const getYearLabel = () => {
    if (selectedYear === 0) return 'All Time'
    if (selectedYear === new Date().getFullYear()) return 'This Year'
    return selectedYear.toString()
  }

  return (
    <div className="analytics-page">
      {/* Page Header */}
      <div style={{
        background: 'linear-gradient(135deg, #f8f9ff 0%, #e8f0ff 100%)',
        borderRadius: '16px',
        padding: '2rem',
        marginBottom: '2rem',
        border: '1px solid #e1e8f7'
      }}>
        <h1 style={{
          fontSize: '2rem',
          margin: '0 0 0.5rem 0',
          color: '#2c3e50',
          fontWeight: '700',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          <span style={{ fontSize: '2.5rem' }}>📈</span>
          Analytics - {getYearLabel()}
        </h1>
        <p style={{
          fontSize: '1rem',
          color: '#666',
          margin: 0
        }}>
          Deep dive into karma patterns, trends, and community engagement metrics.
        </p>
      </div>

      {/* Charts Grid */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
        gap: '2rem',
        marginBottom: '2rem'
      }}>
        {/* Karma Distribution */}
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
              Loading distribution...
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

        {/* Points Over Time */}
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
            Points Trend
          </h3>
          {pointsTimeLoading ? (
            <div style={{ 
              height: '300px', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              color: '#999'
            }}>
              Loading trend...
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

      {/* Top Emojis */}
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
          <span>😊</span>
          Popular Emoji Reactions
        </h3>
        {emojisLoading ? (
          <div style={{ 
            height: '300px', 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            color: '#999'
          }}>
            Loading emoji stats...
          </div>
        ) : topEmojis && topEmojis.length > 0 ? (
          <BarChart 
            data={topEmojis} 
            xKey="emoji_name" 
            yKey="usage_count"
          />
        ) : (
          <div style={{ 
            height: '300px', 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            color: '#999'
          }}>
            No emoji data available
          </div>
        )}
      </div>
    </div>
  )
}

export default AnalyticsPage