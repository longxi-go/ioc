"""Exercise nested-adapter publication independently of root Latest and Pages."""

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import publish_release
import release

REPOSITORY = "longxi-go/ioc"
MODULE = "github.com/longxi-go/ioc/adaptors/gin"
TAG = "adaptors/gin/v0.2.0"
COMMIT = "c" * 40


class AdapterReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.adapter = self.root / "adaptors/gin"
        self.adapter.mkdir(parents=True)
        self.module_file = self.adapter / "go.mod"
        self.module_file.write_text(
            f"module {MODULE}\n\ngo 1.27.0\n\nrequire github.com/longxi-go/ioc v0.2.0\n"
        )
        notes = self.adapter / "docs/releases/v0.2.0.md"
        notes.parent.mkdir(parents=True)
        notes.write_text("# ioc Gin adapter v0.2.0\n\nTyped Gin route generation.\n")

    def test_adapter_tag_preserves_nested_notes_without_claiming_latest(self):
        self.assertEqual("0.2.0", release.validate_tag(self.root, TAG, module="gin"))
        payload = release.release_payload(self.root, TAG, module="gin")
        self.assertEqual("ioc Gin adapter v0.2.0", payload["name"])
        self.assertEqual("false", payload["make_latest"])
        self.assertEqual(TAG, payload["tag_name"])
        self.assertIn("Typed Gin route generation.", payload["body"])

    def test_adapter_requires_correct_identity_stable_root_and_no_replacements(self):
        original = self.module_file.read_text()
        invalid = (
            original.replace(MODULE, "example.org/other"),
            original.replace("ioc v0.2.0", "ioc v0.0.0"),
            original.replace("ioc v0.2.0", "ioc v0.2.0-rc.1"),
            original + "\nreplace github.com/longxi-go/ioc => ../..\n",
        )
        for content in invalid:
            with self.subTest(content=content):
                self.module_file.write_text(content)
                with self.assertRaises(ValueError):
                    release.validate_tag(self.root, TAG, module="gin")
        self.module_file.write_text(original)
        for tag in ("v0.2.0", "adaptors/echo/v0.2.0", "adaptors/gin/v2.0.0"):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                release.validate_tag(self.root, tag, module="gin")

    def test_adapter_preflight_requires_a_published_stable_root(self):
        for draft in (False, True):
            response = {"tag_name": "v0.2.0", "draft": draft, "prerelease": False}
            with self.subTest(draft=draft), patch.object(
                release, "github_json", return_value=response
            ) as api:
                if draft:
                    with self.assertRaises(ValueError):
                        release.check_dependencies(self.root, module="gin")
                else:
                    release.check_dependencies(self.root, module="gin")
                api.assert_called_once_with(f"repos/{REPOSITORY}/releases/tags/v0.2.0")

    def test_adapter_publish_never_reads_or_promotes_root_latest(self):
        payload = release.release_payload(self.root, TAG, module="gin")
        published = {k: v for k, v in payload.items() if k != "make_latest"}
        calls = []

        def respond(path, *, payload=None):
            calls.append((path, payload))
            if "/commits/" in path:
                return {"sha": COMMIT}
            if "/releases/tags/" in path:
                raise publish_release.GitHubError(404, "absent")
            if path.endswith("/releases") and payload is not None:
                self.assertEqual("false", payload["make_latest"])
                return published
            self.fail(f"Unexpected adapter request: {path}")

        with patch.object(publish_release, "github_api", side_effect=respond), patch.object(
            publish_release.time, "sleep"
        ):
            self.assertFalse(publish_release.publish(REPOSITORY, payload, COMMIT))
        self.assertEqual(1, sum(value is not None for _, value in calls))

    def test_adapter_cli_emits_maintained_payload(self):
        output = self.root / "release.json"
        result = subprocess.run(
            [sys.executable, release.__file__, "--root", str(self.root), "--module", "gin",
             "--tag", TAG, "--github-payload", str(output)],
            capture_output=True, text=True, check=False,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual("false", json.loads(output.read_text())["make_latest"])


if __name__ == "__main__":
    unittest.main()
