import contextlib
import io
import json
import unittest
import urllib.error

from production_branch import lookup, production_branch


class ProductionBranchTests(unittest.TestCase):
    def test_uses_project_branch_without_guessing(self):
        for branch in ("main", "master", "production", "release/site"):
            with self.subTest(branch=branch):
                self.assertEqual(
                    production_branch({"success": True, "result": {"production_branch": branch}}),
                    branch,
                )

    def test_rejects_invalid_or_failed_responses(self):
        for payload in (
            None,
            {"success": False},
            {"success": True, "result": None},
            {"success": True, "result": {}},
            *[
                {"success": True, "result": {"production_branch": branch}}
                for branch in ("", " main", "main ", "main\nother=value", "main\r", "a\x00b", "x" * 256, 1)
            ],
        ):
            with self.subTest(payload=payload):
                with self.assertRaises(ValueError):
                    production_branch(payload)

    def test_lookup_is_get_only_and_does_not_log_credentials(self):
        environ = {
            "CLOUDFLARE_ACCOUNT_ID": "a" * 32,
            "PAGES_PROJECT": "la-famille-go",
            "CLOUDFLARE_API_TOKEN": "synthetic-test-token",
        }

        def opener(request, timeout):
            self.assertEqual(request.get_method(), "GET")
            self.assertEqual(timeout, 30)
            self.assertEqual(
                request.full_url,
                "https://api.cloudflare.com/client/v4/accounts/" + "a" * 32 + "/pages/projects/la-famille-go",
            )
            self.assertIsNone(request.data)
            return io.BytesIO(json.dumps({"success": True, "result": {"production_branch": "main"}}).encode())

        output = io.StringIO()
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(output):
            self.assertEqual(lookup(environ, opener), "main")
        self.assertEqual(output.getvalue(), "")

    def test_http_error_does_not_include_response_or_credentials(self):
        environ = {
            "CLOUDFLARE_ACCOUNT_ID": "a" * 32,
            "PAGES_PROJECT": "la-famille-go",
            "CLOUDFLARE_API_TOKEN": "synthetic-test-token",
        }

        response = io.BytesIO(b"synthetic response")

        def opener(request, timeout):
            raise urllib.error.HTTPError(
                request.full_url, 403, environ["CLOUDFLARE_API_TOKEN"], {}, response
            )

        with self.assertRaisesRegex(ValueError, r"^Cloudflare project lookup failed \(HTTP 403\)$"):
            lookup(environ, opener)
        self.assertTrue(response.closed)

    def test_missing_configuration_never_makes_a_request(self):
        def opener(request, timeout):
            self.fail("invalid configuration must not make a request")

        for environ in (
            {},
            {"CLOUDFLARE_ACCOUNT_ID": "not-an-account"},
            {"CLOUDFLARE_ACCOUNT_ID": "a" * 32, "PAGES_PROJECT": "../another-project"},
            {"CLOUDFLARE_ACCOUNT_ID": "a" * 32, "PAGES_PROJECT": "la-famille-go"},
        ):
            with self.subTest(environ=environ), self.assertRaises(ValueError):
                lookup(environ, opener)


if __name__ == "__main__":
    unittest.main()
