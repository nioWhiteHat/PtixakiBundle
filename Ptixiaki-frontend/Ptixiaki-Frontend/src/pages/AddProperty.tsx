import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { addProperty, amenityFields, facadeFields, propertyInput, viewFields } from '../addProperty'
import { PropertyRequestError } from '../property'
import Icon from '../components/Icon'
import AreaSearch from '../components/AreaSearch'
import type { AreaSuggestion } from '../analysis'
import './Properties.css'
import './AddProperty.css'

const heatingTypes = [
  ['autonomous', 'Autonomous'],
  ['autonomous_central', 'Autonomous central'],
  ['autonomous_individual', 'Autonomous individual'],
  ['central_heating', 'Central heating'],
  ['no_heating', 'No heating'],
] as const
const heatingSources = [
  ['gas', 'Gas'], ['natural_gas', 'Natural gas'], ['oil', 'Oil'],
  ['electricity', 'Electricity'], ['heat_pump', 'Heat pump'], ['biomass', 'Biomass'],
  ['geothermal', 'Geothermal'], ['lpg', 'LPG'], ['other', 'Other'],
] as const

export default function AddProperty({ onSignIn }: { onSignIn: () => void }) {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const [transaction, setTransaction] = useState('sell')
  const [address, setAddress] = useState<AreaSuggestion | null>(null)
  const request = useRef<AbortController | null>(null)
  const currentYear = new Date().getFullYear()

  useEffect(() => () => request.current?.abort(), [])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (request.current) {
      return
    }
    setError(null)
    let input
    try {
      input = propertyInput(new FormData(event.currentTarget), address)
    } catch (error) {
      setError(error as Error)
      return
    }
    const controller = new AbortController()
    request.current = controller
    setPending(true)
    try {
      const uid = await addProperty(input, transaction, controller.signal)
      if (!controller.signal.aborted) {
        window.location.replace(`#/portfolio/property?uid=${encodeURIComponent(uid)}`)
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        setError(error as Error)
        if (error instanceof PropertyRequestError && error.code === 'address_selection_invalid') {
          setAddress(null)
        }
      }
    } finally {
      if (!controller.signal.aborted) {
        request.current = null
        setPending(false)
      }
    }
  }

  return (
    <div className="properties-page add-property-page">
      <a className="property-back" href="#/portfolio">
        <Icon name="arrow" />
        Back to portfolio
      </a>
      <header className="properties-heading">
        <p className="eyebrow">GROW YOUR PORTFOLIO</p>
        <h1>Add a property<span>.</span></h1>
        <p>Save a home and open its valuation and property details.</p>
      </header>
      <form className="add-property-form" onSubmit={(event) => void submit(event)} aria-label="Add property" aria-busy={pending}>
        <fieldset disabled={pending}>
          <legend>Property & location</legend>
          <div className="property-form-grid">
            <label>Transaction
              <select name="Transaction" value={transaction} onChange={(event) => setTransaction(event.target.value)}>
                <option value="sell">For sale</option>
                <option value="rent">For rent</option>
              </select>
            </label>
            <label>Property type
              <select name="SubType" defaultValue="apartment">
                <option value="apartment">Apartment</option>
                <option value="detached_house">Detached house</option>
                <option value="maisonette">Maisonette</option>
              </select>
            </label>
            <div className="property-form-wide">
              <AreaSearch
                kind="address"
                selected={address}
                disabled={pending}
                onSignIn={onSignIn}
                onSelect={(selection) => {
                  setAddress(selection)
                  setError(null)
                }}
              />
              {address && <p className="panel-note">Selected location: {address.lat}, {address.lon} (latitude, longitude).</p>}
              {address?.address?.road && !address.address.house_number && (
                <p className="panel-note">This is a street location. Include the house number for a more precise building match.</p>
              )}
            </div>
            <label>Area (m²)
              <input name="SquareMeters" type="number" required min={11} max={2_147_483_647} step={1} />
            </label>
            <label>Construction year
              <input name="ConstructionYear" type="number" required min={1900} max={currentYear} step={1} />
            </label>
            <label>Floors
              <input name="FloorLevel" type="text" required defaultValue="0" placeholder="e.g. 2 or 2, 3" />
              <span>0 is ground floor. Separate multiple floors with commas.</span>
            </label>
            <label>Renovation year (optional)
              <input name="RenovYear" type="number" min={1900} max={currentYear} step={1} />
              <span>Leave blank if the property has not been renovated.</span>
            </label>
            <label className="property-form-wide">{transaction === 'rent' ? 'Monthly asking price / m² (€, optional)' : 'Asking price / m² (€, optional)'}
              <input name="AskingPricePerSqm" type="number" min={0.01} step="any" />
              <span>Leave blank if you do not have an asking price.</span>
            </label>
          </div>
        </fieldset>
        <fieldset disabled={pending}>
          <legend>Rooms & building</legend>
          <div className="property-form-grid">
            <label>Bedrooms
              <input name="Bedrooms" type="number" min={0} max={2_147_483_647} step={1} defaultValue={0} required />
            </label>
            <label>Bathrooms / WC
              <input name="WC" type="number" min={0} max={2_147_483_647} step={1} defaultValue={0} required />
            </label>
            <label>Heating type
              <select name="HeatType" defaultValue="">
                <option value="">Not known</option>
                {heatingTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
              </select>
            </label>
            <label>Heating source
              <select name="HeatFrom" defaultValue="">
                <option value="">Not known</option>
                {heatingSources.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
              </select>
            </label>
            <label>Energy class
              <select name="EnergyPerformance" defaultValue="">
                <option value="">Not known</option>
                {['A+', 'A', 'B+', 'B', 'C', 'D', 'E', 'F', 'G'].map((value) => <option key={value} value={value}>{value}</option>)}
              </select>
            </label>
          </div>
        </fieldset>
        {[
          { title: 'Amenities', fields: amenityFields },
          { title: 'Views', fields: viewFields },
          { title: 'Facade', fields: facadeFields },
        ].map(({ title, fields }) => (
          <fieldset key={title} disabled={pending}>
            <legend>{title}</legend>
            <div className="property-form-checks">
              {fields.map(([name, label]) => (
                <label key={name}>
                  <input type="checkbox" name={name} />
                  {label}
                </label>
              ))}
            </div>
          </fieldset>
        ))}
        {error && (
          <div className="property-form-error" role="alert">
            <p>{error.message}</p>
            {error instanceof PropertyRequestError && error.status === 401 && (
              <button className="property-button" type="button" onClick={onSignIn}>Sign in again</button>
            )}
            <a href="#/portfolio">Open portfolio</a>
          </div>
        )}
        {pending && <p className="panel-note" role="status">Saving your property and preparing its valuation. This can take a minute.</p>}
        <div className="property-form-actions">
          <button className="property-button property-primary" type="submit" disabled={pending || !address || (error instanceof PropertyRequestError && error.status === 401)}>
            {pending ? 'Adding property…' : 'Add property'}
          </button>
          <a className="property-button" href="#/portfolio">Cancel</a>
        </div>
      </form>
    </div>
  )
}
