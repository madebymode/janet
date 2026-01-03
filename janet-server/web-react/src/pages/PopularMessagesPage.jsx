import React from 'react'
import { useApi } from '../hooks/useApi'

function PopularMessagesPage({ selectedYear, selectedUser, onUserChange }) {
  const { data: leaderboard } = useApi(
    selectedYear === 0
      ? '/api/leaderboard?limit=100'
      : `/api/leaderboard?limit=100&year=${selectedYear}`
  )

  const { data: messages, loading, error } = useApi(
    selectedYear !== null
      ? `/api/stats/popular-messages?limit=200&year=${selectedYear === 0 ? '' : selectedYear}`
      : null,
    [selectedYear]
  )

  const filteredMessages = React.useMemo(() => {
    if (!messages || !selectedUser) return messages
    return messages.filter((message) => message.author_name === selectedUser)
  }, [messages, selectedUser])

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '3rem' }}>
        <div className="loading-spinner" style={{
          width: '50px',
          height: '50px',
          border: '4px solid rgba(102, 126, 234, 0.2)',
          borderTop: '4px solid #667eea',
          borderRadius: '50%',
          animation: 'spin 1s linear infinite',
          margin: '0 auto 1rem'
        }}></div>
        <p style={{ color: '#666', fontWeight: '600', marginBottom: '0.5rem' }}>
          Loading popular messages...
        </p>
        <p style={{ color: '#999', fontSize: '0.9rem', maxWidth: '500px', margin: '0 auto' }}>
          This may take 30-60 seconds on first load while fetching message details from Slack.
          <br />
          Subsequent loads will be much faster thanks to caching.
        </p>
      </div>
    )
  }

  if (error) {
    return (
      <div style={{
        background: '#fee',
        border: '1px solid #fcc',
        borderRadius: '8px',
        padding: '1rem',
        margin: '1rem 0'
      }}>
        <p style={{ color: '#c33', margin: 0 }}>
          ⚠️ Error loading popular messages: {error.message || 'Unknown error'}
        </p>
      </div>
    )
  }

  if (!messages || messages.length === 0) {
    return (
      <div style={{
        textAlign: 'center',
        padding: '3rem',
        background: '#f9f9f9',
        borderRadius: '12px',
        margin: '1rem 0'
      }}>
        <p style={{ fontSize: '3rem', margin: '0 0 1rem 0' }}>📭</p>
        <p style={{ color: '#666', margin: 0 }}>No popular messages found for this period</p>
      </div>
    )
  }

  if (filteredMessages && filteredMessages.length === 0) {
    return (
      <div style={{
        textAlign: 'center',
        padding: '3rem',
        background: '#f9f9f9',
        borderRadius: '12px',
        margin: '1rem 0'
      }}>
        <p style={{ fontSize: '3rem', margin: '0 0 1rem 0' }}>🧵</p>
        <p style={{ color: '#666', margin: 0 }}>
          No popular messages for {selectedUser ? `@${selectedUser}` : 'this period'}
        </p>
      </div>
    )
  }

  return (
    <div className="popular-messages-page">
      <div style={{
        marginBottom: '2rem',
        padding: '1.5rem',
        background: 'linear-gradient(135deg, #667eea15 0%, #764ba215 100%)',
        borderRadius: '12px',
        border: '1px solid #667eea30'
      }}>
        <h2 style={{
          margin: '0 0 0.5rem 0',
          fontSize: '1.5rem',
          color: '#333',
          display: 'flex',
          alignItems: 'center',
          gap: '0.5rem'
        }}>
          🔥 Most Popular Messages
        </h2>
        <p style={{ margin: 0, color: '#666' }}>
          Messages with the most reactions and karma points
        </p>
      </div>

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
          <span>🔎</span>
          Filter by User
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
              <option value="">All users</option>
              {leaderboard?.users?.map(user => (
                <option key={user.username} value={user.username}>
                  @{user.username} ({(user.total_points || 0).toLocaleString()} points)
                </option>
              ))}
            </select>
          </div>
          {selectedUser && (
            <div style={{
              background: '#eef2ff',
              color: '#3f51b5',
              padding: '0.5rem 1rem',
              borderRadius: '20px',
              fontSize: '0.875rem',
              fontWeight: '600',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem'
            }}>
              <span>✅</span>
              Filtering: @{selectedUser}
            </div>
          )}
        </div>
      </div>

      <div style={{
        display: 'grid',
        gap: '1rem'
      }}>
        {filteredMessages.map((message, index) => (
          <div
            key={`${message.channel_id}-${message.message_id}`}
            style={{
              background: 'white',
              borderRadius: '12px',
              padding: '1.5rem',
              boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
              border: '1px solid #eee',
              transition: 'all 0.3s ease',
              cursor: message.permalink ? 'pointer' : 'default'
            }}
            onClick={() => {
              if (message.permalink) {
                window.open(message.permalink, '_blank', 'noopener,noreferrer')
              }
            }}
            onMouseEnter={(e) => {
              if (message.permalink) {
                e.currentTarget.style.transform = 'translateY(-2px)'
                e.currentTarget.style.boxShadow = '0 4px 16px rgba(0,0,0,0.15)'
              }
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = 'translateY(0)'
              e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)'
            }}
          >
            <div style={{
              display: 'flex',
              alignItems: 'flex-start',
              justifyContent: 'space-between',
              gap: '1rem'
            }}>
              <div style={{ flex: 1 }}>
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.75rem',
                  marginBottom: '0.75rem'
                }}>
                  <span style={{
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    color: 'white',
                    fontWeight: '700',
                    fontSize: '1.1rem',
                    padding: '0.25rem 0.75rem',
                    borderRadius: '20px',
                    minWidth: '2rem',
                    textAlign: 'center'
                  }}>
                    #{index + 1}
                  </span>
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '1rem',
                    flexWrap: 'wrap'
                  }}>
                    <div style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.5rem',
                      padding: '0.4rem 0.8rem',
                      background: '#f0f0f0',
                      borderRadius: '20px'
                    }}>
                      <span style={{ fontSize: '1.2rem' }}>👍</span>
                      <span style={{ fontWeight: '600', color: '#333' }}>
                        {message.reaction_count} reaction{message.reaction_count !== 1 ? 's' : ''}
                      </span>
                    </div>
                    <div style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.5rem',
                      padding: '0.4rem 0.8rem',
                      background: message.total_points >= 0 ? '#e8f5e9' : '#ffebee',
                      borderRadius: '20px'
                    }}>
                      <span style={{ fontSize: '1.2rem' }}>
                        {message.total_points >= 0 ? '✨' : '💔'}
                      </span>
                      <span style={{
                        fontWeight: '600',
                        color: message.total_points >= 0 ? '#2e7d32' : '#c62828'
                      }}>
                        {message.total_points >= 0 ? '+' : ''}{message.total_points} points
                      </span>
                    </div>
                  </div>
                </div>

                {(message.author_name || message.author_avatar) && (
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '0.5rem',
                    marginBottom: '0.75rem',
                    color: '#555'
                  }}>
                    {message.author_avatar && (
                      <img
                        src={message.author_avatar}
                        alt={message.author_name || 'Slack user'}
                        style={{
                          width: '28px',
                          height: '28px',
                          borderRadius: '50%',
                          objectFit: 'cover'
                        }}
                      />
                    )}
                    {message.author_name && (
                      <span style={{ fontWeight: '600' }}>
                        @{message.author_name}
                      </span>
                    )}
                  </div>
                )}

                {message.text && (
                  <div style={{
                    background: '#f8f9fa',
                    padding: '1rem',
                    borderRadius: '8px',
                    marginBottom: '0.75rem',
                    fontStyle: 'italic',
                    color: '#555',
                    lineHeight: '1.5',
                    borderLeft: '3px solid #667eea',
                    overflowWrap: 'anywhere',
                    wordBreak: 'break-word',
                    whiteSpace: 'pre-wrap'
                  }}>
                    "{message.text}"
                  </div>
                )}

                {message.image_url && (
                  <div style={{
                    marginBottom: '0.75rem',
                    background: '#f8f9fa',
                    borderRadius: '10px',
                    border: '1px solid #eee',
                    padding: '0.5rem'
                  }}>
                    <img
                      src={message.image_url}
                      alt="Message attachment"
                      style={{
                        width: '100%',
                        maxHeight: '360px',
                        objectFit: 'contain',
                        borderRadius: '8px',
                        display: 'block'
                      }}
                      loading="lazy"
                    />
                  </div>
                )}

                {message.permalink && (
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '0.5rem',
                    color: '#667eea',
                    fontSize: '0.9rem',
                    fontWeight: '500'
                  }}>
                    <span>🔗</span>
                    <span>Click to view message in Slack</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default PopularMessagesPage
