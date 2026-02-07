mod app;
mod command_bar;
mod launcher;
mod platform;
mod taskbar;

fn main() -> iced::Result {
    // Initialize logging
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::DEBUG)
        .init();

    tracing::info!("Starting AI-UI Desktop Shell");

    iced::application(app::AiUiShell::new, app::AiUiShell::update, app::AiUiShell::view)
        .title("AI-UI Shell")
        .subscription(app::AiUiShell::subscription)
        .theme(app::AiUiShell::theme)
        .window(iced::window::Settings {
            size: iced::Size::new(1920.0, 1080.0),
            decorations: false,
            transparent: true,
            ..Default::default()
        })
        .antialiasing(true)
        .run()
}
