BEGIN;

-- Latest per-property Phase2 SHAP explanation for residence/property sales.
CREATE TABLE IF NOT EXISTS public.properties_analysis_sell (
    property_uid character varying(255) NOT NULL,
    subtype_impact_per_sqm double precision NOT NULL,
    admin_name_impact_per_sqm double precision NOT NULL,
    square_meters_impact_per_sqm double precision NOT NULL,
    age_impact_per_sqm double precision NOT NULL,
    renov_age_impact_per_sqm double precision NOT NULL,
    no_view_impact_per_sqm double precision NOT NULL,
    view_forest_impact_per_sqm double precision NOT NULL,
    view_mountain_impact_per_sqm double precision NOT NULL,
    view_sea_impact_per_sqm double precision NOT NULL,
    view_unlimited_impact_per_sqm double precision NOT NULL,
    no_facade_impact_per_sqm double precision NOT NULL,
    facade_corner_impact_per_sqm double precision NOT NULL,
    facade_frontage_impact_per_sqm double precision NOT NULL,
    facade_interior_impact_per_sqm double precision NOT NULL,
    facade_through_impact_per_sqm double precision NOT NULL,
    heat_type_impact_per_sqm double precision NOT NULL,
    heat_from_impact_per_sqm double precision NOT NULL,
    energy_performance_impact_per_sqm double precision NOT NULL,
    wc_impact_per_sqm double precision NOT NULL,
    ac_impact_per_sqm double precision NOT NULL,
    elevator_impact_per_sqm double precision NOT NULL,
    parking_impact_per_sqm double precision NOT NULL,
    storage_room_impact_per_sqm double precision NOT NULL,
    aluminum_frames_impact_per_sqm double precision NOT NULL,
    bedrooms_impact_per_sqm double precision NOT NULL,
    pool_impact_per_sqm double precision NOT NULL,
    garden_impact_per_sqm double precision NOT NULL,
    floor_level_impact_per_sqm double precision NOT NULL,
    floor_count_impact_per_sqm double precision NOT NULL,
    geo_premium_impact_per_sqm double precision NOT NULL,
    baseline_price_per_sqm double precision NOT NULL,
    explained_prediction_per_sqm double precision NOT NULL,
    additivity_error double precision NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    explained_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT properties_analysis_sell_pkey PRIMARY KEY (property_uid),
    CONSTRAINT properties_analysis_sell_property_uid_fkey
        FOREIGN KEY (property_uid)
        REFERENCES public.properties(uid)
        ON DELETE CASCADE
);

-- Latest per-property Phase2 SHAP explanation for residence/property rentals.
CREATE TABLE IF NOT EXISTS public.properties_analysis_rent (
    property_uid character varying(255) NOT NULL,
    subtype_impact_per_sqm double precision NOT NULL,
    admin_name_impact_per_sqm double precision NOT NULL,
    square_meters_impact_per_sqm double precision NOT NULL,
    age_impact_per_sqm double precision NOT NULL,
    renov_age_impact_per_sqm double precision NOT NULL,
    no_view_impact_per_sqm double precision NOT NULL,
    view_forest_impact_per_sqm double precision NOT NULL,
    view_mountain_impact_per_sqm double precision NOT NULL,
    view_sea_impact_per_sqm double precision NOT NULL,
    view_unlimited_impact_per_sqm double precision NOT NULL,
    no_facade_impact_per_sqm double precision NOT NULL,
    facade_corner_impact_per_sqm double precision NOT NULL,
    facade_frontage_impact_per_sqm double precision NOT NULL,
    facade_interior_impact_per_sqm double precision NOT NULL,
    facade_through_impact_per_sqm double precision NOT NULL,
    heat_type_impact_per_sqm double precision NOT NULL,
    heat_from_impact_per_sqm double precision NOT NULL,
    energy_performance_impact_per_sqm double precision NOT NULL,
    wc_impact_per_sqm double precision NOT NULL,
    ac_impact_per_sqm double precision NOT NULL,
    elevator_impact_per_sqm double precision NOT NULL,
    parking_impact_per_sqm double precision NOT NULL,
    storage_room_impact_per_sqm double precision NOT NULL,
    aluminum_frames_impact_per_sqm double precision NOT NULL,
    bedrooms_impact_per_sqm double precision NOT NULL,
    pool_impact_per_sqm double precision NOT NULL,
    garden_impact_per_sqm double precision NOT NULL,
    floor_level_impact_per_sqm double precision NOT NULL,
    floor_count_impact_per_sqm double precision NOT NULL,
    geo_premium_impact_per_sqm double precision NOT NULL,
    baseline_price_per_sqm double precision NOT NULL,
    explained_prediction_per_sqm double precision NOT NULL,
    additivity_error double precision NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    explained_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT properties_analysis_rent_pkey PRIMARY KEY (property_uid),
    CONSTRAINT properties_analysis_rent_property_uid_fkey
        FOREIGN KEY (property_uid)
        REFERENCES public.properties(uid)
        ON DELETE CASCADE
);

-- Latest global pairwise SHAP interaction aggregates for the sell model.
CREATE TABLE IF NOT EXISTS public.features_interactions_sell (
    feature_1 text NOT NULL,
    feature_2 text NOT NULL,
    mean_signed_interaction_per_sqm double precision NOT NULL,
    mean_abs_interaction_per_sqm double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT features_interactions_sell_pkey PRIMARY KEY (feature_1, feature_2),
    CONSTRAINT features_interactions_sell_distinct_features_check CHECK (feature_1 <> feature_2),
    CONSTRAINT features_interactions_sell_mean_abs_check CHECK (mean_abs_interaction_per_sqm >= 0),
    CONSTRAINT features_interactions_sell_importance_rank_check CHECK (importance_rank > 0)
);

-- Latest global pairwise SHAP interaction aggregates for the rent model.
CREATE TABLE IF NOT EXISTS public.features_interactions_rent (
    feature_1 text NOT NULL,
    feature_2 text NOT NULL,
    mean_signed_interaction_per_sqm double precision NOT NULL,
    mean_abs_interaction_per_sqm double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT features_interactions_rent_pkey PRIMARY KEY (feature_1, feature_2),
    CONSTRAINT features_interactions_rent_distinct_features_check CHECK (feature_1 <> feature_2),
    CONSTRAINT features_interactions_rent_mean_abs_check CHECK (mean_abs_interaction_per_sqm >= 0),
    CONSTRAINT features_interactions_rent_importance_rank_check CHECK (importance_rank > 0)
);

-- Latest global SHAP feature-importance aggregates for the sell model.
CREATE TABLE IF NOT EXISTS public.feature_importance_sell (
    feature_name text NOT NULL,
    mean_abs_shap_per_sqm double precision NOT NULL,
    mean_signed_shap_per_sqm double precision NOT NULL,
    importance_percentage double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT feature_importance_sell_pkey PRIMARY KEY (feature_name),
    CONSTRAINT feature_importance_sell_percentage_check CHECK (
        importance_percentage >= 0 AND importance_percentage <= 100
    ),
    CONSTRAINT feature_importance_sell_mean_abs_check CHECK (mean_abs_shap_per_sqm >= 0),
    CONSTRAINT feature_importance_sell_importance_rank_check CHECK (importance_rank > 0)
);

-- Latest global SHAP feature-importance aggregates for the rent model.
CREATE TABLE IF NOT EXISTS public.feature_importance_rent (
    feature_name text NOT NULL,
    mean_abs_shap_per_sqm double precision NOT NULL,
    mean_signed_shap_per_sqm double precision NOT NULL,
    importance_percentage double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT feature_importance_rent_pkey PRIMARY KEY (feature_name),
    CONSTRAINT feature_importance_rent_percentage_check CHECK (
        importance_percentage >= 0 AND importance_percentage <= 100
    ),
    CONSTRAINT feature_importance_rent_mean_abs_check CHECK (mean_abs_shap_per_sqm >= 0),
    CONSTRAINT feature_importance_rent_importance_rank_check CHECK (importance_rank > 0)
);

-- global_stats is keyed by semantic stat_name, not by transaction. Keep its
-- existing mean/std rows and place latest model metadata on Price_stats_sell
-- and Price_stats_rent respectively.
ALTER TABLE public.global_stats
    ADD COLUMN IF NOT EXISTS baseline_price_per_sqm double precision,
    ADD COLUMN IF NOT EXISTS training_r2 double precision,
    ADD COLUMN IF NOT EXISTS validation_r2 double precision,
    ADD COLUMN IF NOT EXISTS model_sha256 character varying(64),
    ADD COLUMN IF NOT EXISTS model_trained_at timestamp with time zone;

COMMENT ON COLUMN public.global_stats.baseline_price_per_sqm IS
    'Latest model baseline; populated on Price_stats_sell or Price_stats_rent.';
COMMENT ON COLUMN public.global_stats.training_r2 IS
    'Latest training R-squared; populated on Price_stats_sell or Price_stats_rent.';
COMMENT ON COLUMN public.global_stats.validation_r2 IS
    'Latest validation R-squared; populated on Price_stats_sell or Price_stats_rent.';
COMMENT ON COLUMN public.global_stats.model_sha256 IS
    'Latest model SHA-256; populated on Price_stats_sell or Price_stats_rent.';
COMMENT ON COLUMN public.global_stats.model_trained_at IS
    'Latest model training time; populated on Price_stats_sell or Price_stats_rent.';

COMMIT;
