use global_hotkey::hotkey::{Code, HotKey, Modifiers};
use global_hotkey::GlobalHotKeyManager;

/// Register the Ctrl+Space hotkey to toggle the AI command bar
pub fn register_command_bar_hotkey(
) -> Result<(GlobalHotKeyManager, HotKey), Box<dyn std::error::Error>> {
    let manager = GlobalHotKeyManager::new()?;
    let hotkey = HotKey::new(Some(Modifiers::CONTROL), Code::Space);
    manager.register(hotkey)?;
    Ok((manager, hotkey))
}

/// Register additional shell hotkeys
pub fn register_shell_hotkeys(
    manager: &GlobalHotKeyManager,
) -> Result<Vec<HotKey>, Box<dyn std::error::Error>> {
    let mut hotkeys = Vec::new();

    // Ctrl+Shift+A — Open app launcher
    let launcher = HotKey::new(
        Some(Modifiers::CONTROL | Modifiers::SHIFT),
        Code::KeyA,
    );
    manager.register(launcher)?;
    hotkeys.push(launcher);

    // Ctrl+Shift+S — Open settings
    let settings = HotKey::new(
        Some(Modifiers::CONTROL | Modifiers::SHIFT),
        Code::KeyS,
    );
    manager.register(settings)?;
    hotkeys.push(settings);

    Ok(hotkeys)
}
