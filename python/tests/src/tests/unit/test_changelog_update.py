import os
import shutil
import subprocess
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parents[5]


def _load(name: str):
    # The releng scripts have no .py extension, so load them by path.
    loader = SourceFileLoader(name.replace("-", "_"), str(REPO_ROOT / "releng" / name))
    spec = spec_from_loader(loader.name, loader)
    assert spec is not None
    module = module_from_spec(spec)
    loader.exec_module(module)
    return module


changelog_update = _load("changelog-update")
changelog_consume = _load("changelog-consume")

CHANGELOG = """\
# CHANGELOG

<!-- towncrier release notes start -->

## [4.3.0-rc.2] 2 December 2026
[4.3.0-rc.2]: https://example.invalid/v4.3.0-rc.2

- Fix: second

## [4.3.0-rc.1] 1 December 2026
[4.3.0-rc.1]: https://example.invalid/v4.3.0-rc.1

- Fix: first

## [4.3.0] 3 December 2026

- Fix: ga

## [4.30.0-rc.1] 1 December 2026

- Fix: other version

## [4.2.0] 5 November 2026

- Fix: old
"""


def test_strip_sections_removes_only_matching_versions():
    out = changelog_update.strip_sections(CHANGELOG, "4.3.0")

    for gone in (
        "[4.3.0-rc.1]",
        "[4.3.0-rc.2]",
        "## [4.3.0] ",
        "Fix: first",
        "Fix: second",
        "Fix: ga",
    ):
        assert gone not in out

    # Other versions, including one that merely shares a prefix, are kept
    # intact, along with everything above the first section.
    assert "## [4.30.0-rc.1] 1 December 2026" in out
    assert "Fix: other version" in out
    assert "## [4.2.0] 5 November 2026" in out
    assert "Fix: old" in out
    assert out.startswith("# CHANGELOG\n\n<!-- towncrier release notes start -->\n\n")


def test_strip_sections_without_matches_is_a_noop():
    assert changelog_update.strip_sections(CHANGELOG, "9.9.9") == CHANGELOG


def test_section_has_entries():
    assert changelog_update.section_has_entries(CHANGELOG, "4.3.0")
    assert not changelog_update.section_has_entries(
        "## [1.0.0] today\n[1.0.0]: x\n\n## [0.9.0] x\n- a\n", "1.0.0"
    )
    assert not changelog_update.section_has_entries(CHANGELOG, "9.9.9")


def _write(tmp_path):
    path = tmp_path / "CHANGELOG.md"
    path.write_text(CHANGELOG, encoding="utf-8")
    return path


def test_update_keeps_rebuilt_changelog_on_success(tmp_path, monkeypatch):
    path = _write(tmp_path)

    def fake_towncrier(version):
        with open(path, "a", encoding="utf-8") as f:
            f.write(f"\n## [{version}] today\n\n- Fix: new\n")
        return 0

    monkeypatch.setattr(changelog_update, "run_towncrier", fake_towncrier)

    assert changelog_update.update(str(path), "4.3.0") == 0
    out = path.read_text(encoding="utf-8")
    assert "Fix: new" in out
    assert "Fix: first" not in out


def test_update_restores_changelog_when_towncrier_fails(tmp_path, monkeypatch):
    path = _write(tmp_path)
    monkeypatch.setattr(changelog_update, "run_towncrier", lambda version: 1)

    assert changelog_update.update(str(path), "4.3.0") == 1
    assert path.read_text(encoding="utf-8") == CHANGELOG


def test_update_restores_changelog_when_interrupted(tmp_path, monkeypatch):
    path = _write(tmp_path)

    def interrupted(version):
        raise KeyboardInterrupt

    monkeypatch.setattr(changelog_update, "run_towncrier", interrupted)

    with pytest.raises(KeyboardInterrupt):
        changelog_update.update(str(path), "4.3.0-rc.3")
    assert path.read_text(encoding="utf-8") == CHANGELOG


def test_update_rejects_bad_version_without_touching_changelog(tmp_path, monkeypatch):
    path = _write(tmp_path)

    def must_not_run(version):
        raise AssertionError("towncrier should not run")

    monkeypatch.setattr(changelog_update, "run_towncrier", must_not_run)

    assert changelog_update.update(str(path), "4.3") == 2
    assert path.read_text(encoding="utf-8") == CHANGELOG


@pytest.fixture
def project(tmp_path, monkeypatch):
    """A scratch project with the real towncrier config and template."""
    shutil.copy(REPO_ROOT / "towncrier.toml", tmp_path / "towncrier.toml")
    changes = tmp_path / "docs" / "changes"
    changes.mkdir(parents=True)
    for name in ("_template.md.jinja", "README.md"):
        shutil.copy(REPO_ROOT / "docs" / "changes" / name, changes / name)
    (tmp_path / "CHANGELOG.md").write_text(
        "# CHANGELOG\n\n<!-- towncrier release notes start -->\n\n## [4.2.0] 5 November 2026\n\n- Fix: old\n",
        encoding="utf-8",
    )
    monkeypatch.chdir(tmp_path)
    return tmp_path


def _snippets(project):
    return sorted(
        p.name
        for p in (project / "docs" / "changes").glob("*.md")
        if p.name != "README.md"
    )


def _headings(project):
    return [
        line
        for line in (project / "CHANGELOG.md").read_text(encoding="utf-8").splitlines()
        if line.startswith("## ")
    ]


def test_real_towncrier_rc_then_ga_flow(project):
    changes = project / "docs" / "changes"
    (changes / "100.fix.md").write_text("Fix one.\n")
    (changes / "+ci.skip.md").write_text("CI only.\n")

    assert changelog_update.update("CHANGELOG.md", "4.3.0-rc.1") == 0
    (changes / "101.feature.md").write_text("Feature two.\n")
    assert changelog_update.update("CHANGELOG.md", "4.3.0-rc.2") == 0

    # Only the newest RC section remains, snippets are all still there, and
    # skip snippets never show up.
    assert [h.split("]")[0] for h in _headings(project)] == [
        "## [4.3.0-rc.2",
        "## [4.2.0",
    ]
    text = (project / "CHANGELOG.md").read_text()
    assert "- Feature: Feature two. ([#101]" in text
    assert "- Fix: Fix one. ([#100]" in text
    assert "CI only" not in text
    assert "[4.3.0-rc.2]: https://github.com/" in text
    assert _snippets(project) == ["+ci.skip.md", "100.fix.md", "101.feature.md"]

    # GA replaces the RC section, and re-running GA replaces itself rather
    # than duplicating.
    for _ in range(2):
        assert changelog_update.update("CHANGELOG.md", "4.3.0") == 0
        assert [h.split("]")[0] for h in _headings(project)] == [
            "## [4.3.0",
            "## [4.2.0",
        ]
    assert _snippets(project) == ["+ci.skip.md", "100.fix.md", "101.feature.md"]


def test_real_towncrier_with_only_skip_snippets_fails_and_restores(project):
    (project / "docs" / "changes" / "+ci.skip.md").write_text("CI only.\n")
    before = (project / "CHANGELOG.md").read_text()

    assert changelog_update.update("CHANGELOG.md", "4.3.0") == 1
    assert (project / "CHANGELOG.md").read_text() == before


def test_real_towncrier_failure_restores_changelog(project):
    (project / "docs" / "changes" / "100.fix.md").write_text("Fix one.\n")
    (project / "docs" / "changes" / "_template.md.jinja").write_text("{% bogus %}")
    before = (project / "CHANGELOG.md").read_text()

    assert changelog_update.update("CHANGELOG.md", "4.3.0") != 0
    assert (project / "CHANGELOG.md").read_text() == before
    assert "100.fix.md" in _snippets(project)


def _git(*args):
    subprocess.run(
        [
            "git",
            "-c",
            "user.name=t",
            "-c",
            "user.email=t@example.invalid",
            "-c",
            "tag.gpgsign=false",
            "-c",
            "commit.gpgsign=false",
            *args,
        ],
        check=True,
        capture_output=True,
    )


@pytest.fixture
def repo(tmp_path, monkeypatch):
    monkeypatch.chdir(tmp_path)
    _git("init", "-q", "-b", "main")
    (tmp_path / "docs" / "changes").mkdir(parents=True)
    return tmp_path


def _commit(repo, files, message="c"):
    for name, content in files.items():
        (repo / "docs" / "changes" / name).write_text(content)
    _git("add", "-A")
    _git("commit", "-q", "-m", message)


def _present(repo):
    return sorted(p.name for p in (repo / "docs" / "changes").iterdir())


def test_consume_removes_only_snippets_released_by_non_rc_tags(repo):
    _commit(
        repo,
        {
            "README.md": "r",
            "_template.md.jinja": "t",
            "1.fix.md": "one",
            "2.fix.md": "two",
        },
    )
    _git("tag", "v4.3.0-rc.1")  # RCs don't consume.
    _commit(repo, {"3.fix.md": "three"})
    _git("tag", "v4.3.0")
    _commit(repo, {"4.fix.md": "four"})  # landed after the release was cut

    assert changelog_consume.release_tags() == ["v4.3.0"]
    removed = changelog_consume.consume(changelog_consume.release_tags())

    assert sorted(removed) == [
        "docs/changes/1.fix.md",
        "docs/changes/2.fix.md",
        "docs/changes/3.fix.md",
    ]
    assert _present(repo) == ["4.fix.md", "README.md", "_template.md.jinja"]

    # Idempotent: a second run has nothing to do, so the tree stays clean.
    assert changelog_consume.consume(changelog_consume.release_tags()) == []
    _git("commit", "-q", "-m", "consume")
    assert (
        subprocess.run(
            ["git", "status", "--porcelain"], capture_output=True, text=True
        ).stdout
        == ""
    )


def test_consume_leaves_modified_snippets_alone(repo, capsys):
    _commit(repo, {"1.fix.md": "one"})
    _git("tag", "v4.3.0")
    _commit(repo, {"1.fix.md": "one, but edited"})

    assert changelog_consume.consume(["v4.3.0"]) == []
    assert _present(repo) == ["1.fix.md"]
    assert "differs from v4.3.0" in capsys.readouterr().err


def test_consume_with_no_release_tags_is_a_noop(repo):
    _commit(repo, {"1.fix.md": "one"})

    assert changelog_consume.consume(changelog_consume.release_tags()) == []
    assert _present(repo) == ["1.fix.md"]
    assert os.path.exists("docs/changes/1.fix.md")
