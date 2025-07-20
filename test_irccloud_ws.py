#!/usr/bin/env python3
"""
Simple IRCCloud WebSocket API connection test
Based on: https://github.com/irccloud/irccloud-tools/wiki/API-Overview
"""

import urllib.request
import urllib.parse
import json
import ssl

# Configure these credentials
EMAIL = "riku.lindblad@iki.fi"
PASSWORD = "LJoqV4hkqUZZUt8PLPGY"

# IRCCloud API endpoints
BASE_URL = "https://www.irccloud.com"
WS_URL = "wss://www.irccloud.com/"


def debug_http_request(method, url, data=None, headers=None):
    """Print detailed HTTP request information"""
    print(f"\n--- HTTP {method} Request ---")
    print(f"URL: {url}")
    if headers:
        print("Headers:")
        for key, value in headers.items():
            print(f"  {key}: {value}")
    if data:
        print(f"Data: {data}")
    print("--- End Request ---\n")

def debug_http_response(response, error=None):
    """Print detailed HTTP response information"""
    if error:
        print("\n--- HTTP Error Response ---")
        print(f"Error: {error}")
        if hasattr(error, 'code'):
            print(f"Status Code: {error.code}")
        if hasattr(error, 'headers'):
            print("Headers:")
            for key, value in error.headers.items():
                print(f"  {key}: {value}")
        if hasattr(error, 'read'):
            try:
                body = error.read().decode()
                print(f"Body: {body}")
            except Exception:
                print("Body: (unable to read)")
        print("--- End Error Response ---\n")
    else:
        print("\n--- HTTP Success Response ---")
        print(f"Status Code: {response.status}")
        print("Headers:")
        for key, value in response.headers.items():
            print(f"  {key}: {value}")
        print("--- End Response ---\n")

def test_endpoint_availability():
    """Test basic connectivity to IRCCloud API"""
    endpoints_to_test = [
        f"{BASE_URL}/",
        f"{BASE_URL}/chat/",
        f"{BASE_URL}/chat/auth-formtoken",
        f"{BASE_URL}/formtoken",
        f"{BASE_URL}/api/",
        f"{BASE_URL}/auth/formtoken",
    ]
    
    for endpoint in endpoints_to_test:
        print(f"\nTesting endpoint: {endpoint}")
        try:
            req = urllib.request.Request(endpoint)
            req.add_header("User-Agent", "IRCCloud-Test-Client/1.0")
            req.add_header("Accept", "application/json, text/html, */*")
            
            debug_http_request("GET", endpoint, headers=dict(req.headers))
            
            with urllib.request.urlopen(req) as response:
                debug_http_response(response)
                body = response.read().decode()
                print(f"Response body: {body[:200]}{'...' if len(body) > 200 else ''}")
                
                # Try to parse as JSON
                try:
                    data = json.loads(body)
                    print(f"JSON response: {data}")
                except json.JSONDecodeError:
                    print("Response is not valid JSON")
                    
        except Exception as e:
            debug_http_response(None, e)

def get_auth_token():
    """Get authentication formtoken from IRCCloud using POST with content-length: 0"""
    endpoint = f"{BASE_URL}/chat/auth-formtoken"
    
    print(f"\nTrying auth endpoint: {endpoint}")
    try:
        # Use POST with empty body and content-length: 0 (as per working curl command)
        req = urllib.request.Request(endpoint, data=b"")
        req.add_header("User-Agent", "IRCCloud-Test-Client/1.0")
        req.add_header("Accept", "application/json")
        req.add_header("Content-Length", "0")
        
        debug_http_request("POST", endpoint, data="(empty)", headers=dict(req.headers))
        
        with urllib.request.urlopen(req) as response:
            debug_http_response(response)
            body = response.read().decode()
            print(f"Response body: {body}")
            
            data = json.loads(body)
            if data.get("success"):
                token = data.get("token")
                print(f"✓ Got token from {endpoint}: {token[:10]}...")
                return token
            else:
                print(f"✗ No success in response: {data}")
                
    except Exception as e:
        debug_http_response(None, e)
    
    print("Failed to get auth token")
    return None


def login(email, password, token):
    """Login to IRCCloud and get session cookie with x-auth-formtoken header"""
    endpoint = f"{BASE_URL}/chat/login"
    
    print(f"\nTrying login endpoint: {endpoint}")
    try:
        login_data = {"email": email, "password": password, "token": token}
        data = urllib.parse.urlencode(login_data).encode()
        
        req = urllib.request.Request(endpoint, data=data)
        req.add_header("Content-Type", "application/x-www-form-urlencoded")
        req.add_header("User-Agent", "IRCCloud-Test-Client/1.0")
        req.add_header("Accept", "application/json")
        req.add_header("x-auth-formtoken", token)  # Critical header from curl command
        
        debug_http_request("POST", endpoint, data=login_data, headers=dict(req.headers))

        with urllib.request.urlopen(req) as response:
            debug_http_response(response)
            body = response.read().decode()
            print(f"Response body: {body}")
            
            result = json.loads(body)

            if result.get("success"):
                print("✓ Login successful!")
                
                # Extract session from JSON response (preferred method)
                session = result.get("session")
                if session:
                    print(f"✓ Got session from JSON: {session}")
                
                # Also check for session cookie in headers as backup
                cookies = response.headers.get_all("Set-Cookie")
                session_cookie = None

                if cookies:
                    print("Cookies received:")
                    for cookie in cookies:
                        print(f"  {cookie}")
                        if cookie.startswith("session="):
                            session_cookie = cookie.split(";")[0]
                            print(f"✓ Found session cookie: {session_cookie}")

                # Return the session from JSON response (it's the actual session ID)
                return f"session={session}" if session else session_cookie, result
            else:
                print(f"✗ Login failed: {result}")

    except Exception as e:
        debug_http_response(None, e)
    
    print("Failed to login")
    return None, None


def test_websocket_connection(session_cookie, login_result):
    """Test WebSocket connection to IRCCloud using dynamic host and path"""
    try:
        # Try to import websocket library
        import websocket

        def on_message(ws, message):
            print(f"Received: {message}")

        def on_error(ws, error):
            print(f"WebSocket error: {error}")

        def on_close(ws, close_status_code, close_msg):
            print("WebSocket connection closed")

        def on_open(ws):
            print("WebSocket connection established successfully!")
            print("Waiting for messages... (press Ctrl+C to stop)")

        # Use WebSocket host and path from login response
        if login_result and login_result.get("websocket_host") and login_result.get("websocket_path"):
            ws_host = login_result["websocket_host"]
            ws_path = login_result["websocket_path"]
            ws_url = f"wss://{ws_host}{ws_path}"
            print(f"Using WebSocket URL from login response: {ws_url}")
        else:
            ws_url = WS_URL
            print(f"Using fallback WebSocket URL: {ws_url}")

        # Create WebSocket with required headers
        headers = {
            "Origin": "https://www.irccloud.com",
            "Cookie": session_cookie if session_cookie else "",
        }

        print(f"Connecting to {ws_url}...")
        print(
            f"Using session cookie: {session_cookie[:20]}..."
            if session_cookie
            else "No session cookie"
        )

        ws = websocket.WebSocketApp(
            ws_url,
            header=headers,
            on_open=on_open,
            on_message=on_message,
            on_error=on_error,
            on_close=on_close,
        )

        # Run with SSL context
        ws.run_forever(sslopt={"cert_reqs": ssl.CERT_NONE})

    except ImportError:
        print("websocket-client library not available")
        print("Install with: pip install websocket-client")
        print("Attempting basic connection test instead...")
        test_basic_connection()
    except KeyboardInterrupt:
        print("\nConnection test interrupted by user")
    except Exception as e:
        print(f"WebSocket connection failed: {e}")


def test_basic_connection():
    """Basic connection test without websocket library"""
    import socket
    import ssl

    try:
        # Parse URL
        host = "www.irccloud.com"
        port = 443

        print(f"Testing basic SSL connection to {host}:{port}...")

        # Create SSL socket
        context = ssl.create_default_context()
        sock = socket.create_connection((host, port))
        ssl_sock = context.wrap_socket(sock, server_hostname=host)

        print("SSL connection successful!")
        cert = ssl_sock.getpeercert()
        if cert and 'subject' in cert:
            print(f"Peer certificate: {cert['subject']}")
        else:
            print("Peer certificate: (not available)")

        ssl_sock.close()

    except Exception as e:
        print(f"Basic connection test failed: {e}")


def main():
    """Main test function"""
    print("IRCCloud WebSocket API Connection Test")
    print("=" * 40)

    if EMAIL == "your_email@example.com" or PASSWORD == "your_password":
        print("Please update EMAIL and PASSWORD variables at the top of this script")
        return

    print(f"Testing connection for: {EMAIL}")

    # Step 0: Test endpoint availability
    print("\n0. Testing endpoint availability...")
    test_endpoint_availability()

    # Step 1: Get auth token
    print("\n1. Getting authentication token...")
    token = get_auth_token()
    if not token:
        print("Failed to get authentication token - trying to continue anyway...")
        token = "dummy-token"  # Try with dummy token to see what happens

    print(f"Using token: {token[:10]}...")

    # Step 2: Login
    print("\n2. Logging in...")
    session_cookie, login_result = login(EMAIL, PASSWORD, token)
    if not session_cookie:
        print("Login failed")
        if login_result:
            print(f"Error: {login_result}")
        print("Continuing to test WebSocket without authentication...")

    if session_cookie:
        print(f"Login successful! Session: {session_cookie[:30]}...")

    # Step 3: Test WebSocket connection
    print("\n3. Testing WebSocket connection...")
    test_websocket_connection(session_cookie, login_result)


if __name__ == "__main__":
    main()
