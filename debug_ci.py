# The CI failed again.
# "fixture_test.go:75: content mismatch in pages/index.html:"
# Let's look at the failure:
# Expected: ... <body class="min-h-screen bg-base-200 prose-a:focus-visible:outline prose-a:focus-visible:outline-2 prose-a:focus-visible:outline-primary"> ... <a href="#menu" class="btn btn-square btn-ghost focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"> ...
# Actual: ... <body class="min-h-screen bg-base-200 prose-a:focus-visible:outline prose-a:focus-visible:outline-2 prose-a:focus-visible:outline-primary"> ... <nav aria-label="Main Navigation" class="fixed top-4 right-4 z-50"> ... <div class="toast toast-bottom toast-center z-50 w-full sm:w-auto px-4"> ...

# Wait! The ACTUAL has my `<nav aria-label="Main Navigation" class="fixed top-4 right-4 z-50">` and `<div class="toast ...">`
# But the EXPECTED does NOT have it?
# Wait, look at Expected:
# `<body class="min-h-screen bg-base-200 prose-a:focus-visible:outline prose-a:focus-visible:outline-2 prose-a:focus-visible:outline-primary">`
# `        <nav aria-label="Main Navigation" class="fixed top-4 right-4 z-50">`
# Expected HAS it!
# Wait. What's the difference between Expected and Actual? Let's look closely at the git output I got from `go test ./... && go vet ./...`.
