import type { PortfolioData, PropertyDetailData } from '../property'

export const portfolioFixture: PortfolioData = {
  property_count: 3,
  total_estimated_value: 470000,
  properties: [
    { uid: 'property_test_1', address: 'Egnatia 90, Thessaloniki', square_meters: 82, estimated_price_per_sqm: 4000, asking_price_per_sqm: 2650, elo: 16896, geo_premium: 19.51 },
    { uid: 'property_test_2', address: 'Piraeus, Greece', square_meters: 71, estimated_price_per_sqm: 2000, asking_price_per_sqm: null, elo: 11000, geo_premium: 0, image: null },
    { uid: 'property_test_3', address: null, square_meters: null, estimated_price_per_sqm: null, asking_price_per_sqm: null, elo: 0, geo_premium: null },
  ],
}

const featureNames = ['SubType', 'AdminName', 'SquareMeters', 'Age', 'RenovAge', 'NoView', 'ViewForest', 'ViewMountain', 'ViewSea', 'ViewUnlimited', 'NoFacade', 'FacadeCorner', 'FacadeFrontage', 'FacadeInterior', 'FacadeThrough', 'HeatType', 'HeatFrom', 'EnergyPerformance', 'WC', 'AC', 'Elevator', 'Parking', 'StorageRoom', 'AluminumFrames', 'Bedrooms', 'Pool', 'Garden', 'FloorLevel', 'FloorCount', 'GeoPremium']
export const detailFixture: PropertyDetailData = {
  uid: 'property_test_1', transaction: 'sell',
  features: {
    UID: 'property_test_1', Address: 'Egnatia 90, Thessaloniki', SubType: 'apartment', SquareMeters: 82,
    AskingPricePerSqm: 2650, ConstructionYear: 2004, RenovYear: 2019, IsRenovated: 1,
    AdminName: 'Thessaloniki', GeoLat: 40.633, GeoLon: 22.948, ZoneValue: 1800, Distance: 0,
    NoView: 0, ViewForest: 0, ViewMountain: 0, ViewSea: 1, ViewUnlimited: 0,
    NoFacade: 0, FacadeCorner: 0, FacadeFrontage: 1, FacadeInterior: 0, FacadeThrough: 0,
    HeatType: 'autonomous', HeatFrom: 'gas', EnergyPerformance: 'B', WC: 2, AC: 1, Elevator: 1,
    Parking: 1, StorageRoom: 1, AluminumFrames: 1, Bedrooms: 2, Pool: 0, Garden: 0, FloorLevel: [0, 3],
  },
  estimated_price_per_sqm: 4000, estimated_total_price: 328000,
  asking_price_per_sqm: 2650, asking_total_price: 217300,
  elo: 16896, geo_premium: 19.51, geo_premium_zscore: 0.47, residual_percentage: -50.94,
  explanation: {
    baseline_price_per_sqm: 3000, explained_prediction_per_sqm: 4000, additivity_error: 0,
    model_sha256: 'a'.repeat(64), explained_at: '2026-09-16T09:00:00Z', matches_importance_model: true,
    contributions: featureNames.map((feature_name, index) => ({
      feature_name, impact_per_sqm: index === 0 ? 1200 : index === 1 ? -200 : 0,
      local_importance_percentage: index === 0 ? 85.714 : index === 1 ? 14.286 : 0,
    })),
  },
  feature_importance: featureNames.map((feature_name, index) => ({
    feature_name, mean_abs_shap_per_sqm: 300 - index * 10, mean_signed_shap_per_sqm: 15 - index,
    importance_percentage: (30 - index) / 465 * 100, rank: index + 1,
    model_sha256: 'a'.repeat(64), calculated_at: '2026-09-16T09:00:00Z',
  })),
}
