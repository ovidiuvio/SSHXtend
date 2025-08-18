/**
 * API client for dashboard session management
 */

export interface SessionMetadata {
  sessionName: string;
  url: string;
  writeUrl?: string;
  displayName: string;
  registeredAt: number;
  dashboardKey: string;
}

export interface SessionInfo {
  name: string;
  shellCount: number;
  userCount: number;
  hasWritePassword: boolean;
  lastAccessed: number;
  users: string[];
  metadata?: SessionMetadata;
}

export interface PaginationInfo {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
  hasPrevious: boolean;
  hasNext: boolean;
}

export interface SessionListResponse {
  sessions: SessionInfo[];
  pagination: PaginationInfo;
}

export interface SessionListParams {
  page?: number;
  pageSize?: number;
  search?: string;
  sort?: string;
  order?: 'asc' | 'desc';
}

/**
 * Dashboard information response
 */
export interface DashboardInfo {
  exists: boolean;
  sessionCount: number;
  createdAt?: number;
}

/**
 * Check if a dashboard exists
 */
export async function checkDashboardStatus(dashboardKey: string): Promise<{ enabled: boolean }> {
  try {
    const response = await fetch(`/api/dashboards/${dashboardKey}/status`);
    return { enabled: response.ok };
  } catch (error) {
    console.error('Dashboard status check failed:', error);
    return { enabled: false };
  }
}

/**
 * Get dashboard information
 */
export async function getDashboardInfo(dashboardKey: string): Promise<DashboardInfo> {
  try {
    const response = await fetch(`/api/dashboards/${dashboardKey}/info`);
    if (!response.ok) {
      return { exists: false, sessionCount: 0 };
    }
    return response.json();
  } catch (error) {
    console.error('Failed to get dashboard info:', error);
    return { exists: false, sessionCount: 0 };
  }
}

/**
 * Fetch sessions for a specific dashboard with pagination support
 */
export async function fetchSessions(dashboardKey: string, params: SessionListParams = {}): Promise<SessionListResponse> {
  const searchParams = new URLSearchParams();
  
  if (params.page) searchParams.set('page', params.page.toString());
  if (params.pageSize) searchParams.set('pageSize', params.pageSize.toString());
  if (params.search) searchParams.set('search', params.search);
  if (params.sort) searchParams.set('sort', params.sort);
  if (params.order) searchParams.set('order', params.order);
  
  const url = `/api/dashboards/${dashboardKey}/sessions${searchParams.toString() ? '?' + searchParams.toString() : ''}`;
  
  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
    },
  });
  
  if (!response.ok) {
    if (response.status === 404) {
      throw new Error('Dashboard not found');
    }
    throw new Error(`Failed to fetch sessions: ${response.statusText}`);
  }
  return response.json();
}

/**
 * Fetch all sessions for a dashboard (backwards compatibility)
 */
export async function fetchAllSessions(dashboardKey: string): Promise<SessionInfo[]> {
  const response = await fetchSessions(dashboardKey, { pageSize: 1000 }); // Large page size to get all
  return response.sessions;
}

/**
 * Format last accessed time as human readable string
 */
export function formatLastAccessed(lastAccessedMs: number): string {
  const now = Date.now();
  const diffMs = now - (now - lastAccessedMs); // lastAccessedMs is elapsed time
  const diffSeconds = Math.floor(diffMs / 1000);
  
  if (diffSeconds < 60) {
    return `${diffSeconds}s ago`;
  }
  
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) {
    return `${diffMinutes}m ago`;
  }
  
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }
  
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
}