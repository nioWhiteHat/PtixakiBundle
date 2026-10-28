BEGIN;

CREATE TABLE IF NOT EXISTS public.property_images (
    property_uid character varying(255) NOT NULL,
    position integer DEFAULT 0 NOT NULL,
    image_url text,
    image_data bytea,
    mime_type character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT property_images_pkey PRIMARY KEY (property_uid, position),
    CONSTRAINT property_images_one_source_check CHECK (
        ((image_url IS NOT NULL)::integer + (image_data IS NOT NULL)::integer) = 1
    ),
    CONSTRAINT property_images_property_uid_fkey
        FOREIGN KEY (property_uid)
        REFERENCES public.properties(uid)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_properties_user_uid_created_at
    ON public.user_properties (user_uid, created_at DESC);

COMMIT;
