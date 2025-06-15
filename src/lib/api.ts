/**
 * API client for dashboard session management
 */

export interface DashboardMetadata {
  url: string;
  writeUrl?: string;
  displayName: string;
  registeredAt: number;
}

export interface SessionInfo {
  name: string;
  shellCount: number;
  userCount: number;
  hasWritePassword: boolean;
  lastAccessed: number;
  users: string[];
  dashboard?: DashboardMetadata;
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
 * Get dashboard API headers with authentication if needed
 */
function getDashboardHeaders(): HeadersInit {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };
  
  // Check for dashboard key in environment or browser
  const dashboardKey = typeof window !== 'undefined' 
    ? window.localStorage?.getItem('sshx-dashboard-key') 
    : null;
    
  if (dashboardKey) {
    headers['X-Dashboard-Key'] = dashboardKey;
  }
  
  return headers;
}

/**
 * Check if dashboard authentication is required and valid
 */
export async function checkDashboardAuth(): Promise<{required: boolean, valid: boolean}> {
  try {
    const response = await fetch('/api/dashboard/auth', {
      headers: getDashboardHeaders(),
    });
    
    if (response.ok) {
      const result = await response.text();
      return {
        required: result.includes('authenticated'),
        valid: true
      };
    } else if (response.status === 401) {
      return {
        required: true,
        valid: false
      };
    } else {
      throw new Error(`Auth check failed: ${response.statusText}`);
    }
  } catch (error) {
    console.error('Dashboard auth check failed:', error);
    return {
      required: false,
      valid: true
    };
  }
}

/**
 * Fetch sessions with pagination support
 */
export async function fetchSessions(params: SessionListParams = {}): Promise<SessionListResponse> {
  const searchParams = new URLSearchParams();
  
  if (params.page) searchParams.set('page', params.page.toString());
  if (params.pageSize) searchParams.set('pageSize', params.pageSize.toString());
  if (params.search) searchParams.set('search', params.search);
  if (params.sort) searchParams.set('sort', params.sort);
  if (params.order) searchParams.set('order', params.order);
  
  const url = `/api/sessions${searchParams.toString() ? '?' + searchParams.toString() : ''}`;
  
  const response = await fetch(url, {
    headers: getDashboardHeaders(),
  });
  
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('Dashboard authentication required. Please provide the dashboard key.');
    }
    throw new Error(`Failed to fetch sessions: ${response.statusText}`);
  }
  return response.json();
}

/**
 * Fetch all sessions (backwards compatibility)
 */
export async function fetchAllSessions(): Promise<SessionInfo[]> {
  const response = await fetchSessions({ pageSize: 1000 }); // Large page size to get all
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