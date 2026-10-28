import { PropertyRequestError } from './property'
import type { AreaSuggestion } from './analysis'

export type PropertyInput = {
  SubType: string
  Address: string
  AddressOSMID: number
  AddressOSMType: string
  SquareMeters: number
  ConstructionYear: number
  AskingPricePerSqm?: number
  RenovYear: number
  IsRenovated: number
  FloorLevel: number[]
  Bedrooms: number
  WC: number
  HeatType: string
  HeatFrom: string
  EnergyPerformance: string
  [key: string]: string | number | number[] | undefined
}

export const amenityFields = [
  ['AC', 'Air conditioning'],
  ['Elevator', 'Elevator'],
  ['Parking', 'Parking'],
  ['StorageRoom', 'Storage room'],
  ['AluminumFrames', 'Aluminium frames'],
  ['Pool', 'Pool'],
  ['Garden', 'Garden'],
] as const

export const viewFields = [
  ['ViewForest', 'Forest'],
  ['ViewMountain', 'Mountain'],
  ['ViewSea', 'Sea'],
  ['ViewUnlimited', 'Open view'],
] as const

export const facadeFields = [
  ['FacadeCorner', 'Corner'],
  ['FacadeFrontage', 'Frontage'],
  ['FacadeInterior', 'Interior'],
  ['FacadeThrough', 'Through'],
] as const

export function propertyInput(form: FormData, selected: AreaSuggestion | null): PropertyInput {
  const value = (key: string) => String(form.get(key) ?? '').trim()
  const numeric = (key: string) => Number(value(key))
  const currentYear = new Date().getFullYear()
  const address = value('Address')
  const area = numeric('SquareMeters')
  const year = numeric('ConstructionYear')
  const renovation = value('RenovYear')
  const asking = value('AskingPricePerSqm')
  const floors = value('FloorLevel').split(',').map((floor) => floor.trim())

  if (!selected || address !== selected.display_name || !selected.osm_id || !selected.osm_type) {
    throw new Error('Choose a matching address from the suggestions before saving.')
  }
  if (!selected.lat || !selected.lon || !Number.isFinite(Number(selected.lat)) || Math.abs(Number(selected.lat)) > 90 || !Number.isFinite(Number(selected.lon)) || Math.abs(Number(selected.lon)) > 180) {
    throw new Error('This address has invalid coordinates. Choose another suggestion.')
  }
  if (!address || [...address].length > 255) {
    throw new Error('Enter an address between 1 and 255 characters.')
  }
  if (!Number.isInteger(area) || area <= 10 || area > 2_147_483_647) {
    throw new Error('Enter a whole property area greater than 10 m².')
  }
  if (!Number.isInteger(year) || year < 1900 || year > currentYear) {
    throw new Error(`Enter a construction year between 1900 and ${currentYear}.`)
  }
  if (renovation && (!Number.isInteger(Number(renovation)) || Number(renovation) < year || Number(renovation) > currentYear)) {
    throw new Error('The renovation year must be between the construction year and the current year.')
  }
  if (asking && (!Number.isFinite(Number(asking)) || Number(asking) <= 0 || Number(asking) * area < 1 || Number(asking) * area > 2_147_483_647)) {
    throw new Error('Enter a positive asking price with a total between €1 and €2,147,483,647, or leave it blank.')
  }
  if (floors.some((floor) => !/^-?\d+$/.test(floor) || !Number.isSafeInteger(Number(floor)))) {
    throw new Error('Enter whole floor numbers separated by commas, such as 0 or 2, 3.')
  }
  for (const key of ['Bedrooms', 'WC']) {
    if (!Number.isInteger(numeric(key)) || numeric(key) < 0 || numeric(key) > 2_147_483_647) {
      throw new Error('Bedrooms and bathrooms must be non-negative whole numbers.')
    }
  }
  if (!['apartment', 'detached_house', 'maisonette'].includes(value('SubType'))) {
    throw new Error('Choose a supported property type.')
  }

  const input: PropertyInput = {
    SubType: value('SubType'),
    Address: selected.display_name,
    AddressOSMID: selected.osm_id,
    AddressOSMType: selected.osm_type,
    SquareMeters: area,
    ConstructionYear: year,
    RenovYear: renovation ? Number(renovation) : 0,
    IsRenovated: renovation ? 1 : 0,
    FloorLevel: floors.map(Number),
    Bedrooms: numeric('Bedrooms'),
    WC: numeric('WC'),
    HeatType: value('HeatType'),
    HeatFrom: value('HeatFrom'),
    EnergyPerformance: value('EnergyPerformance'),
  }
  if (asking) {
    input.AskingPricePerSqm = Number(asking)
  }
  for (const [key] of [...amenityFields, ...viewFields, ...facadeFields]) {
    input[key] = form.has(key) ? 1 : 0
  }
  input.NoView = viewFields.some(([key]) => input[key] === 1) ? 0 : 1
  input.NoFacade = facadeFields.some(([key]) => input[key] === 1) ? 0 : 1
  return input
}

export async function addProperty(input: PropertyInput, transaction: string, signal: AbortSignal): Promise<string> {
  const timeout = AbortSignal.timeout(120_000)
  try {
    const response = await fetch(`/User/AddProperty?transaction=${encodeURIComponent(transaction)}`, {
      method: 'POST',
      credentials: 'include',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
      signal: AbortSignal.any([signal, timeout]),
    })
    if (!response.ok) {
      const body = response.headers.get('Content-Type')?.includes('application/json')
        ? await response.json().catch(() => null)
        : null
      const code = typeof body?.error?.code === 'string' ? body.error.code : ''
      if (code === 'address_selection_required' || code === 'address_selection_invalid') {
        throw new PropertyRequestError('Choose the address again from the suggestions. Its Nominatim location could not be verified.', response.status, code)
      }
      const message = response.status === 401
        ? 'Your session has expired. Please sign in again.'
        : response.status === 400
          ? 'Check the property details and try again.'
          : 'The server couldn’t finish adding this property. Check your portfolio before trying again; it may already have been saved.'
      throw new PropertyRequestError(message, response.status)
    }
    const result = await response.json() as { uid?: unknown }
    if (typeof result.uid !== 'string' || !result.uid.trim()) {
      throw new PropertyRequestError('The server saved the property but did not return its ID. Open your portfolio to find it.')
    }
    return result.uid
  } catch (error) {
    if (signal.aborted || error instanceof PropertyRequestError) {
      throw error
    }
    throw new PropertyRequestError(timeout.aborted
      ? 'Saving took too long. Check your portfolio before trying again; the property may already have been saved.'
      : 'Couldn’t confirm the save. Check your connection and portfolio before trying again.')
  }
}
