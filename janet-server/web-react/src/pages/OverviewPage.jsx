import React from 'react'
import { useApi } from '../hooks/useApi'
import StatCard from '../components/StatCard'
import PieChart from '../components/charts/PieChart'
import LineChart from '../components/charts/LineChart'

function MiniMessageCard({ message, index }) {
  const formatTs = (id) => {
    const s = Number.parseFloat(id)
    if (!Number.isFinite(s)) return ''
    const d = new Date(s * 1000)
    if (Number.isNaN(d.getTime())) return ''
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  }

  return (
    <div
      onClick={() => message.permalink && window.open(message.permalink, '_blank', 'noopener,noreferrer')}
      style={{
        background: 'white',
        borderRadius: '10px',
        padding: '1rem',
        border: '1px solid #e8ecf7',
        boxShadow: '0 2px 8px rgba(102,126,234,0.08)',
        cursor: message.permalink ? 'pointer' : 'default',
        transition: 'transform 0.15s, box-shadow 0.15s',
        display: 'flex',
        flexDirection: 'column',
        gap: '0.5rem'
      }}
      onMouseEnter={(e) => { if (message.permalink) { e.currentTarget.style.transform = 'translateY(-2px)'; e.currentTarget.style.boxShadow = '0 6px 16px rgba(102,126,234,0.18)' } }}
      onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = '0 2px 8px rgba(102,126,234,0.08)' }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
        <span style={{
          background: 'linear-gradient(135deg, #ff6b6b 0%, #ffa07a 100%)',
          color: 'white',
          fontSize: '0.75rem',
          fontWeight: '700',
          padding: '0.15rem 0.5rem',
          borderRadius: '12px'
        }}>#{index + 1}</span>
        <span style={{ fontSize: '0.8rem', color: '#888', marginLeft: 'auto' }}>
          {formatTs(message.message_id)}
        </span>
      </div>
      {message.author_name && (
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
          {message.author_avatar && (
            <img src={message.author_avatar} alt={message.author_name} style={{ width: '20px', height: '20px', borderRadius: '50%' }} />
          )}
          <span style={{ fontWeight: '600', fontSize: '0.85rem', color: '#444' }}>@{message.author_name}</span>
        </div>
      )}
      {message.text && (
        <div style={{
          fontSize: '0.875rem',
          color: '#555',
          lineHeight: '1.4',
          overflow: 'hidden',
          display: '-webkit-box',
          WebkitLineClamp: 3,
          WebkitBoxOrient: 'vertical',
          fontStyle: 'italic'
        }}>
          "{message.text.length > 120 ? message.text.slice(0, 120) + '…' : message.text}"
        </div>
      )}
      {(message.image_url || message.attachment_url) && (
        <div style={{ fontSize: '0.8rem', color: '#667eea', fontWeight: '600' }}>📎 Has media</div>
      )}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: '0.4rem',
        background: '#fff3f3',
        padding: '0.25rem 0.6rem',
        borderRadius: '8px',
        width: 'fit-content'
      }}>
        <span>😂</span>
        <span style={{ fontWeight: '700', fontSize: '0.85rem', color: '#e74c3c' }}>
          {message.reaction_count} reaction{message.reaction_count !== 1 ? 's' : ''}
        </span>
      </div>
    </div>
  )
}

function SpotlightCard({ icon, label, children, accent }) {
  return (
    <div style={{
      background: 'white',
      borderRadius: '14px',
      padding: '1.25rem 1.5rem',
      border: `1px solid ${accent}30`,
      boxShadow: `0 4px 16px ${accent}12`,
      display: 'flex',
      flexDirection: 'column',
      gap: '0.5rem',
      flex: '1 1 260px'
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem' }}>
        <span style={{ fontSize: '1.3rem' }}>{icon}</span>
        <span style={{ fontSize: '0.8rem', fontWeight: '700', color: accent, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{label}</span>
      </div>
      {children}
    </div>
  )
}

function OverviewPage({ selectedYear, selectedMonth = 0 }) {
  const monthParam = selectedMonth > 0 ? `&month=${selectedMonth}` : ''
  const yearParam = selectedYear === 0 ? '' : `year=${selectedYear}`
  const statsUrl = selectedYear === 0 && !selectedMonth
    ? '/api/stats/detailed'
    : `/api/stats/detailed?${yearParam}${monthParam}`

  const { data: stats, loading: statsLoading, error: statsError } = useApi(statsUrl)
  const { data: status, loading: statusLoading } = useApi('/api/status')
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

  const highlightYear = selectedYear === 0 ? new Date().getFullYear() : selectedYear
  const { data: funnyHighlights, loading: highlightsLoading } = useApi(
    `/api/stats/popular-messages?limit=4&funny_bias=1&year=${highlightYear}${monthParam}&include_meta=1`
  )

  const { data: topChannelsData, loading: channelsLoading } = useApi(
    selectedYear === 0 && !selectedMonth
      ? '/api/stats/top-channels?limit=8'
      : `/api/stats/top-channels?limit=8&${yearParam}${monthParam}`
  )

  const { data: funFactsData } = useApi(
    selectedYear === 0 && !selectedMonth
      ? '/api/stats/fun-facts'
      : `/api/stats/fun-facts?${yearParam}${monthParam}`
  )

  const { data: miniLeaderboard, loading: miniLbLoading } = useApi(
    selectedYear === 0 && !selectedMonth
      ? '/api/leaderboard?limit=5'
      : `/api/leaderboard/${selectedYear === 0 ? new Date().getFullYear() : selectedYear}?limit=5${monthParam}`
  )

  const positivityRate = stats && stats.totalTransactions > 0
    ? Math.round((stats.positiveTransactions / stats.totalTransactions) * 100)
    : null
  const { data: colludersData, loading: colludersLoading } = useApi(
    `/api/stats/colluders?limit=5${selectedYear !== 0 ? `&year=${selectedYear}` : ''}${monthParam}`
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

  const DOW_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
  const formatHour = (h) => {
    if (h === 0) return '12 AM'
    if (h < 12) return `${h} AM`
    if (h === 12) return '12 PM'
    return `${h - 12} PM`
  }

  const secondaryFacts = React.useMemo(() => {
    const facts = []
    if (funFactsData?.most_active_channel) {
      facts.push({ icon: '📣', text: `#${funFactsData.most_active_channel} is the most karma-active channel with ${funFactsData.most_active_channel_count.toLocaleString()} transactions`, color: '#2980b9' })
    }
    if (funFactsData?.total_unique_channels > 1) {
      facts.push({ icon: '🌐', text: `Karma has spread across ${funFactsData.total_unique_channels} different channels`, color: '#8e44ad' })
    }
    if (stats?.avgPointsPerUser) {
      facts.push({ icon: '⚡', text: `On average, each member has earned ${Math.round(stats.avgPointsPerUser).toLocaleString()} karma points`, color: '#667eea' })
    }
    if (stats?.totalPoints > 1000) {
      facts.push({ icon: '🚀', text: `The team has collectively given ${stats.totalPoints.toLocaleString()} karma points`, color: '#e74c3c' })
    }
    return facts
  }, [stats, funFactsData])

  const highlightList = funnyHighlights?.items || funnyHighlights || []
  const topColluder = colludersData?.colluders?.[0]
  const biggestEvent = funFactsData?.biggest_karma_event

  return (
    <div className="overview-page">
      {/* Hero Section */}
      <div style={{
        background: 'linear-gradient(135deg, #f8f9ff 0%, #e8f0ff 100%)',
        borderRadius: '16px',
        padding: '2rem',
        marginBottom: '1.5rem',
        border: '1px solid #e1e8f7'
      }}>
        <div style={{ textAlign: 'center' }}>
          <h1 style={{
            fontSize: '2.5rem',
            margin: '0 0 0.5rem 0',
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            fontWeight: '700'
          }}>
            Welcome to Karma Dashboard
          </h1>
          <p style={{
            fontSize: '1.1rem',
            color: '#666',
            margin: 0,
            maxWidth: '600px',
            marginLeft: 'auto',
            marginRight: 'auto'
          }}>
            Track team karma, celebrate achievements, and build a positive community culture.
            Currently viewing <strong>{getYearLabel()}</strong> data.
          </p>
        </div>
      </div>

      {/* Top Spotlight Row — most interesting facts first */}
      <div style={{
        display: 'flex',
        gap: '1.25rem',
        flexWrap: 'wrap',
        marginBottom: '1.5rem'
      }}>
        {biggestEvent && (
          <SpotlightCard icon="🏆" label="Biggest Single Karma Event" accent="#e67e22">
            <div style={{ fontSize: '1.6rem', fontWeight: '800', color: '#e67e22', lineHeight: 1 }}>
              +{biggestEvent.points.toLocaleString()} pts
            </div>
            <div style={{ fontSize: '0.9rem', color: '#444', lineHeight: '1.4' }}>
              <strong>@{biggestEvent.from_user}</strong> → <strong>@{biggestEvent.to_user}</strong>
            </div>
            {biggestEvent.reason && (
              <div style={{
                fontSize: '0.82rem',
                color: '#666',
                fontStyle: 'italic',
                lineHeight: '1.35',
                borderTop: '1px solid #f0e0cc',
                paddingTop: '0.4rem',
                marginTop: '0.1rem'
              }}>
                "{biggestEvent.reason.length > 100 ? biggestEvent.reason.slice(0, 100) + '…' : biggestEvent.reason}"
              </div>
            )}
          </SpotlightCard>
        )}

        {topColluder && (
          <SpotlightCard icon="🤝" label="Top Karma Colluders" accent="#667eea">
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
              {topColluder.user1_avatar && (
                <img src={topColluder.user1_avatar} alt="" style={{ width: '28px', height: '28px', borderRadius: '50%' }} />
              )}
              <span style={{ fontWeight: '700', color: '#1a237e', fontSize: '0.95rem' }}>
                {topColluder.user1_display_name || `@${topColluder.user1}`}
              </span>
              <span style={{ color: '#999', fontSize: '0.9rem' }}>⇄</span>
              {topColluder.user2_avatar && (
                <img src={topColluder.user2_avatar} alt="" style={{ width: '28px', height: '28px', borderRadius: '50%' }} />
              )}
              <span style={{ fontWeight: '700', color: '#1a237e', fontSize: '0.95rem' }}>
                {topColluder.user2_display_name || `@${topColluder.user2}`}
              </span>
            </div>
            <div style={{ fontSize: '1.4rem', fontWeight: '800', color: '#667eea', lineHeight: 1 }}>
              {topColluder.total_exchange.toLocaleString()} pts
            </div>
            <div style={{ fontSize: '0.82rem', color: '#666' }}>
              {topColluder.user1_to_user2.toLocaleString()} given · {topColluder.user2_to_user1.toLocaleString()} returned
            </div>
          </SpotlightCard>
        )}

        {funFactsData?.most_active_hour_count > 0 && (
          <SpotlightCard icon="⏰" label="Peak Karma Time" accent="#9b59b6">
            <div style={{ fontSize: '1.6rem', fontWeight: '800', color: '#9b59b6', lineHeight: 1 }}>
              {formatHour(funFactsData.most_active_hour)}
            </div>
            {funFactsData?.most_active_dow_count > 0 && (
              <div style={{ fontSize: '0.9rem', color: '#444' }}>
                on <strong>{DOW_NAMES[funFactsData.most_active_dow]}s</strong>
              </div>
            )}
            <div style={{ fontSize: '0.82rem', color: '#666' }}>
              {funFactsData.most_active_hour_count.toLocaleString()} transactions at this hour
            </div>
          </SpotlightCard>
        )}
      </div>

      {/* Secondary Fun Facts */}
      {secondaryFacts.length > 0 && (
        <div style={{
          background: 'linear-gradient(135deg, #fffef0 0%, #fff8e1 100%)',
          borderRadius: '14px',
          padding: '1.25rem 1.5rem',
          marginBottom: '1.5rem',
          border: '1px solid #ffe082'
        }}>
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
            gap: '0.65rem'
          }}>
            {secondaryFacts.map((fact, i) => (
              <div key={i} style={{
                background: 'white',
                borderRadius: '10px',
                padding: '0.75rem 1rem',
                border: `1px solid ${fact.color}30`,
                display: 'flex',
                alignItems: 'flex-start',
                gap: '0.6rem',
                boxShadow: '0 1px 4px rgba(0,0,0,0.04)'
              }}>
                <span style={{ fontSize: '1.2rem', flexShrink: 0 }}>{fact.icon}</span>
                <span style={{ fontSize: '0.875rem', color: '#444', lineHeight: '1.45' }}>{fact.text}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Key Statistics */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
        gap: '1.25rem',
        marginBottom: '1.25rem'
      }}>
        <StatCard
          icon="👥"
          title="Community Members"
          value={stats?.totalUsers || 0}
          loading={statsLoading}
          error={statsError}
          subtitle={`Active ${getYearLabel().toLowerCase()}`}
          color="#3498db"
        />
        <StatCard
          icon="⭐"
          title="Karma Points"
          value={stats?.totalPoints || 0}
          loading={statsLoading}
          error={statsError}
          subtitle={`Earned ${getYearLabel().toLowerCase()}`}
          color="#e74c3c"
        />
        <StatCard
          icon="🔄"
          title="Interactions"
          value={stats?.totalTransactions || 0}
          loading={statsLoading}
          error={statsError}
          subtitle={`Total ${getYearLabel().toLowerCase()}`}
          color="#9b59b6"
        />
        <StatCard
          icon="🤖"
          title="Janet Bot"
          value={status?.botOnline ? 'Online' : 'Offline'}
          loading={statusLoading}
          isStatus={true}
          subtitle="Current status"
          color={status?.botOnline ? '#27ae60' : '#e74c3c'}
        />
      </div>

      {/* Secondary Stats Row */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
        gap: '1.25rem',
        marginBottom: '2rem'
      }}>
        <StatCard
          icon="😊"
          title="Positivity Rate"
          value={positivityRate !== null ? `${positivityRate}%` : '—'}
          loading={statsLoading}
          subtitle="Good vibes ratio"
          color="#27ae60"
        />
        <StatCard
          icon="📊"
          title="Avg Points / Member"
          value={stats?.avgPointsPerUser ? Math.round(stats.avgPointsPerUser).toLocaleString() : '—'}
          loading={statsLoading}
          subtitle={`Per active member`}
          color="#f39c12"
        />
        <StatCard
          icon="👍"
          title="Positive Karma"
          value={stats?.positiveTransactions || 0}
          loading={statsLoading}
          subtitle="++ transactions"
          color="#27ae60"
        />
        <StatCard
          icon="👎"
          title="Negative Karma"
          value={stats?.negativeTransactions || 0}
          loading={statsLoading}
          subtitle="-- transactions"
          color="#e74c3c"
        />
      </div>

      {/* Mini Leaderboard */}
      <div style={{
        background: 'linear-gradient(135deg, #fffde7 0%, #fff8e1 100%)',
        borderRadius: '14px',
        padding: '1.5rem',
        marginBottom: '2rem',
        border: '1px solid #ffe082'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem', flexWrap: 'wrap', gap: '0.5rem' }}>
          <h3 style={{
            margin: 0,
            fontSize: '1.15rem',
            fontWeight: '700',
            color: '#e65100',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <span>🏆</span> Top 5 — {getYearLabel()}
          </h3>
          <a
            href={`/leaderboard/${selectedYear === 0 ? 'all' : selectedYear}${selectedMonth > 0 ? `?month=${selectedMonth}` : ''}`}
            style={{
              background: 'linear-gradient(135deg, #ff9800 0%, #f44336 100%)',
              color: 'white',
              padding: '0.4rem 1rem',
              borderRadius: '20px',
              fontSize: '0.85rem',
              fontWeight: '600',
              textDecoration: 'none',
              whiteSpace: 'nowrap'
            }}
          >
            Full Board →
          </a>
        </div>
        {miniLbLoading ? (
          <div style={{ color: '#e65100', fontSize: '0.95rem' }}>Loading...</div>
        ) : (miniLeaderboard?.users || []).slice(0, 5).map((user, i) => {
          const displayName = user.display_name || user.real_name || user.username
          const medals = ['🥇', '🥈', '🥉']
          return (
            <div key={user.username} style={{
              background: 'white',
              borderRadius: '10px',
              padding: '0.6rem 1rem',
              marginBottom: '0.5rem',
              border: '1px solid #ffe082',
              display: 'flex',
              alignItems: 'center',
              gap: '0.75rem'
            }}>
              <span style={{ fontSize: '1.1rem', flexShrink: 0 }}>{medals[i] || `#${i + 1}`}</span>
              {user.avatar_url && <img src={user.avatar_url} alt={displayName} style={{ width: '28px', height: '28px', borderRadius: '50%', flexShrink: 0 }} />}
              <span style={{ fontWeight: '700', color: '#e65100', flex: 1, fontSize: '0.9rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{displayName}</span>
              <span style={{ fontSize: '0.9rem', fontWeight: '700', color: '#f57c00', flexShrink: 0 }}>{Number(user.total_points || 0).toLocaleString()} pts</span>
            </div>
          )
        })}
      </div>

      {/* Recent Funny Highlights */}
      <div style={{
        background: 'linear-gradient(135deg, #fff0f5 0%, #fce4ec 100%)',
        borderRadius: '14px',
        padding: '1.5rem',
        marginBottom: '2rem',
        border: '1px solid #f48fb1'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem', flexWrap: 'wrap', gap: '0.5rem' }}>
          <h3 style={{
            margin: 0,
            fontSize: '1.15rem',
            fontWeight: '700',
            color: '#880e4f',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <span>😂</span> Recent Funny Highlights ({highlightYear})
          </h3>
          <a
            href={`/popular/${highlightYear}?funny_bias=1`}
            style={{
              background: 'linear-gradient(135deg, #e91e63 0%, #9c27b0 100%)',
              color: 'white',
              padding: '0.4rem 1rem',
              borderRadius: '20px',
              fontSize: '0.85rem',
              fontWeight: '600',
              textDecoration: 'none',
              whiteSpace: 'nowrap'
            }}
          >
            See All →
          </a>
        </div>
        {highlightsLoading ? (
          <div style={{ color: '#c2185b', fontSize: '0.95rem', padding: '1rem 0' }}>Loading highlights...</div>
        ) : highlightList.length > 0 ? (
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
            gap: '0.75rem'
          }}>
            {highlightList.slice(0, 4).map((msg, i) => (
              <MiniMessageCard key={`${msg.channel_id}-${msg.message_id}`} message={msg} index={i} />
            ))}
          </div>
        ) : (
          <div style={{ color: '#c2185b', fontSize: '0.95rem' }}>No highlights yet for this period.</div>
        )}
      </div>

      {/* Top Channels */}
      {(channelsLoading || (topChannelsData?.channels?.length > 0)) && (
        <div style={{
          background: 'linear-gradient(135deg, #f0fff4 0%, #e8f5e9 100%)',
          borderRadius: '14px',
          padding: '1.5rem',
          marginBottom: '2rem',
          border: '1px solid #a5d6a7'
        }}>
          <h3 style={{
            margin: '0 0 0.4rem 0',
            fontSize: '1.15rem',
            fontWeight: '700',
            color: '#1b5e20',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <span>📣</span> Most Active Channels
          </h3>
          <p style={{ margin: '0 0 1rem 0', fontSize: '0.85rem', color: '#388e3c' }}>
            Channels where karma is flowing most actively.
          </p>
          {channelsLoading ? (
            <div style={{ color: '#388e3c', fontSize: '0.95rem' }}>Loading...</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {topChannelsData.channels.map((ch, i) => {
                const maxCount = topChannelsData.channels[0]?.transaction_count || 1
                const pct = Math.round((ch.transaction_count / maxCount) * 100)
                const medals = ['🥇', '🥈', '🥉']
                return (
                  <div key={ch.channel_name} style={{
                    background: 'white',
                    borderRadius: '10px',
                    padding: '0.6rem 1rem',
                    border: '1px solid #c8e6c9',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '0.75rem'
                  }}>
                    <span style={{ fontSize: '1rem', flexShrink: 0 }}>{medals[i] || `#${i + 1}`}</span>
                    <span style={{ fontWeight: '700', color: '#1b5e20', fontSize: '0.95rem', minWidth: '120px' }}>#{ch.channel_name}</span>
                    <div style={{ flex: 1, background: '#e8f5e9', borderRadius: '4px', height: '8px', overflow: 'hidden' }}>
                      <div style={{ width: `${pct}%`, background: 'linear-gradient(90deg, #43a047, #66bb6a)', height: '100%', borderRadius: '4px' }} />
                    </div>
                    <span style={{ fontSize: '0.85rem', color: '#388e3c', fontWeight: '600', flexShrink: 0 }}>{ch.transaction_count.toLocaleString()} txns</span>
                    <span style={{ fontSize: '0.8rem', color: '#81c784', flexShrink: 0 }}>{ch.total_points.toLocaleString()} pts</span>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* All Colluders */}
      {(colludersLoading || (colludersData?.colluders?.length > 1)) && (
        <div style={{
          background: 'linear-gradient(135deg, #f0f4ff 0%, #e8eeff 100%)',
          borderRadius: '14px',
          padding: '1.5rem',
          marginBottom: '2rem',
          border: '1px solid #c5d0f5'
        }}>
          <h3 style={{
            margin: '0 0 0.4rem 0',
            fontSize: '1.15rem',
            fontWeight: '700',
            color: '#1a237e',
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}>
            <span>🤝</span> Top Colluders
          </h3>
          <p style={{ margin: '0 0 1rem 0', fontSize: '0.85rem', color: '#3949ab' }}>
            Pairs with the highest mutual karma exchange — they're clearly in cahoots.
          </p>
          {colludersLoading ? (
            <div style={{ color: '#3949ab', fontSize: '0.95rem' }}>Loading...</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
              {colludersData.colluders.map((c, i) => {
                const name1 = c.user1_display_name || `@${c.user1}`
                const name2 = c.user2_display_name || `@${c.user2}`
                const medals = ['🥇', '🥈', '🥉']
                return (
                  <div key={`${c.user1}-${c.user2}`} style={{
                    background: 'white',
                    borderRadius: '10px',
                    padding: '0.75rem 1rem',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '0.75rem',
                    flexWrap: 'wrap',
                    border: '1px solid #dde3f7',
                    boxShadow: '0 1px 4px rgba(0,0,0,0.04)'
                  }}>
                    <span style={{ fontSize: '1.2rem', flexShrink: 0 }}>{medals[i] || `#${i + 1}`}</span>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', flex: 1, flexWrap: 'wrap', minWidth: 0 }}>
                      {c.user1_avatar && <img src={c.user1_avatar} alt={name1} style={{ width: '24px', height: '24px', borderRadius: '50%' }} />}
                      <span style={{ fontWeight: '700', color: '#1a237e', fontSize: '0.9rem' }}>{name1}</span>
                      <span style={{ color: '#667eea', fontWeight: '600', fontSize: '0.85rem' }}>↕ {c.user1_to_user2.toLocaleString()} pts</span>
                      <span style={{ color: '#999', fontSize: '0.85rem' }}>⇄</span>
                      <span style={{ color: '#667eea', fontWeight: '600', fontSize: '0.85rem' }}>{c.user2_to_user1.toLocaleString()} pts ↕</span>
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
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* Charts Section */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
        gap: '2rem',
        marginBottom: '2rem'
      }}>
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
            <div style={{ height: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
              Loading chart...
            </div>
          ) : karmaDistribution ? (
            <PieChart data={karmaDistribution} />
          ) : (
            <div style={{ height: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
              No data available
            </div>
          )}
        </div>

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
            Points Over Time
          </h3>
          {pointsTimeLoading ? (
            <div style={{ height: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
              Loading chart...
            </div>
          ) : pointsOverTime ? (
            <LineChart data={pointsOverTime} xKey="month" yKey="totalPoints" />
          ) : (
            <div style={{ height: '300px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>
              No data available
            </div>
          )}
        </div>
      </div>

      {/* About Section */}
      <div className="card" style={{
        background: 'white',
        borderRadius: '12px',
        padding: '2rem',
        boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
        border: '1px solid #f0f0f0'
      }}>
        <h2 style={{ margin: '0 0 1rem 0', fontSize: '1.5rem', fontWeight: '600', color: '#2c3e50' }}>
          About Janet Karma System
        </h2>
        <p style={{ fontSize: '1rem', lineHeight: '1.6', color: '#666', marginBottom: '1.5rem' }}>
          Janet is a Slack bot that tracks karma points for team members, helping build a positive
          and collaborative workplace culture. Team members can recognize each other's contributions
          and achievements using simple commands.
        </p>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '1rem' }}>
          {[
            { command: '@username++', description: 'Give points', icon: '⬆️', color: '#27ae60' },
            { command: '@username--', description: 'Take points', icon: '⬇️', color: '#e74c3c' },
            { command: '@username?', description: 'Check points', icon: '❓', color: '#3498db' },
            { command: 'goodplace leaderboard', description: 'Show rankings', icon: '🏆', color: '#f39c12' }
          ].map((item, index) => (
            <div key={index} style={{
              padding: '1rem',
              background: `${item.color}10`,
              border: `1px solid ${item.color}30`,
              borderRadius: '8px',
              textAlign: 'center'
            }}>
              <div style={{ fontSize: '1.5rem', marginBottom: '0.5rem' }}>{item.icon}</div>
              <code style={{
                background: `${item.color}20`,
                padding: '0.25rem 0.5rem',
                borderRadius: '4px',
                fontSize: '0.875rem',
                fontWeight: '500',
                color: item.color,
                display: 'block',
                marginBottom: '0.5rem'
              }}>
                {item.command}
              </code>
              <div style={{ fontSize: '0.875rem', color: '#666' }}>{item.description}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default OverviewPage
