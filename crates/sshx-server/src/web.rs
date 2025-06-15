//! HTTP and WebSocket handlers for the sshx web interface.

use std::sync::Arc;

use axum::routing::{any, get, get_service, post};
use axum::{Json, Router};
use axum::extract::{Query, State};
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

/// Query parameters for session listing
#[derive(Deserialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct SessionListQuery {
    /// Page number (1-based)
    #[serde(default = "default_page")]
    pub page: u32,
    /// Number of items per page
    #[serde(default = "default_page_size")]
    pub page_size: u32,
    /// Search query for filtering sessions
    #[serde(default)]
    pub search: Option<String>,
    /// Sort field (name, lastAccessed, userCount, shellCount)
    #[serde(default = "default_sort")]
    pub sort: String,
    /// Sort direction (asc, desc)
    #[serde(default = "default_order")]
    pub order: String,
}

fn default_page() -> u32 { 1 }
fn default_page_size() -> u32 { 20 }
fn default_sort() -> String { "lastAccessed".to_string() }
fn default_order() -> String { "asc".to_string() }

/// Paginated response for session listing
#[derive(Serialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct SessionListResponse {
    /// List of sessions for current page
    pub sessions: Vec<SessionInfo>,
    /// Pagination metadata
    pub pagination: PaginationInfo,
}

/// Pagination metadata
#[derive(Serialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct PaginationInfo {
    /// Current page number (1-based)
    pub page: u32,
    /// Number of items per page
    pub page_size: u32,
    /// Total number of sessions
    pub total: u32,
    /// Total number of pages
    pub total_pages: u32,
    /// Whether there's a previous page
    pub has_previous: bool,
    /// Whether there's a next page
    pub has_next: bool,
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
    Query(query): Query<SessionListQuery>,
    headers: HeaderMap,
) -> Result<Json<SessionListResponse>, StatusCode> {
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
    
    // Apply search filter
    if let Some(search_query) = &query.search {
        if !search_query.trim().is_empty() {
            let search_lower = search_query.to_lowercase();
            sessions.retain(|session| {
                session.name.to_lowercase().contains(&search_lower) ||
                session.dashboard.as_ref()
                    .and_then(|d| Some(d.display_name.to_lowercase().contains(&search_lower)))
                    .unwrap_or(false) ||
                session.users.iter().any(|user| user.to_lowercase().contains(&search_lower))
            });
        }
    }
    
    // Apply sorting
    match query.sort.as_str() {
        "name" => {
            if query.order == "desc" {
                sessions.sort_by(|a, b| b.name.cmp(&a.name));
            } else {
                sessions.sort_by(|a, b| a.name.cmp(&b.name));
            }
        },
        "userCount" => {
            if query.order == "desc" {
                sessions.sort_by(|a, b| b.user_count.cmp(&a.user_count));
            } else {
                sessions.sort_by(|a, b| a.user_count.cmp(&b.user_count));
            }
        },
        "shellCount" => {
            if query.order == "desc" {
                sessions.sort_by(|a, b| b.shell_count.cmp(&a.shell_count));
            } else {
                sessions.sort_by(|a, b| a.shell_count.cmp(&b.shell_count));
            }
        },
        "lastAccessed" | _ => {
            if query.order == "desc" {
                sessions.sort_by(|a, b| b.last_accessed.cmp(&a.last_accessed));
            } else {
                sessions.sort_by(|a, b| a.last_accessed.cmp(&b.last_accessed));
            }
        }
    }
    
    let total = sessions.len() as u32;
    let total_pages = ((total as f32) / (query.page_size as f32)).ceil() as u32;
    let page = query.page.max(1).min(total_pages.max(1));
    
    // Apply pagination
    let start_index = ((page - 1) * query.page_size) as usize;
    
    let paginated_sessions = if start_index < sessions.len() {
        sessions.into_iter().skip(start_index).take(query.page_size as usize).collect()
    } else {
        Vec::new()
    };
    
    let pagination = PaginationInfo {
        page,
        page_size: query.page_size,
        total,
        total_pages,
        has_previous: page > 1,
        has_next: page < total_pages,
    };
    
    Ok(Json(SessionListResponse {
        sessions: paginated_sessions,
        pagination,
    }))
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
