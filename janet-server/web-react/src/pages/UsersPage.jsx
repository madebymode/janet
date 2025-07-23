import React from 'react'
import { useApi } from '../hooks/useApi'
import StatCard from '../components/StatCard'
import LineChart from '../components/charts/LineChart'

function UsersPage({ selectedYear, selectedUser, onUserChange, onYearChange }) {
  // Get list of users for selection
  const { data: leaderboard } = useApi(
    selectedYear === 0 
      ? '/api/leaderboard?limit=100' 
      : `/api/leaderboard?limit=100&year=${selectedYear}`
  )

  // Get user-specific data by year (if year is selected) or all-time data
  const { data: userData, loading: userLoading } = useApi(
    selectedUser
      ? selectedYear === 0 
        ? `/api/user/${selectedUser}`
        : `/api/user/${selectedUser}/${selectedYear}`
      : null
  )

  // Get user-specific points over time data
  const { data: userPointsOverTime, loading: monthlyLoading } = useApi(
    selectedUser
      ? selectedYear === 0
        ? `/api/user/${selectedUser}/points-over-time/all`
        : `/api/user/${selectedUser}/${selectedYear}/points-over-time`
      : null
  )

  const getYearLabel = () => {
    if (selectedYear === 0) return 'All Time'
    if (selectedYear === new Date().getFullYear()) return 'This Year'
    return selectedYear.toString()
  }

  // Since we don't have user-specific monthly data, 
  // we'll show a message about this limitation

  return (
    <div className="users-page">
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
          <span style={{ fontSize: '2.5rem' }}>👥</span>
          User Statistics - {getYearLabel()}
        </h1>
        <p style={{
          fontSize: '1rem',
          color: '#666',
          margin: 0
        }}>
          Explore individual user performance, trends, and karma journey over time.
        </p>
      </div>

      {/* User Selection */}
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
          <span>🔍</span>
          Select User
        </h3>
        
        <div style={{
          display: 'flex',
          gap: '1rem',
          alignItems: 'center',
          flexWrap: 'wrap'
        }}>
          <div style={{ flex: 1, minWidth: '200px' }}>
            <select
              value={selectedUser}
              onChange={(e) => onUserChange(e.target.value)}
              style={{
                width: '100%',
                padding: '0.75rem',
                border: '2px solid #e9ecef',
                borderRadius: '8px',
                fontSize: '1rem',
                backgroundColor: 'white',
                color: '#495057',
                fontWeight: '500'
              }}
            >
              <option value="">Choose a user...</option>
              {leaderboard?.users?.map(user => (
                <option key={user.username} value={user.username}>
                  {user.username} ({(user.total_points || 0).toLocaleString()} points)
                </option>
              ))}
            </select>
          </div>
          
          {selectedUser && (
            <div style={{
              background: '#e8f5e8',
              color: '#155724',
              padding: '0.5rem 1rem',
              borderRadius: '20px',
              fontSize: '0.875rem',
              fontWeight: '500',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem'
            }}>
              <span>✅</span>
              Viewing: {selectedUser}
            </div>
          )}
        </div>
      </div>

      {selectedUser ? (
        <>
          {/* User Stats Cards */}
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
            gap: '1.5rem',
            marginBottom: '2rem'
          }}>
            <StatCard
              icon="⭐"
              title="Total Points"
              value={userData?.total_points || userData?.points || 0}
              loading={userLoading}
              subtitle={`${getYearLabel()} performance`}
              color="#e74c3c"
            />
            <StatCard
              icon="🎁"
              title="Points Given"
              value={userData?.points_given || 0}
              loading={userLoading}
              subtitle="Generosity score"
              color="#f39c12"
            />
            <StatCard
              icon="📥"
              title="Points Received"
              value={userData?.points_received || 0}
              loading={userLoading}
              subtitle="Recognition earned"
              color="#27ae60"
            />
            <StatCard
              icon="🏆"
              title="Current Rank"
              value={userData?.rank ? `#${userData.rank}` : 'N/A'}
              loading={userLoading}
              isStatus={true}
              subtitle="Leaderboard position"
              color="#9b59b6"
            />
          </div>

          {/* User Points Over Time Chart */}
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
              <span>📈</span>
              Points Over Time - {selectedUser} ({getYearLabel()})
            </h3>
            
            {monthlyLoading ? (
              <div style={{ 
                height: '300px', 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center',
                color: '#999'
              }}>
                Loading chart...
              </div>
            ) : userPointsOverTime && userPointsOverTime.length > 0 ? (
              <LineChart 
                data={userPointsOverTime} 
                xKey={selectedYear === 0 ? "year" : "month"} 
                yKey="totalPoints"
              />
            ) : (
              <div style={{
                height: '200px',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#666',
                textAlign: 'center',
                background: '#f8f9fa',
                borderRadius: '8px',
                padding: '2rem'
              }}>
                <div style={{ fontSize: '2rem', marginBottom: '1rem' }}>📊</div>
                <div style={{ fontSize: '1.1rem', fontWeight: '600', marginBottom: '0.5rem' }}>
                  No data for {getYearLabel()}
                </div>
                <div style={{ fontSize: '0.9rem', lineHeight: 1.5 }}>
                  {selectedUser} had no karma activity {selectedYear === 0 ? 'in any year' : `in ${selectedYear}`}.<br/>
                  Try selecting a different {selectedYear === 0 ? 'user' : 'year or user'}.
                </div>
              </div>
            )}
          </div>

          {/* User Details Card */}
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
              Detailed Statistics
            </h3>
            
            {userLoading ? (
              <div style={{ padding: '2rem', textAlign: 'center', color: '#999' }}>
                Loading user details...
              </div>
            ) : userData ? (
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                gap: '1rem'
              }}>
                {[
                  { label: 'Total Points', value: (userData.total_points || userData.points || 0).toLocaleString(), icon: '⭐' },
                  { label: 'Points Given', value: (userData.points_given || 0).toLocaleString(), icon: '🎁' },
                  { label: 'Points Received', value: (userData.points_received || 0).toLocaleString(), icon: '📥' },
                  { label: 'Transactions Given', value: (userData.transactions_given || 0).toLocaleString(), icon: '📤' },
                  { label: 'Transactions Received', value: (userData.transactions_received || 0).toLocaleString(), icon: '📨' },
                  { label: 'Emoji Reactions Given', value: (userData.emoji_reactions_given || 0).toLocaleString(), icon: '😊' },
                  { label: 'Emoji Reactions Received', value: (userData.emoji_reactions_received || 0).toLocaleString(), icon: '🎭' },
                  { label: 'Current Rank', value: userData.rank ? `#${userData.rank}` : 'N/A', icon: '🏆' }
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
                      color: '#2c3e50'
                    }}>
                      {stat.value}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div style={{ padding: '2rem', textAlign: 'center', color: '#999' }}>
                No data available for {selectedUser}
              </div>
            )}
          </div>
        </>
      ) : (
        <div className="card" style={{
          background: 'white',
          borderRadius: '12px',
          padding: '3rem',
          boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
          border: '1px solid #f0f0f0',
          textAlign: 'center'
        }}>
          <div style={{ fontSize: '4rem', marginBottom: '1rem' }}>👤</div>
          <h3 style={{
            fontSize: '1.3rem',
            color: '#2c3e50',
            marginBottom: '0.5rem'
          }}>
            Select a User to View Statistics
          </h3>
          <p style={{
            color: '#666',
            fontSize: '1rem',
            marginBottom: '1.5rem'
          }}>
            Choose from the dropdown above to explore individual user performance, 
            track their karma journey, and see detailed analytics.
          </p>
          <div style={{
            background: '#f8f9fa',
            padding: '1rem',
            borderRadius: '8px',
            fontSize: '0.875rem',
            color: '#666'
          }}>
            💡 Users are sorted by total points in the dropdown for easy browsing
          </div>
        </div>
      )}
    </div>
  )
}

export default UsersPage