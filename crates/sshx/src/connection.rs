//! Connection management with automatic gRPC→WebSocket fallback.
//!
//! This module provides high-level connection management that automatically
//! attempts gRPC first, then falls back to WebSocket if gRPC fails.

use anyhow::{Context, Result};
use sshx_core::proto::OpenRequest;
use std::time::Duration;
use tokio::time::timeout;
use tracing::{debug, info, warn};

use crate::transport::{grpc_to_websocket_url, GrpcTransport, SshxTransport, WebSocketTransport};

/// Connection timeout for gRPC connectivity test.
pub const GRPC_TIMEOUT: Duration = Duration::from_secs(3);

/// Connection timeout for WebSocket fallback.
pub const WEBSOCKET_TIMEOUT: Duration = Duration::from_secs(5);

/// Connection strategy configuration.
#[derive(Debug, Clone)]
pub struct ConnectionConfig {
    /// Whether to enable verbose error reporting during fallback attempts.
    pub verbose_errors: bool,
    /// Custom timeout for gRPC connection attempts.
    pub grpc_timeout: Option<Duration>,
    /// Custom timeout for WebSocket connection attempts.
    pub websocket_timeout: Option<Duration>,
}

impl Default for ConnectionConfig {
    fn default() -> Self {
        Self {
            verbose_errors: false,
            grpc_timeout: None,
            websocket_timeout: None,
        }
    }
}

/// Result of a connection attempt with transport details.
#[derive(Debug)]
pub struct ConnectionResult {
    /// The established transport connection.
    pub transport: Box<dyn SshxTransport>,
    /// The connection method that was used.
    pub method: ConnectionMethod,
}

/// The method used to establish the connection.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ConnectionMethod {
    /// Direct gRPC connection succeeded.
    Grpc,
    /// WebSocket fallback was used after gRPC failed.
    WebSocketFallback,
}

/// Connect to an sshx server with automatic gRPC→WebSocket fallback.
///
/// This function attempts to connect using gRPC first, and if that fails,
/// automatically falls back to WebSocket. The connection method is determined
/// by testing actual connectivity to the server.
///
/// # Arguments
/// * `origin` - The server URL to connect to (e.g., "https://sshx.io")
/// * `session_name` - Name for the session (used for WebSocket endpoint)
/// * `config` - Connection configuration options
///
/// # Returns
/// A `ConnectionResult` containing the transport and connection method used
///
/// # Behavior
/// 1. Attempts gRPC connection with 3-second timeout
/// 2. Tests gRPC connectivity by making an actual `Open` call
/// 3. If gRPC fails, converts URL and attempts WebSocket connection
/// 4. Returns the first successful connection method
///
/// # Examples
/// ```no_run
/// # use sshx::connection::{connect_with_fallback, ConnectionConfig, ConnectionMethod};
/// # #[tokio::main]
/// # async fn main() -> anyhow::Result<()> {
/// let config = ConnectionConfig::default();
/// let result = connect_with_fallback("https://sshx.io", "my-session", config).await?;
/// 
/// match result.method {
///     ConnectionMethod::Grpc => println!("Connected via gRPC"),
///     ConnectionMethod::WebSocketFallback => println!("Connected via WebSocket fallback"),
/// }
/// # Ok(())
/// # }
/// ```
pub async fn connect_with_fallback(
    origin: &str,
    session_name: &str,
    config: ConnectionConfig,
) -> Result<ConnectionResult> {
    debug!(%origin, %session_name, "attempting connection with fallback");

    // First, try gRPC connection
    match try_grpc_connection(origin, &config).await {
        Ok(transport) => {
            info!(%origin, "gRPC connection successful");
            return Ok(ConnectionResult {
                transport,
                method: ConnectionMethod::Grpc,
            });
        }
        Err(e) => {
            if config.verbose_errors {
                warn!(%origin, error = %e, "gRPC connection failed, attempting WebSocket fallback");
            } else {
                debug!(%origin, error = %e, "gRPC connection failed, attempting WebSocket fallback");
            }
        }
    }

    // If gRPC failed, try WebSocket fallback
    match try_websocket_connection(origin, session_name, &config).await {
        Ok(transport) => {
            info!(%origin, "WebSocket fallback connection successful");
            Ok(ConnectionResult {
                transport,
                method: ConnectionMethod::WebSocketFallback,
            })
        }
        Err(e) => {
            if config.verbose_errors {
                warn!(%origin, error = %e, "WebSocket fallback also failed");
            }
            Err(e).context(format!(
                "Both gRPC and WebSocket connections failed for {}",
                origin
            ))
        }
    }
}

/// Attempt to establish a gRPC connection and test its connectivity.
///
/// This function not only connects to the gRPC endpoint but also performs
/// a real connectivity test by attempting an `Open` call to ensure the
/// connection is actually working.
async fn try_grpc_connection(
    origin: &str,
    config: &ConnectionConfig,
) -> Result<Box<dyn SshxTransport>> {
    let timeout_duration = config.grpc_timeout.unwrap_or(GRPC_TIMEOUT);
    
    debug!(%origin, timeout_ms = timeout_duration.as_millis(), "attempting gRPC connection");

    // First, test connectivity with a separate connection to avoid consuming the main transport
    debug!(%origin, "testing gRPC connectivity with Open call");
    let mut test_transport = timeout(timeout_duration, GrpcTransport::connect(origin))
        .await
        .context("gRPC connection timed out")?
        .context("gRPC connection failed")?;

    let test_request = OpenRequest {
        origin: origin.to_string(),
        encrypted_zeros: vec![0u8; 32].into(), // Dummy encrypted zeros for connectivity test
        name: "connectivity-test".to_string(),
        write_password_hash: None,
    };

    // Test the connection with the dummy request
    let test_result = timeout(timeout_duration, test_transport.open(test_request)).await;
    
    match test_result {
        Ok(Ok(_)) => {
            // Open succeeded - connection is definitely working
            debug!(%origin, "gRPC connectivity test succeeded");
        }
        Ok(Err(e)) => {
            // Open failed with an error - gRPC is not working properly
            debug!(%origin, error = %e, "gRPC connectivity test failed with error");
            return Err(anyhow::anyhow!("gRPC connectivity test failed: {}", e));
        }
        Err(_) => {
            // Timeout during Open call - connection is not working properly
            debug!(%origin, "gRPC connectivity test timed out");
            return Err(anyhow::anyhow!("gRPC connectivity test timed out"));
        }
    }

    // Now create a fresh transport for actual use (don't reuse the test transport)
    let transport = timeout(timeout_duration, GrpcTransport::connect(origin))
        .await
        .context("gRPC connection timed out")?
        .context("gRPC connection failed")?;

    Ok(Box::new(transport))
}

/// Attempt to establish a WebSocket connection.
async fn try_websocket_connection(
    origin: &str,
    session_name: &str,
    config: &ConnectionConfig,
) -> Result<Box<dyn SshxTransport>> {
    let timeout_duration = config.websocket_timeout.unwrap_or(WEBSOCKET_TIMEOUT);
    let ws_url = grpc_to_websocket_url(origin, session_name);
    
    debug!(%ws_url, timeout_ms = timeout_duration.as_millis(), "attempting WebSocket connection");

    // Attempt to connect with timeout
    let transport = timeout(timeout_duration, WebSocketTransport::connect(&ws_url))
        .await
        .context("WebSocket connection timed out")?
        .context("WebSocket connection failed")?;

    Ok(Box::new(transport))
}

/// Test gRPC connectivity to a server without establishing a full connection.
///
/// This is a lightweight function for testing if gRPC is available without
/// the overhead of creating a full transport connection.
///
/// # Arguments
/// * `origin` - The server URL to test
/// * `timeout_duration` - Maximum time to wait for the test
///
/// # Returns
/// `true` if gRPC connectivity is available, `false` otherwise
pub async fn test_grpc_connectivity(origin: &str, timeout_duration: Duration) -> bool {
    debug!(%origin, "testing gRPC connectivity");
    
    let result = timeout(timeout_duration, async {
        // Try to create a basic gRPC client connection
        GrpcTransport::connect(origin).await
    }).await;

    match result {
        Ok(Ok(_)) => {
            debug!(%origin, "gRPC connectivity test passed");
            true
        }
        Ok(Err(e)) => {
            debug!(%origin, error = %e, "gRPC connectivity test failed");
            false
        }
        Err(_) => {
            debug!(%origin, "gRPC connectivity test timed out");
            false
        }
    }
}

/// Create a connection configuration for verbose error reporting.
///
/// This is useful for debugging connection issues or when you want
/// to show detailed error information to users.
pub fn verbose_config() -> ConnectionConfig {
    ConnectionConfig {
        verbose_errors: true,
        ..Default::default()
    }
}

/// Create a connection configuration with custom timeouts.
///
/// # Arguments
/// * `grpc_timeout` - Timeout for gRPC connection attempts
/// * `websocket_timeout` - Timeout for WebSocket connection attempts
pub fn custom_timeout_config(
    grpc_timeout: Duration,
    websocket_timeout: Duration,
) -> ConnectionConfig {
    ConnectionConfig {
        verbose_errors: false,
        grpc_timeout: Some(grpc_timeout),
        websocket_timeout: Some(websocket_timeout),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_connection_config_default() {
        let config = ConnectionConfig::default();
        assert!(!config.verbose_errors);
        assert!(config.grpc_timeout.is_none());
        assert!(config.websocket_timeout.is_none());
    }

    #[test]
    fn test_verbose_config() {
        let config = verbose_config();
        assert!(config.verbose_errors);
    }

    #[test]
    fn test_custom_timeout_config() {
        let grpc_timeout = Duration::from_secs(5);
        let ws_timeout = Duration::from_secs(10);
        let config = custom_timeout_config(grpc_timeout, ws_timeout);
        
        assert_eq!(config.grpc_timeout, Some(grpc_timeout));
        assert_eq!(config.websocket_timeout, Some(ws_timeout));
    }

    #[test]
    fn test_connection_method_equality() {
        assert_eq!(ConnectionMethod::Grpc, ConnectionMethod::Grpc);
        assert_eq!(ConnectionMethod::WebSocketFallback, ConnectionMethod::WebSocketFallback);
        assert_ne!(ConnectionMethod::Grpc, ConnectionMethod::WebSocketFallback);
    }

    // Note: Testing the actual connection logic would require mocking the transport
    // implementations, which is complex with the current design. The actual connection
    // testing would be done through integration tests with real servers.
}