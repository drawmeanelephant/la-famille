package page

import (
	"html/template"

	"github.com/tbuddy/la-famille/internal/config"
)

type Page struct {
	Site            config.Config
	Title           string
	Author          string
	Date            string
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	Content         template.HTML
	Description     string
	Image           string
}
