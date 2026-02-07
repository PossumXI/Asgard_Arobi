#[cfg(target_os = "macos")]
pub mod shell {
    /// macOS shell integration
    ///
    /// macOS does not allow true shell replacement.
    /// The approach is a borderless, always-on-top window
    /// that hides the Dock and menu bar.

    use objc2_app_kit::*;
    use objc2_foundation::*;

    pub unsafe fn configure_as_shell(window: &NSWindow) {
        // Borderless full-screen, above dock
        window.setLevel(NSWindowLevel(25)); // kCGMainMenuWindowLevel + 1
        window.setCollectionBehavior(
            NSWindowCollectionBehavior::CanJoinAllSpaces
                | NSWindowCollectionBehavior::Stationary,
        );
        window.setOpaque(false);

        // Auto-hide the dock and menu bar
        let app = NSApplication::sharedApplication();
        app.setPresentationOptions(
            NSApplicationPresentationOptions::AutoHideDock
                | NSApplicationPresentationOptions::AutoHideMenuBar,
        );

        // Don't show in Dock
        app.setActivationPolicy(NSApplicationActivationPolicy::Accessory);
    }
}
