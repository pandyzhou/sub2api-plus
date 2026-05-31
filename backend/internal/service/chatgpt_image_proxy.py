#!/usr/bin/env python3
"""
Internal image generation proxy using curl_cffi for proper browser TLS fingerprinting.
Called by Go via subprocess - not a standalone service.
Matches chatgpt2api's exact request format for ChatGPT web API.

Includes automatic token refresh when access_token is expired/invalidated.
"""
import sys
import json
import base64
import uuid
import struct
import hashlib
import random
import time
import re
from curl_cffi.requests import Session as CurlSession

# Import PoW utilities from chat2api
try:
    from pow import build_legacy_requirements_token, build_proof_token, parse_pow_resources
except ImportError:
    # Fallback if pow.py is not in the same directory
    import os
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    from pow import build_legacy_requirements_token, build_proof_token, parse_pow_resources


def new_uuid():
    return str(uuid.uuid4())


def log(msg):
    """Log to stderr for Go to capture."""
    print(msg, file=sys.stderr, flush=True)


# ── OpenAI OAuth Token Refresh ──────────────────────────────────────────────
OPENAI_CLIENT_ID = "app_EMoamEEZ73f0CkXaXp7hrann"
OPENAI_TOKEN_URL = "https://auth.openai.com/oauth/token"
OPENAI_REFRESH_SCOPES = "openid profile email offline_access"


def refresh_access_token(refresh_token, proxy=""):
    """Refresh an expired access_token using the refresh_token.
    
    Returns (new_access_token, new_refresh_token) or raises on failure.
    """
    log(f"[refresh] refreshing token...")
    kwargs = {"impersonate": "edge101", "verify": True}
    if proxy:
        kwargs["proxy"] = proxy
    session = CurlSession(**kwargs)

    form_data = {
        "grant_type": "refresh_token",
        "refresh_token": refresh_token,
        "client_id": OPENAI_CLIENT_ID,
        "scope": OPENAI_REFRESH_SCOPES,
    }

    resp = session.post(
        OPENAI_TOKEN_URL,
        data=form_data,
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "Accept": "application/json",
        },
        timeout=30,
    )
    log(f"[refresh] response: {resp.status_code}")
    resp.raise_for_status()

    data = resp.json()
    new_access = data.get("access_token", "")
    new_refresh = data.get("refresh_token", refresh_token)
    expires_in = data.get("expires_in", 0)
    log(f"[refresh] success, expires_in={expires_in}")
    session.close()
    return new_access, new_refresh


# ── Session Builder ─────────────────────────────────────────────────────────
def build_session(access_token, proxy=""):
    """Build curl_cffi session matching chatgpt2api's exact initialization.
    
    CRITICAL: Authorization is set on session globally.
    Sentinel endpoint ignores it (confirmed via testing).
    """
    kwargs = {"impersonate": "edge101", "verify": True}
    if proxy:
        kwargs["proxy"] = proxy
    session = CurlSession(**kwargs)
    device_id = new_uuid()
    session_id = new_uuid()
    session.headers.update({
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                      "(KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0",
        "Origin": "https://chatgpt.com",
        "Referer": "https://chatgpt.com/",
        "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7",
        "Cache-Control": "no-cache",
        "Pragma": "no-cache",
        "Priority": "u=1, i",
        "Sec-Ch-Ua": '"Microsoft Edge";v="143", "Chromium";v="143", "Not A(Brand";v="24"',
        "Sec-Ch-Ua-Arch": '"x86"',
        "Sec-Ch-Ua-Bitness": '"64"',
        "Sec-Ch-Ua-Full-Version": '"143.0.3650.96"',
        "Sec-Ch-Ua-Full-Version-List": '"Microsoft Edge";v="143.0.3650.96", "Chromium";v="143.0.7499.147", "Not A(Brand";v="24.0.0.0"',
        "Sec-Ch-Ua-Mobile": "?0",
        "Sec-Ch-Ua-Model": '""',
        "Sec-Ch-Ua-Platform": '"Windows"',
        "Sec-Ch-Ua-Platform-Version": '"19.0.0"',
        "Sec-Fetch-Dest": "empty",
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Site": "same-origin",
        "OAI-Device-Id": device_id,
        "OAI-Session-Id": session_id,
        "OAI-Language": "zh-CN",
        "OAI-Client-Version": "prod-be885abbfcfe7b1f511e88b3003d9ee44757fbad",
        "OAI-Client-Build-Number": "5955942",
    })
    if access_token:
        session.headers["Authorization"] = f"Bearer {access_token}"
    return session


def image_headers(session, path, sentinel_token, proof_token="", conduit_token="", accept="*/*", turnstile_token="", so_token=""):
    """Build headers for image API requests (matching basketikun/chatgpt2api _image_headers + _headers)."""
    headers = dict(session.headers)
    headers["X-OpenAI-Target-Path"] = path
    headers["X-OpenAI-Target-Route"] = path
    headers["Content-Type"] = "application/json"
    headers["Accept"] = accept
    headers["OpenAI-Sentinel-Chat-Requirements-Token"] = sentinel_token
    if proof_token:
        headers["OpenAI-Sentinel-Proof-Token"] = proof_token
    if turnstile_token:
        headers["OpenAI-Sentinel-Turnstile-Token"] = turnstile_token
    if so_token:
        headers["OpenAI-Sentinel-SO-Token"] = so_token
    if conduit_token:
        headers["X-Conduit-Token"] = conduit_token
    if accept == "text/event-stream":
        headers["X-Oai-Turn-Trace-Id"] = new_uuid()
    return headers


def image_model_slug(model):
    """Map public image model names to ChatGPT Web internal model slugs."""
    base = str(model or "").strip()
    if base == "gpt-image-2":
        return "gpt-5-3"
    return base or "auto"


def bootstrap(session, base_url):
    """Warm up chatgpt.com and parse PoW resources, matching basketikun/chatgpt2api."""
    headers = {
        "User-Agent": session.headers.get("User-Agent", ""),
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
        "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
        "Sec-Ch-Ua": session.headers.get("Sec-Ch-Ua", ""),
        "Sec-Ch-Ua-Mobile": session.headers.get("Sec-Ch-Ua-Mobile", "?0"),
        "Sec-Ch-Ua-Platform": session.headers.get("Sec-Ch-Ua-Platform", '"Windows"'),
        "Sec-Fetch-Dest": "document",
        "Sec-Fetch-Mode": "navigate",
        "Sec-Fetch-Site": "none",
        "Sec-Fetch-User": "?1",
        "Upgrade-Insecure-Requests": "1",
    }
    log("[bootstrap] warming chatgpt.com...")
    resp = session.get(base_url + "/", headers=headers, timeout=30)
    log(f"[bootstrap] status={resp.status_code}")
    resp.raise_for_status()
    script_sources, data_build = parse_pow_resources(resp.text)
    log(f"[bootstrap] pow scripts={len(script_sources)} data_build={'yes' if data_build else 'no'}")
    return script_sources, data_build


def get_sentinel_token(session, device_id):
    """Get sentinel token from sentinel.openai.com.
    
    Sentinel does NOT require auth. We override Authorization to empty
    to prevent the session-level Authorization from being sent.
    """
    user_agent = session.headers.get("User-Agent", "")
    token = build_legacy_requirements_token(user_agent)
    payload = {"p": token, "id": device_id, "flow": "chat"}
    headers = {
        "Content-Type": "text/plain;charset=UTF-8",
        "Referer": "https://sentinel.openai.com/backend-api/sentinel/frame.html",
        "Origin": "https://sentinel.openai.com",
        "User-Agent": user_agent,
    }
    log(f"[sentinel] requesting token...")
    resp = session.post(
        "https://sentinel.openai.com/backend-api/sentinel/req",
        json=payload,
        headers=headers,
        timeout=30
    )
    log(f"[sentinel] status={resp.status_code}")
    if resp.status_code != 200:
        log(f"[sentinel] response: {resp.text[:200]}")
    resp.raise_for_status()
    data = resp.json()
    return (
        data.get("token", ""),
        data.get("proof_token", ""),
        data.get("turnstile_token", ""),
        data.get("so_token", ""),
    )


def get_chat_requirements(session, base_url, initial_req_token, script_sources=None, data_build=""):
    """Get chat requirements exactly like basketikun/chatgpt2api.

    Important: the correct project does not use sentinel.openai.com for this flow;
    it bootstraps chatgpt.com, builds the legacy `p` token with parsed PoW
    resources, then solves any returned proof-of-work challenge.
    """
    path = "/backend-api/sentinel/chat-requirements"
    payload = {"p": initial_req_token}
    headers = dict(session.headers)
    headers["X-OpenAI-Target-Path"] = path
    headers["X-OpenAI-Target-Route"] = path
    headers["Content-Type"] = "application/json"
    headers["Accept"] = "application/json"

    log(f"[chat-requirements] requesting token...")
    resp = session.post(
        base_url + path,
        json=payload,
        headers=headers,
        timeout=30
    )
    log(f"[chat-requirements] status={resp.status_code}")
    if resp.status_code != 200:
        log(f"[chat-requirements] response: {resp.text[:300]}")
    resp.raise_for_status()

    data = resp.json()
    final_token = data.get("token", "")
    proof_token = ""
    proof_info = data.get("proofofwork") or {}
    if proof_info.get("required"):
        log("[chat-requirements] solving proof token...")
        proof_token = build_proof_token(
            proof_info.get("seed", ""),
            proof_info.get("difficulty", ""),
            session.headers.get("User-Agent", ""),
            script_sources=script_sources,
            data_build=data_build,
        )
    # Some deployments may return already-finalized token names; keep them too.
    proof_token = proof_token or data.get("proof_token", "")
    turnstile_token = ""
    turnstile_info = data.get("turnstile") or {}
    if turnstile_info.get("required"):
        log("[chat-requirements] turnstile required but solver is not configured")
    so_token = data.get("so_token", "")
    log(f"[chat-requirements] got token={'yes' if final_token else 'no'}, proof={'yes' if proof_token else 'no'}, so={'yes' if so_token else 'no'}")
    return final_token, proof_token, turnstile_token, so_token


def image_gen(access_token, proxy, prompt, model="gpt-image-2", refresh_token=""):
    """Generate image using ChatGPT web API (matching chatgpt2api exactly).
    
    If access_token is expired (401), automatically refreshes using refresh_token.
    Returns dict with updated tokens if refresh occurred: {"result": ..., "new_access_token": ..., "new_refresh_token": ...}
    """
    base_url = "https://chatgpt.com"
    device_id = new_uuid()
    original_access_token = access_token

    session = build_session(access_token, proxy)
    log(f"[image_gen] proxy={proxy} model={model}")
    token_refreshed = False

    # 1. Bootstrap chatgpt.com and build chat requirements (matching basketikun/chatgpt2api)
    user_agent = session.headers.get("User-Agent", "")
    script_sources, data_build = bootstrap(session, base_url)
    initial_req_token = build_legacy_requirements_token(user_agent, script_sources, data_build)
    sentinel_token, proof_token, turnstile_token, so_token = get_chat_requirements(
        session, base_url, initial_req_token, script_sources, data_build
    )

    # 3. Get conduit token
    conduit_path = "/backend-api/f/conversation/prepare"
    conduit_payload = {
        "action": "next",
        "fork_from_shared_post": False,
        "parent_message_id": new_uuid(),
        "model": image_model_slug(model),
        "client_prepare_state": "success",
        "timezone_offset_min": -480,
        "timezone": "Asia/Shanghai",
        "conversation_mode": {"kind": "primary_assistant"},
        "system_hints": ["picture_v2"],
        "partial_query": {
            "id": new_uuid(),
            "author": {"role": "user"},
            "content": {"content_type": "text", "parts": [prompt]},
        },
        "supports_buffering": True,
        "supported_encodings": ["v1"],
        "client_contextual_info": {"app_name": "chatgpt.com"},
    }

    log(f"[conduit] requesting...")
    resp = session.post(
        base_url + conduit_path,
        json=conduit_payload,
        headers=image_headers(session, conduit_path, sentinel_token, proof_token, turnstile_token=turnstile_token, so_token=so_token),
        timeout=60
    )
    log(f"[conduit] status={resp.status_code}")

    # If 401/403, try refreshing token
    if resp.status_code in (401, 403) and refresh_token:
        log(f"[conduit] got {resp.status_code}, attempting token refresh...")
        try:
            new_access, new_refresh = refresh_access_token(refresh_token, proxy)
            # Rebuild session with new token
            session.close()
            session = build_session(new_access, proxy)
            access_token = new_access
            token_refreshed = True

            # Re-bootstrap and get chat requirements
            user_agent = session.headers.get("User-Agent", "")
            script_sources, data_build = bootstrap(session, base_url)
            initial_req_token = build_legacy_requirements_token(user_agent, script_sources, data_build)
            sentinel_token, proof_token, turnstile_token, so_token = get_chat_requirements(
                session, base_url, initial_req_token, script_sources, data_build
            )

            # Retry conduit
            resp = session.post(
                base_url + conduit_path,
                json=conduit_payload,
                headers=image_headers(session, conduit_path, sentinel_token, proof_token, turnstile_token=turnstile_token, so_token=so_token),
                timeout=60
            )
            log(f"[conduit] retry status={resp.status_code}")
        except Exception as e:
            log(f"[conduit] refresh failed: {e}")
            # Continue with original error

    if resp.status_code != 200:
        log(f"[conduit] response: {resp.text[:300]}")
    resp.raise_for_status()

    conduit_data = resp.json()
    conduit_token = conduit_data.get("conduit_token", "")
    log(f"[conduit] got token={'yes' if conduit_token else 'no'}")

    # 3. Start image generation (SSE)
    gen_path = "/backend-api/f/conversation"
    gen_payload = {
        "action": "next",
        "messages": [{
            "id": new_uuid(),
            "author": {"role": "user"},
            "create_time": time.time(),
            "content": {"content_type": "text", "parts": [prompt]},
            "metadata": {
                "developer_mode_connector_ids": [],
                "selected_github_repos": [],
                "selected_all_github_repos": False,
                "system_hints": ["picture_v2"],
                "serialization_metadata": {"custom_symbol_offsets": []},
            },
        }],
        "parent_message_id": new_uuid(),
        "model": image_model_slug(model),
        "client_prepare_state": "sent",
        "timezone_offset_min": -480,
        "timezone": "Asia/Shanghai",
        "conversation_mode": {"kind": "primary_assistant"},
        "enable_message_followups": True,
        "system_hints": ["picture_v2"],
        "supports_buffering": True,
        "supported_encodings": ["v1"],
        "client_contextual_info": {
            "is_dark_mode": False,
            "time_since_loaded": 1200,
            "page_height": 1072,
            "page_width": 1724,
            "pixel_ratio": 1.2,
            "screen_height": 1440,
            "screen_width": 2560,
            "app_name": "chatgpt.com",
        },
        "paragen_cot_summary_display_override": "allow",
        "force_parallel_switch": "auto",
    }

    log(f"[conversation] starting SSE...")
    resp = session.post(
        base_url + gen_path,
        json=gen_payload,
        headers=image_headers(session, gen_path, sentinel_token, proof_token, conduit_token, "text/event-stream", turnstile_token, so_token),
        timeout=300,
        stream=True
    )
    log(f"[conversation] status={resp.status_code}")
    
    # If 401/403, try refreshing token (if not already refreshed)
    if resp.status_code in (401, 403) and refresh_token and not token_refreshed:
        log(f"[conversation] got {resp.status_code}, attempting token refresh...")
        try:
            new_access, new_refresh = refresh_access_token(refresh_token, proxy)
            # Rebuild session with new token
            session.close()
            session = build_session(new_access, proxy)
            access_token = new_access
            token_refreshed = True

            # Re-bootstrap and get chat requirements
            user_agent = session.headers.get("User-Agent", "")
            script_sources, data_build = bootstrap(session, base_url)
            initial_req_token = build_legacy_requirements_token(user_agent, script_sources, data_build)
            sentinel_token, proof_token, turnstile_token, so_token = get_chat_requirements(
                session, base_url, initial_req_token, script_sources, data_build
            )
            
            # Re-get conduit token
            resp_conduit = session.post(
                base_url + conduit_path,
                json=conduit_payload,
                headers=image_headers(session, conduit_path, sentinel_token, proof_token, turnstile_token=turnstile_token, so_token=so_token),
                timeout=60
            )
            if resp_conduit.status_code == 200:
                conduit_data = resp_conduit.json()
                conduit_token = conduit_data.get("conduit_token", "")
                log(f"[conduit] retry got token={'yes' if conduit_token else 'no'}")

            # Retry conversation
            resp = session.post(
                base_url + gen_path,
                json=gen_payload,
                headers=image_headers(session, gen_path, sentinel_token, proof_token, conduit_token, "text/event-stream", turnstile_token, so_token),
                timeout=300,
                stream=True
            )
            log(f"[conversation] retry status={resp.status_code}")
        except Exception as e:
            log(f"[conversation] refresh failed: {e}")
            # Continue with original error
    
    resp.raise_for_status()

    # 4. Parse SSE stream and extract conversation_id, file_ids, sediment_ids
    conversation_id = ""
    file_ids = []
    sediment_ids = []
    event_count = 0
    
    for line in resp.iter_lines():
        if not line:
            continue
        line = line.decode("utf-8", errors="ignore")
        if not line.startswith("data: "):
            continue
        data_str = line[6:].strip()
        if data_str == "[DONE]":
            log(f"[conversation] got [DONE], {event_count} events")
            break
        event_count += 1
        
        # Extract conversation_id
        if not conversation_id and 'conversation_id' in data_str:
            match = re.search(r'"conversation_id"\s*:\s*"([^"]+)"', data_str)
            if match:
                conversation_id = match.group(1)
        
        # Extract file-service:// and sediment:// IDs
        for fid in re.findall(r'file-service://([A-Za-z0-9_-]+)', data_str):
            if fid not in file_ids:
                file_ids.append(fid)
        for sid in re.findall(r'sediment://([A-Za-z0-9_-]+)', data_str):
            if sid not in sediment_ids:
                sediment_ids.append(sid)

    log(f"[sse] conversation_id={conversation_id}, file_ids={file_ids}, sediment_ids={sediment_ids}")

    # 5. Poll conversation document for stable image IDs (matching basketikun/chatgpt2api)
    if conversation_id and not (file_ids or sediment_ids):
        log("[poll] no IDs in SSE, polling conversation document...")
        time.sleep(10)  # Initial wait for image generation
        
        for attempt in range(12):  # 120s timeout
            try:
                path = f"/backend-api/conversation/{conversation_id}"
                headers = dict(session.headers)
                headers["X-OpenAI-Target-Path"] = path
                headers["X-OpenAI-Target-Route"] = path
                headers["Accept"] = "application/json"
                
                poll_resp = session.get(base_url + path, headers=headers, timeout=60)
                if poll_resp.status_code == 200:
                    conv_data = poll_resp.json()
                    mapping = conv_data.get("mapping", {})
                    
                    for msg_id, msg_data in mapping.items():
                        message = msg_data.get("message") or {}
                        author = message.get("author") or {}
                        if author.get("role") != "tool":
                            continue
                        
                        metadata = message.get("metadata") or {}
                        if metadata.get("async_task_type") != "image_gen":
                            continue
                        
                        content_str = json.dumps(message.get("content", {}))
                        for fid in re.findall(r'file-service://([A-Za-z0-9_-]+)', content_str):
                            if fid not in file_ids:
                                file_ids.append(fid)
                        for sid in re.findall(r'sediment://([A-Za-z0-9_-]+)', content_str):
                            if sid not in sediment_ids:
                                sediment_ids.append(sid)
                    
                    if file_ids or sediment_ids:
                        log(f"[poll] found: file_ids={file_ids}, sediment_ids={sediment_ids}")
                        time.sleep(2)  # Settle time
                        break
                
                time.sleep(10)
            except Exception as e:
                log(f"[poll] attempt {attempt+1} error: {e}")
                time.sleep(10)

    if not file_ids and not sediment_ids:
        raise RuntimeError(f"no image IDs found after SSE and polling ({event_count} events)")

    # 6. Resolve IDs to download URLs
    urls = []
    for file_id in file_ids:
        try:
            path = f"/backend-api/files/{file_id}/download"
            headers = dict(session.headers)
            headers["X-OpenAI-Target-Path"] = path
            headers["X-OpenAI-Target-Route"] = path
            headers["Accept"] = "application/json"
            
            dl_resp = session.get(base_url + path, headers=headers, timeout=60)
            if dl_resp.status_code == 200:
                data = dl_resp.json()
                url = data.get("download_url") or data.get("url")
                if url:
                    urls.append(url)
                    log(f"[resolve] file {file_id} -> URL")
        except Exception as e:
            log(f"[resolve] file {file_id} failed: {e}")
    
    for sediment_id in sediment_ids:
        try:
            path = f"/backend-api/conversation/{conversation_id}/attachment/{sediment_id}/download"
            headers = dict(session.headers)
            headers["X-OpenAI-Target-Path"] = path
            headers["X-OpenAI-Target-Route"] = path
            headers["Accept"] = "application/json"
            
            dl_resp = session.get(base_url + path, headers=headers, timeout=60)
            if dl_resp.status_code == 200:
                data = dl_resp.json()
                url = data.get("download_url") or data.get("url")
                if url:
                    urls.append(url)
                    log(f"[resolve] sediment {sediment_id} -> URL")
        except Exception as e:
            log(f"[resolve] sediment {sediment_id} failed: {e}")

    if not urls:
        raise RuntimeError("no download URLs resolved")

    # 7. Download first image
    log(f"[download] downloading from {len(urls)} URL(s)...")
    dl_resp = session.get(urls[0], timeout=120)
    log(f"[download] status={dl_resp.status_code} len={len(dl_resp.content)}")
    dl_resp.raise_for_status()

    session.close()

    image_b64 = base64.b64encode(dl_resp.content).decode("utf-8")
    result = {"b64_json": image_b64, "revised_prompt": prompt}

    # Return updated tokens if refresh happened
    output = {"result": result}
    if token_refreshed and access_token != original_access_token:
        output["new_access_token"] = access_token
    return output


def main():
    """Read JSON from stdin, output JSON to stdout."""
    input_data = json.loads(sys.stdin.read())
    access_token = input_data["access_token"]
    proxy = input_data.get("proxy", "")
    prompt = input_data["prompt"]
    model = input_data.get("model", "gpt-image-2")
    refresh_token = input_data.get("refresh_token", "")

    try:
        output = image_gen(access_token, proxy, prompt, model, refresh_token)
        result = output["result"]
        # Output format matching Go expectations
        resp = {
            "success": True,
            "image_b64": result.get("b64_json", ""),
            "image_url": "",  # We return base64, not URL
            "error": ""
        }
        # Include updated tokens if refresh occurred
        if "new_access_token" in output:
            resp["new_access_token"] = output["new_access_token"]
        print(json.dumps(resp))
    except Exception as e:
        output = {"success": False, "image_b64": "", "image_url": "", "error": str(e)}
        print(json.dumps(output))
        sys.exit(1)


if __name__ == "__main__":
    main()
