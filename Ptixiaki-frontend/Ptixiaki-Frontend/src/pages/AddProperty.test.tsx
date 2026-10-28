import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import AddProperty from './AddProperty'
import Workspace from '../components/Workspace'
import { detailFixture } from '../test/propertyFixtures'

const addressFixture = {
  display_name: 'Test street 12, Athens, Greece',
  lat: '37.98', lon: '23.72', osm_id: 123, osm_type: 'way', boundingbox: [],
}

async function fillRequired() {
  const user = userEvent.setup()
  const originalFetch = globalThis.fetch
  vi.stubGlobal('fetch', vi.fn((...args: Parameters<typeof fetch>) => {
    if (String(args[0]).includes('/analysis/autocomplete')) {
      return Promise.resolve(Response.json([addressFixture]))
    }
    return originalFetch(...args)
  }))
  vi.useFakeTimers()
  fireEvent.change(screen.getByRole('combobox', { name: 'Full address' }), { target: { value: 'Test street' } })
  fireEvent.click(screen.getByRole('button', { name: 'Find address' }))
  await act(async () => vi.advanceTimersByTimeAsync(0))
  fireEvent.click(screen.getByRole('option', { name: addressFixture.display_name }))
  vi.useRealTimers()
  await user.type(screen.getByRole('spinbutton', { name: 'Area (m²)' }), '82')
  await user.type(screen.getByRole('spinbutton', { name: 'Construction year' }), '2004')
  return user
}

describe('add property', () => {
  it.each(['sell', 'rent'])('adds a %s property from an empty portfolio and opens that UID', async (transaction) => {
    window.history.replaceState(null, '', '/#/portfolio')
    const fetchMock = vi.fn(async (url: string) => {
      if (url === '/User/Portfolio') {
        return Response.json({ property_count: 0, total_estimated_value: 0, properties: [] })
      }
      if (url.startsWith('/User/AddProperty')) {
        return Response.json({ message: 'Property added successfully', uid: 'created&uid' }, { status: 201 })
      }
      return Response.json({ ...detailFixture, uid: 'created&uid', features: { ...detailFixture.features, Address: 'Test street 12, Athens, Greece' } })
    })
    vi.stubGlobal('fetch', fetchMock)
    render(<Workspace />)
    const user = userEvent.setup()
    expect(await screen.findByText('A place for your properties')).toBeVisible()
    await user.click(screen.getByRole('link', { name: 'Add property' }))
    expect(await screen.findByRole('heading', { name: 'Add a property.' })).toBeVisible()
    expect(document.title).toBe('Add property · Property Co Pilot')
    await fillRequired()
    await user.selectOptions(screen.getByRole('combobox', { name: 'Transaction' }), transaction)
    if (transaction === 'rent') {
      await user.type(screen.getByRole('spinbutton', { name: /^Monthly asking price/ }), '12.5')
    }
    await user.type(screen.getByRole('spinbutton', { name: /^Renovation year/ }), '2019')
    await user.clear(screen.getByRole('textbox', { name: /^Floors/ }))
    await user.type(screen.getByRole('textbox', { name: /^Floors/ }), '2, 3')
    await user.click(screen.getByRole('checkbox', { name: 'Sea' }))
    await user.click(screen.getByRole('checkbox', { name: 'Frontage' }))
    await user.click(screen.getByRole('checkbox', { name: 'Elevator' }))
    await user.click(screen.getByRole('button', { name: 'Add property' }))
    expect(await screen.findByRole('heading', { name: 'Test street 12, Athens, Greece' })).toBeVisible()
    expect(window.location.hash).toBe('#/portfolio/property?uid=created%26uid')
    const call = fetchMock.mock.calls.find(([url]) => url.startsWith('/User/AddProperty')) as unknown as [string, RequestInit]
    expect(call[0]).toBe(`/User/AddProperty?transaction=${transaction}`)
    expect(call[1]).toMatchObject({ method: 'POST', credentials: 'include' })
    const payload = JSON.parse(String(call[1].body))
    expect(payload).toMatchObject({
      Address: 'Test street 12, Athens, Greece', AddressOSMID: 123, AddressOSMType: 'way', SubType: 'apartment', SquareMeters: 82,
      ConstructionYear: 2004, IsRenovated: 1, RenovYear: 2019, FloorLevel: [2, 3],
      ViewSea: 1, NoView: 0, FacadeFrontage: 1, NoFacade: 0, Elevator: 1,
    })
    expect(payload).not.toHaveProperty('GeoLat')
    expect(payload).not.toHaveProperty('GeoLon')
    if (transaction === 'rent') {
      expect(payload.AskingPricePerSqm).toBe(12.5)
    } else {
      expect(payload).not.toHaveProperty('AskingPricePerSqm')
    }
    expect(fetchMock).toHaveBeenCalledWith('/User/ViewProperty?uid=created%26uid', expect.anything())
    await user.click(screen.getByRole('link', { name: 'Back to portfolio' }))
    expect(await screen.findByRole('heading', { name: 'My portfolio.' })).toBeVisible()
  })

  it.each([
    ['RenovYear', '2000', /renovation year must be between/],
    ['FloorLevel', '2, foo', /whole floor numbers/],
  ])('rejects invalid %s before sending a property', async (field, value, message) => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    render(<AddProperty onSignIn={vi.fn()} />)
    await fillRequired()
    const input = document.querySelector(`input[name="${field}"]`) as HTMLInputElement
    fireEvent.change(input, { target: { value } })
    fireEvent.submit(screen.getByRole('form', { name: 'Add property' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(message)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('prevents duplicate saves and aborts the request when leaving', async () => {
    const fetchMock = vi.fn(() => new Promise<Response>(() => {}))
    vi.stubGlobal('fetch', fetchMock)
    const { unmount } = render(<AddProperty onSignIn={vi.fn()} />)
    const user = await fillRequired()
    await user.click(screen.getByRole('button', { name: 'Add property' }))
    expect(screen.getByRole('button', { name: 'Adding property…' })).toBeDisabled()
    expect(screen.getByRole('combobox', { name: 'Full address' })).toBeDisabled()
    fireEvent.submit(screen.getByRole('form', { name: 'Add property' }))
    expect(fetchMock).toHaveBeenCalledOnce()
    const signal = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1].signal
    unmount()
    expect(signal?.aborted).toBe(true)
  })

  it('keeps entered details and offers sign-in after an expired session', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 401 })))
    const onSignIn = vi.fn()
    render(<AddProperty onSignIn={onSignIn} />)
    const user = await fillRequired()
    await user.click(screen.getByRole('button', { name: 'Add property' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('session has expired')
    expect(screen.getByRole('combobox', { name: 'Full address' })).toHaveValue(addressFixture.display_name)
    await user.click(screen.getByRole('button', { name: 'Sign in again' }))
    expect(onSignIn).toHaveBeenCalledOnce()
  })

  it('hides server internals and warns about a potentially saved property', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('private SQL failure', { status: 500 }))
    vi.stubGlobal('fetch', fetchMock)
    render(<AddProperty onSignIn={vi.fn()} />)
    const user = await fillRequired()
    await user.click(screen.getByRole('button', { name: 'Add property' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Check your portfolio before trying again')
    expect(screen.queryByText('private SQL failure')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open portfolio' })).toHaveAttribute('href', '#/portfolio')
    expect(fetchMock).toHaveBeenCalledOnce()
  })

  it('does not redirect or submit again when an older server omits the UID', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ message: 'Property added successfully' }, { status: 201 })))
    render(<AddProperty onSignIn={vi.fn()} />)
    const user = await fillRequired()
    await user.click(screen.getByRole('button', { name: 'Add property' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('did not return its ID')
    expect(window.location.hash).toBe('')
  })

  it('ignores a response that arrives after the form unmounts', async () => {
    let complete!: (response: Response) => void
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>((resolve) => { complete = resolve })))
    const { unmount } = render(<AddProperty onSignIn={vi.fn()} />)
    const user = await fillRequired()
    await user.click(screen.getByRole('button', { name: 'Add property' }))
    unmount()
    await act(async () => complete(Response.json({ uid: 'late-property' }, { status: 201 })))
    await waitFor(() => expect(window.location.hash).toBe(''))
  })
})


describe('address selection', () => {
  it('searches Nominatim only on request and shows selected coordinates', async () => {
    vi.useFakeTimers()
    const fetchMock = vi.fn().mockResolvedValue(Response.json([addressFixture]))
    vi.stubGlobal('fetch', fetchMock)
    render(<AddProperty onSignIn={vi.fn()} />)
    const input = screen.getByRole('combobox', { name: 'Full address' })
    expect(screen.getByRole('button', { name: 'Add property' })).toBeDisabled()
    fireEvent.change(input, { target: { value: 'Test street' } })
    await act(async () => vi.advanceTimersByTimeAsync(5000))
    expect(fetchMock).not.toHaveBeenCalled()
    fireEvent.keyDown(input, { key: 'Enter' })
    await act(async () => vi.advanceTimersByTimeAsync(0))
    expect(fetchMock).toHaveBeenCalledWith('/analysis/autocomplete?q=Test%20street&kind=address', expect.objectContaining({ credentials: 'include' }))
    fireEvent.keyDown(input, { key: 'ArrowDown' })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(input).toHaveValue(addressFixture.display_name)
    expect(screen.getByText(/Selected location: 37.98, 23.72/)).toBeVisible()
    expect(screen.getByRole('button', { name: 'Add property' })).toBeEnabled()
    fireEvent.change(input, { target: { value: 'Another street' } })
    expect(screen.queryByText(/Selected location:/)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add property' })).toBeDisabled()
  })

  it('rejects an edited address even if the form is submitted directly', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    render(<AddProperty onSignIn={vi.fn()} />)
    await fillRequired()
    fireEvent.change(screen.getByRole('combobox', { name: 'Full address' }), { target: { value: 'Another address' } })
    fireEvent.submit(screen.getByRole('form', { name: 'Add property' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Choose a matching address')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('requires a fresh selection after the backend rejects the selected address', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ error: { code: 'address_selection_invalid' } }, { status: 422 })))
    render(<AddProperty onSignIn={vi.fn()} />)
    const user = await fillRequired()
    await user.click(screen.getByRole('button', { name: 'Add property' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Choose the address again')
    expect(screen.getByRole('button', { name: 'Add property' })).toBeDisabled()
  })
})
