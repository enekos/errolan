// Package moderation evaluates an incoming comment against a site's policy
// and returns an Allow / Hold / Reject decision plus a human-readable reason.
//
// The engine is intentionally pure: it takes the policy, the rule list, and a
// snapshot of the relevant author state, and returns a decision. All I/O lives
// at the call site. That makes it trivial to unit-test and lets the HTTP layer
// stay thin.
package moderation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/enekos/errolan/internal/models"
)

// Action is the engine's verdict for a single comment.
type Action string

const (
	ActionAllow  Action = "allow"
	ActionHold   Action = "hold"
	ActionReject Action = "reject"
)

// rank lets us pick the strongest verdict across multiple firing rules.
// reject > hold > allow.
func rank(a Action) int {
	switch a {
	case ActionReject:
		return 2
	case ActionHold:
		return 1
	default:
		return 0
	}
}

type Decision struct {
	Action Action
	Reason string
}

// Input is the per-comment context the engine needs to reach a verdict.
type Input struct {
	Body                string
	Anonymous           bool  // true when the poster is not a registered user
	AuthorAccountAgeSec int64 // 0 for anonymous; seconds since author registered otherwise
	AuthorCommentCount  int   // 0 for anonymous; lifetime comment count for the registered author
}

// Rule is a compiled blocklist entry. CompileRules turns the stored entries
// into Rule values so the engine doesn't re-parse regex per comment.
type Rule struct {
	Kind    string // keyword | regex
	Pattern string
	Action  Action
	re      *regexp.Regexp
}

// CompileRules normalises stored entries into engine-ready Rules. Invalid
// regexes are skipped (admins are warned at write time, so a stale broken
// rule should never silently match).
func CompileRules(entries []*models.BlocklistEntry) []Rule {
	out := make([]Rule, 0, len(entries))
	for _, e := range entries {
		action := Action(e.Action)
		if action != ActionHold && action != ActionReject {
			action = ActionHold
		}
		r := Rule{
			Kind:    e.Kind,
			Pattern: e.Pattern,
			Action:  action,
		}
		if e.Kind == "regex" {
			re, err := regexp.Compile("(?i)" + e.Pattern)
			if err != nil {
				continue
			}
			r.re = re
		}
		out = append(out, r)
	}
	return out
}

// linkPattern matches anything that looks like an external link. We're not
// trying to be RFC-3986 — the engine just needs a stable count for policy.
var linkPattern = regexp.MustCompile(`(?i)\bhttps?://[^\s<>]+`)

// CountLinks reports how many URL-shaped substrings appear in the body.
func CountLinks(body string) int {
	return len(linkPattern.FindAllStringIndex(body, -1))
}

// Evaluate returns the moderation decision for a single comment.
func Evaluate(settings models.ModerationSettings, rules []Rule, in Input) Decision {
	body := strings.TrimSpace(in.Body)

	// 1. Minimum body length is a reject — there's no point holding a single
	//    character for human review.
	if settings.MinBodyLength > 0 && len(body) < settings.MinBodyLength {
		return Decision{
			Action: ActionReject,
			Reason: fmt.Sprintf("comment too short (minimum %d characters)", settings.MinBodyLength),
		}
	}

	verdict := Decision{Action: ActionAllow}
	apply := func(action Action, reason string) {
		if rank(action) > rank(verdict.Action) {
			verdict = Decision{Action: action, Reason: reason}
		}
	}

	// 2. Pre-moderation flips every comment to hold by default.
	if settings.Mode == "pre_moderation" {
		apply(ActionHold, "site is in pre-moderation")
	}

	// 3. Link policy.
	linkCount := CountLinks(body)
	if linkCount > 0 {
		if in.Anonymous {
			if a := linkActionFromPolicy(settings.AnonymousLinkPolicy); a != ActionAllow {
				apply(a, "anonymous comments with links are restricted")
			}
		}
		if a := linkActionFromPolicy(settings.LinkPolicy); a != ActionAllow {
			apply(a, "comments with links are restricted")
		}
		if settings.MaxLinks >= 0 && linkCount > settings.MaxLinks {
			apply(ActionHold, fmt.Sprintf("too many links (%d, max %d)", linkCount, settings.MaxLinks))
		}
	}

	// 4. Blocklist (keyword + regex). Reject short-circuits since nothing
	//    after it can produce a weaker decision.
	bodyLower := strings.ToLower(body)
	for _, r := range rules {
		matched := false
		switch r.Kind {
		case "regex":
			if r.re != nil && r.re.MatchString(body) {
				matched = true
			}
		default: // keyword
			needle := strings.ToLower(r.Pattern)
			if needle != "" && strings.Contains(bodyLower, needle) {
				matched = true
			}
		}
		if matched {
			apply(r.Action, fmt.Sprintf("matched blocked %s %q", r.Kind, r.Pattern))
			if verdict.Action == ActionReject {
				return verdict
			}
		}
	}

	// 5. New-user holds only apply to registered authors. Anonymous posts are
	//    governed by the anonymous-link policy and pre-moderation.
	if !in.Anonymous {
		if settings.HoldNewUsers > 0 && in.AuthorCommentCount < settings.HoldNewUsers {
			apply(ActionHold, "new user — first comments are held for review")
		}
		if settings.MinAccountAgeSeconds > 0 && in.AuthorAccountAgeSec < settings.MinAccountAgeSeconds {
			apply(ActionHold, "new account — comment held for review")
		}
	}

	return verdict
}

func linkActionFromPolicy(p string) Action {
	switch p {
	case "hold":
		return ActionHold
	case "reject":
		return ActionReject
	default:
		return ActionAllow
	}
}

// ValidateRule is used by the HTTP handler before inserting a blocklist entry,
// so admins can't smuggle in a regex that never compiles.
func ValidateRule(kind, pattern, action string) error {
	if pattern == "" {
		return fmt.Errorf("pattern required")
	}
	if len(pattern) > 500 {
		return fmt.Errorf("pattern too long (max 500 chars)")
	}
	switch kind {
	case "keyword", "regex":
	default:
		return fmt.Errorf("kind must be 'keyword' or 'regex'")
	}
	switch action {
	case "hold", "reject":
	default:
		return fmt.Errorf("action must be 'hold' or 'reject'")
	}
	if kind == "regex" {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}
	return nil
}

// ValidateSettings checks an incoming policy payload before persisting. The
// engine itself is forgiving (unknown values fall back to safe defaults) but
// admins should get a useful error rather than a silent reset.
func ValidateSettings(m *models.ModerationSettings) error {
	switch m.Mode {
	case "", "open", "pre_moderation":
		if m.Mode == "" {
			m.Mode = "open"
		}
	default:
		return fmt.Errorf("mode must be 'open' or 'pre_moderation'")
	}
	if err := validateLinkPolicy("link_policy", &m.LinkPolicy); err != nil {
		return err
	}
	if err := validateLinkPolicy("anonymous_link_policy", &m.AnonymousLinkPolicy); err != nil {
		return err
	}
	if m.HoldNewUsers < 0 || m.HoldNewUsers > 1000 {
		return fmt.Errorf("hold_new_users must be between 0 and 1000")
	}
	if m.MinAccountAgeSeconds < 0 {
		return fmt.Errorf("min_account_age_seconds must be non-negative")
	}
	if m.MinBodyLength < 0 || m.MinBodyLength > 8000 {
		return fmt.Errorf("min_body_length must be between 0 and 8000")
	}
	if m.MaxLinks < -1 {
		return fmt.Errorf("max_links must be -1 (unlimited) or non-negative")
	}
	if m.AutoHideFlagCount < 0 || m.AutoHideFlagCount > 1000 {
		return fmt.Errorf("auto_hide_flag_count must be between 0 and 1000")
	}
	return nil
}

func validateLinkPolicy(field string, v *string) error {
	switch *v {
	case "", "allow", "hold", "reject":
		if *v == "" {
			*v = "allow"
		}
		return nil
	default:
		return fmt.Errorf("%s must be 'allow', 'hold', or 'reject'", field)
	}
}
