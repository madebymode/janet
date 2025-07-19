import React from 'react'

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

function BarChart({ data, title, xKey, yKey }) {
  if (!data || data.length === 0) {
    return (
      <div style={{
        height: '300px',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#999',
        fontSize: '1rem'
      }}>
        <div style={{ fontSize: '3rem', marginBottom: '0.5rem' }}>📊</div>
        <div>No data available</div>
      </div>
    )
  }

  const maxY = Math.max(...data.map(d => d[yKey]))
  const width = 600
  const height = 350
  const padding = 60
  const barWidth = Math.max(30, (width - padding * 2) / data.length * 0.7)

  const colors = [
    '#667eea', '#764ba2', '#f093fb', '#f5576c', '#4facfe', 
    '#00f2fe', '#a8edea', '#fed6e3', '#ffecd2', '#fcb69f'
  ]

  return (
    <div>
      <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{ 
        maxWidth: '100%', 
        height: 'auto',
        background: 'white',
        borderRadius: '8px'
      }}>
        <defs>
          {data.map((_, index) => (
            <linearGradient key={index} id={`barGradient-${index}`} x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" stopColor={colors[index % colors.length]} stopOpacity="0.9" />
              <stop offset="100%" stopColor={colors[index % colors.length]} stopOpacity="0.7" />
            </linearGradient>
          ))}
          <filter id="barShadow">
            <feDropShadow dx="0" dy="2" stdDeviation="2" floodOpacity="0.2"/>
          </filter>
        </defs>
        
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
            x={padding - 15}
            y={padding + ratio * (height - padding * 2) + 5}
            textAnchor="end"
            fontSize="12"
            fill="#666"
            fontWeight="500"
          >
            {Math.round(maxY - ratio * maxY).toLocaleString()}
          </text>
        ))}

        {/* Bars */}
        {data.map((d, i) => {
          const x = padding + (i / data.length) * (width - padding * 2) + 
                   (width - padding * 2) / data.length * 0.15
          const barHeight = (d[yKey] / maxY) * (height - padding * 2)
          const y = height - padding - barHeight
          
          return (
            <g key={i}>
              <rect
                x={x}
                y={y}
                width={barWidth}
                height={barHeight}
                fill={`url(#barGradient-${i})`}
                rx="3"
                ry="3"
                filter="url(#barShadow)"
                style={{
                  cursor: 'pointer',
                  transition: 'transform 0.2s ease'
                }}
                onMouseEnter={(e) => {
                  e.target.style.transform = 'scaleY(1.05)'
                  e.target.style.transformOrigin = 'bottom'
                }}
                onMouseLeave={(e) => {
                  e.target.style.transform = 'scaleY(1)'
                }}
                onClick={() => {
                  alert(`${getEmojiFromName(d[xKey])}: ${d[yKey].toLocaleString()} uses`)
                }}
              />
              
              {/* Value label on top of bar */}
              <text
                x={x + barWidth / 2}
                y={y - 8}
                textAnchor="middle"
                fontSize="11"
                fontWeight="600"
                fill="#666"
              >
                {d[yKey].toLocaleString()}
              </text>
              
              {/* Emoji label at bottom */}
              <text
                x={x + barWidth / 2}
                y={height - 15}
                textAnchor="middle"
                fontSize="20"
                fill="#333"
              >
                {getEmojiFromName(d[xKey])}
              </text>
              
              {/* Emoji name label */}
              <text
                x={x + barWidth / 2}
                y={height - 35}
                textAnchor="middle"
                fontSize="10"
                fill="#999"
                fontWeight="500"
              >
                {d[xKey]}
              </text>
            </g>
          )
        })}

        {/* Axes */}
        <line
          x1={padding}
          y1={height - padding}
          x2={width - padding}
          y2={height - padding}
          stroke="#ddd"
          strokeWidth="2"
        />
        <line
          x1={padding}
          y1={padding}
          x2={padding}
          y2={height - padding}
          stroke="#ddd"
          strokeWidth="2"
        />
      </svg>
    </div>
  )
}

export default BarChart