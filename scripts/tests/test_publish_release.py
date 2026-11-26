import json
import os
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import publish_release

REPOSITORY = "longxi-go/ioc"
TAG = "v0.1.0"
COMMIT = "a" * 40
PREFIX = f"repos/{REPOSITORY}"


class PublishReleaseTests(unittest.TestCase):
    def setUp(self):
        self.payload = {
            "tag_name": TAG, "name": f"ioc {TAG}", "body": "Maintained notes.\n",
            "draft": False, "prerelease": False, "make_latest": "true",
        }
        self.published = {key: value for key, value in self.payload.items() if key != "make_latest"}

    def test_matching_release_is_reused_without_writes(self):
        with patch.object(publish_release, "github_api", side_effect=[
            {"sha": COMMIT}, self.published, {"tag_name": TAG},
        ]) as api:
            self.assertTrue(publish_release.publish(REPOSITORY, self.payload, COMMIT))
        self.assertFalse(any(call.kwargs.get("payload") for call in api.call_args_list))

    def test_conflicting_release_and_moved_tag_fail_without_writes(self):
        for field, value in (("body", "Other"), ("draft", True), ("name", "Other"), ("prerelease", True)):
            conflict = dict(self.published, **{field: value})
            with self.subTest(field=field), patch.object(publish_release, "github_api", side_effect=[
                {"sha": COMMIT}, conflict,
            ]) as api:
                with self.assertRaisesRegex(ValueError, "does not match"):
                    publish_release.publish(REPOSITORY, self.payload, COMMIT)
                self.assertFalse(any(call.kwargs.get("payload") for call in api.call_args_list))
        with patch.object(publish_release, "github_api", return_value={"sha": "b" * 40}) as api:
            with self.assertRaisesRegex(ValueError, "commit"):
                publish_release.publish(REPOSITORY, self.payload, COMMIT)
        self.assertEqual(1, api.call_count)

    def test_transient_missing_release_is_retried_before_any_create(self):
        with patch.object(publish_release, "github_api", side_effect=[
            {"sha": COMMIT}, publish_release.GitHubError(404, "propagating"),
            self.published, {"tag_name": TAG},
        ]) as api, patch.object(publish_release.time, "sleep"):
            self.assertTrue(publish_release.publish(REPOSITORY, self.payload, COMMIT))
        self.assertFalse(any(call.kwargs.get("payload") for call in api.call_args_list))

    def test_backfill_creation_does_not_replace_newer_latest(self):
        def respond(path, *, payload=None):
            if path.endswith("/commits/refs%2Ftags%2F" + TAG):
                return {"sha": COMMIT}
            if "/releases/tags/" in path:
                raise publish_release.GitHubError(404, "absent")
            if path.endswith("/latest"):
                return {"tag_name": "v0.2.0"}
            self.assertEqual("false", payload["make_latest"])
            return self.published

        with patch.object(publish_release, "github_api", side_effect=respond) as api, patch.object(
            publish_release.time, "sleep"
        ):
            self.assertFalse(publish_release.publish(REPOSITORY, self.payload, COMMIT))
        self.assertEqual(1, sum(bool(call.kwargs.get("payload")) for call in api.call_args_list))

    def test_ambiguous_create_failure_is_never_retried(self):
        writes = []

        def respond(path, *, payload=None):
            if "/commits/" in path:
                return {"sha": COMMIT}
            if payload is not None:
                writes.append(payload)
                raise publish_release.GitHubError(502, "uncertain write")
            raise publish_release.GitHubError(404, "absent")

        with patch.object(publish_release, "github_api", side_effect=respond), patch.object(
            publish_release.time, "sleep"
        ):
            with self.assertRaises(publish_release.GitHubError):
                publish_release.publish(REPOSITORY, self.payload, COMMIT)
        self.assertEqual([self.payload], writes)

    def test_latest_propagation_and_unavailability_remain_distinct_from_superseded(self):
        with patch.object(publish_release, "github_api", side_effect=[
            publish_release.GitHubError(404, "propagating"), {"tag_name": "v0.0.9"}, {"tag_name": TAG},
        ]), patch.object(publish_release.time, "sleep"):
            self.assertTrue(publish_release.wait_for_latest(REPOSITORY, TAG))
        with patch.object(publish_release, "github_api", return_value={"tag_name": "v0.2.0"}):
            self.assertFalse(publish_release.wait_for_latest(REPOSITORY, TAG))
        for result in (publish_release.GitHubError(404, "absent"), {"tag_name": "v0.0.9"}):
            with self.subTest(result=result), patch.object(
                publish_release, "github_api", side_effect=result if isinstance(result, Exception) else None,
                return_value=result,
            ) as api, patch.object(publish_release.time, "sleep") as sleep:
                with self.assertRaisesRegex(ValueError, "Latest"):
                    publish_release.wait_for_latest(REPOSITORY, TAG)
                self.assertGreater(api.call_count, 1)
                self.assertLessEqual(api.call_count, 10)
                self.assertEqual(api.call_count - 1, sleep.call_count)

    def test_permission_failure_never_creates_release_or_retries(self):
        with patch.object(publish_release, "github_api", side_effect=[
            {"sha": COMMIT}, publish_release.GitHubError(403, "denied"),
        ]) as api, patch.object(publish_release.time, "sleep") as sleep:
            with self.assertRaises(publish_release.GitHubError):
                publish_release.publish(REPOSITORY, self.payload, COMMIT)
            self.assertEqual(2, api.call_count)
            sleep.assert_not_called()

    def test_pages_refuses_content_conflict_or_superseded_release(self):
        with patch.object(publish_release, "github_api", return_value={**self.published, "body": "Other"}):
            with self.assertRaisesRegex(ValueError, "does not match"):
                publish_release.require_latest(REPOSITORY, self.payload)
        with patch.object(publish_release, "github_api", side_effect=[self.published, {"tag_name": "v0.2.0"}]):
            with self.assertRaisesRegex(ValueError, "not Latest"):
                publish_release.require_latest(REPOSITORY, self.payload)

    def test_uncertain_reads_cannot_turn_into_absence(self):
        with patch.object(publish_release, "github_api", side_effect=[
            publish_release.GitHubError(503, "unavailable"),
            publish_release.GitHubError(429, "rate limit"),
            publish_release.GitHubError(404, "not propagated"),
        ]), patch.object(publish_release.time, "sleep"):
            with self.assertRaises(publish_release.GitHubError):
                publish_release.github_read("releases/tags/" + TAG, missing_ok=True)

    def test_read_retries_transient_server_failure_but_not_transport_or_permission_failure(self):
        with patch.object(publish_release, "github_api", side_effect=[
            publish_release.GitHubError(503, "retry"), self.published,
        ]), patch.object(publish_release.time, "sleep"):
            self.assertEqual(self.published, publish_release.github_read("release"))
        for error in (ValueError("transport"), publish_release.GitHubError(401, "unauthenticated")):
            with self.subTest(error=error), patch.object(publish_release, "github_api", side_effect=error) as api:
                with self.assertRaises(ValueError):
                    publish_release.github_read("release", missing_ok=True)
                self.assertEqual(1, api.call_count)

    def test_pages_checks_the_commit_that_produced_the_artifact(self):
        with patch.object(publish_release, "github_api", return_value={"sha": "b" * 40}) as api:
            with self.assertRaisesRegex(ValueError, "built commit"):
                publish_release.require_latest(REPOSITORY, self.payload, COMMIT)
        self.assertEqual(1, api.call_count)
        with patch.object(publish_release, "github_api", side_effect=[
            {"sha": COMMIT}, self.published, {"tag_name": TAG},
        ]):
            publish_release.require_latest(REPOSITORY, self.payload, COMMIT)

    def test_malformed_latest_fails_instead_of_skipping_pages(self):
        for response in ({}, {"tag_name": None}, {"tag_name": "tools/v0.1.0"}):
            with self.subTest(response=response), patch.object(publish_release, "github_api", return_value=response):
                with self.assertRaises(ValueError):
                    publish_release.wait_for_latest(REPOSITORY, TAG)
        with self.assertRaises(ValueError):
            publish_release.stable_version(None)

    def test_transport_serializes_post_and_preserves_http_errors(self):
        completed = subprocess.CompletedProcess([], 0, 'HTTP/2.0 201 Created\n\n{"id":42}\n', "")
        with patch.object(publish_release.subprocess, "run", return_value=completed) as run:
            self.assertEqual({"id": 42}, publish_release.github_api("releases", payload=self.payload))
        self.assertEqual(self.payload, json.loads(run.call_args.kwargs["input"]))
        self.assertIn("POST", run.call_args.args[0])
        self.assertGreater(run.call_args.kwargs["timeout"], 0)
        for status in (404, 403, 502):
            completed = subprocess.CompletedProcess([], 1, f'HTTP/2.0 {status} Error\n\n{{}}', "")
            with self.subTest(status=status), patch.object(publish_release.subprocess, "run", return_value=completed):
                with self.assertRaises(publish_release.GitHubError) as raised:
                    publish_release.github_api("release")
                self.assertEqual(status, raised.exception.status)

    def test_malformed_transport_and_timeout_are_diagnostic(self):
        for response in ('HTTP/2.0 200 OK\n\nnot JSON', 'HTTP/2.0 200 OK\n\n[]', 'not HTTP'):
            completed = subprocess.CompletedProcess([], 0, response, "")
            with self.subTest(response=response), patch.object(publish_release.subprocess, "run", return_value=completed):
                with self.assertRaisesRegex(ValueError, "Malformed GitHub response"):
                    publish_release.github_api("release")
        with patch.object(publish_release.subprocess, "run", side_effect=subprocess.TimeoutExpired(["gh"], 20)):
            with self.assertRaisesRegex(ValueError, "GitHub request failed"):
                publish_release.github_api("release")

    def test_cli_rejects_bad_commit_before_external_commands(self):
        result = subprocess.run(
            [sys.executable, publish_release.__file__, "--tag", TAG, "--commit", "short"],
            env=dict(os.environ, PATH=""), capture_output=True, text=True, check=False,
        )
        self.assertEqual(2, result.returncode)
        self.assertIn("full checked source commit", result.stderr)
        self.assertEqual("", result.stdout)


if __name__ == "__main__":
    unittest.main()
