//! HTTP and WebSocket handlers for the sshx web interface.

use std::sync::Arc;

use axum::routing::{any, get, get_service, post};
use axum::{Json, Router};
use axum::extract::{Path, Query, State};
use axum::http::StatusCode;
use serde::{Deserialize, Serialize};
use tower_http::services::{ServeDir, ServeFile};
use std::collections::{HashMap, HashSet};
use parking_lot::RwLock;
use std::time::{SystemTime, UNIX_EPOCH, Duration};
use once_cell::sync::Lazy;
use rand::Rng;
use tokio::time::interval;

use crate::ServerState;

pub mod protocol;
mod socket;

/// A dashboard that contains multiple sessions
#[derive(Debug, Clone)]
pub struct Dashboard {
    /// Unique dashboard key
    pub key: String,
    /// When this dashboard was created
    pub created_at: u64,
    /// When this dashboard was last accessed
    pub last_accessed: u64,
    /// Session names registered to this dashboard
    pub session_names: HashSet<String>,
}

/// Global registry for all dashboards
static DASHBOARDS: Lazy<RwLock<HashMap<String, Dashboard>>> = 
    Lazy::new(|| RwLock::new(HashMap::new()));

/// Global registry for session metadata (session_name -> metadata)
static SESSION_METADATA: Lazy<RwLock<HashMap<String, SessionMetadata>>> = 
    Lazy::new(|| RwLock::new(HashMap::new()));

/// Session metadata for a specific dashboard
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct SessionMetadata {
    /// Session name/ID
    pub session_name: String,
    /// Complete session URL with encryption key
    pub url: String,
    /// Write URL if read-only mode is enabled
    pub write_url: Option<String>,
    /// Display name for the session
    pub display_name: String,
    /// When this registration was created
    pub registered_at: u64,
    /// Dashboard key this session belongs to
    pub dashboard_key: String,
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
    /// Session metadata if registered to a dashboard
    pub metadata: Option<SessionMetadata>,
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
    /// Optional dashboard key to register to (if not provided, generates new)
    pub dashboard_key: Option<String>,
}

/// Response for dashboard registration
#[derive(Serialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct RegisterDashboardResponse {
    /// The dashboard key (generated or provided)
    pub dashboard_key: String,
    /// Full dashboard URL
    pub dashboard_url: String,
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

/// Generate a new dashboard key
fn generate_dashboard_key() -> String {
    const CHARS: &[u8] = b"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    let mut rng = rand::thread_rng();
    (0..16)
        .map(|_| {
            let idx = rng.gen_range(0..CHARS.len());
            CHARS[idx] as char
        })
        .collect()
}

/// Start background task to clean up empty dashboards
pub fn start_dashboard_cleanup() {
    tokio::spawn(async {
        let mut cleanup_interval = interval(Duration::from_secs(3600)); // Check every hour
        loop {
            cleanup_interval.tick().await;
            
            let now = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_millis() as u64;
            
            let mut dashboards = DASHBOARDS.write();
            dashboards.retain(|_, dashboard| {
                // Keep dashboards that have sessions or were accessed in last 24 hours
                !dashboard.session_names.is_empty() || 
                (now - dashboard.last_accessed) < 86_400_000 // 24 hours in ms
            });
        }
    });
}

/// Handler for registering a session with a dashboard
async fn register_dashboard(
    State(state): axum::extract::State<Arc<ServerState>>,
    Json(request): Json<RegisterDashboardRequest>,
) -> Result<Json<RegisterDashboardResponse>, StatusCode> {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() as u64;

    // Get or generate dashboard key
    let dashboard_key = request.dashboard_key.unwrap_or_else(generate_dashboard_key);
    
    // Get or create dashboard
    let mut dashboards = DASHBOARDS.write();
    let dashboard = dashboards.entry(dashboard_key.clone()).or_insert_with(|| {
        Dashboard {
            key: dashboard_key.clone(),
            created_at: now,
            last_accessed: now,
            session_names: HashSet::new(),
        }
    });
    
    // Add session to dashboard
    dashboard.session_names.insert(request.session_name.clone());
    dashboard.last_accessed = now;
    
    // Store session metadata
    let metadata = SessionMetadata {
        session_name: request.session_name.clone(),
        url: request.url,
        write_url: request.write_url,
        display_name: request.display_name,
        registered_at: now,
        dashboard_key: dashboard_key.clone(),
    };
    drop(dashboards);
    
    SESSION_METADATA.write().insert(request.session_name, metadata);
    
    // Build dashboard URL
    let host = state.options().host.as_deref().unwrap_or("localhost");
    let dashboard_url = format!("https://{}/d/{}", host, dashboard_key);
    
    Ok(Json(RegisterDashboardResponse {
        dashboard_key,
        dashboard_url,
    }))
}

/// Handler for listing sessions in a specific dashboard
async fn list_dashboard_sessions(
    State(state): axum::extract::State<Arc<ServerState>>,
    Path(dashboard_key): Path<String>,
    Query(query): Query<SessionListQuery>,
) -> Result<Json<SessionListResponse>, StatusCode> {
    // Update dashboard last accessed time
    {
        let mut dashboards = DASHBOARDS.write();
        if let Some(dashboard) = dashboards.get_mut(&dashboard_key) {
            dashboard.last_accessed = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_millis() as u64;
        } else {
            return Err(StatusCode::NOT_FOUND);
        }
    }
    
    // Get sessions for this dashboard
    let dashboards = DASHBOARDS.read();
    let dashboard = dashboards.get(&dashboard_key).ok_or(StatusCode::NOT_FOUND)?;
    let session_names = dashboard.session_names.clone();
    drop(dashboards);
    
    let mut sessions = Vec::new();
    
    for (name, session) in state.iter_sessions() {
        // Only include sessions registered to this dashboard
        if session_names.contains(&name) {
            let shell_count = session.shell_count();
            
            let user_list = session.list_users();
            let user_count = user_list.len();
            let users: Vec<String> = user_list.into_iter().map(|(_, u)| u.name).collect();
            
            let last_accessed = session.last_accessed().elapsed().as_millis() as u64;
            
            let has_write_password = session.metadata().write_password_hash.is_some();
            
            // Get stored metadata for this session
            let metadata = SESSION_METADATA.read().get(&name).cloned();
            
            sessions.push(SessionInfo {
                name,
                shell_count,
                user_count,
                has_write_password,
                last_accessed,
                users,
                metadata,
            });
        }
    }
    
    // Apply search filter
    if let Some(search_query) = &query.search {
        if !search_query.trim().is_empty() {
            let search_lower = search_query.to_lowercase();
            sessions.retain(|session| {
                session.name.to_lowercase().contains(&search_lower) ||
                session.metadata.as_ref()
                    .map(|m| m.display_name.to_lowercase().contains(&search_lower))
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

/// Check if a dashboard exists
async fn check_dashboard_status(
    Path(dashboard_key): Path<String>,
) -> StatusCode {
    let dashboards = DASHBOARDS.read();
    if dashboards.contains_key(&dashboard_key) {
        StatusCode::OK
    } else {
        StatusCode::NOT_FOUND
    }
}

/// Response for dashboard info
#[derive(Serialize, Debug)]
#[serde(rename_all = "camelCase")]
pub struct DashboardInfoResponse {
    /// Whether the dashboard exists
    pub exists: bool,
    /// Number of sessions in the dashboard
    pub session_count: usize,
    /// When the dashboard was created (Unix timestamp in ms)
    pub created_at: Option<u64>,
}

/// Get dashboard information
async fn get_dashboard_info(
    Path(dashboard_key): Path<String>,
) -> Json<DashboardInfoResponse> {
    let dashboards = DASHBOARDS.read();
    if let Some(dashboard) = dashboards.get(&dashboard_key) {
        Json(DashboardInfoResponse {
            exists: true,
            session_count: dashboard.session_names.len(),
            created_at: Some(dashboard.created_at),
        })
    } else {
        Json(DashboardInfoResponse {
            exists: false,
            session_count: 0,
            created_at: None,
        })
    }
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
        // Dashboard API routes
        .route("/dashboards/{key}/sessions", get(list_dashboard_sessions))
        .route("/dashboards/{key}/status", get(check_dashboard_status))
        .route("/dashboards/{key}/info", get(get_dashboard_info))
        .route("/dashboards/register", post(register_dashboard))
}
