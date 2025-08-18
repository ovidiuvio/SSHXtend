//! Transport abstraction layer for gRPC and WebSocket connections.
//!
//! This module provides a unified interface for connecting to sshx servers
//! via either gRPC or WebSocket protocols, with automatic fallback capability.

use anyhow::{Context, Result};
use async_trait::async_trait;
use sshx_core::proto::{
    sshx_service_client::SshxServiceClient, CloseRequest, OpenRequest, OpenResponse,
};
use tokio_stream::wrappers::ReceiverStream;
use tonic::transport::Channel;
use tonic::Request;
use tracing::debug;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{mpsc, Mutex};
use tokio::time::{timeout, Duration};
use tokio_tungstenite::{connect_async, tungstenite::Message};
use futures_util::{SinkExt, StreamExt, stream::SplitSink, stream::SplitStream};
use tokio_tungstenite::WebSocketStream;
use tokio_tungstenite::MaybeTlsStream;
use serde_json;
use url::Url;

use sshx_core::proto::{ClientUpdate, ServerUpdate, client_update::ClientMessage, server_update::ServerMessage, TerminalInput, NewShell, SequenceNumbers};
use bytes::Bytes;
use pin_project::pin_project;
use std::pin::Pin;
use std::task::{Context as TaskContext, Poll};
use futures_util::Stream;

/// Wrapper for WebSocket streams to match the tonic Streaming interface
#[pin_project]
pub struct WebSocketStreaming<T> {
    #[pin]
    inner: ReceiverStream<Result<T, tonic::Status>>,
}

impl<T> WebSocketStreaming<T> {
    fn new(stream: ReceiverStream<Result<T, tonic::Status>>) -> Self {
        Self { inner: stream }
    }
}

impl<T> Stream for WebSocketStreaming<T> {
    type Item = Result<T, tonic::Status>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut TaskContext<'_>) -> Poll<Option<Self::Item>> {
        let this = self.project();
        this.inner.poll_next(cx)
    }
}


/// CLI WebSocket request message with correlation ID.
#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
struct CliRequest {
    /// Unique request ID for correlation.
    id: String,
    /// The actual request message.
    message: CliMessage,
}

/// CLI WebSocket response message with correlation ID.
#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
struct CliResponse {
    /// Request ID this response corresponds to.
    id: String,
    /// The actual response message.
    message: CliResponseMessage,
}

/// CLI-specific request message types.
#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
enum CliMessage {
    /// Request to open a new session.
    OpenSession {
        /// The origin hostname for the session.
        origin: String,
        /// Encrypted zeros block for authentication.
        encrypted_zeros: Bytes,
        /// Display name for the session.
        name: String,
        /// Optional write password hash for session protection.
        write_password_hash: Option<Bytes>,
    },
    /// Request to close an existing session.
    CloseSession {
        /// The session name to close.
        name: String,
        /// Authentication token for the session.
        token: String,
    },
    /// Start bidirectional streaming for a session.
    StartChannel {
        /// The session name to start streaming for.
        name: String,
        /// Authentication token for the session.
        token: String,
    },
    /// Terminal data from CLI client.
    TerminalData {
        /// Shell ID this data belongs to.
        id: u32,
        /// Raw terminal data bytes.
        data: Bytes,
        /// Sequence number for ordering.
        seq: u64,
    },
    /// Acknowledge new shell creation.
    CreatedShell {
        /// The newly created shell ID.
        id: u32,
        /// Initial x-coordinate of the shell window.
        x: i32,
        /// Initial y-coordinate of the shell window.
        y: i32,
    },
    /// Acknowledge shell closure.
    ClosedShell {
        /// The shell ID that was closed.
        id: u32,
    },
    /// Pong response for latency measurement.
    Pong {
        /// Unix timestamp for latency calculation.
        timestamp: u64,
    },
    /// Error from CLI client.
    Error {
        /// Error message description.
        message: String,
    },
}

/// CLI-specific response message types.
#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
enum CliResponseMessage {
    /// Response to open session request.
    OpenSession {
        /// The session name that was created.
        name: String,
        /// Authentication token for the session.
        token: String,
        /// Public URL to access the session.
        url: String,
    },
    /// Response to close session request.
    CloseSession {},
    /// Response to start channel request.
    StartChannel {},
    /// Terminal input from web clients.
    TerminalInput {
        /// Shell ID this input is for.
        id: u32,
        /// Input data bytes from the user.
        data: Bytes,
        /// Byte offset in the terminal stream.
        offset: u64,
    },
    /// Request to create new shell.
    CreateShell {
        /// The shell ID to create.
        id: u32,
        /// Initial x-coordinate for the shell window.
        x: i32,
        /// Initial y-coordinate for the shell window.
        y: i32,
    },
    /// Request to close shell.
    CloseShell {
        /// The shell ID to close.
        id: u32,
    },
    /// Sequence number synchronization.
    Sync {
        /// Map of shell IDs to their current sequence numbers.
        sequence_numbers: std::collections::HashMap<u32, u64>,
    },
    /// Terminal resize request.
    Resize {
        /// Shell ID to resize.
        id: u32,
        /// New number of rows for the terminal.
        rows: u32,
        /// New number of columns for the terminal.
        cols: u32,
    },
    /// Ping request for latency measurement.
    Ping {
        /// Unix timestamp for latency calculation.
        timestamp: u64,
    },
    /// Error response.
    Error {
        /// Error message description.
        message: String,
    },
}

/// Transport abstraction for sshx server communication.
///
/// This trait provides a unified interface for both gRPC and WebSocket
/// transports, allowing seamless fallback between connection types.
#[async_trait]
pub trait SshxTransport: Send + Sync + std::fmt::Debug {
    /// Open a new session on the server.
    ///
    /// # Arguments
    /// * `request` - The session opening request containing authentication data
    ///
    /// # Returns
    /// The server response with session details on success
    async fn open(&mut self, request: OpenRequest) -> Result<OpenResponse>;

    /// Establish a bidirectional streaming channel for real-time communication.
    ///
    /// # Arguments
    /// * `outbound` - Receiver stream for outbound messages to the server
    ///
    /// # Returns
    /// A stream of inbound messages from the server
    async fn channel(
        &mut self,
        outbound: ReceiverStream<ClientUpdate>,
    ) -> Result<Box<dyn Stream<Item = Result<ServerUpdate, tonic::Status>> + Send + Unpin>>;

    /// Close an existing session on the server.
    ///
    /// # Arguments
    /// * `request` - The session closing request with authentication
    ///
    /// # Returns
    /// Success on proper session closure
    async fn close(&mut self, request: CloseRequest) -> Result<()>;

    /// Get the connection type for logging/debugging purposes.
    fn connection_type(&self) -> &'static str;
}

/// gRPC transport implementation wrapping the existing tonic client.
#[derive(Debug)]
pub struct GrpcTransport {
    client: SshxServiceClient<Channel>,
}

impl GrpcTransport {
    /// Create a new gRPC transport from an existing client.
    ///
    /// # Arguments
    /// * `client` - Pre-connected gRPC client
    pub fn new(client: SshxServiceClient<Channel>) -> Self {
        Self { client }
    }

    /// Create a new gRPC transport by connecting to a server.
    ///
    /// # Arguments
    /// * `origin` - The server URL to connect to
    ///
    /// # Returns
    /// A connected gRPC transport instance
    pub async fn connect(origin: &str) -> Result<Self, tonic::transport::Error> {
        debug!(%origin, "connecting via gRPC");
        let client = SshxServiceClient::connect(String::from(origin)).await?;
        Ok(Self::new(client))
    }
}

#[async_trait]
impl SshxTransport for GrpcTransport {
    async fn open(&mut self, request: OpenRequest) -> Result<OpenResponse> {
        let response = self
            .client
            .open(Request::new(request))
            .await
            .context("gRPC open request failed")?;
        Ok(response.into_inner())
    }

    async fn channel(
        &mut self,
        outbound: ReceiverStream<ClientUpdate>,
    ) -> Result<Box<dyn Stream<Item = Result<ServerUpdate, tonic::Status>> + Send + Unpin>> {
        let response = self
            .client
            .channel(Request::new(outbound))
            .await
            .context("gRPC channel request failed")?;
        Ok(Box::new(response.into_inner()))
    }

    async fn close(&mut self, request: CloseRequest) -> Result<()> {
        self.client
            .close(Request::new(request))
            .await
            .context("gRPC close request failed")?;
        Ok(())
    }

    fn connection_type(&self) -> &'static str {
        "gRPC"
    }
}

/// WebSocket transport implementation for CLI communication.
///
/// This transport provides WebSocket-based communication using JSON
/// messaging compatible with the server's CLI WebSocket endpoint.
pub struct WebSocketTransport {
    /// WebSocket write half for sending messages.
    write: Arc<Mutex<SplitSink<WebSocketStream<MaybeTlsStream<tokio::net::TcpStream>>, Message>>>,
    /// Channel for receiving server messages.
    server_rx: Arc<Mutex<mpsc::Receiver<ServerUpdate>>>,
    /// Request correlation map for matching responses.
    pending_requests: Arc<Mutex<HashMap<String, tokio::sync::oneshot::Sender<CliResponseMessage>>>>,
    /// Background task handle for the WebSocket reader.
    _reader_task: tokio::task::JoinHandle<()>,
    /// Next request ID counter.
    next_request_id: Arc<Mutex<u64>>,
}

impl WebSocketTransport {
    /// Create a new WebSocket transport by connecting to a server.
    ///
    /// # Arguments
    /// * `endpoint` - The WebSocket server URL to connect to
    ///
    /// # Returns
    /// A connected WebSocket transport instance
    pub async fn connect(endpoint: &str) -> Result<Self> {
        debug!(%endpoint, "connecting via WebSocket");
        
        let url = Url::parse(endpoint).context("Failed to parse WebSocket URL")?;
        let (ws_stream, _) = connect_async(url).await
            .context("Failed to connect to WebSocket")?;
        
        let (write, read) = ws_stream.split();
        let write = Arc::new(Mutex::new(write));
        
        let (server_tx, server_rx) = mpsc::channel(256);
        let server_rx = Arc::new(Mutex::new(server_rx));
        
        let pending_requests: Arc<Mutex<HashMap<String, tokio::sync::oneshot::Sender<CliResponseMessage>>>> = 
            Arc::new(Mutex::new(HashMap::new()));
        
        let next_request_id = Arc::new(Mutex::new(0));
        
        // Spawn background task to handle incoming WebSocket messages
        let reader_task = Self::spawn_reader_task(
            read,
            server_tx,
            pending_requests.clone(),
        );
        
        Ok(Self {
            write,
            server_rx,
            pending_requests,
            _reader_task: reader_task,
            next_request_id,
        })
    }
    
    /// Spawn background task to read WebSocket messages and route them appropriately.
    fn spawn_reader_task(
        mut read: SplitStream<WebSocketStream<MaybeTlsStream<tokio::net::TcpStream>>>,
        server_tx: mpsc::Sender<ServerUpdate>,
        pending_requests: Arc<Mutex<HashMap<String, tokio::sync::oneshot::Sender<CliResponseMessage>>>>,
    ) -> tokio::task::JoinHandle<()> {
        tokio::spawn(async move {
            debug!("WebSocket reader task started");
            let mut message_count = 0u64;
            while let Some(msg) = read.next().await {
                message_count += 1;
                match msg {
                    Ok(Message::Text(text)) => {
                        debug!(message_count = %message_count, text_len = text.len(), "Received WebSocket text message");
                        if let Err(e) = Self::handle_text_message(&text, &server_tx, &pending_requests).await {
                            debug!(message_count = %message_count, "Error handling WebSocket message: {}", e);
                        }
                    }
                    Ok(Message::Close(frame)) => {
                        debug!(message_count = %message_count, ?frame, "WebSocket connection closed by server");
                        break;
                    }
                    Err(e) => {
                        debug!(message_count = %message_count, "WebSocket error: {}", e);
                        break;
                    }
                    _ => {} // Ignore other message types
                }
            }
            debug!(message_count = %message_count, "WebSocket reader task exiting");
        })
    }
    
    /// Handle incoming text messages from the WebSocket.
    async fn handle_text_message(
        text: &str,
        server_tx: &mpsc::Sender<ServerUpdate>,
        pending_requests: &Arc<Mutex<HashMap<String, tokio::sync::oneshot::Sender<CliResponseMessage>>>>,
    ) -> Result<()> {
        // Try to parse as CLI response first
        if let Ok(response) = serde_json::from_str::<CliResponse>(text) {
            // Handle streaming messages (sent with "server_update" ID)
            if response.id == "server_update" {
                debug!("Received server update: {:?}", response.message);
                let server_update = Self::cli_response_to_server_update(response.message)?;
                let _ = server_tx.send(server_update).await;
                return Ok(());
            }
            
            // Handle request-response messages
            let mut pending = pending_requests.lock().await;
            if let Some(sender) = pending.remove(&response.id) {
                let _ = sender.send(response.message);
            }
            return Ok(());
        }
        
        // If we get here, the message format was invalid
        debug!("Failed to parse WebSocket message: {}", text);
        
        Ok(())
    }
    
    /// Convert CLI response message to ServerUpdate for streaming.
    fn cli_response_to_server_update(cli_msg: CliResponseMessage) -> Result<ServerUpdate> {
        let server_message = match cli_msg {
            CliResponseMessage::TerminalInput { id, data, offset } => {
                ServerMessage::Input(TerminalInput {
                    id,
                    data: data.into(),
                    offset,
                })
            }
            CliResponseMessage::CreateShell { id, x, y } => {
                ServerMessage::CreateShell(NewShell { id, x, y })
            }
            CliResponseMessage::CloseShell { id } => {
                ServerMessage::CloseShell(id)
            }
            CliResponseMessage::Sync { sequence_numbers } => {
                ServerMessage::Sync(SequenceNumbers {
                    map: sequence_numbers,
                })
            }
            CliResponseMessage::Resize { id, rows, cols } => {
                ServerMessage::Resize(sshx_core::proto::TerminalSize {
                    id,
                    rows,
                    cols,
                })
            }
            CliResponseMessage::Ping { timestamp } => {
                ServerMessage::Ping(timestamp)
            }
            CliResponseMessage::Error { message } => {
                ServerMessage::Error(message)
            }
            _ => return Err(anyhow::anyhow!("Unsupported CLI response message for streaming")),
        };
        
        Ok(ServerUpdate {
            server_message: Some(server_message),
        })
    }
    
    /// Generate next unique request ID.
    async fn next_id(&self) -> String {
        let mut counter = self.next_request_id.lock().await;
        *counter += 1;
        format!("req_{}", *counter)
    }
    
    /// Send a request and wait for response with timeout.
    async fn send_request(&mut self, message: CliMessage) -> Result<CliResponseMessage> {
        let id = self.next_id().await;
        let request = CliRequest {
            id: id.clone(),
            message,
        };
        
        let (tx, rx) = tokio::sync::oneshot::channel();
        {
            let mut pending = self.pending_requests.lock().await;
            pending.insert(id.clone(), tx);
        }
        
        let json = serde_json::to_string(&request)
            .context("Failed to serialize request")?;
        
        {
            let mut write = self.write.lock().await;
            write.send(Message::Text(json)).await
                .context("Failed to send WebSocket message")?;
        }
        
        // Wait for response with timeout
        match timeout(Duration::from_secs(30), rx).await {
            Ok(Ok(response)) => Ok(response),
            Ok(Err(_)) => {
                // Remove from pending if still there
                let mut pending = self.pending_requests.lock().await;
                pending.remove(&id);
                Err(anyhow::anyhow!("Request sender was dropped"))
            }
            Err(_) => {
                // Remove from pending on timeout
                let mut pending = self.pending_requests.lock().await;
                pending.remove(&id);
                Err(anyhow::anyhow!("Request timed out"))
            }
        }
    }
    
}

#[async_trait]
impl SshxTransport for WebSocketTransport {
    async fn open(&mut self, request: OpenRequest) -> Result<OpenResponse> {
        let cli_message = CliMessage::OpenSession {
            origin: request.origin,
            encrypted_zeros: request.encrypted_zeros.into(),
            name: request.name,
            write_password_hash: request.write_password_hash.map(|h| h.into()),
        };
        
        let response = self.send_request(cli_message).await
            .context("WebSocket open request failed")?;
        
        match response {
            CliResponseMessage::OpenSession { name, token, url } => {
                Ok(OpenResponse { name, token, url })
            }
            CliResponseMessage::Error { message } => {
                Err(anyhow::anyhow!("Server error: {}", message))
            }
            _ => Err(anyhow::anyhow!("Unexpected response type for open request")),
        }
    }

    async fn channel(
        &mut self,
        mut outbound: ReceiverStream<ClientUpdate>,
    ) -> Result<Box<dyn Stream<Item = Result<ServerUpdate, tonic::Status>> + Send + Unpin>> {
        // Wait for the first Hello message to extract session info
        let first_update = outbound.next().await
            .ok_or_else(|| anyhow::anyhow!("No initial message in outbound stream"))?;
        
        let (name, token) = if let Some(ClientMessage::Hello(hello)) = first_update.client_message {
            let parts: Vec<&str> = hello.split(',').collect();
            if parts.len() != 2 {
                return Err(anyhow::anyhow!("Invalid hello format"));
            }
            (parts[0].to_string(), parts[1].to_string())
        } else {
            return Err(anyhow::anyhow!("Expected Hello message as first message"));
        };
        
        // Send StartChannel request and wait for response
        let start_channel = CliMessage::StartChannel { name, token };
        let response = self.send_request(start_channel).await
            .context("Failed to start WebSocket channel")?;
        
        // Verify we got the expected response
        match response {
            CliResponseMessage::StartChannel {} => {
                debug!("WebSocket channel started successfully");
            }
            CliResponseMessage::Error { message } => {
                return Err(anyhow::anyhow!("Server error starting channel: {}", message));
            }
            _ => {
                return Err(anyhow::anyhow!("Unexpected response to StartChannel"));
            }
        }
        
        // Create a channel for the streaming interface
        let (stream_tx, stream_rx) = mpsc::channel(256);
        
        // Clone shared state for the outbound message handler
        let write = self.write.clone();
        let server_rx = self.server_rx.clone();
        
        // Spawn task to handle remaining outbound messages from the CLI
        tokio::spawn(async move {
            let mut outbound_count = 0u64;
            debug!("WebSocket outbound message handler started");
            while let Some(client_update) = outbound.next().await {
                if let Some(client_message) = client_update.client_message {
                    outbound_count += 1;
                    let cli_message = match Self::client_message_to_cli_message(client_message) {
                        Ok(msg) => msg,
                        Err(e) => {
                            debug!(outbound_count = %outbound_count, "Failed to convert client message: {}", e);
                            continue;
                        }
                    };
                    
                    // For streaming messages, we need to wrap in CliRequest but don't wait for response
                    let request_id = format!("stream_{}", 
                        std::time::SystemTime::now()
                            .duration_since(std::time::UNIX_EPOCH)
                            .unwrap_or_default()
                            .as_nanos());
                    
                    let request = CliRequest {
                        id: request_id,
                        message: cli_message,
                    };
                    
                    let json = match serde_json::to_string(&request) {
                        Ok(j) => j,
                        Err(e) => {
                            debug!(outbound_count = %outbound_count, "Failed to serialize client message: {}", e);
                            continue;
                        }
                    };
                    
                    let mut write_guard = write.lock().await;
                    if let Err(e) = write_guard.send(Message::Text(json)).await {
                        debug!(outbound_count = %outbound_count, "Failed to send outbound message: {}", e);
                        break;
                    }
                }
            }
            debug!(outbound_count = %outbound_count, "WebSocket outbound message handler exiting");
        });
        
        // Spawn task to forward server messages to the stream
        tokio::spawn(async move {
            let mut server_rx_guard = server_rx.lock().await;
            while let Some(server_update) = server_rx_guard.recv().await {
                if stream_tx.send(Ok(server_update)).await.is_err() {
                    break; // Stream receiver dropped
                }
            }
        });
        
        // Create a streaming interface from the receiver
        let stream = tokio_stream::wrappers::ReceiverStream::new(stream_rx);
        let wrapper = WebSocketStreaming::new(stream);
        Ok(Box::new(wrapper))
    }

    async fn close(&mut self, request: CloseRequest) -> Result<()> {
        let cli_message = CliMessage::CloseSession {
            name: request.name,
            token: request.token,
        };
        
        let response = self.send_request(cli_message).await
            .context("WebSocket close request failed")?;
        
        match response {
            CliResponseMessage::CloseSession {} => Ok(()),
            CliResponseMessage::Error { message } => {
                Err(anyhow::anyhow!("Server error: {}", message))
            }
            _ => Err(anyhow::anyhow!("Unexpected response type for close request")),
        }
    }

    fn connection_type(&self) -> &'static str {
        "WebSocket"
    }
}

impl std::fmt::Debug for WebSocketTransport {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("WebSocketTransport")
            .field("connection_type", &"WebSocket")
            .finish()
    }
}

impl Drop for WebSocketTransport {
    fn drop(&mut self) {
        debug!("WebSocket transport being dropped, will clean up resources");
        // Send a close message to properly terminate the connection
        let write = self.write.clone();
        tokio::spawn(async move {
            let mut write_guard = write.lock().await;
            let _ = write_guard.close().await;
            debug!("WebSocket connection closed on drop");
        });
    }
}

impl WebSocketTransport {
    /// Convert gRPC ClientMessage to CLI message format.
    fn client_message_to_cli_message(client_message: ClientMessage) -> Result<CliMessage> {
        match client_message {
            ClientMessage::Hello(hello) => {
                // Parse "name,token" format
                let parts: Vec<&str> = hello.split(',').collect();
                if parts.len() != 2 {
                    return Err(anyhow::anyhow!("Invalid hello format"));
                }
                Ok(CliMessage::StartChannel {
                    name: parts[0].to_string(),
                    token: parts[1].to_string(),
                })
            }
            ClientMessage::Data(terminal_data) => {
                Ok(CliMessage::TerminalData {
                    id: terminal_data.id,
                    data: terminal_data.data.into(),
                    seq: terminal_data.seq,
                })
            }
            ClientMessage::CreatedShell(new_shell) => {
                Ok(CliMessage::CreatedShell {
                    id: new_shell.id,
                    x: new_shell.x,
                    y: new_shell.y,
                })
            }
            ClientMessage::ClosedShell(shell_id) => {
                Ok(CliMessage::ClosedShell {
                    id: shell_id,
                })
            }
            ClientMessage::Pong(timestamp) => {
                Ok(CliMessage::Pong { timestamp })
            }
            ClientMessage::Error(message) => {
                Ok(CliMessage::Error { message })
            }
        }
    }
}

/// Convert a gRPC server URL to its corresponding WebSocket CLI endpoint.
///
/// # Arguments
/// * `grpc_url` - The original gRPC server URL (e.g., "https://example.com:8051")
/// * `session_name` - The session name for the CLI WebSocket endpoint
///
/// # Returns
/// The WebSocket CLI endpoint URL (e.g., "wss://example.com:8051/cli/session_name")
///
/// # Examples
/// ```
/// # use sshx::transport::grpc_to_websocket_url;
/// let ws_url = grpc_to_websocket_url("https://example.com:8051", "my-session");
/// assert_eq!(ws_url, "wss://example.com:8051/api/cli/my-session");
/// 
/// let ws_url = grpc_to_websocket_url("http://localhost:8051", "test");
/// assert_eq!(ws_url, "ws://localhost:8051/api/cli/test");
/// ```
pub fn grpc_to_websocket_url(grpc_url: &str, session_name: &str) -> String {
    let url = grpc_url
        .replace("https://", "wss://")
        .replace("http://", "ws://");
    
    // Handle the case where the URL might end with a slash
    let base = url.trim_end_matches('/');
    
    format!("{}/api/cli/{}", base, session_name)
}

/// Test helper to create a mock transport for testing.
pub mod test_helpers {
    use super::*;
    use std::sync::Arc;
    use tokio::sync::Mutex;

    /// Mock transport for testing that records method calls.
    #[derive(Debug)]
    pub struct MockTransport {
        /// Record of all method calls made to this transport.
        pub calls: Arc<Mutex<Vec<String>>>,
        /// Optional error to return from methods.
        pub error: Option<String>,
    }

    impl MockTransport {
        /// Create a new mock transport.
        pub fn new() -> Self {
            Self {
                calls: Arc::new(Mutex::new(Vec::new())),
                error: None,
            }
        }

        /// Create a mock transport that will return errors.
        pub fn with_error(error: String) -> Self {
            Self {
                calls: Arc::new(Mutex::new(Vec::new())),
                error: Some(error),
            }
        }
    }

    #[async_trait]
    impl SshxTransport for MockTransport {
        async fn open(&mut self, _request: OpenRequest) -> Result<OpenResponse> {
            self.calls.lock().await.push("open".to_string());
            if let Some(err) = &self.error {
                return Err(anyhow::anyhow!(err.clone()));
            }
            Ok(OpenResponse {
                name: "test-session".to_string(),
                token: "test-token".to_string(),
                url: "https://test.com/s/test-session".to_string(),
            })
        }

        async fn channel(
            &mut self,
            _outbound: ReceiverStream<ClientUpdate>,
        ) -> Result<Box<dyn Stream<Item = Result<ServerUpdate, tonic::Status>> + Send + Unpin>> {
            self.calls.lock().await.push("channel".to_string());
            if let Some(err) = &self.error {
                return Err(anyhow::anyhow!(err.clone()));
            }
            // Create a mock stream for testing
            let (_tx, rx) = mpsc::channel(1);
            Ok(Box::new(tokio_stream::wrappers::ReceiverStream::new(rx)))
        }

        async fn close(&mut self, _request: CloseRequest) -> Result<()> {
            self.calls.lock().await.push("close".to_string());
            if let Some(err) = &self.error {
                return Err(anyhow::anyhow!(err.clone()));
            }
            Ok(())
        }

        fn connection_type(&self) -> &'static str {
            "Mock"
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_grpc_to_websocket_url_conversion() {
        // HTTPS to WSS conversion
        assert_eq!(
            grpc_to_websocket_url("https://example.com:8051", "my-session"),
            "wss://example.com:8051/api/cli/my-session"
        );

        // HTTP to WS conversion
        assert_eq!(
            grpc_to_websocket_url("http://localhost:8051", "test"),
            "ws://localhost:8051/api/cli/test"
        );

        // URL with trailing slash
        assert_eq!(
            grpc_to_websocket_url("https://sshx.io/", "session-123"),
            "wss://sshx.io/api/cli/session-123"
        );

        // Standard sshx.io
        assert_eq!(
            grpc_to_websocket_url("https://sshx.io", "my-terminal"),
            "wss://sshx.io/api/cli/my-terminal"
        );
    }

    #[tokio::test]
    async fn test_mock_transport() {
        let mut transport = test_helpers::MockTransport::new();
        
        // Test open call
        let request = OpenRequest {
            origin: "test".to_string(),
            encrypted_zeros: vec![].into(),
            name: "test".to_string(),
            write_password_hash: None,
        };
        
        let result = transport.open(request).await;
        assert!(result.is_ok());
        
        let calls = transport.calls.lock().await;
        assert_eq!(calls.len(), 1);
        assert_eq!(calls[0], "open");
        assert_eq!(transport.connection_type(), "Mock");
    }

    #[tokio::test]
    async fn test_mock_transport_with_error() {
        let mut transport = test_helpers::MockTransport::with_error("test error".to_string());
        
        let request = OpenRequest {
            origin: "test".to_string(),
            encrypted_zeros: vec![].into(),
            name: "test".to_string(),
            write_password_hash: None,
        };
        
        let result = transport.open(request).await;
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("test error"));
    }
}