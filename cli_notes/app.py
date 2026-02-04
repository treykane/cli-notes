"""Main application for CLI Notes."""
import os
import sys
from pathlib import Path
from typing import Optional

from textual.app import App, ComposeResult
from textual.containers import Container, Horizontal, Vertical
from textual.widgets import (
    Header,
    Footer,
    DirectoryTree,
    Markdown,
    TextArea,
    Button,
    Static,
    Input,
)
from textual.binding import Binding
from textual.screen import Screen
from textual import events


class NewNoteScreen(Screen):
    """Screen for creating a new note."""

    BINDINGS = [
        Binding("escape", "cancel", "Cancel"),
        Binding("ctrl+s", "save", "Save"),
    ]

    def __init__(self, notes_dir: Path, parent_dir: Optional[Path] = None):
        super().__init__()
        self.notes_dir = notes_dir
        self.parent_dir = parent_dir or notes_dir

    def compose(self) -> ComposeResult:
        yield Header()
        with Vertical():
            yield Static("Create New Note", classes="screen-title")
            # Safely display relative path
            try:
                rel_path = self.parent_dir.relative_to(self.notes_dir)
                location = str(rel_path) if str(rel_path) != "." else "/"
            except ValueError:
                location = str(self.parent_dir)
            yield Static(f"Location: {location}")
            yield Input(placeholder="Note name (without .md extension)", id="note-name")
            with Horizontal():
                yield Button("Save (Ctrl+S)", variant="success", id="save-btn")
                yield Button("Cancel (Esc)", variant="default", id="cancel-btn")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "save-btn":
            self.action_save()
        elif event.button.id == "cancel-btn":
            self.action_cancel()

    def action_save(self) -> None:
        """Save the new note."""
        name_input = self.query_one("#note-name", Input)
        note_name = name_input.value.strip()
        
        if not note_name:
            return
        
        # Add .md extension if not present
        if not note_name.endswith(".md"):
            note_name += ".md"
        
        note_path = self.parent_dir / note_name
        
        # Create the note with basic template
        note_path.write_text(f"# {note_name[:-3]}\n\nYour note content here...\n")
        
        self.dismiss(note_path)

    def action_cancel(self) -> None:
        """Cancel and return to main screen."""
        self.dismiss(None)


class NewFolderScreen(Screen):
    """Screen for creating a new folder."""

    BINDINGS = [
        Binding("escape", "cancel", "Cancel"),
        Binding("ctrl+s", "save", "Save"),
    ]

    def __init__(self, notes_dir: Path, parent_dir: Optional[Path] = None):
        super().__init__()
        self.notes_dir = notes_dir
        self.parent_dir = parent_dir or notes_dir

    def compose(self) -> ComposeResult:
        yield Header()
        with Vertical():
            yield Static("Create New Folder", classes="screen-title")
            # Safely display relative path
            try:
                rel_path = self.parent_dir.relative_to(self.notes_dir)
                location = str(rel_path) if str(rel_path) != "." else "/"
            except ValueError:
                location = str(self.parent_dir)
            yield Static(f"Location: {location}")
            yield Input(placeholder="Folder name", id="folder-name")
            with Horizontal():
                yield Button("Create (Ctrl+S)", variant="success", id="save-btn")
                yield Button("Cancel (Esc)", variant="default", id="cancel-btn")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "save-btn":
            self.action_save()
        elif event.button.id == "cancel-btn":
            self.action_cancel()

    def action_save(self) -> None:
        """Create the new folder."""
        name_input = self.query_one("#folder-name", Input)
        folder_name = name_input.value.strip()
        
        if not folder_name:
            return
        
        folder_path = self.parent_dir / folder_name
        folder_path.mkdir(parents=True, exist_ok=True)
        
        self.dismiss(folder_path)

    def action_cancel(self) -> None:
        """Cancel and return to main screen."""
        self.dismiss(None)


class EditNoteScreen(Screen):
    """Screen for editing a note."""

    BINDINGS = [
        Binding("escape", "cancel", "Cancel"),
        Binding("ctrl+s", "save", "Save"),
    ]

    def __init__(self, note_path: Path):
        super().__init__()
        self.note_path = note_path
        self.original_content = note_path.read_text() if note_path.exists() else ""

    def compose(self) -> ComposeResult:
        yield Header()
        with Vertical():
            yield Static(f"Editing: {self.note_path.name}", classes="screen-title")
            yield TextArea(self.original_content, language="markdown", id="editor")
            with Horizontal():
                yield Button("Save (Ctrl+S)", variant="success", id="save-btn")
                yield Button("Cancel (Esc)", variant="default", id="cancel-btn")
        yield Footer()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "save-btn":
            self.action_save()
        elif event.button.id == "cancel-btn":
            self.action_cancel()

    def action_save(self) -> None:
        """Save the edited note."""
        editor = self.query_one("#editor", TextArea)
        self.note_path.write_text(editor.text)
        self.dismiss(True)

    def action_cancel(self) -> None:
        """Cancel editing and return to main screen."""
        self.dismiss(False)


class NotesApp(App):
    """A TUI application for managing notes."""

    CSS = """
    Screen {
        background: $surface;
    }

    .screen-title {
        background: $primary;
        color: $text;
        padding: 1;
        text-align: center;
        text-style: bold;
    }

    #tree-container {
        width: 40;
        border: solid $primary;
    }

    #content-container {
        border: solid $accent;
    }

    DirectoryTree {
        width: 100%;
        height: 100%;
    }

    Markdown {
        width: 100%;
        height: 100%;
        padding: 1 2;
    }

    TextArea {
        height: 100%;
    }

    Button {
        margin: 1;
    }

    Input {
        margin: 1;
    }

    .note-info {
        background: $boost;
        color: $text;
        padding: 1;
        text-align: center;
    }
    """

    BINDINGS = [
        Binding("q", "quit", "Quit"),
        Binding("n", "new_note", "New Note"),
        Binding("f", "new_folder", "New Folder"),
        Binding("e", "edit_note", "Edit Note"),
        Binding("d", "delete_note", "Delete"),
        Binding("r", "refresh", "Refresh"),
    ]

    def __init__(self):
        super().__init__()
        # Use ~/notes as default notes directory
        self.notes_dir = Path.home() / "notes"
        self.notes_dir.mkdir(exist_ok=True)
        
        # Create a welcome note if directory is empty
        if not any(self.notes_dir.iterdir()):
            welcome_note = self.notes_dir / "Welcome.md"
            welcome_note.write_text("""# Welcome to CLI Notes!

This is your personal notes manager in the terminal.

## Features

- 📝 Create and edit notes in Markdown
- 📁 Organize notes in folders
- 🎨 View rendered Markdown formatting
- ⌨️  Keyboard-driven interface

## Keyboard Shortcuts

- `n` - Create a new note
- `f` - Create a new folder
- `e` - Edit the selected note
- `d` - Delete the selected note
- `r` - Refresh the directory tree
- `q` - Quit the application

## Getting Started

1. Press `n` to create a new note
2. Select a note and press `e` to edit it
3. Press `f` to create folders and organize your notes

Happy note-taking! 📚
""")
        
        self.current_file: Optional[Path] = None

    def compose(self) -> ComposeResult:
        """Create the app layout."""
        yield Header()
        with Horizontal():
            with Container(id="tree-container"):
                yield DirectoryTree(self.notes_dir, id="tree")
            with Container(id="content-container"):
                yield Static("Select a note to view", classes="note-info", id="viewer")
        yield Footer()

    def on_mount(self) -> None:
        """Handle mount event."""
        self.title = "CLI Notes"
        self.sub_title = f"📂 {self.notes_dir}"

    def on_directory_tree_file_selected(
        self, event: DirectoryTree.FileSelected
    ) -> None:
        """Handle file selection in the directory tree."""
        file_path = event.path
        
        # Only show markdown files
        if file_path.suffix == ".md":
            self.current_file = file_path
            self.show_note(file_path)

    def show_note(self, note_path: Path) -> None:
        """Display a note with rendered markdown."""
        try:
            content = note_path.read_text()
            viewer = self.query_one("#viewer")
            
            # Replace the viewer with a Markdown widget
            new_viewer = Markdown(content, id="viewer")
            viewer.remove()
            container = self.query_one("#content-container")
            container.mount(new_viewer)
        except Exception as e:
            self.notify(f"Error reading note: {e}", severity="error")

    def action_new_note(self) -> None:
        """Create a new note."""
        # Get the currently selected directory
        tree = self.query_one("#tree", DirectoryTree)
        selected_path = (tree.cursor_node.data.path 
                        if tree.cursor_node and tree.cursor_node.data 
                        else self.notes_dir)
        
        # If selected path is a file, use its parent directory
        if selected_path.is_file():
            parent_dir = selected_path.parent
        else:
            parent_dir = selected_path
        
        def handle_new_note(result: Optional[Path]) -> None:
            if result:
                self.notify(f"Created note: {result.name}", severity="information")
                self.refresh_tree()
                self.current_file = result
                self.show_note(result)
        
        self.push_screen(NewNoteScreen(self.notes_dir, parent_dir), handle_new_note)

    def action_new_folder(self) -> None:
        """Create a new folder."""
        # Get the currently selected directory
        tree = self.query_one("#tree", DirectoryTree)
        selected_path = (tree.cursor_node.data.path 
                        if tree.cursor_node and tree.cursor_node.data 
                        else self.notes_dir)
        
        # If selected path is a file, use its parent directory
        if selected_path.is_file():
            parent_dir = selected_path.parent
        else:
            parent_dir = selected_path
        
        def handle_new_folder(result: Optional[Path]) -> None:
            if result:
                self.notify(f"Created folder: {result.name}", severity="information")
                self.refresh_tree()
        
        self.push_screen(NewFolderScreen(self.notes_dir, parent_dir), handle_new_folder)

    def action_edit_note(self) -> None:
        """Edit the currently selected note."""
        if not self.current_file:
            self.notify("No note selected", severity="warning")
            return
        
        def handle_edit(saved: bool) -> None:
            if saved:
                self.notify(f"Saved: {self.current_file.name}", severity="information")
                self.show_note(self.current_file)
        
        self.push_screen(EditNoteScreen(self.current_file), handle_edit)

    def action_delete_note(self) -> None:
        """Delete the currently selected note or folder."""
        tree = self.query_one("#tree", DirectoryTree)
        if not tree.cursor_node:
            self.notify("No item selected", severity="warning")
            return
        
        selected_path = tree.cursor_node.data.path
        
        # Don't allow deleting the root notes directory
        if selected_path == self.notes_dir:
            self.notify("Cannot delete the root notes directory", severity="error")
            return
        
        try:
            if selected_path.is_file():
                selected_path.unlink()
                self.notify(f"Deleted: {selected_path.name}", severity="information")
            elif selected_path.is_dir():
                # Only delete if empty
                if any(selected_path.iterdir()):
                    self.notify("Folder is not empty. Delete contents first.", severity="warning")
                    return
                selected_path.rmdir()
                self.notify(f"Deleted folder: {selected_path.name}", severity="information")
            
            self.current_file = None
            self.refresh_tree()
            
            # Clear the viewer
            viewer = self.query_one("#viewer")
            if isinstance(viewer, Markdown):
                new_viewer = Static("Select a note to view", classes="note-info", id="viewer")
                viewer.remove()
                container = self.query_one("#content-container")
                container.mount(new_viewer)
        except Exception as e:
            self.notify(f"Error deleting: {e}", severity="error")

    def action_refresh(self) -> None:
        """Refresh the directory tree."""
        self.refresh_tree()
        self.notify("Refreshed", severity="information")

    def refresh_tree(self) -> None:
        """Refresh the directory tree widget."""
        tree = self.query_one("#tree", DirectoryTree)
        tree.reload()


def main():
    """Main entry point for the application."""
    app = NotesApp()
    app.run()


if __name__ == "__main__":
    main()
