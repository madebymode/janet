import { useState, useEffect, useRef } from 'react'

// Simple in-memory cache for API responses
const apiCache = new Map()

// localStorage cache helpers with expiry
const CACHE_EXPIRY_MS = 5 * 60 * 1000 // 5 minutes

function getCachedData(key) {
  try {
    const cached = localStorage.getItem(`api_cache_${key}`)
    if (!cached) return null

    const { data, timestamp } = JSON.parse(cached)
    const age = Date.now() - timestamp

    if (age < CACHE_EXPIRY_MS) {
      return data
    }

    // Expired, remove it
    localStorage.removeItem(`api_cache_${key}`)
    return null
  } catch (err) {
    // If localStorage is full or unavailable, just return null
    return null
  }
}

function setCachedData(key, data) {
  try {
    const cacheEntry = {
      data,
      timestamp: Date.now()
    }
    localStorage.setItem(`api_cache_${key}`, JSON.stringify(cacheEntry))
  } catch (err) {
    // If localStorage is full or unavailable, just skip caching
    console.warn('Failed to cache data in localStorage:', err)
  }
}

function removeCachedData(key) {
  try {
    localStorage.removeItem(`api_cache_${key}`)
  } catch (err) {
    // Ignore errors
  }
}

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
      // Check in-memory cache first
      if (apiCache.has(url)) {
        if (isMounted.current) {
          setData(apiCache.get(url))
          setLoading(false)
          setError(null)
        }
        return
      }

      // Check localStorage cache
      const cachedData = getCachedData(url)
      if (cachedData) {
        if (isMounted.current) {
          setData(cachedData)
          setLoading(false)
          setError(null)
          apiCache.set(url, cachedData) // Also populate in-memory cache
        }
        return
      }

      try {
        if (isMounted.current) {
          setLoading(true)
          setError(null)
        }
        const result = await api.get(url)
        apiCache.set(url, result) // Store in memory cache
        setCachedData(url, result) // Store in localStorage cache
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
    removeCachedData(url)

    try {
      if (isMounted.current) {
        setLoading(true)
        setError(null)
      }
      const result = await api.get(url)
      apiCache.set(url, result) // Store in memory cache
      setCachedData(url, result) // Store in localStorage cache
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