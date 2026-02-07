use crate::command_bar;
use crate::launcher;
use crate::taskbar;

use ai_ui_system::apps::AppEntry;
use ai_ui_system::status::SystemStatus;

use iced::widget::{column, container, text};
use iced::{Element, Length, Subscription, Task, Theme};

/// Main application state
pub struct AiUiShell {
    pub command_input: String,
    pub ai_response: String,
    pub ai_streaming: bool,
    pub installed_apps: Vec<AppEntry>,
    pub search_results: Vec<AppEntry>,
    pub taskbar_state: taskbar::TaskbarState,
    pub system_status: SystemStatus,
    pub is_command_bar_visible: bool,
    pub is_launcher_visible: bool,
    pub launcher_query: String,
    pub api_key: Option<String>,
}

#[derive(Debug, Clone)]
pub enum Message {
    // AI Command Bar
    CommandInputChanged(String),
    ExecuteCommand,
    AiResponseChunk(String),
    AiResponseComplete(String),
    AiError(String),

    // App Launcher
    ToggleCommandBar,
    ToggleLauncher,
    LauncherQueryChanged(String),
    LaunchApp(String),
    AppsLoaded(Vec<AppEntry>),

    // System
    SystemStatusUpdate(SystemStatus),
    Tick,

    // Taskbar
    TaskbarAction(taskbar::TaskbarAction),
}

impl AiUiShell {
    pub fn new() -> (Self, Task<Message>) {
        let api_key = ai_ui_ai::load_api_key();

        let app = Self {
            command_input: String::new(),
            ai_response: String::new(),
            ai_streaming: false,
            installed_apps: Vec::new(),
            search_results: Vec::new(),
            taskbar_state: taskbar::TaskbarState::default(),
            system_status: SystemStatus::default(),
            is_command_bar_visible: false,
            is_launcher_visible: false,
            launcher_query: String::new(),
            api_key,
        };

        // Load installed apps on startup
        let init_cmd = Task::perform(ai_ui_system::apps::enumerate_apps(), |result| {
            Message::AppsLoaded(result.unwrap_or_default())
        });

        (app, init_cmd)
    }

    pub fn theme(&self) -> Theme {
        Theme::Dark
    }

    pub fn update(&mut self, message: Message) -> Task<Message> {
        match message {
            Message::CommandInputChanged(input) => {
                self.command_input = input;
                Task::none()
            }
            Message::ExecuteCommand => {
                let prompt = self.command_input.clone();
                if prompt.is_empty() {
                    return Task::none();
                }

                self.ai_response.clear();
                self.ai_streaming = true;

                let api_key = self.api_key.clone();
                Task::perform(
                    async move {
                        ai_ui_ai::generate_response(&prompt, api_key.as_deref()).await
                    },
                    |result| match result {
                        Ok(response) => Message::AiResponseComplete(response),
                        Err(e) => Message::AiError(e.to_string()),
                    },
                )
            }
            Message::AiResponseChunk(chunk) => {
                self.ai_response.push_str(&chunk);
                Task::none()
            }
            Message::AiResponseComplete(response) => {
                self.ai_response = response;
                self.ai_streaming = false;
                Task::none()
            }
            Message::AiError(err) => {
                self.ai_response = format!("Error: {}", err);
                self.ai_streaming = false;
                Task::none()
            }
            Message::ToggleCommandBar => {
                self.is_command_bar_visible = !self.is_command_bar_visible;
                if !self.is_command_bar_visible {
                    self.command_input.clear();
                    self.ai_response.clear();
                }
                Task::none()
            }
            Message::ToggleLauncher => {
                self.is_launcher_visible = !self.is_launcher_visible;
                if !self.is_launcher_visible {
                    self.launcher_query.clear();
                    self.search_results.clear();
                }
                Task::none()
            }
            Message::LauncherQueryChanged(query) => {
                self.search_results =
                    ai_ui_system::apps::fuzzy_search(&self.installed_apps, &query);
                self.launcher_query = query;
                Task::none()
            }
            Message::LaunchApp(exec_path) => {
                #[cfg(windows)]
                {
                    let _ = std::process::Command::new("cmd")
                        .args(["/C", "start", "", &exec_path])
                        .spawn();
                }
                #[cfg(not(windows))]
                {
                    let exec = exec_path.split_whitespace().next().unwrap_or(&exec_path);
                    let _ = std::process::Command::new(exec).spawn();
                }
                self.is_launcher_visible = false;
                Task::none()
            }
            Message::AppsLoaded(apps) => {
                tracing::info!("Loaded {} installed apps", apps.len());
                self.installed_apps = apps;
                Task::none()
            }
            Message::SystemStatusUpdate(status) => {
                self.system_status = status;
                Task::none()
            }
            Message::Tick => {
                Task::perform(ai_ui_system::status::read_status(), |status| {
                    Message::SystemStatusUpdate(status)
                })
            }
            Message::TaskbarAction(action) => {
                taskbar::handle_action(&mut self.taskbar_state, action);
                Task::none()
            }
        }
    }

    pub fn view(&self) -> Element<'_, Message> {
        let taskbar_view = taskbar::view(&self.taskbar_state, &self.system_status);

        let main_content: Element<Message> = if self.is_command_bar_visible {
            command_bar::view(&self.command_input, &self.ai_response, self.ai_streaming)
        } else if self.is_launcher_visible {
            launcher::view(
                &self.launcher_query,
                &self.search_results,
                &self.installed_apps,
            )
        } else {
            // Desktop area
            container(
                column![
                    text("AI-UI Desktop Shell").size(32),
                    text("Press Ctrl+Space for AI Command Bar").size(16),
                    text("Press Ctrl+Shift+A for App Launcher").size(16),
                ]
                .spacing(10)
                .align_x(iced::Alignment::Center),
            )
            .width(Length::Fill)
            .height(Length::Fill)
            .center_x(Length::Fill)
            .center_y(Length::Fill)
            .into()
        };

        let desktop = container(main_content)
            .width(Length::Fill)
            .height(Length::Fill);

        column![desktop, taskbar_view]
            .width(Length::Fill)
            .height(Length::Fill)
            .into()
    }

    pub fn subscription(&self) -> Subscription<Message> {
        iced::time::every(std::time::Duration::from_secs(1)).map(|_| Message::Tick)
    }
}
