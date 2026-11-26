"""Verify nested modules against checkout source in a disposable copy."""

import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

from go_environment import verification_environment
from go_module import read_module

MODULE_DIRECTORIES = ("adaptors/gin", "examples/gin", "examples/coffee")
GENERATED_MODULE_DIRECTORIES = ("examples/gin", "examples/coffee")
COPY_EXCLUSIONS = (".git", ".tmp", ".codex", ".superpowers", ".venv", "__pycache__",
                   "node_modules", "site", "bin", "coverage.out")


def run(command: list[str], directory: Path, env: dict) -> None:
    """Preserve the failing command and its diagnostics for CI."""
    print(f"[{directory.name}] {' '.join(command)}", flush=True)
    result = subprocess.run(command, cwd=directory, env=env, check=False)
    if result.returncode:
        raise ValueError(f"Nested module command failed ({result.returncode}): {' '.join(command)}")


def metadata_without_replacements(path: Path) -> dict:
    metadata = read_module(path)
    metadata.pop("Replace", None)
    return metadata


def external_checksums(path: Path, modules: set[str]) -> set[str]:
    if not path.exists():
        return set()
    return {line for line in path.read_text().splitlines()
            if line.strip() and line.split()[0] not in modules}


def check(root: Path, mode: str, *, paths: tuple[str, ...] = MODULE_DIRECTORIES) -> None:
    """Use local replacements only in the copy; original modules remain immutable."""
    env = verification_environment()
    root = root.resolve()
    go = env.get("GO", "go")
    selected = ("",) + paths
    names = {}
    for relative in selected:
        metadata = read_module(root / relative / "go.mod")
        if metadata.get("Replace"):
            raise ValueError(f"Tracked module {relative or '.'} must not use replacements")
        names[relative] = metadata["Module"]["Path"]
    with tempfile.TemporaryDirectory(prefix="ioc-check-modules-") as temporary:
        source = Path(temporary) / "source"
        shutil.copytree(root, source, ignore=shutil.ignore_patterns(*COPY_EXCLUSIONS), symlinks=True)
        for relative in selected:
            directory = source / relative
            for target, name in names.items():
                if target != relative:
                    run([go, "mod", "edit", f"-replace={name}={source / target}"], directory, env)
        for relative in paths:
            directory = source / relative
            before = metadata_without_replacements(directory / "go.mod")
            checksums = external_checksums(directory / "go.sum", set(names.values()))
            run([go, "mod", "tidy"], directory, env)
            if metadata_without_replacements(directory / "go.mod") != before:
                raise ValueError(f"{relative}/go.mod is not tidy against checkout dependencies")
            if external_checksums(directory / "go.sum", set(names.values())) != checksums:
                raise ValueError(f"{relative}/go.sum is not tidy against checkout dependencies")
            run([go, "mod", "verify"], directory, env)
            if mode == "verify":
                continue
            run([go, "vet", "./..."], directory, env)
            run([go, "test", "-cover", "./..."], directory, env)
            if mode == "quality":
                run([go, "test", "-race", "-shuffle=on", "./..."], directory, env)
                linter = env.get("GOLANGCI_LINT", "golangci-lint")
                run([linter, "run", "--config", str(source / ".golangci.yml"), "--timeout=10m"], directory, env)
                if relative in GENERATED_MODULE_DIRECTORIES:
                    run(["make", "check-generated"], directory, env)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--mode", choices=("verify", "compatibility", "quality"), default="quality")
    args = parser.parse_args()
    try:
        check(args.root, args.mode)
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        parser.error(str(error))


if __name__ == "__main__":
    main()
