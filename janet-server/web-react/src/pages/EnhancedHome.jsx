import React, { useState, useEffect } from 'react'
import { useApi } from '../hooks/useApi'
import LeaderboardTable from '../components/LeaderboardTable'

// Function to convert emoji names to actual emojis
const getEmojiFromName = (emojiName) => {
  const emojiMap = {
    'joy': '😂',
    '100': '💯', 
    'heart': '❤️',
    'lol': '😂',
    '+1': '👍',
    'skull': '💀',
    'sparkles': '✨',
    'yellow_heart': '💛',
    '-1': '👎',
    'fire': '🔥',
    'clap': '👏',
    'eyes': '👀',
    'thumbsup': '👍',
    'thumbsdown': '👎',
    'ok_hand': '👌',
    'wave': '👋',
    'raised_hands': '🙌',
    'party': '🎉',
    'tada': '🎉'
  }
  return emojiMap[emojiName] || emojiName
}

// Simple chart components using CSS and basic SVG
const PieChart = ({ data, title }) => {
  if (!data || data.length === 0) return <div className="chart-placeholder">No data available</div>

  const total = data.reduce((sum, item) => sum + item.count, 0)
  let currentAngle = 0
  const radius = 80
  const centerX = 100
  const centerY = 100

  const colors = [
    '#667eea', '#764ba2', '#7b68ee', '#9f7aea', '#a78bfa',
    '#c084fc', '#ddd6fe', '#e0e7ff', '#818cf8', '#6366f1'
  ]

  const slices = data.map((item, index) => {
    const percentage = (item.count / total) * 100
    const angle = (item.count / total) * 360
    const x1 = centerX + radius * Math.cos((currentAngle * Math.PI) / 180)
    const y1 = centerY + radius * Math.sin((currentAngle * Math.PI) / 180)
    const x2 = centerX + radius * Math.cos(((currentAngle + angle) * Math.PI) / 180)
    const y2 = centerY + radius * Math.sin(((currentAngle + angle) * Math.PI) / 180)
    
    const largeArcFlag = angle > 180 ? 1 : 0
    const pathData = `M ${centerX} ${centerY} L ${x1} ${y1} A ${radius} ${radius} 0 ${largeArcFlag} 1 ${x2} ${y2} Z`
    
    currentAngle += angle

    const gradientId = `gradient-${index}`

    return (
      <g key={index}>
        <defs>
          <linearGradient id={gradientId} x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor={colors[index % colors.length]} stopOpacity="1" />
            <stop offset="100%" stopColor={colors[index % colors.length]} stopOpacity="0.7" />
          </linearGradient>
          <filter id={`shadow-${index}`}>
            <feDropShadow dx="2" dy="2" stdDeviation="3" floodOpacity="0.3"/>
          </filter>
        </defs>
        <path
          d={pathData}
          fill={`url(#${gradientId})`}
          stroke="white"
          strokeWidth="2"
          style={{
            cursor: 'pointer'
          }}
          onClick={() => {
            alert(`${item.range}: ${item.count} users (${((item.count / total) * 100).toFixed(1)}%)`)
          }}
        />
      </g>
    )
  })

  return (
    <div className="chart-container">
      <h4>{title}</h4>
      <div style={{ display: 'flex', alignItems: 'center', gap: '20px', flexWrap: 'wrap' }}>
        <svg width="200" height="200" viewBox="0 0 200 200" style={{ flexShrink: 0 }}>
          {slices}
        </svg>
        <div className="chart-legend">
          {data.map((item, index) => (
            <div key={index} style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
              <div
                style={{
                  width: '12px',
                  height: '12px',
                  backgroundColor: colors[index % colors.length],
                  borderRadius: '2px'
                }}
              />
              <span style={{ fontSize: '0.875rem' }}>
                {item.range}: {item.count} ({((item.count / total) * 100).toFixed(1)}%)
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

const LineChart = ({ data, title, xKey, yKey }) => {
  if (!data || data.length === 0) return <div className="chart-placeholder">No data available</div>

  const maxY = Math.max(...data.map(d => d[yKey]))
  const minY = Math.min(...data.map(d => d[yKey]))
  const range = maxY - minY || 1

  const width = 400
  const height = 200
  const padding = 40

  const points = data.map((d, i) => {
    const x = padding + (i / (data.length - 1)) * (width - padding * 2)
    const y = height - padding - ((d[yKey] - minY) / range) * (height - padding * 2)
    return `${x},${y}`
  }).join(' ')

  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

  return (
    <div className="chart-container">
      <h4>{title}</h4>
      <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{ border: '1px solid #ddd', borderRadius: '4px', maxWidth: '100%', height: 'auto' }}>
        {/* Grid lines */}
        {[0, 0.25, 0.5, 0.75, 1].map(ratio => (
          <line
            key={ratio}
            x1={padding}
            y1={padding + ratio * (height - padding * 2)}
            x2={width - padding}
            y2={padding + ratio * (height - padding * 2)}
            stroke="#f0f0f0"
            strokeWidth="1"
          />
        ))}
        
        {/* Y-axis labels */}
        {[0, 0.25, 0.5, 0.75, 1].map(ratio => (
          <text
            key={ratio}
            x={padding - 10}
            y={padding + ratio * (height - padding * 2) + 4}
            textAnchor="end"
            fontSize="12"
            fill="#666"
          >
            {Math.round(maxY - ratio * range).toLocaleString()}
          </text>
        ))}

        {/* X-axis labels */}
        {data.map((d, i) => (
          <text
            key={i}
            x={padding + (i / (data.length - 1)) * (width - padding * 2)}
            y={height - 10}
            textAnchor="middle"
            fontSize="12"
            fill="#666"
          >
            {monthNames[d[xKey] - 1]}
          </text>
        ))}

        {/* Line */}
        <polyline
          points={points}
          fill="none"
          stroke="#667eea"
          strokeWidth="3"
          strokeLinejoin="round"
        />

        {/* Data points */}
        {data.map((d, i) => {
          const x = padding + (i / (data.length - 1)) * (width - padding * 2)
          const y = height - padding - ((d[yKey] - minY) / range) * (height - padding * 2)
          return (
            <circle
              key={i}
              cx={x}
              cy={y}
              r="4"
              fill="#667eea"
              stroke="white"
              strokeWidth="2"
              style={{
                cursor: 'pointer'
              }}
              onClick={() => {
                alert(`${monthNames[d[xKey] - 1]}: ${d[yKey].toLocaleString()} points`)
              }}
            />
          )
        })}
      </svg>
    </div>
  )
}

const BarChart = ({ data, title, xKey, yKey }) => {
  if (!data || data.length === 0) return <div className="chart-placeholder">No data available</div>

  const maxY = Math.max(...data.map(d => d[yKey]))
  const width = 400
  const height = 200
  const padding = 40
  const barWidth = (width - padding * 2) / data.length * 0.8

  const colors = [
    '#667eea', '#764ba2', '#7b68ee', '#9f7aea', '#a78bfa',
    '#c084fc', '#ddd6fe', '#e0e7ff', '#818cf8', '#6366f1'
  ]

  return (
    <div className="chart-container">
      <h4>{title}</h4>
      <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{ border: '1px solid #ddd', borderRadius: '4px', maxWidth: '100%', height: 'auto' }}>
        {/* Grid lines */}
        {[0, 0.25, 0.5, 0.75, 1].map(ratio => (
          <line
            key={ratio}
            x1={padding}
            y1={padding + ratio * (height - padding * 2)}
            x2={width - padding}
            y2={padding + ratio * (height - padding * 2)}
            stroke="#f0f0f0"
            strokeWidth="1"
          />
        ))}
        
        {/* Y-axis labels */}
        {[0, 0.25, 0.5, 0.75, 1].map(ratio => (
          <text
            key={ratio}
            x={padding - 10}
            y={padding + ratio * (height - padding * 2) + 4}
            textAnchor="end"
            fontSize="12"
            fill="#666"
          >
            {Math.round(maxY - ratio * maxY).toLocaleString()}
          </text>
        ))}

        {/* Bars */}
        {data.map((d, i) => {
          const x = padding + (i / data.length) * (width - padding * 2) + (width - padding * 2) / data.length * 0.1
          const barHeight = (d[yKey] / maxY) * (height - padding * 2)
          const y = height - padding - barHeight
          
          return (
            <g key={i}>
              <rect
                x={x}
                y={y}
                width={barWidth}
                height={barHeight}
                fill={colors[i % colors.length]}
                rx="2"
                style={{
                  cursor: 'pointer'
                }}
                onClick={() => {
                  alert(`${d[xKey]}: ${d[yKey].toLocaleString()} uses`)
                }}
              />
              <text
                x={x + barWidth / 2}
                y={height - 10}
                textAnchor="middle"
                fontSize="16"
                fill="#666"
              >
                {getEmojiFromName(d[xKey])}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}

const ActivityFeed = ({ activities }) => {
  if (!activities || activities.length === 0) return (
    <div style={{ 
      textAlign: 'center', 
      padding: '3rem',
      color: '#7f8c8d',
      fontSize: '1.1rem'
    }}>
      <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>😴</div>
      <div>No recent activity... yet!</div>
      <div style={{ fontSize: '0.875rem', marginTop: '0.5rem' }}>
        Be the first to spread some karma! ✨
      </div>
    </div>
  )

  const getActivityIcon = (activity) => {
    if (activity.transactionType === 'emoji') {
      return activity.emojiName ? getEmojiFromName(activity.emojiName) : '😀'
    }
    return activity.points > 0 ? '⬆️' : '⬇️'
  }

  const getRandomFloatingEmoji = () => {
    const emojis = ['✨', '🌟', '💫', '⭐', '🎉', '🎊', '🔥', '💝']
    return emojis[Math.floor(Math.random() * emojis.length)]
  }

  return (
    <div className="activity-feed">
      <h3 style={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: '8px',
        marginBottom: '1.5rem',
        fontSize: '1.5rem'
      }}>
        Recent Activity
      </h3>
      <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
        {activities.slice(0, 15).map((activity, index) => (
          <div 
            key={index} 
            className="activity-item" 
            style={{ 
              padding: '12px', 
              borderRadius: '8px',
              background: '#f8f9fa',
              marginBottom: '8px',
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              border: '1px solid #e9ecef'
            }}
          >
            {/* Activity type indicator */}
            <div style={{
              width: '24px',
              height: '24px',
              borderRadius: '50%',
              backgroundColor: activity.points > 0 ? '#27ae60' : '#e74c3c',
              flexShrink: 0
            }} />
            
            <div style={{ flex: 1, fontSize: '0.875rem' }}>
              <strong>{activity.from}</strong> gave <strong>{activity.to}</strong>
              <span style={{ 
                color: activity.points > 0 ? '#27ae60' : '#e74c3c',
                fontWeight: 'bold',
                margin: '0 4px'
              }}>
                {activity.points > 0 ? '+' : ''}{activity.points}
              </span> points
              {activity.reason && (
                <div style={{ 
                  color: '#666', 
                  fontSize: '0.75rem',
                  marginTop: '2px'
                }}>
                  {activity.transactionType === 'emoji' && activity.emojiName 
                    ? `${activity.from} added a ${getEmojiFromName(activity.emojiName)} emoji`
                    : activity.reason
                  }
                </div>
              )}
              <div style={{ 
                color: '#999', 
                fontSize: '0.75rem',
                marginTop: '2px'
              }}>
                {new Date(activity.date).toLocaleDateString()}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

// Component to display user points over time
const UserPointsOverTimeChart = ({ userData, userMonthlyData, selectedUser, selectedYear }) => {
  if (!userData) return <div className="chart-placeholder">No data available</div>

  // Process real monthly data from the API
  const processMonthlyData = (monthlyApiData) => {
    const months = [
      'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
    ]
    
    // Create array for all 12 months, filling in zeros for missing months
    const monthlyData = []
    
    months.forEach((monthName, index) => {
      const monthNumber = index + 1
      const monthData = monthlyApiData?.find(d => d.month === monthNumber)
      
      monthlyData.push({
        month: monthNumber,
        monthName: monthName,
        monthlyPoints: monthData?.monthlyTotal || 0,
        cumulativePoints: monthData?.cumulativePoints || 0,
        pointsReceived: monthData?.pointsReceived || 0,
        pointsGiven: monthData?.pointsGiven || 0,
        transactionsReceived: monthData?.transactionsReceived || 0,
        transactionsGiven: monthData?.transactionsGiven || 0,
      })
    })
    
    return monthlyData
  }

  const monthlyData = userMonthlyData ? processMonthlyData(userMonthlyData) : []
  const maxPoints = Math.max(...monthlyData.map(d => d.cumulativePoints))
  
  const width = 600
  const height = 300
  const padding = 50

  const points = monthlyData.map((d, i) => {
    const x = padding + (i / (monthlyData.length - 1)) * (width - padding * 2)
    const y = height - padding - ((d.cumulativePoints / maxPoints) * (height - padding * 2))
    return `${x},${y}`
  }).join(' ')

  return (
    <div className="chart-container">
      <div style={{ display: 'flex', gap: '2rem', flexWrap: 'wrap' }}>
        {/* Cumulative Points Chart */}
        <div style={{ flex: 1, minWidth: '400px' }}>
          <h4>📈 Cumulative Points Over Time</h4>
          <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{ border: '1px solid #ddd', borderRadius: '4px', maxWidth: '100%', height: 'auto' }}>
            {/* Grid lines */}
            {[0, 0.25, 0.5, 0.75, 1].map(ratio => (
              <line
                key={ratio}
                x1={padding}
                y1={padding + ratio * (height - padding * 2)}
                x2={width - padding}
                y2={padding + ratio * (height - padding * 2)}
                stroke="#f0f0f0"
                strokeWidth="1"
              />
            ))}
            
            {/* Y-axis labels */}
            {[0, 0.25, 0.5, 0.75, 1].map(ratio => (
              <text
                key={ratio}
                x={padding - 10}
                y={padding + ratio * (height - padding * 2) + 4}
                textAnchor="end"
                fontSize="12"
                fill="#666"
              >
                {Math.round(maxPoints - ratio * maxPoints).toLocaleString()}
              </text>
            ))}

            {/* X-axis labels */}
            {monthlyData.map((d, i) => (
              <text
                key={i}
                x={padding + (i / (monthlyData.length - 1)) * (width - padding * 2)}
                y={height - 10}
                textAnchor="middle"
                fontSize="12"
                fill="#666"
              >
                {d.monthName}
              </text>
            ))}

            {/* Area under the curve */}
            <defs>
              <linearGradient id="userGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stopColor="#667eea" stopOpacity="0.4" />
                <stop offset="100%" stopColor="#667eea" stopOpacity="0.1" />
              </linearGradient>
            </defs>
            <path
              d={`M${padding},${height - padding} L${points} L${width - padding},${height - padding} Z`}
              fill="url(#userGradient)"
            />

            {/* Line */}
            <polyline
              points={points}
              fill="none"
              stroke="#667eea"
              strokeWidth="3"
              strokeLinejoin="round"
            />

            {/* Data points */}
            {monthlyData.map((d, i) => {
              const x = padding + (i / (monthlyData.length - 1)) * (width - padding * 2)
              const y = height - padding - ((d.cumulativePoints / maxPoints) * (height - padding * 2))
              return (
                <circle
                  key={i}
                  cx={x}
                  cy={y}
                  r="5"
                  fill="#667eea"
                  stroke="white"
                  strokeWidth="2"
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    alert(`${d.monthName}: ${d.cumulativePoints.toLocaleString()} total points (+${d.monthlyPoints.toLocaleString()} received, ${d.pointsGiven.toLocaleString()} given)`)
                  }}
                />
              )
            })}
          </svg>
        </div>

        {/* Monthly Summary */}
        <div style={{ minWidth: '250px' }}>
          <h4>📊 Monthly Breakdown</h4>
          <div style={{ maxHeight: '250px', overflowY: 'auto', border: '1px solid #ddd', borderRadius: '4px', padding: '8px' }}>
            {monthlyData.map((d, i) => (
              <div key={i} style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                padding: '4px 8px',
                backgroundColor: i % 2 === 0 ? '#f8f9fa' : 'white',
                borderRadius: '2px',
                marginBottom: '2px'
              }}>
                <span style={{ fontWeight: '500' }}>{d.monthName}</span>
                <span style={{ color: '#667eea', fontWeight: 'bold' }}>
                  +{d.monthlyPoints.toLocaleString()} (↑{d.pointsReceived.toLocaleString()})
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
      
      <div style={{ 
        marginTop: '1rem', 
        padding: '0.75rem', 
        backgroundColor: '#f8f9fa', 
        borderRadius: '4px',
        fontSize: '0.875rem',
        color: '#666'
      }}>
        💡 <strong>Note:</strong> Data shows actual monthly points received by {selectedUser} for {selectedYear}. 
        Click on chart points to see detailed breakdown for each month.
      </div>
    </div>
  )
}

function EnhancedHome() {
  const [activeTab, setActiveTab] = useState('overview')
  const [selectedUser, setSelectedUser] = useState('')
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear())
  
  const { data: stats, loading: statsLoading, error: statsError } = useApi(`/api/stats?year=${selectedYear}`)
  const { data: leaderboard, loading: leaderboardLoading, error: leaderboardError } = useApi(`/api/leaderboard?limit=50&year=${selectedYear}`)
  const { data: status, loading: statusLoading, error: statusError } = useApi('/api/status')
  const { data: topGivers, loading: topGiversLoading } = useApi(`/api/stats/top-givers?limit=5&year=${selectedYear}`)
  const { data: karmaDistribution, loading: karmaDistLoading } = useApi(`/api/stats/karma-distribution?year=${selectedYear}`)
  const { data: pointsOverTime, loading: pointsTimeLoading } = useApi(`/api/stats/points-over-time?year=${selectedYear}`)
  const { data: topEmojis, loading: emojisLoading } = useApi(`/api/stats/emojis?limit=8&year=${selectedYear}`)
  const { data: recentActivity, loading: activityLoading } = useApi(`/api/stats/recent-activity?year=${selectedYear}`)
  
  // Get user-specific data when a user is selected
  const { data: userData, loading: userLoading } = useApi(
    selectedUser ? `/api/user/${selectedUser}/${selectedYear}` : null
  )
  
  // Get user-specific monthly data when a user is selected
  const { data: userMonthlyData, loading: monthlyDataLoading } = useApi(
    selectedUser ? `/api/user/${selectedUser}/${selectedYear}/points-over-time` : null
  )

  return (
    <div className="enhanced-home">
      {/* Tab Navigation */}
      <div className="nav" style={{ 
        marginBottom: '2rem'
      }}>
        <div className="container">
          <ul className="nav-list">
            <li>
              <button 
                className={`nav-link ${activeTab === 'overview' ? 'active' : ''}`}
                onClick={() => setActiveTab('overview')}
                style={{ 
                  border: 'none', 
                  background: 'none',
                  fontSize: '1rem',
                  fontWeight: '600',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }}
              >
                <span style={{ fontSize: '1.2rem' }}>📊</span>
                Dashboard
              </button>
            </li>
            <li>
              <button 
                className={`nav-link ${activeTab === 'leaderboard' ? 'active' : ''}`}
                onClick={() => setActiveTab('leaderboard')}
                style={{ 
                  border: 'none', 
                  background: 'none',
                  fontSize: '1rem',
                  fontWeight: '600',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }}
              >
                <span style={{ fontSize: '1.2rem' }}>🏆</span>
                Leaderboard
              </button>
            </li>
            <li>
              <button 
                className={`nav-link ${activeTab === 'analytics' ? 'active' : ''}`}
                onClick={() => setActiveTab('analytics')}
                style={{ 
                  border: 'none', 
                  background: 'none',
                  fontSize: '1rem',
                  fontWeight: '600',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }}
              >
                <span style={{ fontSize: '1.2rem' }}>📈</span>
                Analytics
              </button>
            </li>
            <li>
              <button 
                className={`nav-link ${activeTab === 'activity' ? 'active' : ''}`}
                onClick={() => setActiveTab('activity')}
                style={{ 
                  border: 'none', 
                  background: 'none',
                  fontSize: '1rem',
                  fontWeight: '600',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }}
              >
                <span style={{ fontSize: '1.2rem' }}>🔄</span>
                Activity
              </button>
            </li>
            <li>
              <button 
                className={`nav-link ${activeTab === 'user' ? 'active' : ''}`}
                onClick={() => setActiveTab('user')}
                style={{ 
                  border: 'none', 
                  background: 'none',
                  fontSize: '1rem',
                  fontWeight: '600',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px'
                }}
              >
                <span style={{ fontSize: '1.2rem' }}>👤</span>
                User Stats
              </button>
            </li>
          </ul>
        </div>
      </div>

      {/* Overview Tab */}
      {activeTab === 'overview' && (
        <>
          {/* Key Stats */}
          <div className="stats-grid">
            <div className="stat-card" style={{ animationDelay: '0.1s' }}>
              <div className="stat-icon" style={{ fontSize: '2rem' }}>👥</div>
              <div className="stat-content">
                <div className="stat-number">
                  {statsLoading ? (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <div className="loading-spinner"></div>
                      <span style={{ fontSize: '1rem' }}>Loading...</span>
                    </div>
                  ) : statsError ? (
                    <span style={{ color: '#e74c3c', fontSize: '1rem' }}>Error</span>
                  ) : (
                    (stats?.totalUsers || 0).toLocaleString()
                  )}
                </div>
                <div className="stat-label">👨‍👩‍👧‍👦 Community Members</div>
              </div>
            </div>
            <div className="stat-card" style={{ animationDelay: '0.2s' }}>
              <div className="stat-icon" style={{ fontSize: '2rem' }}>⭐</div>
              <div className="stat-content">
                <div className="stat-number">
                  {statsLoading ? (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <div className="loading-spinner"></div>
                      <span style={{ fontSize: '1rem' }}>Loading...</span>
                    </div>
                  ) : statsError ? (
                    <span style={{ color: '#e74c3c', fontSize: '1rem' }}>Error</span>
                  ) : (
                    (stats?.totalPoints || 0).toLocaleString()
                  )}
                </div>
                <div className="stat-label">Karma Points Earned</div>
              </div>
            </div>
            <div className="stat-card" style={{ animationDelay: '0.3s' }}>
              <div className="stat-icon" style={{ fontSize: '2rem' }}>🔄</div>
              <div className="stat-content">
                <div className="stat-number">
                  {statsLoading ? (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <div className="loading-spinner"></div>
                      <span style={{ fontSize: '1rem' }}>Loading...</span>
                    </div>
                  ) : statsError ? (
                    <span style={{ color: '#e74c3c', fontSize: '1rem' }}>Error</span>
                  ) : (
                    (stats?.totalTransactions || 0).toLocaleString()
                  )}
                </div>
                <div className="stat-label">Total Interactions</div>
              </div>
            </div>
            <div className="stat-card" style={{ animationDelay: '0.4s' }}>
              <div className="stat-icon" style={{ fontSize: '2rem' }}>🤖</div>
              <div className="stat-content">
                <div className={`status-indicator ${statusLoading ? 'status-loading' : statusError ? 'status-error' : status?.botOnline ? 'status-online' : 'status-offline'}`}></div>
                <div className="stat-label">Janet Bot Status</div>
                <div style={{ 
                  fontSize: '0.875rem', 
                  fontWeight: 'bold',
                  color: status?.botOnline ? '#27ae60' : '#e74c3c'
                }}>
                  {status?.botOnline ? 'Online' : 'Offline'}
                </div>
              </div>
            </div>
          </div>

          {/* Quick Charts Overview */}
          <div className="charts-grid">
            <div className="card">
              {!karmaDistLoading && karmaDistribution && (
                <PieChart 
                  data={karmaDistribution} 
                  title="📊 Karma Distribution"
                />
              )}
            </div>
            <div className="card">
              {!pointsTimeLoading && pointsOverTime && (
                <LineChart 
                  data={pointsOverTime} 
                  title="📈 Points Over Time" 
                  xKey="month" 
                  yKey="totalPoints"
                />
              )}
            </div>
          </div>

          {/* About Section */}
          <div className="card">
            <h2>About Good Place Judge</h2>
            <p>
              Good Place Judge is a Slack bot that tracks karma points for team members. 
              Users can give or take points from each other using simple commands:
            </p>
            <div style={{ 
              display: 'grid', 
              gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
              gap: '1rem', 
              marginTop: '1rem' 
            }}>
              <div style={{ padding: '1rem', background: '#f8f9fa', borderRadius: '6px' }}>
                <code>@username++</code>
                <div style={{ fontSize: '0.875rem', color: '#666', marginTop: '0.5rem' }}>Give points</div>
              </div>
              <div style={{ padding: '1rem', background: '#f8f9fa', borderRadius: '6px' }}>
                <code>@username--</code>
                <div style={{ fontSize: '0.875rem', color: '#666', marginTop: '0.5rem' }}>Take points</div>
              </div>
              <div style={{ padding: '1rem', background: '#f8f9fa', borderRadius: '6px' }}>
                <code>@username?</code>
                <div style={{ fontSize: '0.875rem', color: '#666', marginTop: '0.5rem' }}>Check points</div>
              </div>
              <div style={{ padding: '1rem', background: '#f8f9fa', borderRadius: '6px' }}>
                <code>goodplace leaderboard</code>
                <div style={{ fontSize: '0.875rem', color: '#666', marginTop: '0.5rem' }}>Show leaderboard</div>
              </div>
            </div>
          </div>
        </>
      )}

      {/* Leaderboard Tab */}
      {activeTab === 'leaderboard' && (
        <>
          <div className="card">
            <div className="card-header">
              <span className="icon">🏆</span>
              <h2>Leaderboard</h2>
            </div>
            <div className="table-container">
              <LeaderboardTable 
                data={leaderboard?.users || []} 
                loading={leaderboardLoading} 
                error={leaderboardError} 
              />
            </div>
          </div>

          <div className="card">
            <div className="card-header">
              <span className="icon">🎁</span>
              <h2>Top Givers</h2>
            </div>
            {topGiversLoading ? (
              <div className="loading">Loading top givers...</div>
            ) : topGivers?.users ? (
              <div className="table-container">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Rank</th>
                      <th>User</th>
                      <th>Points Given</th>
                      <th>Transactions</th>
                      <th>Current Points</th>
                    </tr>
                  </thead>
                  <tbody>
                    {topGivers.users.map((user, index) => (
                      <tr key={user.username}>
                        <td>#{index + 1}</td>
                        <td><strong>{user.username}</strong></td>
                        <td style={{ color: '#2ecc71', fontWeight: 'bold' }}>
                          {user.pointsGiven.toLocaleString()}
                        </td>
                        <td>{user.transactionsGiven.toLocaleString()}</td>
                        <td style={{ 
                          color: user.totalPoints >= 0 ? '#2ecc71' : '#e74c3c',
                          fontWeight: 'bold'
                        }}>
                          {user.totalPoints.toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div>No data available</div>
            )}
          </div>
        </>
      )}

      {/* Analytics Tab */}
      {activeTab === 'analytics' && (
        <>
          <div className="charts-grid">
            <div className="card">
              {!karmaDistLoading && karmaDistribution && (
                <PieChart 
                  data={karmaDistribution} 
                  title="📊 Karma Distribution"
                />
              )}
            </div>
            <div className="card">
              {!pointsTimeLoading && pointsOverTime && (
                <LineChart 
                  data={pointsOverTime} 
                  title="📈 Points Over Time" 
                  xKey="month" 
                  yKey="totalPoints"
                />
              )}
            </div>
          </div>

          <div className="card">
            <div className="card-header">
              <span className="icon">😊</span>
              <h2>Top Emojis</h2>
            </div>
            {emojisLoading ? (
              <div className="loading">Loading emoji stats...</div>
            ) : topEmojis && topEmojis.length > 0 ? (
              <BarChart 
                data={topEmojis.slice(0, 8)} 
                title="" 
                xKey="emoji_name" 
                yKey="usage_count"
              />
            ) : (
              <div>No emoji data available</div>
            )}
          </div>
        </>
      )}

      {/* Activity Tab */}
      {activeTab === 'activity' && (
        <div className="card">
          {activityLoading ? (
            <div className="loading">Loading recent activity...</div>
          ) : recentActivity ? (
            <ActivityFeed activities={recentActivity} />
          ) : (
            <div>No recent activity</div>
          )}
        </div>
      )}

      {/* User Stats Tab */}
      {activeTab === 'user' && (
        <>
          {/* User Selection Controls */}
          <div className="card">
            <div className="card-header">
              <span className="icon">👤</span>
              <h2>User Statistics</h2>
            </div>
            <div style={{ 
              display: 'flex', 
              gap: '1rem', 
              alignItems: 'center', 
              flexWrap: 'wrap',
              marginBottom: '1rem'
            }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', minWidth: '200px' }}>
                <label style={{ fontWeight: '600', color: '#2c3e50' }}>Select User:</label>
                <select
                  value={selectedUser}
                  onChange={(e) => setSelectedUser(e.target.value)}
                  style={{
                    padding: '0.5rem',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '1rem',
                    backgroundColor: 'white'
                  }}
                >
                  <option value="">Choose a user...</option>
                  {leaderboard?.users?.map(user => (
                    <option key={user.username} value={user.username}>
                      {user.username}
                    </option>
                  ))}
                </select>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', minWidth: '120px' }}>
                <label style={{ fontWeight: '600', color: '#2c3e50' }}>Year:</label>
                <select
                  value={selectedYear}
                  onChange={(e) => setSelectedYear(e.target.value === "all" ? "all" : parseInt(e.target.value))}
                  style={{
                    padding: '0.5rem',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '1rem',
                    backgroundColor: 'white'
                  }}
                >
                  {(() => {
                    const currentYear = new Date().getFullYear();
                    const years = [];
                    for (let year = currentYear; year >= 2023; year--) {
                      years.push(<option key={year} value={year}>{year}</option>);
                    }
                    return years;
                  })()}
                  <option value="all">All Time</option>
                </select>
              </div>
            </div>
          </div>

          {/* User Stats Display */}
          {selectedUser && (
            <>
              {/* User Overview Stats */}
              <div className="stats-grid">
                <div className="stat-card">
                  <div className="stat-icon" style={{ fontSize: '2rem' }}>⭐</div>
                  <div className="stat-content">
                    <div className="stat-number">
                      {userLoading ? (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <div className="loading-spinner"></div>
                          <span style={{ fontSize: '1rem' }}>Loading...</span>
                        </div>
                      ) : (
                        (userData?.total_points || 0).toLocaleString()
                      )}
                    </div>
                    <div className="stat-label">Total Points ({selectedYear})</div>
                  </div>
                </div>
                <div className="stat-card">
                  <div className="stat-icon" style={{ fontSize: '2rem' }}>🎁</div>
                  <div className="stat-content">
                    <div className="stat-number">
                      {userLoading ? (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <div className="loading-spinner"></div>
                          <span style={{ fontSize: '1rem' }}>Loading...</span>
                        </div>
                      ) : (
                        (userData?.points_given || 0).toLocaleString()
                      )}
                    </div>
                    <div className="stat-label">Points Given</div>
                  </div>
                </div>
                <div className="stat-card">
                  <div className="stat-icon" style={{ fontSize: '2rem' }}>📥</div>
                  <div className="stat-content">
                    <div className="stat-number">
                      {userLoading ? (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <div className="loading-spinner"></div>
                          <span style={{ fontSize: '1rem' }}>Loading...</span>
                        </div>
                      ) : (
                        (userData?.points_received || 0).toLocaleString()
                      )}
                    </div>
                    <div className="stat-label">Points Received</div>
                  </div>
                </div>
                <div className="stat-card">
                  <div className="stat-icon" style={{ fontSize: '2rem' }}>🏆</div>
                  <div className="stat-content">
                    <div className="stat-number">
                      {userLoading ? (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <div className="loading-spinner"></div>
                          <span style={{ fontSize: '1rem' }}>Loading...</span>
                        </div>
                      ) : (
                        `#${userData?.rank || 'N/A'}`
                      )}
                    </div>
                    <div className="stat-label">Current Rank</div>
                  </div>
                </div>
              </div>

              {/* User Points Over Time Chart */}
              <div className="card">
                <div className="card-header">
                  <span className="icon">📈</span>
                  <h3>Points Over Time - {selectedUser} ({selectedYear})</h3>
                </div>
                {!userLoading && !monthlyDataLoading && userData && (
                  <UserPointsOverTimeChart 
                    userData={userData} 
                    userMonthlyData={userMonthlyData}
                    selectedUser={selectedUser} 
                    selectedYear={selectedYear} 
                  />
                )}
              </div>
            </>
          )}

          {!selectedUser && (
            <div className="card">
              <div style={{ 
                textAlign: 'center', 
                padding: '3rem',
                color: '#7f8c8d',
                fontSize: '1.1rem'
              }}>
                <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>👤</div>
                <div>Select a user above to view their detailed statistics</div>
                <div style={{ fontSize: '0.875rem', marginTop: '0.5rem' }}>
                  Choose from the dropdown to see points, rankings, and activity trends
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}

export default EnhancedHome