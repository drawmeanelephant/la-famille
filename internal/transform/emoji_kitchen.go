package transform

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type EmojiKitchenParser struct{}

func (p *EmojiKitchenParser) Trigger() []byte {
	return []byte{'!'}
}

func (p *EmojiKitchenParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	re := regexp.MustCompile(`^!ek\[([^\+\]]+)\+([^\]]+)\]`)
	match := re.FindSubmatchIndex(line)
	if match == nil {
		return nil
	}

	leftStr := strings.TrimSpace(string(line[match[2]:match[3]]))
	rightStr := strings.TrimSpace(string(line[match[4]:match[5]]))

	leftRunes := []rune(leftStr)
	rightRunes := []rune(rightStr)

	if len(leftRunes) == 0 || len(rightRunes) == 0 {
		return nil
	}

	left := leftRunes[0]
	right := rightRunes[0]

	leftHex := fmt.Sprintf("u%x", left)
	rightHex := fmt.Sprintf("u%x", right)

	url := fmt.Sprintf("https://www.gstatic.com/android/keyboard/emojikitchen/20201001/%s/%s_%s.png", leftHex, leftHex, rightHex)

	img := ast.NewImage(ast.NewLink())
	img.Destination = []byte(url)
	img.Title = []byte(fmt.Sprintf("Emoji Kitchen combination of %s and %s", string(left), string(right)))

	altText := fmt.Sprintf("Emoji Kitchen combination of %s and %s", string(left), string(right))
	img.AppendChild(img, ast.NewString([]byte(altText)))

	block.Advance(match[1])

	return img
}
