use anyhow::{anyhow, Result};
use url::Url;

/// Parse an sshx URL to extract connection information
/// 
/// Supports formats:
/// - https://sshx.io/s/session123#key
/// - https://sshx.io/s/session123#key,writepass
/// - sshx.io/s/session123#key  
/// - session123#key (assumes sshx.io)
/// - session123#key@custom.server
pub fn parse_sshx_url(input: &str) -> Result<(String, String, String, Option<String>)> {
    let input = input.trim();
    
    // Handle short form with @ for custom server
    if let Some(at_pos) = input.rfind('@') {
        let (session_part, server) = input.split_at(at_pos);
        let server = &server[1..]; // Remove @
        return parse_session_part(session_part, server);
    }
    
    // Try to parse as full URL
    if let Ok(url) = Url::parse(input) {
        let server = format!("{}://{}", url.scheme(), url.host_str().unwrap_or("sshx.io"));
        let path = url.path();
        
        // Extract session from path /s/session_id
        let session_id = if let Some(captures) = path.strip_prefix("/s/") {
            captures.to_string()
        } else {
            return Err(anyhow!("Invalid sshx URL format: missing /s/session_id"));
        };
        
        // Extract key from fragment
        let fragment = url.fragment().ok_or_else(|| anyhow!("Missing encryption key in URL fragment"))?;
        let (key, write_password) = parse_fragment(fragment)?;
        
        return Ok((server, session_id, key, write_password));
    }
    
    // Try to parse with added protocol
    let with_protocol = if input.starts_with("http://") || input.starts_with("https://") {
        input.to_string()
    } else {
        format!("https://{}", input)
    };
    
    if let Ok(url) = Url::parse(&with_protocol) {
        let server = format!("{}://{}", url.scheme(), url.host_str().unwrap_or("sshx.io"));
        let path = url.path();
        
        let session_id = if let Some(captures) = path.strip_prefix("/s/") {
            captures.to_string()
        } else {
            return Err(anyhow!("Invalid sshx URL format: missing /s/session_id"));
        };
        
        let fragment = url.fragment().ok_or_else(|| anyhow!("Missing encryption key in URL fragment"))?;
        let (key, write_password) = parse_fragment(fragment)?;
        
        return Ok((server, session_id, key, write_password));
    }
    
    // Handle short form (session#key)
    parse_session_part(input, "sshx.io")
}

fn parse_session_part(session_part: &str, server: &str) -> Result<(String, String, String, Option<String>)> {
    let hash_pos = session_part.find('#').ok_or_else(|| anyhow!("Missing # in session identifier"))?;
    let session_id = session_part[..hash_pos].to_string();
    let fragment = &session_part[hash_pos + 1..];
    let (key, write_password) = parse_fragment(fragment)?;
    
    let server_url = if server.starts_with("http://") || server.starts_with("https://") {
        server.to_string()
    } else {
        format!("https://{}", server)
    };
    
    Ok((server_url, session_id, key, write_password))
}

fn parse_fragment(fragment: &str) -> Result<(String, Option<String>)> {
    if fragment.is_empty() {
        return Err(anyhow!("Empty encryption key"));
    }
    
    if let Some(comma_pos) = fragment.find(',') {
        let key = fragment[..comma_pos].to_string();
        let write_password = fragment[comma_pos + 1..].to_string();
        Ok((key, Some(write_password)))
    } else {
        Ok((fragment.to_string(), None))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_full_url() {
        let (server, session, key, write_pass) = parse_sshx_url("https://sshx.io/s/abc123#mykey").unwrap();
        assert_eq!(server, "https://sshx.io");
        assert_eq!(session, "abc123");
        assert_eq!(key, "mykey");
        assert_eq!(write_pass, None);
    }

    #[test] 
    fn test_full_url_with_write_password() {
        let (server, session, key, write_pass) = parse_sshx_url("https://sshx.io/s/abc123#mykey,writepass").unwrap();
        assert_eq!(server, "https://sshx.io");
        assert_eq!(session, "abc123");
        assert_eq!(key, "mykey");
        assert_eq!(write_pass, Some("writepass".to_string()));
    }

    #[test]
    fn test_short_form() {
        let (server, session, key, write_pass) = parse_sshx_url("abc123#mykey").unwrap();
        assert_eq!(server, "https://sshx.io");
        assert_eq!(session, "abc123");
        assert_eq!(key, "mykey");
        assert_eq!(write_pass, None);
    }

    #[test]
    fn test_custom_server() {
        let (server, session, key, write_pass) = parse_sshx_url("abc123#mykey@custom.server").unwrap();
        assert_eq!(server, "https://custom.server");
        assert_eq!(session, "abc123");
        assert_eq!(key, "mykey");
        assert_eq!(write_pass, None);
    }

    #[test]
    fn test_domain_without_protocol() {
        let (server, session, key, write_pass) = parse_sshx_url("sshx.io/s/abc123#mykey").unwrap();
        assert_eq!(server, "https://sshx.io");
        assert_eq!(session, "abc123");
        assert_eq!(key, "mykey");
        assert_eq!(write_pass, None);
    }
}