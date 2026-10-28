export type AuthMode = 'signin' | 'signup'
export type AuthFields = { email: string; password: string; confirmPassword: string }
export type FieldErrors = Partial<Record<keyof AuthFields, string>>

export function validateAuth(mode: AuthMode, fields: AuthFields): FieldErrors {
  const errors: FieldErrors = {}
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(fields.email.trim())) {
    errors.email = 'Enter a valid email address.'
  }
  if (!fields.password) {
    errors.password = 'Enter your password.'
  } else if (mode === 'signup') {
    if (fields.password.length < 9 || fields.password.length > 72) {
      errors.password = 'Use between 9 and 72 characters.'
    } else if (!/^[a-zA-Z0-9]+$/.test(fields.password)) {
      errors.password = 'Use English letters and numbers only (no spaces or symbols).'
    }
  }
  if (mode === 'signup' && fields.password !== fields.confirmPassword) {
    errors.confirmPassword = 'Your passwords don’t match.'
  }
  return errors
}

function responseError(mode: AuthMode, status: number, body: string): string {
  if (status === 429) return 'Too many attempts. Please wait a moment and try again.'
  if (status >= 500) return 'We couldn’t reach the sign-in service. Please try again shortly.'
  if (mode === 'signin' && (status === 400 || status === 401)) {
    return 'That email and password don’t match. Please try again.'
  }
  if (mode === 'signup') {
    if (/duplicate|unique constraint|already exists/i.test(body)) {
      return 'An account with this email already exists. Try signing in.'
    }
    if (/email domain cannot receive mail|invalid email/i.test(body)) {
      return 'Please use a valid email address that can receive mail.'
    }
    if (/password/i.test(body)) {
      return 'Use a password with 9–72 English letters and numbers, without spaces or symbols.'
    }
  }
  return mode === 'signup'
    ? 'We couldn’t create your account. Please check your details and try again.'
    : 'We couldn’t sign you in. Please try again.'
}

export async function authenticate(mode: AuthMode, fields: AuthFields, signal?: AbortSignal): Promise<void> {
  const timeout = AbortSignal.timeout(15_000)
  const requestSignal = signal ? AbortSignal.any([signal, timeout]) : timeout
  let response: Response
  try {
    response = await fetch(`/register/${mode === 'signin' ? 'SignIn' : 'SignUp'}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      signal: requestSignal,
      body: JSON.stringify({
        email: fields.email.trim(),
        password: fields.password,
        ...(mode === 'signup' ? { admin: false } : {}),
      }),
    })
  } catch (error) {
    if (signal?.aborted) throw error
    throw new Error(timeout.aborted
      ? 'The request timed out. Please try again.'
      : 'Cannot connect right now. Check your connection and make sure the Go server is running.', { cause: error })
  }
  if (!response.ok) {
    throw new Error(responseError(mode, response.status, await response.text()))
  }
}
