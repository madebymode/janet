import React from 'react'
import LeaderboardTable from '../components/LeaderboardTable'

function LeaderboardPage({ selectedYear, onUserChange, leaderboardData, leaderboardLoading, leaderboardError, topGiversData, topGiversLoading, topGiversError }) {

  const getYearLabel = () => {
    if (selectedYear === 0) return 'All Time'
    if (selectedYear === new Date().getFullYear()) return 'This Year'
    return selectedYear.toString()
  }

  const handleUserClick = (username) => {
    onUserChange(username)
  }

  return (
    <div className="leaderboard-page">
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
          <span style={{ fontSize: '2.5rem' }}>🏆</span>
          Leaderboard - {getYearLabel()}
        </h1>
        <p style={{
          fontSize: '1rem',
          color: '#666',
          margin: 0
        }}>
          See who's leading the karma race and recognize top contributors to team positivity.
        </p>
      </div>

      <div style={{
        display: 'grid',
        gridTemplateColumns: '2fr 1fr',
        gap: '2rem',
        alignItems: 'start'
      }}>
        {/* Main Leaderboard */}
        <div className="card" style={{
          background: 'white',
          borderRadius: '12px',
          padding: '1.5rem',
          boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
          border: '1px solid #f0f0f0'
        }}>
          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '1.5rem'
          }}>
            <h2 style={{
              margin: 0,
              fontSize: '1.3rem',
              fontWeight: '600',
              color: '#2c3e50',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem'
            }}>
              <span>🏅</span>
              Top Performers
            </h2>
            <div style={{
              fontSize: '0.875rem',
              color: '#666',
              background: '#f8f9fa',
              padding: '0.25rem 0.75rem',
              borderRadius: '12px',
              fontWeight: '500'
            }}>
              Top {Math.min(25, leaderboardData?.users?.length || 0)} of {leaderboardData?.users?.length || 0}
            </div>
          </div>
          
          <div style={{ position: 'relative' }}>
            <LeaderboardTable 
              data={(leaderboardData?.users || []).slice(0, 25)} 
              loading={leaderboardLoading} 
              error={leaderboardError}
              onUserClick={handleUserClick}
              showClickHint={true}
            />
          </div>
        </div>

        {/* Top Givers Sidebar */}
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
            <span>🎁</span>
            Most Generous
          </h3>
          
          {topGiversLoading ? (
            <div style={{
              padding: '2rem',
              textAlign: 'center',
              color: '#999'
            }}>
              Loading top givers...
            </div>
          ) : topGiversError ? (
            <div className="alert alert-error">
              Failed to load top givers: {topGiversError}
            </div>
          ) : topGiversData?.users ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              {topGiversData.users.map((user, index) => {
                const displayName = user.display_name || user.real_name || user.username
                const avatarUrl = user.avatar_url
                
                return (
                  <div
                    key={user.username}
                    style={{
                      padding: '1rem',
                      background: index === 0 ? '#fff8e1' : index === 1 ? '#f3e5f5' : index === 2 ? '#e8f5e8' : '#f8f9fa',
                      border: `1px solid ${index === 0 ? '#ffc107' : index === 1 ? '#9c27b0' : index === 2 ? '#4caf50' : '#e0e0e0'}20`,
                      borderRadius: '8px',
                      cursor: 'pointer',
                      transition: 'transform 0.2s ease, box-shadow 0.2s ease',
                      position: 'relative'
                    }}
                    onClick={() => handleUserClick(user.username)}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.transform = 'translateY(-2px)'
                      e.currentTarget.style.boxShadow = '0 4px 15px rgba(0,0,0,0.1)'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.transform = 'translateY(0)'
                      e.currentTarget.style.boxShadow = 'none'
                    }}
                  >
                    <div style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.75rem'
                    }}>
                      <div style={{
                        width: '32px',
                        height: '32px',
                        borderRadius: '50%',
                        background: avatarUrl ? `url(${avatarUrl})` : (index === 0 ? '#ffc107' : index === 1 ? '#9c27b0' : index === 2 ? '#4caf50' : '#667eea'),
                        backgroundSize: 'cover',
                        backgroundPosition: 'center',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        color: 'white',
                        fontWeight: '600',
                        fontSize: '0.875rem',
                        flexShrink: 0
                      }}>
                        {avatarUrl ? '' : (index === 0 ? '🥇' : index === 1 ? '🥈' : index === 2 ? '🥉' : displayName.charAt(0).toUpperCase())}
                      </div>
                      
                      <div style={{ flex: 1 }}>
                        <div style={{
                          fontWeight: '600',
                          color: '#2c3e50',
                          fontSize: '0.9rem',
                          marginBottom: '0.25rem'
                        }}>
                          {displayName.length > 18 ? `${displayName.substring(0, 18)}...` : displayName}
                        </div>
                        {displayName !== user.username && (
                          <div style={{
                            fontSize: '0.75rem',
                            color: '#999',
                            fontStyle: 'italic',
                            marginBottom: '0.25rem'
                          }}>
                            @{user.username}
                          </div>
                        )}
                        <div style={{
                          fontSize: '0.8rem',
                          color: '#666'
                        }}>
                          <span style={{ color: '#e74c3c', fontWeight: '600' }}>
                            {Number(user.points_given || 0).toLocaleString()}
                          </span> points given
                        </div>
                        <div style={{
                          fontSize: '0.75rem',
                          color: '#999'
                        }}>
                          {Number(user.transactions_given || 0).toLocaleString()} transactions
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <div style={{
              padding: '2rem',
              textAlign: 'center',
              color: '#999'
            }}>
              No data available
            </div>
          )}
          
          <div style={{
            marginTop: '1rem',
            padding: '0.75rem',
            background: '#f8f9fa',
            borderRadius: '8px',
            fontSize: '0.8rem',
            color: '#666',
            textAlign: 'center'
          }}>
            💡 Click on any user to view their detailed stats
          </div>
        </div>
      </div>

      {/* Fun Stats Section */}
      {leaderboardData?.users && leaderboardData.users.length > 0 && (
        <div className="card" style={{
          background: 'white',
          borderRadius: '12px',
          padding: '2rem',
          boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
          border: '1px solid #f0f0f0',
          marginTop: '2rem'
        }}>
          <h3 style={{
            margin: '0 0 1.5rem 0',
            fontSize: '1.2rem',
            fontWeight: '600',
            color: '#2c3e50',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <span>📊</span>
            Quick Stats
          </h3>
          
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
            gap: '1rem'
          }}>
            {[
              {
                icon: '👑',
                label: 'Current Leader',
                value: (() => {
                  const leader = leaderboardData.users[0]
                  if (!leader) return 'N/A'
                  const displayName = leader.display_name || leader.real_name || leader.username
                  return displayName.length > 20 ? `${displayName.substring(0, 20)}...` : displayName
                })(),
                subtitle: `${Number(leaderboardData.users[0]?.total_points || 0).toLocaleString()} points`
              },
              {
                icon: '📈',
                label: 'Average Points',
                value: Math.round(leaderboardData.users.reduce((sum, user) => sum + Number(user.total_points || 0), 0) / leaderboardData.users.length).toLocaleString(),
                subtitle: 'per active member'
              },
              {
                icon: '🎯',
                label: 'Top 10% Threshold',
                value: Number(leaderboardData.users[Math.floor(leaderboardData.users.length * 0.1)]?.total_points || 0).toLocaleString() || 'N/A',
                subtitle: 'points needed'
              },
              {
                icon: '⚡',
                label: 'Most Active',
                value: (() => {
                  const mostActiveUser = leaderboardData.users.reduce((max, user) => 
                    (Number(user.transactions_received || 0) + Number(user.transactions_given || 0)) > 
                    (Number(max.transactions_received || 0) + Number(max.transactions_given || 0)) ? user : max
                  , leaderboardData.users[0]);
                  if (!mostActiveUser) return 'N/A'
                  const displayName = mostActiveUser.display_name || mostActiveUser.real_name || mostActiveUser.username
                  return displayName.length > 20 ? `${displayName.substring(0, 20)}...` : displayName;
                })(),
                subtitle: 'total transactions'
              }
            ].map((stat, index) => (
              <div
                key={index}
                style={{
                  padding: '1rem',
                  background: '#f8f9fa',
                  borderRadius: '8px',
                  textAlign: 'center'
                }}
              >
                <div style={{ fontSize: '1.5rem', marginBottom: '0.5rem' }}>
                  {stat.icon}
                </div>
                <div style={{
                  fontSize: '0.875rem',
                  color: '#666',
                  marginBottom: '0.25rem'
                }}>
                  {stat.label}
                </div>
                <div style={{
                  fontSize: '1.1rem',
                  fontWeight: '600',
                  color: '#2c3e50',
                  marginBottom: '0.25rem'
                }}>
                  {stat.value}
                </div>
                <div style={{
                  fontSize: '0.75rem',
                  color: '#999'
                }}>
                  {stat.subtitle}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default LeaderboardPage