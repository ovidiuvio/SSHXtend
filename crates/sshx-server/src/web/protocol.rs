//! Serializable types sent and received by the web server.

use bytes::Bytes;
use serde::{Deserialize, Serialize};
use sshx_core::{Sid, Uid};

/// Real-time message conveying the position and size of a terminal.
#[derive(Serialize, Deserialize, Debug, Clone, Copy, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct WsWinsize {
    /// The top-left x-coordinate of the window, offset from origin.
    pub x: i32,
    /// The top-left y-coordinate of the window, offset from origin.
    pub y: i32,
    /// The number of rows in the window.
    pub rows: u16,
    /// The number of columns in the terminal.
    pub cols: u16,
}

impl Default for WsWinsize {
    fn default() -> Self {
        WsWinsize {
            x: 0,
            y: 0,
            rows: 24,
            cols: 80,
        }
    }
}

/// Real-time message providing information about a user.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct WsUser {
    /// The user's display name.
    pub name: String,
    /// Live coordinates of the mouse cursor, if available.
    pub cursor: Option<(i32, i32)>,
    /// Currently focused terminal window ID.
    pub focus: Option<Sid>,
    /// Whether the user has write permissions in the session.
    pub can_write: bool,
}

/// A real-time message sent from the server over WebSocket.
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub enum WsServer {
    /// Initial server message, with the user's ID and session metadata.
    Hello(Uid, String),
    /// The user's authentication was invalid.
    InvalidAuth(),
    /// A snapshot of all current users in the session.
    Users(Vec<(Uid, WsUser)>),
    /// Info about a single user in the session: joined, left, or changed.
    UserDiff(Uid, Option<WsUser>),
    /// Notification when the set of open shells has changed.
    Shells(Vec<(Sid, WsWinsize)>),
    /// Subscription results, in the form of terminal data chunks.
    Chunks(Sid, u64, Vec<Bytes>),
    /// Get a chat message tuple `(uid, name, text)` from the room.
    Hear(Uid, String, String),
    /// Forward a latency measurement between the server and backend shell.
    ShellLatency(u64),
    /// Echo back a timestamp, for the the client's own latency measurement.
    Pong(u64),
    /// Alert the client of an application error.
    Error(String),
}

/// A real-time message sent from the client over WebSocket.
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub enum WsClient {
    /// Authenticate the user's encryption key by zeros block and write password
    /// (if provided).
    Authenticate(Bytes, Option<Bytes>),
    /// Set the name of the current user.
    SetName(String),
    /// Send real-time information about the user's cursor.
    SetCursor(Option<(i32, i32)>),
    /// Set the currently focused shell.
    SetFocus(Option<Sid>),
    /// Create a new shell.
    Create(i32, i32),
    /// Close a specific shell.
    Close(Sid),
    /// Move a shell window to a new position and focus it.
    Move(Sid, Option<WsWinsize>),
    /// Add user data to a given shell.
    Data(Sid, Bytes, u64),
    /// Subscribe to a shell, starting at a given chunk index.
    Subscribe(Sid, u64),
    /// Send a a chat message to the room.
    Chat(String),
    /// Send a ping to the server, for latency measurement.
    Ping(u64),
}

/// CLI WebSocket request message with correlation ID.
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct CliRequest {
    /// Unique request ID for correlation.
    pub id: String,
    /// The actual request message.
    pub message: CliMessage,
}

/// CLI WebSocket response message with correlation ID.
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct CliResponse {
    /// Request ID this response corresponds to.
    pub id: String,
    /// The actual response message.
    pub message: CliResponseMessage,
}

/// CLI-specific request message types.
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub enum CliMessage {
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
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub enum CliResponseMessage {
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
