# Database-schema diagrams

`converge-erd.puml` is the single ERD to include in the report. It shows every
table and foreign-key relationship.

The `tables/` directory holds optional per-table figures. They match the old
report layout, but do not include them beside the LaTeX data tables: that would
repeat the same information. Use either the figures or the tables for individual
schemas, and retain the full ERD in both cases.

Each file represents the current PostgreSQL implementation in
`backend/internal/adapter/db/postgres.go`.
