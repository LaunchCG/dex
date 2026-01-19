"""Tests for dex.utils.markers module."""

from dex.utils.markers import (
    find_plugin_section,
    insert_plugin_section,
    list_plugin_sections,
    make_end_marker,
    make_start_marker,
    remove_plugin_section,
    wrap_content,
)


class TestMakeMarkers:
    """Tests for marker creation functions."""

    def test_make_start_marker(self):
        """Creates correct start marker format."""
        marker = make_start_marker("my-plugin")
        assert marker == "<!-- dex:plugin:my-plugin:start -->"

    def test_make_end_marker(self):
        """Creates correct end marker format."""
        marker = make_end_marker("my-plugin")
        assert marker == "<!-- dex:plugin:my-plugin:end -->"

    def test_markers_with_special_chars(self):
        """Handles plugin names with hyphens and underscores."""
        start = make_start_marker("my-cool_plugin")
        end = make_end_marker("my-cool_plugin")
        assert "my-cool_plugin" in start
        assert "my-cool_plugin" in end


class TestWrapContent:
    """Tests for wrap_content function."""

    def test_wraps_content_with_markers(self):
        """Wraps content with start and end markers."""
        content = "# Plugin Instructions\n\nDo this thing."
        wrapped = wrap_content("test-plugin", content)

        expected = """\
<!-- dex:plugin:test-plugin:start -->
# Plugin Instructions

Do this thing.
<!-- dex:plugin:test-plugin:end -->"""
        assert wrapped == expected

    def test_strips_whitespace(self):
        """Strips leading/trailing whitespace from content."""
        content = "\n\n  Content here  \n\n"
        wrapped = wrap_content("test-plugin", content)

        assert wrapped.startswith("<!-- dex:plugin:test-plugin:start -->\n")
        assert "Content here" in wrapped
        assert wrapped.endswith("<!-- dex:plugin:test-plugin:end -->")


class TestFindPluginSection:
    """Tests for find_plugin_section function."""

    def test_finds_existing_section(self):
        """Finds a plugin's section in file content."""
        file_content = """\
# CLAUDE.md

Some user content here.

<!-- dex:plugin:my-plugin:start -->
Plugin instructions here.
<!-- dex:plugin:my-plugin:end -->

More user content.
"""
        section = find_plugin_section(file_content, "my-plugin")

        assert section is not None
        assert section.plugin_name == "my-plugin"
        assert section.content == "Plugin instructions here."

    def test_returns_none_for_missing_section(self):
        """Returns None when plugin section not found."""
        file_content = "# Just some content\nNo markers here."

        section = find_plugin_section(file_content, "nonexistent")

        assert section is None

    def test_returns_none_for_incomplete_markers(self):
        """Returns None when only start marker exists."""
        file_content = """\
<!-- dex:plugin:broken-plugin:start -->
Content without end marker
"""
        section = find_plugin_section(file_content, "broken-plugin")

        assert section is None

    def test_finds_correct_section_among_multiple(self):
        """Finds the right section when multiple plugins present."""
        file_content = """\
<!-- dex:plugin:plugin-a:start -->
Plugin A content
<!-- dex:plugin:plugin-a:end -->

<!-- dex:plugin:plugin-b:start -->
Plugin B content
<!-- dex:plugin:plugin-b:end -->
"""
        section = find_plugin_section(file_content, "plugin-b")

        assert section is not None
        assert section.content == "Plugin B content"


class TestListPluginSections:
    """Tests for list_plugin_sections function."""

    def test_lists_all_plugins(self):
        """Lists all plugin names with sections."""
        file_content = """\
<!-- dex:plugin:plugin-a:start -->
A
<!-- dex:plugin:plugin-a:end -->

<!-- dex:plugin:plugin-b:start -->
B
<!-- dex:plugin:plugin-b:end -->

<!-- dex:plugin:plugin-c:start -->
C
<!-- dex:plugin:plugin-c:end -->
"""
        plugins = list_plugin_sections(file_content)

        assert len(plugins) == 3
        assert "plugin-a" in plugins
        assert "plugin-b" in plugins
        assert "plugin-c" in plugins

    def test_returns_empty_for_no_sections(self):
        """Returns empty list when no plugin sections."""
        file_content = "# Just regular content"

        plugins = list_plugin_sections(file_content)

        assert plugins == []


class TestInsertPluginSection:
    """Tests for insert_plugin_section function."""

    def test_inserts_into_empty_file(self):
        """Inserts section into empty file."""
        result = insert_plugin_section("", "my-plugin", "Plugin content")

        expected = """\
<!-- dex:plugin:my-plugin:start -->
Plugin content
<!-- dex:plugin:my-plugin:end -->
"""
        assert result == expected

    def test_appends_to_existing_content(self):
        """Appends section to file with existing content."""
        file_content = "# My CLAUDE.md\n\nUser instructions here."

        result = insert_plugin_section(file_content, "my-plugin", "Plugin content")

        assert result.startswith("# My CLAUDE.md")
        assert "User instructions here." in result
        assert "<!-- dex:plugin:my-plugin:start -->" in result
        assert "Plugin content" in result

    def test_replaces_existing_section(self):
        """Replaces existing plugin section with new content."""
        file_content = """\
# Header

<!-- dex:plugin:my-plugin:start -->
Old content
<!-- dex:plugin:my-plugin:end -->

# Footer
"""
        result = insert_plugin_section(file_content, "my-plugin", "New content")

        assert "Old content" not in result
        assert "New content" in result
        assert "# Header" in result
        assert "# Footer" in result
        # Should only have one section for this plugin
        assert result.count("my-plugin:start") == 1
        assert result.count("my-plugin:end") == 1

    def test_preserves_other_plugins(self):
        """Preserves other plugin sections when inserting."""
        file_content = """\
<!-- dex:plugin:other-plugin:start -->
Other plugin content
<!-- dex:plugin:other-plugin:end -->
"""
        result = insert_plugin_section(file_content, "my-plugin", "My content")

        assert "Other plugin content" in result
        assert "My content" in result


class TestRemovePluginSection:
    """Tests for remove_plugin_section function."""

    def test_removes_section(self):
        """Removes a plugin's section."""
        file_content = """\
# Header

<!-- dex:plugin:my-plugin:start -->
Plugin content
<!-- dex:plugin:my-plugin:end -->

# Footer
"""
        result = remove_plugin_section(file_content, "my-plugin")

        assert "my-plugin" not in result
        assert "Plugin content" not in result
        assert "# Header" in result
        assert "# Footer" in result

    def test_returns_unchanged_for_missing_section(self):
        """Returns unchanged content if section doesn't exist."""
        file_content = "# Just content\nNo markers."

        result = remove_plugin_section(file_content, "nonexistent")

        assert result == file_content

    def test_preserves_other_plugins(self):
        """Preserves other plugin sections when removing one."""
        file_content = """\
<!-- dex:plugin:keep-me:start -->
Keep this content
<!-- dex:plugin:keep-me:end -->

<!-- dex:plugin:remove-me:start -->
Remove this content
<!-- dex:plugin:remove-me:end -->
"""
        result = remove_plugin_section(file_content, "remove-me")

        assert "Keep this content" in result
        assert "remove-me" not in result
        assert "Remove this content" not in result

    def test_returns_empty_when_only_section_removed(self):
        """Returns empty string when file only contained the removed section."""
        file_content = """\
<!-- dex:plugin:only-one:start -->
Content
<!-- dex:plugin:only-one:end -->
"""
        result = remove_plugin_section(file_content, "only-one")

        assert result == ""
