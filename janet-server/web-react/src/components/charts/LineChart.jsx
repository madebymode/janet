import React from 'react'

function LineChart({ data, title, xKey, yKey }) {
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
        <div style={{ fontSize: '3rem', marginBottom: '0.5rem' }}>📈</div>
        <div>No data available</div>
      </div>
    )
  }

  const maxY = Math.max(...data.map(d => d[yKey]))
  const minY = Math.min(...data.map(d => d[yKey]))
  const range = maxY - minY || 1

  const width = 500
  const height = 300
  const padding = 60

  const points = data.map((d, i) => {
    const x = padding + (i / (data.length - 1)) * (width - padding * 2)
    const y = height - padding - ((d[yKey] - minY) / range) * (height - padding * 2)
    return { x, y, value: d[yKey], label: d[xKey] }
  })

  const pathData = points.map((p, i) => 
    i === 0 ? `M ${p.x} ${p.y}` : `L ${p.x} ${p.y}`
  ).join(' ')

  const areaData = `M ${padding} ${height - padding} L ${pathData.substring(2)} L ${width - padding} ${height - padding} Z`

  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

  return (
    <div>
      <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} style={{ 
        maxWidth: '100%', 
        height: 'auto',
        background: 'white',
        borderRadius: '8px'
      }}>
        <defs>
          <linearGradient id="areaGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#667eea" stopOpacity="0.3" />
            <stop offset="100%" stopColor="#667eea" stopOpacity="0.05" />
          </linearGradient>
          <filter id="lineShadow">
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
            {Math.round(maxY - ratio * range).toLocaleString()}
          </text>
        ))}

        {/* X-axis labels */}
        {points.map((point, i) => (
          <text
            key={i}
            x={point.x}
            y={height - 15}
            textAnchor="middle"
            fontSize="12"
            fill="#666"
            fontWeight="500"
          >
            {typeof point.label === 'number' && point.label <= 12 
              ? monthNames[point.label - 1] 
              : point.label}
          </text>
        ))}

        {/* Area under the curve */}
        <path
          d={areaData}
          fill="url(#areaGradient)"
        />

        {/* Line */}
        <path
          d={pathData}
          fill="none"
          stroke="#667eea"
          strokeWidth="3"
          strokeLinejoin="round"
          strokeLinecap="round"
          filter="url(#lineShadow)"
        />

        {/* Data points */}
        {points.map((point, i) => (
          <g key={i}>
            <circle
              cx={point.x}
              cy={point.y}
              r="6"
              fill="white"
              stroke="#667eea"
              strokeWidth="3"
              style={{ cursor: 'pointer' }}
              onMouseEnter={(e) => {
                e.target.setAttribute('r', '8')
                e.target.style.filter = 'drop-shadow(0 4px 8px rgba(102, 126, 234, 0.3))'
              }}
              onMouseLeave={(e) => {
                e.target.setAttribute('r', '6')
                e.target.style.filter = 'none'
              }}
              onClick={() => {
                const label = typeof point.label === 'number' && point.label <= 12 
                  ? monthNames[point.label - 1] 
                  : point.label
                alert(`${label}: ${point.value.toLocaleString()} points`)
              }}
            />
            {/* Tooltip circle for better hover area */}
            <circle
              cx={point.x}
              cy={point.y}
              r="12"
              fill="transparent"
              style={{ cursor: 'pointer' }}
              onClick={() => {
                const label = typeof point.label === 'number' && point.label <= 12 
                  ? monthNames[point.label - 1] 
                  : point.label
                alert(`${label}: ${point.value.toLocaleString()} points`)
              }}
            />
          </g>
        ))}

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

export default LineChart