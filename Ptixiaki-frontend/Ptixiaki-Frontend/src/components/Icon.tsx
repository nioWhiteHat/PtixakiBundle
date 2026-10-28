type IconName = 'mail' | 'lock' | 'eye' | 'eye-off' | 'arrow' | 'check' | 'home' | 'menu' | 'close' | 'browse' | 'portfolio' | 'analysis' | 'plus'

export default function Icon({ name, className = '' }: { name: IconName; className?: string }) {
  return (
    <svg className={`icon ${className}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {name === 'mail' && <><rect x="3" y="5" width="18" height="14" rx="3" /><path d="m4 7 8 6 8-6" /></>}
      {name === 'lock' && <><rect x="5" y="10" width="14" height="11" rx="3" /><path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3" /></>}
      {(name === 'eye' || name === 'eye-off') && <><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" /><circle cx="12" cy="12" r="3" />{name === 'eye-off' && <path d="m3 3 18 18" />}</>}
      {name === 'arrow' && <path d="M4 12h16m-6-6 6 6-6 6" />}
      {name === 'plus' && <path d="M12 5v14M5 12h14" />}
      {name === 'check' && <path d="m5 12 4 4L19 6" />}
      {name === 'home' && <><path d="m3 11 9-8 9 8M5 9v12h14V9M9 21v-7h6v7" /><path d="M17 3h4v4" /></>}
      {name === 'menu' && <path d="M4 6h16M4 12h16M4 18h16" />}
      {name === 'close' && <path d="m6 6 12 12M6 18 18 6" />}
      {name === 'browse' && <><rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" /><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" /></>}
      {name === 'portfolio' && <><rect x="3" y="7" width="18" height="14" rx="2" /><path d="M8 7V4h8v3M3 12l9 3 9-3M12 12v5" /></>}
      {name === 'analysis' && <path d="M4 3v18h17M8 16v-5M13 16V7M18 16v-3" />}
    </svg>
  )
}
