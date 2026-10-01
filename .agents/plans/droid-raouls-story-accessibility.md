# Preserve Raoul(s)'s story image descriptions

Task ID: `droid-raouls-story-accessibility`

## Finding and scope

Live verification after approved PR #606 found that the Markdown HTML
sanitizer removed three persona-image descriptions from the story page.
The homepage descriptions, all four original images, all three homepage
burgers, and the burger favicon were present.

Adjust the three content descriptions to wording accepted by the existing
sanitizer. Do not weaken sanitization or change original image files,
templates, burger placements, themes, or publishing settings.

## Delivery and validation

1. Extend the generated-site test to require all four image descriptions.
   Confirm it fails on the three missing attributes before changing content.
2. Reword only the three story-page descriptions, retaining their meaning.
3. Run formatting/lint, full tests, vet, shuffled two-run tests, and the
   generated-site publishing contract.
4. Publish this in-scope fix through a normal checked PR under the user's
   existing approval to open, merge, and deploy the mascot addition.
5. Verify both live addresses, descriptions, and original artwork hashes.

## Static asset pipeline compatibility

No pipeline change. Keep the original filenames, image bytes, sanitizer
policy, and all burger artwork unchanged.

## Validation evidence

- Extended full-site test failed first on the three missing persona `alt`
  attributes.
- Replacing the colon separators with commas preserves the descriptions
  through the existing sanitizer; the generated-site test now passes.
- Formatting, lint (zero issues), module hygiene, full tests, vet, shuffled
  two-run tests, and whitespace checks pass.
- Original images, homepage/template burger placements, sanitizer policy,
  and deployment workflows remain unchanged.
