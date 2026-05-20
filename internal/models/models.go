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
	CommentStatusPending = "pending"
)

type Comment struct {
	ID               int64          `json:"id"`
	ThreadID         int64          `json:"thread_id"`
	ParentID         *int64         `json:"parent_id"`
	UserID           *int64         `json:"user_id"`
	AuthorName       string         `json:"author_name"`
	AvatarURL        string         `json:"avatar_url,omitempty"`
	Body             string         `json:"body"`
	Status           string         `json:"status"`
	Pinned           bool           `json:"pinned"`
	Score            int            `json:"score"`
	EditCount        int            `json:"edit_count"`
	Anchor           string         `json:"anchor,omitempty"`
	RangeQuote       string         `json:"range_quote,omitempty"`
	RangePrefix      string         `json:"range_prefix,omitempty"`
	RangeSuffix      string         `json:"range_suffix,omitempty"`
	RangeStart       int            `json:"range_start,omitempty"`
	RangeEnd         int            `json:"range_end,omitempty"`
	ModerationReason string         `json:"moderation_reason,omitempty"`
	CreatedAt        int64          `json:"created_at"`
	UpdatedAt        int64          `json:"updated_at"`
	MyVote           int            `json:"my_vote"`
	Reactions        map[string]int `json:"reactions,omitempty"`
	MyReacts         []string       `json:"my_reacts,omitempty"`
	Replies          []*Comment     `json:"replies,omitempty"`
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

// ModerationSettings is a site's flexible moderation policy. A missing row
// reads as zero values which match the "open" defaults — every site behaves
// as if it has settings even before an admin saves any.
type ModerationSettings struct {
	SiteID               int64  `json:"site_id"`
	Mode                 string `json:"mode"`                    // open | pre_moderation
	HoldNewUsers         int    `json:"hold_new_users"`          // first N comments from each registered user are held
	MinAccountAgeSeconds int64  `json:"min_account_age_seconds"` // hold posts from accounts younger than this
	MinBodyLength        int    `json:"min_body_length"`         // reject anything shorter
	MaxLinks             int    `json:"max_links"`               // -1 = unlimited; otherwise hold above this
	LinkPolicy           string `json:"link_policy"`             // allow | hold | reject when any link present
	AnonymousLinkPolicy  string `json:"anonymous_link_policy"`   // as above, for anonymous posters only
	AutoHideFlagCount    int    `json:"auto_hide_flag_count"`    // 0 = off; otherwise hide once N distinct flags land
	UpdatedAt            int64  `json:"updated_at"`
}

// DefaultModerationSettings is the policy applied when no row exists yet:
// preserves the pre-feature open behaviour exactly.
func DefaultModerationSettings(siteID int64) ModerationSettings {
	return ModerationSettings{
		SiteID:              siteID,
		Mode:                "open",
		MaxLinks:            -1,
		LinkPolicy:          "allow",
		AnonymousLinkPolicy: "allow",
	}
}

// BlocklistEntry is a single keyword/regex rule applied to comment bodies.
type BlocklistEntry struct {
	ID        int64  `json:"id"`
	SiteID    int64  `json:"site_id"`
	Kind      string `json:"kind"`   // keyword | regex
	Pattern   string `json:"pattern"`
	Action    string `json:"action"` // hold | reject
	CreatedAt int64  `json:"created_at"`
}

// Mention is an inbound federated reference to a thread — either a W3C
// Webmention or a minimal ActivityPub Create(Note). Rendered alongside the
// thread's native comments once verified.
type Mention struct {
	ID         int64  `json:"id"`
	SiteID     int64  `json:"site_id"`
	ThreadID   int64  `json:"thread_id"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	AuthorName string `json:"author_name,omitempty"`
	AuthorURL  string `json:"author_url,omitempty"`
	Snippet    string `json:"snippet,omitempty"`
	Kind       string `json:"kind"`   // "webmention" | "activitypub"
	Status     string `json:"status"` // "pending" | "verified" | "rejected"
	Reason     string `json:"reason,omitempty"`
	CreatedAt  int64  `json:"created_at"`
	VerifiedAt int64  `json:"verified_at,omitempty"`
}

const (
	MentionKindWebmention   = "webmention"
	MentionKindActivityPub  = "activitypub"
	MentionStatusPending    = "pending"
	MentionStatusVerified   = "verified"
	MentionStatusRejected   = "rejected"
)

// OAuthIdentity links an external provider account to a local user. Created
// on first OAuth login; consulted on every subsequent OAuth login to look up
// the existing user. The pair (Provider, Subject) is globally unique.
type OAuthIdentity struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Provider  string `json:"provider"`
	Subject   string `json:"subject"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

// UserProfile is the editable public profile (bio, website, avatar override).
// Distinct from the User record so OAuth-imported avatars and user-supplied
// bios live in one place without bloating the hot users row.
type UserProfile struct {
	UserID    int64  `json:"user_id"`
	Bio       string `json:"bio,omitempty"`
	Website   string `json:"website,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
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
