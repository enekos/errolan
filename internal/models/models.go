package models

type Site struct {
	ID             int64  `json:"id"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	APIKey         string `json:"api_key,omitempty"`
	AllowedOrigins string `json:"allowed_origins"`
	RequireAuth    bool   `json:"require_auth"`
	CreatedAt      int64  `json:"created_at"`
}

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	IsAdmin      bool   `json:"is_admin"`
	IsBanned     bool   `json:"is_banned"`
	CreatedAt    int64  `json:"created_at"`
}

type Thread struct {
	ID            int64  `json:"id"`
	SiteID        int64  `json:"site_id"`
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	Locked        bool   `json:"locked"`
	CommentCount  int    `json:"comment_count"`
	LastCommentAt int64  `json:"last_comment_at"`
	CreatedAt     int64  `json:"created_at"`
}

const (
	CommentStatusVisible = "visible"
	CommentStatusDeleted = "deleted"
	CommentStatusHidden  = "hidden"
)

type Comment struct {
	ID         int64          `json:"id"`
	ThreadID   int64          `json:"thread_id"`
	ParentID   *int64         `json:"parent_id"`
	UserID     *int64         `json:"user_id"`
	AuthorName string         `json:"author_name"`
	AvatarURL  string         `json:"avatar_url,omitempty"`
	Body       string         `json:"body"`
	Status     string         `json:"status"`
	Pinned     bool           `json:"pinned"`
	Score      int            `json:"score"`
	EditCount  int            `json:"edit_count"`
	Anchor     string         `json:"anchor,omitempty"`
	CreatedAt  int64          `json:"created_at"`
	UpdatedAt  int64          `json:"updated_at"`
	MyVote     int            `json:"my_vote"`
	Reactions  map[string]int `json:"reactions,omitempty"`
	MyReacts   []string       `json:"my_reacts,omitempty"`
	Replies    []*Comment     `json:"replies,omitempty"`
}

type Emoji struct {
	ID        int64  `json:"id"`
	SiteID    int64  `json:"site_id"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	SVG       string `json:"svg"`
	Sort      int    `json:"sort"`
	CreatedAt int64  `json:"created_at"`
}

type AuditEntry struct {
	ID         int64  `json:"id"`
	ActorID    *int64 `json:"actor_id"`
	ActorName  string `json:"actor_name"`
	Action     string `json:"action"`
	TargetKind string `json:"target_kind"`
	TargetID   int64  `json:"target_id"`
	Metadata   string `json:"metadata"`
	CreatedAt  int64  `json:"created_at"`
}
