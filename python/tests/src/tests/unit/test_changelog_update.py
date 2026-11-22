from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
from pathlib import Path

import pytest

# releng/changelog-update has no .py extension, so load it by path.
_script = Path(__file__).resolve().parents[5] / "releng" / "changelog-update"
_loader = SourceFileLoader("changelog_update", str(_script))
_spec = spec_from_loader("changelog_update", _loader)
assert _spec is not None
changelog_update = module_from_spec(_spec)
_loader.exec_module(changelog_update)

CHANGELOG = """\
# CHANGELOG

<!-- towncrier release notes start -->

## [4.3.0-rc.2] 2 December 2026
[4.3.0-rc.2]: https://example.invalid/v4.3.0-rc.2

- Fix: second

## [4.3.0-rc.1] 1 December 2026
[4.3.0-rc.1]: https://example.invalid/v4.3.0-rc.1

- Fix: first

## [4.30.0-rc.1] 1 December 2026

- Fix: other version

## [4.2.0] 5 November 2026

- Fix: old
"""


def test_strip_rc_sections_removes_only_matching_rcs():
    out = changelog_update.strip_rc_sections(CHANGELOG, "4.3.0")

    assert "[4.3.0-rc.1]" not in out
    assert "[4.3.0-rc.2]" not in out
    assert "Fix: first" not in out
    assert "Fix: second" not in out

    # Other versions, including one that merely shares a prefix, are kept
    # intact, along with everything above the first section.
    assert "## [4.30.0-rc.1] 1 December 2026" in out
    assert "Fix: other version" in out
    assert "## [4.2.0] 5 November 2026" in out
    assert "Fix: old" in out
    assert out.startswith("# CHANGELOG\n\n<!-- towncrier release notes start -->\n\n")


def test_strip_rc_sections_without_rcs_is_a_noop():
    assert changelog_update.strip_rc_sections(CHANGELOG, "9.9.9") == CHANGELOG


def _write(tmp_path):
    path = tmp_path / "CHANGELOG.md"
    path.write_text(CHANGELOG, encoding="utf-8")
    return path


def test_update_keeps_stripped_changelog_on_success(tmp_path, monkeypatch):
    path = _write(tmp_path)
    monkeypatch.setattr(changelog_update, "run_towncrier", lambda version, is_rc: 0)

    assert changelog_update.update(str(path), "4.3.0") == 0
    assert path.read_text(encoding="utf-8") == changelog_update.strip_rc_sections(
        CHANGELOG, "4.3.0"
    )


def test_update_restores_changelog_when_towncrier_fails(tmp_path, monkeypatch):
    path = _write(tmp_path)
    monkeypatch.setattr(changelog_update, "run_towncrier", lambda version, is_rc: 1)

    assert changelog_update.update(str(path), "4.3.0") == 1
    assert path.read_text(encoding="utf-8") == CHANGELOG


def test_update_restores_changelog_when_interrupted(tmp_path, monkeypatch):
    path = _write(tmp_path)

    def interrupted(version, is_rc):
        raise KeyboardInterrupt

    monkeypatch.setattr(changelog_update, "run_towncrier", interrupted)

    with pytest.raises(KeyboardInterrupt):
        changelog_update.update(str(path), "4.3.0-rc.3")
    assert path.read_text(encoding="utf-8") == CHANGELOG


def test_update_rejects_bad_version_without_touching_changelog(tmp_path, monkeypatch):
    path = _write(tmp_path)

    def must_not_run(version, is_rc):
        raise AssertionError("towncrier should not run")

    monkeypatch.setattr(changelog_update, "run_towncrier", must_not_run)

    assert changelog_update.update(str(path), "4.3") == 2
    assert path.read_text(encoding="utf-8") == CHANGELOG


def test_update_passes_rc_and_ga_modes_to_towncrier(tmp_path, monkeypatch):
    path = _write(tmp_path)
    calls = []
    monkeypatch.setattr(
        changelog_update,
        "run_towncrier",
        lambda version, is_rc: calls.append((version, is_rc)) or 0,
    )

    changelog_update.update(str(path), "4.3.0-rc.3")
    changelog_update.update(str(path), "4.3.0")

    assert calls == [("4.3.0-rc.3", True), ("4.3.0", False)]
