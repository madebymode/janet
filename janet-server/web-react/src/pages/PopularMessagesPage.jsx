import React from 'react'
import { useSearchParams } from 'react-router-dom'
import { useApi } from '../hooks/useApi'

function LazyMedia({ as, src, mime, alt, className, style }) {
  const [isVisible, setIsVisible] = React.useState(false)
  const ref = React.useRef(null)

  React.useEffect(() => {
    const node = ref.current
    if (!node) return undefined

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setIsVisible(true)
          observer.disconnect()
        }
      },
      { rootMargin: '200px 0px' }
    )

    observer.observe(node)
    return () => observer.disconnect()
  }, [])

  if (as === 'img') {
    return (
      <img
        ref={ref}
        src={isVisible ? src : undefined}
        alt={alt}
        className={className}
        style={style}
        loading="lazy"
      />
    )
  }

  if (as === 'video') {
    return (
      <video
        ref={ref}
        controls
        preload={isVisible ? 'metadata' : 'none'}
        src={isVisible ? src : undefined}
        className={className}
        style={style}
      />
    )
  }

  if (as === 'audio') {
    return (
      <audio
        ref={ref}
        controls
        preload={isVisible ? 'metadata' : 'none'}
        src={isVisible ? src : undefined}
        className={className}
        style={style}
      />
    )
  }

  return null
}

function PopularMessagesPage({ selectedYear, selectedUser, onUserChange }) {
  const [searchParams, setSearchParams] = useSearchParams()
  const page = Math.max(0, parseInt(searchParams.get('page') || '0', 10) || 0)
  const minReactions = Math.max(0, parseInt(searchParams.get('min_reactions') || '0', 10) || 0)
  const [minReactionsInput, setMinReactionsInput] = React.useState(minReactions > 0 ? String(minReactions) : '')
  const mediaOnly = searchParams.get('media') === '1'
  const userFilter = searchParams.get('user') || ''
  const pageSize = 15

  React.useEffect(() => {
    if (userFilter !== selectedUser) {
      onUserChange(userFilter)
    }
  }, [userFilter, selectedUser, onUserChange])

  React.useEffect(() => {
    setMinReactionsInput(minReactions > 0 ? String(minReactions) : '')
  }, [minReactions])

  const updateSearchParams = (updates) => {
    const next = new URLSearchParams(searchParams)
    Object.entries(updates).forEach(([key, value]) => {
      if (value === null || value === '' || value === false || value === 0) {
        next.delete(key)
      } else {
        next.set(key, String(value))
      }
    })
    setSearchParams(next)
  }

  const handleUserFilterChange = (value) => {
    updateSearchParams({ user: value || null, page: 0 })
    onUserChange(value)
  }

  const handleMinReactionsChange = (value) => {
    updateSearchParams({ min_reactions: value > 0 ? value : null, page: 0 })
  }

  const commitMinReactions = () => {
    const nextValue = parseInt(minReactionsInput, 10)
    handleMinReactionsChange(Number.isFinite(nextValue) ? nextValue : 0)
  }

  const handleMediaOnlyChange = (value) => {
    updateSearchParams({ media: value ? 1 : null, page: 0 })
  }

  const { data: leaderboard } = useApi(
    selectedYear === 0
      ? '/api/leaderboard?limit=100'
      : `/api/leaderboard?limit=100&year=${selectedYear}`
  )

  const { data: messages, loading, error } = useApi(
    selectedYear !== null
      ? `/api/stats/popular-messages?limit=${pageSize}&offset=${page * pageSize}&year=${selectedYear === 0 ? '' : selectedYear}&include_meta=1${userFilter ? `&user=${userFilter}` : ''}${minReactions > 0 ? `&min_reactions=${minReactions}` : ''}${mediaOnly ? '&has_media=1' : ''}`
      : null,
    [selectedYear, userFilter, page, minReactions, mediaOnly]
  )

  const list = messages?.items || messages
  const totalCount = messages?.total || (list ? list.length : 0)
  const pendingCount = messages?.pending || 0
  const queueSize = messages?.queue_size || 0
  const queuePosition = messages?.queue_position || 0
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize))

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

  if (!list || list.length === 0) {
    const filterLabel = userFilter ? `@${userFilter}` : 'this period'
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
          No popular messages for {filterLabel}
        </p>
      </div>
    )
  }

  const paginationControls = (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      marginBottom: '1rem',
      flexWrap: 'wrap',
      gap: '0.75rem'
    }}>
      <div style={{ color: '#666', fontWeight: '500' }}>
        Showing {list.length} of {totalCount.toLocaleString()} messages
      </div>
      <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
        <button
          onClick={() => updateSearchParams({ page: Math.max(0, page - 1) })}
          disabled={page === 0}
          style={{
            padding: '0.5rem 0.9rem',
            borderRadius: '8px',
            border: '1px solid #e1e8f7',
            background: page === 0 ? '#f3f4f6' : '#fff',
            color: '#334155',
            cursor: page === 0 ? 'not-allowed' : 'pointer',
            fontWeight: '600'
          }}
        >
          ← Prev
        </button>
        <span style={{ fontWeight: '600', color: '#334155' }}>
          Page {page + 1} / {totalPages}
        </span>
        <button
          onClick={() => updateSearchParams({ page: Math.min(totalPages - 1, page + 1) })}
          disabled={page >= totalPages - 1}
          style={{
            padding: '0.5rem 0.9rem',
            borderRadius: '8px',
            border: '1px solid #e1e8f7',
            background: page >= totalPages - 1 ? '#f3f4f6' : '#fff',
            color: '#334155',
            cursor: page >= totalPages - 1 ? 'not-allowed' : 'pointer',
            fontWeight: '600'
          }}
        >
          Next →
        </button>
      </div>
    </div>
  )

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

      {(pendingCount > 0 || queueSize > 0) && (
        <div style={{
          marginBottom: '1rem',
          padding: '0.9rem 1.2rem',
          background: '#fff8e1',
          borderRadius: '10px',
          border: '1px solid #ffe082',
          color: '#8a6d3b',
          display: 'flex',
          alignItems: 'center',
          gap: '0.6rem',
          fontWeight: '600'
        }}>
          <span>⏳</span>
          {pendingCount > 0 && (
            <span>{pendingCount} messages for this report are still fetching details.</span>
          )}
          {queueSize > 0 && (
            <span>Queue size: {queueSize}.</span>
          )}
          <span>Refresh in a minute for richer results.</span>
        </div>
      )}

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
              value={userFilter}
              onChange={(e) => handleUserFilterChange(e.target.value)}
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
          <div style={{ minWidth: '140px' }}>
            <input
              type="number"
              min="0"
              value={minReactionsInput}
              onChange={(e) => setMinReactionsInput(e.target.value)}
              onBlur={commitMinReactions}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  commitMinReactions()
                }
              }}
              placeholder="Min reactions"
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
            />
          </div>
          <label style={{
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem',
            fontSize: '0.95rem',
            color: '#2c3e50',
            fontWeight: '600'
          }}>
            <input
              type="checkbox"
              checked={mediaOnly}
              onChange={(e) => handleMediaOnlyChange(e.target.checked)}
            />
            Media only
          </label>
          {userFilter && (
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
              Filtering: @{userFilter}
            </div>
          )}
          {minReactions > 0 && (
            <div style={{
              background: '#fff3cd',
              color: '#856404',
              padding: '0.5rem 1rem',
              borderRadius: '20px',
              fontSize: '0.875rem',
              fontWeight: '600'
            }}>
              Min reactions: {minReactions}
            </div>
          )}
          {mediaOnly && (
            <div style={{
              background: '#e8f5e9',
              color: '#2e7d32',
              padding: '0.5rem 1rem',
              borderRadius: '20px',
              fontSize: '0.875rem',
              fontWeight: '600'
            }}>
              Media only
            </div>
          )}
        </div>
      </div>

      {paginationControls}

      <div style={{
        display: 'grid',
        gap: '1rem'
      }}>
        {list.map((message, index) => (
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
                    {message.pending_details && (
                      <span style={{
                        marginLeft: '0.5rem',
                        padding: '0.2rem 0.6rem',
                        borderRadius: '12px',
                        background: '#fff8e1',
                        border: '1px solid #ffe082',
                        color: '#8a6d3b',
                        fontSize: '0.8rem',
                        fontWeight: '600'
                      }}>
                        Details loading{message.queue_position ? ` (queue: ${message.queue_position})` : ''}
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
                    <LazyMedia
                      as="img"
                      src={message.image_url}
                      alt="Message attachment"
                      style={{
                        width: '100%',
                        maxHeight: '360px',
                        objectFit: 'contain',
                        borderRadius: '8px',
                        display: 'block'
                      }}
                    />
                  </div>
                )}

                {message.attachment_url && (
                  <div style={{
                    marginBottom: '0.75rem',
                    background: '#f8f9fa',
                    borderRadius: '10px',
                    border: '1px solid #eee',
                    padding: '0.75rem'
                  }}>
                    {message.attachment_mime?.startsWith('video/') ? (
                      <LazyMedia
                        as="video"
                        src={message.attachment_url}
                        mime={message.attachment_mime}
                        style={{
                          width: '100%',
                          maxHeight: '360px',
                          borderRadius: '8px',
                          display: 'block'
                        }}
                      />
                    ) : message.attachment_mime?.startsWith('audio/') ? (
                      <LazyMedia
                        as="audio"
                        src={message.attachment_url}
                        mime={message.attachment_mime}
                        style={{ width: '100%' }}
                      />
                    ) : (
                      <a
                        href={message.attachment_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ color: '#667eea', fontWeight: '600' }}
                      >
                        Download attachment
                      </a>
                    )}
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

      <div style={{ marginTop: '1.5rem' }}>
        {paginationControls}
      </div>
    </div>
  )
}

export default PopularMessagesPage
