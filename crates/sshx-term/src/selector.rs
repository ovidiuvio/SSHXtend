use anyhow::Result;
use crossterm::{
    event::{self, Event, KeyCode, KeyEventKind},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{
    backend::CrosstermBackend,
    layout::{Alignment, Constraint, Direction, Layout},
    style::{Color, Modifier, Style},
    widgets::{
        Block, Borders, Paragraph, Table, Row, Cell
    },
    Terminal,
};
use std::io;
use sshx_core::Sid;

use crate::client::{ShellInfo, TerminalStatus};

pub async fn show_terminal_selector(shells: &[ShellInfo]) -> Result<Sid> {
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let result = run_selector(&mut terminal, shells).await;

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    result
}

async fn run_selector(
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    shells: &[ShellInfo],
) -> Result<Sid> {
    let mut selected = 0;

    loop {
        terminal.draw(|f| {
            let size = f.size();
            
            // Simple layout - just table and footer
            let chunks = Layout::default()
                .direction(Direction::Vertical)
                .constraints([
                    Constraint::Min(0),    // Table
                    Constraint::Length(3), // Footer
                ])
                .split(size);

            // Simple table with essential info
            let header_cells = ["#", "ID", "Title/Process", "Size", "Activity", "Status"]
                .iter()
                .map(|h| Cell::from(*h).style(Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD)));
            
            let header_row = Row::new(header_cells).height(1);

            let mut rows: Vec<Row> = shells
                .iter()
                .enumerate()
                .map(|(i, shell)| {
                    let style = if i == selected {
                        Style::default().bg(Color::Blue).fg(Color::White).add_modifier(Modifier::BOLD)
                    } else {
                        Style::default().fg(Color::White)
                    };

                    let status_style = match shell.status {
                        TerminalStatus::Active => Style::default().fg(Color::Green),
                        TerminalStatus::Busy => Style::default().fg(Color::Yellow),
                        TerminalStatus::Idle => Style::default().fg(Color::Gray),
                        TerminalStatus::Focused => Style::default().fg(Color::Cyan),
                    };

                    let status_text = match shell.status {
                        TerminalStatus::Active => "Active",
                        TerminalStatus::Busy => "Busy",
                        TerminalStatus::Idle => "Idle",
                        TerminalStatus::Focused => "Focused",
                    };

                    let activity_text = format_duration(shell.last_activity.elapsed());

                    Row::new([
                        Cell::from((i + 1).to_string()),
                        Cell::from(shell.id.0.to_string()),
                        Cell::from(shell.title.clone()),
                        Cell::from(format!("{}×{}", shell.winsize.cols, shell.winsize.rows)),
                        Cell::from(activity_text),
                        Cell::from(status_text).style(status_style),
                    ]).style(style)
                })
                .collect();

            // Add "Create New" option
            let create_style = if selected == shells.len() {
                Style::default().bg(Color::Blue).fg(Color::White).add_modifier(Modifier::BOLD)
            } else {
                Style::default().fg(Color::Green)
            };

            rows.push(
                Row::new([
                    Cell::from("n"),
                    Cell::from("NEW"),
                    Cell::from("Create new terminal"),
                    Cell::from("-"),
                    Cell::from("-"),
                    Cell::from("Ready").style(Style::default().fg(Color::Green)),
                ]).style(create_style)
            );

            let table = Table::new(rows, [
                Constraint::Length(3),  // #
                Constraint::Length(4),  // ID
                Constraint::Min(30),    // Title
                Constraint::Length(8),  // Size
                Constraint::Length(10), // Activity
                Constraint::Length(10), // Status
            ])
            .header(header_row)
            .block(Block::default()
                .title("Select Terminal")
                .borders(Borders::ALL)
                .border_style(Style::default().fg(Color::White)))
            .column_spacing(1);

            f.render_widget(table, chunks[0]);

            // Simple footer
            let footer_text = format!(
                "Use ↑↓ to navigate, ENTER to select, 'n' for new terminal, 'q' to quit | {} terminals available",
                shells.len()
            );

            let footer = Paragraph::new(footer_text)
                .block(Block::default()
                    .borders(Borders::ALL)
                    .border_style(Style::default().fg(Color::Gray)))
                .alignment(Alignment::Center)
                .style(Style::default().fg(Color::White));

            f.render_widget(footer, chunks[1]);
        })?;

        // Handle input
        if event::poll(std::time::Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if key.kind == KeyEventKind::Press {
                    match key.code {
                        KeyCode::Up => {
                            if selected > 0 {
                                selected -= 1;
                            }
                        }
                        KeyCode::Down => {
                            if selected < shells.len() {
                                selected += 1;
                            }
                        }
                        KeyCode::Char('q') | KeyCode::Esc => {
                            std::process::exit(0);
                        }
                        KeyCode::Char(c) if c.is_ascii_digit() => {
                            let num = c.to_digit(10).unwrap() as usize;
                            if num > 0 && num <= shells.len() {
                                selected = num - 1;
                            }
                        }
                        KeyCode::Char('n') => {
                            // Jump to "Create new terminal" option
                            selected = shells.len();
                        }
                        KeyCode::Char('r') => {
                            // Refresh - just redraw for now
                        }
                        KeyCode::Enter => {
                            if selected < shells.len() {
                                return Ok(shells[selected].id);
                            } else {
                                // Create new terminal
                                return Ok(sshx_core::Sid(u32::MAX));
                            }
                        }
                        _ => {}
                    }
                }
            }
        }
    }
}

fn format_duration(duration: std::time::Duration) -> String {
    let total_seconds = duration.as_secs();
    let hours = total_seconds / 3600;
    let minutes = (total_seconds % 3600) / 60;
    let seconds = total_seconds % 60;

    if hours > 0 {
        format!("{}h{}m", hours, minutes)
    } else if minutes > 0 {
        format!("{}m{}s", minutes, seconds)
    } else {
        format!("{}s", seconds)
    }
}

