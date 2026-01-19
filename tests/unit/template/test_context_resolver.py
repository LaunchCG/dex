"""Tests for dex.template.context_resolver module."""

from pathlib import Path

from dex.template.context_resolver import (
    PLATFORM_IDENTIFIERS,
    extract_platform_from_filename,
    find_platform_specific_file,
    normalize_adapter_name,
    parse_brace_expansion,
    resolve_context_spec,
)


class TestNormalizeAdapterName:
    """Tests for normalize_adapter_name()."""

    def test_converts_hyphens_to_underscores(self):
        """Converts hyphens to underscores."""
        assert normalize_adapter_name("claude-code") == "claude_code"
        assert normalize_adapter_name("github-copilot") == "github_copilot"

    def test_preserves_underscores(self):
        """Preserves existing underscores."""
        assert normalize_adapter_name("claude_code") == "claude_code"

    def test_handles_simple_names(self):
        """Handles names without hyphens or underscores."""
        assert normalize_adapter_name("cursor") == "cursor"
        assert normalize_adapter_name("codex") == "codex"


class TestParseBraceExpansion:
    """Tests for parse_brace_expansion()."""

    def test_parses_single_platform(self):
        """Parses single platform in braces."""
        result = parse_brace_expansion("context.{claude_code}.md")
        assert result == {"claude_code"}

    def test_parses_multiple_platforms(self):
        """Parses multiple platforms in braces."""
        result = parse_brace_expansion("context.{claude_code,cursor}.md")
        assert result == {"claude_code", "cursor"}

    def test_parses_three_platforms(self):
        """Parses three platforms in braces."""
        result = parse_brace_expansion("context.{claude_code,cursor,codex}.md")
        assert result == {"claude_code", "cursor", "codex"}

    def test_returns_empty_for_no_braces(self):
        """Returns empty set when no brace expansion present."""
        result = parse_brace_expansion("context.md")
        assert result == set()

    def test_filters_invalid_platforms(self):
        """Filters out invalid platform identifiers."""
        result = parse_brace_expansion("context.{claude_code,invalid_platform}.md")
        assert result == {"claude_code"}

    def test_handles_whitespace(self):
        """Handles whitespace around platform names."""
        result = parse_brace_expansion("context.{claude_code, cursor}.md")
        assert result == {"claude_code", "cursor"}


class TestExtractPlatformFromFilename:
    """Tests for extract_platform_from_filename()."""

    def test_extracts_claude_code(self):
        """Extracts claude_code platform."""
        result = extract_platform_from_filename("context.claude_code.md", "context")
        assert result == "claude_code"

    def test_extracts_cursor(self):
        """Extracts cursor platform."""
        result = extract_platform_from_filename("skill.cursor.md", "skill")
        assert result == "cursor"

    def test_returns_none_for_default_file(self):
        """Returns None for default file without platform suffix."""
        result = extract_platform_from_filename("context.md", "context")
        assert result is None

    def test_returns_none_for_invalid_platform(self):
        """Returns None for invalid platform identifier."""
        result = extract_platform_from_filename("context.invalid.md", "context")
        assert result is None

    def test_returns_none_for_brace_expansion(self):
        """Returns None for brace expansion files (handled separately)."""
        result = extract_platform_from_filename("context.{claude_code,cursor}.md", "context")
        assert result is None

    def test_returns_none_for_wrong_base_name(self):
        """Returns None when base name doesn't match."""
        result = extract_platform_from_filename("other.claude_code.md", "context")
        assert result is None


class TestFindPlatformSpecificFile:
    """Tests for find_platform_specific_file()."""

    def test_returns_platform_specific_override(self, temp_dir: Path):
        """Returns platform-specific override when it exists."""
        # Create files
        (temp_dir / "context.md").write_text("default")
        (temp_dir / "context.claude_code.md").write_text("claude code")

        result = find_platform_specific_file(temp_dir, "context.md", "claude-code")

        assert result == "context.claude_code.md"

    def test_returns_multi_platform_override(self, temp_dir: Path):
        """Returns multi-platform override when platform matches."""
        # Create files
        (temp_dir / "context.md").write_text("default")
        (temp_dir / "context.{claude_code,cursor}.md").write_text("shared")

        result = find_platform_specific_file(temp_dir, "context.md", "claude-code")

        assert result == "context.{claude_code,cursor}.md"

    def test_returns_default_when_no_override(self, temp_dir: Path):
        """Returns default file when no platform-specific override exists."""
        # Create only default file
        (temp_dir / "context.md").write_text("default")

        result = find_platform_specific_file(temp_dir, "context.md", "claude-code")

        assert result == "context.md"

    def test_prefers_exact_platform_over_multi_platform(self, temp_dir: Path):
        """Prefers exact platform match over multi-platform override."""
        # Create all three types
        (temp_dir / "context.md").write_text("default")
        (temp_dir / "context.claude_code.md").write_text("exact")
        (temp_dir / "context.{claude_code,cursor}.md").write_text("multi")

        result = find_platform_specific_file(temp_dir, "context.md", "claude-code")

        assert result == "context.claude_code.md"

    def test_handles_subdirectory(self, temp_dir: Path):
        """Handles context files in subdirectories."""
        # Create subdirectory with files
        (temp_dir / "context").mkdir()
        (temp_dir / "context" / "skill.md").write_text("default")
        (temp_dir / "context" / "skill.claude_code.md").write_text("claude")

        result = find_platform_specific_file(temp_dir, "context/skill.md", "claude-code")

        assert result == "context/skill.claude_code.md"

    def test_handles_leading_dot_slash(self, temp_dir: Path):
        """Handles paths with leading ./."""
        (temp_dir / "context.md").write_text("default")
        (temp_dir / "context.cursor.md").write_text("cursor")

        result = find_platform_specific_file(temp_dir, "./context.md", "cursor")

        assert result == "context.cursor.md"

    def test_returns_original_when_directory_missing(self, temp_dir: Path):
        """Returns original path when directory doesn't exist."""
        result = find_platform_specific_file(temp_dir, "nonexistent/context.md", "claude-code")

        assert result == "nonexistent/context.md"


class TestResolveContextSpec:
    """Tests for resolve_context_spec()."""

    def test_resolves_single_string(self, temp_dir: Path):
        """Resolves single string context spec."""
        (temp_dir / "context.md").write_text("default")
        (temp_dir / "context.claude_code.md").write_text("claude")

        result = resolve_context_spec("context.md", temp_dir, "claude-code")

        assert result == "context.claude_code.md"

    def test_resolves_list_of_strings(self, temp_dir: Path):
        """Resolves list of string paths."""
        (temp_dir / "intro.md").write_text("intro")
        (temp_dir / "intro.claude_code.md").write_text("intro claude")
        (temp_dir / "details.md").write_text("details")

        result = resolve_context_spec(["intro.md", "details.md"], temp_dir, "claude-code")

        assert result == ["intro.claude_code.md", "details.md"]

    def test_resolves_conditional_includes(self, temp_dir: Path):
        """Resolves paths within conditional includes."""
        (temp_dir / "optional.md").write_text("optional")
        (temp_dir / "optional.cursor.md").write_text("cursor optional")

        result = resolve_context_spec(
            [{"path": "optional.md", "if": "some_condition"}],
            temp_dir,
            "cursor",
        )

        assert result == [{"path": "optional.cursor.md", "if": "some_condition"}]

    def test_handles_mixed_list(self, temp_dir: Path):
        """Handles list with both strings and conditional includes."""
        (temp_dir / "always.md").write_text("always")
        (temp_dir / "always.codex.md").write_text("codex always")
        (temp_dir / "conditional.md").write_text("conditional")

        result = resolve_context_spec(
            [
                "always.md",
                {"path": "conditional.md", "if": "condition"},
            ],
            temp_dir,
            "codex",
        )

        assert result == [
            "always.codex.md",
            {"path": "conditional.md", "if": "condition"},
        ]

    def test_returns_unchanged_for_non_string_non_list(self, temp_dir: Path):
        """Returns unchanged for unexpected types."""
        result = resolve_context_spec(None, temp_dir, "claude-code")  # type: ignore
        assert result is None


class TestPlatformIdentifiers:
    """Tests for PLATFORM_IDENTIFIERS constant."""

    def test_contains_expected_platforms(self):
        """Contains all expected platform identifiers."""
        expected = {"claude_code", "cursor", "codex", "github_copilot", "antigravity"}
        assert expected == PLATFORM_IDENTIFIERS

    def test_uses_underscores(self):
        """All identifiers use underscores, not hyphens."""
        for identifier in PLATFORM_IDENTIFIERS:
            assert "-" not in identifier
