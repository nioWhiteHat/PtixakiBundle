import type { AnalysisCluster, AnalysisData, AnalysisPoint, AreaModelReport, AreaSuggestion } from '../analysis'

export const areaFixture: AreaSuggestion = {
  display_name: 'Piraeus, Attica, Greece', lat: '37.943', lon: '23.647',
  boundingbox: ['37.90', '37.98', '23.60', '23.70'], address: { city: 'Piraeus' },
}

const points: AnalysisPoint[] = Array.from({ length: 80 }, (_, index) => ({
  Uid: `property_area_${index}`, TransactionType: index < 60 ? 'sell' : 'rent', ClusterID: index % 3,
  SquareMeters: 40 + (index % 3) * 25 + index % 13, Age: 5 + index % 3 * 14 + index % 7,
  RenovAge: index % 12, FloorLevel: index % 5, AskingPricePerSqm: index < 60 ? 2200 + index * 15 : 10 + index / 100,
  EstimatedPricePerSqm: index < 60 ? 2500 + index * 20 : 12 + index / 100,
  GeoPremium: -5 + index % 20, Elo: 9000 + index * 50,
  AgeZ: 0.3, AskingZ: -0.4, EstimatedZ: 0.1, FloorZ: 0.2, GeoPremiumZ: -0.2, EloZ: 0.5,
}))
const clusters: AnalysisCluster[] = ['sell', 'rent'].flatMap((transaction) => [0, 1, 2].map((id) => ({
  ClusterID: id, TransactionType: transaction,
  AgeMean: 10 + id * 14, AgeStdDev: 3.7, AskingMean: transaction === 'sell' ? 2600 : 11, AskingStdDev: transaction === 'sell' ? 400 : 1.1,
  EstimatedMean: transaction === 'sell' ? 3200 : 13, EstimatedStdDev: transaction === 'sell' ? 500 : 1.4,
  FloorMean: 2, FloorStdDev: 1.3, GeoPremiumMean: 5, GeoPremiumStdDev: 6, EloMean: 12000, EloStdDev: 1800,
  Top5: points.filter((point) => point.TransactionType === transaction && point.ClusterID === id).slice(-1),
  Bottom5: points.filter((point) => point.TransactionType === transaction && point.ClusterID === id).slice(0, 1),
})))
const names = ['SquareMeters', 'Age', 'GeoPremium', 'RenovAge', 'FloorLevel', 'Bedrooms', 'ViewSea', 'Parking', 'HeatType', 'WC', 'Garden', 'Pool']
const report: AreaModelReport = {
  r2_score: 0.7342, base_price: 2300, message: 'Success',
  feature_importances: names.map((feature, index) => ({ feature, importance: (12 - index) / 78 })),
  top_interactions: names.slice(0, 5).map((feature_1, index) => ({ feature_1, feature_2: names[index + 1], interaction_strength: 75 - index * 12 })),
  examples: points.slice(0, 10).map((point, index) => ({ uid: point.Uid, actual_price: point.AskingPricePerSqm, predicted_price: 2500 + index * 50,
    feature_impacts: [{ feature: 'SquareMeters', money_added: 300 }, { feature: 'Age', money_added: -100 + index * 50 }],
  })),
}
export const analysisFixture: AnalysisData = { points, clusters, sales_report: report, rent_report: { ...report, r2_score: 0.61, base_price: 10.5, examples: points.slice(60, 70).map((point) => ({ uid: point.Uid, actual_price: point.AskingPricePerSqm, predicted_price: 12.5, feature_impacts: [{ feature: 'SquareMeters', money_added: 3 }, { feature: 'Age', money_added: -1 }] })) } }
