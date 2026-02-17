package models

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// ID prefixes for each entity type
const (
	PrefixInitiative   = "i_"
	PrefixEpic         = "e_"
	PrefixStory        = "s_"
	PrefixDoc          = "d_"
	PrefixDecision     = "dec_"
	PrefixPlan         = "p_"
	PrefixExecution    = "x_"
	PrefixVerification = "v_"
)

// NewID generates a new prefixed ULID.
func NewID(prefix string) string {
	id := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
	return prefix + id.String()
}
