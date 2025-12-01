package service

import (
	"context" 
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings" 
	"time"

	"github.com/AnshulDekate/urlShortener/repository" 
)

var (
	ErrNotFound = errors.New("short code not found")
)

type URLListResponse struct {
    URLs        []repository.URL `json:"urls"`
    TotalCount  int              `json:"total_count"`
    Page        int              `json:"page"`
    Limit       int              `json:"limit"`
    TotalPages  int              `json:"total_pages"`
}

const (
	MaxShortCodeLength = 10
	Base62Alphabet     = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

type Service struct {
	Repo           *repository.Repository
	MaxRetries     int 
	DesiredLength  int 
}

func generateRandomCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	result := make([]byte, length)
	alphabetLength := len(Base62Alphabet)
	
	for i, b := range bytes {
		result[i] = Base62Alphabet[int(b)%alphabetLength]
	}
	
	return string(result), nil
}

func (s *Service) HealthCheck(ctx context.Context) error {
	return s.Repo.HealthCheck(ctx)
}

func (s *Service) CreateShortURL(longURL string) (string, error) {
	desiredLen := s.DesiredLength
	if desiredLen == 0 {
		desiredLen = MaxShortCodeLength 
	}
	maxRetries := s.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5 
	}
	
	if _, err := url.ParseRequestURI(longURL); err != nil {
		return "", errors.New("invalid URL format")
	}

	// Idempotency Check
	existingShortCode, err := s.Repo.FindExistingShortCode(longURL)
	if err != nil {
		log.Printf("FATAL ERROR: Idempotency check failed for %s: %v", longURL, err)
		return "", err
	}
	if existingShortCode != "" {
		log.Printf("INFO: Idempotency hit for %s. Returning existing code: %s", longURL, existingShortCode)
		return existingShortCode, nil
	}
    log.Printf("INFO: No existing short code found for %s. Proceeding to insert.", longURL)


	// Insert the long URL first
	newID, err := s.Repo.InsertURL(longURL) 
	if err != nil {
		if strings.Contains(err.Error(), "unique_long_url") {
			log.Printf("WARN: Concurrent insertion detected for %s. Retrying idempotency check.", longURL)
			return s.Repo.FindExistingShortCode(longURL)
		}
		log.Printf("FATAL ERROR: Primary InsertURL failed for %s: %v", longURL, err)
		return "", err
	}
    log.Printf("INFO: Successfully inserted new row with ID: %d", newID)

	var shortCode string
	// Random Generation with Configurable Collision Retry Loop
	for i := 0; i < maxRetries; i++ {
		code, err := generateRandomCode(desiredLen)
		if err != nil {
			log.Printf("FATAL ERROR: Code generation failed: %v", err)
			return "", fmt.Errorf("code generation failed: %w", err)
		}

		isUnique, err := s.Repo.IsShortCodeUnique(code)
		if err != nil {
			log.Printf("FATAL ERROR: Uniqueness check failed for code %s: %v", code, err)
			return "", err
		}

		if isUnique {
			shortCode = code
			log.Printf("INFO: Found unique code %s on attempt %d.", shortCode, i+1)
			break
		}
		
		log.Printf("COLLISION: Detected for code: %s. Retrying... (%d/%d)", code, i+1, maxRetries)
		time.Sleep(10 * time.Millisecond) 
	}

	if shortCode == "" {
		log.Printf("FATAL ERROR: Failed to find unique code after %d retries.", maxRetries)
		return "", errors.New("service capacity exhausted")
	}
	
	// Final check against the 10-character assignment requirement
	if len(shortCode) > MaxShortCodeLength {
		log.Printf("FATAL ERROR: Generated code length %d exceeds max %d.", len(shortCode), MaxShortCodeLength)
		return "", errors.New("internal error: generated code exceeds max length")
	}

	// Update the row with the unique short code
	if err := s.Repo.UpdateShortCode(newID, shortCode); err != nil {
		log.Printf("FATAL ERROR: UpdateShortCode failed for ID %d and code %s: %v", newID, shortCode, err)
		return "", err
	}
    log.Printf("INFO: Successfully updated ID %d with short code %s.", newID, shortCode)

	return shortCode, nil
}

func (s *Service) GetLongURL(shortCode string) (string, error) {
	longURL, err := s.Repo.LookupAndTrack(shortCode)
	
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("short code not found")
	}
    if err != nil {
        log.Printf("FATAL ERROR: LookupAndTrack failed for code %s: %v", shortCode, err)
    }
	return longURL, err
}


func (s *Service) ListURLs(ctx context.Context, page int, limit int) (*URLListResponse, error) {

    totalCount, err := s.Repo.GetTotalURLCount(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get total URL count: %w", err)
    }

    offset := (page - 1) * limit
    totalPages := (totalCount + limit - 1) / limit 
    
    if totalPages == 0 {
        totalPages = 1
    } else if page > totalPages {
        page = totalPages
        offset = (page - 1) * limit
    }

    urls, err := s.Repo.ListURLs(ctx, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch paginated URLs: %w", err)
    }
    
    return &URLListResponse{
        URLs: urls,
        TotalCount: totalCount,
        Page: page,
        Limit: limit,
        TotalPages: totalPages,
    }, nil
}

