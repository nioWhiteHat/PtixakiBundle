export type PropertyOverview = {
  uid: string
  address: string | null
  square_meters: number | null
  estimated_price_per_sqm: number | null
  asking_price_per_sqm?: number | null
  elo: number | null
  geo_premium: number | null
  image?: string | null
}

export type PortfolioData = {
  property_count: number
  total_estimated_value: number
  properties: PropertyOverview[]
}

export type PropertyFeatures = {
  UID: string
  Address: string
  SubType: string
  SquareMeters: number
  [name: string]: string | number | number[] | null | undefined
}

export type Contribution = {
  feature_name: string
  impact_per_sqm: number
  local_importance_percentage: number
}

export type GlobalImportance = {
  feature_name: string
  mean_abs_shap_per_sqm: number
  mean_signed_shap_per_sqm: number
  importance_percentage: number
  rank: number
  model_sha256: string
  calculated_at: string
}

export type PropertyDetailData = Omit<PropertyOverview, 'address' | 'square_meters'> & {
  transaction: string
  features: PropertyFeatures
  estimated_total_price: number | null
  asking_total_price?: number | null
  geo_premium_zscore: number | null
  residual_percentage: number | null
  explanation: {
    baseline_price_per_sqm: number
    explained_prediction_per_sqm: number
    additivity_error: number
    model_sha256: string
    explained_at: string
    matches_importance_model: boolean | null
    contributions: Contribution[]
  } | null
  feature_importance: GlobalImportance[]
}

export class PropertyRequestError extends Error {
  constructor(message: string, public status = 0, public code = '') { super(message) }
}

export async function getPropertyData<T>(url: string, signal: AbortSignal, options: { json?: unknown; timeoutMs?: number } = {}): Promise<T> {
  const timeout = AbortSignal.timeout(options.timeoutMs ?? 20_000)
  try {
    const response = await fetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json', ...(options.json !== undefined ? { 'Content-Type': 'application/json' } : {}) },
      ...(options.json !== undefined ? { method: 'POST', body: JSON.stringify(options.json) } : {}),
      signal: AbortSignal.any([signal, timeout]),
    })
    if (!response.ok) {
      const body = response.headers.get('Content-Type')?.includes('application/json') ? await response.json().catch(() => null) : null
      const code = typeof body?.error?.code === 'string' ? body.error.code : ''
      const message = response.status === 401
        ? 'Your session has expired. Please sign in again.'
        : response.status === 404
          ? 'This property is no longer available or you do not have access to it.'
          : 'We couldn’t load your property data. Please try again.'
      throw new PropertyRequestError(message, response.status, code)
    }
    return await response.json() as T
  } catch (error) {
    if (signal.aborted || error instanceof PropertyRequestError) throw error
    throw new PropertyRequestError(timeout.aborted
      ? 'The request timed out. Please try again.'
      : 'Couldn’t load property data. Check your connection and that the Go server is running.')
  }
}

export const number = (value: number | null | undefined, digits = 0) => value == null
  ? '—'
  : new Intl.NumberFormat('en-IE', { maximumFractionDigits: digits }).format(value)
export const money = (value: number | null | undefined, digits = 2) => value == null
  ? '—'
  : new Intl.NumberFormat('en-IE', { style: 'currency', currency: 'EUR', maximumFractionDigits: digits, minimumFractionDigits: digits }).format(value)
export const signed = (value: number | null | undefined, suffix = '%') => value == null
  ? '—'
  : `${value > 0 ? '+' : ''}${number(value, 2)}${suffix}`

const labels: Record<string, string> = {
  UID: 'Property UID', SubType: 'Property type', AdminName: 'Administrative area',
  SquareMeters: 'Area (m²)', AskingPricePerSqm: 'Asking price / m²',
  RenovYear: 'Renovation year', IsRenovated: 'Renovated', HeatType: 'Heating type',
  HeatFrom: 'Heating source', EnergyPerformance: 'Energy class', WC: 'Bathrooms / WC',
  AC: 'Air conditioning', GeoLat: 'Latitude', GeoLon: 'Longitude',
  NoView: 'No view', NoFacade: 'No facade', ZoneValue: 'Zone value',
  Age: 'Property age', RenovAge: 'Years since renovation', GeoPremium: 'GeoPremium',
  Distance: 'Distance (legacy field)',
}
export const featureLabel = (name: string) => labels[name] || name.replace(/([a-z])([A-Z])/g, '$1 $2').replaceAll('_', ' ')
