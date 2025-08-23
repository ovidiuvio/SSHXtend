use std::collections::HashSet;
use std::sync::Arc;

use anyhow::{Context, Result};
use axum::extract::{
    ws::{CloseFrame, Message, WebSocket, WebSocketUpgrade},
    Path, State,
};
use axum::response::IntoResponse;
use bytes::Bytes;
use futures_util::SinkExt;
use sshx_core::proto::{
    server_update::ServerMessage, NewShell, ServerUpdate, TerminalInput, TerminalSize,
    SequenceNumbers,
};
use sshx_core::Sid;
use subtle::ConstantTimeEq;
use tokio::sync::mpsc;
use tokio_stream::StreamExt;
use tracing::{debug, error, info_span, warn, Instrument};

use crate::session::Session;
use crate::web::protocol::{WsClient, WsServer};
use sshx_core::proto::{CliRequest, CliResponse, cli_request, cli_response};
use prost::Message as ProstMessage;

type ActiveSession = (
    Arc<Session>,
    mpsc::Receiver<Result<ServerUpdate, tonic::Status>>,
);
use crate::ServerState;

pub async fn get_session_ws(
    Path(name): Path<String>,
    ws: WebSocketUpgrade,
    State(state): State<Arc<ServerState>>,
) -> impl IntoResponse {
    ws.on_upgrade(move |mut socket| {
        let span = info_span!("ws", %name);
        async move {
            match state.frontend_connect(&name).await {
                Ok(Ok(session)) => {
                    if let Err(err) = handle_socket(&mut socket, session).await {
                        // Distinguish between normal connection closures and actual errors
                        let err_msg = err.to_string();
                        if err_msg.contains("Connection reset without closing handshake") 
                            || err_msg.contains("connection was reset") 
                            || err_msg.contains("broken pipe") {
                            debug!(?err, "websocket closed by client");
                        } else {
                            warn!(?err, "websocket exiting early");
                        }
                    } else {
                        socket.close().await.ok();
                    }
                }
                Ok(Err(Some(host))) => {
                    if let Err(err) = proxy_redirect(&mut socket, &host, &name).await {
                        error!(?err, "failed to proxy websocket");
                        let frame = CloseFrame {
                            code: 4500,
                            reason: format!("proxy redirect: {err}").into(),
                        };
                        socket.send(Message::Close(Some(frame))).await.ok();
                    } else {
                        socket.close().await.ok();
                    }
                }
                Ok(Err(None)) => {
                    let frame = CloseFrame {
                        code: 4404,
                        reason: "could not find the requested session".into(),
                    };
                    socket.send(Message::Close(Some(frame))).await.ok();
                }
                Err(err) => {
                    error!(?err, "failed to connect to frontend session");
                    let frame = CloseFrame {
                        code: 4500,
                        reason: format!("session connect: {err}").into(),
                    };
                    socket.send(Message::Close(Some(frame))).await.ok();
                }
            }
        }
        .instrument(span)
    })
}

/// Handle an incoming live WebSocket connection to a given session.
async fn handle_socket(socket: &mut WebSocket, session: Arc<Session>) -> Result<()> {
    /// Send a message to the client over WebSocket.
    async fn send(socket: &mut WebSocket, msg: WsServer) -> Result<()> {
        let mut buf = Vec::new();
        ciborium::ser::into_writer(&msg, &mut buf)?;
        socket.send(Message::Binary(Bytes::from(buf))).await?;
        Ok(())
    }

    /// Receive a message from the client over WebSocket.
    async fn recv(socket: &mut WebSocket) -> Result<Option<WsClient>> {
        Ok(loop {
            match socket.recv().await.transpose()? {
                Some(Message::Text(_)) => warn!("ignoring text message over WebSocket"),
                Some(Message::Binary(msg)) => break Some(ciborium::de::from_reader(&*msg)?),
                Some(_) => (), // ignore other message types, keep looping
                None => break None,
            }
        })
    }

    let metadata = session.metadata();
    let user_id = session.counter().next_uid();
    session.sync_now();
    send(socket, WsServer::Hello(user_id, metadata.name.clone())).await?;

    let can_write = match recv(socket).await? {
        Some(WsClient::Authenticate(bytes, write_password_bytes)) => {
            tracing::debug!(
                browser_bytes_len = bytes.len(),
                stored_bytes_len = metadata.encrypted_zeros.len(),
                bytes_equal = bool::from(bytes.ct_eq(metadata.encrypted_zeros.as_ref())),
                "Browser authentication attempt"
            );
            
            // Constant-time comparison of bytes, converting Choice to bool
            if !bool::from(bytes.ct_eq(metadata.encrypted_zeros.as_ref())) {
                tracing::debug!("Authentication failed: encrypted_zeros mismatch");
                send(socket, WsServer::InvalidAuth()).await?;
                return Ok(());
            }

            match (write_password_bytes, &metadata.write_password_hash) {
                // No password needed, so all users can write (default).
                (_, None) => true,

                // Password stored but not provided, user is read-only.
                (None, Some(_)) => false,

                // Password stored and provided, compare them.
                (Some(provided), Some(stored)) => {
                    if !bool::from(provided.ct_eq(stored)) {
                        send(socket, WsServer::InvalidAuth()).await?;
                        return Ok(());
                    }
                    true
                }
            }
        }
        _ => {
            send(socket, WsServer::InvalidAuth()).await?;
            return Ok(());
        }
    };

    let _user_guard = session.user_scope(user_id, can_write)?;

    let update_tx = session.update_tx(); // start listening for updates before any state reads
    let mut broadcast_stream = session.subscribe_broadcast();
    send(socket, WsServer::Users(session.list_users())).await?;

    let mut subscribed = HashSet::new(); // prevent duplicate subscriptions
    let (chunks_tx, mut chunks_rx) = mpsc::channel::<(Sid, u64, Vec<Bytes>)>(1);

    let mut shells_stream = session.subscribe_shells();
    loop {
        let msg = tokio::select! {
            _ = session.terminated() => break,
            Some(result) = broadcast_stream.next() => {
                let msg = result.context("client fell behind on broadcast stream")?;
                send(socket, msg).await?;
                continue;
            }
            Some(shells) = shells_stream.next() => {
                send(socket, WsServer::Shells(shells)).await?;
                continue;
            }
            Some((id, seqnum, chunks)) = chunks_rx.recv() => {
                send(socket, WsServer::Chunks(id, seqnum, chunks)).await?;
                continue;
            }
            result = recv(socket) => {
                match result? {
                    Some(msg) => msg,
                    None => break,
                }
            }
        };

        match msg {
            WsClient::Authenticate(_, _) => {}
            WsClient::SetName(name) => {
                if !name.is_empty() {
                    session.update_user(user_id, |user| user.name = name)?;
                }
            }
            WsClient::SetCursor(cursor) => {
                session.update_user(user_id, |user| user.cursor = cursor)?;
            }
            WsClient::SetFocus(id) => {
                session.update_user(user_id, |user| user.focus = id)?;
            }
            WsClient::Create(x, y) => {
                if let Err(e) = session.check_write_permission(user_id) {
                    send(socket, WsServer::Error(e.to_string())).await?;
                    continue;
                }
                let id = session.counter().next_sid();
                session.sync_now();
                let new_shell = NewShell { id: id.0, x, y };
                update_tx
                    .send(ServerMessage::CreateShell(new_shell))
                    .await?;
            }
            WsClient::Close(id) => {
                if let Err(e) = session.check_write_permission(user_id) {
                    send(socket, WsServer::Error(e.to_string())).await?;
                    continue;
                }
                update_tx.send(ServerMessage::CloseShell(id.0)).await?;
            }
            WsClient::Move(id, winsize) => {
                if let Err(e) = session.check_write_permission(user_id) {
                    send(socket, WsServer::Error(e.to_string())).await?;
                    continue;
                }
                if let Err(err) = session.move_shell(id, winsize) {
                    send(socket, WsServer::Error(err.to_string())).await?;
                    continue;
                }
                if let Some(winsize) = winsize {
                    let msg = ServerMessage::Resize(TerminalSize {
                        id: id.0,
                        rows: winsize.rows as u32,
                        cols: winsize.cols as u32,
                    });
                    session.update_tx().send(msg).await?;
                }
            }
            WsClient::Data(id, data, offset) => {
                if let Err(e) = session.check_write_permission(user_id) {
                    send(socket, WsServer::Error(e.to_string())).await?;
                    continue;
                }
                let input = TerminalInput {
                    id: id.0,
                    data,
                    offset,
                };
                update_tx.send(ServerMessage::Input(input)).await?;
            }
            WsClient::Subscribe(id, chunknum) => {
                if subscribed.contains(&id) {
                    continue;
                }
                subscribed.insert(id);
                let session = Arc::clone(&session);
                let chunks_tx = chunks_tx.clone();
                tokio::spawn(async move {
                    let stream = session.subscribe_chunks(id, chunknum);
                    tokio::pin!(stream);
                    while let Some((seqnum, chunks)) = stream.next().await {
                        if chunks_tx.send((id, seqnum, chunks)).await.is_err() {
                            break;
                        }
                    }
                });
            }
            WsClient::Chat(msg) => {
                session.send_chat(user_id, &msg)?;
            }
            WsClient::Ping(ts) => {
                send(socket, WsServer::Pong(ts)).await?;
            }
        }
    }
    Ok(())
}

/// Transparently reverse-proxy a WebSocket connection to a different host.
async fn proxy_redirect(socket: &mut WebSocket, host: &str, name: &str) -> Result<()> {
    use tokio_tungstenite::{
        connect_async,
        tungstenite::protocol::{CloseFrame as TCloseFrame, Message as TMessage},
    };

    let (mut upstream, _) = connect_async(format!("ws://{host}/api/s/{name}")).await?;
    loop {
        // Due to axum having its own WebSocket API types, we need to manually translate
        // between it and tungstenite's message type.
        tokio::select! {
            Some(client_msg) = socket.recv() => {
                let msg = match client_msg {
                    Ok(Message::Text(s)) => Some(TMessage::Text(s.as_str().into())),
                    Ok(Message::Binary(b)) => Some(TMessage::Binary(b)),
                    Ok(Message::Close(frame)) => {
                        let frame = frame.map(|frame| TCloseFrame {
                            code: frame.code.into(),
                            reason: frame.reason.as_str().into(),
                        });
                        Some(TMessage::Close(frame))
                    }
                    Ok(_) => None,
                    Err(_) => break,
                };
                if let Some(msg) = msg {
                    if upstream.send(msg).await.is_err() {
                        break;
                    }
                }
            }
            Some(server_msg) = upstream.next() => {
                let msg = match server_msg {
                    Ok(TMessage::Text(s)) => Some(Message::Text(s.as_str().into())),
                    Ok(TMessage::Binary(b)) => Some(Message::Binary(b)),
                    Ok(TMessage::Close(frame)) => {
                        let frame = frame.map(|frame| CloseFrame {
                            code: frame.code.into(),
                            reason: frame.reason.as_str().into(),
                        });
                        Some(Message::Close(frame))
                    }
                    Ok(_) => None,
                    Err(_) => break,
                };
                if let Some(msg) = msg {
                    if socket.send(msg).await.is_err() {
                        break;
                    }
                }
            }
            else => break,
        }
    }

    Ok(())
}

/// Handle CLI WebSocket connection for direct gRPC-like operations.
pub async fn get_cli_ws(
    Path(name): Path<String>,
    ws: WebSocketUpgrade,
    State(state): State<Arc<ServerState>>,
) -> impl IntoResponse {
    ws.on_upgrade(move |socket| {
        let span = info_span!("cli_ws", %name);
        async move {
            if let Err(err) = handle_cli_socket(socket, state, name).await {
                // Distinguish between normal connection closures and actual errors
                let err_msg = err.to_string();
                if err_msg.contains("Connection reset without closing handshake") 
                    || err_msg.contains("connection was reset") 
                    || err_msg.contains("broken pipe") {
                    debug!(?err, "CLI websocket closed by client");
                } else {
                    warn!(?err, "CLI websocket exiting early");
                }
            }
        }
        .instrument(span)
    })
}

/// Handle CLI WebSocket connection with protobuf messaging.
async fn handle_cli_socket(
    mut socket: WebSocket,
    state: Arc<ServerState>,
    name: String,
) -> Result<()> {
    use tracing::debug;
    debug!(session_name = %name, "CLI WebSocket connection established");
    use base64::prelude::{Engine as _, BASE64_STANDARD};
    use hmac::Mac;
    use sshx_core::{rand_alphanumeric, Sid};
    use std::time::SystemTime;
    use tokio::sync::mpsc;

    /// Send a binary protobuf response to the CLI client.
    async fn send_response(socket: &mut WebSocket, response: CliResponse) -> Result<()> {
        let mut buf = Vec::new();
        ProstMessage::encode(&response, &mut buf)?;
        socket.send(Message::Binary(buf.into())).await?;
        Ok(())
    }

    /// Receive a binary protobuf request from the CLI client.
    async fn recv_request(socket: &mut WebSocket) -> Result<Option<CliRequest>> {
        Ok(loop {
            match socket.recv().await.transpose()? {
                Some(Message::Binary(data)) => match ProstMessage::decode(data.as_ref()) {
                    Ok(req) => break Some(req),
                    Err(err) => {
                        warn!(?err, "failed to parse CLI protobuf request");
                        continue;
                    }
                },
                Some(Message::Text(_)) => warn!("ignoring text message from CLI client"),
                Some(_) => (), // ignore other message types, keep looping
                None => break None,
            }
        })
    }

    /// Validate the client token for a session.
    fn validate_token(mac: impl Mac, name: &str, token: &str) -> Result<(), String> {
        if let Ok(token) = BASE64_STANDARD.decode(token) {
            if mac.chain_update(name).verify_slice(&token).is_ok() {
                return Ok(());
            }
        }
        Err("invalid token".to_string())
    }

    /// Get current time in milliseconds.
    fn get_time_ms() -> u64 {
        SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .expect("system time is before the UNIX epoch")
            .as_millis() as u64
    }

    // Main CLI WebSocket message loop
    let mut active_session: Option<ActiveSession> = None;
    let mut streaming_task_handle: Option<tokio::task::JoinHandle<()>> = None;
    let connection_id = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos();
    debug!(session_name = %name, connection_id = %connection_id, "Starting CLI message loop");

    loop {
        tokio::select! {
            // Handle incoming CLI requests
            request = recv_request(&mut socket) => {
                match request? {
                    Some(req) => {
                        let response = match req.cli_message {
                            Some(cli_request::CliMessage::OpenSession(open_req)) => {
                                let origin = open_req.origin;
                                let encrypted_zeros = open_req.encrypted_zeros;
                                let name = open_req.name;
                                let write_password_hash = open_req.write_password_hash;
                                tracing::debug!(
                                    encrypted_zeros_len = encrypted_zeros.len(),
                                    "Received OpenSession request with encrypted_zeros"
                                );
                                let origin = state.override_origin().unwrap_or(origin);
                                if origin.is_empty() {
                                    CliResponse {
                                        id: req.id,
                                        cli_response_message: Some(cli_response::CliResponseMessage::Error("origin is empty".to_string()))
                                    }
                                } else {
                                    let session_name = rand_alphanumeric(10);

                                    match state.lookup(&session_name) {
                                        Some(_) => CliResponse {
                                            id: req.id,
                                            cli_response_message: Some(cli_response::CliResponseMessage::Error("generated duplicate ID".to_string()))
                                        },
                                        None => {
                                            let metadata = crate::session::Metadata {
                                                encrypted_zeros: encrypted_zeros.clone(),
                                                name,
                                                write_password_hash,
                                            };
                                            tracing::debug!(
                                                session_name = %session_name,
                                                encrypted_zeros_len = encrypted_zeros.len(),
                                                "WebSocket CLI session created with encrypted_zeros"
                                            );
                                            state.insert(&session_name, Arc::new(Session::new(metadata)));
                                            let token = state.mac().chain_update(&session_name).finalize();
                                            let url = format!("{origin}/s/{session_name}");

                                            CliResponse {
                                                id: req.id,
                                                cli_response_message: Some(cli_response::CliResponseMessage::OpenSession(
                                                    sshx_core::proto::OpenResponse {
                                                        name: session_name,
                                                        token: BASE64_STANDARD.encode(token.into_bytes()),
                                                        url,
                                                    }
                                                ))
                                            }
                                        }
                                    }
                                }
                            }

                            Some(cli_request::CliMessage::CloseSession(close_req)) => {
                                let name = close_req.name;
                                let token = close_req.token;
                                match validate_token(state.mac(), &name, &token) {
                                    Ok(()) => {
                                        match state.close_session(&name).await {
                                            Ok(()) => CliResponse {
                                                id: req.id,
                                                cli_response_message: Some(cli_response::CliResponseMessage::CloseSession(
                                                    sshx_core::proto::CloseResponse {}
                                                ))
                                            },
                                            Err(err) => CliResponse {
                                                id: req.id,
                                                cli_response_message: Some(cli_response::CliResponseMessage::Error(err.to_string()))
                                            }
                                        }
                                    }
                                    Err(err) => CliResponse {
                                        id: req.id,
                                        cli_response_message: Some(cli_response::CliResponseMessage::Error(err))
                                    }
                                }
                            }

                            Some(cli_request::CliMessage::StartChannel(channel_req)) => {
                                let session_name = channel_req.name;
                                let token = channel_req.token;
                                match validate_token(state.mac(), &session_name, &token) {
                                    Ok(()) => {
                                        match state.backend_connect(&session_name).await {
                                            Ok(Some(session)) => {
                                                // Set up streaming channel similar to gRPC
                                                let (tx, rx) = mpsc::channel::<Result<ServerUpdate, tonic::Status>>(16);
                                                let session_clone = Arc::clone(&session);
                                                let conn_id = connection_id;

                                                // Cancel any existing streaming task
                                                if let Some(handle) = streaming_task_handle.take() {
                                                    debug!(session_name = %session_name, connection_id = %conn_id, "Cancelling previous streaming task");
                                                    handle.abort();
                                                }

                                                debug!(session_name = %session_name, connection_id = %conn_id, "Starting CLI streaming task");
                                                streaming_task_handle = Some(tokio::spawn(async move {
                                                    if let Err(err) = handle_cli_streaming(&tx, &session_clone, conn_id).await {
                                                        // Connection failures during ping/sync are expected when clients disconnect
                                                        if err.contains("client disconnected") {
                                                            debug!(session_name = %session_name, connection_id = %conn_id, "CLI streaming ended: {}", err);
                                                        } else {
                                                            warn!(session_name = %session_name, connection_id = %conn_id, ?err, "CLI streaming exiting early due to error");
                                                        }
                                                    } else {
                                                        debug!(session_name = %session_name, connection_id = %conn_id, "CLI streaming task completed normally");
                                                    }
                                                }));

                                                active_session = Some((session, rx));

                                                CliResponse {
                                                    id: req.id,
                                                    cli_response_message: Some(cli_response::CliResponseMessage::StartChannel(
                                                        sshx_core::proto::ChannelStartResponse {}
                                                    ))
                                                }
                                            }
                                            Ok(None) => CliResponse {
                                                id: req.id,
                                                cli_response_message: Some(cli_response::CliResponseMessage::Error("session not found".to_string()))
                                            },
                                            Err(err) => CliResponse {
                                                id: req.id,
                                                cli_response_message: Some(cli_response::CliResponseMessage::Error(err.to_string()))
                                            }
                                        }
                                    }
                                    Err(err) => CliResponse {
                                        id: req.id,
                                        cli_response_message: Some(cli_response::CliResponseMessage::Error(err))
                                    }
                                }
                            }

                            Some(cli_request::CliMessage::TerminalData(data)) => {
                                if let Some((session, _)) = &active_session {
                                    session.access();
                                    if let Err(err) = session.add_data(Sid(data.id), data.data, data.seq) {
                                        CliResponse {
                                            id: req.id.clone(),
                                            cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                                format!("add data: {:?}", err)
                                            ))
                                        }
                                    } else {
                                        continue; // No response needed for data
                                    }
                                } else {
                                    CliResponse {
                                        id: req.id.clone(),
                                        cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                            "no active session".to_string()
                                        ))
                                    }
                                }
                            }

                            Some(cli_request::CliMessage::CreatedShell(new_shell)) => {
                                if let Some((session, _)) = &active_session {
                                    session.access();
                                    if let Err(err) = session.add_shell(Sid(new_shell.id), (new_shell.x, new_shell.y)) {
                                        CliResponse {
                                            id: req.id.clone(),
                                            cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                                format!("add shell: {:?}", err)
                                            ))
                                        }
                                    } else {
                                        continue; // No response needed
                                    }
                                } else {
                                    CliResponse {
                                        id: req.id.clone(),
                                        cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                            "no active session".to_string()
                                        ))
                                    }
                                }
                            }

                            Some(cli_request::CliMessage::ClosedShell(shell_id)) => {
                                if let Some((session, _)) = &active_session {
                                    session.access();
                                    if let Err(err) = session.close_shell(Sid(shell_id)) {
                                        CliResponse {
                                            id: req.id.clone(),
                                            cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                                format!("close shell: {:?}", err)
                                            ))
                                        }
                                    } else {
                                        continue; // No response needed
                                    }
                                } else {
                                    CliResponse {
                                        id: req.id.clone(),
                                        cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                            "no active session".to_string()
                                        ))
                                    }
                                }
                            }

                            Some(cli_request::CliMessage::Pong(timestamp)) => {
                                if let Some((session, _)) = &active_session {
                                    session.access();
                                    let latency = get_time_ms().saturating_sub(timestamp);
                                    session.send_latency_measurement(latency);
                                }
                                continue; // No response needed
                            }

                            Some(cli_request::CliMessage::Error(message)) => {
                                error!(?message, "error received from CLI client");
                                continue; // No response needed
                            }

                            None => {
                                CliResponse {
                                    id: req.id.clone(),
                                    cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                        "empty message received".to_string()
                                    ))
                                }
                            }
                        };

                        send_response(&mut socket, response).await?;
                    }
                    None => {
                        debug!(session_name = %name, connection_id = %connection_id, "CLI WebSocket connection closed by client");
                        break;
                    }
                }
            }

            // Handle outgoing server messages if we have an active session
            server_msg = async {
                if let Some((_, rx)) = &mut active_session {
                    rx.recv().await
                } else {
                    std::future::pending().await
                }
            } => {
                if let Some(result) = server_msg {
                    match result {
                        Ok(server_update) => {
                            if let Some(server_message) = server_update.server_message {
                                let response = convert_server_message_to_cli(server_message);
                                send_response(&mut socket, response).await?;
                            }
                        }
                        Err(err) => {
                            let response = CliResponse {
                                id: "server_error".to_string(),
                                cli_response_message: Some(cli_response::CliResponseMessage::Error(
                                    err.to_string()
                                ))
                            };
                            send_response(&mut socket, response).await?;
                        }
                    }
                }
            }
        }
    }

    // Clean up any remaining streaming task
    if let Some(handle) = streaming_task_handle.take() {
        debug!(session_name = %name, connection_id = %connection_id, "Cleaning up streaming task on connection close");
        handle.abort();
    }

    debug!(session_name = %name, connection_id = %connection_id, "CLI WebSocket handler exiting");
    Ok(())
}

/// Convert gRPC ServerMessage to CLI response format.
fn convert_server_message_to_cli(message: ServerMessage) -> CliResponse {
    let response_message = match message {
        ServerMessage::Input(input) => {
            cli_response::CliResponseMessage::TerminalInput(TerminalInput {
                id: input.id,
                data: input.data,
                offset: input.offset,
            })
        },
        ServerMessage::CreateShell(new_shell) => {
            cli_response::CliResponseMessage::CreateShell(NewShell {
                id: new_shell.id,
                x: new_shell.x,
                y: new_shell.y,
            })
        },
        ServerMessage::CloseShell(id) => {
            cli_response::CliResponseMessage::CloseShell(id)
        },
        ServerMessage::Sync(seq_nums) => {
            cli_response::CliResponseMessage::Sync(SequenceNumbers {
                map: seq_nums.map,
            })
        },
        ServerMessage::Resize(resize) => {
            cli_response::CliResponseMessage::Resize(TerminalSize {
                id: resize.id,
                rows: resize.rows,
                cols: resize.cols,
            })
        },
        ServerMessage::Ping(timestamp) => {
            cli_response::CliResponseMessage::Ping(timestamp)
        },
        ServerMessage::Error(err) => {
            cli_response::CliResponseMessage::Error(err)
        },
    };

    CliResponse {
        id: "server_update".to_string(),
        cli_response_message: Some(response_message),
    }
}

/// Handle CLI streaming similar to gRPC handle_streaming function.
async fn handle_cli_streaming(
    tx: &mpsc::Sender<Result<ServerUpdate, tonic::Status>>,
    session: &Session,
    connection_id: u128,
) -> Result<(), &'static str> {
    debug!(connection_id = %connection_id, "CLI streaming task started");
    use std::time::{Duration, SystemTime};
    use tokio::time::{self, MissedTickBehavior};

    const SYNC_INTERVAL: Duration = Duration::from_secs(5);
    const PING_INTERVAL: Duration = Duration::from_secs(2);

    /// Send a server message to the client.
    async fn send_msg(
        tx: &mpsc::Sender<Result<ServerUpdate, tonic::Status>>,
        message: ServerMessage,
    ) -> bool {
        let update = Ok(ServerUpdate {
            server_message: Some(message),
        });
        tx.send(update).await.is_ok()
    }

    /// Get current time in milliseconds.
    fn get_time_ms() -> u64 {
        SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .expect("system time is before the UNIX epoch")
            .as_millis() as u64
    }

    let mut sync_interval = time::interval(SYNC_INTERVAL);
    sync_interval.set_missed_tick_behavior(MissedTickBehavior::Delay);

    let mut ping_interval = time::interval(PING_INTERVAL);
    ping_interval.set_missed_tick_behavior(MissedTickBehavior::Delay);

    loop {
        tokio::select! {
            // Send periodic sync messages to the client.
            _ = sync_interval.tick() => {
                let msg = ServerMessage::Sync(session.sequence_numbers());
                if !send_msg(tx, msg).await {
                    debug!(connection_id = %connection_id, "Client disconnected during sync message send");
                    return Err("client disconnected during sync");
                }
            }
            // Send periodic pings to the client.
            _ = ping_interval.tick() => {
                if !send_msg(tx, ServerMessage::Ping(get_time_ms())).await {
                    debug!(connection_id = %connection_id, "Client disconnected during ping message send");
                    return Err("client disconnected during ping");
                }
            }
            // Send buffered server updates to the client.
            Ok(msg) = session.update_rx().recv() => {
                if !send_msg(tx, msg).await {
                    debug!(connection_id = %connection_id, "Client disconnected during update message send");
                    return Err("client disconnected during update");
                }
            }
            // Exit on a session shutdown signal.
            _ = session.terminated() => {
                let msg = String::from("disconnecting because session is closed");
                send_msg(tx, ServerMessage::Error(msg)).await;
                debug!(connection_id = %connection_id, "Session terminated, closing streaming");
                return Ok(());
            }
        };
    }
}
