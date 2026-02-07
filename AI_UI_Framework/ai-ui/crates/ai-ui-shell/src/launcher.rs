use crate::app::Message;
use ai_ui_system::apps::AppEntry;
use iced::widget::{button, column, container, scrollable, text, text_input};
use iced::{Element, Length};

/// Render the app launcher overlay
pub fn view<'a>(
    query: &str,
    search_results: &'a [AppEntry],
    all_apps: &'a [AppEntry],
) -> Element<'a, Message> {
    let search_field = text_input("Search apps...", query)
        .on_input(Message::LauncherQueryChanged)
        .padding(12)
        .size(18);

    let apps_to_show = if query.is_empty() {
        all_apps
    } else {
        search_results
    };

    let app_list: Element<Message> = if apps_to_show.is_empty() {
        container(text("No apps found").size(14))
            .padding(20)
            .width(Length::Fill)
            .into()
    } else {
        let items: Vec<Element<Message>> = apps_to_show
            .iter()
            .take(20)
            .map(|app| {
                let exec = app.exec.clone();
                let desc = app.description.as_deref().unwrap_or("").to_string();

                let label = if desc.is_empty() {
                    column![text(&app.name).size(14)]
                } else {
                    column![
                        text(&app.name).size(14),
                        text(desc).size(11),
                    ]
                    .spacing(2)
                };

                button(
                    container(label)
                        .padding(8)
                        .width(Length::Fill),
                )
                .on_press(Message::LaunchApp(exec))
                .width(Length::Fill)
                .into()
            })
            .collect();

        scrollable(
            column(items)
                .spacing(4)
                .width(Length::Fill),
        )
        .height(Length::Fixed(400.0))
        .into()
    };

    let launcher = column![search_field, app_list]
        .spacing(8)
        .width(Length::Fixed(500.0));

    container(launcher)
        .width(Length::Fill)
        .height(Length::Fill)
        .center_x(Length::Fill)
        .padding(100)
        .into()
}
