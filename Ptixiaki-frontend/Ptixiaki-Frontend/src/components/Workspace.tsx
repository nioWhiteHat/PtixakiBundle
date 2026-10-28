import { useEffect, useRef, useState, useSyncExternalStore } from 'react'
import Icon from './Icon'
import Portfolio from '../pages/Portfolio'
import AddProperty from '../pages/AddProperty'
import PropertyDetail from '../pages/PropertyDetail'
import Analysis from '../pages/Analysis'
import Browse from '../pages/Browse'
import './Workspace.css'

const sections = [
  { label: 'Browse', icon: 'browse' },
  { label: 'Portfolio', icon: 'portfolio' },
  { label: 'Analysis', icon: 'analysis' },
] as const

function subscribeToNavigation(onChange: () => void) {
  window.addEventListener('hashchange', onChange)
  return () => window.removeEventListener('hashchange', onChange)
}

export default function Workspace({ onSignIn = () => {} }: { onSignIn?: () => void }) {
  const hash = useSyncExternalStore(subscribeToNavigation, () => window.location.hash)
  const [path, query = ''] = hash.slice(1).split('?')
  const isAddProperty = path === '/portfolio/add'
  const isPortfolio = path === '/portfolio' || path === '/portfolio/property' || isAddProperty
  const isProperty = path === '/portfolio/property' || path === '/browse/property'
  const active = isPortfolio ? 'Portfolio' : path === '/analysis' ? 'Analysis' : 'Browse'
  const parameters = new URLSearchParams(query)
  const uid = isProperty ? parameters.get('uid') : null
  const pageTitle = uid ? 'Property details' : isAddProperty ? 'Add property' : active
  parameters.delete('uid')
  const browseBackHref = parameters.size ? `#/browse?${parameters}` : '#/browse'
  const [open, setOpen] = useState(false)
  const toggle = useRef<HTMLButtonElement>(null)
  const main = useRef<HTMLElement>(null)

  useEffect(() => {
    toggle.current?.focus()
  }, [])

  useEffect(() => {
    document.title = `${pageTitle} · Property Co Pilot`
  }, [pageTitle])

  useEffect(() => {
    if (hash) {
      main.current?.focus()
      window.scrollTo(0, 0)
    }
  }, [hash])

  function closeMenu() {
    setOpen(false)
    toggle.current?.focus()
  }

  return (
    <div
      className="workspace"
      onKeyDown={(event) => {
        if (open && event.key === 'Escape') {
          closeMenu()
        }
      }}
    >
      <header className="workspace-header">
        <button
          ref={toggle}
          type="button"
          className="navigation-toggle"
          aria-label={open ? 'Close navigation' : 'Open navigation'}
          aria-expanded={open}
          aria-controls="workspace-navigation"
          onClick={() => setOpen(!open)}
        >
          <Icon name={open ? 'close' : 'menu'} />
        </button>
        <div className="brand" aria-label="Property Co Pilot">
          <span>Property <strong>Co Pilot</strong><span className="brand-dot">.</span></span>
        </div>
      </header>

      {open && (
        <button
          type="button"
          className="navigation-backdrop"
          aria-label="Dismiss navigation"
          tabIndex={-1}
          onClick={closeMenu}
        />
      )}

      <nav id="workspace-navigation" className="workspace-sidebar" aria-label="Main navigation" hidden={!open}>
        <p className="navigation-caption">YOUR WORKSPACE</p>
        <div className="navigation-items">
          {sections.map(({ label, icon }) => (
            <a
              key={label}
              href={`#/${label.toLowerCase()}`}
              className="navigation-item"
              aria-current={active === label ? 'page' : undefined}
              onClick={closeMenu}
            >
              <Icon name={icon} />
              <span>{label}</span>
              {active === label && <span className="navigation-dot" aria-hidden="true" />}
            </a>
          ))}
        </div>
        <p className="navigation-footer">A clearer view.<br /><span>A better next move.</span></p>
      </nav>

      <main
        ref={main}
        tabIndex={-1}
        className="workspace-content"
        aria-label={pageTitle}
        inert={open}
      >
        {active === 'Portfolio' && (isAddProperty ? (
          <AddProperty onSignIn={onSignIn} />
        ) : uid ? (
          <PropertyDetail key={uid} uid={uid} onSignIn={onSignIn} />
        ) : (
          <Portfolio onSignIn={onSignIn} />
        ))}
        {active === 'Analysis' && <Analysis onSignIn={onSignIn} />}
        {active === 'Browse' && (uid ? (
          <PropertyDetail
            key={uid}
            uid={uid}
            source="browse"
            backHref={browseBackHref}
            onSignIn={onSignIn}
          />
        ) : (
          <Browse key={query} query={query} onSignIn={onSignIn} />
        ))}
      </main>
    </div>
  )
}
