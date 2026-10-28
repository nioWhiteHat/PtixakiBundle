export default function PropertyIllustration() {
  return (
    <div className="property-illustration" aria-hidden="true">
      <svg viewBox="0 0 520 340" fill="none">
        <defs>
          <linearGradient id="building-front" x1="200" y1="60" x2="320" y2="300" gradientUnits="userSpaceOnUse">
            <stop stopColor="#65798c" stopOpacity=".7" /><stop offset="1" stopColor="#253a4b" stopOpacity=".8" />
          </linearGradient>
          <linearGradient id="building-side" x1="290" y1="60" x2="400" y2="280" gradientUnits="userSpaceOnUse">
            <stop stopColor="#344b60" /><stop offset="1" stopColor="#1b2b3a" />
          </linearGradient>
          <linearGradient id="copper-glass" x1="0" y1="0" x2="1" y2="1">
            <stop stopColor="#e3b28d" stopOpacity=".55" /><stop offset="1" stopColor="#e3b28d" stopOpacity=".1" />
          </linearGradient>
        </defs>
        <g stroke="#7692a8" strokeOpacity=".15" strokeWidth=".8">
          {[0, 1, 2, 3, 4, 5, 6].map((i) => <path key={`a${i}`} d={`M${12 + i * 49} ${218 - i * 22}l218 100`} />)}
          {[0, 1, 2, 3, 4, 5, 6].map((i) => <path key={`b${i}`} d={`M${30 + i * 43} ${228 + i * 16}l228-111`} />)}
        </g>
        <path d="m71 226 187-91 198 95-189 92L71 226Z" fill="#152632" fillOpacity=".6" stroke="#9bb5c8" strokeOpacity=".22" />
        <g stroke="#7290a5" strokeWidth="1">
          <path d="m98 160 52-25 50 24-52 25-50-24Z" fill="#364d5d" />
          <path d="m98 160 50 24v83l-50-24v-83Z" fill="#263b4b" />
          <path d="m148 184 52-25v84l-52 24v-83Z" fill="#1d3040" />
          {[0, 1, 2].map((i) => <path key={i} d={`m107 ${177 + i * 23} 31 15m21 2 31-15`} strokeOpacity=".65" />)}
          <path d="M122 172v79m51-78v79" strokeOpacity=".4" />
        </g>
        <g stroke="#90a9bb" strokeWidth="1.2" strokeLinejoin="round">
          <path d="m201 81 89-43 90 43-89 44-90-44Z" fill="#567084" />
          <path d="m213 81 77-36 77 36-76 37-78-37Z" fill="#2d4254" strokeOpacity=".65" />
          <path d="m201 81 90 44v179l-90-43V81Z" fill="url(#building-front)" />
          <path d="m291 125 89-44v178l-89 45V125Z" fill="url(#building-side)" />
          {[0, 1, 2, 3, 4].map((i) => <path key={i} d={`m202 ${111 + i * 31} 89 43 88-43`} strokeOpacity=".5" />)}
          <path d="M223 92v179m23-168v179m23-168v179m45-179v179m22-191v180m22-191v180" strokeOpacity=".45" />
          <path d="m224 155 43 21v29l-43-21v-29Z" fill="url(#copper-glass)" stroke="#e3b28d" />
          <path d="M246 166v29" stroke="#e3b28d" />
          <path d="m314 237 21-10v29l-21 11v-30Z" fill="#99c1ce" fillOpacity=".17" />
        </g>
        <g stroke="#dfad88" strokeWidth="1">
          <path d="M245 181h-82l-32-38H44" strokeDasharray="3 5" strokeOpacity=".7" />
          <circle cx="245" cy="181" r="4" fill="#dfad88" stroke="#203040" strokeWidth="3" />
          <path d="M400 83v176m-4-174 8-4m-8 180 8-4" strokeOpacity=".55" />
        </g>
        <g transform="translate(47 55)" stroke="#93a9b7" strokeOpacity=".6">
          <circle r="18" strokeOpacity=".25" /><path d="M0-24v10M0 14v10M-24 0h10M14 0h10" />
          <path d="m0-10 5 15-5-3-5 3 5-15Z" fill="#93a9b7" fillOpacity=".3" />
        </g>
        <text x="43" y="129" fill="#e3b28d" fontSize="9" letterSpacing="2">A NEW PERSPECTIVE</text>
        <text x="418" y="183" fill="#8ba1b2" fontSize="9" letterSpacing="1.5" transform="rotate(90 418 183)">PROPERTY CO PILOT</text>
      </svg>
    </div>
  )
}
