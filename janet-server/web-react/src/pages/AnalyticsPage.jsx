import React from 'react'
import { useApi } from '../hooks/useApi'
import PieChart from '../components/charts/PieChart'
import LineChart from '../components/charts/LineChart'
import BarChart from '../components/charts/BarChart'

function AnalyticsPage({ selectedYear, selectedMonth = 0 }) {
  const monthParam = selectedMonth > 0 ? `&month=${selectedMonth}` : ''
  const yearParam = selectedYear === 0 ? '' : `year=${selectedYear}`

  const { data: karmaDistribution, loading: karmaDistLoading } = useApi(
    selectedYear === 0 && !selectedMonth
      ? '/api/stats/karma-distribution'
      : `/api/stats/karma-distribution?${yearParam}${monthParam}`
  )
  const { data: pointsOverTime, loading: pointsTimeLoading } = useApi(
    selectedYear === 0
      ? '/api/stats/points-over-time'
      : `/api/stats/points-over-time?year=${selectedYear}`
  )
  const { data: topEmojis, loading: emojisLoading } = useApi(
    selectedYear === 0 && !selectedMonth
      ? '/api/stats/emojis?limit=10'
      : `/api/stats/emojis?limit=10&${yearParam}${monthParam}`
  )
  const { data: colludersData, loading: colludersLoading } = useApi(
    selectedYear === 0 && !selectedMonth
      ? '/api/stats/colluders?limit=10'
      : `/api/stats/colluders?limit=10&${yearParam}${monthParam}`
  )

  const getYearLabel = () => {
    if (selectedMonth > 0) {
      const monthName = new Date(selectedYear, selectedMonth - 1).toLocaleString('en-US', { month: 'long' })
      return `${monthName} ${selectedYear}`
    }
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
        border: '1px solid #f0f0f0',
        marginBottom: '2rem'
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

      {/* Karma Colluders */}
      <div style={{
        background: 'linear-gradient(135deg, #f0f4ff 0%, #e8eeff 100%)',
        borderRadius: '14px',
        padding: '1.5rem',
        border: '1px solid #c5d0f5'
      }}>
        <h3 style={{
          margin: '0 0 0.4rem 0',
          fontSize: '1.2rem',
          fontWeight: '600',
          color: '#1a237e',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          <span>🤝</span>
          Karma Colluders — {getYearLabel()}
        </h3>
        <p style={{ margin: '0 0 1rem 0', fontSize: '0.85rem', color: '#3949ab' }}>
          Pairs with the highest mutual karma exchange, and the channels where it happens most.
        </p>
        {colludersLoading ? (
          <div style={{ color: '#3949ab', fontSize: '0.95rem' }}>Loading colluders...</div>
        ) : colludersData?.colluders?.length > 0 ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            {colludersData.colluders.map((c, i) => {
              const name1 = c.user1_display_name || `@${c.user1}`
              const name2 = c.user2_display_name || `@${c.user2}`
              const medals = ['🥇', '🥈', '🥉']
              return (
                <div key={`${c.user1}-${c.user2}`} style={{
                  background: 'white',
                  borderRadius: '10px',
                  padding: '1rem',
                  border: '1px solid #dde3f7',
                  boxShadow: '0 1px 4px rgba(0,0,0,0.04)'
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', flexWrap: 'wrap', marginBottom: c.top_channels?.length > 0 ? '0.75rem' : 0 }}>
                    <span style={{ fontSize: '1.2rem', flexShrink: 0 }}>{medals[i] || `#${i + 1}`}</span>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', flex: 1, flexWrap: 'wrap', minWidth: 0 }}>
                      {c.user1_avatar && <img src={c.user1_avatar} alt={name1} style={{ width: '24px', height: '24px', borderRadius: '50%' }} />}
                      <span style={{ fontWeight: '700', color: '#1a237e', fontSize: '0.9rem' }}>{name1}</span>
                      <span style={{ color: '#667eea', fontWeight: '600', fontSize: '0.82rem' }}>{c.user1_to_user2.toLocaleString()} pts</span>
                      <span style={{ color: '#999', fontSize: '0.85rem' }}>⇄</span>
                      <span style={{ color: '#667eea', fontWeight: '600', fontSize: '0.82rem' }}>{c.user2_to_user1.toLocaleString()} pts</span>
                      {c.user2_avatar && <img src={c.user2_avatar} alt={name2} style={{ width: '24px', height: '24px', borderRadius: '50%' }} />}
                      <span style={{ fontWeight: '700', color: '#1a237e', fontSize: '0.9rem' }}>{name2}</span>
                    </div>
                    <div style={{
                      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                      color: 'white',
                      padding: '0.25rem 0.75rem',
                      borderRadius: '12px',
                      fontSize: '0.8rem',
                      fontWeight: '700',
                      flexShrink: 0
                    }}>
                      {c.total_exchange.toLocaleString()} total
                    </div>
                  </div>
                  {c.top_channels?.length > 0 && (
                    <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', paddingLeft: '2rem' }}>
                      <span style={{ fontSize: '0.78rem', color: '#666', alignSelf: 'center' }}>mainly in:</span>
                      {c.top_channels.map(ch => (
                        <span key={ch.channel_name} style={{
                          background: '#eef0ff',
                          color: '#3949ab',
                          padding: '0.15rem 0.6rem',
                          borderRadius: '10px',
                          fontSize: '0.78rem',
                          fontWeight: '600'
                        }}>
                          #{ch.channel_name} <span style={{ opacity: 0.7 }}>({ch.count})</span>
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        ) : (
          <div style={{ color: '#3949ab', fontSize: '0.9rem' }}>No colluder data for this period.</div>
        )}
      </div>
    </div>
  )
}

export default AnalyticsPage