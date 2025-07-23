import React, { useState, useEffect } from 'react'
import { useApi } from '../hooks/useApi'

// Simple emoji name to emoji mapping
const getEmojiFromName = (emojiName) => {
  const emojiMap = {
    'joy': '😂', '100': '💯', 'heart': '❤️', 'lol': '😂', '+1': '👍', 'skull': '💀',
    'sparkles': '✨', 'yellow_heart': '💛', '-1': '👎', 'fire': '🔥', 'clap': '👏',
    'eyes': '👀', 'thumbsup': '👍', 'thumbsdown': '👎', 'ok_hand': '👌', 'wave': '👋',
    'raised_hands': '🙌', 'party': '🎉', 'tada': '🎉', 'star': '⭐', 'rocket': '🚀'
  }
  return emojiMap[emojiName] || emojiName || '😀'
}

function ActivityPage({ selectedYear, onUserChange }) {
  const [currentPage, setCurrentPage] = useState(1)
  const [fromUser, setFromUser] = useState('')
  const [toUser, setToUser] = useState('')
  const [searchFromUser, setSearchFromUser] = useState('')
  const [searchToUser, setSearchToUser] = useState('')
  const itemsPerPage = 20

  // Reset to first page when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [selectedYear, searchFromUser, searchToUser])

  // Build API URL with pagination and search parameters
  const buildApiUrl = () => {
    const params = new URLSearchParams()
    params.set('limit', itemsPerPage.toString())
    params.set('offset', ((currentPage - 1) * itemsPerPage).toString())
    
    if (selectedYear !== 0) {
      params.set('year', selectedYear.toString())
    }
    
    if (searchFromUser.trim()) {
      params.set('from', searchFromUser.trim())
    }
    
    if (searchToUser.trim()) {
      params.set('to', searchToUser.trim())
    }
    
    return `/api/stats/recent-activity?${params.toString()}`
  }

  const { data: activityData, loading: activityLoading, refetch } = useApi(buildApiUrl())
  
  const recentActivity = activityData?.activities || []
  const pagination = activityData?.pagination || {}

  const getYearLabel = () => {
    if (selectedYear === 0) return 'All Time'
    if (selectedYear === new Date().getFullYear()) return 'This Year'
    return selectedYear.toString()
  }

  const handleUserClick = (username) => {
    onUserChange(username)
  }

  const handleSearch = () => {
    setSearchFromUser(fromUser)
    setSearchToUser(toUser)
  }

  const handleClearSearch = () => {
    setFromUser('')
    setToUser('')
    setSearchFromUser('')
    setSearchToUser('')
  }

  const handlePageChange = (newPage) => {
    setCurrentPage(newPage)
  }

  const renderPaginationControls = () => {
    if (!pagination.total || pagination.total <= itemsPerPage) return null

    const totalPages = pagination.totalPages || 1
    const currentPageNum = pagination.currentPage || 1
    
    return (
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginTop: '1.5rem',
        padding: '1rem',
        background: '#f8f9fa',
        borderRadius: '8px'
      }}>
        <div style={{ fontSize: '0.875rem', color: '#666' }}>
          Showing {Math.min(pagination.offset + 1, pagination.total)}-{Math.min(pagination.offset + itemsPerPage, pagination.total)} of {pagination.total} transactions
        </div>
        
        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
          <button
            onClick={() => handlePageChange(currentPageNum - 1)}
            disabled={currentPageNum <= 1}
            style={{
              padding: '0.5rem 1rem',
              border: '1px solid #e1e8f7',
              borderRadius: '6px',
              background: currentPageNum <= 1 ? '#f8f9fa' : 'white',
              color: currentPageNum <= 1 ? '#999' : '#2c3e50',
              cursor: currentPageNum <= 1 ? 'not-allowed' : 'pointer',
              fontSize: '0.875rem',
              fontWeight: '500'
            }}
          >
            ← Previous
          </button>
          
          <span style={{ 
            fontSize: '0.875rem', 
            color: '#666',
            margin: '0 1rem'
          }}>
            Page {currentPageNum} of {totalPages}
          </span>
          
          <button
            onClick={() => handlePageChange(currentPageNum + 1)}
            disabled={!pagination.hasMore}
            style={{
              padding: '0.5rem 1rem',
              border: '1px solid #e1e8f7',
              borderRadius: '6px',
              background: !pagination.hasMore ? '#f8f9fa' : 'white',
              color: !pagination.hasMore ? '#999' : '#2c3e50',
              cursor: !pagination.hasMore ? 'not-allowed' : 'pointer',
              fontSize: '0.875rem',
              fontWeight: '500'
            }}
          >
            Next →
          </button>
        </div>
      </div>
    )
  }

  if (activityLoading) {
    return (
      <div className="activity-page">
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '400px',
          color: '#999',
          fontSize: '1.1rem'
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: '1rem'
          }}>
            <div style={{
              width: '24px',
              height: '24px',
              border: '3px solid #f0f0f0',
              borderTop: '3px solid #667eea',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite'
            }} />
            Loading recent activity...
          </div>
        </div>
      </div>
    )
  }

  if (!recentActivity || recentActivity.length === 0) {
    return (
      <div className="activity-page">
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
            <span style={{ fontSize: '2.5rem' }}>🔄</span>
            Activity Feed - {getYearLabel()}
          </h1>
          <p style={{
            fontSize: '1rem',
            color: '#666',
            margin: 0
          }}>
            Real-time karma transactions and community interactions.
          </p>
        </div>

        <div style={{
          textAlign: 'center',
          padding: '4rem',
          color: '#999',
          fontSize: '1.1rem'
        }}>
          <div style={{ fontSize: '4rem', marginBottom: '1rem' }}>😴</div>
          <div style={{ fontSize: '1.3rem', marginBottom: '0.5rem' }}>No recent activity... yet!</div>
          <div style={{ fontSize: '0.9rem' }}>
            Be the first to spread some karma! ✨
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="activity-page">
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
          <span style={{ fontSize: '2.5rem' }}>🔄</span>
          Activity Feed - {getYearLabel()}
        </h1>
        <p style={{
          fontSize: '1rem',
          color: '#666',
          margin: 0
        }}>
          Real-time karma transactions and community interactions.
        </p>
      </div>

      {/* Search and Filter Controls */}
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
          Search & Filter
        </h3>
        
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
          gap: '1rem',
          marginBottom: '1rem'
        }}>
          <div>
            <label style={{
              display: 'block',
              fontSize: '0.875rem',
              fontWeight: '500',
              color: '#2c3e50',
              marginBottom: '0.5rem'
            }}>
              From User
            </label>
            <input
              type="text"
              value={fromUser}
              onChange={(e) => setFromUser(e.target.value)}
              placeholder="Search by sender..."
              style={{
                width: '100%',
                padding: '0.75rem',
                border: '2px solid #e9ecef',
                borderRadius: '8px',
                fontSize: '1rem',
                backgroundColor: 'white',
                color: '#495057'
              }}
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            />
          </div>
          
          <div>
            <label style={{
              display: 'block',
              fontSize: '0.875rem',
              fontWeight: '500',
              color: '#2c3e50',
              marginBottom: '0.5rem'
            }}>
              To User
            </label>
            <input
              type="text"
              value={toUser}
              onChange={(e) => setToUser(e.target.value)}
              placeholder="Search by receiver..."
              style={{
                width: '100%',
                padding: '0.75rem',
                border: '2px solid #e9ecef',
                borderRadius: '8px',
                fontSize: '1rem',
                backgroundColor: 'white',
                color: '#495057'
              }}
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            />
          </div>
        </div>
        
        <div style={{
          display: 'flex',
          gap: '1rem',
          alignItems: 'center',
          flexWrap: 'wrap'
        }}>
          <button
            onClick={handleSearch}
            style={{
              padding: '0.75rem 1.5rem',
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              fontSize: '0.875rem',
              fontWeight: '600',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem'
            }}
          >
            <span>🔍</span>
            Search
          </button>
          
          <button
            onClick={handleClearSearch}
            style={{
              padding: '0.75rem 1.5rem',
              background: '#f8f9fa',
              color: '#6c757d',
              border: '1px solid #e9ecef',
              borderRadius: '8px',
              fontSize: '0.875rem',
              fontWeight: '500',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem'
            }}
          >
            <span>🗑️</span>
            Clear
          </button>
          
          {(searchFromUser || searchToUser) && (
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
              Active filters: {[searchFromUser && `from: ${searchFromUser}`, searchToUser && `to: ${searchToUser}`].filter(Boolean).join(', ')}
            </div>
          )}
        </div>
      </div>

      {/* Activity Feed */}
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
            <span>📝</span>
            Recent Transactions
          </h2>
          <div style={{
            fontSize: '0.875rem',
            color: '#666',
            background: '#f8f9fa',
            padding: '0.25rem 0.75rem',
            borderRadius: '12px',
            fontWeight: '500'
          }}>
            {pagination.total ? `${pagination.total} total activities` : `${recentActivity.length} activities`}
          </div>
        </div>

        <div style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '0.75rem',
          maxHeight: '600px',
          overflowY: 'auto'
        }}>
          {recentActivity.map((activity, index) => (
            <div
              key={index}
              style={{
                padding: '1rem',
                borderRadius: '10px',
                background: '#f8f9fa',
                border: '1px solid #e9ecef',
                display: 'flex',
                alignItems: 'center',
                gap: '1rem',
                transition: 'transform 0.2s ease, box-shadow 0.2s ease',
                cursor: 'default'
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-2px)'
                e.currentTarget.style.boxShadow = '0 4px 15px rgba(0,0,0,0.1)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = 'none'
              }}
            >
              {/* Activity Icon */}
              <div style={{
                width: '40px',
                height: '40px',
                borderRadius: '50%',
                background: activity.points > 0 
                  ? 'linear-gradient(135deg, #27ae60, #2ecc71)' 
                  : 'linear-gradient(135deg, #e74c3c, #c0392b)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'white',
                fontSize: '1.1rem',
                fontWeight: '600',
                flexShrink: 0
              }}>
                {activity.transactionType === 'emoji' 
                  ? getEmojiFromName(activity.emojiName) 
                  : activity.points > 0 ? '⬆️' : '⬇️'
                }
              </div>

              {/* Activity Details */}
              <div style={{ flex: 1 }}>
                <div style={{ 
                  fontSize: '0.95rem',
                  marginBottom: '0.25rem',
                  lineHeight: 1.4
                }}>
                  <span
                    style={{
                      fontWeight: '600',
                      color: '#2c3e50',
                      cursor: 'pointer',
                      textDecoration: 'underline',
                      textDecorationColor: 'transparent',
                      transition: 'text-decoration-color 0.2s ease'
                    }}
                    onClick={() => handleUserClick(activity.from)}
                    onMouseEnter={(e) => {
                      e.target.style.textDecorationColor = '#667eea'
                    }}
                    onMouseLeave={(e) => {
                      e.target.style.textDecorationColor = 'transparent'
                    }}
                  >
                    {activity.from}
                  </span>
                  <span style={{ color: '#666', margin: '0 0.5rem' }}>gave</span>
                  <span
                    style={{
                      fontWeight: '600',
                      color: '#2c3e50',
                      cursor: 'pointer',
                      textDecoration: 'underline',
                      textDecorationColor: 'transparent',
                      transition: 'text-decoration-color 0.2s ease'
                    }}
                    onClick={() => handleUserClick(activity.to)}
                    onMouseEnter={(e) => {
                      e.target.style.textDecorationColor = '#667eea'
                    }}
                    onMouseLeave={(e) => {
                      e.target.style.textDecorationColor = 'transparent'
                    }}
                  >
                    {activity.to}
                  </span>
                  <span
                    style={{
                      color: activity.points > 0 ? '#27ae60' : '#e74c3c',
                      fontWeight: '700',
                      margin: '0 0.5rem'
                    }}
                  >
                    {activity.points > 0 ? '+' : ''}{activity.points}
                  </span>
                  <span style={{ color: '#666' }}>points</span>
                </div>

                {activity.reason && (
                  <div style={{
                    color: '#666',
                    fontSize: '0.8rem',
                    marginBottom: '0.25rem',
                    fontStyle: 'italic'
                  }}>
                    {activity.transactionType === 'emoji' && activity.emojiName
                      ? `Added a ${getEmojiFromName(activity.emojiName)} emoji reaction`
                      : activity.reason
                    }
                  </div>
                )}

                <div style={{
                  color: '#999',
                  fontSize: '0.75rem',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.5rem'
                }}>
                  <span>🕐</span>
                  {new Date(activity.date).toLocaleString()}
                </div>
              </div>

              {/* Points Badge */}
              <div style={{
                background: activity.points > 0 ? '#e8f5e8' : '#fde8e8',
                color: activity.points > 0 ? '#27ae60' : '#e74c3c',
                padding: '0.25rem 0.75rem',
                borderRadius: '15px',
                fontSize: '0.8rem',
                fontWeight: '600',
                flexShrink: 0
              }}>
                {activity.points > 0 ? '+' : ''}{activity.points}
              </div>
            </div>
          ))}
        </div>

        <div style={{
          marginTop: '1rem',
          padding: '0.75rem',
          background: '#f8f9fa',
          borderRadius: '8px',
          fontSize: '0.8rem',
          color: '#666',
          textAlign: 'center'
        }}>
          💡 Click on usernames to view their detailed statistics
        </div>

        {/* Pagination Controls */}
        {renderPaginationControls()}
      </div>

      <style>
        {`
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
        `}
      </style>
    </div>
  )
}

export default ActivityPage