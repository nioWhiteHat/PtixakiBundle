import usePropertyData from '../hooks/usePropertyData'
import { featureLabel, money, number, signed } from '../property'
import type { PropertyDetailData, PropertyFeatures } from '../property'
import PropertyImage from '../components/PropertyImage'
import PropertyExplanation from '../components/PropertyExplanation'
import PropertyRequestState from '../components/PropertyRequestState'
import Icon from '../components/Icon'
import './Properties.css'

const featureGroups = [
  { title: 'Property & building', keys: ['SubType', 'SquareMeters', 'ConstructionYear', 'RenovYear', 'IsRenovated', 'Bedrooms', 'WC', 'FloorLevel', 'HeatType', 'HeatFrom', 'EnergyPerformance'] },
  { title: 'Amenities', keys: ['AC', 'Elevator', 'Parking', 'StorageRoom', 'AluminumFrames', 'Pool', 'Garden'] },
  { title: 'Views & facade', keys: ['NoView', 'ViewForest', 'ViewMountain', 'ViewSea', 'ViewUnlimited', 'NoFacade', 'FacadeCorner', 'FacadeFrontage', 'FacadeInterior', 'FacadeThrough'] },
  { title: 'Location & stored data', keys: ['UID', 'Address', 'AdminName', 'GeoLat', 'GeoLon', 'ZoneValue', 'Distance'] },
]
const flags = new Set(['IsRenovated', ...featureGroups[1].keys, ...featureGroups[2].keys])

function featureValue(key: string, value: PropertyFeatures[string]) {
  if (value == null || value === '') {
    return 'Not recorded'
  }
  if (flags.has(key)) {
    return value === 1 ? 'Yes' : 'No'
  }
  if (Array.isArray(value)) {
    return value.length ? value.join(', ') : 'Not recorded'
  }
  if ((key === 'ConstructionYear' || key === 'RenovYear') && value === 0) {
    return 'Not recorded'
  }
  return typeof value === 'string' ? value.replaceAll('_', ' ') : String(value)
}

type PropertyDetailProps = {
  uid: string
  onSignIn: () => void
  source?: 'portfolio' | 'browse'
  backHref?: string
}

export default function PropertyDetail({
  uid,
  onSignIn,
  source = 'portfolio',
  backHref = '#/portfolio',
}: PropertyDetailProps) {
  const endpoint = source === 'browse' ? 'BrowseProperty' : 'ViewProperty'
  const { data, error, retry } = usePropertyData<PropertyDetailData>(`/User/${endpoint}?uid=${encodeURIComponent(uid)}`)
  return (
    <div className="properties-page property-detail-page">
      <a className="property-back" href={backHref}>
        <Icon name="arrow" />
        {source === 'browse' ? 'Back to results' : 'Back to portfolio'}
      </a>
      {!data ? (
        <PropertyRequestState
          error={error}
          retry={retry}
          onSignIn={onSignIn}
          loading="Loading property details…"
        />
      ) : <>
        <header className="properties-heading detail-heading">
          <div className="property-tags">
            <span className="property-tag">{data.transaction === 'rent' ? 'For rent' : 'For sale'}</span>
            <span>{featureValue('SubType', data.features.SubType)}</span>
          </div>
          <h1>{data.features.Address || 'Property details'}</h1>
          <p>{number(data.features.SquareMeters)} m² <span aria-hidden="true">·</span> {source === 'browse' ? 'Market listing' : 'Your portfolio'}</p>
        </header>
        <div className="property-hero">
          <PropertyImage src={data.image} address={data.features.Address || 'Property'} />
          <section className="property-valuation" aria-label="Property valuation">
            <span className="eyebrow">ESTIMATED PRICE</span>
            <p className="estimate-value">{money(data.estimated_price_per_sqm)}<span> / m²</span></p>
            <p className="estimate-total">{money(data.estimated_total_price, 0)} <span>total estimate</span></p>
            <dl className="valuation-stats">
              <div>
                <dt>Asking / m²</dt>
                <dd>{data.asking_price_per_sqm == null ? 'Not provided' : money(data.asking_price_per_sqm)}</dd>
              </div>
              {data.asking_total_price != null && (
                <div>
                  <dt>Total asking</dt>
                  <dd>{money(data.asking_total_price, 0)}</dd>
                </div>
              )}
              <div>
                <dt>Elo</dt>
                <dd>{number(data.elo)}</dd>
              </div>
              <div>
                <dt>GeoPremium</dt>
                <dd>{signed(data.geo_premium)}</dd>
              </div>
              <div>
                <dt>GeoPremium z-score</dt>
                <dd>{number(data.geo_premium_zscore, 3)}</dd>
              </div>
              <div>
                <dt>Price residual</dt>
                <dd>{signed(data.residual_percentage)}</dd>
              </div>
            </dl>
            {data.asking_price_per_sqm == null && (
              <p className="panel-note">
                No asking price was supplied. Elo uses location information without the price-deal terms.
              </p>
            )}
          </section>
        </div>

        <section className="property-panel" aria-labelledby="features-title">
          <div className="panel-heading">
            <span className="section-number">01</span>
            <div>
              <h2 id="features-title">The property, in detail</h2>
              <p>Features and location saved for this home</p>
            </div>
          </div>
          {featureGroups.map((group) => (
            <div className="feature-group" key={group.title}>
              <h3>{group.title}</h3>
              <dl className="property-features">
                {group.keys.map((key) => (
                  <div key={key}>
                    <dt>{featureLabel(key)}</dt>
                    <dd>{key === 'UID' ? data.uid : featureValue(key, data.features[key])}</dd>
                  </div>
                ))}
              </dl>
            </div>
          ))}
          <p className="panel-note">Shown as stored by the backend. Older records can represent missing values as 0 or No; these do not necessarily mean a feature was explicitly ruled out.</p>
        </section>
        <PropertyExplanation property={data} />
      </>}
    </div>
  )
}
