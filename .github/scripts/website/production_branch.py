"""Read an existing Pages project's production branch without changing it."""

import json
import os
import re
import sys
import urllib.error
import urllib.request


def production_branch(payload):
    if not isinstance(payload, dict) or payload.get("success") is not True:
        raise ValueError("Cloudflare did not return a successful project lookup")
    result = payload.get("result")
    branch = result.get("production_branch") if isinstance(result, dict) else None
    if (
        not isinstance(branch, str)
        or not branch
        or branch != branch.strip()
        or len(branch.encode("utf-8")) > 255
        or any(ord(character) < 32 or ord(character) == 127 for character in branch)
    ):
        raise ValueError("Cloudflare returned an invalid production branch")
    return branch


def lookup(environ, opener=urllib.request.urlopen):
    account = environ.get("CLOUDFLARE_ACCOUNT_ID", "")
    project = environ.get("PAGES_PROJECT", "")
    token = environ.get("CLOUDFLARE_API_TOKEN", "")
    if not re.fullmatch(r"[a-fA-F0-9]{32}", account):
        raise ValueError("CLOUDFLARE_ACCOUNT_ID is missing or invalid")
    if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,62}", project):
        raise ValueError("PAGES_PROJECT is missing or invalid")
    if not token or "\r" in token or "\n" in token:
        raise ValueError("CLOUDFLARE_API_TOKEN is missing or invalid")
    request = urllib.request.Request(
        f"https://api.cloudflare.com/client/v4/accounts/{account}/pages/projects/{project}",
        headers={"Authorization": f"Bearer {token}", "Accept": "application/json"},
        method="GET",
    )
    try:
        with opener(request, timeout=30) as response:
            payload = json.loads(response.read(1024 * 1024))
    except urllib.error.HTTPError as error:
        code = error.code
        error.close()
        raise ValueError(f"Cloudflare project lookup failed (HTTP {code})") from None
    except (urllib.error.URLError, OSError, json.JSONDecodeError):
        raise ValueError("Cloudflare project lookup failed; check connectivity and token scope") from None
    return production_branch(payload)


def main():
    try:
        branch = lookup(os.environ)
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            output.write(f"production_branch={branch}\n")
        print(f"Using the Pages project's existing production branch: {branch}")
    except (ValueError, OSError, KeyError) as error:
        print(f"Pages deployment blocked: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
