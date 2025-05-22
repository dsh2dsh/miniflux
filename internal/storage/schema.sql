-- -*- mode: sql; sql-product: postgres; -*-

CREATE TABLE schema_version (
    version text NOT NULL
);
INSERT INTO schema_version (version) VALUES('112');

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
    external_font_hosts text DEFAULT '' NOT NULL
);

CREATE UNIQUE INDEX ON users (google_id) WHERE (google_id <> '');
CREATE UNIQUE INDEX ON users (openid_connect_id) WHERE (openid_connect_id <> '');

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
    blocklist_rules text DEFAULT '' NOT NULL,
    keeplist_rules text DEFAULT '' NOT NULL,
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
    document_vectors tsvector,
    changed_at timestamp with time zone NOT NULL,
    share_code text DEFAULT '' NOT NULL,
    reading_time integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tags text[] DEFAULT '{}',
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

CREATE TABLE enclosures (
    id bigserial NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id bigint NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    url text NOT NULL,
    size bigint DEFAULT 0,
    mime_type text DEFAULT '',
    media_progression integer DEFAULT 0
);

CREATE INDEX ON enclosures (entry_id);
CREATE UNIQUE INDEX ON enclosures (user_id, entry_id, md5(url));

CREATE TABLE integrations (
    user_id integer NOT NULL PRIMARY KEY,
    pinboard_enabled boolean DEFAULT false,
    pinboard_token text DEFAULT '',
    pinboard_tags text DEFAULT 'miniflux',
    pinboard_mark_as_unread boolean DEFAULT false,
    instapaper_enabled boolean DEFAULT false,
    instapaper_username text DEFAULT '',
    instapaper_password text DEFAULT '',
    fever_enabled boolean DEFAULT false,
    fever_username text DEFAULT '',
    fever_token text DEFAULT '',
    wallabag_enabled boolean DEFAULT false,
    wallabag_url text DEFAULT '',
    wallabag_client_id text DEFAULT '',
    wallabag_client_secret text DEFAULT '',
    wallabag_username text DEFAULT '',
    wallabag_password text DEFAULT '',
    nunux_keeper_enabled boolean DEFAULT false,
    nunux_keeper_url text DEFAULT '',
    nunux_keeper_api_key text DEFAULT '',
    pocket_enabled boolean DEFAULT false,
    pocket_access_token text DEFAULT '',
    pocket_consumer_key text DEFAULT '',
    telegram_bot_enabled boolean DEFAULT false,
    telegram_bot_token text DEFAULT '',
    telegram_bot_chat_id text DEFAULT '',
    googlereader_enabled boolean DEFAULT false,
    googlereader_username text DEFAULT '',
    googlereader_password text DEFAULT '',
    espial_enabled boolean DEFAULT false,
    espial_url text DEFAULT '',
    espial_api_key text DEFAULT '',
    espial_tags text DEFAULT 'miniflux',
    linkding_enabled boolean DEFAULT false,
    linkding_url text DEFAULT '',
    linkding_api_key text DEFAULT '',
    wallabag_only_url boolean DEFAULT false,
    matrix_bot_enabled boolean DEFAULT false,
    matrix_bot_user text DEFAULT '',
    matrix_bot_password text DEFAULT '',
    matrix_bot_url text DEFAULT '',
    matrix_bot_chat_id text DEFAULT '',
    linkding_tags text DEFAULT '',
    linkding_mark_as_unread boolean DEFAULT false,
    notion_enabled boolean DEFAULT false,
    notion_token text DEFAULT '',
    notion_page_id text DEFAULT '',
    readwise_enabled boolean DEFAULT false,
    readwise_api_key text DEFAULT '',
    apprise_enabled boolean DEFAULT false,
    apprise_url text DEFAULT '',
    apprise_services_url text DEFAULT '',
    shiori_enabled boolean DEFAULT false,
    shiori_url text DEFAULT '',
    shiori_username text DEFAULT '',
    shiori_password text DEFAULT '',
    shaarli_enabled boolean DEFAULT false,
    shaarli_url text DEFAULT '',
    shaarli_api_secret text DEFAULT '',
    webhook_enabled boolean DEFAULT false,
    webhook_url text DEFAULT '',
    webhook_secret text DEFAULT '',
    telegram_bot_topic_id integer,
    telegram_bot_disable_web_page_preview boolean DEFAULT false,
    telegram_bot_disable_notification boolean DEFAULT false,
    telegram_bot_disable_buttons boolean DEFAULT false,
    rssbridge_enabled boolean DEFAULT false,
    rssbridge_url text DEFAULT '',
    omnivore_enabled boolean DEFAULT false,
    omnivore_api_key text DEFAULT '',
    omnivore_url text DEFAULT '',
    linkace_enabled boolean DEFAULT false,
    linkace_url text DEFAULT '',
    linkace_api_key text DEFAULT '',
    linkace_tags text DEFAULT '',
    linkace_is_private boolean DEFAULT true,
    linkace_check_disabled boolean DEFAULT true,
    linkwarden_enabled boolean DEFAULT false,
    linkwarden_url text DEFAULT '',
    linkwarden_api_key text DEFAULT '',
    readeck_enabled boolean DEFAULT false,
    readeck_only_url boolean DEFAULT false,
    readeck_url text DEFAULT '',
    readeck_api_key text DEFAULT '',
    readeck_labels text DEFAULT '',
    raindrop_enabled boolean DEFAULT false,
    raindrop_token text DEFAULT '',
    raindrop_collection_id text DEFAULT '',
    raindrop_tags text DEFAULT '',
    betula_url text DEFAULT '',
    betula_token text DEFAULT '',
    betula_enabled boolean DEFAULT false,
    ntfy_enabled boolean DEFAULT false,
    ntfy_url text DEFAULT '',
    ntfy_topic text DEFAULT '',
    ntfy_api_token text DEFAULT '',
    ntfy_username text DEFAULT '',
    ntfy_password text DEFAULT '',
    ntfy_icon_url text DEFAULT '',
    cubox_enabled boolean DEFAULT false,
    cubox_api_link text DEFAULT '',
    discord_enabled boolean DEFAULT false,
    discord_webhook_link text DEFAULT '',
    ntfy_internal_links boolean DEFAULT false,
    slack_enabled boolean DEFAULT false,
    slack_webhook_link text DEFAULT '',
    pushover_enabled boolean DEFAULT false,
    pushover_user text DEFAULT '',
    pushover_token text DEFAULT '',
    pushover_device text DEFAULT '',
    pushover_prefix text DEFAULT '',
    extra jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE sessions (
    id text NOT NULL PRIMARY KEY,
    data jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE user_sessions (
    id serial NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token text NOT NULL UNIQUE,
    created_at timestamp with time zone DEFAULT now(),
    user_agent text,
    ip inet,
    UNIQUE (user_id, token)
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
