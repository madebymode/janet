import React, { useState, useEffect } from 'react'

function LeaderboardTable({ data = [], loading, error, onUserClick, showClickHint = false }) {
  // Mobile-responsive state
  const [isMobile, setIsMobile] = useState(window.innerWidth <= 768)
  
  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth <= 768)
    }
    
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])
  
  const mobileStyles = {
    table: {
      fontSize: isMobile ? '0.8rem' : '0.9rem'
    },
    headerPadding: isMobile ? '0.75rem 0.5rem' : '1rem 0.75rem',
    cellPadding: isMobile ? '0.75rem 0.5rem' : '1rem 0.75rem',
    avatarSize: isMobile ? '32px' : '40px',
    maxNameLength: isMobile ? 12 : 18
  }
  if (loading) {
    return (
      <div className="table-container" style={{ 
        overflowX: 'auto',
        margin: isMobile ? '0 -0.5rem' : '0',
        padding: isMobile ? '0 0.5rem' : '0'
      }}>
        <table className="table" style={{
          width: '100%',
          borderCollapse: 'collapse',
          ...mobileStyles.table
        }}>
          <thead>
            <tr style={{
              background: '#f8f9fa',
              borderBottom: '2px solid #e9ecef'
            }}>
              <th style={{ padding: mobileStyles.headerPadding, textAlign: 'left', fontWeight: '600', color: '#495057' }}>Rank</th>
              <th style={{ padding: mobileStyles.headerPadding, textAlign: 'left', fontWeight: '600', color: '#495057' }}>User</th>
              <th style={{ padding: mobileStyles.headerPadding, textAlign: 'right', fontWeight: '600', color: '#495057' }}>Total Points</th>
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
          padding: isMobile ? '0.5rem' : '0.75rem',
          marginBottom: '1rem',
          fontSize: isMobile ? '0.8rem' : '0.875rem',
          color: '#1976d2',
          textAlign: 'center'
        }}>
          💡 {isMobile ? 'Tap any user for details' : 'Click on any user to view their detailed statistics'}
        </div>
      )}
      
      <div className="table-container" style={{ 
        overflowX: 'auto',
        margin: isMobile ? '0 -0.5rem' : '0',
        padding: isMobile ? '0 0.5rem' : '0'
      }}>
        <table className="table" style={{
          width: '100%',
          borderCollapse: 'collapse',
          ...mobileStyles.table
        }}>
          <thead>
            <tr style={{
              background: '#f8f9fa',
              borderBottom: '2px solid #e9ecef'
            }}>
              <th style={{ padding: mobileStyles.headerPadding, textAlign: 'left', fontWeight: '600', color: '#495057' }}>Rank</th>
              <th style={{ padding: mobileStyles.headerPadding, textAlign: 'left', fontWeight: '600', color: '#495057' }}>User</th>
              <th style={{ padding: mobileStyles.headerPadding, textAlign: 'right', fontWeight: '600', color: '#495057' }}>Total Points</th>
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
                      transition: 'background-color 0.2s ease',
                      minHeight: isMobile ? '56px' : 'auto'
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
                    <td style={{ padding: mobileStyles.cellPadding }}>
                      <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: isMobile ? '0.25rem' : '0.5rem'
                      }}>
                        {rank <= 3 && (
                          <span style={{ fontSize: isMobile ? '1rem' : '1.2rem' }}>
                            {rank === 1 ? '🥇' : rank === 2 ? '🥈' : '🥉'}
                          </span>
                        )}
                        <span style={{
                          fontWeight: rank <= 3 ? '600' : '500',
                          color: rank <= 3 ? '#2c3e50' : '#666',
                          fontSize: isMobile ? '0.8rem' : '1rem'
                        }}>
                          #{rank}
                        </span>
                      </div>
                    </td>
                    <td style={{ padding: mobileStyles.cellPadding }}>
                      <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: isMobile ? '0.5rem' : '0.75rem'
                      }}>
                        <div style={{
                          width: mobileStyles.avatarSize,
                          height: mobileStyles.avatarSize,
                          borderRadius: '50%',
                          background: avatarUrl ? `url(${avatarUrl})` : '#667eea linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                          backgroundSize: 'cover',
                          backgroundPosition: 'center',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          color: 'white',
                          fontWeight: '600',
                          fontSize: isMobile ? '0.75rem' : '0.9rem',
                          flexShrink: 0
                        }}>
                          {!avatarUrl && (displayName.charAt(0).toUpperCase())}
                        </div>
                        <div style={{ minWidth: 0, flex: 1 }}>
                          <div style={{
                            fontWeight: '600',
                            color: '#2c3e50',
                            fontSize: isMobile ? '0.85rem' : '0.95rem',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap'
                          }}>
                            {displayName.length > mobileStyles.maxNameLength ? `${displayName.substring(0, mobileStyles.maxNameLength)}...` : displayName}
                          </div>
                          {displayName !== username && (
                            <div style={{
                              fontSize: isMobile ? '0.7rem' : '0.8rem',
                              color: '#999',
                              fontStyle: 'italic',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap'
                            }}>
                              @{username.length > (mobileStyles.maxNameLength - 2) ? `${username.substring(0, mobileStyles.maxNameLength - 2)}...` : username}
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td style={{ 
                      padding: mobileStyles.cellPadding, 
                      textAlign: 'right',
                      fontWeight: '600',
                      color: totalPoints >= 0 ? '#27ae60' : '#e74c3c',
                      fontSize: isMobile ? '0.85rem' : '1rem'
                    }}>
                      {isNaN(totalPoints) ? '0' : totalPoints.toLocaleString()}
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
