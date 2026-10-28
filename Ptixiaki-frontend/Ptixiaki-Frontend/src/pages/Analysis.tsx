import { useEffect, useRef, useState } from 'react'
import { areaBounds } from '../analysis'
import type { AnalysisData, AreaSuggestion } from '../analysis'
import { getPropertyData, PropertyRequestError } from '../property'
import AreaSearch from '../components/AreaSearch'
import AnalysisResults from '../components/AnalysisResults'
import './Properties.css'
import './Analysis.css'

const analysisErrors: Record<string, string> = {
  invalid_area: 'Choose an area from the dropdown so we can use its boundary.',
  analysis_data_unavailable: 'The server couldn’t load this area’s property data. Check the Go terminal for the [ANALYSIS] error.',
}

export default function Analysis({ onSignIn }: { onSignIn: () => void }) {
  const [selected, setSelected] = useState<AreaSuggestion | null>(null)
  const [result, setResult] = useState<{ data: AnalysisData; areaName: string } | null>(null)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<PropertyRequestError | null>(null)
  const request = useRef<AbortController | null>(null)
  useEffect(() => () => request.current?.abort(), [])

  async function analyse() {
    if (!selected || request.current) return
    const controller = new AbortController()
    request.current = controller
    setPending(true); setError(null); setResult(null)
    try {
      const data = await getPropertyData<AnalysisData>('/analysis/CreateAnalysisForArea', controller.signal, { json: areaBounds(selected), timeoutMs: 300_000 })
      if (!controller.signal.aborted) setResult({ data, areaName: selected.display_name })
    } catch (error) {
      if (!controller.signal.aborted) setError(error instanceof PropertyRequestError ? error : new PropertyRequestError((error as Error).message))
    } finally {
      if (!controller.signal.aborted) { setPending(false); request.current = null }
    }
  }

  return (
    <div className={`properties-page analysis-page ${!result ? 'analysis-search-page' : ''}`}>
      <header className="properties-heading"><p className="eyebrow">A LITTLE LOCAL PERSPECTIVE</p><h1>Understand an area<span>.</span></h1><p>Explore similar homes, local price patterns, and what drives the market.</p></header>
      <form className="analysis-search-form" onSubmit={(event) => { event.preventDefault(); void analyse() }}>
        <AreaSearch selected={selected} onSelect={(area) => { setSelected(area); setError(null) }} disabled={pending} onSignIn={onSignIn} />
        <button type="submit" className="property-button analyse-button" disabled={!selected || pending}>{pending ? 'Analysing…' : 'Analyse area'}</button>
      </form>
      {selected && <p className="panel-note analysis-bounds">Selected bounds (south, north, west, east): {selected.boundingbox.join(', ')}</p>}
      <p className="panel-note analysis-source">Area search: stored administrative boundaries · © OpenStreetMap contributors</p>
      {pending && <div className="property-state"><span className="property-loading" aria-hidden="true" /><p role="status">Building clusters and training the area’s sale and rental models. This can take several minutes.</p><button type="button" className="property-button" onClick={() => { request.current?.abort(); request.current = null; setPending(false) }}>Stop waiting</button><p className="panel-note">Stopping the browser request may not stop work already running in Go or Python.</p></div>}
      {error && <div className="property-state" role="alert"><h2>Analysis couldn’t finish</h2><p>{analysisErrors[error.code] || (error.status >= 500 ? 'The analysis server could not complete the request. Check the Go terminal for details.' : error.message)}</p><button type="button" className="property-button" onClick={error.status === 401 ? onSignIn : () => void analyse()} disabled={error.status !== 401 && !selected}>{error.status === 401 ? 'Sign in again' : 'Try again'}</button></div>}
      {result && <AnalysisResults data={result.data} areaName={result.areaName} />}
    </div>
  )
}
