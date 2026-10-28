import { useEffect, useRef, useState } from 'react'
import type { AuthMode } from './auth'
import AuthForm from './components/AuthForm'
import Icon from './components/Icon'
import PropertyIllustration from './components/PropertyIllustration'
import Workspace from './components/Workspace'
import './App.css'

export default function App() {
  const [mode, setMode] = useState<AuthMode>('signin')
  const [signedIn, setSignedIn] = useState(false)
  const formPanel = useRef<HTMLElement>(null)
  const switched = useRef(false)

  useEffect(() => {
    if (signedIn) return
    document.title = `${mode === 'signin' ? 'Sign in' : 'Create account'} · Property Co Pilot`
    if (switched.current) formPanel.current?.querySelector<HTMLInputElement>('input')?.focus()
  }, [mode, signedIn])

  function switchMode() {
    switched.current = true
    setMode((current) => current === 'signin' ? 'signup' : 'signin')
  }

  if (signedIn) return <Workspace onSignIn={() => { setMode('signin'); setSignedIn(false) }} />

  return (
    <div className="auth-page">
      <header className="site-header">
        <div className="brand" aria-label="Property Co Pilot">
          <span className="brand-mark"><Icon name="home" /></span>
          <span>Property <strong>Co Pilot</strong><span className="brand-dot">.</span></span>
        </div>
        <span className="header-note">A little insight. A better next move.</span>
      </header>
      <main className={`auth-card ${mode === 'signup' ? 'is-signup' : ''}`}>
        <aside className="story-panel" aria-label="About Property Co Pilot">
          <div className="story-intro">
            <span className="story-tag"><span /> PROPERTY, WITH PERSPECTIVE</span>
            <h2>Good decisions<br />{' '}start with <em>perspective.</em></h2>
            <p>A place for your properties.<br />A clearer view of the possibilities.</p>
          </div>
          <PropertyIllustration />
          <div className="story-footer"><span className="story-footer-line" /><p>Look closer. See further.</p><span className="edition">01 / A NEW VIEW</span></div>
        </aside>
        <section className="form-panel" ref={formPanel} aria-label={mode === 'signup' ? 'Sign up' : 'Sign in'}>
          <AuthForm key={mode} mode={mode} onSwitch={switchMode} onAuthenticated={() => setSignedIn(true)} />
          <div className="form-footnote"><Icon name="lock" /><span>Your space. Your next chapter.</span></div>
        </section>
      </main>
      <footer className="site-footer"><span>© {new Date().getFullYear()} Property Co Pilot</span><span>PROPERTY INSIGHT, MADE PERSONAL.</span></footer>
    </div>
  )
}
