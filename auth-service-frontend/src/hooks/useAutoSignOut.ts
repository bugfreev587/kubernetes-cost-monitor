import { useState, useEffect, useRef, useCallback } from 'react'
import { useAuth } from '@clerk/clerk-react'

const SESSION_DURATION_MS = 15 * 60 * 1000 // 15 minutes
const WARNING_TIME_MS = 3 * 60 * 1000 // 3 minutes before timeout

export function useAutoSignOut() {
  const { isSignedIn, signOut } = useAuth()
  const [showWarning, setShowWarning] = useState(false)
  const [timeRemaining, setTimeRemaining] = useState(0)
  const timerRef = useRef<NodeJS.Timeout | null>(null)
  const warningTimerRef = useRef<NodeJS.Timeout | null>(null)
  const countdownIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const sessionStartRef = useRef<number | null>(null)

  const clearAllTimers = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    if (warningTimerRef.current) {
      clearTimeout(warningTimerRef.current)
      warningTimerRef.current = null
    }
    if (countdownIntervalRef.current) {
      clearInterval(countdownIntervalRef.current)
      countdownIntervalRef.current = null
    }
  }, [])

  const resetTimer = useCallback(() => {
    // Clear existing timers
    clearAllTimers()

    // Reset session start time
    sessionStartRef.current = Date.now()
    setShowWarning(false)
    setTimeRemaining(0)

    // Set warning timer (12 minutes from now)
    warningTimerRef.current = setTimeout(() => {
      setShowWarning(true)
      // Set initial countdown value
      setTimeRemaining(Math.ceil(WARNING_TIME_MS / 1000))
      
      // Start countdown from 3 minutes
      countdownIntervalRef.current = setInterval(() => {
        const elapsed = Date.now() - (sessionStartRef.current || Date.now())
        const remaining = SESSION_DURATION_MS - elapsed
        const secondsRemaining = Math.max(0, Math.ceil(remaining / 1000))
        setTimeRemaining(secondsRemaining)

        if (remaining <= 0) {
          if (countdownIntervalRef.current) {
            clearInterval(countdownIntervalRef.current)
            countdownIntervalRef.current = null
          }
        }
      }, 1000)
    }, SESSION_DURATION_MS - WARNING_TIME_MS)

    // Set sign-out timer (15 minutes from now)
    timerRef.current = setTimeout(async () => {
      if (isSignedIn) {
        try {
          await signOut()
        } catch (error) {
          console.error('Error signing out:', error)
        }
      }
    }, SESSION_DURATION_MS)
  }, [isSignedIn, signOut, clearAllTimers])

  const handleContinue = useCallback(() => {
    resetTimer()
  }, [resetTimer])

  const handleSignOut = useCallback(async () => {
    clearAllTimers()
    try {
      await signOut()
    } catch (error) {
      console.error('Error signing out:', error)
    }
  }, [signOut, clearAllTimers])

  useEffect(() => {
    if (isSignedIn) {
      resetTimer()
    } else {
      // Clear timers when user signs out
      clearAllTimers()
      setShowWarning(false)
    }

    return () => {
      clearAllTimers()
    }
  }, [isSignedIn, resetTimer, clearAllTimers])

  // Note: Timer only resets when user clicks "Continue" in the warning modal
  // This ensures the warning always appears after 12 minutes of inactivity

  return {
    showWarning,
    timeRemaining,
    handleContinue,
    handleSignOut,
  }
}
