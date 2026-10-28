import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import Workspace from '../components/Workspace'
import Browse from './Browse'
import { detailFixture, portfolioFixture } from '../test/propertyFixtures'
import { areaFixture } from '../test/analysisFixtures'

function pageData(index = 1, count = 20, hasNext = true) {
  return {
    index,
    page_size: 20,
    has_next: hasNext,
    properties: Array.from({ length: count }, (_, offset) => ({
      ...portfolioFixture.properties[0],
      uid: `property_${index}_${offset}`,
      address: `Page ${index} home ${offset}`,
      image: null,
    })),
  }
}

describe('Browse', () => {
  it('loads twenty real-contract overviews with SVG fallbacks and a scrollable list', async () => {
    const fetchMock = vi.fn().mockResolvedValue(Response.json(pageData()))
    vi.stubGlobal('fetch', fetchMock)
    render(<Workspace />)

    const list = await screen.findByRole('region', { name: 'Property listings' })
    expect(within(list).getAllByRole('link')).toHaveLength(20)
    expect(within(list).getAllByText('No photo available')).toHaveLength(20)
    expect(within(list).getAllByText('Estimation / m²')).toHaveLength(20)
    expect(within(list).getAllByText('Asking / m²')).toHaveLength(20)
    expect(within(list).getAllByText('GeoPremium')).toHaveLength(20)
    expect(list).toHaveClass('browse-results-scroll')
    expect(fetchMock).toHaveBeenCalledWith('/User/BrowseProperties?transaction=sell&index=1', expect.objectContaining({ credentials: 'include' }))
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Next' })).toBeEnabled()
  })

  it('submits selected bounds, transaction, asking range and sqm, resetting to page one', async () => {
    vi.useFakeTimers()
    const fetchMock = vi.fn((url: string) => Promise.resolve(Response.json(
      url.startsWith('/analysis/autocomplete') ? [areaFixture] : pageData(),
    )))
    vi.stubGlobal('fetch', fetchMock)
    render(<Workspace />)

    fireEvent.change(screen.getByRole('combobox', { name: 'Area name' }), { target: { value: 'Piraeus' } })
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    fireEvent.click(screen.getByRole('option', { name: areaFixture.display_name }))
    fireEvent.change(screen.getByLabelText('Transaction type'), { target: { value: 'rent' } })
    fireEvent.change(screen.getByLabelText('Minimum asking / m²'), { target: { value: '5.25' } })
    fireEvent.change(screen.getByLabelText('Maximum asking / m²'), { target: { value: '15.75' } })
    fireEvent.change(screen.getByLabelText('Minimum size'), { target: { value: '50' } })
    fireEvent.change(screen.getByLabelText('Maximum size'), { target: { value: '100' } })
    fireEvent.submit(screen.getByRole('form', { name: 'Property filters' }))
    await act(async () => vi.advanceTimersByTimeAsync(10))

    const lastURL = fetchMock.mock.calls.at(-1)![0]
    const params = new URL(lastURL, 'http://localhost').searchParams
    expect(Object.fromEntries(params)).toEqual({
      transaction: 'rent', index: '1',
      asking_price_per_sqm_min: '5.25', asking_price_per_sqm_max: '15.75',
      square_meters_min: '50', square_meters_max: '100',
      min_lat: '37.9', max_lat: '37.98', min_lon: '23.6', max_lon: '23.7',
    })
  })

  it('opens a browse property and returns to the same filtered page', async () => {
    const user = userEvent.setup()
    window.location.hash = '/browse?transaction=rent&square_meters_min=50&index=1'
    const fetchMock = vi.fn((url: string) => {
      if (url.startsWith('/User/BrowseProperty?')) {
        return Promise.resolve(Response.json(detailFixture))
      }
      const index = Number(new URL(url, 'http://localhost').searchParams.get('index'))
      return Promise.resolve(Response.json(pageData(index, 20, index === 1)))
    })
    vi.stubGlobal('fetch', fetchMock)
    render(<Workspace />)

    await screen.findByRole('link', { name: 'View property: Page 1 home 0' })
    await user.click(screen.getByRole('button', { name: 'Next' }))
    await user.click(await screen.findByRole('link', { name: 'View property: Page 2 home 0' }))
    expect(await screen.findByRole('heading', { name: 'The property, in detail' })).toBeVisible()
    expect(fetchMock).toHaveBeenCalledWith('/User/BrowseProperty?uid=property_2_0', expect.anything())
    expect(screen.getByText(/Market listing/)).toBeVisible()
    await user.click(screen.getByRole('link', { name: 'Back to results' }))
    expect(await screen.findByRole('link', { name: 'View property: Page 2 home 0' })).toBeVisible()
    expect(screen.getByLabelText('Transaction type')).toHaveValue('rent')
    expect(screen.getByLabelText('Minimum size')).toHaveValue(50)
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
    expect(screen.getByText('Page 2 · 20 per page')).toBeVisible()
  })

  it('does not submit an unselected area or an inverted range', async () => {
    const fetchMock = vi.fn().mockResolvedValue(Response.json(pageData()))
    vi.stubGlobal('fetch', fetchMock)
    render(<Workspace />)
    await screen.findByRole('region', { name: 'Property listings' })
    const area = screen.getByRole('combobox', { name: 'Area name' })
    fireEvent.change(area, { target: { value: 'Not selected' } })
    fireEvent.submit(screen.getByRole('form', { name: 'Property filters' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Choose an area')
    fireEvent.change(area, { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('Minimum size'), { target: { value: '100' } })
    fireEvent.change(screen.getByLabelText('Maximum size'), { target: { value: '50' } })
    fireEvent.submit(screen.getByRole('form', { name: 'Property filters' }))
    expect(screen.getByRole('alert')).toHaveTextContent('minimum must not be greater')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('clears filters and shows an empty last page without a Next button enabled', async () => {
    const user = userEvent.setup()
    window.location.hash = '/browse?transaction=rent&square_meters_min=50&index=2'
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(Response.json(pageData(1, 0, false)))))
    render(<Workspace />)
    expect(await screen.findByRole('heading', { name: 'No properties found' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Clear filters' }))
    await waitFor(() => expect(window.location.hash).toBe('#/browse'))
    expect(screen.getByLabelText('Minimum size')).toHaveValue(null)
    expect(screen.getByLabelText('Transaction type')).toHaveValue('sell')
  })

  it.each([401, 500])('handles HTTP %i and exposes the appropriate recovery action', async (status) => {
    const onSignIn = vi.fn()
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response('private SQL', { status }))))
    render(<Browse query="" onSignIn={onSignIn} />)
    expect(await screen.findByRole('alert')).not.toHaveTextContent('SQL')
    const name = status === 401 ? 'Sign in again' : 'Try again'
    expect(screen.getByRole('button', { name })).toBeVisible()
    if (status === 401) {
      fireEvent.click(screen.getByRole('button', { name }))
      expect(onSignIn).toHaveBeenCalledOnce()
    }
  })

  it('uses an available picture and falls back to SVG when it fails', async () => {
    const page = pageData(1, 1, false)
    const property = { ...page.properties[0], image: 'https://example.com/property.jpg' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ ...page, properties: [property] })))
    render(<Browse query="" onSignIn={vi.fn()} />)
    const image = await screen.findByRole('img')
    expect(image).toHaveAttribute('src', property.image)
    fireEvent.error(image)
    expect(screen.getByText('No photo available')).toBeVisible()
  })
})
