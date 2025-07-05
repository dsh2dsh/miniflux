-- -*- mode: sql; sql-product: postgres; -*-

CREATE TABLE schema_version (
    version text NOT NULL
);
INSERT INTO schema_version (version) VALUES('120');

CREATE TABLE acme_cache (
    key character varying(400) NOT NULL PRIMARY KEY,
    data bytea NOT NULL,
    updated_at timestamp with time zone NOT NULL
);

CREATE TYPE entry_sorting_direction AS ENUM (
    'asc',
    'desc'
);

CREATE TYPE entry_sorting_order AS ENUM (
    'published_at',
    'created_at'
);

CREATE TYPE webapp_display_mode AS ENUM (
    'fullscreen',
    'standalone',
    'minimal-ui',
    'browser'
);

CREATE TABLE users (
    id serial NOT NULL PRIMARY KEY,
    username text NOT NULL UNIQUE,
    password text,
    is_admin boolean DEFAULT false,
    language text DEFAULT 'en_US',
    timezone text DEFAULT 'UTC',
    theme text DEFAULT 'light_serif',
    last_login_at timestamp with time zone,
    entry_direction entry_sorting_direction DEFAULT 'asc',
    keyboard_shortcuts boolean DEFAULT true,
    entries_per_page integer DEFAULT 100,
    show_reading_time boolean DEFAULT true,
    entry_swipe boolean DEFAULT true,
    stylesheet text DEFAULT '' NOT NULL,
    google_id text DEFAULT '' NOT NULL,
    openid_connect_id text DEFAULT '' NOT NULL,
    display_mode webapp_display_mode DEFAULT 'standalone',
    entry_order entry_sorting_order DEFAULT 'published_at',
    default_reading_speed integer DEFAULT 265,
    cjk_reading_speed integer DEFAULT 500,
    default_home_page text DEFAULT 'unread',
    categories_sorting_order text DEFAULT 'unread_count' NOT NULL,
    gesture_nav text DEFAULT 'tap',
    mark_read_on_view boolean DEFAULT true,
    media_playback_rate numeric DEFAULT 1,
    block_filter_entry_rules text DEFAULT '' NOT NULL,
    keep_filter_entry_rules text DEFAULT '' NOT NULL,
    mark_read_on_media_player_completion boolean DEFAULT false,
    custom_js text DEFAULT '' NOT NULL,
    external_font_hosts text DEFAULT '' NOT NULL,
    extra jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE UNIQUE INDEX ON users (google_id) WHERE (google_id <> '');
CREATE UNIQUE INDEX ON users (openid_connect_id) WHERE (openid_connect_id <> '');
CREATE INDEX ON users ((extra->'integration'->>'fever_token'));

CREATE TABLE api_keys (
    id serial NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token text NOT NULL UNIQUE,
    description text NOT NULL,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    UNIQUE (user_id, description)
);

CREATE TABLE categories (
    id serial NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title text NOT NULL,
    hide_globally boolean DEFAULT false NOT NULL,
    UNIQUE (user_id, title)
);

CREATE TABLE feeds (
    id bigserial NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id integer NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    title text NOT NULL,
    feed_url text NOT NULL,
    site_url text NOT NULL,
    checked_at timestamp with time zone DEFAULT now(),
    etag_header text DEFAULT '',
    last_modified_header text DEFAULT '',
    parsing_error_msg text DEFAULT '',
    parsing_error_count integer DEFAULT 0,
    scraper_rules text DEFAULT '',
    rewrite_rules text DEFAULT '',
    crawler boolean DEFAULT false,
    username text DEFAULT '',
    password text DEFAULT '',
    user_agent text DEFAULT '',
    disabled boolean DEFAULT false,
    next_check_at timestamp with time zone DEFAULT now(),
    ignore_http_cache boolean DEFAULT false,
    fetch_via_proxy boolean DEFAULT false,
    allow_self_signed_certificates boolean DEFAULT false NOT NULL,
    cookie text DEFAULT '',
    hide_globally boolean DEFAULT false NOT NULL,
    url_rewrite_rules text DEFAULT '' NOT NULL,
    no_media_player boolean DEFAULT false,
    apprise_service_urls text DEFAULT '',
    disable_http2 boolean DEFAULT false,
    description text DEFAULT '',
    ntfy_enabled boolean DEFAULT false,
    ntfy_priority integer DEFAULT 3,
    webhook_url text DEFAULT '',
    pushover_enabled boolean DEFAULT false,
    pushover_priority integer DEFAULT 0,
    ntfy_topic text DEFAULT '',
    proxy_url text DEFAULT '',
    extra jsonb NOT NULL DEFAULT '{}'::jsonb,
    runtime jsonb NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (user_id, feed_url)
);

CREATE INDEX ON feeds (id, hide_globally);
CREATE INDEX ON feeds (user_id, category_id);
CREATE INDEX ON feeds (next_check_at);
CREATE INDEX ON feeds (user_id, parsing_error_count);

CREATE TABLE icons (
    id bigserial NOT NULL PRIMARY KEY,
    hash text NOT NULL UNIQUE,
    mime_type text NOT NULL,
    content bytea NOT NULL,
    external_id text DEFAULT ''
);

CREATE UNIQUE INDEX ON icons (external_id) WHERE (external_id <> '');

CREATE TABLE feed_icons (
    feed_id bigint NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    icon_id bigint NOT NULL REFERENCES icons(id) ON DELETE CASCADE,
    PRIMARY KEY (feed_id, icon_id)
);

CREATE TYPE entry_status AS ENUM (
    'unread',
    'read',
    'removed'
);

CREATE TABLE entries (
    id bigserial NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feed_id bigint NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    hash text NOT NULL,
    published_at timestamp with time zone NOT NULL,
    title text NOT NULL,
    url text NOT NULL,
    author text,
    content text,
    status entry_status DEFAULT 'unread',
    starred boolean DEFAULT false,
    comments_url text DEFAULT '',
    document_vectors tsvector GENERATED ALWAYS AS (
      setweight(to_tsvector('simple', left(coalesce(title,   ''), 500000)), 'A') ||
      setweight(to_tsvector('simple', left(coalesce(content, ''), 500000)), 'B')
    ) STORED,
    changed_at timestamp with time zone NOT NULL,
    share_code text DEFAULT '' NOT NULL,
    reading_time integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tags text[] DEFAULT '{}',
    extra jsonb NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (feed_id, hash)
);

CREATE INDEX ON entries USING gin (document_vectors);
CREATE INDEX ON entries (feed_id, status, hash);
CREATE INDEX ON entries (feed_id);
CREATE INDEX ON entries (id, user_id, status);
CREATE UNIQUE INDEX ON entries (share_code) WHERE (share_code <> '');
CREATE INDEX ON entries (user_id, feed_id);
CREATE INDEX ON entries (user_id, status, starred);
CREATE INDEX ON entries (user_id, status, changed_at);
CREATE INDEX ON entries (user_id, status, changed_at, published_at);
CREATE INDEX ON entries (user_id, status, created_at);
CREATE INDEX ON entries (user_id, status, feed_id);
CREATE INDEX ON entries (user_id, status);
CREATE INDEX ON entries (user_id, status, published_at);

CREATE TABLE sessions (
    id text NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE TABLE webauthn_credentials (
    handle bytea NOT NULL PRIMARY KEY,
    cred_id bytea NOT NULL UNIQUE,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key bytea NOT NULL,
    attestation_type character varying(255) NOT NULL,
    aaguid bytea,
    sign_count bigint,
    clone_warning boolean,
    name text,
    added_on timestamp with time zone DEFAULT now(),
    last_seen_on timestamp with time zone DEFAULT now()
);
