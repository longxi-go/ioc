"""Validate module releases and read-only GitHub publication prerequisites."""

import argparse
import fnmatch
import json
from pathlib import Path
import re
import subprocess
import time

from go_module import read_module

MODULE = "github.com/longxi-go/ioc"
REPOSITORY = MODULE.removeprefix("github.com/")
STABLE_TAG = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
READ_ATTEMPTS = 3
READ_RETRY_SECONDS = 2
DEPENDENCY_REPOSITORIES = {
    "github.com/longxi-go/gosr": ("longxi-go/gosr", ""),
    "github.com/longxi-go/gosr/tools": ("longxi-go/gosr", "tools/"),
    "github.com/longxi-go/golayout": ("longxi-go/golayout", ""),
    "github.com/longxi-go/nemo": ("longxi-go/nemo", ""),
}


def dependencies(root: Path) -> dict:
    """Reject workspace substitutes and require published-version candidates."""
    module = read_module(root / "go.mod")
    if (module.get("Module") or {}).get("Path") != MODULE:
        raise ValueError(f"Expected module {MODULE}")
    if module.get("Replace"):
        raise ValueError("Release dependencies must not use module replacements")
    versions = {item["Path"]: item["Version"] for item in module.get("Require") or []}
    for path in DEPENDENCY_REPOSITORIES:
        version = versions.get(path, "")
        if not STABLE_TAG.fullmatch(version) or version == "v0.0.0":
            raise ValueError(f"A stable published dependency version is required for {path}")
    return versions


def validate_tag(root: Path, tag: str) -> str:
    """Validate the version, module identity and maintained notes before tagging."""
    match = STABLE_TAG.fullmatch(tag)
    if match is None:
        raise ValueError("Release tags must use stable vX.Y.Z versions without leading zeroes")
    if int(match[1]) >= 2:
        raise ValueError("v2+ requires a versioned module path and a release-policy update")
    dependencies(root)
    notes = root / "docs/releases" / f"{tag}.md"
    if not notes.is_file():
        raise ValueError(f"Missing release notes: {notes}")
    lines = notes.read_text(encoding="utf-8").strip().splitlines()
    if not lines or lines[0] != f"# ioc {tag}" or not "\n".join(lines[1:]).strip():
        raise ValueError("Release notes require a matching ioc tag title and a body")
    return tag[1:]


def release_payload(root: Path, tag: str) -> dict:
    """Keep maintained notes and an explicit GitHub Latest-selection policy."""
    validate_tag(root, tag)
    return {
        "tag_name": tag,
        "name": f"ioc {tag}",
        "body": (root / "docs/releases" / f"{tag}.md").read_text(encoding="utf-8"),
        "draft": False,
        "prerelease": False,
        "make_latest": "true",
    }


def github_json(path: str) -> dict:
    """Retry transient reads while preserving auth, HTTP and parse failures."""
    for attempt in range(READ_ATTEMPTS):
        try:
            result = subprocess.run(
                ["gh", "api", path], capture_output=True, text=True, check=False, timeout=30
            )
            if result.returncode:
                detail = (result.stderr or result.stdout).strip()
                if attempt + 1 < READ_ATTEMPTS and _transient(detail):
                    time.sleep(READ_RETRY_SECONDS)
                    continue
                raise ValueError(detail or f"gh exited with status {result.returncode}")
            data = json.loads(result.stdout)
            if not isinstance(data, dict):
                raise ValueError("expected a GitHub JSON object")
            return data
        except OSError as error:
            if attempt + 1 < READ_ATTEMPTS and _transient(str(error)):
                time.sleep(READ_RETRY_SECONDS)
                continue
            raise ValueError(f"GitHub preflight failed for {path}: {error}") from error
        except (subprocess.SubprocessError, ValueError) as error:
            raise ValueError(f"GitHub preflight failed for {path}: {error}") from error
    raise AssertionError("unreachable")


def _transient(detail: str) -> bool:
    text = detail.lower()
    return any(value in text for value in ("eof", "timeout", "tls handshake", "connection reset", "temporarily unavailable"))


def check_pages(repository: str, tag: str) -> None:
    """Check repository settings that workflow YAML cannot configure."""
    if github_json(f"repos/{repository}/pages").get("build_type") != "workflow":
        raise ValueError("GitHub Pages must use the workflow build type")
    environment = github_json(f"repos/{repository}/environments/github-pages")
    if "deployment_branch_policy" not in environment:
        raise ValueError("Missing github-pages deployment policy metadata")
    policy = environment["deployment_branch_policy"]
    if policy is None:
        return
    if not isinstance(policy, dict) or not policy.get("custom_branch_policies"):
        raise ValueError("github-pages requires a tag-compatible deployment policy")
    rules = github_json(
        f"repos/{repository}/environments/github-pages/deployment-branch-policies?per_page=100"
    )
    entries = rules.get("branch_policies") or []
    if rules.get("total_count") != len(entries):
        raise ValueError("Cannot verify the complete github-pages deployment policy")
    if not any(
        item.get("type") == "tag" and fnmatch.fnmatchcase(tag, item.get("name", ""))
        for item in entries
    ):
        raise ValueError(f"github-pages environment does not allow tag {tag}; add a v* tag rule")


def check_dependencies(root: Path) -> None:
    """Check stable upstream Releases separately from Go's checksum verification."""
    versions = dependencies(root)
    for path, (repository, prefix) in DEPENDENCY_REPOSITORIES.items():
        tag = prefix + versions[path]
        data = github_json(f"repos/{repository}/releases/tags/{tag}")
        if data.get("tag_name") != tag or data.get("draft") is not False or data.get("prerelease") is not False:
            raise ValueError(f"Required stable Release {tag} is not published in {repository}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--tag", required=True)
    parser.add_argument("--check-github", action="store_true")
    parser.add_argument("--github-payload", type=Path)
    parser.add_argument("--repository", default=REPOSITORY)
    args = parser.parse_args()
    try:
        version = validate_tag(args.root, args.tag)
        if args.check_github:
            check_pages(args.repository, args.tag)
            check_dependencies(args.root)
        if args.github_payload:
            args.github_payload.write_text(json.dumps(release_payload(args.root, args.tag)) + "\n", encoding="utf-8")
        print(version)
    except (ValueError, OSError) as error:
        parser.error(str(error))


if __name__ == "__main__":
    main()
