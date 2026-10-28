export type AreaSuggestion = {
  display_name: string
  osm_id?: number
  osm_type?: 'node' | 'way' | 'relation'
  lat: string
  lon: string
  boundingbox: string[]
  address?: { road?: string; house_number?: string; suburb?: string; city_district?: string; city?: string; town?: string; village?: string }
}

export function areaBounds(area: AreaSuggestion) {
  const bounds = area.boundingbox ?? []
  const [MinLat, MaxLat, MinLon, MaxLon] = bounds.map(Number)
  if (bounds.length !== 4 || ![MinLat, MaxLat, MinLon, MaxLon].every(Number.isFinite) || MinLat >= MaxLat || MinLon >= MaxLon) {
    throw new Error('This area has no usable bounding box. Please choose another suggestion.')
  }
  return { MinLat, MaxLat, MinLon, MaxLon }
}

export type AnalysisPoint = {
  Uid: string
  TransactionType: string
  ClusterID: number
  SquareMeters: number
  Age: number
  RenovAge: number
  FloorLevel: number
  AskingPricePerSqm: number
  EstimatedPricePerSqm: number
  GeoPremium: number
  Elo: number
  AgeZ: number
  AskingZ: number
  EstimatedZ: number
  FloorZ: number
  GeoPremiumZ: number
  EloZ: number
}

export type AnalysisCluster = {
  ClusterID: number
  TransactionType: string
  AgeMean: number; AgeStdDev: number
  AskingMean: number; AskingStdDev: number
  EstimatedMean: number; EstimatedStdDev: number
  FloorMean: number; FloorStdDev: number
  GeoPremiumMean: number; GeoPremiumStdDev: number
  EloMean: number; EloStdDev: number
  Top5: AnalysisPoint[] | null
  Bottom5: AnalysisPoint[] | null
}

export type AreaModelReport = {
  r2_score: number
  base_price: number
  message: string
  feature_importances: { feature: string; importance: number }[] | null
  top_interactions: { feature_1: string; feature_2: string; interaction_strength: number }[] | null
  examples: { uid: string; actual_price: number; predicted_price: number; feature_impacts: { feature: string; money_added: number }[] | null }[] | null
}

export type AnalysisData = {
  warnings?: string[]
  points: AnalysisPoint[] | null
  clusters: AnalysisCluster[] | null
  sales_report: AreaModelReport | null
  rent_report: AreaModelReport | null
}

export const clusterColor = (id: number) => ['#e3b28d', '#86b4d6', '#a4c39c', '#c1a0d4', '#e0cf8b', '#78c0b5', '#d89197', '#a6b5e2', '#c3b7a1', '#b2c4cd'][Math.abs(id) % 10]
