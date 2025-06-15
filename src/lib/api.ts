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
 * Fetch all active sessions from the server
 */
export async function fetchSessions(): Promise<SessionInfo[]> {
  const response = await fetch('/api/sessions', {
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