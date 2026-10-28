import { useState } from 'react'
import type { FormEvent } from 'react'
import type { AreaSuggestion } from '../analysis'
import type { PropertyOverview } from '../property'
import { areaBounds } from '../analysis'
import usePropertyData from '../hooks/usePropertyData'
import AreaSearch from '../components/AreaSearch'
import PropertyCard from '../components/PropertyCard'
import PropertyRequestState from '../components/PropertyRequestState'
import './Properties.css'
import './Browse.css'

type BrowseData = {
  index: number
  page_size: number
  has_next: boolean
  properties: PropertyOverview[]
}

type BrowseProps = {
  query: string
  onSignIn: () => void
}

function savedArea(params: URLSearchParams): AreaSuggestion | null {
  const name = params.get('area_name')
  const bounds = ['min_lat', 'max_lat', 'min_lon', 'max_lon'].map((key) => params.get(key))

  if (!name || bounds.some((value) => value === null)) {
    return null
  }

  return {
    display_name: name,
    boundingbox: bounds as string[],
    lat: '',
    lon: '',
  }
}

export default function Browse({ query, onSignIn }: BrowseProps) {
  const params = new URLSearchParams(query)
  const [area, setArea] = useState(() => savedArea(params))
  const [areaQuery, setAreaQuery] = useState(area?.display_name ?? '')
  const [searchKey, setSearchKey] = useState(0)
  const [filterError, setFilterError] = useState('')
  const requestParams = new URLSearchParams(params)
  requestParams.delete('area_name')
  requestParams.set('transaction', params.get('transaction') || 'sell')
  requestParams.set('index', params.get('index') || '1')

  const url = `/User/BrowseProperties?${requestParams}`
  const { data, error, retry } = usePropertyData<BrowseData>(url)
  const page = data?.index ?? Number(requestParams.get('index'))
  const transactionLabel = requestParams.get('transaction') === 'rent' ? 'For rent' : 'For sale'

  function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFilterError('')

    if (areaQuery.trim() && !area) {
      setFilterError('Choose an area from the suggestions, or clear the area to search everywhere.')
      return
    }

    const fields = new FormData(event.currentTarget)
    const next = new URLSearchParams()
    next.set('transaction', String(fields.get('transaction')))
    next.set('index', '1')

    for (const name of ['asking_price_per_sqm', 'square_meters']) {
      const minimum = String(fields.get(`${name}_min`) ?? '').trim()
      const maximum = String(fields.get(`${name}_max`) ?? '').trim()

      if (minimum && maximum && Number(minimum) > Number(maximum)) {
        setFilterError('The minimum must not be greater than the maximum.')
        return
      }

      if (minimum) {
        next.set(`${name}_min`, minimum)
      }
      if (maximum) {
        next.set(`${name}_max`, maximum)
      }
    }

    if (area) {
      const bounds = areaBounds(area)
      next.set('min_lat', String(bounds.MinLat))
      next.set('max_lat', String(bounds.MaxLat))
      next.set('min_lon', String(bounds.MinLon))
      next.set('max_lon', String(bounds.MaxLon))
      next.set('area_name', area.display_name)
    }

    window.location.hash = `/browse?${next}`
  }

  function resetFilters() {
    setArea(null)
    setAreaQuery('')
    setFilterError('')
    setSearchKey((value) => value + 1)
    window.location.hash = '/browse'
  }

  function changePage(nextPage: number) {
    const next = new URLSearchParams(params)
    next.set('index', String(nextPage))
    window.location.hash = `/browse?${next}`
  }

  function propertyLink(uid: string) {
    const next = new URLSearchParams(params)
    next.set('uid', uid)
    return `#/browse/property?${next}`
  }

  return (
    <div className="properties-page browse-page">
      <header className="properties-heading">
        <p className="eyebrow">FIND YOUR NEXT MOVE</p>
        <h1>Browse properties<span>.</span></h1>
        <p>A clearer look at what’s out there. Explore the homes and the numbers behind them.</p>
      </header>

      <form className="browse-filters" aria-label="Property filters" onSubmit={applyFilters} onReset={resetFilters}>
        <div className="browse-location">
          <AreaSearch
            key={searchKey}
            selected={area}
            onSelect={setArea}
            onQueryChange={setAreaQuery}
            disabled={false}
            onSignIn={onSignIn}
          />
          <label className="browse-field">
            Transaction type
            <select name="transaction" defaultValue={params.get('transaction') || 'sell'}>
              <option value="sell">For sale</option>
              <option value="rent">For rent</option>
            </select>
          </label>
        </div>

        <div className="browse-ranges">
          <fieldset>
            <legend>Asking price (€/m²)</legend>
            <label className="browse-field">
              Minimum asking / m²
              <input
                type="number"
                name="asking_price_per_sqm_min"
                min="0"
                step="0.01"
                placeholder="No minimum"
                defaultValue={params.get('asking_price_per_sqm_min') || ''}
              />
            </label>
            <label className="browse-field">
              Maximum asking / m²
              <input
                type="number"
                name="asking_price_per_sqm_max"
                min="0"
                step="0.01"
                placeholder="No maximum"
                defaultValue={params.get('asking_price_per_sqm_max') || ''}
              />
            </label>
          </fieldset>
          <fieldset>
            <legend>Property size (m²)</legend>
            <label className="browse-field">
              Minimum size
              <input
                type="number"
                name="square_meters_min"
                min="0"
                step="1"
                placeholder="No minimum"
                defaultValue={params.get('square_meters_min') || ''}
              />
            </label>
            <label className="browse-field">
              Maximum size
              <input
                type="number"
                name="square_meters_max"
                min="0"
                step="1"
                placeholder="No maximum"
                defaultValue={params.get('square_meters_max') || ''}
              />
            </label>
          </fieldset>
        </div>

        {filterError && <p className="area-error" role="alert">{filterError}</p>}
        <div className="browse-filter-actions">
          <p className="panel-note">Leave the area blank to search all areas. Rental prices are monthly.</p>
          <button type="reset" className="property-button">Clear filters</button>
          <button type="submit" className="property-button browse-search-button">Search properties</button>
        </div>
      </form>

      <section aria-label="Search results">
        <div className="portfolio-list-heading browse-results-heading">
          <div>
            <h2>{params.get('area_name') || 'All areas'}</h2>
            <p className="panel-note">{transactionLabel} · Highest Elo first</p>
          </div>
          {data && <span>{data.properties.length} on this page</span>}
        </div>

        {!data && (
          <PropertyRequestState error={error} retry={retry} onSignIn={onSignIn} loading="Loading properties…" />
        )}
        {data && data.properties.length === 0 && (
          <div className="property-state" role="status">
            <h2>No properties found</h2>
            <p>Try a wider area or different price and size limits.</p>
          </div>
        )}
        {data && data.properties.length > 0 && (
          <div className="browse-results-scroll" role="region" aria-label="Property listings" tabIndex={0}>
            <ul className="portfolio-list">
              {data.properties.map((property) => (
                <li key={property.uid}>
                  <PropertyCard property={property} href={propertyLink(property.uid)} />
                </li>
              ))}
            </ul>
          </div>
        )}

        <nav className="browse-pagination" aria-label="Property pages">
          <button
            type="button"
            className="property-button"
            disabled={page <= 1}
            onClick={() => changePage(page - 1)}
          >
            Previous
          </button>
          <span aria-live="polite">Page {page} · 20 per page</span>
          <button
            type="button"
            className="property-button"
            disabled={!data?.has_next}
            onClick={() => changePage(page + 1)}
          >
            Next
          </button>
        </nav>
      </section>
      <p className="panel-note browse-attribution">Area boundaries: © OpenStreetMap contributors</p>
    </div>
  )
}
