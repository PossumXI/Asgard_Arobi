use crate::app::Message;
use ai_ui_system::status::SystemStatus;
use iced::widget::{button, container, row, text};
use iced::{Element, Length};

#[derive(Debug, Clone, Default)]
pub struct TaskbarState {
    pub show_system_tray: bool,
    pub active_windows: Vec<String>,
}

#[derive(Debug, Clone)]
pub enum TaskbarAction {
    ToggleSystemTray,
    OpenCommandBar,
    OpenLauncher,
}

pub fn handle_action(state: &mut TaskbarState, action: TaskbarAction) {
    match action {
        TaskbarAction::ToggleSystemTray => {
            state.show_system_tray = !state.show_system_tray;
        }
        TaskbarAction::OpenCommandBar | TaskbarAction::OpenLauncher => {
            // Handled by parent
        }
    }
}

/// Render the taskbar at the bottom of the screen
pub fn view<'a>(
    _state: &'a TaskbarState,
    status: &'a SystemStatus,
) -> Element<'a, Message> {
    // Left side: launcher button + AI button
    let launcher_btn = button(text("Apps").size(13))
        .on_press(Message::ToggleLauncher)
        .padding(6);

    let ai_btn = button(text("AI").size(13))
        .on_press(Message::ToggleCommandBar)
        .padding(6);

    let left = row![launcher_btn, ai_btn].spacing(4);

    // Center: spacer
    let center = container(text("").size(1)).width(Length::Fill);

    // Right side: system status
    let time_text = text(&status.time).size(13);

    let battery_text = match status.battery_percent {
        Some(pct) => {
            let icon = if status.battery_charging { "+" } else { "" };
            format!("{}{}%", icon, pct as u32)
        }
        None => String::new(),
    };
    let battery = text(battery_text).size(13);

    let memory = text(format!(
        "{:.1}/{:.1}GB",
        status.memory_used_gb, status.memory_total_gb
    ))
    .size(13);

    let wifi = text(
        status
            .wifi_ssid
            .as_deref()
            .unwrap_or(if status.wifi_connected {
                "WiFi"
            } else {
                "No WiFi"
            }),
    )
    .size(13);

    let right = row![wifi, memory, battery, time_text].spacing(16);

    let bar = row![left, center, right]
        .spacing(8)
        .padding(6)
        .width(Length::Fill);

    container(bar)
        .width(Length::Fill)
        .height(Length::Fixed(40.0))
        .into()
}
