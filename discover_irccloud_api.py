#!/usr/bin/env python3
"""
IRCCloud API Discovery Script
Analyzes web interface and tests header-based authentication
"""

import urllib.request
import urllib.parse
import json
import ssl
import re

# Configure these credentials
EMAIL = "riku.lindblad@iki.fi"
PASSWORD = "LJoqV4hkqUZZUt8PLPGY"

# IRCCloud endpoints to investigate
BASE_URLS = [
    "https://api.irccloud.com",
    "https://www.irccloud.com",
    "https://irccloud.com"
]

WS_URLS = [
    "wss://api.irccloud.com/",
    "wss://www.irccloud.com/",
    "wss://irccloud.com/"
]

def fetch_web_interface():
    """Fetch and analyze the main web interface for API endpoints"""
    print("\n=== ANALYZING WEB INTERFACE ===")
    
    for base_url in BASE_URLS:
        print(f"\nFetching web interface from: {base_url}")
        try:
            req = urllib.request.Request(base_url)
            req.add_header("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
            req.add_header("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
            
            with urllib.request.urlopen(req) as response:
                html = response.read().decode()
                
                print(f"✓ Got {len(html)} bytes of HTML")
                
                # Look for JavaScript containing API endpoints
                api_patterns = [
                    r'["\']https?://[^"\']*api[^"\']*["\']',
                    r'["\']wss?://[^"\']*["\']',
                    r'["\'][^"\']*(?:auth|login|token|session)[^"\']*["\']',
                    r'["\'][^"\']*formtoken[^"\']*["\']',
                    r'api\s*[:=]\s*["\'][^"\']+["\']',
                    r'websocket\s*[:=]\s*["\'][^"\']+["\']',
                    r'["\'][^"\']*\/chat\/[^"\']*["\']',
                ]
                
                print("\nFound potential API endpoints:")
                all_matches = set()
                for pattern in api_patterns:
                    matches = re.findall(pattern, html, re.IGNORECASE)
                    for match in matches:
                        clean_match = match.strip('"\'')
                        if clean_match and len(clean_match) > 5:
                            all_matches.add(clean_match)
                
                for match in sorted(all_matches):
                    print(f"  {match}")
                
                # Look for authentication-related code
                print("\nLooking for authentication patterns:")
                auth_patterns = [
                    r'x-auth-formtoken[^,;}]+',
                    r'x-irccloud-session[^,;}]+',
                    r'formtoken[^,;}]*=\s*[^,;}&]+',
                    r'session[^,;}]*=\s*[^,;}&]+',
                ]
                
                for pattern in auth_patterns:
                    matches = re.findall(pattern, html, re.IGNORECASE)
                    for match in matches:
                        print(f"  {match}")
                
                return html
                
        except Exception as e:
            print(f"✗ Error fetching {base_url}: {e}")
    
    return None

def test_header_based_auth():
    """Test authentication using headers instead of POST data"""
    print("\n=== TESTING HEADER-BASED AUTHENTICATION ===")
    
    # Try to get a formtoken using headers
    for base_url in BASE_URLS:
        endpoints_to_try = [
            f"{base_url}/chat/auth-formtoken",
            f"{base_url}/formtoken",
            f"{base_url}/auth/formtoken",
            f"{base_url}/api/formtoken",
            f"{base_url}/chat/formtoken",
        ]
        
        for endpoint in endpoints_to_try:
            print(f"\nTrying header-based auth: {endpoint}")
            try:
                req = urllib.request.Request(endpoint)
                req.add_header("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
                req.add_header("Accept", "application/json")
                req.add_header("Origin", "https://www.irccloud.com")
                req.add_header("Referer", "https://www.irccloud.com/")
                
                # Try with common session IDs
                for sid in ["1", "2", "3", "4", "5"]:
                    req.add_header("x-irccloud-sid", sid)
                    
                    with urllib.request.urlopen(req) as response:
                        body = response.read().decode()
                        print(f"✓ Success with sid={sid}: {body}")
                        
                        try:
                            data = json.loads(body)
                            if data.get("success") and data.get("token"):
                                print(f"✓ Got formtoken: {data['token'][:10]}...")
                                return data["token"]
                        except:
                            pass
                        break
                        
            except Exception as e:
                if hasattr(e, 'code') and e.code == 501:
                    print(f"✗ 501 Not Implemented")
                elif hasattr(e, 'code') and e.code == 404:
                    print(f"✗ 404 Not Found")
                else:
                    print(f"✗ Error: {e}")
    
    return None

def test_alternative_endpoints():
    """Test alternative API discovery methods"""
    print("\n=== TESTING ALTERNATIVE ENDPOINTS ===")
    
    # Try common REST API discovery endpoints
    discovery_paths = [
        "/",
        "/api",
        "/api/",
        "/api/v1",
        "/api/v1/",
        "/v1",
        "/v1/",
        "/.well-known/",
        "/health",
        "/status",
        "/version",
        "/openapi.json",
        "/swagger.json",
        "/docs",
    ]
    
    for base_url in BASE_URLS:
        print(f"\nTesting discovery endpoints for: {base_url}")
        
        for path in discovery_paths:
            url = f"{base_url}{path}"
            try:
                req = urllib.request.Request(url)
                req.add_header("User-Agent", "IRCCloud-API-Discovery/1.0")
                req.add_header("Accept", "application/json, text/plain, */*")
                
                with urllib.request.urlopen(req) as response:
                    if response.status == 200:
                        body = response.read().decode()
                        content_type = response.headers.get('content-type', '')
                        
                        if 'json' in content_type:
                            print(f"✓ {path}: JSON response ({len(body)} bytes)")
                            try:
                                data = json.loads(body)
                                print(f"  Data: {data}")
                            except:
                                print(f"  Raw: {body[:100]}...")
                        elif len(body) < 500:
                            print(f"✓ {path}: Short response ({len(body)} bytes): {body}")
                        else:
                            print(f"✓ {path}: {content_type} ({len(body)} bytes)")
                            
            except Exception as e:
                if hasattr(e, 'code') and e.code in [404, 501]:
                    continue  # Skip common errors
                else:
                    print(f"✗ {path}: {e}")

def test_websocket_discovery():
    """Test WebSocket endpoint discovery"""
    print("\n=== TESTING WEBSOCKET ENDPOINTS ===")
    
    # Try to connect to different WebSocket URLs without auth
    for ws_url in WS_URLS:
        print(f"\nTesting WebSocket: {ws_url}")
        
        try:
            # Convert WSS to HTTPS to test basic connectivity
            https_url = ws_url.replace("wss://", "https://").replace("ws://", "http://")
            
            req = urllib.request.Request(https_url)
            req.add_header("User-Agent", "IRCCloud-WS-Discovery/1.0")
            req.add_header("Upgrade", "websocket")
            req.add_header("Connection", "Upgrade")
            req.add_header("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
            req.add_header("Sec-WebSocket-Version", "13")
            req.add_header("Origin", "https://www.irccloud.com")
            
            with urllib.request.urlopen(req) as response:
                print(f"✓ HTTP response: {response.status}")
                for header, value in response.headers.items():
                    if 'websocket' in header.lower() or 'upgrade' in header.lower():
                        print(f"  {header}: {value}")
                        
        except Exception as e:
            if hasattr(e, 'code'):
                if e.code == 426:  # Upgrade Required
                    print(f"✓ WebSocket upgrade required (good sign)")
                elif e.code == 400:  # Bad Request
                    print(f"✓ WebSocket endpoint exists but needs proper handshake")
                elif e.code == 404:
                    print(f"✗ WebSocket endpoint not found")
                else:
                    print(f"? HTTP {e.code}: {e}")
            else:
                print(f"✗ Connection error: {e}")

def test_mobile_api():
    """Test mobile app API endpoints (often different from web)"""
    print("\n=== TESTING MOBILE API PATTERNS ===")
    
    mobile_endpoints = [
        "/mobile/auth",
        "/mobile/login", 
        "/mobile/formtoken",
        "/m/auth",
        "/m/login",
        "/app/auth",
        "/app/login",
        "/client/auth",
        "/client/login",
    ]
    
    for base_url in BASE_URLS:
        for endpoint in mobile_endpoints:
            url = f"{base_url}{endpoint}"
            try:
                req = urllib.request.Request(url)
                req.add_header("User-Agent", "IRCCloud/1.0 (iPhone; iOS 15.0; Scale/3.00)")
                req.add_header("Accept", "application/json")
                
                with urllib.request.urlopen(req) as response:
                    body = response.read().decode()
                    print(f"✓ {endpoint}: {body}")
                    
            except Exception as e:
                if hasattr(e, 'code') and e.code in [404, 501]:
                    continue
                else:
                    print(f"? {endpoint}: {e}")

def main():
    """Main discovery function"""
    print("IRCCloud API Discovery Tool")
    print("=" * 50)
    
    print(f"Target account: {EMAIL}")
    
    # Step 1: Analyze web interface
    html = fetch_web_interface()
    
    # Step 2: Test header-based authentication
    token = test_header_based_auth()
    
    # Step 3: Test alternative endpoints
    test_alternative_endpoints()
    
    # Step 4: Test WebSocket discovery
    test_websocket_discovery()
    
    # Step 5: Test mobile API patterns
    test_mobile_api()
    
    print("\n" + "=" * 50)
    print("DISCOVERY COMPLETE")
    print("=" * 50)
    
    if token:
        print(f"✓ Found working formtoken: {token[:10]}...")
    else:
        print("✗ No working authentication method found")
        print("\nNext steps:")
        print("1. Check browser network tab while logging into IRCCloud")
        print("2. Look for newer API documentation")
        print("3. Try IRCCloud mobile app API endpoints")
        print("4. Consider using unofficial/reverse-engineered clients")

if __name__ == "__main__":
    main()