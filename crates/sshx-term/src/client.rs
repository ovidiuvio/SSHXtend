use anyhow::{anyhow, Context, Result};
use bytes::Bytes;
use ciborium;
use futures_util::{SinkExt, StreamExt};
use sshx::encrypt::Encrypt;
use sshx_core::Sid;
use std::collections::HashMap;
use tokio::net::TcpStream;
use tokio_tungstenite::{
    connect_async, tungstenite::Message, MaybeTlsStream, WebSocketStream,
};
use tracing::{debug, error, warn};

// WebSocket protocol types (minimal subset)
#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct WsWinsize {
    pub x: i32,
    pub y: i32,
    pub rows: u16,
    pub cols: u16,
}

#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct WsUser {
    pub name: String,
    pub cursor: Option<(i32, i32)>,
    pub focus: Option<Sid>,
    pub can_write: bool,
}

#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub enum WsServer {
    Hello(u32, String),
    InvalidAuth(),
    Users(Vec<(u32, WsUser)>),
    UserDiff(u32, Option<WsUser>),
    Shells(Vec<(Sid, WsWinsize)>),
    Chunks(Sid, u64, Vec<Bytes>),
    Hear(u32, String, String),
    ShellLatency(u64),
    Pong(u64),
    Error(String),
}

#[derive(serde::Serialize, serde::Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub enum WsClient {
    Authenticate(Bytes, Option<Bytes>),
    SetName(String),
    SetCursor(Option<(i32, i32)>),
    SetFocus(Option<Sid>),
    Create(i32, i32),
    Close(Sid),
    Move(Sid, Option<WsWinsize>),
    Data(Sid, Bytes, u64),
    Subscribe(Sid, u64),
    Chat(String),
    Ping(u64),
}

#[derive(Debug, Clone)]
pub struct ShellInfo {
    pub id: Sid,
    pub winsize: WsWinsize,
    pub title: String,
    pub last_activity: std::time::Instant,
    pub bytes_sent: u64,
    pub bytes_received: u64,
    pub is_focused: bool,
    pub focused_by_users: Vec<String>,
    pub status: TerminalStatus,
}

#[derive(Debug, Clone, PartialEq)]
pub enum TerminalStatus {
    Active,    // Recently active
    Idle,      // No activity for a while  
    Busy,      // High activity
    Focused,   // Currently focused by users
}

pub struct SshxClient {
    ws_stream: WebSocketStream<MaybeTlsStream<TcpStream>>,
    encrypt: Encrypt,
    user_id: u32,
    session_name: String,
    can_write: bool,
    shells: Vec<ShellInfo>,
    users: Vec<(u32, WsUser)>,
    chunk_counter: u64,
    subscription_counters: HashMap<Sid, u64>,
}

impl SshxClient {
    pub async fn connect(
        server: String,
        session_id: String,
        key: String,
        write_password: Option<String>,
    ) -> Result<Self> {
        // Create encryption context
        let encrypt = Encrypt::new(&key);
        let encrypted_zeros = encrypt.zeros();

        let write_password_hash = if let Some(pass) = write_password {
            let write_encrypt = Encrypt::new(&pass);
            Some(write_encrypt.zeros())
        } else {
            None
        };

        // Connect WebSocket
        let ws_url = format!("{}/api/s/{}", server.replace("http", "ws"), session_id);
        debug!("Connecting to WebSocket: {}", ws_url);

        let (ws_stream, _) = connect_async(&ws_url)
            .await
            .context("Failed to connect to WebSocket")?;

        let mut client = Self {
            ws_stream,
            encrypt,
            user_id: 0,
            session_name: String::new(),
            can_write: false,
            shells: Vec::new(),
            users: Vec::new(),
            chunk_counter: 0,
            subscription_counters: HashMap::new(),
        };

        // Authenticate
        client.authenticate(encrypted_zeros, write_password_hash).await?;

        Ok(client)
    }

    async fn authenticate(
        &mut self,
        encrypted_zeros: Vec<u8>,
        write_password_hash: Option<Vec<u8>>,
    ) -> Result<()> {
        // Send authentication
        let auth_msg = WsClient::Authenticate(
            Bytes::from(encrypted_zeros),
            write_password_hash.map(Bytes::from),
        );
        self.send_message(auth_msg).await?;

        // Wait for Hello or InvalidAuth
        match self.receive_message().await? {
            WsServer::Hello(user_id, session_name) => {
                self.user_id = user_id;
                self.session_name = session_name;
                self.can_write = true; // Assume write access unless write password failed
                debug!("Authenticated as user {}", user_id);
            }
            WsServer::InvalidAuth() => {
                return Err(anyhow!("Authentication failed - invalid encryption key or write password"));
            }
            msg => {
                return Err(anyhow!("Unexpected message during auth: {:?}", msg));
            }
        }

        // Set name to identify as terminal client
        self.send_message(WsClient::SetName("sshx-term".to_string()))
            .await?;

        Ok(())
    }

    pub async fn get_shells(&mut self) -> Result<Vec<ShellInfo>> {
        // Wait for initial shells message
        loop {
            match self.receive_message().await? {
                WsServer::Shells(shells) => {
                    self.update_shells(shells);
                    return Ok(self.shells.clone());
                }
                WsServer::Users(users) => {
                    self.users = users;
                    self.update_shell_focus_info();
                }
                msg => {
                    debug!("Received message while waiting for shells: {:?}", msg);
                }
            }
        }
    }

    pub async fn create_shell(&mut self, x: i32, y: i32) -> Result<Sid> {
        if !self.can_write {
            return Err(anyhow!("Cannot create shell in read-only mode"));
        }

        self.send_message(WsClient::Create(x, y)).await?;

        // Wait for updated shells list with the new shell
        loop {
            match self.receive_message().await? {
                WsServer::Shells(shells) => {
                    let new_shells: Vec<ShellInfo> = shells
                        .into_iter()
                        .map(|(id, winsize)| ShellInfo { 
                            id, 
                            winsize,
                            title: format!("Terminal {}", id.0),
                            last_activity: std::time::Instant::now(),
                            bytes_sent: 0,
                            bytes_received: 0,
                            is_focused: false,
                            focused_by_users: Vec::new(),
                            status: TerminalStatus::Active,
                        })
                        .collect();

                    // Find the new shell (one that wasn't in the previous list)
                    for shell in &new_shells {
                        if !self.shells.iter().any(|s| s.id == shell.id) {
                            let new_shell_id = shell.id;
                            self.shells = new_shells;
                            return Ok(new_shell_id);
                        }
                    }
                }
                msg => {
                    debug!("Received message while waiting for new shell: {:?}", msg);
                }
            }
        }
    }

    pub async fn subscribe_to_shell(&mut self, shell_id: Sid) -> Result<()> {
        let start_chunk = self.subscription_counters.get(&shell_id).copied().unwrap_or(0);
        self.send_message(WsClient::Subscribe(shell_id, start_chunk))
            .await?;
        Ok(())
    }

    pub async fn send_input(&mut self, shell_id: Sid, data: &[u8]) -> Result<()> {
        if !self.can_write {
            return Err(anyhow!("Cannot send input in read-only mode"));
        }

        // Encrypt the data using stream number 0x200000000
        let encrypted = self.encrypt.segment(0x200000000, self.chunk_counter, data);
        
        self.send_message(WsClient::Data(
            shell_id,
            Bytes::from(encrypted),
            self.chunk_counter,
        ))
        .await?;

        // Update bytes sent counter
        if let Some(shell) = self.shells.iter_mut().find(|s| s.id == shell_id) {
            shell.bytes_sent += data.len() as u64;
        }

        self.chunk_counter += data.len() as u64;
        Ok(())
    }

    pub async fn resize_shell(&mut self, shell_id: Sid, rows: u16, cols: u16) -> Result<()> {
        if !self.can_write {
            return Ok(()); // Silently ignore resize in read-only mode
        }

        // Find current shell to preserve x,y position
        let shell = self.shells.iter().find(|s| s.id == shell_id);
        if let Some(shell) = shell {
            let new_winsize = WsWinsize {
                x: shell.winsize.x,
                y: shell.winsize.y,
                rows,
                cols,
            };
            self.send_message(WsClient::Move(shell_id, Some(new_winsize)))
                .await?;
        }
        Ok(())
    }

    pub async fn receive_terminal_data(&mut self, monitored_shell_id: Option<Sid>) -> Result<Option<(Sid, Vec<u8>)>> {
        match self.receive_message().await? {
            WsServer::Chunks(shell_id, seqnum, chunks) => {
                let mut output = Vec::new();
                let mut current_seq = seqnum;

                for chunk in chunks {
                    // Decrypt chunk using stream number 0x100000000 | shell_id
                    let stream_num = 0x100000000u64 | (shell_id.0 as u64);
                    let decrypted = self.encrypt.segment(stream_num, current_seq, &chunk);
                    
                    // Check for terminal title in the data
                    if let Some(title) = self.extract_title_from_data(&decrypted) {
                        if let Some(shell) = self.shells.iter_mut().find(|s| s.id == shell_id) {
                            shell.title = title;
                        }
                    }
                    
                    // Update activity and byte count
                    if let Some(shell) = self.shells.iter_mut().find(|s| s.id == shell_id) {
                        shell.last_activity = std::time::Instant::now();
                        shell.bytes_received += decrypted.len() as u64;
                    }
                    
                    output.extend_from_slice(&decrypted);
                    current_seq += chunk.len() as u64;
                }

                // Update subscription counter
                self.subscription_counters.insert(shell_id, current_seq);

                Ok(Some((shell_id, output)))
            }
            WsServer::Shells(shells) => {
                // Check if the monitored shell is still present
                if let Some(monitored_id) = monitored_shell_id {
                    if !shells.iter().any(|(id, _)| *id == monitored_id) {
                        debug!("Shell {} was removed from shells list, exiting session", monitored_id.0);
                        return Err(anyhow!("Remote shell {} has been closed", monitored_id.0));
                    }
                }
                
                // Update shells list
                self.update_shells(shells);
                Ok(None)
            }
            WsServer::Error(msg) => {
                error!("Server error: {}", msg);
                Err(anyhow!("Server error: {}", msg))
            }
            msg => {
                debug!("Ignoring message: {:?}", msg);
                Ok(None)
            }
        }
    }

    async fn send_message(&mut self, message: WsClient) -> Result<()> {
        let mut buf = Vec::new();
        ciborium::ser::into_writer(&message, &mut buf)?;
        self.ws_stream
            .send(Message::Binary(buf))
            .await
            .context("Failed to send WebSocket message")?;
        Ok(())
    }

    async fn receive_message(&mut self) -> Result<WsServer> {
        loop {
            match self.ws_stream.next().await {
                Some(Ok(Message::Binary(data))) => {
                    let message: WsServer = ciborium::de::from_reader(&*data)
                        .context("Failed to deserialize message")?;
                    return Ok(message);
                }
                Some(Ok(Message::Text(_))) => {
                    warn!("Ignoring text message");
                }
                Some(Ok(Message::Close(_))) => {
                    return Err(anyhow!("WebSocket connection closed"));
                }
                Some(Ok(_)) => {
                    // Ignore other message types
                }
                Some(Err(e)) => {
                    return Err(anyhow!("WebSocket error: {}", e));
                }
                None => {
                    return Err(anyhow!("WebSocket stream ended"));
                }
            }
        }
    }


    fn update_shells(&mut self, shells: Vec<(Sid, WsWinsize)>) {
        let now = std::time::Instant::now();
        
        // Create a map of existing shells for quick lookup
        let mut existing_shells: std::collections::HashMap<Sid, ShellInfo> = 
            self.shells.drain(..).map(|shell| (shell.id, shell)).collect();
        
        self.shells = shells
            .into_iter()
            .map(|(id, winsize)| {
                if let Some(mut existing) = existing_shells.remove(&id) {
                    // Update existing shell
                    existing.winsize = winsize;
                    existing.last_activity = now;
                    existing
                } else {
                    // New shell
                    ShellInfo {
                        id,
                        winsize,
                        title: format!("Terminal {}", id.0),
                        last_activity: now,
                        bytes_sent: 0,
                        bytes_received: 0,
                        is_focused: false,
                        focused_by_users: Vec::new(),
                        status: TerminalStatus::Active,
                    }
                }
            })
            .collect();
        
        self.update_shell_focus_info();
        self.update_shell_status();
    }

    fn update_shell_focus_info(&mut self) {
        // Reset focus info
        for shell in &mut self.shells {
            shell.is_focused = false;
            shell.focused_by_users.clear();
        }
        
        // Update focus info from users
        for (_uid, user) in &self.users {
            if let Some(focus_id) = user.focus {
                if let Some(shell) = self.shells.iter_mut().find(|s| s.id == focus_id) {
                    shell.is_focused = true;
                    shell.focused_by_users.push(user.name.clone());
                }
            }
        }
    }

    fn update_shell_status(&mut self) {
        let now = std::time::Instant::now();
        
        for shell in &mut self.shells {
            let idle_time = now.duration_since(shell.last_activity);
            
            shell.status = if shell.is_focused {
                TerminalStatus::Focused
            } else if idle_time > std::time::Duration::from_secs(300) { // 5 minutes
                TerminalStatus::Idle
            } else if idle_time < std::time::Duration::from_secs(10) {
                TerminalStatus::Active
            } else {
                TerminalStatus::Busy
            };
        }
    }

    fn extract_title_from_data(&self, data: &[u8]) -> Option<String> {
        let text = String::from_utf8_lossy(data);
        
        // Look for OSC 0 (set window title) escape sequence: \x1b]0;title\x07 or \x1b]0;title\x1b\\
        if let Some(start) = text.find("\x1b]0;") {
            let title_start = start + 4;
            if let Some(end) = text[title_start..].find(|c| c == '\x07' || c == '\x1b') {
                let title = &text[title_start..title_start + end];
                if !title.is_empty() {
                    return Some(self.clean_terminal_title(title));
                }
            }
        }
        
        // Also look for OSC 2 (set window title) sequence: \x1b]2;title\x07
        if let Some(start) = text.find("\x1b]2;") {
            let title_start = start + 4;
            if let Some(end) = text[title_start..].find(|c| c == '\x07' || c == '\x1b') {
                let title = &text[title_start..title_start + end];
                if !title.is_empty() {
                    return Some(self.clean_terminal_title(title));
                }
            }
        }
        
        None
    }

    fn clean_terminal_title(&self, title: &str) -> String {
        let title = title.trim();
        
        // Extract useful information from common title formats
        if title.contains('@') && title.contains(':') {
            // Format: user@host:path - extract just the command or path
            if let Some(colon_pos) = title.rfind(':') {
                let path_part = &title[colon_pos + 1..];
                if path_part.starts_with('~') || path_part.starts_with('/') {
                    // It's a path, try to get the directory name
                    let clean_path = path_part.trim_start_matches('~').trim_start_matches('/');
                    if clean_path.is_empty() {
                        return "bash".to_string();
                    } else {
                        return format!("bash:{}", clean_path.split('/').last().unwrap_or(clean_path));
                    }
                }
            }
        }
        
        // Look for common process names
        let common_processes = [
            "vim", "nvim", "nano", "emacs", "code", "htop", "top", "less", "more",
            "git", "ssh", "curl", "wget", "docker", "kubectl", "npm", "yarn",
            "python", "node", "cargo", "make", "cmake", "gcc", "rustc"
        ];
        
        let lower_title = title.to_lowercase();
        for process in &common_processes {
            if lower_title.contains(process) {
                // Try to extract filename or argument
                if let Some(space_pos) = title.find(' ') {
                    let args = &title[space_pos + 1..];
                    if !args.is_empty() && !args.starts_with('-') {
                        let filename = args.split_whitespace().next().unwrap_or("");
                        if !filename.starts_with('-') && !filename.is_empty() {
                            let basename = filename.split('/').last().unwrap_or(filename);
                            return format!("{}:{}", process, basename);
                        }
                    }
                }
                return process.to_string();
            }
        }
        
        // Fallback: if it's too long, truncate intelligently
        if title.len() > 25 {
            // Try to find a good truncation point
            if let Some(space_pos) = title[..25].rfind(' ') {
                return format!("{}…", &title[..space_pos]);
            } else {
                return format!("{}…", &title[..22]);
            }
        }
        
        title.to_string()
    }
}