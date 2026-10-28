import { act, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import App from './App'

describe('sign-in and signup screens', () => {
  it('opens signup from Create account, moves focus, and can return to sign in', async () => {
    const user = userEvent.setup()
    render(<App />)
    expect(screen.getByRole('heading', { name: 'Welcome back.' })).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    expect(screen.getByRole('heading', { name: 'Create your account.' })).toBeVisible()
    expect(screen.getByLabelText('Email address')).toHaveFocus()
    expect(screen.getByLabelText('Confirm password')).toHaveAttribute('autocomplete', 'new-password')
    expect(document.title).toBe('Create account · Property Co Pilot')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(screen.queryByLabelText('Confirm password')).not.toBeInTheDocument()
  })

  it('toggles password visibility without submitting or losing the value', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)
    const password = screen.getByLabelText('Password')
    await user.type(password, 'SamplePass123')
    await user.click(screen.getByRole('button', { name: 'Show password' }))
    expect(password).toHaveAttribute('type', 'text')
    expect(password).toHaveValue('SamplePass123')
    await user.click(screen.getByRole('button', { name: 'Hide password' }))
    expect(password).toHaveAttribute('type', 'password')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('focuses the first invalid field and does not send invalid data', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(screen.getByLabelText('Email address')).toHaveFocus()
    expect(screen.getByLabelText('Email address')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByText('Enter a valid email address.')).toBeVisible()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('shows password confirmation errors locally', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    await user.type(screen.getByLabelText('Email address'), 'test@example.com')
    await user.type(screen.getByLabelText('Password'), 'SamplePass123')
    await user.type(screen.getByLabelText('Confirm password'), 'Different123')
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    expect(screen.getByText('Your passwords don’t match.')).toBeVisible()
    expect(screen.getByLabelText('Confirm password')).toHaveFocus()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('prevents duplicate submissions and opens Browse only after the server responds', async () => {
    const user = userEvent.setup()
    let complete!: (response: Response) => void
    const fetchMock = vi.fn(() => new Promise<Response>((resolve) => { complete = resolve }))
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)
    await user.type(screen.getByLabelText('Email address'), 'test@example.com')
    await user.type(screen.getByLabelText('Password'), 'SamplePass123')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(screen.getByRole('button', { name: 'Signing in…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Create account' })).toBeDisabled()
    fireEvent.submit(screen.getByRole('form', { name: 'Sign in' }))
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('main', { name: 'Browse' })).not.toBeInTheDocument()
    await act(async () => complete(new Response('User signed in successfully!')))
    expect(screen.getByRole('heading', { name: 'Browse properties.' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Open navigation' })).toHaveFocus()
    expect(document.title).toBe('Browse · Property Co Pilot')
    expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
    expect(window.localStorage.length).toBe(0)
  })

  it('also opens Browse after successful signup, which signs the user in', async () => {
    const user = userEvent.setup()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('User created successfully!', { status: 201 })))
    render(<App />)
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    await user.type(screen.getByLabelText('Email address'), 'test@example.com')
    await user.type(screen.getByLabelText('Password'), 'SamplePass123')
    await user.type(screen.getByLabelText('Confirm password'), 'SamplePass123')
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    expect(await screen.findByRole('heading', { name: 'Browse properties.' })).toBeVisible()
    expect(screen.queryByRole('form', { name: 'Sign up' })).not.toBeInTheDocument()
  })

  it('shows an accessible server error and permits retry', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn().mockResolvedValue(new Response('bad credentials', { status: 401 }))
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)
    await user.type(screen.getByLabelText('Email address'), 'test@example.com')
    await user.type(screen.getByLabelText('Password'), 'SamplePass123')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('That email and password don’t match.')
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeEnabled()
  })

  it('clears entered passwords when changing screens', async () => {
    const user = userEvent.setup()
    render(<App />)
    await user.type(screen.getByLabelText('Password'), 'SamplePass123')
    await user.click(screen.getByRole('button', { name: 'Create account' }))
    expect(screen.getByLabelText('Password')).toHaveValue('')
  })

  it('cancels an in-flight request when unmounted', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn(() => new Promise<Response>(() => {}))
    vi.stubGlobal('fetch', fetchMock)
    const { unmount } = render(<App />)
    await user.type(screen.getByLabelText('Email address'), 'test@example.com')
    await user.type(screen.getByLabelText('Password'), 'SamplePass123')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    const signal = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1].signal
    unmount()
    expect(signal?.aborted).toBe(true)
  })
})
