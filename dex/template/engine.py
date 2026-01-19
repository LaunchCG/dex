"""Jinja2 template engine wrapper for Dex."""

from pathlib import Path
from typing import Any

from jinja2 import (
    BaseLoader,
    ChoiceLoader,
    Environment,
    FileSystemLoader,
    TemplateError,
    TemplateSyntaxError,
    UndefinedError,
)

from dex.template.filters import CUSTOM_FILTERS


class TemplateRenderError(Exception):
    """Error rendering a template."""

    def __init__(self, message: str, source: str | None = None, line: int | None = None):
        self.source = source
        self.line = line
        super().__init__(message)


class StringLoader(BaseLoader):
    """Jinja2 loader for string templates."""

    def get_source(self, environment: Environment, template: str) -> tuple[str, str | None, Any]:
        """Return the template source.

        For string templates, the template name IS the source.
        """
        return template, None, lambda: True


class TemplateEngine:
    """Jinja2-based template engine for Dex.

    This engine renders context files (.md files) with platform and
    environment-specific content.

    Supports sub-templates via {% include "path/to/partial.md" %} when
    a base_path is provided during rendering.
    """

    def __init__(self, base_path: Path | None = None) -> None:
        """Initialize the template engine.

        Args:
            base_path: Optional base directory for resolving includes.
                       If provided, enables {% include %} and {% extends %}.
        """
        self._base_path = base_path
        self._env = self._create_environment(base_path)

    def _create_environment(self, base_path: Path | None) -> Environment:
        """Create Jinja2 environment with appropriate loader."""
        if base_path and base_path.exists():
            # Use ChoiceLoader: try filesystem first, fall back to string
            loader: BaseLoader = ChoiceLoader(
                [
                    FileSystemLoader(str(base_path)),
                    StringLoader(),
                ]
            )
        else:
            loader = StringLoader()

        env = Environment(
            loader=loader,
            autoescape=False,  # MD files don't need HTML escaping
            trim_blocks=True,
            lstrip_blocks=True,
            keep_trailing_newline=True,
        )

        # Register custom filters
        env.filters.update(CUSTOM_FILTERS)

        # Register custom tests
        env.tests["windows"] = lambda x: x == "windows"
        env.tests["linux"] = lambda x: x == "linux"
        env.tests["macos"] = lambda x: x == "macos"
        env.tests["unix"] = lambda x: x in ("linux", "macos")

        return env

    def with_base_path(self, base_path: Path) -> "TemplateEngine":
        """Create a new engine with a different base path for includes.

        Args:
            base_path: Directory to resolve includes from.

        Returns:
            New TemplateEngine instance with the base path set.
        """
        return TemplateEngine(base_path=base_path)

    def render_string(self, template_str: str, context: dict[str, Any]) -> str:
        """Render a template string.

        Args:
            template_str: Template content with Jinja2 syntax
            context: Context dictionary for variable substitution

        Returns:
            Rendered template string

        Raises:
            TemplateRenderError: If rendering fails
        """
        try:
            template = self._env.from_string(template_str)
            return template.render(context)
        except TemplateSyntaxError as e:
            raise TemplateRenderError(
                f"Template syntax error: {e.message}",
                source=template_str[:100],
                line=e.lineno,
            ) from e
        except UndefinedError as e:
            raise TemplateRenderError(
                f"Undefined variable in template: {e}",
                source=template_str[:100],
            ) from e
        except TemplateError as e:
            raise TemplateRenderError(f"Template error: {e}") from e

    def render_file(self, path: Path, context: dict[str, Any]) -> str:
        """Render a template file.

        Args:
            path: Path to the template file
            context: Context dictionary for variable substitution

        Returns:
            Rendered template content

        Raises:
            FileNotFoundError: If the file doesn't exist
            TemplateRenderError: If rendering fails
        """
        if not path.exists():
            raise FileNotFoundError(f"Template file not found: {path}")

        template_str = path.read_text(encoding="utf-8")

        try:
            return self.render_string(template_str, context)
        except TemplateRenderError as e:
            # Add file path to error
            raise TemplateRenderError(
                f"Error rendering {path}: {e}",
                source=str(path),
                line=e.line,
            ) from e

    def check_syntax(self, template_str: str) -> list[str]:
        """Check template syntax without rendering.

        Args:
            template_str: Template content to check

        Returns:
            List of error messages (empty if valid)
        """
        errors = []
        try:
            self._env.parse(template_str)
        except TemplateSyntaxError as e:
            errors.append(f"Line {e.lineno}: {e.message}")
        return errors


# Global engine instance
_engine: TemplateEngine | None = None


def get_engine() -> TemplateEngine:
    """Get the global template engine instance."""
    global _engine
    if _engine is None:
        _engine = TemplateEngine()
    return _engine


def render(template_str: str, context: dict[str, Any]) -> str:
    """Render a template string using the global engine.

    Args:
        template_str: Template content with Jinja2 syntax
        context: Context dictionary

    Returns:
        Rendered string
    """
    return get_engine().render_string(template_str, context)


def render_file(path: Path, context: dict[str, Any], base_path: Path | None = None) -> str:
    """Render a template file with optional base path for includes.

    Args:
        path: Path to the template file
        context: Context dictionary
        base_path: Optional base directory for resolving includes.
                   If not provided, uses the template file's parent directory.

    Returns:
        Rendered content
    """
    # Use the template file's directory as the default base_path for includes
    if base_path is None:
        base_path = path.parent

    engine = TemplateEngine(base_path=base_path)
    return engine.render_file(path, context)
