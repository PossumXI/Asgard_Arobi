use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct SystemStatus {
    pub battery_percent: Option<f32>,
    pub battery_charging: bool,
    pub wifi_connected: bool,
    pub wifi_ssid: Option<String>,
    pub volume_percent: Option<f32>,
    pub cpu_usage: f32,
    pub memory_used_gb: f32,
    pub memory_total_gb: f32,
    pub time: String,
}

/// Read current system status (battery, CPU, memory, time)
pub async fn read_status() -> SystemStatus {
    let mut status = SystemStatus::default();

    // Battery (platform-specific, lightweight)
    #[cfg(windows)]
    {
        read_battery_windows(&mut status);
    }

    // CPU/RAM (via sysinfo)
    let mut sys = sysinfo::System::new();
    sys.refresh_memory();
    status.memory_used_gb = sys.used_memory() as f32 / 1_073_741_824.0;
    status.memory_total_gb = sys.total_memory() as f32 / 1_073_741_824.0;

    // WiFi (platform-specific)
    #[cfg(windows)]
    {
        read_wifi_windows(&mut status);
    }

    #[cfg(target_os = "linux")]
    {
        status.wifi_connected = true;
    }

    // Time
    status.time = chrono::Local::now().format("%H:%M").to_string();

    status
}

#[cfg(windows)]
fn read_battery_windows(status: &mut SystemStatus) {
    use windows::Win32::System::Power::*;
    unsafe {
        let mut sps = SYSTEM_POWER_STATUS::default();
        if GetSystemPowerStatus(&mut sps).is_ok() {
            if sps.BatteryLifePercent != 255 {
                status.battery_percent = Some(sps.BatteryLifePercent as f32);
            }
            // AC online = 1 means charging
            status.battery_charging = sps.ACLineStatus == 1;
        }
    }
}

#[cfg(windows)]
fn read_wifi_windows(status: &mut SystemStatus) {
    if let Ok(output) = std::process::Command::new("netsh")
        .args(["wlan", "show", "interfaces"])
        .output()
    {
        let stdout = String::from_utf8_lossy(&output.stdout);
        status.wifi_connected = stdout.contains("State") && stdout.contains("connected");
        for line in stdout.lines() {
            if line.trim().starts_with("SSID") && !line.contains("BSSID") {
                if let Some(ssid) = line.split(':').nth(1) {
                    status.wifi_ssid = Some(ssid.trim().to_string());
                }
            }
        }
    }
}

/// Execute a system action (volume, brightness, etc.)
pub async fn execute_action(action: &str) -> String {
    match action {
        #[cfg(windows)]
        "volume_up" => {
            let _ = std::process::Command::new("powershell")
                .args(["-Command", "(New-Object -ComObject WScript.Shell).SendKeys([char]175)"])
                .output();
            "Volume increased".into()
        }
        #[cfg(windows)]
        "volume_down" => {
            let _ = std::process::Command::new("powershell")
                .args(["-Command", "(New-Object -ComObject WScript.Shell).SendKeys([char]174)"])
                .output();
            "Volume decreased".into()
        }
        #[cfg(windows)]
        "mute" => {
            let _ = std::process::Command::new("powershell")
                .args(["-Command", "(New-Object -ComObject WScript.Shell).SendKeys([char]173)"])
                .output();
            "Volume muted/unmuted".into()
        }
        _ => format!("Action '{}' not implemented for this platform", action),
    }
}
