//go:build notmuch
// +build notmuch

package notmuch

import "git.sr.ht/~rjarry/aerc/models"

var tagToFlag = map[string]models.Flags{
	"unread":  models.SeenFlag,
	"replied": models.AnsweredFlag,
	"draft":   models.DraftFlag,
	"flagged": models.FlaggedFlag,
}

var flagToTag = map[models.Flags]string{
	models.SeenFlag:     "unread",
	models.AnsweredFlag: "replied",
	models.DraftFlag:    "draft",
	models.FlaggedFlag:  "flagged",
}

var flagToInvert = map[models.Flags]bool{
	models.SeenFlag:     true,
	models.AnsweredFlag: false,
	models.DraftFlag:    false,
	models.FlaggedFlag:  false,
}
