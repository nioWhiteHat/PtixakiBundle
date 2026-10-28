import { useEffect, useState } from 'react'
import { getPropertyData, PropertyRequestError } from '../property'

export default function usePropertyData<T>(url: string) {
  const [result, setResult] = useState<{ url: string; data?: T; error?: PropertyRequestError } | null>(null)
  const [attempt, setAttempt] = useState(0)
  useEffect(() => {
    const controller = new AbortController()
    getPropertyData<T>(url, controller.signal).then(
      (data) => {
        if (!controller.signal.aborted) {
          setResult({ url, data })
        }
      },
      (error: PropertyRequestError) => {
        if (!controller.signal.aborted) {
          setResult({ url, error })
        }
      },
    )
    return () => controller.abort()
  }, [url, attempt])
  const current = result?.url === url ? result : null
  return {
    data: current?.data,
    error: current?.error,
    retry: () => {
      setResult(null)
      setAttempt((value) => value + 1)
    },
  }
}
