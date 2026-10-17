package gitprovider

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Read until an empty page: Git servers can cap the requested page size (for
// example Gitea returns 50 even when limit=100). A short page is not the end.
// We construct page URLs ourselves instead of forwarding tokens to Link URLs.
func readAllPages[T any](ctx context.Context, path string, get func(context.Context, string, interface{}) error) ([]T, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	items := make([]T, 0)
	seen := map[[32]byte]bool{}
	for page := 1; page <= 1000; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		query := u.Query()
		query.Set("page", strconv.Itoa(page))
		u.RawQuery = query.Encode()
		var rows []T
		if err := get(ctx, u.String(), &rows); err != nil {
			return nil, fmt.Errorf("read Git list page %d: %w", page, err)
		}
		if len(rows) == 0 {
			return items, nil
		}
		content, err := json.Marshal(rows)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(content)
		if seen[digest] {
			return nil, fmt.Errorf("%w: Git API repeated page %d instead of advancing pagination", ErrInvalidResponse, page)
		}
		seen[digest] = true
		items = append(items, rows...)
	}
	return nil, fmt.Errorf("%w: Git list exceeded 1000 pages", ErrInvalidResponse)
}
