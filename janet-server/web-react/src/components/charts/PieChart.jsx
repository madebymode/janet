import React from 'react'

function PieChart({ data, title }) {
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

  const total = data.reduce((sum, item) => sum + item.count, 0)
  let currentAngle = 0
  const radius = 80
  const centerX = 150
  const centerY = 150

  const colors = [
    '#667eea', '#764ba2', '#f093fb', '#f5576c', '#4facfe', '#00f2fe',
    '#a8edea', '#fed6e3', '#ffecd2', '#fcb69f', '#667eea', '#764ba2'
  ]

  const slices = data.map((item, index) => {
    const percentage = (item.count / total) * 100
    const angle = (item.count / total) * 360
    const x1 = centerX + radius * Math.cos((currentAngle * Math.PI) / 180)
    const y1 = centerY + radius * Math.sin((currentAngle * Math.PI) / 180)
    const x2 = centerX + radius * Math.cos(((currentAngle + angle) * Math.PI) / 180)
    const y2 = centerY + radius * Math.sin(((currentAngle + angle) * Math.PI) / 180)
    
    const largeArcFlag = angle > 180 ? 1 : 0
    const pathData = `M ${centerX} ${centerY} L ${x1} ${y1} A ${radius} ${radius} 0 ${largeArcFlag} 1 ${x2} ${y2} Z`
    
    currentAngle += angle

    return {
      path: pathData,
      color: colors[index % colors.length],
      percentage,
      ...item
    }
  })

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '2rem', flexWrap: 'wrap' }}>
      <div style={{ position: 'relative' }}>
        <svg width="300" height="300" viewBox="0 0 300 300" style={{ flexShrink: 0 }}>
          <defs>
            {slices.map((slice, index) => (
              <linearGradient key={index} id={`gradient-${index}`} x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stopColor={slice.color} stopOpacity="1" />
                <stop offset="100%" stopColor={slice.color} stopOpacity="0.8" />
              </linearGradient>
            ))}
            <filter id="shadow">
              <feDropShadow dx="2" dy="2" stdDeviation="3" floodOpacity="0.3"/>
            </filter>
          </defs>
          
          {slices.map((slice, index) => (
            <path
              key={index}
              d={slice.path}
              fill={`url(#gradient-${index})`}
              stroke="white"
              strokeWidth="2"
              filter="url(#shadow)"
              style={{
                cursor: 'pointer',
                transition: 'transform 0.2s ease'
              }}
              onMouseEnter={(e) => {
                e.target.style.transform = 'scale(1.05)'
                e.target.style.transformOrigin = `${centerX}px ${centerY}px`
              }}
              onMouseLeave={(e) => {
                e.target.style.transform = 'scale(1)'
              }}
              onClick={() => {
                alert(`${slice.range}: ${slice.count} users (${slice.percentage.toFixed(1)}%)`)
              }}
            />
          ))}
          
          {/* Center circle for donut effect */}
          <circle
            cx={centerX}
            cy={centerY}
            r="35"
            fill="white"
            stroke="#f0f0f0"
            strokeWidth="2"
          />
          
          {/* Center text */}
          <text
            x={centerX}
            y={centerY - 5}
            textAnchor="middle"
            fontSize="14"
            fontWeight="600"
            fill="#333"
          >
            Total
          </text>
          <text
            x={centerX}
            y={centerY + 10}
            textAnchor="middle"
            fontSize="16"
            fontWeight="700"
            fill="#667eea"
          >
            {total}
          </text>
        </svg>
      </div>
      
      <div style={{ flex: 1, minWidth: '200px' }}>
        <div style={{
          display: 'grid',
          gap: '0.5rem',
          maxHeight: '250px',
          overflowY: 'auto'
        }}>
          {slices.map((slice, index) => (
            <div
              key={index}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '0.75rem',
                padding: '0.5rem',
                borderRadius: '6px',
                background: '#f8f9fa',
                cursor: 'pointer',
                transition: 'background 0.2s ease'
              }}
              onMouseEnter={(e) => {
                e.target.style.background = '#e9ecef'
              }}
              onMouseLeave={(e) => {
                e.target.style.background = '#f8f9fa'
              }}
              onClick={() => {
                alert(`${slice.range}: ${slice.count} users (${slice.percentage.toFixed(1)}%)`)
              }}
            >
              <div
                style={{
                  width: '12px',
                  height: '12px',
                  borderRadius: '3px',
                  backgroundColor: slice.color,
                  flexShrink: 0
                }}
              />
              <div style={{ flex: 1, fontSize: '0.875rem' }}>
                <div style={{ fontWeight: '500', color: '#333' }}>
                  {slice.range}
                </div>
                <div style={{ color: '#666', fontSize: '0.8rem' }}>
                  {slice.count} users ({slice.percentage.toFixed(1)}%)
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default PieChart