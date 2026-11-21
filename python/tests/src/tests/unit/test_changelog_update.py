from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
from pathlib import Path

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
