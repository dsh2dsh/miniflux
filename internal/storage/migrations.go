package storage

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"miniflux.app/v2/internal/crypto"
)

type migration struct {
	sql    string
	txFunc func(ctx context.Context, tx pgx.Tx) error
}

func sqlMigration(s string) migration { return migration{sql: s} }

func txMigration(fn func(ctx context.Context, tx pgx.Tx) error) migration {
	return migration{txFunc: fn}
}

func (self *migration) Do(ctx context.Context, tx pgx.Tx) error {
	if fn := self.txFunc; fn != nil {
		if err := fn(ctx, tx); err != nil {
			return fmt.Errorf("migrate by fn: %w", err)
		}
		return nil
	}

	if _, err := tx.Exec(ctx, self.sql); err != nil {
		return fmt.Errorf("migrate by SQL: %w", err)
	}
	return nil
}

var schemaVersion = len(migrations)

//go:embed schema.sql
var fullSchema string

// Order is important. Add new migrations at the end of the list.
//
//nolint:wrapcheck // Migrate() wraps errors
var migrations = []migration{
	sqlMigration(fullSchema),

	sqlMigration(`
CREATE EXTENSION IF NOT EXISTS hstore;
ALTER TABLE users ADD COLUMN extra hstore;
CREATE INDEX users_extra_idx ON users using gin(extra);`),

	sqlMigration(`
CREATE TABLE tokens (
	id text not null,
	value text not null,
	created_at timestamp with time zone not null default now(),
	primary key(id, value)
);`),

	sqlMigration(`
CREATE TYPE entry_sorting_direction AS enum('asc', 'desc');
ALTER TABLE users
 ADD COLUMN entry_direction entry_sorting_direction default 'asc';`),

	sqlMigration(`
CREATE TABLE integrations (
	user_id int not null,
	pinboard_enabled bool default 'f',
	pinboard_token text default '',
	pinboard_tags text default 'miniflux',
	pinboard_mark_as_unread bool default 'f',
	instapaper_enabled bool default 'f',
	instapaper_username text default '',
	instapaper_password text default '',
	fever_enabled bool default 'f',
	fever_username text default '',
	fever_password text default '',
	fever_token text default '',
	primary key(user_id)
);`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN scraper_rules text default ''`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN rewrite_rules text default ''`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN crawler boolean default 'f'`),

	sqlMigration(`ALTER TABLE sessions rename to user_sessions`),

	sqlMigration(`
DROP TABLE tokens;
CREATE TABLE sessions (
	id text not null,
	data jsonb not null,
	created_at timestamp with time zone not null default now(),
	primary key(id)
);`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN wallabag_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN wallabag_url text default '';
ALTER TABLE integrations ADD COLUMN wallabag_client_id text default '';
ALTER TABLE integrations ADD COLUMN wallabag_client_secret text default '';
ALTER TABLE integrations ADD COLUMN wallabag_username text default '';
ALTER TABLE integrations ADD COLUMN wallabag_password text default '';`),

	sqlMigration(`ALTER TABLE entries ADD COLUMN starred bool default 'f'`),

	sqlMigration(`
CREATE INDEX entries_user_status_idx ON entries(user_id, status);
CREATE INDEX feeds_user_category_idx ON feeds(user_id, category_id);`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN nunux_keeper_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN nunux_keeper_url text default '';
ALTER TABLE integrations ADD COLUMN nunux_keeper_api_key text default '';`),

	sqlMigration(`ALTER TABLE enclosures ALTER COLUMN size SET DATA TYPE bigint`),

	sqlMigration(`ALTER TABLE entries ADD COLUMN comments_url text default ''`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN pocket_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN pocket_access_token text default '';
ALTER TABLE integrations ADD COLUMN pocket_consumer_key text default '';`),

	sqlMigration(`
ALTER TABLE user_sessions ALTER COLUMN ip SET DATA TYPE inet using ip::inet;`),

	sqlMigration(`
ALTER TABLE feeds ADD COLUMN username text default '';
ALTER TABLE feeds ADD COLUMN password text default '';`),

	sqlMigration(`
ALTER TABLE entries ADD COLUMN document_vectors tsvector;
UPDATE entries SET document_vectors = to_tsvector(substring(title || ' ' || coalesce(content, '') for 1000000));
CREATE INDEX document_vectors_idx ON entries USING gin(document_vectors);`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN user_agent text default ''`),

	sqlMigration(`
UPDATE entries
SET
	document_vectors = setweight(to_tsvector(substring(coalesce(title, '') for 1000000)), 'A') || setweight(to_tsvector(substring(coalesce(content, '') for 1000000)), 'B')`),

	sqlMigration(
		`ALTER TABLE users ADD COLUMN keyboard_shortcuts boolean default 't'`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN disabled boolean default 'f';`),

	sqlMigration(`
ALTER TABLE users ALTER COLUMN theme SET DEFAULT 'light_serif';
UPDATE users SET theme='light_serif' WHERE theme='default';
UPDATE users SET theme='light_sans_serif' WHERE theme='sansserif';
UPDATE users SET theme='dark_serif' WHERE theme='black';`),

	sqlMigration(`
ALTER TABLE entries ADD COLUMN changed_at timestamp with time zone;
UPDATE entries SET changed_at = published_at;
ALTER TABLE entries ALTER COLUMN changed_at SET not null;`),

	sqlMigration(`
CREATE TABLE api_keys (
	id serial not null,
	user_id int not null references users(id) on delete cascade,
	token text not null unique,
	description text not null,
	last_used_at timestamp with time zone,
	created_at timestamp with time zone default now(),
	primary key(id),
	unique (user_id, description)
);`),

	sqlMigration(`
ALTER TABLE entries ADD COLUMN share_code text not null default '';
CREATE UNIQUE INDEX entries_share_code_idx
  ON entries USING btree(share_code) WHERE share_code <> '';`),

	sqlMigration(`
CREATE INDEX enclosures_user_entry_url_idx
  ON enclosures(user_id, entry_id, md5(url))`),

	sqlMigration(`
ALTER TABLE feeds ADD COLUMN next_check_at timestamp with time zone default now();
CREATE INDEX entries_user_feed_idx ON entries (user_id, feed_id);`),

	sqlMigration(
		`ALTER TABLE feeds ADD COLUMN ignore_http_cache bool default false`),

	sqlMigration(`ALTER TABLE users ADD COLUMN entries_per_page int default 100`),

	sqlMigration(
		`ALTER TABLE users ADD COLUMN show_reading_time boolean default 't'`),

	sqlMigration(`
CREATE INDEX entries_id_user_status_idx
  ON entries USING btree (id, user_id, status)`),

	sqlMigration(
		`ALTER TABLE feeds ADD COLUMN fetch_via_proxy bool default false`),

	sqlMigration(`
CREATE INDEX entries_feed_id_status_hash_idx
  ON entries USING btree (feed_id, status, hash)`),

	sqlMigration(`
CREATE INDEX entries_user_id_status_starred_idx
  ON entries (user_id, status, starred)`),

	sqlMigration(`ALTER TABLE users ADD COLUMN entry_swipe boolean default 't'`),

	sqlMigration(`ALTER TABLE integrations DROP COLUMN fever_password`),

	sqlMigration(`
ALTER TABLE feeds
	ADD COLUMN blocklist_rules text not null default '',
	ADD COLUMN keeplist_rules text not null default ''`),

	sqlMigration(
		`ALTER TABLE entries ADD COLUMN reading_time int not null default 0`),

	sqlMigration(`
ALTER TABLE entries
  ADD COLUMN created_at timestamp with time zone not null default now();
UPDATE entries SET created_at = published_at;`),

	txMigration(func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
ALTER TABLE users
	ADD column stylesheet text not null default '',
	ADD column google_id text not null default '',
	ADD column openid_connect_id text not null default ''`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
DECLARE my_cursor CURSOR FOR
SELECT
	id,
	COALESCE(extra->'custom_css', '') as custom_css,
	COALESCE(extra->'google_id', '') as google_id,
	COALESCE(extra->'oidc_id', '') as oidc_id
FROM users FOR UPDATE`)
		if err != nil {
			return err
		}
		defer func() { _, _ = tx.Exec(ctx, "CLOSE my_cursor") }()

		for {
			var (
				userID           int64
				customStylesheet string
				googleID         string
				oidcID           string
			)

			err := tx.QueryRow(ctx, `FETCH NEXT FROM my_cursor`).
				Scan(&userID, &customStylesheet, &googleID, &oidcID)
			if errors.Is(err, pgx.ErrNoRows) {
				break
			} else if err != nil {
				return err
			}

			_, err = tx.Exec(ctx, `
UPDATE users
SET
	stylesheet=$2,
	google_id=$3,
	openid_connect_id=$4
WHERE id=$1`, userID, customStylesheet, googleID, oidcID)
			if err != nil {
				return err
			}
		}
		return nil
	}),

	sqlMigration(`
ALTER TABLE users DROP COLUMN extra;
CREATE UNIQUE INDEX users_google_id_idx
  ON users(google_id) WHERE google_id <> '';
CREATE UNIQUE INDEX users_openid_connect_id_idx
  ON users(openid_connect_id) WHERE openid_connect_id <> '';`),

	sqlMigration(`
CREATE INDEX entries_feed_url_idx ON entries(feed_id, url);
CREATE INDEX entries_user_status_feed_idx ON entries(user_id, status, feed_id);
CREATE INDEX entries_user_status_changed_idx
  ON entries(user_id, status, changed_at);`),

	sqlMigration(`
CREATE TABLE acme_cache (
	key varchar(400) not null primary key,
	data bytea not null,
	updated_at timestamptz not null
);`),

	sqlMigration(`
ALTER TABLE feeds
  ADD COLUMN allow_self_signed_certificates boolean not null default false`),

	sqlMigration(`
CREATE TYPE webapp_display_mode
  AS enum('fullscreen', 'standalone', 'minimal-ui', 'browser');
ALTER TABLE users
  ADD COLUMN display_mode webapp_display_mode default 'standalone';`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN cookie text default ''`),

	sqlMigration(`
ALTER TABLE categories
  ADD COLUMN hide_globally boolean not null default false`),

	sqlMigration(`
ALTER TABLE feeds
  ADD COLUMN hide_globally boolean not null default false`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN telegram_bot_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN telegram_bot_token text default '';
ALTER TABLE integrations ADD COLUMN telegram_bot_chat_id text default '';`),

	sqlMigration(`
CREATE TYPE entry_sorting_order AS enum('published_at', 'created_at');
ALTER TABLE users
  ADD COLUMN entry_order entry_sorting_order default 'published_at';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN googlereader_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN googlereader_username text default '';
ALTER TABLE integrations ADD COLUMN googlereader_password text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN espial_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN espial_url text default '';
ALTER TABLE integrations ADD COLUMN espial_api_key text default '';
ALTER TABLE integrations ADD COLUMN espial_tags text default 'miniflux';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN linkding_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN linkding_url text default '';
ALTER TABLE integrations ADD COLUMN linkding_api_key text default '';`),

	sqlMigration(`
ALTER TABLE feeds ADD COLUMN url_rewrite_rules text not null default ''`),

	sqlMigration(`
ALTER TABLE users ADD COLUMN default_reading_speed int default 265;
ALTER TABLE users ADD COLUMN cjk_reading_speed int default 500;`),

	sqlMigration(`
ALTER TABLE users ADD COLUMN default_home_page text default 'unread';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN wallabag_only_url bool default 'f';`),

	sqlMigration(`
ALTER TABLE users
  ADD COLUMN categories_sorting_order text not null default 'unread_count';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN matrix_bot_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN matrix_bot_user text default '';
ALTER TABLE integrations ADD COLUMN matrix_bot_password text default '';
ALTER TABLE integrations ADD COLUMN matrix_bot_url text default '';
ALTER TABLE integrations ADD COLUMN matrix_bot_chat_id text default '';`),

	sqlMigration(`ALTER TABLE users ADD COLUMN double_tap boolean default 't'`),

	sqlMigration(`ALTER TABLE entries ADD COLUMN tags text[] default '{}';`),

	sqlMigration(`
ALTER TABLE users RENAME double_tap TO gesture_nav;
ALTER TABLE users
  ALTER COLUMN gesture_nav
  SET DATA TYPE text using case when gesture_nav = true then 'tap' when gesture_nav = false then 'none' end;
ALTER TABLE users ALTER COLUMN gesture_nav SET default 'tap';`),

	sqlMigration(
		`ALTER TABLE integrations ADD COLUMN linkding_tags text default '';`),

	sqlMigration(`
ALTER TABLE feeds ADD COLUMN no_media_player boolean default 'f';
ALTER TABLE enclosures ADD COLUMN media_progression int default 0;`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN linkding_mark_as_unread bool default 'f';`),

	txMigration(func(ctx context.Context, tx pgx.Tx) error {
		// Delete duplicated rows
		_, err := tx.Exec(ctx, `
DELETE FROM enclosures a USING enclosures b
WHERE a.id < b.id
	AND a.user_id = b.user_id
	AND a.entry_id = b.entry_id
	AND a.url = b.url;`)
		if err != nil {
			return err
		}

		// Remove previous index
		_, err = tx.Exec(ctx, `DROP INDEX enclosures_user_entry_url_idx`)
		if err != nil {
			return err
		}

		// Create unique index
		_, err = tx.Exec(ctx, `
CREATE UNIQUE INDEX enclosures_user_entry_url_unique_idx
  ON enclosures(user_id, entry_id, md5(url))`)
		return err
	}),

	sqlMigration(
		`ALTER TABLE users ADD COLUMN mark_read_on_view boolean default 't'`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN notion_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN notion_token text default '';
ALTER TABLE integrations ADD COLUMN notion_page_id text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN readwise_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN readwise_api_key text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN apprise_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN apprise_url text default '';
ALTER TABLE integrations ADD COLUMN apprise_services_url text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN shiori_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN shiori_url text default '';
ALTER TABLE integrations ADD COLUMN shiori_username text default '';
ALTER TABLE integrations ADD COLUMN shiori_password text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN shaarli_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN shaarli_url text default '';
ALTER TABLE integrations ADD COLUMN shaarli_api_secret text default '';`),

	sqlMigration(
		`ALTER TABLE feeds ADD COLUMN apprise_service_urls text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN webhook_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN webhook_url text default '';
ALTER TABLE integrations ADD COLUMN webhook_secret text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN telegram_bot_topic_id int;
ALTER TABLE integrations
  ADD COLUMN telegram_bot_disable_web_page_preview bool default 'f';
ALTER TABLE integrations
  ADD COLUMN telegram_bot_disable_notification bool default 'f';`),

	sqlMigration(`
ALTER TABLE integrations
  ADD COLUMN telegram_bot_disable_buttons bool default 'f';`),

	sqlMigration(`
-- Speed up has_enclosure
CREATE INDEX enclosures_entry_id_idx ON enclosures(entry_id);

-- Speed up unread page
CREATE INDEX entries_user_status_published_idx
  ON entries(user_id, status, published_at);
CREATE INDEX entries_user_status_created_idx
  ON entries(user_id, status, created_at);
CREATE INDEX feeds_feed_id_hide_globally_idx
  ON feeds(id, hide_globally);

-- Speed up history page
CREATE INDEX entries_user_status_changed_published_idx
  ON entries(user_id, status, changed_at, published_at);`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN rssbridge_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN rssbridge_url text default '';`),

	sqlMigration(`
CREATE TABLE webauthn_credentials (
	handle bytea primary key,
	cred_id bytea unique not null,
	user_id int references users(id) on delete cascade not null,
	public_key bytea not null,
	attestation_type varchar(255) not null,
	aaguid bytea,
	sign_count bigint,
	clone_warning bool,
	name text,
	added_on timestamp with time zone default now(),
	last_seen_on timestamp with time zone default now()
);`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN omnivore_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN omnivore_api_key text default '';
ALTER TABLE integrations ADD COLUMN omnivore_url text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN linkace_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN linkace_url text default '';
ALTER TABLE integrations ADD COLUMN linkace_api_key text default '';
ALTER TABLE integrations ADD COLUMN linkace_tags text default '';
ALTER TABLE integrations ADD COLUMN linkace_is_private bool default 't';
ALTER TABLE integrations ADD COLUMN linkace_check_disabled bool default 't';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN linkwarden_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN linkwarden_url text default '';
ALTER TABLE integrations ADD COLUMN linkwarden_api_key text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN readeck_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN readeck_only_url bool default 'f';
ALTER TABLE integrations ADD COLUMN readeck_url text default '';
ALTER TABLE integrations ADD COLUMN readeck_api_key text default '';
ALTER TABLE integrations ADD COLUMN readeck_labels text default '';`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN disable_http2 bool default 'f'`),

	sqlMigration(
		`ALTER TABLE users ADD COLUMN media_playback_rate numeric default 1;`),

	// the WHERE part speed-up the request a lot
	sqlMigration(
		`UPDATE entries SET tags = array_remove(tags, '') WHERE '' = ANY(tags);`),

	// Entry URLs can exceeds btree maximum size. Checking entry existence is now
	// using entries_feed_id_status_hash_idx index
	sqlMigration(`DROP INDEX entries_feed_url_idx`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN raindrop_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN raindrop_token text default '';
ALTER TABLE integrations ADD COLUMN raindrop_collection_id text default '';
ALTER TABLE integrations ADD COLUMN raindrop_tags text default '';`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN description text default ''`),

	sqlMigration(`
ALTER TABLE users
	ADD COLUMN block_filter_entry_rules text not null default '',
	ADD COLUMN keep_filter_entry_rules text not null default ''`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN betula_url text default '';
ALTER TABLE integrations ADD COLUMN betula_token text default '';
ALTER TABLE integrations ADD COLUMN betula_enabled bool default 'f';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN ntfy_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN ntfy_url text default '';
ALTER TABLE integrations ADD COLUMN ntfy_topic text default '';
ALTER TABLE integrations ADD COLUMN ntfy_api_token text default '';
ALTER TABLE integrations ADD COLUMN ntfy_username text default '';
ALTER TABLE integrations ADD COLUMN ntfy_password text default '';
ALTER TABLE integrations ADD COLUMN ntfy_icon_url text default '';

ALTER TABLE feeds ADD COLUMN ntfy_enabled bool default 'f';
ALTER TABLE feeds ADD COLUMN ntfy_priority int default '3';`),

	sqlMigration(`
ALTER TABLE users
  ADD COLUMN mark_read_on_media_player_completion bool default 'f';`),

	sqlMigration(
		`ALTER TABLE users ADD COLUMN custom_js text not null default '';`),

	sqlMigration(`
ALTER TABLE users ADD COLUMN external_font_hosts text not null default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN cubox_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN cubox_api_link text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN discord_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN discord_webhook_link text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN ntfy_internal_links bool default 'f';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN slack_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN slack_webhook_link text default '';`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN webhook_url text default '';`),

	sqlMigration(`
ALTER TABLE integrations ADD COLUMN pushover_enabled bool default 'f';
ALTER TABLE integrations ADD COLUMN pushover_user text default '';
ALTER TABLE integrations ADD COLUMN pushover_token text default '';
ALTER TABLE integrations ADD COLUMN pushover_device text default '';
ALTER TABLE integrations ADD COLUMN pushover_prefix text default '';

ALTER TABLE feeds ADD COLUMN pushover_enabled bool default 'f';
ALTER TABLE feeds ADD COLUMN pushover_priority int default '0';`),

	sqlMigration(`ALTER TABLE feeds ADD COLUMN ntfy_topic text default '';`),

	sqlMigration(`
ALTER TABLE icons ADD COLUMN external_id text default '';
CREATE UNIQUE INDEX icons_external_id_idx
  ON icons USING btree(external_id) WHERE external_id <> '';`),

	txMigration(func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
DECLARE id_cursor CURSOR FOR
SELECT id
  FROM icons
 WHERE external_id = '' FOR UPDATE`)
		if err != nil {
			return err
		}
		defer func() { _, _ = tx.Exec(ctx, "CLOSE id_cursor") }()

		for {
			var id int64
			err := tx.QueryRow(ctx, `FETCH NEXT FROM id_cursor`).Scan(&id)
			if errors.Is(err, pgx.ErrNoRows) {
				break
			} else if err != nil {
				return err
			}

			_, err = tx.Exec(ctx,
				`UPDATE icons SET external_id = $1 WHERE id = $2`,
				crypto.GenerateRandomStringHex(20), id)
			if err != nil {
				return err
			}
		}
		return nil
	}),

	// 108
	sqlMigration(`ALTER TABLE feeds ADD COLUMN proxy_url text default ''`),
}
