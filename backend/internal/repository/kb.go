package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"aibuddy/internal/domain"

	"github.com/google/uuid"
)

type KB struct{ DB *sql.DB }

var tagStripper = strings.NewReplacer("<b>", "", "</b>", "")

func (r *KB) Search(ctx context.Context, id domain.Identity, query string, limit int) ([]domain.KBHit, error) {
	var team any
	if id.TeamID != uuid.Nil {
		team = id.TeamID
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT title, source, snippet FROM (
		  SELECT DISTINCT ON (d.id) d.title, d.source, ts_rank(c.tsv, s.q) AS rank,
		         ts_headline('russian', c.content, s.q, 'MaxFragments=1,MaxWords=30,MinWords=10') AS snippet
		  FROM kb_chunks c
		  JOIN kb_documents d ON d.id = c.document_id
		  CROSS JOIN LATERAL (
		    SELECT replace(websearch_to_tsquery('russian', $1)::text, '&', '|')::tsquery AS q) s
		  WHERE c.tsv @@ s.q
		    AND $2 = ANY(d.allowed_roles)
		    AND (d.team_id IS NULL OR d.team_id = $3::uuid)
		  ORDER BY d.id, rank DESC) t
		ORDER BY rank DESC, title
		LIMIT $4`, query, string(id.Role), team, limit)
	if err != nil {
		return nil, fmt.Errorf("kb search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hits := []domain.KBHit{}
	for rows.Next() {
		var h domain.KBHit
		if err := rows.Scan(&h.Title, &h.Source, &h.Snippet); err != nil {
			return nil, fmt.Errorf("scan kb hit: %w", err)
		}
		h.Snippet = tagStripper.Replace(h.Snippet)
		hits = append(hits, h)
	}
	return hits, rows.Err()
}
