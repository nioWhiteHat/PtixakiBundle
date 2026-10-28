import { useState } from 'react'
import type { AnalysisData } from '../analysis'
import { clusterColor } from '../analysis'
import { number } from '../property'
import ClusterPlot from './ClusterPlot'
import AnalysisPoints from './AnalysisPoints'
import AreaModelReport from './AreaModelReport'

const metrics = [
  ['Age (years)', 'AgeMean', 'AgeStdDev'], ['Asking (€/m²)', 'AskingMean', 'AskingStdDev'],
  ['Estimate (€/m²)', 'EstimatedMean', 'EstimatedStdDev'], ['Floor', 'FloorMean', 'FloorStdDev'],
  ['GeoPremium (%)', 'GeoPremiumMean', 'GeoPremiumStdDev'], ['Elo', 'EloMean', 'EloStdDev'],
] as const

export default function AnalysisResults({ data, areaName }: { data: AnalysisData; areaName: string }) {
  const points = data.points ?? []
  const clusters = data.clusters ?? []
  const [transaction, setTransaction] = useState(points.some((point) => point.TransactionType === 'sell') || data.sales_report ? 'sell' : 'rent')
  const [selected, setSelected] = useState<number | null>(null)
  const transactionPoints = points.filter((point) => point.TransactionType === transaction)
  const transactionClusters = clusters.filter((cluster) => cluster.TransactionType === transaction).sort((a, b) => a.ClusterID - b.ClusterID)
  const visiblePoints = selected == null ? transactionPoints : transactionPoints.filter((point) => point.ClusterID === selected)
  const visibleClusters = selected == null ? transactionClusters : transactionClusters.filter((cluster) => cluster.ClusterID === selected)

  return (
    <div className="analysis-results">
      <div className="analysis-results-heading"><p className="eyebrow">AREA SNAPSHOT</p><h2>{areaName}</h2></div>
      {data.warnings?.map((warning, index) => <p key={index} className="property-warning" role="status">{warning}</p>)}
      <dl className="portfolio-summary analysis-summary">
        <div><dt>Properties analysed</dt><dd>{number(points.length)}</dd></div>
        <div><dt>K-means groups</dt><dd>{new Set(clusters.map((cluster) => cluster.ClusterID)).size}</dd></div>
        <div><dt>Sales / rentals</dt><dd>{number(points.filter((point) => point.TransactionType === 'sell').length)} / {number(points.filter((point) => point.TransactionType === 'rent').length)}</dd></div>
      </dl>
      {!points.length && <p className="property-warning" role="status">No eligible properties were returned for this area.</p>}
      <div className="analysis-transactions" role="group" aria-label="Transaction type">
        {['sell', 'rent'].map((value) => <button type="button" key={value} aria-pressed={transaction === value} onClick={() => { setTransaction(value); setSelected(null) }}>{value === 'sell' ? 'Sales' : 'Rentals'}</button>)}
      </div>
      <section className="property-panel" aria-labelledby="clusters-title">
        <div className="panel-heading"><span className="section-number">01</span><div><h2 id="clusters-title">Similar properties, grouped</h2><p>K-means uses standardised area, age, renovation age, and floor. Groups are not geographic neighbourhoods.</p></div></div>
        <div className="cluster-filters" role="group" aria-label="Filter by cluster">
          <button type="button" aria-pressed={selected == null} onClick={() => setSelected(null)}>All groups</button>
          {transactionClusters.map((cluster) => <button type="button" key={cluster.ClusterID} aria-pressed={selected === cluster.ClusterID} onClick={() => setSelected(cluster.ClusterID)}><span style={{ background: clusterColor(cluster.ClusterID) }} />Cluster {cluster.ClusterID}</button>)}
        </div>
        <ClusterPlot points={visiblePoints} />
        {visibleClusters.map((cluster) => <section key={cluster.ClusterID} className="cluster-stats" aria-label={`Cluster ${cluster.ClusterID} statistics`}>
          <div className="portfolio-list-heading"><h3><span className="cluster-color" style={{ background: clusterColor(cluster.ClusterID) }} />Cluster {cluster.ClusterID}</h3><span>{number(transactionPoints.filter((point) => point.ClusterID === cluster.ClusterID).length)} properties</span></div>
          <div className="property-table-wrap"><table className="property-table">
            <caption className="sr-only">Cluster {cluster.ClusterID} means and standard deviations</caption>
            <thead><tr><th scope="col">Metric</th><th scope="col">Mean</th><th scope="col">Standard deviation</th></tr></thead>
            <tbody>{metrics.map(([label, mean, std]) => <tr key={mean}><th scope="row">{label}</th><td>{number(cluster[mean], 2)}</td><td>{number(cluster[std], 2)}</td></tr>)}</tbody>
          </table></div>
          <details className="cluster-extremes"><summary>Highest Elo · top 5% ({cluster.Top5?.length ?? 0})</summary><AnalysisPoints points={cluster.Top5 ?? []} title={`Cluster ${cluster.ClusterID} top 5 percent by Elo`} /></details>
          <details className="cluster-extremes"><summary>Lowest Elo · bottom 5% ({cluster.Bottom5?.length ?? 0})</summary><AnalysisPoints points={cluster.Bottom5 ?? []} title={`Cluster ${cluster.ClusterID} bottom 5 percent by Elo`} /></details>
        </section>)}
        {!transactionClusters.length && <p className="panel-note">No clusters were returned for this transaction type.</p>}
        <details className="cluster-extremes"><summary>All property data and cluster z-scores ({number(visiblePoints.length)})</summary><AnalysisPoints key={`${transaction}-${selected}`} points={visiblePoints} title="All analysed properties" /></details>
        <p className="panel-note">Z-scores are relative to each cluster and transaction group. Top/bottom groups are the endpoint’s 5% Elo tails, not five properties.</p>
      </section>
      <AreaModelReport report={transaction === 'sell' ? data.sales_report : data.rent_report} transaction={transaction} />
    </div>
  )
}
