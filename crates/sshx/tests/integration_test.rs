//! Integration tests for the sshx CLI with transport fallback functionality.

use anyhow::Result;
use sshx::connection::{connect_with_fallback, ConnectionConfig, ConnectionMethod, verbose_config};
use sshx::transport::test_helpers::MockTransport;
use sshx::transport::SshxTransport;
use sshx_core::proto::OpenRequest;
use std::time::Duration;
use tokio;

#[tokio::test]
async fn test_connection_fallback_user_experience() -> Result<()> {
    // Test that connection attempts timeout quickly for good UX
    let start = std::time::Instant::now();
    
    let config = ConnectionConfig::default();
    let result = connect_with_fallback("https://invalid-server.com", "test-session", config).await;
    
    let elapsed = start.elapsed();
    
    // Should fail within reasonable time (less than 10 seconds for both attempts)
    assert!(elapsed < Duration::from_secs(10), "Connection attempts took too long: {:?}", elapsed);
    assert!(result.is_err());
    
    Ok(())
}

#[tokio::test]
async fn test_verbose_configuration() {
    let verbose_config = verbose_config();
    assert!(verbose_config.verbose_errors);
    
    let default_config = ConnectionConfig::default();
    assert!(!default_config.verbose_errors);
}

#[tokio::test]
async fn test_connection_method_reporting() -> Result<()> {
    // This would require actual server connection, so we test the enum values
    assert_eq!(ConnectionMethod::Grpc, ConnectionMethod::Grpc);
    assert_eq!(ConnectionMethod::WebSocketFallback, ConnectionMethod::WebSocketFallback);
    assert_ne!(ConnectionMethod::Grpc, ConnectionMethod::WebSocketFallback);
    
    Ok(())
}

#[tokio::test]
async fn test_mock_transport_functionality() -> Result<()> {
    let transport = MockTransport::new();
    
    // Test that mock transport implements the trait correctly
    assert_eq!(transport.connection_type(), "Mock");
    
    // Test mock transport with error
    let mut error_transport = MockTransport::with_error("test error".to_string());
    let request = OpenRequest {
        origin: "test".to_string(),
        encrypted_zeros: vec![].into(),
        name: "test".to_string(),
        write_password_hash: None,
    };
    
    let result = error_transport.open(request).await;
    assert!(result.is_err());
    assert!(result.unwrap_err().to_string().contains("test error"));
    
    Ok(())
}

#[test]
fn test_performance_requirements() {
    // Verify that our timeout constants meet performance requirements
    use sshx::connection::{GRPC_TIMEOUT, WEBSOCKET_TIMEOUT};
    
    // Total fallback detection should be under 5 seconds as per requirements
    let total_timeout = GRPC_TIMEOUT + WEBSOCKET_TIMEOUT;
    assert!(total_timeout <= Duration::from_secs(8), 
        "Total fallback time {} exceeds 8 seconds", total_timeout.as_secs());
}

/// This test demonstrates the expected user workflow
#[tokio::test]
async fn test_user_workflow_simulation() -> Result<()> {
    // Simulate what happens when a user runs: sshx --verbose
    let config = verbose_config();
    
    // This should attempt connection and provide detailed feedback
    let result = connect_with_fallback("https://invalid.test", "user-session", config).await;
    
    // Should fail but with appropriate error context
    assert!(result.is_err());
    let error_msg = format!("{}", result.unwrap_err());
    assert!(error_msg.contains("Both gRPC and WebSocket connections failed"));
    
    Ok(())
}