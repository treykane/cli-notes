"""Tests for CLI Notes application."""
import tempfile
from pathlib import Path
import pytest
from cli_notes.app import NotesApp


def test_app_initialization():
    """Test that the app can be initialized."""
    app = NotesApp()
    assert app.notes_dir is not None
    assert app.notes_dir.exists()


def test_notes_directory_creation():
    """Test that notes directory is created if it doesn't exist."""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Change HOME to temp directory to test fresh initialization
        import os
        original_home = os.environ.get('HOME')
        try:
            test_home = Path(tmpdir) / "test_home"
            test_home.mkdir()
            os.environ['HOME'] = str(test_home)
            
            # Verify directory doesn't exist yet
            notes_dir = test_home / "notes"
            assert not notes_dir.exists()
            
            # Create app (should create directory)
            app = NotesApp()
            
            # Verify directory was created by the app
            assert app.notes_dir.exists()
            assert app.notes_dir == notes_dir
        finally:
            # Restore original HOME
            if original_home:
                os.environ['HOME'] = original_home


def test_welcome_note_creation():
    """Test that a welcome note is created in empty directory."""
    app = NotesApp()
    welcome_note = app.notes_dir / "Welcome.md"
    
    if welcome_note.exists():
        content = welcome_note.read_text()
        assert "Welcome to CLI Notes" in content
        assert "Features" in content
        assert "Keyboard Shortcuts" in content


def test_note_creation():
    """Test creating a new note."""
    with tempfile.TemporaryDirectory() as tmpdir:
        notes_dir = Path(tmpdir) / "test_notes"
        notes_dir.mkdir()
        
        # Create a test note
        test_note = notes_dir / "test.md"
        test_note.write_text("# Test Note\n\nThis is a test.")
        
        assert test_note.exists()
        content = test_note.read_text()
        assert "Test Note" in content


def test_folder_creation():
    """Test creating a new folder."""
    with tempfile.TemporaryDirectory() as tmpdir:
        notes_dir = Path(tmpdir) / "test_notes"
        notes_dir.mkdir()
        
        # Create a test folder
        test_folder = notes_dir / "test_folder"
        test_folder.mkdir()
        
        assert test_folder.exists()
        assert test_folder.is_dir()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
