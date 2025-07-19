import { useState, useEffect, useRef } from 'react'

// Simple in-memory cache for API responses
const apiCache = new Map()

// API helper functions
export const api = {
  async get(url) {
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    return response.json()
  },

  async post(url, data) {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data)
    })
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    return response.json()
  },

  async delete(url) {
    const response = await fetch(url, {
      method: 'DELETE'
    })
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    return response.json()
  }
}

// Hook for fetching data with caching
export function useApi(url, dependencies = []) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const isMounted = useRef(true) // To prevent state updates on unmounted components

  useEffect(() => {
    isMounted.current = true
    return () => {
      isMounted.current = false
    }
  }, [])

  useEffect(() => {
    if (!url) {
      if (isMounted.current) {
        setLoading(false)
      }
      return
    }

    const fetchData = async () => {
      // Check cache first
      if (apiCache.has(url)) {
        if (isMounted.current) {
          setData(apiCache.get(url))
          setLoading(false)
          setError(null)
        }
        return
      }

      try {
        if (isMounted.current) {
          setLoading(true)
          setError(null)
        }
        const result = await api.get(url)
        apiCache.set(url, result) // Store in cache
        if (isMounted.current) {
          setData(result)
        }
      } catch (err) {
        if (isMounted.current) {
          setError(err.message)
        }
      } finally {
        if (isMounted.current) {
          setLoading(false)
        }
      }
    }

    fetchData()
  }, [url, ...dependencies])

  const refetch = async () => {
    if (!url) return
    
    // Invalidate cache for this URL on refetch
    apiCache.delete(url)

    try {
      if (isMounted.current) {
        setLoading(true)
        setError(null)
      }
      const result = await api.get(url)
      apiCache.set(url, result) // Store in cache
      if (isMounted.current) {
        setData(result)
      }
    } catch (err) {
      if (isMounted.current) {
        setError(err.message)
      }
    } finally {
      if (isMounted.current) {
        setLoading(false)
      }
    }
  }

  return { data, loading, error, refetch }
}