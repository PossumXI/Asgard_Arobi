use nucleo_matcher::pattern::{Atom, AtomKind, CaseMatching, Normalization};
use nucleo_matcher::{Config, Matcher, Utf32Str};
use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct AppEntry {
    pub name: String,
    pub exec: String,
    pub icon_path: Option<PathBuf>,
    pub description: Option<String>,
}

/// Enumerate installed applications (cross-platform)
pub async fn enumerate_apps() -> anyhow::Result<Vec<AppEntry>> {
    let mut apps = Vec::new();

    #[cfg(target_os = "linux")]
    {
        use freedesktop_desktop_entry::{default_paths, get_languages_from_env, Iter};

        let locales = get_languages_from_env();
        for entry in Iter::new(default_paths()).entries(Some(&locales)) {
            if let (Some(name), Some(exec)) = (entry.name(&locales), entry.exec()) {
                apps.push(AppEntry {
                    name: name.to_string(),
                    exec: exec.to_string(),
                    icon_path: entry.icon().map(PathBuf::from),
                    description: entry.comment(&locales).map(|c| c.to_string()),
                });
            }
        }
    }

    #[cfg(windows)]
    {
        use winreg::enums::*;
        use winreg::RegKey;

        let hklm = RegKey::predef(HKEY_LOCAL_MACHINE);
        if let Ok(uninstall) = hklm.open_subkey(
            r"SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
        ) {
            for key_name in uninstall.enum_keys().filter_map(|k| k.ok()) {
                if let Ok(subkey) = uninstall.open_subkey(&key_name) {
                    if let Ok(name) = subkey.get_value::<String, _>("DisplayName") {
                        let exec = subkey
                            .get_value::<String, _>("InstallLocation")
                            .unwrap_or_default();
                        apps.push(AppEntry {
                            name,
                            exec,
                            icon_path: None,
                            description: None,
                        });
                    }
                }
            }
        }

        // Also scan Start Menu .lnk files for better coverage
        if let Some(start_menu) = dirs::data_dir() {
            let start_menu_path = start_menu
                .parent()
                .unwrap_or(&start_menu)
                .join("Microsoft")
                .join("Windows")
                .join("Start Menu")
                .join("Programs");
            scan_start_menu(&start_menu_path, &mut apps);
        }

        // Also scan common Start Menu
        let common_start = PathBuf::from(
            r"C:\ProgramData\Microsoft\Windows\Start Menu\Programs",
        );
        if common_start.exists() {
            scan_start_menu(&common_start, &mut apps);
        }
    }

    #[cfg(target_os = "macos")]
    {
        for dir in &["/Applications", "/System/Applications"] {
            if let Ok(entries) = std::fs::read_dir(dir) {
                for entry in entries.flatten() {
                    let path = entry.path();
                    if path.extension().map_or(false, |e| e == "app") {
                        let name = path
                            .file_stem()
                            .map(|s| s.to_string_lossy().to_string())
                            .unwrap_or_default();
                        apps.push(AppEntry {
                            name,
                            exec: path.to_string_lossy().to_string(),
                            icon_path: None,
                            description: None,
                        });
                    }
                }
            }
        }
    }

    Ok(apps)
}

#[cfg(windows)]
fn scan_start_menu(dir: &std::path::Path, apps: &mut Vec<AppEntry>) {
    if let Ok(entries) = std::fs::read_dir(dir) {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                scan_start_menu(&path, apps);
            } else if path.extension().map_or(false, |e| e == "lnk") {
                let name = path
                    .file_stem()
                    .map(|s| s.to_string_lossy().to_string())
                    .unwrap_or_default();
                if !name.is_empty() {
                    apps.push(AppEntry {
                        name,
                        exec: path.to_string_lossy().to_string(),
                        icon_path: None,
                        description: None,
                    });
                }
            }
        }
    }
}

/// Fuzzy search installed applications using nucleo (6x faster than skim)
pub fn fuzzy_search(apps: &[AppEntry], query: &str) -> Vec<AppEntry> {
    if query.is_empty() {
        return apps.to_vec();
    }

    let mut matcher = Matcher::new(Config::DEFAULT);
    let atom = Atom::new(
        query,
        CaseMatching::Ignore,
        Normalization::Smart,
        AtomKind::Fuzzy,
        false,
    );

    let mut results: Vec<(i32, &AppEntry)> = apps
        .iter()
        .filter_map(|app| {
            let mut buf = Vec::new();
            let haystack = Utf32Str::new(&app.name, &mut buf);
            atom.score(haystack, &mut matcher)
                .map(|score| (score as i32, app))
        })
        .collect();

    results.sort_by(|a, b| b.0.cmp(&a.0));
    results.into_iter().map(|(_, app)| app.clone()).collect()
}

/// Launch an application by name
pub async fn launch_by_name(name: &str) -> anyhow::Result<()> {
    let apps = enumerate_apps().await?;

    let results = fuzzy_search(&apps, name);
    let app = results
        .first()
        .ok_or_else(|| anyhow::anyhow!("No app found matching: {}", name))?;

    tracing::info!("Launching app: {} ({})", app.name, app.exec);

    #[cfg(windows)]
    {
        if app.exec.ends_with(".lnk") {
            std::process::Command::new("cmd")
                .args(["/C", "start", "", &app.exec])
                .spawn()?;
        } else if !app.exec.is_empty() {
            std::process::Command::new(&app.exec).spawn()?;
        } else {
            anyhow::bail!("No executable path for: {}", app.name);
        }
    }

    #[cfg(not(windows))]
    {
        let exec = app.exec.split_whitespace().next().unwrap_or(&app.exec);
        std::process::Command::new(exec).spawn()?;
    }

    Ok(())
}
