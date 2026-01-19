"""Tests for dex.template.context module."""

from pathlib import Path
from unittest.mock import patch

from dex.config.schemas import PluginManifest, SkillConfig
from dex.template.context import TemplateContext, build_context


class TestTemplateContextWithPlatform:
    """Tests for TemplateContext.with_platform()."""

    @patch("dex.template.context.get_os")
    @patch("dex.template.context.get_arch")
    def test_adds_platform_info(self, mock_arch, mock_os):
        """Adds platform information to context."""
        mock_os.return_value = "linux"
        mock_arch.return_value = "x64"

        ctx = TemplateContext().with_platform().build()

        assert ctx["platform"]["os"] == "linux"
        assert ctx["platform"]["arch"] == "x64"

    def test_returns_self_for_chaining(self):
        """Returns self for method chaining."""
        tc = TemplateContext()
        result = tc.with_platform()
        assert result is tc


class TestTemplateContextWithAgent:
    """Tests for TemplateContext.with_agent()."""

    def test_adds_agent_info(self):
        """Adds agent information to context."""
        ctx = TemplateContext().with_agent("claude-code").build()

        assert ctx["agent"]["name"] == "claude-code"

    def test_adds_additional_properties(self):
        """Adds additional agent properties."""
        ctx = (
            TemplateContext()
            .with_agent(
                "claude-code",
                version="1.0.0",
                supports_mcp=True,
            )
            .build()
        )

        assert ctx["agent"]["name"] == "claude-code"
        assert ctx["agent"]["version"] == "1.0.0"
        assert ctx["agent"]["supports_mcp"] is True


class TestTemplateContextWithEnvironment:
    """Tests for TemplateContext.with_environment()."""

    def test_adds_project_info(self, temp_dir: Path):
        """Adds project information to context."""
        ctx = TemplateContext().with_environment(temp_dir, project_name="test-project").build()

        assert ctx["env"]["project"]["root"] == str(temp_dir)
        assert ctx["env"]["project"]["name"] == "test-project"

    def test_uses_directory_name_as_default(self, temp_dir: Path):
        """Uses directory name if project_name not specified."""
        ctx = TemplateContext().with_environment(temp_dir).build()

        assert ctx["env"]["project"]["name"] == temp_dir.name

    def test_adds_home_directory(self, temp_dir: Path):
        """Adds home directory to context."""
        ctx = TemplateContext().with_environment(temp_dir).build()

        assert "home" in ctx["env"]
        assert len(ctx["env"]["home"]) > 0

    def test_adds_environment_variables(self, temp_dir: Path):
        """Adds environment variables to context."""
        import os

        with patch.dict(os.environ, {"TEST_VAR": "test_value"}):
            ctx = TemplateContext().with_environment(temp_dir).build()

        assert ctx["env"]["TEST_VAR"] == "test_value"


class TestTemplateContextWithPlugin:
    """Tests for TemplateContext.with_plugin()."""

    def test_adds_plugin_info(self, sample_plugin_manifest: PluginManifest):
        """Adds plugin information to context."""
        ctx = TemplateContext().with_plugin(sample_plugin_manifest).build()

        assert ctx["plugin"]["name"] == "test-plugin"
        assert ctx["plugin"]["version"] == "1.0.0"
        assert ctx["plugin"]["description"] == "A test plugin"

    def test_adds_dependencies(self):
        """Adds dependencies list."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Test",
            dependencies={"dep1": "^1.0.0", "dep2": "^2.0.0"},
        )

        ctx = TemplateContext().with_plugin(manifest).build()

        assert "dep1" in ctx["plugin"]["dependencies"]
        assert "dep2" in ctx["plugin"]["dependencies"]


class TestTemplateContextWithComponent:
    """Tests for TemplateContext.with_component()."""

    def test_adds_skill_component(self, sample_skill_config: SkillConfig):
        """Adds skill component info."""
        ctx = TemplateContext().with_component(sample_skill_config, "skill").build()

        assert ctx["component"]["name"] == "test-skill"
        assert ctx["component"]["type"] == "skill"

    def test_adds_context_root(self, sample_skill_config: SkillConfig):
        """Adds context root when provided."""
        ctx = (
            TemplateContext()
            .with_component(
                sample_skill_config,
                "skill",
                context_root=".claude/skills/test-plugin-test-skill/",
            )
            .build()
        )

        assert ctx["context"]["root"] == ".claude/skills/test-plugin-test-skill/"

    def test_no_context_without_root(self, sample_skill_config: SkillConfig):
        """No context.root if not provided."""
        ctx = TemplateContext().with_component(sample_skill_config, "skill").build()

        assert "context" not in ctx or "root" not in ctx.get("context", {})


class TestTemplateContextWithCustom:
    """Tests for TemplateContext.with_custom()."""

    def test_adds_simple_value(self):
        """Adds a simple custom value."""
        ctx = TemplateContext().with_custom("key", "value").build()
        assert ctx["key"] == "value"

    def test_adds_nested_value(self):
        """Adds a nested custom value using dot notation."""
        ctx = TemplateContext().with_custom("foo.bar.baz", "value").build()

        assert ctx["foo"]["bar"]["baz"] == "value"

    def test_returns_self_for_chaining(self):
        """Returns self for method chaining."""
        tc = TemplateContext()
        result = tc.with_custom("key", "value")
        assert result is tc


class TestTemplateContextMerge:
    """Tests for TemplateContext.merge()."""

    def test_merges_dict(self):
        """Merges another dictionary."""
        ctx = TemplateContext()
        ctx.merge({"key1": "value1", "key2": "value2"})
        result = ctx.build()

        assert result["key1"] == "value1"
        assert result["key2"] == "value2"

    def test_deep_merges(self):
        """Deep merges nested dictionaries."""
        ctx = TemplateContext()
        ctx.with_custom("nested.a", "original")
        ctx.merge({"nested": {"b": "added"}})
        result = ctx.build()

        assert result["nested"]["a"] == "original"
        assert result["nested"]["b"] == "added"

    def test_override_existing_values(self):
        """Override works correctly."""
        ctx = TemplateContext()
        ctx.with_custom("key", "original")
        ctx.merge({"key": "overridden"})
        result = ctx.build()

        assert result["key"] == "overridden"


class TestTemplateContextBuild:
    """Tests for TemplateContext.build()."""

    def test_returns_copy(self):
        """Returns a copy of the context."""
        tc = TemplateContext()
        tc.with_custom("key", "value")

        result1 = tc.build()
        result2 = tc.build()

        # Should be equal but not the same object
        assert result1 == result2
        assert result1 is not result2


class TestBuildContext:
    """Tests for build_context convenience function."""

    @patch("dex.template.context.get_os", return_value="linux")
    @patch("dex.template.context.get_arch", return_value="x64")
    def test_builds_complete_context(self, mock_arch, mock_os, temp_dir: Path):
        """Builds a complete context with all components."""
        ctx = build_context(
            project_root=temp_dir,
            agent_name="claude-code",
            project_name="test-project",
        )

        assert ctx["platform"]["os"] == "linux"
        assert ctx["agent"]["name"] == "claude-code"
        assert ctx["env"]["project"]["name"] == "test-project"

    @patch("dex.template.context.get_os", return_value="linux")
    @patch("dex.template.context.get_arch", return_value="x64")
    def test_includes_plugin_info(
        self, mock_arch, mock_os, temp_dir: Path, sample_plugin_manifest: PluginManifest
    ):
        """Includes plugin info when provided."""
        ctx = build_context(
            project_root=temp_dir,
            agent_name="claude-code",
            plugin=sample_plugin_manifest,
        )

        assert ctx["plugin"]["name"] == "test-plugin"

    @patch("dex.template.context.get_os", return_value="linux")
    @patch("dex.template.context.get_arch", return_value="x64")
    def test_includes_component_info(
        self, mock_arch, mock_os, temp_dir: Path, sample_skill_config: SkillConfig
    ):
        """Includes component info when provided."""
        ctx = build_context(
            project_root=temp_dir,
            agent_name="claude-code",
            component=sample_skill_config,
            component_type="skill",
        )

        assert ctx["component"]["name"] == "test-skill"

    @patch("dex.template.context.get_os", return_value="linux")
    @patch("dex.template.context.get_arch", return_value="x64")
    def test_includes_adapter_variables(self, mock_arch, mock_os, temp_dir: Path):
        """Includes adapter-specific variables."""
        ctx = build_context(
            project_root=temp_dir,
            agent_name="claude-code",
            adapter_variables={"custom": {"setting": "value"}},
        )

        assert ctx["custom"]["setting"] == "value"

    @patch("dex.template.context.get_os", return_value="linux")
    @patch("dex.template.context.get_arch", return_value="x64")
    def test_includes_context_root(
        self, mock_arch, mock_os, temp_dir: Path, sample_skill_config: SkillConfig
    ):
        """Includes context root when provided."""
        ctx = build_context(
            project_root=temp_dir,
            agent_name="claude-code",
            component=sample_skill_config,
            component_type="skill",
            context_root=".claude/skills/my-plugin-test-skill/",
        )

        assert ctx["context"]["root"] == ".claude/skills/my-plugin-test-skill/"
