import type { AreaModelReport as ModelReport } from '../analysis'
import { featureLabel, money, number } from '../property'

export default function AreaModelReport({ report, transaction }: { report: ModelReport | null; transaction: string }) {
  if (!report) return <section className="property-panel"><h2>No {transaction === 'sell' ? 'sale' : 'rental'} model returned</h2><p className="panel-note">The endpoint did not return a model report for this transaction type.</p></section>
  return (
    <section className="property-panel area-model-report" aria-labelledby="area-model-title">
      <div className="panel-heading"><span className="section-number">02</span><div><h2 id="area-model-title">The area’s {transaction === 'sell' ? 'sale' : 'rental'} model</h2><p>A fresh XGBoost model built for this area, not the saved global valuation model.</p></div></div>
      <dl className="explanation-summary">
        <div><dt>Training R²</dt><dd>{number(report.r2_score, 4)}</dd></div>
        <div><dt>SHAP baseline / m²</dt><dd>{money(report.base_price)}</dd></div>
        <div><dt>Explained examples</dt><dd>{report.examples?.length ?? 0}</dd></div>
      </dl>
      <p className="panel-note">R² is measured on the same rows used to train this area model, not an independent validation score.</p>
      {report.message && report.message !== 'Success' && <p className="property-warning">{report.message}</p>}

      <h3>Feature importance</h3>
      <p className="panel-note">XGBoost importance weights shown as percentages. This endpoint does not return global SHAP importance percentages.</p>
      {report.feature_importances?.length ? <div className="property-table-wrap"><table className="property-table">
        <caption className="sr-only">Area XGBoost feature importance</caption>
        <thead><tr><th scope="col">Feature</th><th scope="col">Importance</th></tr></thead>
        <tbody>{report.feature_importances.map((row) => <tr key={row.feature}><th scope="row">{featureLabel(row.feature)}</th><td>{number(row.importance * 100, 2)}%<span className="importance-track" aria-hidden="true"><span style={{ width: `${row.importance * 100}%` }} /></span></td></tr>)}</tbody>
      </table></div> : <p className="panel-note">No feature importance returned.</p>}

      <h3>Feature interactions</h3>
      <p className="panel-note">The strongest returned pairs, ranked by mean absolute SHAP interaction strength (€/m²). Strength has no positive/negative direction.</p>
      {report.top_interactions?.length ? <div className="property-table-wrap"><table className="property-table">
        <caption className="sr-only">Area SHAP feature interactions</caption>
        <thead><tr><th scope="col">Feature pair</th><th scope="col">Strength / m²</th></tr></thead>
        <tbody>{report.top_interactions.map((row) => <tr key={`${row.feature_1}-${row.feature_2}`}><th scope="row">{featureLabel(row.feature_1)} <span aria-hidden="true">×</span> {featureLabel(row.feature_2)}</th><td>{money(row.interaction_strength)}</td></tr>)}</tbody>
      </table></div> : <p className="panel-note">No interactions returned.</p>}

      <h3>Explained properties</h3>
      <p className="panel-note">All {report.examples?.length ?? 0} examples returned by the area model. Expand a property to see every returned feature contribution.</p>
      {(report.examples ?? []).map((example) => (
        <details key={example.uid} className="area-example">
          <summary><span>{example.uid}</span><span>Asking {money(example.actual_price)} / m² <span aria-hidden="true">→</span> Model {money(example.predicted_price)} / m²</span></summary>
          <p className="panel-note">Baseline {money(report.base_price)} / m² + signed contributions ≈ {money(example.predicted_price)} / m².</p>
          <div className="property-table-wrap"><table className="property-table">
            <caption className="sr-only">Area-model contributions for {example.uid}</caption>
            <thead><tr><th scope="col">Feature</th><th scope="col">Contribution / m²</th></tr></thead>
            <tbody>{(example.feature_impacts ?? []).map((impact) => <tr key={impact.feature}><th scope="row">{featureLabel(impact.feature)}</th><td className={impact.money_added < 0 ? 'impact-negative' : 'impact-positive'}>{impact.money_added > 0 ? '+' : ''}{money(impact.money_added)}</td></tr>)}</tbody>
          </table></div>
        </details>
      ))}
      {!report.examples?.length && <p className="panel-note">No explained examples returned.</p>}
    </section>
  )
}
