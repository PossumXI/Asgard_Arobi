use crate::app::Message;
use iced::widget::{column, container, scrollable, text, text_input};
use iced::{Element, Length};

/// Render the AI command bar overlay
pub fn view<'a>(
    input: &str,
    response: &str,
    streaming: bool,
) -> Element<'a, Message> {
    let input_field = text_input("Ask AI anything... (launch apps, control system, ask questions)", input)
        .on_input(Message::CommandInputChanged)
        .on_submit(Message::ExecuteCommand)
        .padding(12)
        .size(18);

    let response_area: Element<'a, Message> = if !response.is_empty() {
        let status = if streaming { " (streaming...)" } else { "" };
        let response_text = format!("{}{}", response, status);
        container(
            scrollable(
                container(text(response_text).size(14))
                    .padding(16)
                    .width(Length::Fill),
            )
            .height(Length::Fill),
        )
        .width(Length::Fill)
        .height(Length::Fixed(300.0))
        .into()
    } else {
        container(
            text("Type a command and press Enter").size(14),
        )
        .padding(16)
        .width(Length::Fill)
        .into()
    };

    let bar = column![input_field, response_area]
        .spacing(8)
        .width(Length::Fixed(700.0));

    container(bar)
        .width(Length::Fill)
        .height(Length::Fill)
        .center_x(Length::Fill)
        .padding(100)
        .into()
}
