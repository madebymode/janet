import React from 'react'

function StatCard({ icon, title, value, loading, error, subtitle, color = '#667eea', isStatus = false }) {
  return (
    <div style={{
      background: 'white',
      borderRadius: '12px',
      padding: '1.5rem',
      boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
      border: '1px solid #f0f0f0',
      transition: 'transform 0.2s ease, box-shadow 0.2s ease',
      cursor: 'default'
    }}
    onMouseEnter={(e) => {
      e.currentTarget.style.transform = 'translateY(-2px)'
      e.currentTarget.style.boxShadow = '0 8px 30px rgba(0,0,0,0.12)'
    }}
    onMouseLeave={(e) => {
      e.currentTarget.style.transform = 'translateY(0)'
      e.currentTarget.style.boxShadow = '0 4px 20px rgba(0,0,0,0.08)'
    }}
    >
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: '1rem'
      }}>
        <div style={{
          fontSize: '2.5rem',
          background: `${color}15`,
          padding: '0.75rem',
          borderRadius: '12px',
          lineHeight: 1
        }}>
          {icon}
        </div>
        
        <div style={{ flex: 1 }}>
          <div style={{
            fontSize: '0.875rem',
            fontWeight: '500',
            color: '#666',
            marginBottom: '0.25rem'
          }}>
            {title}
          </div>
          
          <div style={{
            fontSize: '1.75rem',
            fontWeight: '700',
            color: color,
            lineHeight: 1,
            marginBottom: '0.25rem'
          }}>
            {loading ? (
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '0.5rem',
                fontSize: '1rem'
              }}>
                <div 
                  style={{
                    width: '16px',
                    height: '16px',
                    border: `2px solid ${color}30`,
                    borderTop: `2px solid ${color}`,
                    borderRadius: '50%',
                    animation: 'spin 1s linear infinite'
                  }}
                />
                Loading...
              </div>
            ) : error ? (
              <span style={{ color: '#e74c3c', fontSize: '1rem' }}>Error</span>
            ) : isStatus ? (
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '0.5rem'
              }}>
                <div style={{
                  width: '8px',
                  height: '8px',
                  borderRadius: '50%',
                  backgroundColor: color,
                  animation: value === 'Online' ? 'pulse 2s infinite' : 'none'
                }} />
                {value}
              </div>
            ) : (
              typeof value === 'number' ? value.toLocaleString() : value
            )}
          </div>
          
          {subtitle && (
            <div style={{
              fontSize: '0.75rem',
              color: '#999',
              fontWeight: '500'
            }}>
              {subtitle}
            </div>
          )}
        </div>
      </div>
      
      <style>
        {`
          @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
          }
          
          @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
          }
        `}
      </style>
    </div>
  )
}

export default StatCard