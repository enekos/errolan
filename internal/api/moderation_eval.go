package api

import (
	"time"

	"github.com/enekos/errolan/internal/models"
	"github.com/enekos/errolan/internal/moderation"
)

// evaluateComment runs the site's moderation policy against a body the user is
// about to post (create or edit). Admins are always allowed — pre-moderation
// sites shouldn't force the owner to approve themselves.
//
// The function reads site policy + blocklist + author state from the store.
// Errors loading those fall back to "Allow" rather than blocking the request,
// matching the engine's defensive defaults.
func (s *Server) evaluateComment(siteID int64, user *models.User, anonymous bool, body string) moderation.Decision {
	if user != nil && user.IsAdmin {
		return moderation.Decision{Action: moderation.ActionAllow}
	}
	settings, _ := s.Store.ModerationSettings(siteID)
	blocklist, _ := s.Store.ListBlocklist(siteID)
	rules := moderation.CompileRules(blocklist)

	in := moderation.Input{Body: body, Anonymous: anonymous}
	if user != nil {
		in.AuthorAccountAgeSec = time.Now().Unix() - user.CreatedAt
		if n, err := s.Store.CountUserComments(user.ID); err == nil {
			in.AuthorCommentCount = n
		}
	}
	return moderation.Evaluate(*settings, rules, in)
}
