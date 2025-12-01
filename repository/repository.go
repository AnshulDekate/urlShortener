package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)
type URL struct {
	ID             int64     `json:"id"`
	LongURL        string    `json:"long_url"`
	ShortCode      string    `json:"short_url"`
	ClickCount     int       `json:"click_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) HealthCheck(ctx context.Context) error {
	return r.DB.PingContext(ctx)
}

func (r *Repository) InsertURL(longURL string) (int64, error) {
	const insertQuery = `
	INSERT INTO urls (long_url, short_url, updated_at) 
	VALUES ($1, '', NOW()) RETURNING id
	`
	var id int64
	err := r.DB.QueryRowContext(context.Background(), insertQuery, longURL).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert URL: %w", err)
	}
	return id, nil
}

func (r *Repository) UpdateShortCode(id int64, shortCode string) error {
	const updateQuery = `
	UPDATE urls SET short_url = $1, updated_at = NOW() WHERE id = $2
	`
	_, err := r.DB.ExecContext(context.Background(), updateQuery, shortCode, id)
	if err != nil {
		return fmt.Errorf("failed to update short code for ID %d: %w", id, err)
	}
	return nil
}

func (r *Repository) FindExistingShortCode(longURL string) (string, error) {
	const query = "SELECT short_url FROM urls WHERE long_url = $1 AND short_url != ''"
	var shortCode string
	
	err := r.DB.QueryRowContext(context.Background(), query, longURL).Scan(&shortCode)
	
	if err == sql.ErrNoRows {
		return "", nil 
	}
	if err != nil {
		return "", fmt.Errorf("error querying for existing URL: %w", err)
	}
	
	return shortCode, nil 
}

func (r *Repository) IsShortCodeUnique(code string) (bool, error) {
	const query = "SELECT EXISTS (SELECT 1 FROM urls WHERE short_url = $1)"
	var exists bool
	
	err := r.DB.QueryRowContext(context.Background(), query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking short code uniqueness: %w", err)
	}
	
	return !exists, nil
}


func (r *Repository) LookupAndTrack(shortCode string) (string, error) {
	const selectAndUpdateQuery = `
	UPDATE urls 
	SET 
		click_count = click_count + 1, 
		last_accessed_at = NOW(), 
		updated_at = NOW() 
	WHERE short_url = $1
	RETURNING long_url`
	
	var longURL string
	
	err := r.DB.QueryRowContext(context.Background(), selectAndUpdateQuery, shortCode).Scan(&longURL)
	
	if err == sql.ErrNoRows {
		return "", sql.ErrNoRows 
	}
	if err != nil {
		return "", fmt.Errorf("error tracking click for short code %s: %w", shortCode, err)
	}
	
	return longURL, nil
}


func (r *Repository) ListURLs(ctx context.Context, limit int, offset int) ([]URL, error) {
    query := `
        SELECT id, long_url, short_url, click_count, created_at, updated_at, last_accessed_at
        FROM urls
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
    rows, err := r.DB.QueryContext(ctx, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to query URLs: %w", err)
    }
    defer rows.Close()

    var urls []URL
    for rows.Next() {
        var u URL
        var lastAccessedAt sql.NullTime
        
        err := rows.Scan(
            &u.ID,
            &u.LongURL,
            &u.ShortCode,
            &u.ClickCount,
            &u.CreatedAt,
            &u.UpdatedAt,
            &lastAccessedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan URL row: %w", err)
        }
        
        if lastAccessedAt.Valid {
            u.LastAccessedAt = lastAccessedAt.Time
        }
        
        urls = append(urls, u)
    }
    
    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error during rows iteration: %w", err)
    }
    
    return urls, nil
}

func (r *Repository) GetTotalURLCount(ctx context.Context) (int, error) {
    var count int
    query := `SELECT COUNT(id) FROM urls`
    
    err := r.DB.QueryRowContext(ctx, query).Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("failed to query total count: %w", err)
    }
    return count, nil
}

