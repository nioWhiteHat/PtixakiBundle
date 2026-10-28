import { describe, expect, it, vi } from 'vitest'
import { authenticate, validateAuth } from './auth'

const fields = { email: '  test@example.com  ', password: 'SamplePass123', confirmPassword: 'SamplePass123' }

describe('authentication validation', () => {
  it('accepts valid signup and trims the email for validation', () => {
    expect(validateAuth('signup', fields)).toEqual({})
  })

  it.each(['short123', 'x'.repeat(73), 'SamplePass!', 'Sample Pass123', 'κωδικός123'])('rejects a signup password unsupported by Go: %s', (password) => {
    expect(validateAuth('signup', { ...fields, password, confirmPassword: password }).password).toBeTruthy()
  })

  it('requires valid email, password, and matching confirmation', () => {
    expect(validateAuth('signup', { email: 'invalid', password: '', confirmPassword: '' })).toHaveProperty('email')
    expect(validateAuth('signin', { ...fields, password: '' })).toHaveProperty('password')
    expect(validateAuth('signup', { ...fields, confirmPassword: 'Different123' })).toHaveProperty('confirmPassword')
  })

  it('does not impose new signup rules on existing sign-in passwords', () => {
    expect(validateAuth('signin', { ...fields, password: 'existing!' })).toEqual({})
  })
})

describe('Go authentication contract', () => {
  it('posts only sign-in fields and includes the session cookie', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('User signed in successfully!', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    await authenticate('signin', fields)
    expect(fetchMock).toHaveBeenCalledWith('/register/SignIn', expect.objectContaining({
      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'test@example.com', password: 'SamplePass123' }),
    }))
  })

  it('creates only regular users and accepts the plain-text 201 response', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('User created successfully!', { status: 201 }))
    vi.stubGlobal('fetch', fetchMock)
    await authenticate('signup', fields)
    expect(fetchMock).toHaveBeenCalledWith('/register/SignUp', expect.objectContaining({
      body: JSON.stringify({ email: 'test@example.com', password: 'SamplePass123', admin: false }),
    }))
  })

  it.each([
    ['signin', 401, 'password mismatch', 'That email and password don’t match.'],
    ['signup', 400, 'ERROR: duplicate key value violates unique constraint "user_credentials_email_key"', 'An account with this email already exists.'],
    ['signup', 400, 'email domain cannot receive mail', 'Please use a valid email address'],
    ['signup', 400, 'internal SQL details', 'We couldn’t create your account.'],
    ['signin', 500, 'internal SQL details', 'We couldn’t reach the sign-in service.'],
    ['signin', 429, 'slow down', 'Too many attempts.'],
  ] as const)('sanitizes %s HTTP %s errors', async (mode, status, body, message) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(body, { status })))
    await expect(authenticate(mode, fields)).rejects.toThrow(message)
  })

  it('gives an actionable connection error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    await expect(authenticate('signin', fields)).rejects.toThrow('Cannot connect right now.')
  })
})
