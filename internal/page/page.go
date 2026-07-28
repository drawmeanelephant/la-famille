package page

import (
	"html/template"

	"github.com/tbuddy/la-famille/internal/config"
)

type Page struct {
	AnimationCues   string
	Content         template.HTML
	Title           string
	Author          string
	Date            string
	VideoScript     string
	SoundtrackTheme string
	Layout          string
	ComplianceModal string
	Description     string
	Image           string
	CanonicalURL    string
	Site            config.Config
}
