package page

import (
	"html/template"

	"github.com/tbuddy/la-famille/internal/config"
)

type Page struct {
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
	Content         template.HTML
	Site            config.Config
}
