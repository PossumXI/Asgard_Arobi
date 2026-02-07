#[cfg(windows)]
pub mod shell {
    use windows::Win32::Foundation::*;
    use windows::Win32::UI::Shell::*;
    use windows::Win32::UI::WindowsAndMessaging::*;

    /// Register the window as an AppBar (reserves screen space like a taskbar)
    pub unsafe fn register_appbar(hwnd: HWND, height: u32) {
        let mut abd = APPBARDATA {
            cbSize: std::mem::size_of::<APPBARDATA>() as u32,
            hWnd: hwnd,
            ..Default::default()
        };

        // Register
        SHAppBarMessage(ABM_NEW, &mut abd);

        // Set position (bottom of screen)
        abd.uEdge = ABE_BOTTOM as u32;
        abd.rc = RECT {
            left: 0,
            top: GetSystemMetrics(SM_CYSCREEN) - height as i32,
            right: GetSystemMetrics(SM_CXSCREEN),
            bottom: GetSystemMetrics(SM_CYSCREEN),
        };

        SHAppBarMessage(ABM_QUERYPOS, &mut abd);
        SHAppBarMessage(ABM_SETPOS, &mut abd);
    }

    /// Enumerate all visible windows for the task list
    pub fn list_windows() -> Vec<(String, HWND)> {
        let mut windows_list = Vec::new();
        unsafe {
            let _ = EnumWindows(
                Some(enum_callback),
                LPARAM(&mut windows_list as *mut Vec<(String, HWND)> as isize),
            );
        }
        windows_list
    }

    unsafe extern "system" fn enum_callback(
        hwnd: HWND,
        lparam: LPARAM,
    ) -> windows::core::BOOL {
        let windows_list = &mut *(lparam.0 as *mut Vec<(String, HWND)>);
        if IsWindowVisible(hwnd).as_bool() {
            let mut title = [0u16; 256];
            let len = GetWindowTextW(hwnd, &mut title);
            if len > 0 {
                let title = String::from_utf16_lossy(&title[..len as usize]);
                if !title.is_empty() {
                    windows_list.push((title, hwnd));
                }
            }
        }
        true.into()
    }

    /// Set this application as the Windows shell (replaces explorer.exe)
    pub fn set_as_shell(exe_path: &str) -> std::io::Result<()> {
        let hkcu = winreg::RegKey::predef(winreg::enums::HKEY_CURRENT_USER);
        let (key, _) = hkcu.create_subkey(
            r"Software\Microsoft\Windows NT\CurrentVersion\Winlogon",
        )?;
        key.set_value("Shell", &exe_path)?;
        Ok(())
    }

    /// Restore explorer.exe as the Windows shell
    pub fn restore_default_shell() -> std::io::Result<()> {
        set_as_shell("explorer.exe")
    }
}
