import { useState } from 'react'
import { featureLabel, money, number } from '../property'
import type { PropertyDetailData } from '../property'

export default function PropertyExplanation({ property }: { property: PropertyDetailData }) {
  const [allLocal, setAllLocal] = useState(false)
  const [allGlobal, setAllGlobal] = useState(false)
  const explanation = property.explanation
  const contributions = [...(explanation?.contributions ?? [])].sort((a, b) => Math.abs(b.impact_per_sqm) - Math.abs(a.impact_per_sqm))
  const importance = [...property.feature_importance].sort((a, b) => a.rank - b.rank)
  const maxImpact = Math.max(...contributions.map((row) => Math.abs(row.impact_per_sqm)), 1)
  const maxImportance = Math.max(...importance.map((row) => row.importance_percentage), 1)
  const signedMoney = (value: number) => `${value > 0 ? '+' : ''}${money(value)}`

  return (
    <>
      <section className="property-panel" aria-labelledby="explanation-title">
        <div className="panel-heading"><span className="section-number">02</span><div><h2 id="explanation-title">Why this estimate?</h2><p>SHAP explanation · this property</p></div></div>
        {!explanation ? <p className="panel-note">No explanation has been stored for this property yet. Its other details are still available.</p> : <>
          <p className="panel-note">Each feature adds to or subtracts from the model’s baseline. These are model contributions, not guaranteed renovation returns.</p>
          <dl className="explanation-summary">
            <div><dt>Baseline / m²</dt><dd>{money(explanation.baseline_price_per_sqm)}</dd></div>
            <div><dt>Feature adjustments / m²</dt><dd>{signedMoney(contributions.reduce((total, row) => total + row.impact_per_sqm, 0))}</dd></div>
            <div><dt>Explained estimate / m²</dt><dd>{money(explanation.explained_prediction_per_sqm)}</dd></div>
          </dl>
          <div className="shap-legend"><span className="impact-negative">− Lowers estimate</span><span className="impact-positive">+ Raises estimate</span></div>
          <div className="property-table-wrap">
            <table className="property-table local-shap-table">
              <caption className="sr-only">Property-specific SHAP contributions in euros per square metre</caption>
              <thead><tr><th scope="col">Feature</th><th scope="col">Impact / m²</th><th scope="col">Local share</th></tr></thead>
              <tbody>{(allLocal ? contributions : contributions.slice(0, 8)).map((row) => (
                <tr key={row.feature_name}>
                  <th scope="row" title={row.feature_name}>{featureLabel(row.feature_name)}</th>
                  <td>
                    <span className={`impact-value ${row.impact_per_sqm < 0 ? 'impact-negative' : row.impact_per_sqm > 0 ? 'impact-positive' : ''}`}>{signedMoney(row.impact_per_sqm)}</span>
                    <span className="impact-track" aria-hidden="true"><span className={row.impact_per_sqm < 0 ? 'negative-bar' : 'positive-bar'} style={{ width: `${Math.abs(row.impact_per_sqm) / maxImpact * 50}%`, [row.impact_per_sqm < 0 ? 'right' : 'left']: '50%' }} /></span>
                  </td>
                  <td>{number(row.local_importance_percentage, 2)}%</td>
                </tr>
              ))}</tbody>
            </table>
          </div>
          <p className="panel-note table-note">Local share is the feature’s share of absolute impact for this property, not a percentage change in price.</p>
          {contributions.length > 8 && <button className="property-button" onClick={() => setAllLocal(!allLocal)} aria-expanded={allLocal}>{allLocal ? 'Show top 8 contributions' : `Show all ${contributions.length} contributions`}</button>}
          <details className="model-details"><summary>Explanation model information</summary><dl className="property-features">
            <div><dt>Model SHA-256</dt><dd>{explanation.model_sha256}</dd></div>
            <div><dt>Explained at</dt><dd>{new Date(explanation.explained_at).toLocaleString('en-GB')}</dd></div>
            <div><dt>Additivity error / m²</dt><dd>{number(explanation.additivity_error, 6)}</dd></div>
          </dl></details>
        </>}
      </section>

      <section className="property-panel" aria-labelledby="importance-title">
        <div className="panel-heading"><span className="section-number">03</span><div><h2 id="importance-title">What matters across the market</h2><p>Global feature importance · {property.transaction === 'rent' ? 'rental' : 'sale'} model</p></div></div>
        <p className="panel-note">Average absolute SHAP impacts across the training dataset, expressed as a share of total importance. These percentages are not specific to this property.</p>
        {explanation?.matches_importance_model === false && <p className="property-warning" role="status">The property explanation and global importance come from different model versions.</p>}
        {importance.length === 0 ? <p className="panel-note">Global feature importance is not available yet.</p> : <>
          <div className="property-table-wrap">
            <table className="property-table global-shap-table">
              <caption className="sr-only">Global SHAP feature importance for the {property.transaction} model</caption>
              <thead><tr><th scope="col">Feature</th><th scope="col">Importance</th><th scope="col">Mean |impact| / m²</th><th scope="col">Mean signed / m²</th></tr></thead>
              <tbody>{(allGlobal ? importance : importance.slice(0, 8)).map((row) => (
                <tr key={row.feature_name}>
                  <th scope="row" title={row.feature_name}><span className="feature-rank">{row.rank}</span>{featureLabel(row.feature_name)}</th>
                  <td>{number(row.importance_percentage, 2)}%<span className="importance-track" aria-hidden="true"><span style={{ width: `${row.importance_percentage / maxImportance * 100}%` }} /></span></td>
                  <td>{money(row.mean_abs_shap_per_sqm)}</td><td>{signedMoney(row.mean_signed_shap_per_sqm)}</td>
                </tr>
              ))}</tbody>
            </table>
          </div>
          {importance.length > 8 && <button className="property-button" onClick={() => setAllGlobal(!allGlobal)} aria-expanded={allGlobal}>{allGlobal ? 'Show top 8 features' : `Show all ${importance.length} features`}</button>}
          <details className="model-details"><summary>Global model information</summary><dl className="property-features">
            <div><dt>Model SHA-256</dt><dd>{importance[0].model_sha256}</dd></div>
            <div><dt>Calculated at</dt><dd>{new Date(importance[0].calculated_at).toLocaleString('en-GB')}</dd></div>
          </dl></details>
        </>}
      </section>
    </>
  )
}
