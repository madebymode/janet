import React from 'react'

function LeaderboardTable({ data = [], loading, error, onUserClick, showClickHint = false }) {
  if (loading) {
    return (
      <div className="table-container" style={{ overflowX: 'auto' }}>
        <table className="table" style={{
          width: '100%',
          borderCollapse: 'collapse',
          fontSize: '0.9rem'
        }}>
          <thead>
            <tr style={{
              background: '#f8f9fa',
              borderBottom: '2px solid #e9ecef'
            }}>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'left', fontWeight: '600', color: '#495057' }}>Rank</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'left', fontWeight: '600', color: '#495057' }}>User</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'right', fontWeight: '600', color: '#495057' }}>Total Points</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'right', fontWeight: '600', color: '#495057' }}>Given</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'right', fontWeight: '600', color: '#495057' }}>Received</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'center', fontWeight: '600', color: '#495057' }}>Activity</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td colSpan="6" style={{ 
                textAlign: 'center', 
                padding: '3rem',
                color: '#999',
                fontSize: '1rem'
              }}>
                <div style={{ fontSize: '2rem', marginBottom: '0.5rem' }}>⏳</div>
                <div>Loading leaderboard...</div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        Failed to load leaderboard: {error}
      </div>
    )
  }

  return (
    <div>
      {showClickHint && (
        <div style={{
          background: '#e3f2fd',
          border: '1px solid #2196f3',
          borderRadius: '8px',
          padding: '0.75rem',
          marginBottom: '1rem',
          fontSize: '0.875rem',
          color: '#1976d2',
          textAlign: 'center'
        }}>
          💡 Click on any user to view their detailed statistics
        </div>
      )}
      
      <div className="table-container" style={{ overflowX: 'auto' }}>
        <table className="table" style={{
          width: '100%',
          borderCollapse: 'collapse',
          fontSize: '0.9rem'
        }}>
          <thead>
            <tr style={{
              background: '#f8f9fa',
              borderBottom: '2px solid #e9ecef'
            }}>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'left', fontWeight: '600', color: '#495057' }}>Rank</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'left', fontWeight: '600', color: '#495057' }}>User</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'right', fontWeight: '600', color: '#495057' }}>Total Points</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'right', fontWeight: '600', color: '#495057' }}>Given</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'right', fontWeight: '600', color: '#495057' }}>Received</th>
              <th style={{ padding: '1rem 0.75rem', textAlign: 'center', fontWeight: '600', color: '#495057' }}>Activity</th>
            </tr>
          </thead>
          <tbody>
            {!data || data.length === 0 ? (
              <tr>
                <td colSpan="6" style={{ 
                  textAlign: 'center', 
                  padding: '3rem',
                  color: '#999',
                  fontSize: '1rem'
                }}>
                  <div style={{ fontSize: '3rem', marginBottom: '0.5rem' }}>🏆</div>
                  <div>No users found</div>
                </td>
              </tr>
            ) : (
              data.map((user, index) => {
                const username = user.username || user.Name || user.name || ''
                const displayName = user.display_name || user.real_name || username
                const avatarUrl = user.avatar_url
                const rank = user.rank || index + 1
                const totalPoints = Number(user.total_points || user.totalPoints || user.Points || user.points || 0)
                const pointsGiven = Number(user.points_given || user.pointsGiven || 0)
                const pointsReceived = Number(user.points_received || user.pointsReceived || totalPoints)
                const transactions = Number(user.transactions_given || user.transactionsGiven || 0) + 
                                   Number(user.transactions_received || user.transactionsReceived || 0)
                
                return (
                  <tr 
                    key={username || index}
                    style={{
                      borderBottom: '1px solid #f0f0f0',
                      cursor: onUserClick ? 'pointer' : 'default',
                      transition: 'background-color 0.2s ease'
                    }}
                    onClick={() => onUserClick && onUserClick(username)}
                    onMouseEnter={(e) => {
                      if (onUserClick) {
                        e.currentTarget.style.backgroundColor = '#f8f9fa'
                      }
                    }}
                    onMouseLeave={(e) => {
                      if (onUserClick) {
                        e.currentTarget.style.backgroundColor = 'transparent'
                      }
                    }}
                  >
                    <td style={{ padding: '1rem 0.75rem' }}>
                      <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '0.5rem'
                      }}>
                        {rank <= 3 && (
                          <span style={{ fontSize: '1.2rem' }}>
                            {rank === 1 ? '🥇' : rank === 2 ? '🥈' : '🥉'}
                          </span>
                        )}
                        <span style={{
                          fontWeight: rank <= 3 ? '600' : '500',
                          color: rank <= 3 ? '#2c3e50' : '#666'
                        }}>
                          #{rank}
                        </span>
                      </div>
                    </td>
                    <td style={{ padding: '1rem 0.75rem' }}>
                      <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '0.75rem'
                      }}>
                        <div style={{
                          width: '40px',
                          height: '40px',
                          borderRadius: '50%',
                          background: avatarUrl ? `url(${avatarUrl})` : '#667eea linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                          backgroundSize: 'cover',
                          backgroundPosition: 'center',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          color: 'white',
                          fontWeight: '600',
                          fontSize: '0.9rem',
                          flexShrink: 0
                        }}>
                          {!avatarUrl && (displayName.charAt(0).toUpperCase())}
                        </div>
                        <div>
                          <div style={{
                            fontWeight: '600',
                            color: '#2c3e50',
                            fontSize: '0.95rem'
                          }}>
                            {displayName.length > 18 ? `${displayName.substring(0, 18)}...` : displayName}
                          </div>
                          {displayName !== username && (
                            <div style={{
                              fontSize: '0.8rem',
                              color: '#999',
                              fontStyle: 'italic'
                            }}>
                              @{username}
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td style={{ 
                      padding: '1rem 0.75rem', 
                      textAlign: 'right',
                      fontWeight: '600',
                      color: totalPoints >= 0 ? '#27ae60' : '#e74c3c'
                    }}>
                      {isNaN(totalPoints) ? '0' : totalPoints.toLocaleString()}
                    </td>
                    <td style={{ 
                      padding: '1rem 0.75rem', 
                      textAlign: 'right',
                      color: '#666'
                    }}>
                      {isNaN(pointsGiven) ? '0' : pointsGiven.toLocaleString()}
                    </td>
                    <td style={{ 
                      padding: '1rem 0.75rem', 
                      textAlign: 'right',
                      color: '#666'
                    }}>
                      {isNaN(pointsReceived) ? '0' : pointsReceived.toLocaleString()}
                    </td>
                    <td style={{ 
                      padding: '1rem 0.75rem', 
                      textAlign: 'center'
                    }}>
                      <div style={{
                        background: transactions > 50 ? '#e8f5e8' : transactions > 20 ? '#fff3cd' : '#f8d7da',
                        color: transactions > 50 ? '#155724' : transactions > 20 ? '#856404' : '#721c24',
                        padding: '0.25rem 0.5rem',
                        borderRadius: '12px',
                        fontSize: '0.8rem',
                        fontWeight: '500',
                        display: 'inline-block'
                      }}>
                        {isNaN(transactions) ? '0' : transactions}
                      </div>
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default LeaderboardTable