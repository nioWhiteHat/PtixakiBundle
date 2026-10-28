import { useState } from 'react'
import { number } from '../property'
import type { AnalysisPoint } from '../analysis'

const columns: [keyof AnalysisPoint, string][] = [
  ['Uid', 'Property UID'], ['ClusterID', 'Cluster'], ['TransactionType', 'Transaction'], ['SquareMeters', 'Area (m²)'],
  ['Age', 'Age'], ['RenovAge', 'Renovation age'], ['FloorLevel', 'Floor'], ['AskingPricePerSqm', 'Asking €/m²'],
  ['EstimatedPricePerSqm', 'Estimate €/m²'], ['Elo', 'Elo'], ['GeoPremium', 'GeoPremium %'],
  ['AgeZ', 'Age z-score'], ['AskingZ', 'Asking z-score'], ['EstimatedZ', 'Estimate z-score'],
  ['FloorZ', 'Floor z-score'], ['GeoPremiumZ', 'GeoPremium z-score'], ['EloZ', 'Elo z-score'],
]

export default function AnalysisPoints({ points, title }: { points: AnalysisPoint[]; title: string }) {
  const [page, setPage] = useState(1)
  const lastPage = Math.max(1, Math.ceil(points.length / 20))
  const currentPage = Math.min(page, lastPage)
  if (!points.length) return <p className="panel-note">No properties in this group.</p>
  return (
    <div className="analysis-point-list">
      <div className="property-table-wrap" tabIndex={0} aria-label={`${title} — scroll horizontally for all fields`}>
        <table className="property-table analysis-point-table">
          <caption className="sr-only">{title}</caption>
          <thead><tr>{columns.map(([key, label]) => <th key={key} scope="col">{label}</th>)}</tr></thead>
          <tbody>{points.slice((currentPage - 1) * 20, currentPage * 20).map((point) => <tr key={point.Uid}>{columns.map(([key]) => <td key={key}>{typeof point[key] === 'number' ? number(point[key] as number, 3) : point[key]}</td>)}</tr>)}</tbody>
        </table>
      </div>
      <div className="analysis-pagination"><span>{number(points.length)} properties · Page {currentPage} of {lastPage}</span><div><button type="button" className="property-button" disabled={currentPage === 1} onClick={() => setPage(currentPage - 1)}>Previous</button><button type="button" className="property-button" disabled={currentPage === lastPage} onClick={() => setPage(currentPage + 1)}>Next</button></div></div>
    </div>
  )
}
