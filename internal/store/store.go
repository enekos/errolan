// Package store is the SQL repository layer. The single Store value holds the
// database handle; per-entity files (sites.go, users.go, comments.go, …) group
// the queries by concern so no one file becomes a god-object.
package store

import (
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/enekos/errolan/internal/models"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

var placeholderCache sync.Map // int → "?,?,?"


type reactCacheEntry struct {
	reactions map[string]int
	myReacts  map[int64][]string
}

type Store struct {
	DB         *sql.DB
	emojiMu    sync.RWMutex
	emojiCache map[int64][]*models.Emoji
	userMu     sync.RWMutex
	userCache  map[int64]*models.User
	reactMu    sync.RWMutex
	reactCache map[int64]reactCacheEntry
}

func New(db *sql.DB) *Store {
	return &Store{
		DB:         db,
		emojiCache: make(map[int64][]*models.Emoji),
		userCache:  make(map[int64]*models.User),
		reactCache: make(map[int64]reactCacheEntry),
	}
}

// newAPIKey returns a fresh "erl_…" site key. 24 random bytes = 192 bits of
// entropy, hex-encoded so the result is safe in URLs and headers.
func newAPIKey() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "erl_" + hex.EncodeToString(b)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// inPlaceholders builds "?,?,?,…" with n placeholders. Used for IN (…) lists.
// Results are cached so repeated sizes avoid string allocation.
func inPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	if s, ok := placeholderCache.Load(n); ok {
		return s.(string)
	}
	s := strings.Repeat("?,", n-1) + "?"
	placeholderCache.Store(n, s)
	return s
}

// gravatarURL is the canonical Gravatar URL for an email address. Empty email
// returns the empty string so callers can attach unconditionally.
var gravatarCache sync.Map // email → url

func gravatarURL(email string) string {
	if email == "" {
		return ""
	}
	if url, ok := gravatarCache.Load(email); ok {
		return url.(string)
	}
	sum := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	url := fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=64&d=mp", hex.EncodeToString(sum[:]))
	gravatarCache.Store(email, url)
	return url
}

// scanner is the common shape between *sql.Row and *sql.Rows so per-entity
// scan helpers can take either.
type scanner interface {
	Scan(...any) error
}
