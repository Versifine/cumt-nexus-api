package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	platformdb "github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type seedConfig struct {
	Users         int  `json:"users"`
	Communities   int  `json:"communities"`
	Posts         int  `json:"posts"`
	Comments      int  `json:"comments"`
	PostVotes     int  `json:"post_votes"`
	PostSaves     int  `json:"post_saves"`
	Notifications int  `json:"notifications"`
	Reports       int  `json:"reports"`
	Reset         bool `json:"reset"`
	UnsafeReset   bool `json:"unsafe_reset"`
}

type seedSummary struct {
	Database       string      `json:"database"`
	StartedAt      time.Time   `json:"started_at"`
	FinishedAt     time.Time   `json:"finished_at"`
	ElapsedSeconds float64     `json:"elapsed_seconds"`
	Config         seedConfig  `json:"config"`
	Tables         tableCounts `json:"tables"`
}

type tableCounts struct {
	Users         int `json:"users"`
	Communities   int `json:"communities"`
	Posts         int `json:"posts"`
	Comments      int `json:"comments"`
	PostVotes     int `json:"post_votes"`
	PostSaves     int `json:"post_saves"`
	Notifications int `json:"notifications"`
	Reports       int `json:"reports"`
}

func main() {
	cfg := seedConfig{}
	flag.IntVar(&cfg.Users, "users", 1000, "number of users to seed")
	flag.IntVar(&cfg.Communities, "communities", 50, "number of communities to seed")
	flag.IntVar(&cfg.Posts, "posts", 20000, "number of posts to seed")
	flag.IntVar(&cfg.Comments, "comments", 80000, "number of comments to seed")
	flag.IntVar(&cfg.PostVotes, "post-votes", 120000, "number of post votes to seed")
	flag.IntVar(&cfg.PostSaves, "post-saves", 30000, "number of post saves to seed")
	flag.IntVar(&cfg.Notifications, "notifications", 12000, "number of notifications for the primary test user")
	flag.IntVar(&cfg.Reports, "reports", 3000, "number of pending content reports to seed")
	flag.BoolVar(&cfg.Reset, "reset", true, "truncate application tables before seeding")
	flag.BoolVar(&cfg.UnsafeReset, "unsafe-reset", false, "allow reset even when database name does not contain loadtest")
	flag.Parse()

	if err := validateSeedConfig(cfg); err != nil {
		fatal(err)
	}

	appCfg, err := config.Load()
	if err != nil {
		fatal(fmt.Errorf("load config: %w", err))
	}

	ctx := context.Background()
	pool, err := platformdb.Open(ctx, appCfg.Postgres)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()

	startedAt := time.Now().UTC()
	if cfg.Reset {
		if err := resetDatabase(ctx, pool, appCfg.Postgres.Database, cfg.UnsafeReset); err != nil {
			fatal(err)
		}
	}
	if err := seed(ctx, pool, cfg); err != nil {
		fatal(err)
	}
	counts, err := countTables(ctx, pool)
	if err != nil {
		fatal(err)
	}
	finishedAt := time.Now().UTC()

	summary := seedSummary{
		Database:       appCfg.Postgres.Database,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		ElapsedSeconds: finishedAt.Sub(startedAt).Seconds(),
		Config:         cfg,
		Tables:         counts,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		fatal(err)
	}
}

func validateSeedConfig(cfg seedConfig) error {
	switch {
	case cfg.Users < 20:
		return errors.New("users must be at least 20")
	case cfg.Communities < 1:
		return errors.New("communities must be positive")
	case cfg.Posts < cfg.Communities:
		return errors.New("posts must be at least communities")
	case cfg.Comments < cfg.Posts:
		return errors.New("comments must be at least posts so tree endpoints have data")
	case cfg.PostVotes < 0 || cfg.PostSaves < 0 || cfg.Notifications < 0 || cfg.Reports < 0:
		return errors.New("counts cannot be negative")
	default:
		return nil
	}
}

func resetDatabase(ctx context.Context, pool *pgxpool.Pool, database string, unsafe bool) error {
	if !unsafe && !strings.Contains(strings.ToLower(database), "loadtest") {
		return fmt.Errorf("refusing to reset database %q; use a database name containing loadtest or pass -unsafe-reset", database)
	}

	rows, err := pool.Query(ctx, `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
			AND tablename <> 'schema_migrations'
	`)
	if err != nil {
		return fmt.Errorf("list public tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(tables) == 0 {
		return nil
	}
	sort.Strings(tables)
	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		quoted = append(quoted, "public."+quoteIdent(table))
	}
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+strings.Join(quoted, ", ")+" RESTART IDENTITY CASCADE"); err != nil {
		return fmt.Errorf("truncate loadtest tables: %w", err)
	}
	return nil
}

func seed(ctx context.Context, pool *pgxpool.Pool, cfg seedConfig) error {
	steps := []struct {
		name string
		sql  string
		args []any
	}{
		{"users", seedUsersSQL, []any{cfg.Users}},
		{"progressions", seedProgressionsSQL, []any{cfg.Users}},
		{"communities", seedCommunitiesSQL, []any{cfg.Communities}},
		{"memberships", seedMembershipsSQL, []any{cfg.Communities, cfg.Users}},
		{"community_follows", seedCommunityFollowsSQL, []any{cfg.Communities, cfg.Users}},
		{"user_follows", seedUserFollowsSQL, []any{cfg.Users}},
		{"posts", seedPostsSQL, []any{cfg.Posts, cfg.Communities, cfg.Users}},
		{"comments", seedCommentsSQL, []any{cfg.Comments, cfg.Posts, cfg.Users}},
		{"post_votes", seedPostVotesSQL, []any{cfg.PostVotes, cfg.Posts, cfg.Users}},
		{"post_saves", seedPostSavesSQL, []any{cfg.PostSaves, cfg.Posts, cfg.Users}},
		{"notifications", seedNotificationsSQL, []any{cfg.Notifications, cfg.Posts, cfg.Comments, cfg.Users}},
		{"reports", seedReportsSQL, []any{cfg.Reports, cfg.Posts, cfg.Comments, cfg.Users}},
	}
	for _, step := range steps {
		if _, err := pool.Exec(ctx, step.sql, step.args...); err != nil {
			return fmt.Errorf("seed %s: %w", step.name, err)
		}
	}
	return nil
}

func countTables(ctx context.Context, pool *pgxpool.Pool) (tableCounts, error) {
	var counts tableCounts
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM users),
			(SELECT COUNT(*)::int FROM communities),
			(SELECT COUNT(*)::int FROM posts),
			(SELECT COUNT(*)::int FROM comments),
			(SELECT COUNT(*)::int FROM post_votes),
			(SELECT COUNT(*)::int FROM post_saves),
			(SELECT COUNT(*)::int FROM notifications),
			(SELECT COUNT(*)::int FROM content_reports)
	`).Scan(
		&counts.Users,
		&counts.Communities,
		&counts.Posts,
		&counts.Comments,
		&counts.PostVotes,
		&counts.PostSaves,
		&counts.Notifications,
		&counts.Reports,
	)
	if err != nil {
		return tableCounts{}, err
	}
	return counts, nil
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "loadtest seed:", err)
	os.Exit(1)
}

const seedUsersSQL = `
INSERT INTO users (
	id,
	username,
	password_hash,
	status,
	is_platform_staff,
	display_name,
	avatar_url,
	banner_url,
	headline,
	bio,
	email,
	email_verified_at,
	last_login_at,
	password_updated_at,
	tokens_revoked_after,
	deleted_at,
	platform_role,
	created_at,
	updated_at
)
SELECT
	('00000000-0000-1000-0000-' || lpad(i::text, 12, '0'))::uuid,
	'ltuser' || lpad(i::text, 6, '0'),
	'loadtest-no-login',
	'active',
	i <= 10,
	'Load Test User ' || i,
	'',
	'',
	'Load test participant',
	'Synthetic user for local load testing.',
	'ltuser' || lpad(i::text, 6, '0') || '@cumt.edu.cn',
	NOW() - INTERVAL '30 days',
	NOW() - (i % 1440) * INTERVAL '1 minute',
	NOW() - INTERVAL '30 days',
	NULL,
	NULL,
	CASE
		WHEN i = 1 THEN 'owner'
		WHEN i <= 10 THEN 'staff'
		ELSE NULL
	END,
	NOW() - INTERVAL '60 days' + (i % 1000) * INTERVAL '1 minute',
	NOW() - (i % 1440) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (id) DO NOTHING
`

const seedProgressionsSQL = `
INSERT INTO user_progressions (user_id, xp_total, active_title_grant_id, updated_at)
SELECT
	('00000000-0000-1000-0000-' || lpad(i::text, 12, '0'))::uuid,
	(i % 3000)::int,
	NULL,
	NOW() - (i % 1440) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (user_id) DO NOTHING
`

const seedCommunitiesSQL = `
INSERT INTO communities (
	id,
	slug,
	name,
	description,
	kind,
	status,
	visibility,
	created_by,
	avatar_url,
	banner_url,
	created_at,
	updated_at
)
SELECT
	('00000000-0000-2000-0000-' || lpad(i::text, 12, '0'))::uuid,
	'lt-community-' || lpad(i::text, 4, '0'),
	'Load Community ' || i,
	'Synthetic community used for repeatable API load tests.',
	'user_created',
	'active',
	'public',
	'00000000-0000-1000-0000-000000000001'::uuid,
	'',
	'',
	NOW() - INTERVAL '30 days' + (i % 2000) * INTERVAL '1 minute',
	NOW() - (i % 1440) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (id) DO NOTHING
`

const seedMembershipsSQL = `
WITH memberships AS (
	SELECT
		c.i AS community_i,
		1 AS user_i,
		'owner'::text AS role
	FROM generate_series(1, $1::int) AS c(i)
	UNION ALL
	SELECT
		c.i AS community_i,
		2 + ((c.i + m.i) % GREATEST($2::int - 1, 1)) AS user_i,
		CASE WHEN m.i <= 2 THEN 'moderator' ELSE 'member' END AS role
	FROM generate_series(1, $1::int) AS c(i)
	CROSS JOIN generate_series(1, 18) AS m(i)
)
INSERT INTO community_memberships (
	community_id,
	user_id,
	role,
	status,
	created_at,
	updated_at
)
SELECT DISTINCT ON (community_i, user_i)
	('00000000-0000-2000-0000-' || lpad(community_i::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad(user_i::text, 12, '0'))::uuid,
	role,
	'active',
	NOW() - INTERVAL '20 days',
	NOW() - INTERVAL '1 day'
FROM memberships
ORDER BY community_i, user_i, CASE role WHEN 'owner' THEN 0 WHEN 'moderator' THEN 1 ELSE 2 END
ON CONFLICT (community_id, user_id) DO NOTHING
`

const seedCommunityFollowsSQL = `
INSERT INTO community_follows (community_id, user_id, created_at)
SELECT
	('00000000-0000-2000-0000-' || lpad((((i - 1) % $1::int) + 1)::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad((((i * 13 - 1) % $2::int) + 1)::text, 12, '0'))::uuid,
	NOW() - (i % 10000) * INTERVAL '1 minute'
FROM generate_series(1, LEAST($1::int * $2::int, 20000)) AS s(i)
ON CONFLICT (community_id, user_id) DO NOTHING
`

const seedUserFollowsSQL = `
INSERT INTO user_follows (follower_id, following_id, created_at)
SELECT
	('00000000-0000-1000-0000-' || lpad((((i - 1) % $1::int) + 1)::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad((((i * 17 - 1) % $1::int) + 1)::text, 12, '0'))::uuid,
	NOW() - (i % 10000) * INTERVAL '1 minute'
FROM generate_series(1, LEAST($1::int * 10, 50000)) AS s(i)
WHERE (((i - 1) % $1::int) + 1) <> (((i * 17 - 1) % $1::int) + 1)
ON CONFLICT (follower_id, following_id) DO NOTHING
`

const seedPostsSQL = `
INSERT INTO posts (
	id,
	community_id,
	author_id,
	title,
	body,
	status,
	is_locked,
	is_pinned,
	is_nsfw,
	is_spoiler,
	flair_text,
	created_at,
	updated_at
)
SELECT
	('00000000-0000-3000-0000-' || lpad(i::text, 12, '0'))::uuid,
	('00000000-0000-2000-0000-' || lpad((((i - 1) % $2::int) + 1)::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad((((i * 7 - 1) % $3::int) + 1)::text, 12, '0'))::uuid,
	'Load test post ' || i || ' about nexus search and community governance',
	'This synthetic post body gives the PostgreSQL search index and feed queries enough text to work with. Post number ' || i || ' mentions nexus, moderation, feed, comments, and notifications.',
	'visible',
	false,
	(i % 997 = 0),
	(i % 89 = 0),
	(i % 131 = 0),
	CASE WHEN i % 11 = 0 THEN 'discussion' ELSE '' END,
	NOW() - (i % 200000) * INTERVAL '1 minute',
	NOW() - (i % 200000) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (id) DO NOTHING
`

const seedCommentsSQL = `
INSERT INTO comments (
	id,
	post_id,
	author_id,
	parent_id,
	body,
	status,
	is_locked,
	created_at,
	updated_at
)
SELECT
	('00000000-0000-4000-0000-' || lpad(i::text, 12, '0'))::uuid,
	('00000000-0000-3000-0000-' || lpad((((i - 1) % $2::int) + 1)::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad((((i * 11 - 1) % $3::int) + 1)::text, 12, '0'))::uuid,
	CASE
		WHEN i > $2::int AND i % 3 <> 0
			THEN ('00000000-0000-4000-0000-' || lpad((i - $2::int)::text, 12, '0'))::uuid
		ELSE NULL
	END,
	'Load test comment ' || i || ' with enough text for excerpt generation and tree rendering.',
	'visible',
	false,
	NOW() - (i % 200000) * INTERVAL '1 minute',
	NOW() - (i % 200000) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (id) DO NOTHING
`

const seedPostVotesSQL = `
INSERT INTO post_votes (post_id, user_id, value, created_at, updated_at)
SELECT
	('00000000-0000-3000-0000-' || lpad((((i - 1) % $2::int) + 1)::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad(((((i - 1) / $2::int) % $3::int) + 1)::text, 12, '0'))::uuid,
	CASE WHEN i % 7 = 0 THEN -1 ELSE 1 END,
	NOW() - (i % 200000) * INTERVAL '1 minute',
	NOW() - (i % 200000) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (post_id, user_id) DO NOTHING
`

const seedPostSavesSQL = `
INSERT INTO post_saves (post_id, user_id, created_at)
SELECT
	('00000000-0000-3000-0000-' || lpad((((i - 1) % $2::int) + 1)::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad(((((i - 1) / $2::int) % $3::int) + 1)::text, 12, '0'))::uuid,
	NOW() - (i % 100000) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (post_id, user_id) DO NOTHING
`

const seedNotificationsSQL = `
INSERT INTO notifications (
	id,
	recipient_id,
	type,
	title,
	body,
	source_type,
	source_id,
	read_at,
	aggregate_key,
	aggregate_count,
	last_actor_id,
	created_at,
	updated_at
)
SELECT
	('00000000-0000-6000-0000-' || lpad(i::text, 12, '0'))::uuid,
	'00000000-0000-1000-0000-000000000001'::uuid,
	CASE
		WHEN i % 5 = 0 THEN 'mention'
		WHEN i % 2 = 0 THEN 'comment_upvote'
		ELSE 'reply'
	END,
	'Load test notification ' || i,
	'Synthetic notification for load testing list joins and pagination.',
	CASE WHEN i % 2 = 0 THEN 'post' ELSE 'comment' END,
	CASE
		WHEN i % 2 = 0 THEN ('00000000-0000-3000-0000-' || lpad((((i - 1) % $2::int) + 1)::text, 12, '0'))
		ELSE ('00000000-0000-4000-0000-' || lpad((((i - 1) % $3::int) + 1)::text, 12, '0'))
	END,
	CASE WHEN i % 4 = 0 THEN NOW() - (i % 10000) * INTERVAL '1 minute' ELSE NULL END,
	'',
	1 + (i % 5),
	('00000000-0000-1000-0000-' || lpad((((i * 23 - 1) % $4::int) + 1)::text, 12, '0'))::uuid,
	NOW() - (i % 100000) * INTERVAL '1 minute',
	NOW() - (i % 100000) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (id) DO NOTHING
`

const seedReportsSQL = `
INSERT INTO content_reports (
	id,
	reporter_id,
	post_id,
	comment_id,
	reason,
	status,
	reviewed_by,
	reviewed_at,
	created_at,
	updated_at
)
SELECT
	('00000000-0000-5000-0000-' || lpad(i::text, 12, '0'))::uuid,
	('00000000-0000-1000-0000-' || lpad((((i * 29 - 1) % $4::int) + 1)::text, 12, '0'))::uuid,
	CASE
		WHEN i % 2 = 0 THEN ('00000000-0000-3000-0000-' || lpad((((i - 1) % $2::int) + 1)::text, 12, '0'))::uuid
		ELSE NULL
	END,
	CASE
		WHEN i % 2 <> 0 THEN ('00000000-0000-4000-0000-' || lpad((((i - 1) % $3::int) + 1)::text, 12, '0'))::uuid
		ELSE NULL
	END,
	'Load test pending report',
	'pending',
	NULL,
	NULL,
	NOW() - (i % 100000) * INTERVAL '1 minute',
	NOW() - (i % 100000) * INTERVAL '1 minute'
FROM generate_series(1, $1::int) AS s(i)
ON CONFLICT (id) DO NOTHING
`
