import type { PropertyOverview } from '../property'
import { money, number, signed } from '../property'
import Icon from './Icon'
import PropertyImage from './PropertyImage'

type PropertyCardProps = {
  property: PropertyOverview
  href: string
}

export default function PropertyCard({ property, href }: PropertyCardProps) {
  return (
    <a
      className="property-card"
      href={href}
      aria-label={`View property: ${property.address || property.uid}`}
    >
      <PropertyImage src={property.image} address={property.address || 'Property'} />
      <div className="property-card-body">
        <div className="property-card-heading">
          <h3>{property.address || 'Address unavailable'}</h3>
          <Icon name="arrow" />
        </div>
        <span className="property-area">
          {property.square_meters == null ? 'Area unavailable' : `${number(property.square_meters)} m²`}
        </span>
        <dl className="property-card-stats">
          <div>
            <dt>Estimation / m²</dt>
            <dd>{money(property.estimated_price_per_sqm)}</dd>
          </div>
          <div>
            <dt>Asking / m²</dt>
            <dd>{property.asking_price_per_sqm == null ? 'Not provided' : money(property.asking_price_per_sqm)}</dd>
          </div>
          <div>
            <dt>Elo</dt>
            <dd>{number(property.elo)}</dd>
          </div>
          <div>
            <dt>GeoPremium</dt>
            <dd>{signed(property.geo_premium)}</dd>
          </div>
        </dl>
      </div>
    </a>
  )
}
