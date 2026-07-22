package sqlite

import "embed"

//go:embed migrations/*.sql
var migrationsFS embed.FS

// latestMigrationVersion must match the highest NNNN prefix under
// migrations/.
const latestMigrationVersion = 15

// legacyBaselineVersion is the version pre-existing databases (created by
// the old ad-hoc, string-matching migrator before schema_migrations
// tracking existed) are force-baselined to. It must stay pinned to the
// last migration whose schema that old migrator actually produced —
// NOT latestMigrationVersion — otherwise any later migration that does
// something the ad-hoc migrator never did (e.g. renaming a column) gets
// silently skipped for legacy databases instead of applied via m.Up().
const legacyBaselineVersion = 14
