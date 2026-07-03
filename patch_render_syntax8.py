# The difference in the diff from CI:
# Expected:
# <body class="min-h-screen bg-base-200 prose-a:focus-visible:outline prose-a:focus-visible:outline-2 prose-a:focus-visible:outline-primary">
#         <nav aria-label="Main Navigation" class="fixed top-4 right-4 z-50">
#           <div class="dropdown dropdown-end">
#
# Actual:
# <body class="min-h-screen bg-base-200 prose-a:focus-visible:outline prose-a:focus-visible:outline-2 prose-a:focus-visible:outline-primary">
#
#         <nav aria-label="Main Navigation" class="fixed top-4 right-4 z-50">
#           <div class="dropdown dropdown-end">
#
# It's an extra blank line!
# `Actual` has a blank line between `<body>` and `<nav>`.
# `Expected` does not.
# WHY?
# Let's check `templates/layout.html`
