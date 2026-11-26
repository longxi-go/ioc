"""Publish or verify an immutable release and wait for GitHub Latest visibility."""

import argparse
import json
from pathlib import Path
import re
import subprocess
import time
from urllib.parse import quote

from release import REPOSITORY, STABLE_TAG, release_payload

API_TIMEOUT_SECONDS = 20
LATEST_ATTEMPTS = 6
LATEST_RETRY_SECONDS = 5
READ_ATTEMPTS = 3
READ_RETRY_SECONDS = 2


class GitHubError(ValueError):
    """Keep HTTP failures distinct from transport errors and malformed responses."""

    def __init__(self, status: int, detail: str):
        self.status = status
        super().__init__(f"GitHub HTTP {status}: {detail}")


def github_api(path: str, *, payload: dict | None = None) -> dict:
    """Use gh credentials without putting tokens in arguments or diagnostics."""
    command = ["gh", "api", "--include", "--method", "GET" if payload is None else "POST", path]
    if payload is not None:
        command.extend(["--input", "-"])
    try:
        result = subprocess.run(
            command, input=None if payload is None else json.dumps(payload),
            capture_output=True, text=True, check=False, timeout=API_TIMEOUT_SECONDS,
        )
    except (OSError, subprocess.SubprocessError) as error:
        raise ValueError(f"GitHub request failed for {path}: {error}") from error
    headers, separator, body = result.stdout.partition("\n\n")
    status = re.match(r"HTTP/\S+\s+(\d{3})\b", headers)
    if not separator or status is None:
        raise ValueError(f"Malformed GitHub response for {path}: {result.stderr.strip()}")
    code = int(status[1])
    if not 200 <= code < 300:
        raise GitHubError(code, f"request failed for {path}")
    if result.returncode:
        raise ValueError(f"GitHub request failed for {path}: {result.stderr.strip()}")
    try:
        data = json.loads(body)
        if not isinstance(data, dict):
            raise ValueError("expected a JSON object")
        return data
    except ValueError as error:
        raise ValueError(f"Malformed GitHub response for {path}: {error}") from error


def github_read(path: str, *, missing_ok: bool = False) -> dict | None:
    """Retry read propagation failures; only a repeated 404 may mean absence."""
    only_missing = True
    for attempt in range(READ_ATTEMPTS):
        try:
            return github_api(path)
        except GitHubError as error:
            if error.status not in (404, 429) and error.status < 500:
                raise
            only_missing = only_missing and error.status == 404
            if attempt + 1 == READ_ATTEMPTS:
                if missing_ok and only_missing:
                    return None
                raise
            time.sleep(READ_RETRY_SECONDS)
    raise AssertionError("unreachable")


def stable_version(tag: str) -> tuple[int, int, int]:
    match = STABLE_TAG.fullmatch(tag) if isinstance(tag, str) else None
    if match is None:
        raise ValueError(f"Expected a stable Latest tag, received {tag!r}")
    return tuple(int(part) for part in match.groups())


def wait_for_latest(repository: str, tag: str) -> bool:
    """Bound propagation waits without promoting an older release over Latest."""
    current = None
    for attempt in range(LATEST_ATTEMPTS):
        try:
            current = github_api(f"repos/{repository}/releases/latest").get("tag_name")
            if not isinstance(current, str):
                raise ValueError("Latest response is missing its tag_name")
            if current == tag:
                return True
            if stable_version(current) > stable_version(tag):
                return False
        except GitHubError as error:
            if error.status not in (404, 429) and error.status < 500:
                raise
            current = None
        if attempt + 1 < LATEST_ATTEMPTS:
            time.sleep(LATEST_RETRY_SECONDS)
    if current is None:
        raise ValueError("GitHub Latest remains unavailable after bounded retries")
    raise ValueError(f"GitHub Latest did not converge to {tag} after bounded retries; received {current}")


def publish(repository: str, payload: dict, commit: str) -> bool:
    """Reuse matching published content; never edit notes or retry uncertain writes."""
    tag = payload["tag_name"]
    prefix = f"repos/{repository}"
    tag_ref = quote(f"refs/tags/{tag}", safe="")
    if github_read(f"{prefix}/commits/{tag_ref}").get("sha") != commit:
        raise ValueError(f"Release tag {tag} no longer matches checked commit {commit}")
    published = github_read(f"{prefix}/releases/tags/{tag}", missing_ok=True)
    if published is None:
        creation = dict(payload)
        if payload["make_latest"] == "true":
            latest = github_read(f"{prefix}/releases/latest", missing_ok=True)
            if latest is not None and stable_version(latest.get("tag_name", "")) > stable_version(tag):
                creation["make_latest"] = "false"
        # A failed POST has an uncertain result. Recovery reruns the read-first flow.
        published = github_api(f"{prefix}/releases", payload=creation)
    verify_content(published, payload)
    if payload["make_latest"] == "false":
        return False
    return wait_for_latest(repository, tag)


def verify_content(published: dict, payload: dict) -> None:
    """Refuse to rewrite maintained notes or change existing release semantics."""
    for field in ("tag_name", "name", "body", "draft", "prerelease"):
        if published.get(field) != payload[field]:
            raise ValueError(f"Published release {payload['tag_name']} does not match maintained {field}")


def require_latest(repository: str, payload: dict, commit: str | None = None) -> None:
    """Refuse stale Pages builds and recheck the immutable source when provided."""
    tag = payload["tag_name"]
    stable_version(tag)
    prefix = f"repos/{repository}"
    if commit is not None:
        tag_ref = quote(f"refs/tags/{tag}", safe="")
        if github_read(f"{prefix}/commits/{tag_ref}").get("sha") != commit:
            raise ValueError(f"Pages tag {tag} no longer matches built commit {commit}")
    verify_content(github_read(f"{prefix}/releases/tags/{tag}"), payload)
    if not wait_for_latest(repository, tag):
        raise ValueError(f"Pages requires the current Latest release; {tag} is not Latest")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--repository", default=REPOSITORY)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--module", choices=("root", "gin"), default="root")
    parser.add_argument("--commit")
    parser.add_argument("--require-latest", action="store_true")
    args = parser.parse_args()
    try:
        if (not args.require_latest or args.commit is not None) and not re.fullmatch(r"[0-9a-f]{40}", args.commit or ""):
            raise ValueError("A full checked source commit SHA is required")
        payload = release_payload(args.root, args.tag, args.module)
        if args.require_latest:
            require_latest(args.repository, payload, args.commit)
            return
        latest = publish(args.repository, payload, args.commit)
        print(f"latest={str(latest).lower()}")
    except (ValueError, OSError) as error:
        parser.error(str(error))


if __name__ == "__main__":
    main()
