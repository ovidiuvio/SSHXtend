//! HTTP and WebSocket handlers for the sshx web interface.

use std::sync::Arc;

use axum::routing::{any, get, get_service, post};
use axum::{Json, Router};
use axum::extract::State;
use axum::http::{HeaderMap, StatusCode};
use serde::{Deserialize, Serialize};
use tower_http::services::{ServeDir, ServeFile};
use std::collections::HashMap;
use parking_lot::RwLock;
use std::time::{SystemTime, UNIX_EPOCH};
use once_cell::sync::Lazy;

use crate::ServerState;

pub mod protocol;
mod socket;

/// Global registry for dashboard session metadata
static DASHBOARD_REGISTRY: Lazy<RwLock<HashMap<String, DashboardMetadata>>> = 
    Lazy::new(|| RwLock::new(HashMap::new()));

/// Dashboard registration metadata stored per session
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct DashboardMetadata {
    /// Complete session URL with encryption key
    pub url: String,
    /// Write URL if read-only mode is enabled
    pub write_url: Option<String>,
    /// Display name for the session
    pub display_name: String,
    /// When this registration was created
    pub registered_at: u64,
}

/// Session information for the dashboard API.
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct SessionInfo {
    /// Session name/ID
    pub name: String,
    /// Number of active terminal shells
    pub shell_count: usize,
    /// Number of connected users
    pub user_count: usize,
    /// Whether the session requires a write password
    pub has_write_password: bool,
    /// Unix timestamp of last activity (milliseconds)
    pub last_accessed: u64,
    /// List of connected user names
    pub users: Vec<String>,
    /// Dashboard metadata if session was registered
    pub dashboard: Option<DashboardMetadata>,
}

/// Request payload for dashboard registration
#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct RegisterDashboardRequest {
    /// Session name/ID
    pub session_name: String,
    /// Complete session URL with encryption key
    pub url: String,
    /// Write URL if read-only mode is enabled
    pub write_url: Option<String>,
    /// Display name for the session
    pub display_name: String,
}

/// Handler for registering a session with the dashboard
async fn register_dashboard(
    Json(request): Json<RegisterDashboardRequest>,
) -> Result<Json<&'static str>, axum::http::StatusCode> {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() as u64;

    let metadata = DashboardMetadata {
        url: request.url,
        write_url: request.write_url,
        display_name: request.display_name,
        registered_at: now,
    };

    DASHBOARD_REGISTRY.write().insert(request.session_name, metadata);
    
    Ok(Json("OK"))
}

/// Handler for listing all dashboard-registered sessions
async fn list_sessions(
    State(state): axum::extract::State<Arc<ServerState>>,
    headers: HeaderMap,
) -> Result<Json<Vec<SessionInfo>>, StatusCode> {
    // Check if dashboard key is configured
    if let Some(expected_key) = &state.options().dashboard_key {
        // Check for X-Dashboard-Key header
        if let Some(provided_key) = headers.get("X-Dashboard-Key") {
            if let Ok(key_str) = provided_key.to_str() {
                if key_str == expected_key {
                    // Authentication successful, proceed
                } else {
                    return Err(StatusCode::UNAUTHORIZED);
                }
            } else {
                return Err(StatusCode::UNAUTHORIZED);
            }
        } else {
            return Err(StatusCode::UNAUTHORIZED);
        }
    }
    
    let mut sessions = Vec::new();
    let registry = DASHBOARD_REGISTRY.read();
    
    for (name, session) in state.iter_sessions() {
        // Only include sessions that are registered with the dashboard
        if let Some(dashboard_metadata) = registry.get(&name) {
            let shell_count = session.shell_count();
            
            let user_list = session.list_users();
            let user_count = user_list.len();
            let users: Vec<String> = user_list.into_iter().map(|(_, u)| u.name).collect();
            
            let last_accessed = session.last_accessed().elapsed().as_millis() as u64;
            
            let has_write_password = session.metadata().write_password_hash.is_some();
            
            sessions.push(SessionInfo {
                name,
                shell_count,
                user_count,
                has_write_password,
                last_accessed,
                users,
                dashboard: Some(dashboard_metadata.clone()),
            });
        }
    }
    
    // Sort by most recently accessed
    sessions.sort_by(|a, b| a.last_accessed.cmp(&b.last_accessed));
    
    Ok(Json(sessions))
}

/// Simple authentication handler for dashboard access
async fn check_dashboard_auth(
    State(state): axum::extract::State<Arc<ServerState>>,
    headers: HeaderMap,
) -> Result<Json<&'static str>, StatusCode> {
    // Check if dashboard key is configured
    if let Some(expected_key) = &state.options().dashboard_key {
        // Check for X-Dashboard-Key header
        if let Some(provided_key) = headers.get("X-Dashboard-Key") {
            if let Ok(key_str) = provided_key.to_str() {
                if key_str == expected_key {
                    return Ok(Json("authenticated"));
                }
            }
        }
        // Authentication required but not provided/invalid
        return Err(StatusCode::UNAUTHORIZED);
    }
    
    // No authentication required
    Ok(Json("no-auth-required"))
}

/// Returns the web application server, routed with Axum.
pub fn app() -> Router<Arc<ServerState>> {
    let root_spa = ServeFile::new("build/spa.html")
        .precompressed_gzip()
        .precompressed_br();

    // Serves static SvelteKit build files.
    let static_files = ServeDir::new("build")
        .precompressed_gzip()
        .precompressed_br()
        .fallback(root_spa);

    Router::new()
        .nest("/api", backend())
        .fallback_service(get_service(static_files))
}

/// Routes for the backend web API server.
fn backend() -> Router<Arc<ServerState>> {
    Router::new()
        // Session WebSocket routes (unprotected - clients need direct access)
        .route("/s/{name}", any(socket::get_session_ws))
        // Dashboard API routes (protected)
        .route("/sessions", get(list_sessions))
        .route("/dashboard/register", post(register_dashboard))
        .route("/dashboard/auth", get(check_dashboard_auth))
}
