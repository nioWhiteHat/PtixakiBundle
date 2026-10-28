import { clusterColor } from '../analysis'
import type { AnalysisPoint } from '../analysis'
import { number } from '../property'

export default function ClusterPlot({ points }: { points: AnalysisPoint[] }) {
  if (!points.length) return <p className="panel-note">No points to plot.</p>
  const plotted = points.filter((_, index) => index % Math.ceil(points.length / 2000) === 0)
  const maxArea = Math.max(1, ...plotted.map((point) => point.SquareMeters))
  const minAge = Math.min(0, ...plotted.map((point) => point.Age))
  const maxAge = Math.max(minAge + 1, ...plotted.map((point) => point.Age))
  return (
    <figure className="cluster-plot">
      <svg viewBox="0 0 760 340" role="img" aria-label="K-means clusters: property area against age">
        {[0, 1, 2, 3, 4].map((tick) => <g key={tick}>
          <path d={`M64 ${280 - tick * 58}H730`} stroke="#334554" strokeDasharray="3 5" />
          <text x="53" y={284 - tick * 58} textAnchor="end">{number(minAge + (maxAge - minAge) * tick / 4, 0)}</text>
          <text x={64 + tick * 166.5} y="303" textAnchor="middle">{number(maxArea * tick / 4, 0)}</text>
        </g>)}
        <path d="M64 35V280H730" fill="none" stroke="#718191" />
        {plotted.map((point) => <circle key={point.Uid} cx={64 + point.SquareMeters / maxArea * 666} cy={280 - (point.Age - minAge) / (maxAge - minAge) * 232} r="3.5" fill={clusterColor(point.ClusterID)} opacity="0.8"><title>{`${point.Uid} · Cluster ${point.ClusterID} · ${number(point.SquareMeters)} m² · Age ${point.Age}`}</title></circle>)}
        <text x="400" y="332" textAnchor="middle">Property area (m²)</text>
        <text transform="translate(16,160) rotate(-90)" textAnchor="middle">Property age (years)</text>
      </svg>
      <figcaption>{plotted.length < points.length ? `Plot shows ${number(plotted.length)} of ${number(points.length)} properties for performance. All properties remain available below.` : `${number(points.length)} properties shown.`} This is a two-dimensional view of four-feature clusters, not a geographic map.</figcaption>
    </figure>
  )
}
