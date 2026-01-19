"""Tests for dex.template.engine module."""

from pathlib import Path

import pytest

from dex.template.engine import (
    TemplateEngine,
    TemplateRenderError,
    get_engine,
    render,
    render_file,
)


class TestTemplateEngineRenderString:
    """Tests for TemplateEngine.render_string()."""

    def test_renders_simple_template(self):
        """Renders a simple template string."""
        engine = TemplateEngine()
        result = engine.render_string("Hello, {{ name }}!", {"name": "World"})
        assert result == "Hello, World!"

    def test_renders_with_multiple_variables(self):
        """Renders template with multiple variables."""
        engine = TemplateEngine()
        result = engine.render_string(
            "{{ greeting }}, {{ name }}!",
            {"greeting": "Hello", "name": "Dex"},
        )
        assert result == "Hello, Dex!"

    def test_renders_with_nested_context(self):
        """Renders template with nested context."""
        engine = TemplateEngine()
        result = engine.render_string(
            "OS: {{ platform.os }}, Arch: {{ platform.arch }}",
            {"platform": {"os": "linux", "arch": "x64"}},
        )
        assert result == "OS: linux, Arch: x64"

    def test_renders_conditional(self):
        """Renders template with conditional."""
        engine = TemplateEngine()
        template = "{% if show %}Visible{% else %}Hidden{% endif %}"

        assert engine.render_string(template, {"show": True}) == "Visible"
        assert engine.render_string(template, {"show": False}) == "Hidden"

    def test_renders_loop(self):
        """Renders template with loop."""
        engine = TemplateEngine()
        template = "{% for item in items %}{{ item }}{% endfor %}"
        result = engine.render_string(template, {"items": ["a", "b", "c"]})
        assert result == "abc"

    def test_raises_for_syntax_error(self):
        """Raises TemplateRenderError for syntax error."""
        engine = TemplateEngine()
        with pytest.raises(TemplateRenderError, match="Template syntax error"):
            engine.render_string("{{ unclosed", {})

    def test_undefined_variable_renders_empty(self):
        """Undefined variables render as empty by default.

        Note: Jinja2's default Undefined class doesn't raise errors,
        it just renders as empty string. This is intentional for
        template flexibility.
        """
        engine = TemplateEngine()
        result = engine.render_string("Hello {{ undefined_var }}!", {})
        # Undefined variables render as empty string
        assert result == "Hello !"

    def test_platform_tests_work(self):
        """Platform-specific tests work."""
        engine = TemplateEngine()

        template = "{% if platform.os is windows %}Windows{% else %}Not Windows{% endif %}"
        assert engine.render_string(template, {"platform": {"os": "windows"}}) == "Windows"
        assert engine.render_string(template, {"platform": {"os": "linux"}}) == "Not Windows"

    def test_unix_test_matches_linux_and_macos(self):
        """Unix test matches both Linux and macOS."""
        engine = TemplateEngine()
        template = "{% if platform.os is unix %}Unix{% else %}Not Unix{% endif %}"

        assert engine.render_string(template, {"platform": {"os": "linux"}}) == "Unix"
        assert engine.render_string(template, {"platform": {"os": "macos"}}) == "Unix"
        assert engine.render_string(template, {"platform": {"os": "windows"}}) == "Not Unix"


class TestTemplateEngineRenderFile:
    """Tests for TemplateEngine.render_file()."""

    def test_renders_file(self, temp_dir: Path):
        """Renders a template file."""
        template_file = temp_dir / "template.md"
        template_file.write_text("Hello, {{ name }}!")

        engine = TemplateEngine()
        result = engine.render_file(template_file, {"name": "World"})

        assert result == "Hello, World!"

    def test_raises_for_missing_file(self, temp_dir: Path):
        """Raises FileNotFoundError for missing file."""
        engine = TemplateEngine()
        with pytest.raises(FileNotFoundError, match="Template file not found"):
            engine.render_file(temp_dir / "nonexistent.md", {})

    def test_includes_file_path_in_error(self, temp_dir: Path):
        """Includes file path in render error."""
        template_file = temp_dir / "bad_template.md"
        template_file.write_text("{{ unclosed")

        engine = TemplateEngine()
        with pytest.raises(TemplateRenderError) as exc_info:
            engine.render_file(template_file, {})
        assert exc_info.value.source is not None
        assert str(template_file) in exc_info.value.source


class TestTemplateEngineCheckSyntax:
    """Tests for TemplateEngine.check_syntax()."""

    def test_valid_template_returns_empty(self):
        """Valid template returns empty list."""
        engine = TemplateEngine()
        errors = engine.check_syntax("Hello, {{ name }}!")
        assert errors == []

    def test_invalid_template_returns_errors(self):
        """Invalid template returns error messages."""
        engine = TemplateEngine()
        errors = engine.check_syntax("{{ unclosed")
        assert len(errors) > 0
        assert "Line" in errors[0]

    def test_multiple_errors(self):
        """Returns all syntax errors."""
        engine = TemplateEngine()
        errors = engine.check_syntax("{% for x in %}{{ x }}{% endfor %}")
        assert len(errors) > 0


class TestCustomFilters:
    """Tests for custom filters in TemplateEngine."""

    def test_basename_filter(self):
        """basename filter works."""
        engine = TemplateEngine()
        result = engine.render_string("{{ path | basename }}", {"path": "/path/to/file.txt"})
        assert result == "file.txt"

    def test_dirname_filter(self):
        """dirname filter works."""
        engine = TemplateEngine()
        result = engine.render_string("{{ path | dirname }}", {"path": "/path/to/file.txt"})
        assert result == "/path/to"

    def test_extension_filter(self):
        """extension filter works."""
        engine = TemplateEngine()
        result = engine.render_string("{{ path | extension }}", {"path": "/path/to/file.txt"})
        assert result == ".txt"

    def test_to_posix_filter(self):
        """to_posix filter works."""
        engine = TemplateEngine()
        result = engine.render_string("{{ path | to_posix }}", {"path": "path\\to\\file"})
        assert result == "path/to/file"

    def test_default_value_filter(self):
        """default_value filter works."""
        engine = TemplateEngine()
        result = engine.render_string(
            "{{ value | default_value('default') }}",
            {"value": ""},
        )
        assert result == "default"


class TestGetEngine:
    """Tests for get_engine function."""

    def test_returns_engine_instance(self):
        """Returns a TemplateEngine instance."""
        engine = get_engine()
        assert isinstance(engine, TemplateEngine)

    def test_returns_same_instance(self):
        """Returns the same singleton instance."""
        engine1 = get_engine()
        engine2 = get_engine()
        assert engine1 is engine2


class TestRenderFunction:
    """Tests for render convenience function."""

    def test_renders_template(self):
        """Renders template using global engine."""
        result = render("Hello, {{ name }}!", {"name": "World"})
        assert result == "Hello, World!"


class TestRenderFileFunction:
    """Tests for render_file convenience function."""

    def test_renders_file(self, temp_dir: Path):
        """Renders file using global engine."""
        template_file = temp_dir / "template.md"
        template_file.write_text("Hello, {{ name }}!")

        result = render_file(template_file, {"name": "World"})
        assert result == "Hello, World!"


class TestTemplateRenderError:
    """Tests for TemplateRenderError exception."""

    def test_error_attributes(self):
        """Error has source and line attributes."""
        error = TemplateRenderError("Test error", source="template.md", line=5)
        assert error.source == "template.md"
        assert error.line == 5

    def test_error_message(self):
        """Error has message."""
        error = TemplateRenderError("Something went wrong")
        assert str(error) == "Something went wrong"
