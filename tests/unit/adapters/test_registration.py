"""Tests for adapter registration in dex.adapters module."""

import pytest

from dex.adapters import get_adapter, list_adapters
from dex.adapters.base import PlatformAdapter


class TestRegisterAdapter:
    """Tests for register_adapter decorator."""

    def test_registers_adapter(self):
        """Decorator registers adapter class."""
        # Claude-code is already registered
        adapters = list_adapters()
        assert "claude-code" in adapters

    def test_decorator_returns_class(self):
        """Decorator returns the original class."""
        # The registration should not modify the class
        from dex.adapters.claude_code import ClaudeCodeAdapter

        assert ClaudeCodeAdapter is not None
        assert issubclass(ClaudeCodeAdapter, PlatformAdapter)


class TestGetAdapter:
    """Tests for get_adapter function."""

    def test_returns_adapter_instance(self):
        """Returns an instantiated adapter."""
        adapter = get_adapter("claude-code")

        assert adapter is not None
        assert isinstance(adapter, PlatformAdapter)

    def test_raises_for_unknown_adapter(self):
        """Raises ValueError for unknown adapter."""
        with pytest.raises(ValueError, match="Unknown adapter"):
            get_adapter("nonexistent-adapter")

    def test_error_lists_available_adapters(self):
        """Error message lists available adapters."""
        with pytest.raises(ValueError) as exc_info:
            get_adapter("nonexistent")

        assert "claude-code" in str(exc_info.value)


class TestListAdapters:
    """Tests for list_adapters function."""

    def test_returns_list(self):
        """Returns a list."""
        adapters = list_adapters()
        assert isinstance(adapters, list)

    def test_includes_claude_code(self):
        """Includes claude-code adapter."""
        adapters = list_adapters()
        assert "claude-code" in adapters

    def test_list_is_not_empty(self):
        """List is not empty (at least claude-code is registered)."""
        adapters = list_adapters()
        assert len(adapters) > 0
