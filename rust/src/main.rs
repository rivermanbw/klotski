mod app;
mod board;
mod league;
mod presets;
mod save;
mod solver;
mod ui;

use std::io;
use std::time::Duration;

use crossterm::{
    event::{self, Event, KeyCode, KeyEvent, KeyModifiers},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{backend::CrosstermBackend, text::Text, widgets::Paragraph, Terminal};

use app::App;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Setup terminal.
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let mut app = App::new();

    loop {
        // Poll background messages.
        app.poll_bg();

        // Draw.
        terminal.draw(|f| {
            let lines = ui::render(&app);
            let text = Text::from(lines);
            let paragraph = Paragraph::new(text);
            f.render_widget(paragraph, f.area());
        })?;

        // Handle input with a short timeout so we can poll bg messages.
        if event::poll(Duration::from_millis(50))? {
            if let Event::Key(key_event) = event::read()? {
                // Skip key release events.
                if key_event.kind != event::KeyEventKind::Press {
                    continue;
                }
                let key_str = key_event_to_string(&key_event);
                if !key_str.is_empty() {
                    if app.handle_key(&key_str) {
                        break;
                    }
                }
            }
        }
    }

    // Restore terminal.
    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    Ok(())
}

/// Convert a crossterm KeyEvent into the string format used by the app.
fn key_event_to_string(event: &KeyEvent) -> String {
    let modifiers = event.modifiers;

    // Handle Ctrl+key combos.
    if modifiers.contains(KeyModifiers::CONTROL) {
        match event.code {
            KeyCode::Char('c') => return "ctrl+c".to_string(),
            KeyCode::Char('u') => return "ctrl+u".to_string(),
            KeyCode::Char('d') => return "ctrl+d".to_string(),
            _ => return String::new(),
        }
    }

    match event.code {
        KeyCode::Char(c) => c.to_string(),
        KeyCode::Enter => "enter".to_string(),
        KeyCode::Esc => "esc".to_string(),
        KeyCode::Backspace => "backspace".to_string(),
        KeyCode::Delete => "delete".to_string(),
        KeyCode::Up => "up".to_string(),
        KeyCode::Down => "down".to_string(),
        KeyCode::Left => "left".to_string(),
        KeyCode::Right => "right".to_string(),
        KeyCode::Tab => "tab".to_string(),
        KeyCode::Home => "home".to_string(),
        KeyCode::End => "end".to_string(),
        _ => String::new(),
    }
}
