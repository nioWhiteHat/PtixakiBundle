import { act, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import Analysis from './Analysis'
import AnalysisResults from '../components/AnalysisResults'
import AnalysisPoints from '../components/AnalysisPoints'
import AreaSearch from '../components/AreaSearch'
import ClusterPlot from '../components/ClusterPlot'
import { areaBounds } from '../analysis'
import { analysisFixture, areaFixture } from '../test/analysisFixtures'

describe('area search and requests', () => {
  it('defaults to autocomplete without an environment flag or Find areas button', () => {
    render(<Analysis onSignIn={vi.fn()} />)
    expect(screen.queryByRole('button', { name: 'Find areas' })).not.toBeInTheDocument()
    expect(screen.getByText(/Suggestions appear two seconds/)).toBeVisible()
  })
  it('maps Nominatim south/north/west/east to the exact Go bounding-box body', () => {
    expect(areaBounds(areaFixture)).toEqual({ MinLat: 37.9, MaxLat: 37.98, MinLon: 23.6, MaxLon: 23.7 })
    expect(() => areaBounds({ ...areaFixture, boundingbox: ['bad'] })).toThrow('bounding box')
  })

  it('waits two seconds after the last edit, selects an area, and submits a single authenticated POST', async () => {
    vi.useFakeTimers()
    const fetchMock = vi.fn((url: string) => Promise.resolve(Response.json(url.startsWith('/analysis/autocomplete') ? [areaFixture] : analysisFixture)))
    vi.stubGlobal('fetch', fetchMock)
    render(<Analysis onSignIn={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Analyse area' })).toBeDisabled()
    const search = screen.getByRole('combobox', { name: 'Area name' })
    fireEvent.change(search, { target: { value: 'Pir' } })
    await act(async () => vi.advanceTimersByTimeAsync(1500))
    fireEvent.change(search, { target: { value: 'Piraeus' } })
    await act(async () => vi.advanceTimersByTimeAsync(1999))
    expect(fetchMock).not.toHaveBeenCalled()
    await act(async () => vi.advanceTimersByTimeAsync(1))
    expect(fetchMock).toHaveBeenCalledTimes(1)
    fireEvent.keyDown(search, { key: 'ArrowDown' })
    fireEvent.keyDown(search, { key: 'Enter' })
    expect(search).toHaveValue(areaFixture.display_name)
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: 'Analyse area' })) })
    await act(async () => vi.advanceTimersByTimeAsync(0))
    expect(fetchMock).toHaveBeenLastCalledWith('/analysis/CreateAnalysisForArea', expect.objectContaining({ method: 'POST', credentials: 'include', body: JSON.stringify(areaBounds(areaFixture)) }))
    expect(screen.getByRole('heading', { name: 'Piraeus, Attica, Greece' })).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Similar properties, grouped' })).toBeVisible()
  })

  it('cancels stale autocomplete requests and ignores late responses', async () => {
    vi.useFakeTimers()
    let complete!: (response: Response) => void
    const fetchMock = vi.fn().mockImplementationOnce(() => new Promise<Response>((resolve) => { complete = resolve })).mockResolvedValueOnce(Response.json([]))
    vi.stubGlobal('fetch', fetchMock)
    render(<AreaSearch selected={null} onSelect={vi.fn()} disabled={false} onSignIn={vi.fn()} />)
    const input = screen.getByRole('combobox')
    fireEvent.change(input, { target: { value: 'Piraeus' } })
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    const signal = fetchMock.mock.calls[0][1].signal
    fireEvent.change(input, { target: { value: 'Athens' } })
    expect(signal.aborted).toBe(true)
    await act(async () => complete(Response.json([areaFixture])))
    expect(screen.queryByRole('option')).not.toBeInTheDocument()
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    expect(screen.getByText(/No matching areas found/)).toBeVisible()
  })

  it('blocks duplicate analysis, supports stopping the wait, and does not display a cancelled result', async () => {
    vi.useFakeTimers()
    let complete!: (response: Response) => void
    const fetchMock = vi.fn().mockResolvedValueOnce(Response.json([areaFixture])).mockImplementationOnce(() => new Promise<Response>((resolve) => { complete = resolve }))
    vi.stubGlobal('fetch', fetchMock)
    render(<Analysis onSignIn={vi.fn()} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Piraeus' } })
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    fireEvent.click(screen.getByRole('option'))
    fireEvent.click(screen.getByRole('button', { name: 'Analyse area' }))
    expect(screen.getByRole('button', { name: 'Analysing…' })).toBeDisabled()
    fireEvent.submit(screen.getByRole('button', { name: 'Analysing…' }).closest('form')!)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    fireEvent.click(screen.getByRole('button', { name: 'Stop waiting' }))
    expect(fetchMock.mock.calls[1][1].signal.aborted).toBe(true)
    await act(async () => complete(Response.json(analysisFixture)))
    expect(screen.queryByRole('heading', { name: 'Similar properties, grouped' })).not.toBeInTheDocument()
  })

  it('shows authentication failures from autocomplete', async () => {
    vi.useFakeTimers()
    const onSignIn = vi.fn()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 401 })))
    render(<AreaSearch selected={null} onSelect={vi.fn()} disabled={false} onSignIn={onSignIn} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Piraeus' } })
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    expect(screen.getByRole('alert')).toHaveTextContent('session has expired')
    fireEvent.click(screen.getByRole('button', { name: 'Sign in again' }))
    expect(onSignIn).toHaveBeenCalledOnce()
  })

  it('clearing or editing a selected area prevents using its old bounds', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json([areaFixture])))
    render(<Analysis onSignIn={vi.fn()} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Piraeus' } })
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    fireEvent.click(screen.getByRole('option'))
    expect(screen.getByRole('button', { name: 'Analyse area' })).toBeEnabled()
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Athens' } })
    expect(screen.getByRole('button', { name: 'Analyse area' })).toBeDisabled()
  })

  it.each([401, 500])('handles analysis HTTP %i without exposing backend internals', async (status) => {
    vi.useFakeTimers()
    const onSignIn = vi.fn()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(Response.json([areaFixture])).mockResolvedValueOnce(new Response('private SQL details', { status })))
    render(<Analysis onSignIn={onSignIn} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Piraeus' } })
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    fireEvent.click(screen.getByRole('option'))
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: 'Analyse area' })) })
    expect(screen.getByRole('alert')).not.toHaveTextContent('SQL')
    if (status === 401) {
      fireEvent.click(screen.getByRole('button', { name: 'Sign in again' }))
      expect(onSignIn).toHaveBeenCalledOnce()
    } else expect(screen.getByRole('button', { name: 'Try again' })).toBeEnabled()
  })

  it('explains a database error and clears it when the area is edited', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(Response.json([areaFixture])).mockResolvedValueOnce(Response.json({ error: { code: 'analysis_data_unavailable' } }, { status: 500 })))
    render(<Analysis onSignIn={vi.fn()} />)
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Pireaus' } })
    await act(async () => vi.advanceTimersByTimeAsync(2000))
    fireEvent.click(screen.getByRole('option'))
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: 'Analyse area' })) })
    expect(screen.getByRole('alert')).toHaveTextContent('property data')
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'Athens' } })
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})

describe('analysis results', () => {
  it('preserves clusters and explains why a model was skipped for a small area', () => {
    const warning = 'rent model skipped: 1 eligible properties; at least 10 are required. Available cluster statistics are still shown.'
    render(<AnalysisResults data={{ ...analysisFixture, rent_report: null, warnings: [warning] }} areaName="Small area" />)
    expect(screen.getByRole('status')).toHaveTextContent(warning)
    expect(screen.getByRole('heading', { name: 'Similar properties, grouped' })).toBeVisible()
    expect(screen.getByText('0.7342')).toBeVisible()
  })
  it('displays clusters, all mean/std fields, Elo tails, model reports, interactions and examples', async () => {
    const user = userEvent.setup()
    render(<AnalysisResults data={analysisFixture} areaName="Piraeus" />)
    expect(screen.getByText('80', { selector: 'dd' })).toBeVisible()
    const table = screen.getByRole('table', { name: 'Cluster 0 means and standard deviations' })
    expect(within(table).getAllByRole('row')).toHaveLength(7)
    expect(screen.getAllByText('Highest Elo · top 5% (1)')).toHaveLength(3)
    expect(screen.getByText('0.7342')).toBeVisible()
    expect(screen.getByRole('table', { name: 'Area XGBoost feature importance' })).toBeVisible()
    expect(screen.getByRole('table', { name: 'Area SHAP feature interactions' })).toBeVisible()
    expect(document.querySelectorAll('.area-example')).toHaveLength(10)
    fireEvent.click(document.querySelector('.area-example summary')!)
    expect(screen.getByRole('table', { name: 'Area-model contributions for property_area_0' })).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Cluster 1' }))
    expect(screen.queryByRole('table', { name: 'Cluster 0 means and standard deviations' })).not.toBeInTheDocument()
    expect(screen.getByRole('table', { name: 'Cluster 1 means and standard deviations' })).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Rentals' }))
    expect(screen.getByRole('heading', { name: 'The area’s rental model' })).toBeVisible()
    expect(screen.getByText('0.61')).toBeVisible()
    expect(screen.getByText('€10.50')).toBeVisible()
  })

  it('shows every property field with 20 rows per page', async () => {
    const user = userEvent.setup()
    render(<AnalysisPoints points={analysisFixture.points!} title="All points" />)
    const table = screen.getByRole('table', { name: 'All points' })
    expect(within(table).getAllByRole('columnheader')).toHaveLength(17)
    expect(within(table).getAllByRole('row')).toHaveLength(21)
    expect(within(table).getByText('property_area_0')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(within(table).queryByText('property_area_0')).not.toBeInTheDocument()
    expect(within(table).getByText('property_area_20')).toBeVisible()
  })

  it('handles null arrays and missing transaction reports', () => {
    render(<AnalysisResults data={{ points: null, clusters: null, sales_report: null, rent_report: null }} areaName="Empty area" />)
    expect(screen.getByText('No eligible properties were returned for this area.')).toBeVisible()
    expect(screen.getByText(/No clusters were returned/)).toBeVisible()
    expect(screen.getByRole('heading', { name: 'No rental model returned' })).toBeVisible()
  })

  it('bounds only the visual plot while keeping original point data intact', () => {
    const points = Array.from({ length: 10001 }, (_, index) => ({ ...analysisFixture.points![0], Uid: String(index) }))
    const { container } = render(<ClusterPlot points={points} />)
    expect(container.querySelectorAll('circle').length).toBeLessThanOrEqual(2000)
    expect(screen.getByText(/All properties remain available below/)).toBeVisible()
    expect(points).toHaveLength(10001)
  })
})
