import usePropertyData from '../hooks/usePropertyData'
import { money, number } from '../property'
import type { PortfolioData } from '../property'
import Icon from '../components/Icon'
import PropertyCard from '../components/PropertyCard'
import PropertyRequestState from '../components/PropertyRequestState'
import './Properties.css'
import './AddProperty.css'

export default function Portfolio({ onSignIn }: { onSignIn: () => void }) {
  const { data, error, retry } = usePropertyData<PortfolioData>('/User/Portfolio')
  return (
    <div className="properties-page">
      <div className="portfolio-heading">
        <header className="properties-heading">
          <p className="eyebrow">YOUR PROPERTY WORKSPACE</p>
          <h1>My portfolio<span>.</span></h1>
          <p>Your properties, with a little more perspective.</p>
        </header>
        <a className="property-button property-primary" href="#/portfolio/add">
          <Icon name="plus" />
          Add property
        </a>
      </div>
      {!data ? (
        <PropertyRequestState
          error={error}
          retry={retry}
          onSignIn={onSignIn}
          loading="Loading your portfolio…"
        />
      ) : <>
        <dl className="portfolio-summary">
          <div>
            <dt><Icon name="portfolio" />Properties in your portfolio</dt>
            <dd>{number(data.property_count)}</dd>
          </div>
          <div>
            <dt><Icon name="analysis" />Total estimated value</dt>
            <dd>{money(data.total_estimated_value, 0)}</dd>
          </div>
        </dl>
        <p className="portfolio-summary-note">
          The total uses available estimates × property area.
          The server does not separate sale and rental values in this total.
        </p>
        <div className="portfolio-list-heading">
          <h2>Your properties</h2>
          <span>{number(data.property_count)} {data.property_count === 1 ? 'property' : 'properties'}</span>
        </div>
        {data.properties.length === 0 ? (
          <section className="property-state">
            <Icon name="home" />
            <h2>A place for your properties</h2>
            <p>Your portfolio is empty. Add your first property to see its valuation here.</p>
          </section>
        ) : (
          <ul className="portfolio-list">
            {data.properties.map((property) => (
              <li key={property.uid}>
                <PropertyCard
                  property={property}
                  href={`#/portfolio/property?uid=${encodeURIComponent(property.uid)}`}
                />
              </li>
            ))}
          </ul>
        )}
      </>}
    </div>
  )
}
