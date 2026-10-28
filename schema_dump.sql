--
-- PostgreSQL database dump
--

\restrict AlDwefuEOy4a3wh2nAqcitRFQeY7ptGeEreUwds6Mnef425q8iXVWSFj1QcRBcs

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg110+1)
-- Dumped by pg_dump version 18.0

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pg_cron; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_cron WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION pg_cron; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pg_cron IS 'Job scheduler for PostgreSQL';


--
-- Name: partman; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA partman;


ALTER SCHEMA partman OWNER TO postgres;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: postgres
--

-- *not* creating schema, since initdb creates it


ALTER SCHEMA public OWNER TO postgres;

--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON SCHEMA public IS '';


--
-- Name: pg_partman; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_partman WITH SCHEMA public;


--
-- Name: EXTENSION pg_partman; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pg_partman IS 'Extension to manage partitioned tables by time or ID';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: postgis; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;


--
-- Name: EXTENSION postgis; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION postgis IS 'PostGIS geometry and geography spatial types and functions';


--
-- Name: unaccent; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;


--
-- Name: EXTENSION unaccent; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION unaccent IS 'text search dictionary that removes accents';


--
-- Name: apply_estimated_value_adjustment(integer, timestamp with time zone); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.apply_estimated_value_adjustment(raw_estimated_value integer, as_of timestamp with time zone DEFAULT now()) RETURNS integer
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE
        WHEN raw_estimated_value IS NULL THEN NULL
        ELSE LEAST(
            2147483647::BIGINT,
            GREATEST(
                0::BIGINT,
                ROUND(
                    raw_estimated_value::NUMERIC * (
                        1 + public.get_active_estimated_value_adjustment_percent(as_of) / 100.0
                    )
                )::BIGINT
            )
        )::INTEGER
    END;
$$;


ALTER FUNCTION public.apply_estimated_value_adjustment(raw_estimated_value integer, as_of timestamp with time zone) OWNER TO postgres;

--
-- Name: apply_estimated_value_adjustment(integer, timestamp with time zone, character varying, character varying); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.apply_estimated_value_adjustment(raw_estimated_value integer, as_of timestamp with time zone DEFAULT now(), p_item_type character varying DEFAULT NULL::character varying, p_transaction_type character varying DEFAULT NULL::character varying) RETURNS integer
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE
        WHEN raw_estimated_value IS NULL THEN NULL
        ELSE LEAST(
            2147483647::BIGINT,
            GREATEST(0::BIGINT,
                ROUND(raw_estimated_value::NUMERIC * (
                    1 + public.get_active_estimated_value_adjustment_percent(as_of, p_item_type, p_transaction_type) / 100.0
                ))::BIGINT
            )
        )::INTEGER
    END;
$$;


ALTER FUNCTION public.apply_estimated_value_adjustment(raw_estimated_value integer, as_of timestamp with time zone, p_item_type character varying, p_transaction_type character varying) OWNER TO postgres;

--
-- Name: apply_estimated_value_adjustment(integer, timestamp with time zone, character varying, character varying, character varying, double precision, double precision); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.apply_estimated_value_adjustment(raw_estimated_value integer, as_of timestamp with time zone DEFAULT now(), p_item_type character varying DEFAULT NULL::character varying, p_transaction_type character varying DEFAULT NULL::character varying, p_subtype character varying DEFAULT NULL::character varying, p_lat double precision DEFAULT NULL::double precision, p_lon double precision DEFAULT NULL::double precision) RETURNS integer
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE
        WHEN raw_estimated_value IS NULL THEN NULL
        ELSE LEAST(
            2147483647::BIGINT,
            GREATEST(
                0::BIGINT,
                ROUND(
                    raw_estimated_value::NUMERIC
                    * (1 + public.get_active_estimated_value_adjustment_percent(as_of, p_item_type, p_transaction_type, p_subtype) / 100.0)
                    * (1 + GREATEST(-30.0, LEAST(30.0, public.get_market_premium_for_point(p_lat, p_lon, p_item_type, p_subtype))) / 100.0)
                )::BIGINT
            )
        )::INTEGER
    END;
$$;


ALTER FUNCTION public.apply_estimated_value_adjustment(raw_estimated_value integer, as_of timestamp with time zone, p_item_type character varying, p_transaction_type character varying, p_subtype character varying, p_lat double precision, p_lon double precision) OWNER TO postgres;

--
-- Name: apply_market_premium_only(integer, timestamp with time zone, double precision, double precision, text, text); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.apply_market_premium_only(raw_value integer, as_of timestamp with time zone DEFAULT now(), p_lat double precision DEFAULT NULL::double precision, p_lon double precision DEFAULT NULL::double precision, p_item_type text DEFAULT NULL::text, p_subtype text DEFAULT NULL::text) RETURNS integer
    LANGUAGE sql STABLE
    AS $$
    SELECT CASE
        WHEN raw_value IS NULL THEN NULL
        ELSE LEAST(2147483647::BIGINT, GREATEST(0::BIGINT,
            ROUND(raw_value::NUMERIC * (1 + GREATEST(-30.0, LEAST(30.0, public.get_market_premium_for_point(p_lat, p_lon, p_item_type, p_subtype))) / 100.0))::BIGINT
        ))::INTEGER
    END;
$$;


ALTER FUNCTION public.apply_market_premium_only(raw_value integer, as_of timestamp with time zone, p_lat double precision, p_lon double precision, p_item_type text, p_subtype text) OWNER TO postgres;

--
-- Name: archive_monthly_valuations(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.archive_monthly_valuations() RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO valuations_history (
        property_uid,
        asking_price,
        real_value,
        estimated_value,
        estimated_value_confidence,
        created_at,
        archived_at
    )
    SELECT
        property_uid,
        asking_price,
        real_value,
        estimated_value,
        estimated_value_confidence,
        created_at,
        NOW()
    FROM
        valuations;

    RAISE NOTICE 'Archived % rows to valuations_history', (SELECT COUNT(*) FROM valuations);
END;
$$;


ALTER FUNCTION public.archive_monthly_valuations() OWNER TO postgres;

--
-- Name: archive_property_before_delete(character varying, character varying); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.archive_property_before_delete(p_uid character varying, p_reason character varying) RETURNS void
    LANGUAGE sql
    AS $$
    INSERT INTO deleted_properties (
        property_uid,
        item_type,
        transaction_type,
        subtype,
        admin_name,
        ad_creation_date,
        last_seen_at,
        days_listed,
        square_meters,
        final_asking_price,
        final_estimated_value,
        final_estimated_value_confidence,
        final_grounded_estimated_value,
        final_grounded_confidence,
        deletion_reason
    )
    SELECT
        p.uid,
        p.item_type,
        p.transaction_type,
        p.subtype,
        admin_res.name,
        p.ad_creation_date,
        p.updated_at,
        CASE WHEN (COALESCE(p.ad_creation_date, p.entry_created_at)) IS NOT NULL AND p.updated_at IS NOT NULL
             THEN (EXTRACT(EPOCH FROM (p.updated_at - COALESCE(p.ad_creation_date, p.entry_created_at))) / 86400)::INTEGER
             ELSE NULL END,
        ba.square_meters,
        p.asking_price,
        v.estimated_value,
        v.estimated_value_confidence,
        v.grounded_estimated_value,
        v.grounded_confidence,
        p_reason
    FROM properties p
    LEFT JOIN base_attributes ba ON ba.property_uid = p.uid
    LEFT JOIN valuations v ON v.property_uid = p.uid
    LEFT JOIN property_geo_data pgd ON pgd.property_uid = p.uid
    LEFT JOIN LATERAL (
        SELECT ab.name, ab.admin_level
        FROM admin_boundaries ab
        WHERE ab.admin_level IN (9, 8, 7, 6)
          AND ST_Contains(ab.geom, pgd.geom)
        ORDER BY ab.admin_level DESC
        LIMIT 1
    ) admin_res ON true
    WHERE p.uid = p_uid;
$$;


ALTER FUNCTION public.archive_property_before_delete(p_uid character varying, p_reason character varying) OWNER TO postgres;

--
-- Name: cascade_delete_billing_preferences_for_business(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.cascade_delete_billing_preferences_for_business() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM billing_preferences WHERE owner_type = 'business' AND owner_uid = OLD.uid;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.cascade_delete_billing_preferences_for_business() OWNER TO postgres;

--
-- Name: cascade_delete_billing_preferences_for_user(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.cascade_delete_billing_preferences_for_user() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM billing_preferences WHERE owner_type = 'individual' AND owner_uid = OLD.uid;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.cascade_delete_billing_preferences_for_user() OWNER TO postgres;

--
-- Name: cascade_delete_payment_methods_for_business(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.cascade_delete_payment_methods_for_business() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM payment_methods WHERE owner_type = 'business' AND owner_uid = OLD.uid;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.cascade_delete_payment_methods_for_business() OWNER TO postgres;

--
-- Name: cascade_delete_payment_methods_for_user(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.cascade_delete_payment_methods_for_user() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM payment_methods WHERE owner_type = 'individual' AND owner_uid = OLD.uid;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.cascade_delete_payment_methods_for_user() OWNER TO postgres;

--
-- Name: cascade_delete_payments_for_business(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.cascade_delete_payments_for_business() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM payments WHERE owner_type = 'business' AND owner_uid = OLD.uid;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.cascade_delete_payments_for_business() OWNER TO postgres;

--
-- Name: cascade_delete_payments_for_user(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.cascade_delete_payments_for_user() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM payments WHERE owner_type = 'individual' AND owner_uid = OLD.uid;
    RETURN OLD;
END;
$$;


ALTER FUNCTION public.cascade_delete_payments_for_user() OWNER TO postgres;

--
-- Name: compute_valuation_range(integer, numeric, numeric, numeric); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.compute_valuation_range(p_effective_value integer, p_confidence numeric, p_mae numeric, p_offset_max numeric) RETURNS TABLE(min_val integer, max_val integer)
    LANGUAGE plpgsql STABLE
    AS $$
    DECLARE
        conf      NUMERIC;
        off_mult  NUMERIC;
        half_band NUMERIC;
        lower_r   NUMERIC;
        upper_r   NUMERIC;
        half_w    NUMERIC;
        min_half  NUMERIC;
    BEGIN
        conf := GREATEST(0, LEAST(1, COALESCE(p_confidence, 0)));
        IF conf < 0.40 THEN
            RETURN;
        END IF;

        IF p_offset_max <= 0.9 THEN
            -- grounded: off_mult 0.60 at conf=0.40 → 0.25 at conf=1.0
            off_mult := GREATEST(0.25, 0.60 - (conf - 0.40) / 0.60 * 0.35);
        ELSE
            -- non-grounded: off_mult 0.90 at conf=0.40 → 0.40 at conf=1.0
            off_mult := GREATEST(0.40, 0.90 - (conf - 0.40) / 0.60 * 0.50);
        END IF;

        -- symmetric 1.0×/1.0× band
        half_band := p_effective_value::NUMERIC * p_mae * off_mult;
        lower_r  := p_effective_value::NUMERIC - half_band;
        upper_r  := p_effective_value::NUMERIC + half_band;

        -- ±5% minimum width floor
        half_w   := (upper_r - lower_r) / 2.0;
        min_half := p_effective_value::NUMERIC * 0.05;
        IF half_w < min_half THEN
            lower_r := p_effective_value::NUMERIC - min_half;
            upper_r := p_effective_value::NUMERIC + min_half;
        END IF;

        min_val := GREATEST(0, LEAST(2147483647, FLOOR(lower_r)))::INTEGER;
        max_val := GREATEST(0, LEAST(2147483647, CEIL(upper_r)))::INTEGER;
        RETURN NEXT;
    END;
$$;


ALTER FUNCTION public.compute_valuation_range(p_effective_value integer, p_confidence numeric, p_mae numeric, p_offset_max numeric) OWNER TO postgres;

--
-- Name: get_active_estimated_value_adjustment_percent(timestamp with time zone); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_active_estimated_value_adjustment_percent(as_of timestamp with time zone DEFAULT now()) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
    SELECT COALESCE(
        (
            SELECT vap.estimated_value_percent
            FROM valuation_adjustment_policies vap
            WHERE vap.effective_from <= as_of
            ORDER BY vap.effective_from DESC, vap.created_at DESC, vap.id DESC
            LIMIT 1
        ),
        0
    );
$$;


ALTER FUNCTION public.get_active_estimated_value_adjustment_percent(as_of timestamp with time zone) OWNER TO postgres;

--
-- Name: get_active_estimated_value_adjustment_percent(timestamp with time zone, character varying, character varying); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_active_estimated_value_adjustment_percent(as_of timestamp with time zone DEFAULT now(), p_item_type character varying DEFAULT NULL::character varying, p_transaction_type character varying DEFAULT NULL::character varying) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
    SELECT COALESCE(
        (
            SELECT vap.estimated_value_percent
            FROM valuation_adjustment_policies vap
            WHERE vap.effective_from <= as_of
              AND (vap.item_type IS NULL OR vap.item_type = p_item_type)
              AND (vap.transaction_type IS NULL OR vap.transaction_type = p_transaction_type)
            ORDER BY
                (vap.item_type IS NOT NULL)::int + (vap.transaction_type IS NOT NULL)::int DESC,
                vap.effective_from DESC,
                vap.created_at DESC,
                vap.id DESC
            LIMIT 1
        ),
        0
    );
$$;


ALTER FUNCTION public.get_active_estimated_value_adjustment_percent(as_of timestamp with time zone, p_item_type character varying, p_transaction_type character varying) OWNER TO postgres;

--
-- Name: get_active_estimated_value_adjustment_percent(timestamp with time zone, character varying, character varying, character varying); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_active_estimated_value_adjustment_percent(as_of timestamp with time zone DEFAULT now(), p_item_type character varying DEFAULT NULL::character varying, p_transaction_type character varying DEFAULT NULL::character varying, p_subtype character varying DEFAULT NULL::character varying) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
    SELECT COALESCE(
        (
            SELECT vap.estimated_value_percent
            FROM valuation_adjustment_policies vap
            WHERE vap.effective_from <= as_of
              AND (vap.item_type IS NULL OR vap.item_type = p_item_type)
              AND (vap.transaction_type IS NULL OR vap.transaction_type = p_transaction_type)
              AND (vap.subtype IS NULL OR vap.subtype = p_subtype)
            ORDER BY
                (vap.item_type IS NOT NULL)::int
                    + (vap.transaction_type IS NOT NULL)::int
                    + (vap.subtype IS NOT NULL)::int DESC,
                vap.effective_from DESC,
                vap.created_at DESC,
                vap.id DESC
            LIMIT 1
        ),
        0
    );
$$;


ALTER FUNCTION public.get_active_estimated_value_adjustment_percent(as_of timestamp with time zone, p_item_type character varying, p_transaction_type character varying, p_subtype character varying) OWNER TO postgres;

--
-- Name: get_active_market_premium_percent(timestamp with time zone); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_active_market_premium_percent(as_of timestamp with time zone DEFAULT now()) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
    SELECT COALESCE(
        (
            SELECT mpp.market_premium_percent
            FROM market_premium_policies mpp
            WHERE mpp.effective_from <= as_of
            ORDER BY mpp.effective_from DESC, mpp.created_at DESC, mpp.id DESC
            LIMIT 1
        ),
        0
    );
$$;


ALTER FUNCTION public.get_active_market_premium_percent(as_of timestamp with time zone) OWNER TO postgres;

--
-- Name: get_market_premium_for_point(double precision, double precision, text, text); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_market_premium_for_point(p_lat double precision, p_lon double precision, p_item_type text DEFAULT NULL::text, p_subtype text DEFAULT NULL::text) RETURNS numeric
    LANGUAGE sql STABLE
    AS $$
    SELECT COALESCE(
        (SELECT amp.market_premium_percent
         FROM area_market_premiums amp
         JOIN admin_boundaries ab ON amp.admin_name = ab.name
         WHERE ab.admin_level IN (9, 8, 7, 6)
           AND ST_Contains(ab.geom, ST_SetSRID(ST_MakePoint(p_lon, p_lat), 4326))
           AND (
               (amp.item_type IS NOT DISTINCT FROM p_item_type AND amp.subtype IS NOT DISTINCT FROM p_subtype)
               OR (amp.item_type IS NOT DISTINCT FROM p_item_type AND amp.subtype IS NULL)
               OR (amp.item_type IS NULL AND amp.subtype IS NULL)
           )
         ORDER BY ab.admin_level DESC,
                  CASE WHEN amp.item_type IS NOT NULL THEN 0 ELSE 1 END,
                  CASE WHEN amp.subtype IS NOT NULL THEN 0 ELSE 1 END
         LIMIT 1),
        public.get_active_market_premium_percent()
    );
$$;


ALTER FUNCTION public.get_market_premium_for_point(p_lat double precision, p_lon double precision, p_item_type text, p_subtype text) OWNER TO postgres;

--
-- Name: update_marketabillity_geom(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.update_marketabillity_geom() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- Case 1: Check for a full, valid LineString
    IF NEW.geo_lon_start IS NOT NULL AND NEW.geo_lon_start != 0 AND
       NEW.geo_lat_start IS NOT NULL AND NEW.geo_lat_start != 0 AND
       NEW.geo_lon_end IS NOT NULL AND NEW.geo_lon_end != 0 AND
       NEW.geo_lat_end IS NOT NULL AND NEW.geo_lat_end != 0 THEN

        -- If all coordinates are valid, create the LineString
        NEW.geom := ST_SetSRID(
            ST_MakeLine(
                ST_MakePoint(NEW.geo_lon_start, NEW.geo_lat_start),
                ST_MakePoint(NEW.geo_lon_end, NEW.geo_lat_end)
            ), 4326);

    -- Case 2: If no line, check for a valid start point
    ELSIF NEW.geo_lon_start IS NOT NULL AND NEW.geo_lon_start != 0 AND
          NEW.geo_lat_start IS NOT NULL AND NEW.geo_lat_start != 0 THEN

        -- Create a Point from the start coordinates
        NEW.geom := ST_SetSRID(
            ST_MakePoint(NEW.geo_lon_start, NEW.geo_lat_start),
            4326);

    -- Case 3: If no line or start point, check for a valid end point
    ELSIF NEW.geo_lon_end IS NOT NULL AND NEW.geo_lon_end != 0 AND
          NEW.geo_lat_end IS NOT NULL AND NEW.geo_lat_end != 0 THEN

        -- Create a Point from the end coordinates
        NEW.geom := ST_SetSRID(
            ST_MakePoint(NEW.geo_lon_end, NEW.geo_lat_end),
            4326);

    -- Case 4: If no valid coordinates exist
    ELSE
        -- Set the geometry to NULL
        NEW.geom := NULL;
    END IF;

    -- Return the modified row
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_marketabillity_geom() OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: admin_action_records; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.admin_action_records (
    uid character varying(255) NOT NULL,
    action character varying(100) NOT NULL,
    resource_type character varying(100) NOT NULL,
    resource_uid character varying(255) NOT NULL,
    idempotency_key character varying(255) NOT NULL,
    actor_uid character varying(255),
    reason text NOT NULL,
    result_json jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.admin_action_records OWNER TO postgres;

--
-- Name: admin_active_user_snapshots; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.admin_active_user_snapshots (
    sampled_at timestamp with time zone NOT NULL,
    active_user_count integer NOT NULL,
    connection_count integer NOT NULL,
    user_uids jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT admin_active_user_snapshots_active_user_count_check CHECK ((active_user_count >= 0)),
    CONSTRAINT admin_active_user_snapshots_connection_count_check CHECK ((connection_count >= 0))
);


ALTER TABLE public.admin_active_user_snapshots OWNER TO postgres;

--
-- Name: admin_audit_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.admin_audit_events (
    uid character varying(255) NOT NULL,
    actor_user_uid character varying(255) NOT NULL,
    method character varying(16) NOT NULL,
    route text NOT NULL,
    target_type character varying(255) NOT NULL,
    target_uid character varying(255) NOT NULL,
    risk_level character varying(32) NOT NULL,
    reason text NOT NULL,
    before_summary jsonb,
    after_summary jsonb,
    ip_address text,
    user_agent text,
    idempotency_key text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT admin_audit_events_risk_level_check CHECK (((risk_level)::text = ANY ((ARRAY['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[])))
);


ALTER TABLE public.admin_audit_events OWNER TO postgres;

--
-- Name: admin_boundaries; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.admin_boundaries (
    cache_key text NOT NULL,
    name text NOT NULL,
    admin_level integer NOT NULL,
    geom public.geometry(Geometry,4326) NOT NULL,
    cached_at timestamp with time zone DEFAULT now(),
    osm_id bigint,
    osm_type text
);


ALTER TABLE public.admin_boundaries OWNER TO postgres;

--
-- Name: admin_boundary_negatives; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.admin_boundary_negatives (
    cache_key text NOT NULL,
    cached_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.admin_boundary_negatives OWNER TO postgres;

--
-- Name: admin_charge_waivers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.admin_charge_waivers (
    uid character varying(255) NOT NULL,
    owner_type character varying(20) NOT NULL,
    owner_uid character varying(255) NOT NULL,
    operation character varying(100) NOT NULL,
    target_record_uid character varying(255),
    intended_amount_cents integer NOT NULL,
    net_amount_cents integer DEFAULT 0 NOT NULL,
    tax_amount_cents integer DEFAULT 0 NOT NULL,
    currency character varying(10) DEFAULT 'eur'::character varying NOT NULL,
    should_charge boolean DEFAULT false NOT NULL,
    reason text NOT NULL,
    actor_uid character varying(255) NOT NULL,
    idempotency_key character varying(255) NOT NULL,
    result_json jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT admin_charge_waivers_intended_amount_cents_check CHECK ((intended_amount_cents >= 0)),
    CONSTRAINT admin_charge_waivers_net_amount_cents_check CHECK ((net_amount_cents >= 0)),
    CONSTRAINT admin_charge_waivers_owner_type_check CHECK (((owner_type)::text = ANY ((ARRAY['individual'::character varying, 'business'::character varying])::text[]))),
    CONSTRAINT admin_charge_waivers_should_charge_check CHECK ((should_charge = false)),
    CONSTRAINT admin_charge_waivers_tax_amount_cents_check CHECK ((tax_amount_cents >= 0))
);


ALTER TABLE public.admin_charge_waivers OWNER TO postgres;

--
-- Name: area_market_premiums; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.area_market_premiums (
    id bigint NOT NULL,
    admin_name text NOT NULL,
    item_type text,
    subtype text,
    market_premium_percent numeric(10,4) NOT NULL,
    sample_size integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.area_market_premiums OWNER TO postgres;

--
-- Name: area_market_premiums_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.area_market_premiums_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.area_market_premiums_id_seq OWNER TO postgres;

--
-- Name: area_market_premiums_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.area_market_premiums_id_seq OWNED BY public.area_market_premiums.id;


--
-- Name: auth_providers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.auth_providers (
    uid character varying(255) NOT NULL,
    user_uid character varying(255) NOT NULL,
    provider character varying(50) NOT NULL,
    provider_user_id character varying(255),
    password_hash text,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.auth_providers OWNER TO postgres;

--
-- Name: base_attributes; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.base_attributes (
    property_uid character varying(255) NOT NULL,
    square_meters integer,
    zone_value integer,
    is_zone_value_progenitor boolean DEFAULT false,
    marketabillity_factor numeric(8,1) DEFAULT NULL::numeric,
    facade_type text[],
    view text[]
);


ALTER TABLE public.base_attributes OWNER TO postgres;

--
-- Name: billing_preferences; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.billing_preferences (
    uid character varying(255) NOT NULL,
    preference_type character varying(20) NOT NULL,
    vat_number character varying(50),
    first_name character varying(100),
    last_name character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    address_line character varying(200),
    city character varying(100),
    zip character varying(20),
    country character varying(2),
    owner_type character varying(20) NOT NULL,
    owner_uid character varying(255) NOT NULL,
    CONSTRAINT billing_preferences_invoice_fields CHECK ((((preference_type)::text <> 'invoice'::text) OR ((vat_number IS NOT NULL) AND (first_name IS NOT NULL) AND (last_name IS NOT NULL) AND (address_line IS NOT NULL) AND (city IS NOT NULL) AND (zip IS NOT NULL) AND (country IS NOT NULL)))),
    CONSTRAINT billing_preferences_owner_type_check CHECK (((owner_type)::text = ANY ((ARRAY['individual'::character varying, 'business'::character varying])::text[]))),
    CONSTRAINT billing_preferences_preference_type_check CHECK (((preference_type)::text = ANY ((ARRAY['invoice'::character varying, 'receipt'::character varying])::text[])))
);


ALTER TABLE public.billing_preferences OWNER TO postgres;

--
-- Name: building_details; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.building_details (
    property_uid character varying(255) NOT NULL,
    construction_year integer,
    renovation_year integer,
    heat character varying(100) DEFAULT NULL::character varying,
    heat_from character varying(100) DEFAULT NULL::character varying,
    landplot_size integer,
    bathrooms integer,
    wc integer,
    ac boolean,
    elevator boolean,
    parking boolean,
    storage_room boolean,
    aluminum_frames boolean,
    under_construction boolean,
    energy_performance_certificate character varying(50) DEFAULT NULL::character varying,
    floor_level text[],
    building_ratio numeric(8,4) DEFAULT NULL::numeric
);


ALTER TABLE public.building_details OWNER TO postgres;

--
-- Name: bulk_subscription_change_jobs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.bulk_subscription_change_jobs (
    uid character varying(255) NOT NULL,
    from_plan_uid character varying(255) NOT NULL,
    to_plan_uid character varying(255) NOT NULL,
    billing_interval character varying(20),
    status character varying(50) NOT NULL,
    reason text NOT NULL,
    actor_uid character varying(255),
    preview_idempotency_key character varying(255),
    execution_idempotency_key character varying(255),
    total_targets integer DEFAULT 0 NOT NULL,
    affected_count integer DEFAULT 0 NOT NULL,
    skipped_count integer DEFAULT 0 NOT NULL,
    succeeded_count integer DEFAULT 0 NOT NULL,
    failed_count integer DEFAULT 0 NOT NULL,
    estimated_billing_impact_cents integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone
);


ALTER TABLE public.bulk_subscription_change_jobs OWNER TO postgres;

--
-- Name: bulk_subscription_change_targets; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.bulk_subscription_change_targets (
    job_uid character varying(255) NOT NULL,
    user_uid character varying(255) NOT NULL,
    subscription_uid character varying(255),
    current_plan_uid character varying(255),
    target_plan_uid character varying(255) NOT NULL,
    current_billing_interval character varying(20),
    target_billing_interval character varying(20),
    status character varying(50) NOT NULL,
    reason text,
    estimated_impact_cents integer DEFAULT 0 NOT NULL,
    error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.bulk_subscription_change_targets OWNER TO postgres;

--
-- Name: business_commission_settings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.business_commission_settings (
    business_uid character varying(255) NOT NULL,
    agency_commission_percent numeric(7,4) DEFAULT 0 NOT NULL,
    agent_commission_percent numeric(7,4) DEFAULT 0 NOT NULL,
    updated_by character varying(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_business_commission_agency_percent CHECK (((agency_commission_percent >= (0)::numeric) AND (agency_commission_percent <= (100)::numeric))),
    CONSTRAINT chk_business_commission_agent_percent CHECK (((agent_commission_percent >= (0)::numeric) AND (agent_commission_percent <= (100)::numeric)))
);


ALTER TABLE public.business_commission_settings OWNER TO postgres;

--
-- Name: business_contracts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.business_contracts (
    uid character varying(255) NOT NULL,
    business_uid character varying(255) NOT NULL,
    billing_interval character varying(20) DEFAULT 'monthly'::character varying NOT NULL,
    price_cents integer DEFAULT 0 NOT NULL,
    max_seats integer DEFAULT 1 NOT NULL,
    max_saved_properties integer DEFAULT 0 NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    current_period_start timestamp with time zone NOT NULL,
    current_period_end timestamp with time zone NOT NULL,
    cancel_at_period_end boolean DEFAULT false NOT NULL,
    cancelled_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    retry_count integer DEFAULT 0 NOT NULL,
    last_retry_at timestamp with time zone,
    is_trial boolean DEFAULT false NOT NULL,
    price_includes_tax boolean DEFAULT true NOT NULL,
    free_of_charge boolean DEFAULT false NOT NULL,
    pending_payment_intent_id character varying(255)
);


ALTER TABLE public.business_contracts OWNER TO postgres;

--
-- Name: business_features; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.business_features (
    uid character varying(255) NOT NULL,
    business_uid character varying(255) NOT NULL,
    feature_key character varying(100) NOT NULL,
    price_cents_paid integer DEFAULT 0 NOT NULL,
    purchased_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.business_features OWNER TO postgres;

--
-- Name: business_portfolio_monthly_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.business_portfolio_monthly_history (
    business_uid character varying(255) NOT NULL,
    month_start date NOT NULL,
    manual_properties_count integer DEFAULT 0 NOT NULL,
    auto_imported_count integer DEFAULT 0 NOT NULL,
    total_properties_count integer DEFAULT 0 NOT NULL,
    max_properties integer DEFAULT 0 NOT NULL,
    estimated_value_min bigint DEFAULT 0 NOT NULL,
    estimated_value_max bigint DEFAULT 0 NOT NULL,
    estimated_value_property_count integer DEFAULT 0 NOT NULL,
    asking_value_total bigint DEFAULT 0 NOT NULL,
    asking_value_property_count integer DEFAULT 0 NOT NULL,
    agency_commission_percent numeric(7,4) DEFAULT 0 NOT NULL,
    agent_commission_percent numeric(7,4) DEFAULT 0 NOT NULL,
    estimated_agency_commission_min bigint DEFAULT 0 NOT NULL,
    estimated_agency_commission_max bigint DEFAULT 0 NOT NULL,
    estimated_agent_commission_min bigint DEFAULT 0 NOT NULL,
    estimated_agent_commission_max bigint DEFAULT 0 NOT NULL,
    asking_agency_commission_value bigint DEFAULT 0 NOT NULL,
    asking_agent_commission_value bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.business_portfolio_monthly_history OWNER TO postgres;

--
-- Name: business_saved_properties; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.business_saved_properties (
    business_uid character varying(255) NOT NULL,
    property_uid character varying(255) NOT NULL,
    saved_by character varying(255),
    saved_at timestamp with time zone DEFAULT now(),
    last_edited_by character varying(255),
    last_edited_at timestamp with time zone
);


ALTER TABLE public.business_saved_properties OWNER TO postgres;

--
-- Name: businesses; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.businesses (
    uid character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    stripe_customer_id character varying(255),
    admin_uid character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    has_used_trial boolean DEFAULT false,
    elorus_contact_id character varying(255),
    address_text character varying(255),
    geo_lat double precision,
    geo_lon double precision,
    lead_recipient_email character varying(255),
    geom public.geometry(Point,4326) GENERATED ALWAYS AS (
CASE
    WHEN ((geo_lon IS NOT NULL) AND (geo_lat IS NOT NULL)) THEN public.st_setsrid(public.st_makepoint(geo_lon, geo_lat), 4326)
    ELSE NULL::public.geometry
END) STORED
);


ALTER TABLE public.businesses OWNER TO postgres;

--
-- Name: call_requests; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.call_requests (
    uid character varying(255) NOT NULL,
    first_name character varying(100),
    last_name character varying(100),
    email character varying(255),
    phone character varying(50) NOT NULL,
    when_to_call character varying(50) NOT NULL,
    called_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    interested_in character varying(50) DEFAULT 'not_sure'::character varying NOT NULL
);


ALTER TABLE public.call_requests OWNER TO postgres;

--
-- Name: commercial_specifics; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.commercial_specifics (
    property_uid character varying(255) NOT NULL,
    loading_ramp boolean,
    cargo_elevator boolean,
    power_supply text[]
);


ALTER TABLE public.commercial_specifics OWNER TO postgres;

--
-- Name: contract_offers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.contract_offers (
    uid character varying(255) NOT NULL,
    business_uid character varying(255) NOT NULL,
    billing_interval character varying(20) NOT NULL,
    price_cents integer NOT NULL,
    max_seats integer NOT NULL,
    max_saved_properties integer NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    created_by character varying(255) NOT NULL,
    accepted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    price_includes_tax boolean DEFAULT true NOT NULL,
    cancelled_at timestamp with time zone,
    cancelled_by character varying(255),
    cancel_reason text,
    updated_at timestamp with time zone DEFAULT now(),
    last_resent_at timestamp with time zone,
    last_resend_actor_uid character varying(255),
    last_resend_reason text,
    last_resend_recipient text,
    free_of_charge boolean DEFAULT false NOT NULL
);


ALTER TABLE public.contract_offers OWNER TO postgres;

--
-- Name: credit_adjustments; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.credit_adjustments (
    uid character varying(255) NOT NULL,
    user_uid character varying(255) NOT NULL,
    operation character varying(20) NOT NULL,
    delta integer NOT NULL,
    balance_before integer NOT NULL,
    balance_after integer NOT NULL,
    credits_total integer NOT NULL,
    reason text NOT NULL,
    actor_uid character varying(255),
    idempotency_key character varying(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT credit_adjustments_operation_check CHECK (((operation)::text = ANY ((ARRAY['delta'::character varying, 'set'::character varying])::text[])))
);


ALTER TABLE public.credit_adjustments OWNER TO postgres;

--
-- Name: deleted_properties; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.deleted_properties (
    property_uid character varying(255) NOT NULL,
    item_type character varying(255),
    transaction_type character varying(255),
    subtype character varying(255),
    ad_creation_date timestamp with time zone,
    last_seen_at timestamp with time zone,
    days_listed integer,
    square_meters integer,
    final_asking_price integer,
    final_estimated_value integer,
    final_estimated_value_confidence numeric(5,4),
    final_grounded_estimated_value integer,
    final_grounded_confidence real,
    deletion_reason character varying(50) NOT NULL,
    deleted_at timestamp with time zone DEFAULT now() NOT NULL,
    admin_name text
);


ALTER TABLE public.deleted_properties OWNER TO postgres;

--
-- Name: detailed_report_forwards; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.detailed_report_forwards (
    uid character varying(255) NOT NULL,
    report_uid character varying(255) NOT NULL,
    sender_user_uid character varying(255) NOT NULL,
    recipient_business_uid character varying(255) NOT NULL,
    recipient_business_name character varying(255) NOT NULL,
    recipient_email character varying(255) NOT NULL,
    token_hash character varying(128) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    opened_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.detailed_report_forwards OWNER TO postgres;

--
-- Name: detailed_report_views; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.detailed_report_views (
    uid character varying(255) NOT NULL,
    report_uid character varying(255) NOT NULL,
    viewer_user_uid character varying(255),
    viewer_ip text,
    viewed_at timestamp with time zone DEFAULT now() NOT NULL,
    viewer_location text
);


ALTER TABLE public.detailed_report_views OWNER TO postgres;

--
-- Name: detailed_reports; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.detailed_reports (
    uid character varying(255) NOT NULL,
    property_uid character varying(255) NOT NULL,
    content jsonb NOT NULL,
    is_public boolean DEFAULT false NOT NULL,
    created_by_user_uid character varying(255),
    created_by_business_uid character varying(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    view_count integer DEFAULT 0 NOT NULL
);


ALTER TABLE public.detailed_reports OWNER TO postgres;

--
-- Name: estimated_value_snapshot; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.estimated_value_snapshot (
    property_uid character varying(255) NOT NULL,
    candidates jsonb,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.estimated_value_snapshot OWNER TO postgres;

--
-- Name: estimation_data; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.estimation_data (
    uid text NOT NULL,
    geo_premium numeric,
    local_pred_std numeric,
    local_pred_mean numeric,
    geo_premium_zscore numeric,
    asking_price_per_sqm numeric,
    residual_percentage numeric,
    predicted_price_per_sqm numeric,
    attr_price numeric
);


ALTER TABLE public.estimation_data OWNER TO postgres;

--
-- Name: feature_importance_rent; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.feature_importance_rent (
    feature_name text NOT NULL,
    mean_abs_shap_per_sqm double precision NOT NULL,
    mean_signed_shap_per_sqm double precision NOT NULL,
    importance_percentage double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT feature_importance_rent_importance_rank_check CHECK ((importance_rank > 0)),
    CONSTRAINT feature_importance_rent_mean_abs_check CHECK ((mean_abs_shap_per_sqm >= (0)::double precision)),
    CONSTRAINT feature_importance_rent_percentage_check CHECK (((importance_percentage >= (0)::double precision) AND (importance_percentage <= (100)::double precision)))
);


ALTER TABLE public.feature_importance_rent OWNER TO postgres;

--
-- Name: feature_importance_sell; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.feature_importance_sell (
    feature_name text NOT NULL,
    mean_abs_shap_per_sqm double precision NOT NULL,
    mean_signed_shap_per_sqm double precision NOT NULL,
    importance_percentage double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT feature_importance_sell_importance_rank_check CHECK ((importance_rank > 0)),
    CONSTRAINT feature_importance_sell_mean_abs_check CHECK ((mean_abs_shap_per_sqm >= (0)::double precision)),
    CONSTRAINT feature_importance_sell_percentage_check CHECK (((importance_percentage >= (0)::double precision) AND (importance_percentage <= (100)::double precision)))
);


ALTER TABLE public.feature_importance_sell OWNER TO postgres;

--
-- Name: features_interactions_rent; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.features_interactions_rent (
    feature_1 text NOT NULL,
    feature_2 text NOT NULL,
    mean_signed_interaction_per_sqm double precision NOT NULL,
    mean_abs_interaction_per_sqm double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT features_interactions_rent_distinct_features_check CHECK ((feature_1 <> feature_2)),
    CONSTRAINT features_interactions_rent_importance_rank_check CHECK ((importance_rank > 0)),
    CONSTRAINT features_interactions_rent_mean_abs_check CHECK ((mean_abs_interaction_per_sqm >= (0)::double precision))
);


ALTER TABLE public.features_interactions_rent OWNER TO postgres;

--
-- Name: features_interactions_sell; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.features_interactions_sell (
    feature_1 text NOT NULL,
    feature_2 text NOT NULL,
    mean_signed_interaction_per_sqm double precision NOT NULL,
    mean_abs_interaction_per_sqm double precision NOT NULL,
    importance_rank integer NOT NULL,
    calculated_at timestamp with time zone DEFAULT now() NOT NULL,
    model_sha256 character varying(64) NOT NULL,
    CONSTRAINT features_interactions_sell_distinct_features_check CHECK ((feature_1 <> feature_2)),
    CONSTRAINT features_interactions_sell_importance_rank_check CHECK ((importance_rank > 0)),
    CONSTRAINT features_interactions_sell_mean_abs_check CHECK ((mean_abs_interaction_per_sqm >= (0)::double precision))
);


ALTER TABLE public.features_interactions_sell OWNER TO postgres;

--
-- Name: global_stats; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.global_stats (
    stat_name character varying NOT NULL,
    mean numeric NOT NULL,
    std numeric NOT NULL,
    baseline_price_per_sqm double precision,
    training_r2 double precision,
    validation_r2 double precision,
    model_sha256 character varying(64),
    model_trained_at timestamp with time zone
);


ALTER TABLE public.global_stats OWNER TO postgres;

--
-- Name: COLUMN global_stats.baseline_price_per_sqm; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.global_stats.baseline_price_per_sqm IS 'Latest model baseline; populated on Price_stats_sell or Price_stats_rent.';


--
-- Name: COLUMN global_stats.training_r2; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.global_stats.training_r2 IS 'Latest training R-squared; populated on Price_stats_sell or Price_stats_rent.';


--
-- Name: COLUMN global_stats.validation_r2; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.global_stats.validation_r2 IS 'Latest validation R-squared; populated on Price_stats_sell or Price_stats_rent.';


--
-- Name: COLUMN global_stats.model_sha256; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.global_stats.model_sha256 IS 'Latest model SHA-256; populated on Price_stats_sell or Price_stats_rent.';


--
-- Name: COLUMN global_stats.model_trained_at; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.global_stats.model_trained_at IS 'Latest model training time; populated on Price_stats_sell or Price_stats_rent.';

--
-- Name: gov_area_geocoding; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.gov_area_geocoding (
    id bigint NOT NULL,
    municipality_gov text NOT NULL,
    district_gov text DEFAULT ''::text NOT NULL,
    admin_cache_key text,
    admin_name text,
    admin_level integer,
    centroid_lat double precision,
    centroid_lon double precision,
    resolution_method text,
    resolved_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.gov_area_geocoding OWNER TO postgres;

--
-- Name: gov_area_geocoding_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.gov_area_geocoding_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.gov_area_geocoding_id_seq OWNER TO postgres;

--
-- Name: gov_area_geocoding_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.gov_area_geocoding_id_seq OWNED BY public.gov_area_geocoding.id;


--
-- Name: gov_verified_sales; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.gov_verified_sales (
    id bigint NOT NULL,
    row_hash text NOT NULL,
    prefecture text,
    municipality text NOT NULL,
    district text,
    in_city_plan boolean,
    property_category text,
    item_type text NOT NULL,
    subtype text,
    facade_count integer,
    zone_value integer,
    main_area_sqm double precision,
    auxiliary_area_sqm double precision,
    construction_year integer,
    ownership_type_building text,
    co_ownership_pct_building double precision,
    special_conditions text,
    floor_level text,
    landplot_area_sqm double precision,
    ownership_type_land text,
    co_ownership_pct_land double precision,
    total_built_area_sqm double precision,
    contract_date date NOT NULL,
    verified_sale_price integer NOT NULL,
    price_per_sqm double precision,
    zone_value_ratio double precision,
    quality_tier text DEFAULT 'pending'::text,
    quality_score double precision DEFAULT 0,
    quality_flags text[],
    admin_cache_key text,
    admin_name text,
    admin_level integer,
    centroid_lat double precision,
    centroid_lon double precision,
    source_file text,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.gov_verified_sales OWNER TO postgres;

--
-- Name: gov_verified_sales_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.gov_verified_sales_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.gov_verified_sales_id_seq OWNER TO postgres;

--
-- Name: gov_verified_sales_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.gov_verified_sales_id_seq OWNED BY public.gov_verified_sales.id;


--
-- Name: individual_plan_features; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.individual_plan_features (
    plan_uid character varying(255) NOT NULL,
    feature_key character varying(100) NOT NULL
);


ALTER TABLE public.individual_plan_features OWNER TO postgres;

--
-- Name: individual_plans; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.individual_plans (
    uid character varying(255) NOT NULL,
    name character varying(100) NOT NULL,
    monthly_price_cents integer DEFAULT 0 NOT NULL,
    yearly_price_cents integer DEFAULT 0 NOT NULL,
    credits_per_cycle integer DEFAULT 0 NOT NULL,
    max_saved_properties integer DEFAULT 0 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    signup_bonus_credits integer DEFAULT 0 NOT NULL,
    price_includes_tax boolean DEFAULT true NOT NULL
);


ALTER TABLE public.individual_plans OWNER TO postgres;

--
-- Name: individual_subscriptions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.individual_subscriptions (
    uid character varying(255) NOT NULL,
    user_uid character varying(255) NOT NULL,
    plan_uid character varying(255) NOT NULL,
    billing_interval character varying(20) DEFAULT 'monthly'::character varying NOT NULL,
    status character varying(50) DEFAULT 'active'::character varying NOT NULL,
    current_period_start timestamp with time zone NOT NULL,
    current_period_end timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    retry_count integer DEFAULT 0 NOT NULL,
    last_retry_at timestamp with time zone,
    pending_plan_uid character varying(255),
    pending_billing_interval character varying(20),
    free_of_charge boolean DEFAULT false NOT NULL,
    pending_payment_intent_id character varying(255)
);


ALTER TABLE public.individual_subscriptions OWNER TO postgres;

--
-- Name: invoice_tasks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.invoice_tasks (
    uid character varying(255) NOT NULL,
    payment_uid character varying(255) NOT NULL,
    owner_type character varying(20) NOT NULL,
    owner_uid character varying(255) NOT NULL,
    document_type character varying(20) NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    elorus_doc_id character varying(255),
    elorus_doc_type character varying(20),
    error text,
    retry_count integer DEFAULT 0 NOT NULL,
    last_retry_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.invoice_tasks OWNER TO postgres;

--
-- Name: landplot_specifics; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.landplot_specifics (
    property_uid character varying(255) NOT NULL,
    building_ratio numeric(8,4) DEFAULT NULL::numeric,
    coverage_factor numeric(8,4) DEFAULT NULL::numeric,
    facade_meters integer,
    buildable boolean,
    in_city_plan boolean,
    angle text[]
);


ALTER TABLE public.landplot_specifics OWNER TO postgres;

--
-- Name: market_premium_policies; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.market_premium_policies (
    id bigint NOT NULL,
    market_premium_percent numeric(10,4) NOT NULL,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    note text
);


ALTER TABLE public.market_premium_policies OWNER TO postgres;

--
-- Name: market_premium_policies_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.market_premium_policies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.market_premium_policies_id_seq OWNER TO postgres;

--
-- Name: market_premium_policies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.market_premium_policies_id_seq OWNED BY public.market_premium_policies.id;


--
-- Name: marketabillity_factor; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.marketabillity_factor (
    area text NOT NULL,
    main_street text NOT NULL,
    geo_lat_start double precision,
    geo_lon_start double precision,
    geo_lat_end double precision,
    geo_lon_end double precision,
    marketabillity_factor numeric(8,1) DEFAULT NULL::numeric,
    geom public.geometry(Geometry,4326)
);


ALTER TABLE public.marketabillity_factor OWNER TO postgres;

--
-- Name: marketabillity_factor_batches; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.marketabillity_factor_batches (
    external_url text NOT NULL,
    area text NOT NULL,
    is_processed boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    processing_started_at timestamp with time zone,
    processing_owner text
);


ALTER TABLE public.marketabillity_factor_batches OWNER TO postgres;

--
-- Name: notification_log; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.notification_log (
    id bigint NOT NULL,
    entity_type character varying(50) NOT NULL,
    entity_uid character varying(255) NOT NULL,
    notification_type character varying(100) NOT NULL,
    sent_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.notification_log OWNER TO postgres;

--
-- Name: notification_log_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.notification_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.notification_log_id_seq OWNER TO postgres;

--
-- Name: notification_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.notification_log_id_seq OWNED BY public.notification_log.id;


--
-- Name: payment_methods; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.payment_methods (
    uid character varying(255) NOT NULL,
    owner_type character varying(20) NOT NULL,
    owner_uid character varying(255) NOT NULL,
    stripe_payment_method_id character varying(255) NOT NULL,
    card_last4 character varying(4),
    card_brand character varying(50),
    exp_month smallint,
    exp_year smallint,
    is_default boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    pm_type character varying(50) DEFAULT 'card'::character varying NOT NULL
);


ALTER TABLE public.payment_methods OWNER TO postgres;

--
-- Name: payment_refunds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.payment_refunds (
    uid character varying(255) NOT NULL,
    payment_uid character varying(255) NOT NULL,
    amount_cents integer NOT NULL,
    mode character varying(20) NOT NULL,
    status character varying(50) NOT NULL,
    stripe_refund_id character varying(255),
    reason text NOT NULL,
    actor_uid character varying(255),
    idempotency_key character varying(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT payment_refunds_amount_cents_check CHECK ((amount_cents > 0)),
    CONSTRAINT payment_refunds_mode_check CHECK (((mode)::text = ANY ((ARRAY['stripe'::character varying, 'manual'::character varying, 'auto'::character varying])::text[]))),
    CONSTRAINT payment_refunds_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'succeeded'::character varying, 'failed'::character varying, 'manual'::character varying])::text[])))
);


ALTER TABLE public.payment_refunds OWNER TO postgres;

--
-- Name: payments; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.payments (
    uid character varying(255) NOT NULL,
    owner_type character varying(20) NOT NULL,
    owner_uid character varying(255) NOT NULL,
    stripe_payment_intent_id character varying(255),
    amount_cents integer NOT NULL,
    currency character varying(10) DEFAULT 'eur'::character varying NOT NULL,
    status character varying(50) DEFAULT 'pending'::character varying NOT NULL,
    payment_type character varying(50) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    net_amount_cents integer DEFAULT 0 NOT NULL,
    tax_amount_cents integer DEFAULT 0 NOT NULL,
    payment_method_uid character varying(255),
    stripe_payment_method_id character varying(255),
    subscription_uid character varying(255)
);


ALTER TABLE public.payments OWNER TO postgres;

--
-- Name: properties; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.properties (
    uid character varying(255) NOT NULL,
    external_id character varying(255),
    item_type character varying(50) NOT NULL,
    transaction_type character varying(50) NOT NULL,
    asking_price integer,
    ad_creation_date date,
    subtype text,
    updated_at timestamp with time zone DEFAULT now(),
    source_url text,
    entry_created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.properties OWNER TO postgres;

--
-- Name: properties_analysis_rent; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.properties_analysis_rent (
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
    explained_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.properties_analysis_rent OWNER TO postgres;

--
-- Name: properties_analysis_sell; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.properties_analysis_sell (
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
    explained_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.properties_analysis_sell OWNER TO postgres;

--
-- Name: properties_elo; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.properties_elo (
    property_uid character varying(255) NOT NULL,
    elo integer NOT NULL
);


ALTER TABLE public.properties_elo OWNER TO postgres;

--
-- Name: property_geo_data; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.property_geo_data (
    property_uid character varying(255) NOT NULL,
    geo_lat double precision,
    geo_lon double precision,
    geom public.geometry(Point,4326) GENERATED ALWAYS AS (public.st_setsrid(public.st_makepoint(geo_lon, geo_lat), 4326)) STORED,
    address_text character varying(255) DEFAULT NULL::character varying
);


ALTER TABLE public.property_geo_data OWNER TO postgres;


--
-- Name: property_images; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.property_images (
    property_uid character varying(255) NOT NULL,
    position integer DEFAULT 0 NOT NULL,
    image_url text,
    image_data bytea,
    mime_type character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT property_images_one_source_check CHECK ((((image_url IS NOT NULL))::integer + ((image_data IS NOT NULL))::integer) = 1)
);


ALTER TABLE public.property_images OWNER TO postgres;

--
-- Name: property_rankings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.property_rankings (
    property_uid character varying(255) NOT NULL,
    predicted_price_per_sqm double precision,
    estimated_value integer,
    geo_premium double precision,
    geo_premium_zscore double precision,
    elo double precision,
    model_trained_at timestamp with time zone,
    computed_at timestamp with time zone DEFAULT now() NOT NULL,
    estimated_value_confidence double precision,
    ml_local_spread double precision,
    velocity_score double precision,
    model_revision bigint DEFAULT 0 NOT NULL
);


ALTER TABLE public.property_rankings OWNER TO postgres;

--
-- Name: property_realtors; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.property_realtors (
    property_uid character varying(255) NOT NULL,
    realtor_uid text NOT NULL
);


ALTER TABLE public.property_realtors OWNER TO postgres;

--
-- Name: push_subscriptions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.push_subscriptions (
    id character varying(26) NOT NULL,
    user_uid character varying(255) NOT NULL,
    endpoint text NOT NULL,
    p256dh text NOT NULL,
    auth text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.push_subscriptions OWNER TO postgres;

--
-- Name: ranking_models; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ranking_models (
    id integer NOT NULL,
    item_type text NOT NULL,
    subtype text NOT NULL,
    transaction text NOT NULL,
    phase2_model bytea NOT NULL,
    category_mappings text NOT NULL,
    geo_premiums jsonb NOT NULL,
    phase2_predictions jsonb NOT NULL,
    standard_values jsonb NOT NULL,
    feature_manifest jsonb NOT NULL,
    trained_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.ranking_models OWNER TO postgres;

--
-- Name: ranking_models_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.ranking_models_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ranking_models_id_seq OWNER TO postgres;

--
-- Name: ranking_models_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.ranking_models_id_seq OWNED BY public.ranking_models.id;


--
-- Name: realtors; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.realtors (
    uid text NOT NULL,
    title text NOT NULL
);


ALTER TABLE public.realtors OWNER TO postgres;

--
-- Name: residential_specifics; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.residential_specifics (
    property_uid character varying(255) NOT NULL,
    bedrooms integer,
    pool boolean,
    garden boolean
);


ALTER TABLE public.residential_specifics OWNER TO postgres;

--
-- Name: scheduler_job_runs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.scheduler_job_runs (
    run_uid text NOT NULL,
    job_uid text NOT NULL,
    status text NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    finished_at timestamp with time zone,
    error_msg text,
    run_kind text DEFAULT 'scheduled'::text NOT NULL,
    chain_run_uid text,
    chain_step integer,
    scheduled_for timestamp with time zone
);


ALTER TABLE public.scheduler_job_runs OWNER TO postgres;

--
-- Name: scheduler_job_scrape_sources; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.scheduler_job_scrape_sources (
    job_uid text NOT NULL,
    source_order integer NOT NULL,
    urls text[] NOT NULL,
    scrape_profile_hint text
);


ALTER TABLE public.scheduler_job_scrape_sources OWNER TO postgres;

--
-- Name: scheduler_jobs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.scheduler_jobs (
    job_uid text NOT NULL,
    name text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    repeat_every integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    retry_count integer DEFAULT 0 NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_started_at timestamp with time zone,
    last_finished_at timestamp with time zone,
    next_run_at timestamp with time zone DEFAULT now() NOT NULL,
    business_uid text,
    scrape_profile_hint text,
    locked_by text,
    lock_token text,
    locked_until timestamp with time zone,
    active_chain_run_uid text,
    active_chain_step integer,
    chain_original_next_run_at timestamp with time zone,
    chain_consumes_scheduled_run boolean DEFAULT false NOT NULL,
    CONSTRAINT scheduler_jobs_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'running'::text, 'finished'::text, 'failed'::text, 'paused'::text])))
);


ALTER TABLE public.scheduler_jobs OWNER TO postgres;

--
-- Name: scheduler_maintenance_chain_runs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.scheduler_maintenance_chain_runs (
    chain_run_uid text NOT NULL,
    trigger_run_uid text NOT NULL,
    trigger_job_uid text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    current_step integer DEFAULT 0 NOT NULL,
    total_steps integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    error_msg text,
    CONSTRAINT scheduler_maintenance_chain_runs_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'running'::text, 'failed'::text, 'finished'::text])))
);


ALTER TABLE public.scheduler_maintenance_chain_runs OWNER TO postgres;

--
-- Name: segment_metadata; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.segment_metadata (
    singleton boolean DEFAULT true NOT NULL,
    generated_at timestamp with time zone NOT NULL,
    report jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT segment_metadata_singleton CHECK ((singleton = true))
);


ALTER TABLE public.segment_metadata OWNER TO postgres;

--
-- Name: signup_whitelist; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.signup_whitelist (
    email text NOT NULL,
    created_by character varying(255) DEFAULT 'system'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    note text
);


ALTER TABLE public.signup_whitelist OWNER TO postgres;

--
-- Name: template_public_valuations_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.template_public_valuations_history (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL
);


ALTER TABLE public.template_public_valuations_history OWNER TO postgres;

--
-- Name: user_businesses; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_businesses (
    user_uid character varying(255) NOT NULL,
    business_uid character varying(255) NOT NULL,
    role character varying(50) DEFAULT 'agent'::character varying NOT NULL,
    joined_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.user_businesses OWNER TO postgres;

--
-- Name: user_credentials; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_credentials (
    email text NOT NULL,
    password character varying(60) NOT NULL,
    admin boolean NOT NULL,
    uid character varying(255) NOT NULL
);


ALTER TABLE public.user_credentials OWNER TO postgres;

--
-- Name: user_credits; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_credits (
    user_uid character varying(255) NOT NULL,
    credits_remaining integer DEFAULT 0 NOT NULL,
    credits_total integer DEFAULT 0 NOT NULL,
    cycle_start timestamp with time zone NOT NULL,
    cycle_end timestamp with time zone NOT NULL,
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.user_credits OWNER TO postgres;

--
-- Name: user_feedback; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_feedback (
    uid character varying(255) NOT NULL,
    user_uid character varying(255) NOT NULL,
    active_role character varying(50) NOT NULL,
    active_business_uid character varying(255),
    feedback_type character varying(50) NOT NULL,
    subject character varying(255),
    message text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.user_feedback OWNER TO postgres;

--
-- Name: user_properties; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_properties (
    property_uid character varying(255) NOT NULL,
    user_uid character varying(255) NOT NULL,
    is_published boolean,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.user_properties OWNER TO postgres;

--
-- Name: user_saved_properties; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_saved_properties (
    user_uid character varying(255) NOT NULL,
    property_uid character varying(255) NOT NULL,
    saved_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.user_saved_properties OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    uid character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    display_name text,
    roles text[] DEFAULT ARRAY['guest'::text] NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    email_verified boolean DEFAULT false NOT NULL,
    stripe_customer_id character varying(255),
    elorus_contact_id character varying(255),
    disabled_at timestamp with time zone,
    disabled_reason text,
    disabled_by character varying(255)
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: valuation_adjustment_policies; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuation_adjustment_policies (
    id bigint NOT NULL,
    estimated_value_percent numeric(10,4) NOT NULL,
    effective_from timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by character varying(255),
    note text,
    item_type character varying(255),
    transaction_type character varying(255),
    subtype character varying(255) DEFAULT NULL::character varying,
    policy_month date
);


ALTER TABLE public.valuation_adjustment_policies OWNER TO postgres;

--
-- Name: COLUMN valuation_adjustment_policies.policy_month; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.valuation_adjustment_policies.policy_month IS 'First day of the UTC month for automated segment-metadata policies. NULL for manual/global policies.';


--
-- Name: valuation_adjustment_policies_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.valuation_adjustment_policies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.valuation_adjustment_policies_id_seq OWNER TO postgres;

--
-- Name: valuation_adjustment_policies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.valuation_adjustment_policies_id_seq OWNED BY public.valuation_adjustment_policies.id;


--
-- Name: valuation_backtest_jobs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuation_backtest_jobs (
    job_uid text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    progress_current integer DEFAULT 0 NOT NULL,
    progress_total integer DEFAULT 0 NOT NULL,
    limit_count integer DEFAULT 500 NOT NULL,
    item_type text DEFAULT ''::text NOT NULL,
    transaction_type text DEFAULT ''::text NOT NULL,
    subtype text DEFAULT ''::text NOT NULL,
    use_source_asking_price boolean DEFAULT false NOT NULL,
    apply_adjustments boolean DEFAULT false NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    generated_at timestamp with time zone,
    total_sources integer DEFAULT 0 NOT NULL,
    successful integer DEFAULT 0 NOT NULL,
    failed integer DEFAULT 0 NOT NULL,
    error_msg text,
    report jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.valuation_backtest_jobs OWNER TO postgres;

--
-- Name: valuations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    grounded_estimated_value integer,
    grounded_confidence real,
    grounded_mae_eff real,
    grounded_computed_at timestamp with time zone,
    comparables_mae_eff real,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations OWNER TO postgres;

--
-- Name: valuations_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
)
PARTITION BY RANGE (archived_at);


ALTER TABLE public.valuations_history OWNER TO postgres;

--
-- Name: valuations_history_default; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_default (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_default OWNER TO postgres;

--
-- Name: valuations_history_p20260201; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260201 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260201 OWNER TO postgres;

--
-- Name: valuations_history_p20260301; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260301 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260301 OWNER TO postgres;

--
-- Name: valuations_history_p20260401; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260401 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260401 OWNER TO postgres;

--
-- Name: valuations_history_p20260501; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260501 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260501 OWNER TO postgres;

--
-- Name: valuations_history_p20260601; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260601 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260601 OWNER TO postgres;

--
-- Name: valuations_history_p20260701; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260701 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260701 OWNER TO postgres;

--
-- Name: valuations_history_p20260801; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260801 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260801 OWNER TO postgres;

--
-- Name: valuations_history_p20260901; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20260901 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20260901 OWNER TO postgres;

--
-- Name: valuations_history_p20261001; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20261001 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20261001 OWNER TO postgres;

--
-- Name: valuations_history_p20261101; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.valuations_history_p20261101 (
    property_uid character varying(255) NOT NULL,
    asking_price integer NOT NULL,
    real_value integer NOT NULL,
    estimated_value integer NOT NULL,
    created_at timestamp with time zone,
    archived_at timestamp with time zone NOT NULL,
    estimated_value_confidence numeric(5,4) DEFAULT 0 NOT NULL,
    effective_estimated_value integer,
    effective_estimated_min integer,
    effective_estimated_max integer,
    effective_confidence numeric(5,4),
    estimated_value_adjustment_percent numeric(10,4),
    market_premium_percent numeric(10,4)
);


ALTER TABLE public.valuations_history_p20261101 OWNER TO postgres;

--
-- Name: white_label_settings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.white_label_settings (
    business_uid character varying(255) NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    brand_name character varying(255) NOT NULL,
    brand_site_url text DEFAULT ''::text NOT NULL,
    icon_file character varying(255),
    logo_mime_type character varying(100),
    logo_size_bytes integer,
    logo_bytes bytea,
    settings jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT white_label_logo_size CHECK (((logo_bytes IS NULL) OR (octet_length(logo_bytes) <= 2097152)))
);


ALTER TABLE public.white_label_settings OWNER TO postgres;

--
-- Name: worker_artifact_revisions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.worker_artifact_revisions (
    artifact text NOT NULL,
    revision bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT worker_artifact_revisions_artifact_check CHECK ((artifact = ANY (ARRAY['ranking_models'::text, 'segment_metadata'::text]))),
    CONSTRAINT worker_artifact_revisions_revision_check CHECK ((revision >= 0))
);


ALTER TABLE public.worker_artifact_revisions OWNER TO postgres;

--
-- Name: valuations_history_default; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_default DEFAULT;


--
-- Name: valuations_history_p20260201; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260201 FOR VALUES FROM ('2026-02-01 00:00:00+00') TO ('2026-03-01 00:00:00+00');


--
-- Name: valuations_history_p20260301; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260301 FOR VALUES FROM ('2026-03-01 00:00:00+00') TO ('2026-04-01 00:00:00+00');


--
-- Name: valuations_history_p20260401; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260401 FOR VALUES FROM ('2026-04-01 00:00:00+00') TO ('2026-05-01 00:00:00+00');


--
-- Name: valuations_history_p20260501; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260501 FOR VALUES FROM ('2026-05-01 00:00:00+00') TO ('2026-06-01 00:00:00+00');


--
-- Name: valuations_history_p20260601; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260601 FOR VALUES FROM ('2026-06-01 00:00:00+00') TO ('2026-07-01 00:00:00+00');


--
-- Name: valuations_history_p20260701; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260701 FOR VALUES FROM ('2026-07-01 00:00:00+00') TO ('2026-08-01 00:00:00+00');


--
-- Name: valuations_history_p20260801; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260801 FOR VALUES FROM ('2026-08-01 00:00:00+00') TO ('2026-09-01 00:00:00+00');


--
-- Name: valuations_history_p20260901; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20260901 FOR VALUES FROM ('2026-09-01 00:00:00+00') TO ('2026-10-01 00:00:00+00');


--
-- Name: valuations_history_p20261001; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20261001 FOR VALUES FROM ('2026-10-01 00:00:00+00') TO ('2026-11-01 00:00:00+00');


--
-- Name: valuations_history_p20261101; Type: TABLE ATTACH; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations_history ATTACH PARTITION public.valuations_history_p20261101 FOR VALUES FROM ('2026-11-01 00:00:00+00') TO ('2026-12-01 00:00:00+00');


--
-- Name: area_market_premiums id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.area_market_premiums ALTER COLUMN id SET DEFAULT nextval('public.area_market_premiums_id_seq'::regclass);


--
-- Name: gov_area_geocoding id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gov_area_geocoding ALTER COLUMN id SET DEFAULT nextval('public.gov_area_geocoding_id_seq'::regclass);


--
-- Name: gov_verified_sales id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gov_verified_sales ALTER COLUMN id SET DEFAULT nextval('public.gov_verified_sales_id_seq'::regclass);


--
-- Name: market_premium_policies id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.market_premium_policies ALTER COLUMN id SET DEFAULT nextval('public.market_premium_policies_id_seq'::regclass);


--
-- Name: notification_log id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notification_log ALTER COLUMN id SET DEFAULT nextval('public.notification_log_id_seq'::regclass);


--
-- Name: ranking_models id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ranking_models ALTER COLUMN id SET DEFAULT nextval('public.ranking_models_id_seq'::regclass);


--
-- Name: valuation_adjustment_policies id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuation_adjustment_policies ALTER COLUMN id SET DEFAULT nextval('public.valuation_adjustment_policies_id_seq'::regclass);


--
-- Name: admin_action_records admin_action_idempotency_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_action_records
    ADD CONSTRAINT admin_action_idempotency_unique UNIQUE (action, idempotency_key);


--
-- Name: admin_action_records admin_action_records_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_action_records
    ADD CONSTRAINT admin_action_records_pkey PRIMARY KEY (uid);


--
-- Name: admin_active_user_snapshots admin_active_user_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_active_user_snapshots
    ADD CONSTRAINT admin_active_user_snapshots_pkey PRIMARY KEY (sampled_at);


--
-- Name: admin_audit_events admin_audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_audit_events
    ADD CONSTRAINT admin_audit_events_pkey PRIMARY KEY (uid);


--
-- Name: admin_boundaries admin_boundaries_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_boundaries
    ADD CONSTRAINT admin_boundaries_pkey PRIMARY KEY (cache_key);


--
-- Name: admin_boundary_negatives admin_boundary_negatives_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_boundary_negatives
    ADD CONSTRAINT admin_boundary_negatives_pkey PRIMARY KEY (cache_key);


--
-- Name: admin_charge_waivers admin_charge_waiver_idempotency_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_charge_waivers
    ADD CONSTRAINT admin_charge_waiver_idempotency_unique UNIQUE (operation, idempotency_key);


--
-- Name: admin_charge_waivers admin_charge_waivers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.admin_charge_waivers
    ADD CONSTRAINT admin_charge_waivers_pkey PRIMARY KEY (uid);


--
-- Name: area_market_premiums area_market_premiums_admin_name_item_type_subtype_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.area_market_premiums
    ADD CONSTRAINT area_market_premiums_admin_name_item_type_subtype_key UNIQUE (admin_name, item_type, subtype);


--
-- Name: area_market_premiums area_market_premiums_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.area_market_premiums
    ADD CONSTRAINT area_market_premiums_pkey PRIMARY KEY (id);


--
-- Name: auth_providers auth_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_providers
    ADD CONSTRAINT auth_providers_pkey PRIMARY KEY (uid);


--
-- Name: base_attributes base_attributes_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_attributes
    ADD CONSTRAINT base_attributes_pkey PRIMARY KEY (property_uid);


--
-- Name: billing_preferences billing_preferences_owner_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.billing_preferences
    ADD CONSTRAINT billing_preferences_owner_unique UNIQUE (owner_type, owner_uid);


--
-- Name: billing_preferences billing_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.billing_preferences
    ADD CONSTRAINT billing_preferences_pkey PRIMARY KEY (uid);


--
-- Name: building_details building_details_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.building_details
    ADD CONSTRAINT building_details_pkey PRIMARY KEY (property_uid);


--
-- Name: bulk_subscription_change_jobs bulk_subscription_change_jobs_execution_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bulk_subscription_change_jobs
    ADD CONSTRAINT bulk_subscription_change_jobs_execution_idempotency_key_key UNIQUE (execution_idempotency_key);


--
-- Name: bulk_subscription_change_jobs bulk_subscription_change_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bulk_subscription_change_jobs
    ADD CONSTRAINT bulk_subscription_change_jobs_pkey PRIMARY KEY (uid);


--
-- Name: bulk_subscription_change_jobs bulk_subscription_change_jobs_preview_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bulk_subscription_change_jobs
    ADD CONSTRAINT bulk_subscription_change_jobs_preview_idempotency_key_key UNIQUE (preview_idempotency_key);


--
-- Name: bulk_subscription_change_targets bulk_subscription_change_targets_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bulk_subscription_change_targets
    ADD CONSTRAINT bulk_subscription_change_targets_pkey PRIMARY KEY (job_uid, user_uid);


--
-- Name: business_commission_settings business_commission_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_commission_settings
    ADD CONSTRAINT business_commission_settings_pkey PRIMARY KEY (business_uid);


--
-- Name: business_contracts business_contracts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_contracts
    ADD CONSTRAINT business_contracts_pkey PRIMARY KEY (uid);


--
-- Name: business_features business_features_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_features
    ADD CONSTRAINT business_features_pkey PRIMARY KEY (uid);


--
-- Name: business_portfolio_monthly_history business_portfolio_monthly_history_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_portfolio_monthly_history
    ADD CONSTRAINT business_portfolio_monthly_history_pkey PRIMARY KEY (business_uid, month_start);


--
-- Name: business_saved_properties business_saved_properties_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_saved_properties
    ADD CONSTRAINT business_saved_properties_pkey PRIMARY KEY (business_uid, property_uid);


--
-- Name: businesses businesses_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.businesses
    ADD CONSTRAINT businesses_pkey PRIMARY KEY (uid);


--
-- Name: businesses businesses_stripe_customer_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.businesses
    ADD CONSTRAINT businesses_stripe_customer_id_key UNIQUE (stripe_customer_id);


--
-- Name: call_requests call_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.call_requests
    ADD CONSTRAINT call_requests_pkey PRIMARY KEY (uid);


--
-- Name: commercial_specifics commercial_specifics_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.commercial_specifics
    ADD CONSTRAINT commercial_specifics_pkey PRIMARY KEY (property_uid);


--
-- Name: contract_offers contract_offers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contract_offers
    ADD CONSTRAINT contract_offers_pkey PRIMARY KEY (uid);


--
-- Name: credit_adjustments credit_adjustments_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.credit_adjustments
    ADD CONSTRAINT credit_adjustments_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: credit_adjustments credit_adjustments_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.credit_adjustments
    ADD CONSTRAINT credit_adjustments_pkey PRIMARY KEY (uid);


--
-- Name: detailed_report_forwards detailed_report_forwards_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_forwards
    ADD CONSTRAINT detailed_report_forwards_pkey PRIMARY KEY (uid);


--
-- Name: detailed_report_forwards detailed_report_forwards_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_forwards
    ADD CONSTRAINT detailed_report_forwards_token_hash_key UNIQUE (token_hash);


--
-- Name: detailed_report_views detailed_report_views_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_views
    ADD CONSTRAINT detailed_report_views_pkey PRIMARY KEY (uid);


--
-- Name: detailed_reports detailed_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_reports
    ADD CONSTRAINT detailed_reports_pkey PRIMARY KEY (uid);


--
-- Name: estimated_value_snapshot estimated_value_snapshot_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.estimated_value_snapshot
    ADD CONSTRAINT estimated_value_snapshot_pkey PRIMARY KEY (property_uid);


--
-- Name: estimation_data estimation_data_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.estimation_data
    ADD CONSTRAINT estimation_data_pkey PRIMARY KEY (uid);


--
-- Name: feature_importance_rent feature_importance_rent_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.feature_importance_rent
    ADD CONSTRAINT feature_importance_rent_pkey PRIMARY KEY (feature_name);


--
-- Name: feature_importance_sell feature_importance_sell_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.feature_importance_sell
    ADD CONSTRAINT feature_importance_sell_pkey PRIMARY KEY (feature_name);


--
-- Name: features_interactions_rent features_interactions_rent_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.features_interactions_rent
    ADD CONSTRAINT features_interactions_rent_pkey PRIMARY KEY (feature_1, feature_2);


--
-- Name: features_interactions_sell features_interactions_sell_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.features_interactions_sell
    ADD CONSTRAINT features_interactions_sell_pkey PRIMARY KEY (feature_1, feature_2);


--
-- Name: global_stats global_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.global_stats
    ADD CONSTRAINT global_stats_pkey PRIMARY KEY (stat_name);


--
-- Name: gov_area_geocoding gov_area_geocoding_municipality_gov_district_gov_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gov_area_geocoding
    ADD CONSTRAINT gov_area_geocoding_municipality_gov_district_gov_key UNIQUE (municipality_gov, district_gov);


--
-- Name: gov_area_geocoding gov_area_geocoding_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gov_area_geocoding
    ADD CONSTRAINT gov_area_geocoding_pkey PRIMARY KEY (id);


--
-- Name: gov_verified_sales gov_verified_sales_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gov_verified_sales
    ADD CONSTRAINT gov_verified_sales_pkey PRIMARY KEY (id);


--
-- Name: individual_plan_features individual_plan_features_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_plan_features
    ADD CONSTRAINT individual_plan_features_pkey PRIMARY KEY (plan_uid, feature_key);


--
-- Name: individual_plans individual_plans_name_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_plans
    ADD CONSTRAINT individual_plans_name_key UNIQUE (name);


--
-- Name: individual_plans individual_plans_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_plans
    ADD CONSTRAINT individual_plans_pkey PRIMARY KEY (uid);


--
-- Name: individual_subscriptions individual_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_subscriptions
    ADD CONSTRAINT individual_subscriptions_pkey PRIMARY KEY (uid);


--
-- Name: invoice_tasks invoice_tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.invoice_tasks
    ADD CONSTRAINT invoice_tasks_pkey PRIMARY KEY (uid);


--
-- Name: landplot_specifics landplot_specifics_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.landplot_specifics
    ADD CONSTRAINT landplot_specifics_pkey PRIMARY KEY (property_uid);


--
-- Name: market_premium_policies market_premium_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.market_premium_policies
    ADD CONSTRAINT market_premium_policies_pkey PRIMARY KEY (id);


--
-- Name: marketabillity_factor_batches marketabillity_factor_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.marketabillity_factor_batches
    ADD CONSTRAINT marketabillity_factor_batches_pkey PRIMARY KEY (external_url);


--
-- Name: notification_log notification_log_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notification_log
    ADD CONSTRAINT notification_log_pkey PRIMARY KEY (id);


--
-- Name: payment_methods payment_methods_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment_methods
    ADD CONSTRAINT payment_methods_pkey PRIMARY KEY (uid);


--
-- Name: payment_methods payment_methods_stripe_payment_method_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment_methods
    ADD CONSTRAINT payment_methods_stripe_payment_method_id_key UNIQUE (stripe_payment_method_id);


--
-- Name: payment_refunds payment_refunds_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment_refunds
    ADD CONSTRAINT payment_refunds_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: payment_refunds payment_refunds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment_refunds
    ADD CONSTRAINT payment_refunds_pkey PRIMARY KEY (uid);


--
-- Name: payment_refunds payment_refunds_stripe_refund_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment_refunds
    ADD CONSTRAINT payment_refunds_stripe_refund_id_key UNIQUE (stripe_refund_id);


--
-- Name: payments payments_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (uid);


--
-- Name: payments payments_stripe_payment_intent_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payments
    ADD CONSTRAINT payments_stripe_payment_intent_id_key UNIQUE (stripe_payment_intent_id);


--
-- Name: properties_elo properties_elo_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties_elo
    ADD CONSTRAINT properties_elo_pkey PRIMARY KEY (property_uid);


--
-- Name: properties properties_external_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties
    ADD CONSTRAINT properties_external_id_key UNIQUE (external_id);


--
-- Name: properties properties_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties
    ADD CONSTRAINT properties_pkey PRIMARY KEY (uid);


--
-- Name: properties_analysis_rent properties_analysis_rent_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties_analysis_rent
    ADD CONSTRAINT properties_analysis_rent_pkey PRIMARY KEY (property_uid);


--
-- Name: properties_analysis_sell properties_analysis_sell_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties_analysis_sell
    ADD CONSTRAINT properties_analysis_sell_pkey PRIMARY KEY (property_uid);


--
-- Name: property_geo_data property_geo_data_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_geo_data
    ADD CONSTRAINT property_geo_data_pkey PRIMARY KEY (property_uid);


--
-- Name: property_images property_images_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_images
    ADD CONSTRAINT property_images_pkey PRIMARY KEY (property_uid, position);


--
-- Name: property_rankings property_rankings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_rankings
    ADD CONSTRAINT property_rankings_pkey PRIMARY KEY (property_uid);


--
-- Name: property_realtors property_realtors_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_realtors
    ADD CONSTRAINT property_realtors_pkey PRIMARY KEY (property_uid, realtor_uid);


--
-- Name: push_subscriptions push_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.push_subscriptions
    ADD CONSTRAINT push_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: ranking_models ranking_models_item_type_subtype_transaction_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ranking_models
    ADD CONSTRAINT ranking_models_item_type_subtype_transaction_key UNIQUE (item_type, subtype, transaction);


--
-- Name: ranking_models ranking_models_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ranking_models
    ADD CONSTRAINT ranking_models_pkey PRIMARY KEY (id);


--
-- Name: realtors realtors_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.realtors
    ADD CONSTRAINT realtors_pkey PRIMARY KEY (uid);


--
-- Name: realtors realtors_title_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.realtors
    ADD CONSTRAINT realtors_title_key UNIQUE (title);


--
-- Name: residential_specifics residential_specifics_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.residential_specifics
    ADD CONSTRAINT residential_specifics_pkey PRIMARY KEY (property_uid);


--
-- Name: scheduler_job_runs scheduler_job_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_job_runs
    ADD CONSTRAINT scheduler_job_runs_pkey PRIMARY KEY (run_uid);


--
-- Name: scheduler_job_scrape_sources scheduler_job_scrape_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_job_scrape_sources
    ADD CONSTRAINT scheduler_job_scrape_sources_pkey PRIMARY KEY (job_uid, source_order);


--
-- Name: scheduler_jobs scheduler_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_jobs
    ADD CONSTRAINT scheduler_jobs_pkey PRIMARY KEY (job_uid);


--
-- Name: scheduler_maintenance_chain_runs scheduler_maintenance_chain_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_maintenance_chain_runs
    ADD CONSTRAINT scheduler_maintenance_chain_runs_pkey PRIMARY KEY (chain_run_uid);


--
-- Name: scheduler_maintenance_chain_runs scheduler_maintenance_chain_runs_trigger_run_uid_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_maintenance_chain_runs
    ADD CONSTRAINT scheduler_maintenance_chain_runs_trigger_run_uid_key UNIQUE (trigger_run_uid);


--
-- Name: segment_metadata segment_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.segment_metadata
    ADD CONSTRAINT segment_metadata_pkey PRIMARY KEY (singleton);


--
-- Name: signup_whitelist signup_whitelist_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.signup_whitelist
    ADD CONSTRAINT signup_whitelist_pkey PRIMARY KEY (email);


--
-- Name: auth_providers uq_auth_provider_external; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_providers
    ADD CONSTRAINT uq_auth_provider_external UNIQUE (provider, provider_user_id);


--
-- Name: auth_providers uq_auth_provider_user; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_providers
    ADD CONSTRAINT uq_auth_provider_user UNIQUE (user_uid, provider);


--
-- Name: business_features uq_business_feature; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_features
    ADD CONSTRAINT uq_business_feature UNIQUE (business_uid, feature_key);


--
-- Name: user_businesses user_businesses_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_businesses
    ADD CONSTRAINT user_businesses_pkey PRIMARY KEY (user_uid, business_uid);


--
-- Name: user_credentials user_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_credentials
    ADD CONSTRAINT user_credentials_pkey PRIMARY KEY (uid);


--
-- Name: user_credits user_credits_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_credits
    ADD CONSTRAINT user_credits_pkey PRIMARY KEY (user_uid);


--
-- Name: user_feedback user_feedback_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_feedback
    ADD CONSTRAINT user_feedback_pkey PRIMARY KEY (uid);


--
-- Name: user_properties user_properties_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_properties
    ADD CONSTRAINT user_properties_pkey PRIMARY KEY (property_uid, user_uid);


--
-- Name: user_saved_properties user_saved_properties_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_saved_properties
    ADD CONSTRAINT user_saved_properties_pkey PRIMARY KEY (user_uid, property_uid);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (uid);


--
-- Name: users users_stripe_customer_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_stripe_customer_id_key UNIQUE (stripe_customer_id);


--
-- Name: valuation_adjustment_policies valuation_adjustment_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuation_adjustment_policies
    ADD CONSTRAINT valuation_adjustment_policies_pkey PRIMARY KEY (id);


--
-- Name: valuation_backtest_jobs valuation_backtest_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuation_backtest_jobs
    ADD CONSTRAINT valuation_backtest_jobs_pkey PRIMARY KEY (job_uid);


--
-- Name: valuations valuations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations
    ADD CONSTRAINT valuations_pkey PRIMARY KEY (property_uid);


--
-- Name: white_label_settings white_label_settings_icon_file_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.white_label_settings
    ADD CONSTRAINT white_label_settings_icon_file_key UNIQUE (icon_file);


--
-- Name: white_label_settings white_label_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.white_label_settings
    ADD CONSTRAINT white_label_settings_pkey PRIMARY KEY (business_uid);


--
-- Name: worker_artifact_revisions worker_artifact_revisions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.worker_artifact_revisions
    ADD CONSTRAINT worker_artifact_revisions_pkey PRIMARY KEY (artifact);


--
-- Name: idx_admin_action_records_resource; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_action_records_resource ON public.admin_action_records USING btree (resource_type, resource_uid, created_at DESC);


--
-- Name: idx_admin_active_user_snapshots_sampled_at_desc; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_active_user_snapshots_sampled_at_desc ON public.admin_active_user_snapshots USING btree (sampled_at DESC);


--
-- Name: idx_admin_audit_events_actor_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_audit_events_actor_created_at ON public.admin_audit_events USING btree (actor_user_uid, created_at DESC);


--
-- Name: idx_admin_audit_events_route_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_audit_events_route_created_at ON public.admin_audit_events USING btree (route, created_at DESC);


--
-- Name: idx_admin_audit_events_target_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_audit_events_target_created_at ON public.admin_audit_events USING btree (target_type, target_uid, created_at DESC);


--
-- Name: idx_admin_boundaries_geom; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_boundaries_geom ON public.admin_boundaries USING gist (geom);


--
-- Name: idx_admin_charge_waivers_owner; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_charge_waivers_owner ON public.admin_charge_waivers USING btree (owner_type, owner_uid, created_at DESC);


--
-- Name: idx_admin_charge_waivers_target; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_admin_charge_waivers_target ON public.admin_charge_waivers USING btree (target_record_uid) WHERE (target_record_uid IS NOT NULL);


--
-- Name: idx_area_market_premiums_lookup; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_area_market_premiums_lookup ON public.area_market_premiums USING btree (admin_name, item_type, subtype);


--
-- Name: idx_bsp_business_saved_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bsp_business_saved_uid ON public.business_saved_properties USING btree (business_uid, saved_at, property_uid);


--
-- Name: idx_bulk_subscription_change_jobs_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bulk_subscription_change_jobs_status ON public.bulk_subscription_change_jobs USING btree (status, created_at DESC);


--
-- Name: idx_bulk_subscription_change_targets_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_bulk_subscription_change_targets_status ON public.bulk_subscription_change_targets USING btree (job_uid, status);


--
-- Name: idx_business_contracts_active; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_business_contracts_active ON public.business_contracts USING btree (business_uid) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_business_contracts_business; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_business_contracts_business ON public.business_contracts USING btree (business_uid);


--
-- Name: idx_business_portfolio_monthly_history_month; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_business_portfolio_monthly_history_month ON public.business_portfolio_monthly_history USING btree (month_start DESC);


--
-- Name: idx_business_saved_props_editor; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_business_saved_props_editor ON public.business_saved_properties USING btree (last_edited_by);


--
-- Name: idx_business_saved_props_user; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_business_saved_props_user ON public.business_saved_properties USING btree (saved_by);


--
-- Name: idx_businesses_admin_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_businesses_admin_uid ON public.businesses USING btree (admin_uid);


--
-- Name: idx_businesses_geog; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_businesses_geog ON public.businesses USING gist (((geom)::public.geography)) WHERE (geom IS NOT NULL);


--
-- Name: idx_businesses_geom; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_businesses_geom ON public.businesses USING gist (geom) WHERE (geom IS NOT NULL);


--
-- Name: idx_businesses_lead_recipient_email; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_businesses_lead_recipient_email ON public.businesses USING btree (lead_recipient_email) WHERE (lead_recipient_email IS NOT NULL);


--
-- Name: idx_call_requests_called_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_call_requests_called_at ON public.call_requests USING btree (called_at);


--
-- Name: idx_call_requests_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_call_requests_created_at ON public.call_requests USING btree (created_at DESC);


--
-- Name: idx_contract_offers_pending; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_contract_offers_pending ON public.contract_offers USING btree (business_uid) WHERE ((status)::text = 'pending'::text);


--
-- Name: idx_credit_adjustments_user_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_credit_adjustments_user_created ON public.credit_adjustments USING btree (user_uid, created_at DESC);


--
-- Name: idx_deleted_properties_deleted_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_deleted_properties_deleted_at ON public.deleted_properties USING btree (deleted_at DESC);


--
-- Name: idx_deleted_properties_segment; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_deleted_properties_segment ON public.deleted_properties USING btree (item_type, transaction_type, subtype);


--
-- Name: idx_deleted_properties_segment_admin; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_deleted_properties_segment_admin ON public.deleted_properties USING btree (item_type, transaction_type, subtype, admin_name);


--
-- Name: idx_detailed_report_forwards_business; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_detailed_report_forwards_business ON public.detailed_report_forwards USING btree (recipient_business_uid, created_at DESC);


--
-- Name: idx_detailed_report_forwards_report; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_detailed_report_forwards_report ON public.detailed_report_forwards USING btree (report_uid, created_at DESC);


--
-- Name: idx_detailed_report_forwards_sender; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_detailed_report_forwards_sender ON public.detailed_report_forwards USING btree (sender_user_uid, created_at DESC);


--
-- Name: idx_detailed_report_forwards_token_hash; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_detailed_report_forwards_token_hash ON public.detailed_report_forwards USING btree (token_hash);


--
-- Name: idx_detailed_report_views_report_viewed_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_detailed_report_views_report_viewed_at ON public.detailed_report_views USING btree (report_uid, viewed_at DESC);


--
-- Name: idx_detailed_reports_property_created_at_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_detailed_reports_property_created_at_uid ON public.detailed_reports USING btree (property_uid, created_at, uid);


--
-- Name: idx_detailed_reports_user_property_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_detailed_reports_user_property_created_at ON public.detailed_reports USING btree (created_by_user_uid, property_uid, created_at DESC) WHERE (created_by_user_uid IS NOT NULL);


--
-- Name: idx_gov_sales_admin_name; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_gov_sales_admin_name ON public.gov_verified_sales USING btree (admin_name, item_type, subtype);


--
-- Name: idx_gov_sales_contract_date; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_gov_sales_contract_date ON public.gov_verified_sales USING btree (contract_date DESC);


--
-- Name: idx_gov_sales_municipality_type; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_gov_sales_municipality_type ON public.gov_verified_sales USING btree (municipality, item_type, subtype);


--
-- Name: idx_gov_sales_quality_tier; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_gov_sales_quality_tier ON public.gov_verified_sales USING btree (quality_tier);


--
-- Name: idx_gov_sales_row_hash; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_gov_sales_row_hash ON public.gov_verified_sales USING btree (row_hash);


--
-- Name: idx_gov_sales_zone_value; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_gov_sales_zone_value ON public.gov_verified_sales USING btree (zone_value);


--
-- Name: idx_individual_subscriptions_active; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_individual_subscriptions_active ON public.individual_subscriptions USING btree (user_uid) WHERE ((status)::text = 'active'::text);


--
-- Name: idx_individual_subscriptions_user; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_individual_subscriptions_user ON public.individual_subscriptions USING btree (user_uid);


--
-- Name: idx_invoice_tasks_owner; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_invoice_tasks_owner ON public.invoice_tasks USING btree (owner_type, owner_uid);


--
-- Name: idx_invoice_tasks_payment; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_invoice_tasks_payment ON public.invoice_tasks USING btree (payment_uid);


--
-- Name: idx_invoice_tasks_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_invoice_tasks_status ON public.invoice_tasks USING btree (status);


--
-- Name: idx_invoice_tasks_status_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_invoice_tasks_status_created ON public.invoice_tasks USING btree (status, created_at);


--
-- Name: idx_market_premium_policies_effective_from; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_market_premium_policies_effective_from ON public.market_premium_policies USING btree (effective_from DESC, created_at DESC);


--
-- Name: idx_marketability_factor_batches_claimable; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_marketability_factor_batches_claimable ON public.marketabillity_factor_batches USING btree (is_processed, processing_started_at, created_at);


--
-- Name: idx_marketabillity_factor_geom_covering; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_marketabillity_factor_geom_covering ON public.marketabillity_factor USING gist (geom) INCLUDE (marketabillity_factor);


--
-- Name: idx_notification_log_entity; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_notification_log_entity ON public.notification_log USING btree (entity_type, entity_uid, notification_type);


--
-- Name: idx_payment_methods_default; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_payment_methods_default ON public.payment_methods USING btree (owner_type, owner_uid) WHERE (is_default = true);


--
-- Name: idx_payment_methods_owner; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_payment_methods_owner ON public.payment_methods USING btree (owner_type, owner_uid);


--
-- Name: idx_payment_refunds_payment_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_payment_refunds_payment_created ON public.payment_refunds USING btree (payment_uid, created_at DESC);


--
-- Name: idx_payments_created; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_payments_created ON public.payments USING btree (created_at);


--
-- Name: idx_payments_owner; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_payments_owner ON public.payments USING btree (owner_type, owner_uid);


--
-- Name: idx_payments_payment_method; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_payments_payment_method ON public.payments USING btree (payment_method_uid, stripe_payment_method_id);


--
-- Name: idx_payments_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_payments_status ON public.payments USING btree (status);


--
-- Name: idx_properties_item_tx_subtype; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_properties_item_tx_subtype ON public.properties USING btree (item_type, transaction_type, subtype) INCLUDE (uid, asking_price);


--
-- Name: idx_property_geo_data_geog; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_property_geo_data_geog ON public.property_geo_data USING gist (((geom)::public.geography));
ALTER INDEX public.idx_property_geo_data_geog ALTER COLUMN 1 SET STATISTICS 1000;


--
-- Name: idx_property_geo_data_geom; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_property_geo_data_geom ON public.property_geo_data USING gist (geom);


--
-- Name: idx_property_rankings_computed_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_property_rankings_computed_at ON public.property_rankings USING btree (computed_at);


--
-- Name: idx_property_rankings_elo; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_property_rankings_elo ON public.property_rankings USING btree (elo DESC);


--
-- Name: idx_property_rankings_model_revision; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_property_rankings_model_revision ON public.property_rankings USING btree (model_revision);


--
-- Name: idx_property_rankings_model_trained_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_property_rankings_model_trained_at ON public.property_rankings USING btree (model_trained_at);


--
-- Name: idx_scheduler_job_runs_chain; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_scheduler_job_runs_chain ON public.scheduler_job_runs USING btree (chain_run_uid, chain_step) WHERE (chain_run_uid IS NOT NULL);


--
-- Name: idx_scheduler_job_runs_job_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_scheduler_job_runs_job_uid ON public.scheduler_job_runs USING btree (job_uid, started_at DESC);


--
-- Name: idx_scheduler_jobs_business_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_scheduler_jobs_business_uid ON public.scheduler_jobs USING btree (business_uid) WHERE (business_uid IS NOT NULL);


--
-- Name: idx_scheduler_jobs_expired_leases; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_scheduler_jobs_expired_leases ON public.scheduler_jobs USING btree (locked_until) WHERE ((status = 'running'::text) AND (locked_until IS NOT NULL));


--
-- Name: idx_scheduler_jobs_next_run; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_scheduler_jobs_next_run ON public.scheduler_jobs USING btree (next_run_at, priority DESC) WHERE (status = ANY (ARRAY['pending'::text, 'failed'::text]));


--
-- Name: idx_scheduler_jobs_one_active_chain_step; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_scheduler_jobs_one_active_chain_step ON public.scheduler_jobs USING btree (active_chain_run_uid) WHERE (active_chain_run_uid IS NOT NULL);


--
-- Name: idx_scheduler_maintenance_chain_open; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_scheduler_maintenance_chain_open ON public.scheduler_maintenance_chain_runs USING btree (created_at) WHERE (status <> 'finished'::text);


--
-- Name: idx_signup_whitelist_email_lower; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_signup_whitelist_email_lower ON public.signup_whitelist USING btree (lower(email));


--
-- Name: idx_user_businesses_business; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_user_businesses_business ON public.user_businesses USING btree (business_uid);


--
-- Name: idx_user_feedback_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_user_feedback_created_at ON public.user_feedback USING btree (created_at DESC);


--
-- Name: idx_user_feedback_type; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_user_feedback_type ON public.user_feedback USING btree (feedback_type);


--
-- Name: idx_user_feedback_user_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_user_feedback_user_uid ON public.user_feedback USING btree (user_uid);


--
-- Name: idx_user_properties_user_uid_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_user_properties_user_uid_created_at ON public.user_properties USING btree (user_uid, created_at DESC);


--
-- Name: idx_user_saved_properties_property_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_user_saved_properties_property_uid ON public.user_saved_properties USING btree (property_uid);


--
-- Name: idx_valuation_adjustment_policies_effective_from; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuation_adjustment_policies_effective_from ON public.valuation_adjustment_policies USING btree (effective_from DESC, created_at DESC);


--
-- Name: idx_valuation_adjustment_policies_lookup; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuation_adjustment_policies_lookup ON public.valuation_adjustment_policies USING btree (item_type, transaction_type, subtype, effective_from DESC, created_at DESC) WHERE (effective_from IS NOT NULL);


--
-- Name: idx_valuation_adjustment_policies_segment; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuation_adjustment_policies_segment ON public.valuation_adjustment_policies USING btree (item_type, transaction_type, subtype, effective_from DESC, created_at DESC);


--
-- Name: idx_valuation_adjustment_policies_segment_month; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX idx_valuation_adjustment_policies_segment_month ON public.valuation_adjustment_policies USING btree (COALESCE(item_type, ''::character varying), COALESCE(transaction_type, ''::character varying), COALESCE(subtype, ''::character varying), policy_month) WHERE (policy_month IS NOT NULL);


--
-- Name: idx_valuation_backtest_jobs_status; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuation_backtest_jobs_status ON public.valuation_backtest_jobs USING btree (status, created_at DESC);


--
-- Name: idx_valuations_effective_estimated_max; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuations_effective_estimated_max ON public.valuations USING btree (effective_estimated_max);


--
-- Name: idx_valuations_effective_estimated_min; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuations_effective_estimated_min ON public.valuations USING btree (effective_estimated_min);


--
-- Name: idx_valuations_history_archived_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuations_history_archived_at ON ONLY public.valuations_history USING btree (archived_at);


--
-- Name: idx_valuations_history_effective_max; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuations_history_effective_max ON ONLY public.valuations_history USING btree (effective_estimated_max);


--
-- Name: idx_valuations_history_property_uid; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_valuations_history_property_uid ON ONLY public.valuations_history USING btree (property_uid);


--
-- Name: idx_white_label_settings_enabled; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_white_label_settings_enabled ON public.white_label_settings USING btree (enabled);


--
-- Name: marketabillity_factor_geom_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX marketabillity_factor_geom_idx ON public.marketabillity_factor USING gist (geom);


--
-- Name: marketabillity_factor_identity_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX marketabillity_factor_identity_idx ON public.marketabillity_factor USING btree (area, main_street, COALESCE(geo_lat_start, (0)::double precision), COALESCE(geo_lon_start, (0)::double precision), COALESCE(geo_lat_end, (0)::double precision), COALESCE(geo_lon_end, (0)::double precision));


--
-- Name: push_subscriptions_endpoint_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX push_subscriptions_endpoint_idx ON public.push_subscriptions USING btree (endpoint);


--
-- Name: push_subscriptions_user_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX push_subscriptions_user_uid_idx ON public.push_subscriptions USING btree (user_uid);


--
-- Name: valuations_history_default_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_default_archived_at_idx ON public.valuations_history_default USING btree (archived_at);


--
-- Name: valuations_history_default_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_default_effective_estimated_max_idx ON public.valuations_history_default USING btree (effective_estimated_max);


--
-- Name: valuations_history_default_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_default_property_uid_idx ON public.valuations_history_default USING btree (property_uid);


--
-- Name: valuations_history_p20260201_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260201_archived_at_idx ON public.valuations_history_p20260201 USING btree (archived_at);


--
-- Name: valuations_history_p20260201_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260201_effective_estimated_max_idx ON public.valuations_history_p20260201 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260201_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260201_property_uid_idx ON public.valuations_history_p20260201 USING btree (property_uid);


--
-- Name: valuations_history_p20260301_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260301_archived_at_idx ON public.valuations_history_p20260301 USING btree (archived_at);


--
-- Name: valuations_history_p20260301_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260301_effective_estimated_max_idx ON public.valuations_history_p20260301 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260301_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260301_property_uid_idx ON public.valuations_history_p20260301 USING btree (property_uid);


--
-- Name: valuations_history_p20260401_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260401_archived_at_idx ON public.valuations_history_p20260401 USING btree (archived_at);


--
-- Name: valuations_history_p20260401_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260401_effective_estimated_max_idx ON public.valuations_history_p20260401 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260401_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260401_property_uid_idx ON public.valuations_history_p20260401 USING btree (property_uid);


--
-- Name: valuations_history_p20260501_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260501_archived_at_idx ON public.valuations_history_p20260501 USING btree (archived_at);


--
-- Name: valuations_history_p20260501_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260501_effective_estimated_max_idx ON public.valuations_history_p20260501 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260501_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260501_property_uid_idx ON public.valuations_history_p20260501 USING btree (property_uid);


--
-- Name: valuations_history_p20260601_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260601_archived_at_idx ON public.valuations_history_p20260601 USING btree (archived_at);


--
-- Name: valuations_history_p20260601_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260601_effective_estimated_max_idx ON public.valuations_history_p20260601 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260601_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260601_property_uid_idx ON public.valuations_history_p20260601 USING btree (property_uid);


--
-- Name: valuations_history_p20260701_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260701_archived_at_idx ON public.valuations_history_p20260701 USING btree (archived_at);


--
-- Name: valuations_history_p20260701_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260701_effective_estimated_max_idx ON public.valuations_history_p20260701 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260701_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260701_property_uid_idx ON public.valuations_history_p20260701 USING btree (property_uid);


--
-- Name: valuations_history_p20260801_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260801_archived_at_idx ON public.valuations_history_p20260801 USING btree (archived_at);


--
-- Name: valuations_history_p20260801_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260801_effective_estimated_max_idx ON public.valuations_history_p20260801 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260801_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260801_property_uid_idx ON public.valuations_history_p20260801 USING btree (property_uid);


--
-- Name: valuations_history_p20260901_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260901_archived_at_idx ON public.valuations_history_p20260901 USING btree (archived_at);


--
-- Name: valuations_history_p20260901_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260901_effective_estimated_max_idx ON public.valuations_history_p20260901 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20260901_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20260901_property_uid_idx ON public.valuations_history_p20260901 USING btree (property_uid);


--
-- Name: valuations_history_p20261001_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20261001_archived_at_idx ON public.valuations_history_p20261001 USING btree (archived_at);


--
-- Name: valuations_history_p20261001_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20261001_effective_estimated_max_idx ON public.valuations_history_p20261001 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20261001_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20261001_property_uid_idx ON public.valuations_history_p20261001 USING btree (property_uid);


--
-- Name: valuations_history_p20261101_archived_at_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20261101_archived_at_idx ON public.valuations_history_p20261101 USING btree (archived_at);


--
-- Name: valuations_history_p20261101_effective_estimated_max_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20261101_effective_estimated_max_idx ON public.valuations_history_p20261101 USING btree (effective_estimated_max);


--
-- Name: valuations_history_p20261101_property_uid_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX valuations_history_p20261101_property_uid_idx ON public.valuations_history_p20261101 USING btree (property_uid);


--
-- Name: valuations_history_default_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_default_archived_at_idx;


--
-- Name: valuations_history_default_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_default_effective_estimated_max_idx;


--
-- Name: valuations_history_default_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_default_property_uid_idx;


--
-- Name: valuations_history_p20260201_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260201_archived_at_idx;


--
-- Name: valuations_history_p20260201_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260201_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260201_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260201_property_uid_idx;


--
-- Name: valuations_history_p20260301_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260301_archived_at_idx;


--
-- Name: valuations_history_p20260301_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260301_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260301_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260301_property_uid_idx;


--
-- Name: valuations_history_p20260401_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260401_archived_at_idx;


--
-- Name: valuations_history_p20260401_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260401_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260401_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260401_property_uid_idx;


--
-- Name: valuations_history_p20260501_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260501_archived_at_idx;


--
-- Name: valuations_history_p20260501_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260501_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260501_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260501_property_uid_idx;


--
-- Name: valuations_history_p20260601_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260601_archived_at_idx;


--
-- Name: valuations_history_p20260601_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260601_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260601_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260601_property_uid_idx;


--
-- Name: valuations_history_p20260701_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260701_archived_at_idx;


--
-- Name: valuations_history_p20260701_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260701_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260701_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260701_property_uid_idx;


--
-- Name: valuations_history_p20260801_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260801_archived_at_idx;


--
-- Name: valuations_history_p20260801_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260801_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260801_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260801_property_uid_idx;


--
-- Name: valuations_history_p20260901_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20260901_archived_at_idx;


--
-- Name: valuations_history_p20260901_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20260901_effective_estimated_max_idx;


--
-- Name: valuations_history_p20260901_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20260901_property_uid_idx;


--
-- Name: valuations_history_p20261001_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20261001_archived_at_idx;


--
-- Name: valuations_history_p20261001_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20261001_effective_estimated_max_idx;


--
-- Name: valuations_history_p20261001_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20261001_property_uid_idx;


--
-- Name: valuations_history_p20261101_archived_at_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_archived_at ATTACH PARTITION public.valuations_history_p20261101_archived_at_idx;


--
-- Name: valuations_history_p20261101_effective_estimated_max_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_effective_max ATTACH PARTITION public.valuations_history_p20261101_effective_estimated_max_idx;


--
-- Name: valuations_history_p20261101_property_uid_idx; Type: INDEX ATTACH; Schema: public; Owner: postgres
--

ALTER INDEX public.idx_valuations_history_property_uid ATTACH PARTITION public.valuations_history_p20261101_property_uid_idx;


--
-- Name: marketabillity_factor marketabillity_geom_update; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER marketabillity_geom_update BEFORE INSERT OR UPDATE ON public.marketabillity_factor FOR EACH ROW EXECUTE FUNCTION public.update_marketabillity_geom();


--
-- Name: businesses trg_delete_billing_preferences_on_business_delete; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER trg_delete_billing_preferences_on_business_delete BEFORE DELETE ON public.businesses FOR EACH ROW EXECUTE FUNCTION public.cascade_delete_billing_preferences_for_business();


--
-- Name: users trg_delete_billing_preferences_on_user_delete; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER trg_delete_billing_preferences_on_user_delete BEFORE DELETE ON public.users FOR EACH ROW EXECUTE FUNCTION public.cascade_delete_billing_preferences_for_user();


--
-- Name: businesses trg_delete_payment_methods_on_business_delete; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER trg_delete_payment_methods_on_business_delete BEFORE DELETE ON public.businesses FOR EACH ROW EXECUTE FUNCTION public.cascade_delete_payment_methods_for_business();


--
-- Name: users trg_delete_payment_methods_on_user_delete; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER trg_delete_payment_methods_on_user_delete BEFORE DELETE ON public.users FOR EACH ROW EXECUTE FUNCTION public.cascade_delete_payment_methods_for_user();


--
-- Name: businesses trg_delete_payments_on_business_delete; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER trg_delete_payments_on_business_delete BEFORE DELETE ON public.businesses FOR EACH ROW EXECUTE FUNCTION public.cascade_delete_payments_for_business();


--
-- Name: users trg_delete_payments_on_user_delete; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER trg_delete_payments_on_user_delete BEFORE DELETE ON public.users FOR EACH ROW EXECUTE FUNCTION public.cascade_delete_payments_for_user();


--
-- Name: auth_providers auth_providers_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_providers
    ADD CONSTRAINT auth_providers_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: bulk_subscription_change_jobs bulk_subscription_change_jobs_from_plan_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bulk_subscription_change_jobs
    ADD CONSTRAINT bulk_subscription_change_jobs_from_plan_uid_fkey FOREIGN KEY (from_plan_uid) REFERENCES public.individual_plans(uid);


--
-- Name: bulk_subscription_change_jobs bulk_subscription_change_jobs_to_plan_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bulk_subscription_change_jobs
    ADD CONSTRAINT bulk_subscription_change_jobs_to_plan_uid_fkey FOREIGN KEY (to_plan_uid) REFERENCES public.individual_plans(uid);


--
-- Name: bulk_subscription_change_targets bulk_subscription_change_targets_job_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bulk_subscription_change_targets
    ADD CONSTRAINT bulk_subscription_change_targets_job_uid_fkey FOREIGN KEY (job_uid) REFERENCES public.bulk_subscription_change_jobs(uid) ON DELETE CASCADE;


--
-- Name: business_commission_settings business_commission_settings_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_commission_settings
    ADD CONSTRAINT business_commission_settings_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: business_commission_settings business_commission_settings_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_commission_settings
    ADD CONSTRAINT business_commission_settings_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(uid) ON DELETE SET NULL;


--
-- Name: business_contracts business_contracts_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_contracts
    ADD CONSTRAINT business_contracts_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: business_features business_features_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_features
    ADD CONSTRAINT business_features_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: business_portfolio_monthly_history business_portfolio_monthly_history_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_portfolio_monthly_history
    ADD CONSTRAINT business_portfolio_monthly_history_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: business_saved_properties business_saved_properties_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_saved_properties
    ADD CONSTRAINT business_saved_properties_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: business_saved_properties business_saved_properties_last_edited_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_saved_properties
    ADD CONSTRAINT business_saved_properties_last_edited_by_fkey FOREIGN KEY (last_edited_by) REFERENCES public.users(uid) ON DELETE SET NULL;


--
-- Name: business_saved_properties business_saved_properties_property_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_saved_properties
    ADD CONSTRAINT business_saved_properties_property_uid_fkey FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE;


--
-- Name: business_saved_properties business_saved_properties_saved_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.business_saved_properties
    ADD CONSTRAINT business_saved_properties_saved_by_fkey FOREIGN KEY (saved_by) REFERENCES public.users(uid) ON DELETE SET NULL;


--
-- Name: businesses businesses_admin_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.businesses
    ADD CONSTRAINT businesses_admin_uid_fkey FOREIGN KEY (admin_uid) REFERENCES public.users(uid) ON DELETE RESTRICT;


--
-- Name: contract_offers contract_offers_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.contract_offers
    ADD CONSTRAINT contract_offers_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: credit_adjustments credit_adjustments_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.credit_adjustments
    ADD CONSTRAINT credit_adjustments_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: detailed_report_forwards detailed_report_forwards_recipient_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_forwards
    ADD CONSTRAINT detailed_report_forwards_recipient_business_uid_fkey FOREIGN KEY (recipient_business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: detailed_report_forwards detailed_report_forwards_report_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_forwards
    ADD CONSTRAINT detailed_report_forwards_report_uid_fkey FOREIGN KEY (report_uid) REFERENCES public.detailed_reports(uid) ON DELETE CASCADE;


--
-- Name: detailed_report_forwards detailed_report_forwards_sender_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_forwards
    ADD CONSTRAINT detailed_report_forwards_sender_user_uid_fkey FOREIGN KEY (sender_user_uid) REFERENCES public.users(uid) ON DELETE RESTRICT;


--
-- Name: detailed_report_views detailed_report_views_report_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_views
    ADD CONSTRAINT detailed_report_views_report_uid_fkey FOREIGN KEY (report_uid) REFERENCES public.detailed_reports(uid) ON DELETE CASCADE;


--
-- Name: detailed_report_views detailed_report_views_viewer_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_report_views
    ADD CONSTRAINT detailed_report_views_viewer_user_uid_fkey FOREIGN KEY (viewer_user_uid) REFERENCES public.users(uid) ON DELETE SET NULL;


--
-- Name: detailed_reports fk_created_by_business; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_reports
    ADD CONSTRAINT fk_created_by_business FOREIGN KEY (created_by_business_uid) REFERENCES public.businesses(uid) ON DELETE SET NULL;


--
-- Name: detailed_reports fk_created_by_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_reports
    ADD CONSTRAINT fk_created_by_user FOREIGN KEY (created_by_user_uid) REFERENCES public.users(uid) ON DELETE SET NULL;


--
-- Name: base_attributes fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.base_attributes
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: building_details fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.building_details
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: residential_specifics fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.residential_specifics
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: commercial_specifics fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.commercial_specifics
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: landplot_specifics fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.landplot_specifics
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: property_realtors fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_realtors
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: property_geo_data fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_geo_data
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: property_images property_images_property_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_images
    ADD CONSTRAINT property_images_property_uid_fkey FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE;


--
-- Name: valuations fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuations
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimated_value_snapshot fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.estimated_value_snapshot
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: detailed_reports fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.detailed_reports
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: valuations_history fk_property; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE public.valuations_history
    ADD CONSTRAINT fk_property FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: properties_analysis_rent properties_analysis_rent_property_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties_analysis_rent
    ADD CONSTRAINT properties_analysis_rent_property_uid_fkey FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE;


--
-- Name: properties_analysis_sell properties_analysis_sell_property_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties_analysis_sell
    ADD CONSTRAINT properties_analysis_sell_property_uid_fkey FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE;


--
-- Name: properties_elo fk_propuid; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.properties_elo
    ADD CONSTRAINT fk_propuid FOREIGN KEY (property_uid) REFERENCES public.properties(uid);


--
-- Name: property_realtors fk_realtor; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_realtors
    ADD CONSTRAINT fk_realtor FOREIGN KEY (realtor_uid) REFERENCES public.realtors(uid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_properties fk_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_properties
    ADD CONSTRAINT fk_user FOREIGN KEY (user_uid) REFERENCES public.user_credentials(uid);


--
-- Name: individual_plan_features individual_plan_features_plan_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_plan_features
    ADD CONSTRAINT individual_plan_features_plan_uid_fkey FOREIGN KEY (plan_uid) REFERENCES public.individual_plans(uid) ON DELETE CASCADE;


--
-- Name: individual_subscriptions individual_subscriptions_pending_plan_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_subscriptions
    ADD CONSTRAINT individual_subscriptions_pending_plan_uid_fkey FOREIGN KEY (pending_plan_uid) REFERENCES public.individual_plans(uid) ON DELETE SET NULL;


--
-- Name: individual_subscriptions individual_subscriptions_plan_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_subscriptions
    ADD CONSTRAINT individual_subscriptions_plan_uid_fkey FOREIGN KEY (plan_uid) REFERENCES public.individual_plans(uid) ON DELETE RESTRICT;


--
-- Name: individual_subscriptions individual_subscriptions_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.individual_subscriptions
    ADD CONSTRAINT individual_subscriptions_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: invoice_tasks invoice_tasks_payment_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.invoice_tasks
    ADD CONSTRAINT invoice_tasks_payment_uid_fkey FOREIGN KEY (payment_uid) REFERENCES public.payments(uid) ON DELETE CASCADE;


--
-- Name: payment_refunds payment_refunds_payment_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.payment_refunds
    ADD CONSTRAINT payment_refunds_payment_uid_fkey FOREIGN KEY (payment_uid) REFERENCES public.payments(uid) ON DELETE CASCADE;


--
-- Name: property_rankings property_rankings_property_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.property_rankings
    ADD CONSTRAINT property_rankings_property_uid_fkey FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE;


--
-- Name: push_subscriptions push_subscriptions_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.push_subscriptions
    ADD CONSTRAINT push_subscriptions_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: scheduler_job_runs scheduler_job_runs_job_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_job_runs
    ADD CONSTRAINT scheduler_job_runs_job_uid_fkey FOREIGN KEY (job_uid) REFERENCES public.scheduler_jobs(job_uid) ON DELETE CASCADE;


--
-- Name: scheduler_job_scrape_sources scheduler_job_scrape_sources_job_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_job_scrape_sources
    ADD CONSTRAINT scheduler_job_scrape_sources_job_uid_fkey FOREIGN KEY (job_uid) REFERENCES public.scheduler_jobs(job_uid) ON DELETE CASCADE;


--
-- Name: scheduler_jobs scheduler_jobs_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.scheduler_jobs
    ADD CONSTRAINT scheduler_jobs_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: user_businesses user_businesses_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_businesses
    ADD CONSTRAINT user_businesses_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: user_businesses user_businesses_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_businesses
    ADD CONSTRAINT user_businesses_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_credits user_credits_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_credits
    ADD CONSTRAINT user_credits_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_feedback user_feedback_active_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_feedback
    ADD CONSTRAINT user_feedback_active_business_uid_fkey FOREIGN KEY (active_business_uid) REFERENCES public.businesses(uid) ON DELETE SET NULL;


--
-- Name: user_feedback user_feedback_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_feedback
    ADD CONSTRAINT user_feedback_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: user_saved_properties user_saved_properties_property_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_saved_properties
    ADD CONSTRAINT user_saved_properties_property_uid_fkey FOREIGN KEY (property_uid) REFERENCES public.properties(uid) ON DELETE CASCADE;


--
-- Name: user_saved_properties user_saved_properties_user_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_saved_properties
    ADD CONSTRAINT user_saved_properties_user_uid_fkey FOREIGN KEY (user_uid) REFERENCES public.users(uid) ON DELETE CASCADE;


--
-- Name: valuation_backtest_jobs valuation_backtest_jobs_job_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.valuation_backtest_jobs
    ADD CONSTRAINT valuation_backtest_jobs_job_uid_fkey FOREIGN KEY (job_uid) REFERENCES public.scheduler_jobs(job_uid) ON DELETE CASCADE;


--
-- Name: white_label_settings white_label_settings_business_uid_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.white_label_settings
    ADD CONSTRAINT white_label_settings_business_uid_fkey FOREIGN KEY (business_uid) REFERENCES public.businesses(uid) ON DELETE CASCADE;


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;
GRANT USAGE ON SCHEMA public TO mcp_readonly;


--
-- Name: TABLE admin_action_records; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.admin_action_records TO mcp_readonly;


--
-- Name: TABLE admin_active_user_snapshots; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.admin_active_user_snapshots TO mcp_readonly;


--
-- Name: TABLE admin_audit_events; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.admin_audit_events TO mcp_readonly;


--
-- Name: TABLE admin_boundaries; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.admin_boundaries TO mcp_readonly;


--
-- Name: TABLE admin_boundary_negatives; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.admin_boundary_negatives TO mcp_readonly;


--
-- Name: TABLE admin_charge_waivers; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.admin_charge_waivers TO mcp_readonly;


--
-- Name: TABLE area_market_premiums; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.area_market_premiums TO mcp_readonly;


--
-- Name: TABLE auth_providers; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.auth_providers TO mcp_readonly;


--
-- Name: TABLE base_attributes; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.base_attributes TO mcp_readonly;


--
-- Name: TABLE billing_preferences; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.billing_preferences TO mcp_readonly;


--
-- Name: TABLE building_details; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.building_details TO mcp_readonly;


--
-- Name: TABLE bulk_subscription_change_jobs; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.bulk_subscription_change_jobs TO mcp_readonly;


--
-- Name: TABLE bulk_subscription_change_targets; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.bulk_subscription_change_targets TO mcp_readonly;


--
-- Name: TABLE business_commission_settings; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.business_commission_settings TO mcp_readonly;


--
-- Name: TABLE business_contracts; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.business_contracts TO mcp_readonly;


--
-- Name: TABLE business_features; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.business_features TO mcp_readonly;


--
-- Name: TABLE business_portfolio_monthly_history; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.business_portfolio_monthly_history TO mcp_readonly;


--
-- Name: TABLE business_saved_properties; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.business_saved_properties TO mcp_readonly;


--
-- Name: TABLE businesses; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.businesses TO mcp_readonly;


--
-- Name: TABLE call_requests; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.call_requests TO mcp_readonly;


--
-- Name: TABLE commercial_specifics; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.commercial_specifics TO mcp_readonly;


--
-- Name: TABLE contract_offers; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.contract_offers TO mcp_readonly;


--
-- Name: TABLE credit_adjustments; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.credit_adjustments TO mcp_readonly;


--
-- Name: TABLE deleted_properties; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.deleted_properties TO mcp_readonly;


--
-- Name: TABLE detailed_report_forwards; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.detailed_report_forwards TO mcp_readonly;


--
-- Name: TABLE detailed_report_views; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.detailed_report_views TO mcp_readonly;


--
-- Name: TABLE detailed_reports; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.detailed_reports TO mcp_readonly;


--
-- Name: TABLE estimated_value_snapshot; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.estimated_value_snapshot TO mcp_readonly;


--
-- Name: TABLE estimation_data; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.estimation_data TO mcp_readonly;


--
-- Name: TABLE geography_columns; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.geography_columns TO mcp_readonly;


--
-- Name: TABLE geometry_columns; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.geometry_columns TO mcp_readonly;


--
-- Name: TABLE global_stats; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.global_stats TO mcp_readonly;


--
-- Name: TABLE gov_area_geocoding; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.gov_area_geocoding TO mcp_readonly;


--
-- Name: TABLE gov_verified_sales; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.gov_verified_sales TO mcp_readonly;


--
-- Name: TABLE individual_plan_features; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.individual_plan_features TO mcp_readonly;


--
-- Name: TABLE individual_plans; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.individual_plans TO mcp_readonly;


--
-- Name: TABLE individual_subscriptions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.individual_subscriptions TO mcp_readonly;


--
-- Name: TABLE invoice_tasks; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.invoice_tasks TO mcp_readonly;


--
-- Name: TABLE landplot_specifics; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.landplot_specifics TO mcp_readonly;


--
-- Name: TABLE market_premium_policies; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.market_premium_policies TO mcp_readonly;


--
-- Name: TABLE marketabillity_factor; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.marketabillity_factor TO mcp_readonly;


--
-- Name: TABLE marketabillity_factor_batches; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.marketabillity_factor_batches TO mcp_readonly;


--
-- Name: TABLE notification_log; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.notification_log TO mcp_readonly;


--
-- Name: TABLE part_config; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.part_config TO mcp_readonly;


--
-- Name: TABLE part_config_sub; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.part_config_sub TO mcp_readonly;


--
-- Name: TABLE payment_methods; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.payment_methods TO mcp_readonly;


--
-- Name: TABLE payment_refunds; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.payment_refunds TO mcp_readonly;


--
-- Name: TABLE payments; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.payments TO mcp_readonly;


--
-- Name: TABLE properties; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.properties TO mcp_readonly;


--
-- Name: TABLE properties_elo; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.properties_elo TO mcp_readonly;


--
-- Name: TABLE property_geo_data; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.property_geo_data TO mcp_readonly;


--
-- Name: TABLE property_rankings; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.property_rankings TO mcp_readonly;


--
-- Name: TABLE property_realtors; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.property_realtors TO mcp_readonly;


--
-- Name: TABLE push_subscriptions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.push_subscriptions TO mcp_readonly;


--
-- Name: TABLE ranking_models; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.ranking_models TO mcp_readonly;


--
-- Name: TABLE realtors; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.realtors TO mcp_readonly;


--
-- Name: TABLE residential_specifics; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.residential_specifics TO mcp_readonly;


--
-- Name: TABLE scheduler_job_runs; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.scheduler_job_runs TO mcp_readonly;


--
-- Name: TABLE scheduler_job_scrape_sources; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.scheduler_job_scrape_sources TO mcp_readonly;


--
-- Name: TABLE scheduler_jobs; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.scheduler_jobs TO mcp_readonly;


--
-- Name: TABLE scheduler_maintenance_chain_runs; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.scheduler_maintenance_chain_runs TO mcp_readonly;


--
-- Name: TABLE segment_metadata; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.segment_metadata TO mcp_readonly;


--
-- Name: TABLE signup_whitelist; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.signup_whitelist TO mcp_readonly;


--
-- Name: TABLE spatial_ref_sys; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.spatial_ref_sys TO mcp_readonly;


--
-- Name: TABLE table_privs; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.table_privs TO mcp_readonly;


--
-- Name: TABLE template_public_valuations_history; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.template_public_valuations_history TO mcp_readonly;


--
-- Name: TABLE user_businesses; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.user_businesses TO mcp_readonly;


--
-- Name: TABLE user_credentials; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.user_credentials TO mcp_readonly;


--
-- Name: TABLE user_credits; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.user_credits TO mcp_readonly;


--
-- Name: TABLE user_feedback; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.user_feedback TO mcp_readonly;


--
-- Name: TABLE user_properties; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.user_properties TO mcp_readonly;


--
-- Name: TABLE user_saved_properties; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.user_saved_properties TO mcp_readonly;


--
-- Name: TABLE users; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.users TO mcp_readonly;


--
-- Name: TABLE valuation_adjustment_policies; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuation_adjustment_policies TO mcp_readonly;


--
-- Name: TABLE valuation_backtest_jobs; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuation_backtest_jobs TO mcp_readonly;


--
-- Name: TABLE valuations; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations TO mcp_readonly;


--
-- Name: TABLE valuations_history; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history TO mcp_readonly;


--
-- Name: TABLE valuations_history_default; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_default TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260201; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260201 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260301; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260301 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260401; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260401 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260501; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260501 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260601; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260601 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260701; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260701 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260801; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260801 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20260901; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20260901 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20261001; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20261001 TO mcp_readonly;


--
-- Name: TABLE valuations_history_p20261101; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.valuations_history_p20261101 TO mcp_readonly;


--
-- Name: TABLE white_label_settings; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.white_label_settings TO mcp_readonly;


--
-- Name: TABLE worker_artifact_revisions; Type: ACL; Schema: public; Owner: postgres
--

GRANT SELECT ON TABLE public.worker_artifact_revisions TO mcp_readonly;


--
-- Name: DEFAULT PRIVILEGES FOR TABLES; Type: DEFAULT ACL; Schema: public; Owner: postgres
--

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT SELECT ON TABLES TO mcp_readonly;


--
-- PostgreSQL database dump complete
--

\unrestrict AlDwefuEOy4a3wh2nAqcitRFQeY7ptGeEreUwds6Mnef425q8iXVWSFj1QcRBcs
