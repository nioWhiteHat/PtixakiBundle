import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import Portfolio from './Portfolio'
import PropertyDetail from './PropertyDetail'
import Workspace from '../components/Workspace'
import PropertyImage from '../components/PropertyImage'
import { detailFixture, portfolioFixture } from '../test/propertyFixtures'

const mockJSON = (value: unknown) => vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(Response.json(value))))

describe('portfolio', () => {
  it('shows loading, then real summary values and property overviews', async () => {
    let complete!: (response: Response) => void
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>((resolve) => { complete = resolve })))
    render(<Portfolio onSignIn={vi.fn()} />)
    expect(screen.getByRole('status')).toHaveTextContent('Loading your portfolio')
    await act(async () => complete(Response.json(portfolioFixture)))
    expect(screen.getByText('€470,000')).toBeVisible()
    expect(screen.getAllByRole('link', { name: /^View property:/ })).toHaveLength(3)
    const card = screen.getByRole('link', { name: 'View property: Egnatia 90, Thessaloniki' })
    for (const text of ['82 m²', '€4,000.00', '€2,650.00', '16,896', '+19.51%']) expect(within(card).getByText(text)).toBeVisible()
    expect(screen.getAllByRole('img', { name: 'No property image available' })).toHaveLength(3)
    expect(screen.getAllByText('Not provided')).toHaveLength(2)
    expect(screen.getByText('0%')).toBeVisible()
    expect(screen.getByText('0', { selector: 'dd' })).toBeVisible()
  })

  it('renders the empty portfolio without inventing listings', async () => {
    mockJSON({ property_count: 0, total_estimated_value: 0, properties: [] })
    render(<Portfolio onSignIn={vi.fn()} />)
    expect(await screen.findByText('A place for your properties')).toBeVisible()
    expect(screen.getByText('€0')).toBeVisible()
    expect(screen.queryByRole('link', { name: /^View property:/ })).not.toBeInTheDocument()
  })

  it('allows retry after a server error', async () => {
    const user = userEvent.setup()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(new Response('', { status: 500 })).mockResolvedValueOnce(Response.json(portfolioFixture)))
    render(<Portfolio onSignIn={vi.fn()} />)
    await user.click(await screen.findByRole('button', { name: 'Try again' }))
    expect(await screen.findByText('€470,000')).toBeVisible()
  })

  it('offers sign-in when the cookie is missing or expired', async () => {
    const user = userEvent.setup()
    const onSignIn = vi.fn()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 401 })))
    render(<Portfolio onSignIn={onSignIn} />)
    await user.click(await screen.findByRole('button', { name: 'Sign in again' }))
    expect(onSignIn).toHaveBeenCalledOnce()
  })

  it('cancels pending portfolio requests on unmount', () => {
    const fetchMock = vi.fn(() => new Promise<Response>(() => {}))
    vi.stubGlobal('fetch', fetchMock)
    const { unmount } = render(<Portfolio onSignIn={vi.fn()} />)
    const signal = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1].signal
    unmount()
    expect(signal?.aborted).toBe(true)
  })

  it('opens the clicked UID as a detail page and returns to the portfolio', async () => {
    const user = userEvent.setup()
    const fetchMock = vi.fn(async (url: string) => {
      if (url.startsWith('/User/BrowseProperties')) {
        return Response.json({ index: 1, page_size: 20, has_next: false, properties: [] })
      }
      if (url === '/User/Portfolio') {
        return Response.json(portfolioFixture)
      }
      return Response.json(detailFixture)
    })
    vi.stubGlobal('fetch', fetchMock)
    render(<Workspace />)
    await user.click(screen.getByRole('button', { name: 'Open navigation' }))
    await user.click(screen.getByRole('link', { name: 'Portfolio' }))
    await user.click(await screen.findByRole('link', { name: 'View property: Egnatia 90, Thessaloniki' }))
    expect(await screen.findByRole('heading', { name: 'Egnatia 90, Thessaloniki' })).toBeVisible()
    expect(window.location.hash).toBe('#/portfolio/property?uid=property_test_1')
    expect(fetchMock).toHaveBeenCalledWith('/User/ViewProperty?uid=property_test_1', expect.anything())
    expect(document.title).toBe('Property details · Property Co Pilot')
    await user.click(screen.getByRole('link', { name: 'Back to portfolio' }))
    expect(await screen.findByRole('heading', { name: 'My portfolio.' })).toBeVisible()
    expect(window.location.hash).toBe('#/portfolio')
  })
})

describe('property details', () => {
  it('shows all stored input groups and separates local SHAP from global importance', async () => {
    const user = userEvent.setup()
    mockJSON(detailFixture)
    render(<PropertyDetail uid="property_test_1" onSignIn={vi.fn()} />)
    expect(await screen.findByRole('heading', { name: 'Egnatia 90, Thessaloniki' })).toBeVisible()
    for (const text of ['2004', '2019', '0, 3', 'autonomous', 'gas', 'property_test_1', '€328,000', '€217,300']) expect(screen.getByText(text)).toBeVisible()
    const local = screen.getByRole('table', { name: /Property-specific SHAP/ })
    expect(within(local).getByText('+€1,200.00')).toBeVisible()
    expect(within(local).getByText('-€200.00')).toBeVisible()
    expect(within(local).getByText('85.71%')).toBeVisible()
    expect(within(local).getAllByRole('row')).toHaveLength(9)
    await user.click(screen.getByRole('button', { name: 'Show all 30 contributions' }))
    expect(within(local).getAllByRole('row')).toHaveLength(31)
    const global = screen.getByRole('table', { name: /Global SHAP/ })
    expect(within(global).getByText('6.45%')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Show all 30 features' }))
    expect(within(global).getAllByRole('row')).toHaveLength(31)
  })

  it('handles unpriced rentals and missing SHAP without hiding the property', async () => {
    mockJSON({ ...detailFixture, transaction: 'rent', asking_price_per_sqm: null, asking_total_price: null, residual_percentage: null, explanation: null, feature_importance: [], estimated_price_per_sqm: 12.34 })
    render(<PropertyDetail uid="property_test_1" onSignIn={vi.fn()} />)
    expect(await screen.findByText('For rent')).toBeVisible()
    expect(screen.getByText('Not provided')).toBeVisible()
    expect(screen.getByText('€12.34')).toBeVisible()
    expect(screen.getByText('16,896')).toBeVisible()
    expect(screen.getByText(/No explanation has been stored/)).toBeVisible()
    expect(screen.getByText(/Global feature importance is not available/)).toBeVisible()
    expect(screen.queryByText('Total asking')).not.toBeInTheDocument()
  })

  it('warns about explanation/global-model mismatches', async () => {
    mockJSON({ ...detailFixture, explanation: { ...detailFixture.explanation, matches_importance_model: false } })
    render(<PropertyDetail uid="property_test_1" onSignIn={vi.fn()} />)
    expect(await screen.findByText(/different model versions/)).toBeVisible()
  })

  it('shows a not-found message and keeps the way back available', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 404 })))
    render(<PropertyDetail uid="missing" onSignIn={vi.fn()} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('no longer available')
    expect(screen.getByRole('link', { name: 'Back to portfolio' })).toBeVisible()
  })

  it('encodes the UID, cancels old detail requests, and ignores late responses', async () => {
    let completeOld!: (response: Response) => void
    const fetchMock = vi.fn().mockImplementationOnce(() => new Promise<Response>((resolve) => { completeOld = resolve })).mockResolvedValueOnce(Response.json({ ...detailFixture, uid: 'new' }))
    vi.stubGlobal('fetch', fetchMock)
    const { rerender } = render(<PropertyDetail uid="old&other=1" onSignIn={vi.fn()} />)
    expect(fetchMock).toHaveBeenCalledWith('/User/ViewProperty?uid=old%26other%3D1', expect.anything())
    const signal = fetchMock.mock.calls[0][1].signal
    rerender(<PropertyDetail uid="new" onSignIn={vi.fn()} />)
    await waitFor(() => expect(screen.getByText('new')).toBeVisible())
    await act(async () => completeOld(Response.json(detailFixture)))
    expect(signal.aborted).toBe(true)
    expect(screen.getByText('new')).toBeVisible()
    expect(screen.queryByText('property_test_1')).not.toBeInTheDocument()
  })
})

describe('property images', () => {
  it.each(['https://example.com/house.jpg', 'data:image/png;base64,test'])('renders an image source without injecting markup: %s', (src) => {
    render(<PropertyImage src={src} address="Test home" />)
    expect(screen.getByRole('img', { name: 'Test home' })).toHaveAttribute('src', src)
  })

  it('uses the SVG placeholder on failure and tries a different image source', () => {
    const { rerender } = render(<PropertyImage src="https://example.com/broken.jpg" address="Test home" />)
    fireEvent.error(screen.getByRole('img', { name: 'Test home' }))
    expect(screen.getByRole('img', { name: 'No property image available' })).toBeVisible()
    rerender(<PropertyImage src="https://example.com/other.jpg" address="Test home" />)
    expect(screen.getByRole('img', { name: 'Test home' })).toHaveAttribute('src', 'https://example.com/other.jpg')
  })
})
