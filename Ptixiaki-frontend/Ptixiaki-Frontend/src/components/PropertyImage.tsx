import { useState } from 'react'

export default function PropertyImage({ src, address }: { src?: string | null; address: string }) {
  const [failedSource, setFailedSource] = useState<string | null>(null)
  return (
    <div className="property-image">
      {src && src !== failedSource ? (
        <img src={src} alt={address} loading="lazy" referrerPolicy="no-referrer" onError={() => setFailedSource(src)} />
      ) : (
        <div className="property-image-placeholder" role="img" aria-label="No property image available">
          <svg viewBox="0 0 160 100" fill="none" aria-hidden="true">
            <path d="M16 82h128M33 82V46l33-25 32 25v36M92 42l16-12 23 19v33M59 82V61h16v21M44 47h12v10H44M77 47h10v10H77M107 52h12v13h-12" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
            <circle cx="126" cy="21" r="7" stroke="currentColor" />
          </svg>
          <span>No photo available</span>
        </div>
      )}
    </div>
  )
}
