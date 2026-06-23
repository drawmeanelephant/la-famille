---
title: XSS Test
---
# XSS Test

This should not have a script tag:
<script>alert("hack");</script>

This should not have a javascript link:
[Evil Link](javascript:alert("hack"))
