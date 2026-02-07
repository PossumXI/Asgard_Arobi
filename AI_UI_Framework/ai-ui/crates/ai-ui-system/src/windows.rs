/// Window management utilities per platform

#[derive(Debug, Clone)]
pub struct WindowInfo {
    pub title: String,
    pub is_visible: bool,
    #[cfg(windows)]
    pub hwnd: usize,
}

/// List all visible windows on the system
#[cfg(windows)]
pub fn list_windows() -> Vec<WindowInfo> {
    use windows::Win32::Foundation::*;
    use windows::Win32::UI::WindowsAndMessaging::*;

    let mut result: Vec<WindowInfo> = Vec::new();

    unsafe {
        let _ = EnumWindows(
            Some(enum_callback),
            LPARAM(&mut result as *mut Vec<WindowInfo> as isize),
        );
    }

    result
}

#[cfg(windows)]
unsafe extern "system" fn enum_callback(
    hwnd: windows::Win32::Foundation::HWND,
    lparam: windows::Win32::Foundation::LPARAM,
) -> windows::core::BOOL {
    use windows::Win32::UI::WindowsAndMessaging::*;

    let windows_list = &mut *(lparam.0 as *mut Vec<WindowInfo>);

    if IsWindowVisible(hwnd).as_bool() {
        let mut title = [0u16; 256];
        let len = GetWindowTextW(hwnd, &mut title);
        if len > 0 {
            let title = String::from_utf16_lossy(&title[..len as usize]);
            if !title.is_empty() {
                windows_list.push(WindowInfo {
                    title,
                    is_visible: true,
                    hwnd: hwnd.0 as usize,
                });
            }
        }
    }

    true.into()
}

#[cfg(not(windows))]
pub fn list_windows() -> Vec<WindowInfo> {
    Vec::new()
}
