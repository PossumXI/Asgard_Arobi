use tracing;

/// MCP (Model Context Protocol) client integration
///
/// Uses rmcp (official Rust SDK) for connecting to MCP servers.
/// MCP servers provide tools that extend Claude's capabilities
/// within the desktop shell.

/// MCP server configuration
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct McpServerConfig {
    pub name: String,
    pub command: String,
    pub args: Vec<String>,
    pub enabled: bool,
}

/// Available MCP tool info
#[derive(Debug, Clone)]
pub struct McpTool {
    pub name: String,
    pub description: String,
    pub server: String,
}

/// Connect to an MCP server via child process transport
///
/// Note: rmcp integration requires the rmcp crate which has complex
/// build requirements. This provides the interface - enable full
/// MCP support by adding rmcp to dependencies when needed.
pub async fn connect_mcp_server(
    command: &str,
    args: &[&str],
) -> Result<Vec<McpTool>, Box<dyn std::error::Error>> {
    tracing::info!("Connecting to MCP server: {} {:?}", command, args);

    // Verify the command exists
    let output = tokio::process::Command::new(command)
        .args(args)
        .arg("--version")
        .output()
        .await;

    match output {
        Ok(out) => {
            tracing::info!(
                "MCP server responded: {}",
                String::from_utf8_lossy(&out.stdout)
            );
        }
        Err(e) => {
            tracing::warn!("MCP server not available: {}", e);
            return Err(Box::new(e));
        }
    }

    // Placeholder for full rmcp integration
    Ok(Vec::new())
}

/// Load MCP server configurations from config file
pub fn load_mcp_configs() -> Vec<McpServerConfig> {
    let config_dir = match dirs::config_dir() {
        Some(d) => d.join("ai-ui"),
        None => return Vec::new(),
    };

    let config_path = config_dir.join("mcp-servers.toml");
    let content = match std::fs::read_to_string(&config_path) {
        Ok(c) => c,
        Err(_) => return Vec::new(),
    };

    #[derive(serde::Deserialize)]
    struct McpConfig {
        #[serde(default)]
        servers: Vec<McpServerConfig>,
    }

    match toml::from_str::<McpConfig>(&content) {
        Ok(config) => config.servers,
        Err(e) => {
            tracing::warn!("Failed to parse MCP config: {}", e);
            Vec::new()
        }
    }
}
