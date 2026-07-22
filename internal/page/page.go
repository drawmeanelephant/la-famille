package page

import (
	"html/template"

	"github.com/tbuddy/la-famille/internal/config"
)

type Page struct {
	Site            config.Config
	Content         template.HTML
	Title           string
	Author          string
	Date            string
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	ComplianceModal string
	Description     string
	Image           string
	CanonicalURL    string
}
