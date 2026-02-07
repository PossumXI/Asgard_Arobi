#[cfg(target_os = "linux")]
pub mod shell {
    /// Linux shell integration
    ///
    /// Two approaches:
    /// 1. Layer-shell panel on existing compositors (Sway, Hyprland, COSMIC)
    ///    - Uses wayland-client + wlr-layer-shell-unstable-v1
    ///    - Creates overlay surfaces anchored to screen edges
    ///    - No need to handle DRM or display management
    ///
    /// 2. Full Wayland compositor with smithay (advanced)
    ///    - Handles Wayland protocol, DRM/GBM, libinput
    ///    - Reference: COSMIC's cosmic-comp

    /// Check if running under Wayland
    pub fn is_wayland() -> bool {
        std::env::var("WAYLAND_DISPLAY").is_ok()
    }

    /// Check if running under X11
    pub fn is_x11() -> bool {
        std::env::var("DISPLAY").is_ok()
    }

    /// Get the current desktop session type
    pub fn session_type() -> String {
        std::env::var("XDG_SESSION_TYPE").unwrap_or_else(|_| "unknown".into())
    }
}
