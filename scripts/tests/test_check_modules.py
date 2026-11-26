"""Validate nested checkout modules without changing publishable metadata."""

import os
from pathlib import Path
import sys
import tempfile
import unittest

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
        from unittest.mock import patch
        with patch.dict(os.environ, {"GOFLAGS": "-mod=mod"}):
            with self.assertRaisesRegex(ValueError, "GOFLAGS"):
                check_modules.check(self.root, "verify", paths=("adaptors/gin",))


if __name__ == "__main__":
    unittest.main()
