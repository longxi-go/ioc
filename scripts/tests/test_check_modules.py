"""Validate nested checkout modules without changing publishable metadata."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import check_modules


class CheckoutModulesTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="ioc modules ")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        (self.root / "go.mod").write_text("module example.org/project\n\ngo 1.27.0\n")
        (self.root / "root.go").write_text("package project\n\nconst Value = 42\n")
        self.nested = self.root / "adaptors/gin"
        self.nested.mkdir(parents=True)
        (self.nested / "go.mod").write_text(
            "module example.org/project/adaptors/gin\n\ngo 1.27.0\n\n"
            "require example.org/project v0.2.0\n"
        )
        (self.nested / "value.go").write_text(
            'package gin\n\nimport "example.org/project"\n\nvar Value = project.Value\n'
        )

    def test_unpublished_checkout_dependency_verifies_without_source_changes(self):
        before = {p.relative_to(self.root): p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        check_modules.check(self.root, "verify", paths=("adaptors/gin",))
        after = {p.relative_to(self.root): p.read_bytes() for p in self.root.rglob("*") if p.is_file()}
        self.assertEqual(before, after)

    def test_stale_nested_requirements_fail_without_being_rewritten(self):
        metadata = self.nested / "go.mod"
        original = metadata.read_text() + "\nrequire example.org/unused v0.2.0\n"
        metadata.write_text(original)
        with self.assertRaisesRegex(ValueError, "tidy"):
            check_modules.check(self.root, "verify", paths=("adaptors/gin",))
        self.assertEqual(original, metadata.read_text())

    def test_hidden_module_flags_are_rejected_before_copying(self):
        with patch.dict(os.environ, {"GOFLAGS": "-mod=mod"}):
            with self.assertRaisesRegex(ValueError, "GOFLAGS"):
                check_modules.check(self.root, "verify", paths=("adaptors/gin",))

    def test_reordered_external_checksums_fail_without_being_rewritten(self):
        # A real local proxy keeps Go's checksum generation and tidy behavior offline.
        proxy = Path(self.temporary.name) / "proxy"
        versions = proxy / "example.org/external/@v"
        versions.mkdir(parents=True)
        module = "module example.org/external\n\ngo 1.27.0\n"
        (versions / "v1.0.0.mod").write_text(module)
        (versions / "v1.0.0.info").write_text(
            '{"Version":"v1.0.0","Time":"2026-01-01T00:00:00Z"}\n'
        )
        (versions / "list").write_text("v1.0.0\n")
        with zipfile.ZipFile(versions / "v1.0.0.zip", "w") as archive:
            archive.writestr("example.org/external@v1.0.0/go.mod", module)
            archive.writestr("example.org/external@v1.0.0/value.go",
                             "package external\n\nconst Value = 7\n")
        (self.nested / "go.mod").write_text(
            "module example.org/project/adaptors/gin\n\ngo 1.27.0\n\n"
            "require (\n\texample.org/external v1.0.0\n\texample.org/project v0.2.0\n)\n"
        )
        (self.nested / "external.go").write_text(
            'package gin\n\nimport "example.org/external"\n\nvar External = external.Value\n'
        )
        with patch.dict(os.environ, {
            "GOPROXY": proxy.as_uri(), "GOSUMDB": "off",
            "GOMODCACHE": str(Path(self.temporary.name) / "cache"),
        }):
            env = check_modules.verification_environment()
            subprocess.run([env.get("GO", "go"), "mod", "download", "example.org/external"],
                           cwd=self.nested, env=env, check=True, capture_output=True, timeout=30)
            checksums = self.nested / "go.sum"
            canonical = checksums.read_text().splitlines()
            self.assertEqual(len(canonical), 2)
            check_modules.check(self.root, "verify", paths=("adaptors/gin",))
            reordered = "\n".join(reversed(canonical)) + "\n"
            checksums.write_text(reordered)
            with self.assertRaisesRegex(ValueError, "tidy"):
                check_modules.check(self.root, "verify", paths=("adaptors/gin",))
            self.assertEqual(checksums.read_text(), reordered)


if __name__ == "__main__":
    unittest.main()
