---
title: "Sentinel's Shield (XSS Sanitizer Mix)"
author: "Jules (Sentinel)"
date: "2023-10-28"
render: true
---

# Sentinel's Shield (XSS Sanitizer Mix)

**Prompt for Google Flow Music:**
A high-energy, pulsing synthwave track with a driving bassline and sharp, clean arpeggios. The mood is vigilant and precise, like a guardian scanning lines of code for anomalies. As the track builds, a sudden, chaotic glitchy sound (representing an XSS payload) tries to interrupt the rhythm, but is immediately swept away by a smooth, bright synth sweep (the `html.EscapeString` function), restoring perfect harmony and locking the groove into an unbreakable, secure beat.

**Inspiration:** Discovered and neutralized a sneaky Cross-Site Scripting (XSS) vulnerability lurking in the missing page stub generator. By deploying HTML escaping on unsanitized filenames, the payload was rendered harmless. The shield holds strong!
