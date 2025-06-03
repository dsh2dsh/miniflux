// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/logging"
	"miniflux.app/v2/internal/model"
)

// CountUsers returns the total number of users.
func (s *Storage) CountUsers(ctx context.Context) (int, error) {
	rows, _ := s.db.Query(ctx, `SELECT count(*) FROM users`)
	count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return 0, fmt.Errorf("storage: unable count users: %w", err)
	}
	return count, nil
}

// SetLastLogin updates the last login date of a user.
func (s *Storage) SetLastLogin(ctx context.Context, userID int64) error {
	_, err := s.db.Exec(ctx,
		`UPDATE users SET last_login_at=now() WHERE id=$1`, userID)
	if err != nil {
		return fmt.Errorf(`store: unable to update last login date: %w`, err)
	}
	return nil
}

// UserExists checks if a user exists by using the given username.
func (s *Storage) UserExists(ctx context.Context, username string) bool {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS(
  SELECT FROM users WHERE username=LOWER($1))`, username)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error("storage: unable check user exists",
			slog.String("username", username), slog.Any("error", err))
		return false
	}
	return result
}

// AnotherUserExists checks if another user exists with the given username.
func (s *Storage) AnotherUserExists(ctx context.Context, userID int64,
	username string,
) bool {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS (
  SELECT FROM users WHERE id != $1 AND username=LOWER($2))`,
		userID, username)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		logging.FromContext(ctx).Error(
			"storage: unable check another user exists",
			slog.Int64("user_id", userID),
			slog.String("username", username),
			slog.Any("error", err))
		return false
	}
	return result
}

// CreateUser creates a new user.
func (s *Storage) CreateUser(ctx context.Context,
	userCreationRequest *model.UserCreationRequest,
) (user *model.User, _ error) {
	var hashedPassword string
	if userCreationRequest.Password != "" {
		hash, err := crypto.HashPassword(userCreationRequest.Password)
		if err != nil {
			return nil, err
		}
		hashedPassword = hash
	}

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rows, _ := tx.Query(ctx, `
INSERT INTO users
  (username, password, is_admin, google_id, openid_connect_id)
VALUES
  (LOWER($1), $2,      $3,       $4,        $5)
RETURNING
  id,
  username,
  is_admin,
  language,
  theme,
  timezone,
  entry_direction,
  entries_per_page,
  keyboard_shortcuts,
  show_reading_time,
  entry_swipe,
  gesture_nav,
  stylesheet,
  custom_js,
  external_font_hosts,
  google_id,
  openid_connect_id,
  display_mode,
  entry_order,
  default_reading_speed,
  cjk_reading_speed,
  default_home_page,
  categories_sorting_order,
  mark_read_on_view,
  media_playback_rate,
  block_filter_entry_rules,
  keep_filter_entry_rules,
  extra`,
			userCreationRequest.Username,
			hashedPassword,
			userCreationRequest.IsAdmin,
			userCreationRequest.GoogleID,
			userCreationRequest.OpenIDConnectID)

		u, err := pgx.CollectExactlyOneRow(rows,
			pgx.RowToAddrOfStructByNameLax[model.User])
		if err != nil {
			return fmt.Errorf(`create user: %w`, err)
		}
		user = u

		_, err = tx.Exec(ctx,
			`INSERT INTO categories (user_id, title) VALUES ($1, $2)`,
			user.ID, "All")
		if err != nil {
			return fmt.Errorf(`create default category: %w`, err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO integrations (user_id) VALUES ($1)`, user.ID)
		if err != nil {
			return fmt.Errorf(`create integration row: %w`, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf(`store: unable to create user: %w`, err)
	}
	return user, nil
}

// UpdateUser updates a user.
func (s *Storage) UpdateUser(ctx context.Context, user *model.User) error {
	user.ExternalFontHosts = strings.TrimSpace(user.ExternalFontHosts)

	var hashedPassword string
	if user.Password != "" {
		hash, err := crypto.HashPassword(user.Password)
		if err != nil {
			return err
		}
		hashedPassword = hash
	}

	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := s.db.Exec(ctx, `
UPDATE users SET
  username = LOWER($1),
  is_admin = $2,
  theme = $3,
  language = $4,
  timezone = $5,
  entry_direction = $6,
  entries_per_page = $7,
  keyboard_shortcuts = $8,
  show_reading_time = $9,
  entry_swipe = $10,
  gesture_nav = $11,
  stylesheet = $12,
  custom_js = $13,
  external_font_hosts = $14,
  google_id = $15,
  openid_connect_id = $16,
  display_mode = $17,
  entry_order = $18,
  default_reading_speed = $19,
  cjk_reading_speed = $20,
  default_home_page = $21,
  categories_sorting_order = $22,
  mark_read_on_view = $23,
  mark_read_on_media_player_completion = $24,
  media_playback_rate = $25,
  block_filter_entry_rules = $26,
  keep_filter_entry_rules = $27,
  extra = $28
WHERE id = $29`,
			user.Username,
			user.IsAdmin,
			user.Theme,
			user.Language,
			user.Timezone,
			user.EntryDirection,
			user.EntriesPerPage,
			user.KeyboardShortcuts,
			user.ShowReadingTime,
			user.EntrySwipe,
			user.GestureNav,
			user.Stylesheet,
			user.CustomJS,
			user.ExternalFontHosts,
			user.GoogleID,
			user.OpenIDConnectID,
			user.DisplayMode,
			user.EntryOrder,
			user.DefaultReadingSpeed,
			user.CJKReadingSpeed,
			user.DefaultHomePage,
			user.CategoriesSortingOrder,
			user.MarkReadOnView,
			user.MarkReadOnMediaPlayerCompletion,
			user.MediaPlaybackRate,
			user.BlockFilterEntryRules,
			user.KeepFilterEntryRules,
			user.Extra,
			user.ID)
		if err != nil {
			return fmt.Errorf(`update user: %w`, err)
		}

		if hashedPassword != "" {
			_, err := tx.Exec(ctx, `UPDATE users SET password=$1 WHERE id=$2`,
				user.ID)
			if err != nil {
				return fmt.Errorf(`update password: %w`, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf(`store: unable to update user: %w`, err)
	}
	return nil
}

// UserByID finds a user by the ID.
func (s *Storage) UserByID(ctx context.Context, userID int64,
) (*model.User, error) {
	query := `
SELECT
  id,
  username,
  is_admin,
  theme,
  language,
  timezone,
  entry_direction,
  entries_per_page,
  keyboard_shortcuts,
  show_reading_time,
  entry_swipe,
  gesture_nav,
  last_login_at,
  stylesheet,
  custom_js,
  external_font_hosts,
  google_id,
  openid_connect_id,
  display_mode,
  entry_order,
  default_reading_speed,
  cjk_reading_speed,
  default_home_page,
  categories_sorting_order,
  mark_read_on_view,
  mark_read_on_media_player_completion,
  media_playback_rate,
  block_filter_entry_rules,
  keep_filter_entry_rules,
  extra
FROM users WHERE id = $1`
	return s.fetchUser(ctx, query, userID)
}

// UserByUsername finds a user by the username.
func (s *Storage) UserByUsername(ctx context.Context, username string,
) (*model.User, error) {
	query := `
SELECT
  id,
  username,
  is_admin,
  theme,
  language,
  timezone,
  entry_direction,
  entries_per_page,
  keyboard_shortcuts,
  show_reading_time,
  entry_swipe,
  gesture_nav,
  last_login_at,
  stylesheet,
  custom_js,
  external_font_hosts,
  google_id,
  openid_connect_id,
  display_mode,
  entry_order,
  default_reading_speed,
  cjk_reading_speed,
  default_home_page,
  categories_sorting_order,
  mark_read_on_view,
  mark_read_on_media_player_completion,
  media_playback_rate,
  block_filter_entry_rules,
  keep_filter_entry_rules,
  extra
FROM users WHERE username=LOWER($1)`
	return s.fetchUser(ctx, query, username)
}

// UserByField finds a user by a field value.
func (s *Storage) UserByField(ctx context.Context, field, value string,
) (*model.User, error) {
	query := fmt.Sprintf(`
SELECT
  id,
  username,
  is_admin,
  theme,
  language,
  timezone,
  entry_direction,
  entries_per_page,
  keyboard_shortcuts,
  show_reading_time,
  entry_swipe,
  gesture_nav,
  last_login_at,
  stylesheet,
  custom_js,
  external_font_hosts,
  google_id,
  openid_connect_id,
  display_mode,
  entry_order,
  default_reading_speed,
  cjk_reading_speed,
  default_home_page,
  categories_sorting_order,
  mark_read_on_view,
  mark_read_on_media_player_completion,
  media_playback_rate,
  block_filter_entry_rules,
  keep_filter_entry_rules,
  extra
FROM users WHERE %s = $1`,
		pgx.Identifier([]string{field}).Sanitize())
	return s.fetchUser(ctx, query, value)
}

// AnotherUserWithFieldExists returns true if a user has the value set for the
// given field.
func (s *Storage) AnotherUserWithFieldExists(ctx context.Context, userID int64,
	field, value string,
) (bool, error) {
	rows, _ := s.db.Query(ctx,
		fmt.Sprintf(`
SELECT EXISTS(
  SELECT FROM users WHERE id <> $1 AND %s=$2)`,
			pgx.Identifier([]string{field}).Sanitize()),
		userID, value)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf(
			"storage: unable check another user exists: %w", err)
	}
	return result, nil
}

// UserByAPIKey returns a User from an API Key.
func (s *Storage) UserByAPIKey(ctx context.Context, token string,
) (*model.User, error) {
	query := `
SELECT
  u.id,
  u.username,
  u.is_admin,
  u.theme,
  u.language,
  u.timezone,
  u.entry_direction,
  u.entries_per_page,
  u.keyboard_shortcuts,
  u.show_reading_time,
  u.entry_swipe,
  u.gesture_nav,
  u.last_login_at,
  u.stylesheet,
  u.custom_js,
  u.external_font_hosts,
  u.google_id,
  u.openid_connect_id,
  u.display_mode,
  u.entry_order,
  u.default_reading_speed,
  u.cjk_reading_speed,
  u.default_home_page,
  u.categories_sorting_order,
  u.mark_read_on_view,
  u.mark_read_on_media_player_completion,
  u.media_playback_rate,
  u.block_filter_entry_rules,
  u.keep_filter_entry_rules,
  u.extra
FROM users u
     LEFT JOIN api_keys ON api_keys.user_id=u.id
WHERE api_keys.token = $1`
	return s.fetchUser(ctx, query, token)
}

func (s *Storage) fetchUser(ctx context.Context, query string, args ...any,
) (*model.User, error) {
	rows, _ := s.db.Query(ctx, query, args...)
	user, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByNameLax[model.User])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("storage: unable to fetch user: %w", err)
	}
	return user, nil
}

// RemoveUser deletes user data.
func (s *Storage) RemoveUser(ctx context.Context, userID int64) error {
	log := logging.FromContext(ctx).With(slog.Int64("user_id", userID))
	if err := s.deleteUserFeeds(ctx, userID); err != nil {
		log.Error("Unable to delete user feeds", slog.Any("error", err))
		return err
	}

	if err := s.removeUser(ctx, userID); err != nil {
		log.Error("storage: failed delete user", slog.Any("error", err))
		return err
	}
	return nil
}

func (s *Storage) deleteUserFeeds(ctx context.Context, userID int64) error {
	rows, _ := s.db.Query(ctx, `SELECT id FROM feeds WHERE user_id=$1`, userID)

	feedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return fmt.Errorf(`store: unable to get user feeds: %w`, err)
	}

	if err := s.RemoveMultipleFeeds(ctx, userID, feedIDs); err != nil {
		return err
	}
	return nil
}

// removeUser deletes a user.
func (s *Storage) removeUser(ctx context.Context, userID int64) error {
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
		if err != nil {
			return fmt.Errorf("remove user: %w", err)
		}

		_, err = tx.Exec(ctx, `DELETE FROM integrations WHERE user_id=$1`, userID)
		if err != nil {
			return fmt.Errorf("remove integration settings: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf(`storage: unable to remove user #%d: %w`, userID, err)
	}
	return nil
}

// Users returns all users.
func (s *Storage) Users(ctx context.Context) (model.Users, error) {
	rows, _ := s.db.Query(ctx, `
SELECT
  id,
  username,
  is_admin,
  theme,
  language,
  timezone,
  entry_direction,
  entries_per_page,
  keyboard_shortcuts,
  show_reading_time,
  entry_swipe,
  gesture_nav,
  last_login_at,
  stylesheet,
  custom_js,
  external_font_hosts,
  google_id,
  openid_connect_id,
  display_mode,
  entry_order,
  default_reading_speed,
  cjk_reading_speed,
  default_home_page,
  categories_sorting_order,
  mark_read_on_view,
  mark_read_on_media_player_completion,
  media_playback_rate,
  block_filter_entry_rules,
  keep_filter_entry_rules,
  extra
FROM users ORDER BY username ASC`)

	users, err := pgx.CollectRows(rows,
		pgx.RowToAddrOfStructByNameLax[model.User])
	if err != nil {
		return nil, fmt.Errorf(`store: unable to fetch users: %w`, err)
	}
	return users, nil
}

// CheckPassword validate the hashed password.
func (s *Storage) CheckPassword(ctx context.Context, username, password string,
) error {
	var hash string
	username = strings.ToLower(username)

	rows, _ := s.db.Query(ctx,
		`SELECT password FROM users WHERE username=$1`, username)

	hash, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[string])
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf(`store: unable to find this user: %s`, username)
	} else if err != nil {
		return fmt.Errorf(`store: unable to fetch user: %w`, err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf(`store: invalid password for %q: %w`, username, err)
	}
	return nil
}

// HasPassword returns true if the given user has a password defined.
func (s *Storage) HasPassword(ctx context.Context, userID int64) (bool, error) {
	rows, _ := s.db.Query(ctx, `
SELECT EXISTS(
  SELECT FROM users WHERE id=$1 AND password <> '')`, userID)

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf(`store: unable check password: %w`, err)
	}
	return result, nil
}

func (s *Storage) UserSession(ctx context.Context, id string,
) (*model.User, *model.Session, error) {
	query := `
SELECT
  s.user_id,
  s.data,
  s.created_at,
  s.updated_at,
  u.id,
  u.username,
  u.is_admin,
  u.language,
  u.timezone,
  u.theme,
  u.entry_direction,
  u.keyboard_shortcuts,
  u.entries_per_page,
  u.show_reading_time,
  u.entry_swipe,
  u.gesture_nav,
  u.last_login_at,
  u.stylesheet,
  u.custom_js,
  u.external_font_hosts,
  u.google_id,
  u.openid_connect_id,
  u.display_mode,
  u.entry_order,
  u.default_reading_speed,
  u.cjk_reading_speed,
  u.default_home_page,
  u.categories_sorting_order,
  u.mark_read_on_view,
  u.mark_read_on_media_player_completion,
  u.media_playback_rate,
  u.block_filter_entry_rules,
  u.keep_filter_entry_rules,
  u.extra
FROM sessions s, users u
WHERE s.id = $1 AND u.id = s.user_id`

	user := &model.User{}
	sess := &model.Session{ID: id}

	err := s.db.QueryRow(ctx, query, id).Scan(
		&sess.UserID,
		&sess.Data,
		&sess.CreatedAt,
		&sess.UpdatedAt,
		&user.ID,
		&user.Username,
		&user.IsAdmin,
		&user.Language,
		&user.Timezone,
		&user.Theme,
		&user.EntryDirection,
		&user.KeyboardShortcuts,
		&user.EntriesPerPage,
		&user.ShowReadingTime,
		&user.EntrySwipe,
		&user.GestureNav,
		&user.LastLoginAt,
		&user.Stylesheet,
		&user.CustomJS,
		&user.ExternalFontHosts,
		&user.GoogleID,
		&user.OpenIDConnectID,
		&user.DisplayMode,
		&user.EntryOrder,
		&user.DefaultReadingSpeed,
		&user.CJKReadingSpeed,
		&user.DefaultHomePage,
		&user.CategoriesSortingOrder,
		&user.MarkReadOnView,
		&user.MarkReadOnMediaPlayerCompletion,
		&user.MediaPlaybackRate,
		&user.BlockFilterEntryRules,
		&user.KeepFilterEntryRules,
		&user.Extra)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("storage: fetch user with session: %w", err)
	}
	return user, sess, nil
}
