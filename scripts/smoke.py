#!/usr/bin/env python3
from __future__ import annotations

import argparse
import base64
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Callable


REPO = Path(__file__).resolve().parents[1]
PNG_BYTES = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="
)


class HTTPStatusError(Exception):
    def __init__(self, status: int, body: str) -> None:
        self.status = status
        self.body = body
        try:
            self.json_body = json.loads(body)
        except json.JSONDecodeError:
            self.json_body = None
        super().__init__(f"HTTP {status}: {body}")


def assert_true(condition: bool, message: str) -> None:
    if not condition:
        raise RuntimeError(message)


def run(cmd: list[str], *, cwd: Path = REPO, env: dict[str, str] | None = None) -> None:
    subprocess.run(cmd, cwd=cwd, env=env, check=True)


def request_json(
    method: str,
    url: str,
    *,
    token: str | None = None,
    body: dict[str, Any] | None = None,
    timeout: float = 15,
) -> Any:
    headers: dict[str, str] = {}
    data = None
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body, separators=(",", ":")).encode("utf-8")
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            raw = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        raise HTTPStatusError(exc.code, raw) from exc
    if not raw.strip():
        return None
    return json.loads(raw)


def assert_http_error_code(call: Callable[[], Any], status: int, code: str) -> None:
    try:
        call()
    except HTTPStatusError as exc:
        if exc.status != status:
            raise RuntimeError(f"expected HTTP {status}, got {exc.status}") from exc
        body = exc.json_body
        actual = None
        if isinstance(body, dict):
            actual = ((body.get("error") or {}) if isinstance(body.get("error"), dict) else {}).get("code")
        if actual != code:
            raise RuntimeError(f"expected error code {code}, got {actual}") from exc
        return
    raise RuntimeError(f"expected HTTP {status} with error code {code}, got success")


def upload_image(base_url: str, token: str, image_path: Path, alt_text: str) -> dict[str, Any]:
    completed = subprocess.run(
        [
            "curl",
            "-sS",
            "-X",
            "POST",
            f"{base_url}/api/v1/uploads/images",
            "-H",
            f"Authorization: Bearer {token}",
            "-F",
            f"file=@{image_path}",
            "-F",
            f"alt_text={alt_text}",
        ],
        cwd=REPO,
        text=True,
        capture_output=True,
    )
    if completed.returncode != 0:
        raise RuntimeError(f"curl upload failed: {completed.stderr.strip() or completed.stdout.strip()}")
    try:
        return json.loads(completed.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"curl upload returned non-JSON output: {completed.stdout}") from exc


def build_api(binary: Path) -> None:
    run(["go", "build", "-buildvcs=false", "-o", str(binary), "./cmd/api"])


def migrate_if_needed(skip_migration: bool) -> None:
    if not skip_migration:
        run(["go", "run", "./cmd/migrate", "up"])


def start_api(binary: Path, env: dict[str, str], stdout_log: Path, stderr_log: Path) -> subprocess.Popen[bytes]:
    stdout = stdout_log.open("wb")
    stderr = stderr_log.open("wb")
    try:
        return subprocess.Popen([str(binary)], cwd=REPO, env=env, stdout=stdout, stderr=stderr)
    finally:
        stdout.close()
        stderr.close()


def stop_process(process: subprocess.Popen[bytes] | None) -> None:
    if not process or process.poll() is not None:
        return
    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)


def wait_ready(process: subprocess.Popen[bytes], base_url: str, stdout_log: Path, stderr_log: Path) -> None:
    deadline = time.time() + 30
    while time.time() < deadline:
        if process.poll() is not None:
            stdout = stdout_log.read_text(encoding="utf-8", errors="replace") if stdout_log.exists() else ""
            stderr = stderr_log.read_text(encoding="utf-8", errors="replace") if stderr_log.exists() else ""
            raise RuntimeError(f"API stopped before readiness: exit={process.returncode}\nSTDOUT:\n{stdout}\nSTDERR:\n{stderr}")
        try:
            health = request_json("GET", f"{base_url}/healthz", timeout=2)
            if health.get("status") == "ok":
                return
        except Exception:
            time.sleep(0.5)
    raise RuntimeError(f"API did not become ready at {base_url}")


def smoke_env(port: int, token_secret: str, storage: dict[str, str]) -> dict[str, str]:
    env = os.environ.copy()
    env.update(
        {
            "APP_NAME": "cumt-nexus-api-smoke",
            "APP_ENV": "local",
            "APP_STARTUP_TIMEOUT": "10s",
            "HTTP_ADDR": f"127.0.0.1:{port}",
            "AUTH_TOKEN_SECRET": token_secret,
            "UPLOAD_IMAGE_MAX_BYTES": "5242880",
            "UPLOAD_IMAGE_MAX_COUNT_PER_POST": "9",
            "UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT": "1",
        }
    )
    env.update(storage)
    return env


def with_api(prefix: str, port: int, skip_migration: bool, env: dict[str, str], action: Callable[[str, Path], dict[str, Any]]) -> dict[str, Any]:
    tmp_base = REPO / ".tmp"
    tmp_base.mkdir(exist_ok=True)
    root = Path(tempfile.mkdtemp(prefix=f"{prefix}-", dir=tmp_base))
    binary = root / "api-smoke"
    process: subprocess.Popen[bytes] | None = None
    try:
        migrate_if_needed(skip_migration)
        build_api(binary)
        base_url = f"http://127.0.0.1:{port}"
        stdout_log = root / "api.stdout.log"
        stderr_log = root / "api.stderr.log"
        process = start_api(binary, env, stdout_log, stderr_log)
        wait_ready(process, base_url, stdout_log, stderr_log)
        return action(base_url, root)
    finally:
        stop_process(process)
        if root.is_relative_to(tmp_base):
            shutil.rmtree(root, ignore_errors=True)


def register(base_url: str, username: str) -> str:
    response = request_json(
        "POST",
        f"{base_url}/api/v1/auth/register",
        body={"username": username, "password": "password123"},
    )
    token = response.get("access_token")
    assert_true(bool(token), "register did not return access_token")
    return token


def stage13(args: argparse.Namespace) -> None:
    port = args.port
    uploads = REPO / ".tmp" / f"s13-uploads-{int(time.time() * 1000)}"
    uploads.mkdir(parents=True, exist_ok=True)
    env = smoke_env(
        port,
        "SmokeAuthSecretForStage13",
        {
            "OBJECT_STORAGE_PROVIDER": "local",
            "OBJECT_STORAGE_PUBLIC_BASE_URL": f"http://127.0.0.1:{port}/uploads",
            "OBJECT_STORAGE_LOCAL_ROOT": str(uploads),
        },
    )

    def action(base_url: str, root: Path) -> dict[str, Any]:
        suffix = time.strftime("%y%m%d%H%M%S") + f"{int(time.time() * 1000) % 1000:03d}"
        username = f"s13_smoke_{suffix}"
        token = register(base_url, username)
        png_path = root / "smoke.png"
        png_path.write_bytes(PNG_BYTES)

        post_upload = upload_image(base_url, token, png_path, "post smoke")
        post_attachment_id = post_upload["attachment"]["id"]
        assert_true(bool(post_attachment_id), f"post upload missing attachment id: {post_upload}")

        post = request_json(
            "POST",
            f"{base_url}/api/v1/communities/public/posts",
            token=token,
            body={
                "title": "Stage 13 smoke post",
                "body": "Markdown-like post body",
                "attachment_ids": [post_attachment_id],
            },
        )
        post_id = post["post"]["id"]
        assert_true(post["post"]["format"] == "nexus_markdown", "expected post format nexus_markdown")
        assert_true(len(post["post"].get("attachments") or []) == 1, "post create did not return one attachment")

        root_upload = upload_image(base_url, token, png_path, "root comment smoke")
        root_attachment_id = root_upload["attachment"]["id"]
        root_comment = request_json(
            "POST",
            f"{base_url}/api/v1/posts/{post_id}/comments",
            token=token,
            body={"body": "Markdown-like root comment body", "attachment_ids": [root_attachment_id]},
        )
        root_comment_id = root_comment["comment"]["id"]
        assert_true(len(root_comment["comment"].get("attachments") or []) == 1, "root comment attachment missing")

        child_upload = upload_image(base_url, token, png_path, "child comment smoke")
        child_attachment_id = child_upload["attachment"]["id"]
        child = request_json(
            "POST",
            f"{base_url}/api/v1/posts/{post_id}/comments",
            token=token,
            body={
                "body": "Markdown-like child comment body",
                "parent_id": root_comment_id,
                "attachment_ids": [child_attachment_id],
            },
        )
        child_comment_id = child["comment"]["id"]
        assert_true(len(child["comment"].get("attachments") or []) == 1, "child comment attachment missing")

        tree = request_json(
            "GET",
            f"{base_url}/api/v1/posts/{post_id}/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6",
            token=token,
        )
        comments = tree.get("comments") or []
        root_from_tree = next((item for item in comments if item.get("id") == root_comment_id), None)
        child_from_tree = next((item for item in comments if item.get("id") == child_comment_id), None)
        assert_true(tree.get("view") == "tree", f"expected tree view, got {tree.get('view')}")
        assert_true(root_from_tree is not None, "root comment missing from tree")
        assert_true(child_from_tree is not None, "child comment missing from tree")
        assert_true(root_from_tree.get("parent_id") is None, "root parent_id should be null")
        assert_true(root_from_tree.get("depth") == 0, "root depth should be 0")
        assert_true(child_from_tree.get("parent_id") == root_comment_id, "child parent_id mismatch")
        assert_true(child_from_tree.get("depth") == 1, "child depth should be 1")
        assert_true(len(root_from_tree.get("attachments") or []) == 1, "root tree attachment missing")
        assert_true(len(child_from_tree.get("attachments") or []) == 1, "child tree attachment missing")
        assert_true(root_from_tree.get("format") == "nexus_markdown", "root format should be nexus_markdown")
        assert_true(child_from_tree.get("format") == "nexus_markdown", "child format should be nexus_markdown")
        root_index = next((idx for idx, item in enumerate(comments) if item.get("id") == root_comment_id), -1)
        child_index = next((idx for idx, item in enumerate(comments) if item.get("id") == child_comment_id), -1)
        assert_true(root_index >= 0 and child_index >= 0 and root_index < child_index, "tree preorder is invalid")

        assert_http_error_code(
            lambda: request_json(
                "GET",
                f"{base_url}/api/v1/posts/{post_id}/comments?view=nested",
                token=token,
            ),
            400,
            "invalid_argument",
        )

        return {
            "status": "passed",
            "user": username,
            "post_id": post_id,
            "post_attachment_id": post_attachment_id,
            "root_comment_id": root_comment_id,
            "root_attachment_id": root_attachment_id,
            "child_comment_id": child_comment_id,
            "child_attachment_id": child_attachment_id,
            "tree_view": tree.get("view"),
            "storage_provider": "local",
            "base_url": base_url,
        }

    try:
        result = with_api("s13-smoke", port, args.skip_migration, env, action)
        print(json.dumps(result, separators=(",", ":")))
    finally:
        shutil.rmtree(uploads, ignore_errors=True)


def stage14(args: argparse.Namespace) -> None:
    port = args.port
    uploads = REPO / ".tmp" / f"s14-uploads-{int(time.time() * 1000)}"
    uploads.mkdir(parents=True, exist_ok=True)
    env = smoke_env(
        port,
        "SmokeAuthSecretForStage14",
        {
            "OBJECT_STORAGE_PROVIDER": "local",
            "OBJECT_STORAGE_PUBLIC_BASE_URL": f"http://127.0.0.1:{port}/uploads",
            "OBJECT_STORAGE_LOCAL_ROOT": str(uploads),
        },
    )

    def action(base_url: str, _root: Path) -> dict[str, Any]:
        suffix = time.strftime("%y%m%d%H%M%S") + f"{int(time.time() * 1000) % 1000:03d}"
        author_name = f"s14_author_{suffix}"
        intruder_name = f"s14_intruder_{suffix}"
        author_token = register(base_url, author_name)
        intruder_token = register(base_url, intruder_name)

        post = request_json(
            "POST",
            f"{base_url}/api/v1/communities/public/posts",
            token=author_token,
            body={"title": "Stage 14 smoke post", "body": "Original Markdown-like post body"},
        )
        post_id = post["post"]["id"]
        comment = request_json(
            "POST",
            f"{base_url}/api/v1/posts/{post_id}/comments",
            token=author_token,
            body={"body": "Original Markdown-like comment body"},
        )
        comment_id = comment["comment"]["id"]

        assert_http_error_code(
            lambda: request_json(
                "PATCH",
                f"{base_url}/api/v1/posts/{post_id}",
                token=intruder_token,
                body={"title": "Intruder title", "body": "Intruder body"},
            ),
            403,
            "forbidden",
        )

        updated_post = request_json(
            "PATCH",
            f"{base_url}/api/v1/posts/{post_id}",
            token=author_token,
            body={"title": "Stage 14 smoke post updated", "body": "Updated Markdown-like post body"},
        )
        assert_true(updated_post["post"]["title"] == "Stage 14 smoke post updated", "post title was not updated")
        assert_true(updated_post["post"]["body"] == "Updated Markdown-like post body", "post body was not updated")
        assert_true(updated_post["post"]["format"] == "nexus_markdown", "post format should remain nexus_markdown")
        detail = request_json("GET", f"{base_url}/api/v1/posts/{post_id}", token=author_token)
        assert_true(detail["post"]["title"] == "Stage 14 smoke post updated", "post detail did not reflect update")

        assert_http_error_code(
            lambda: request_json(
                "PATCH",
                f"{base_url}/api/v1/comments/{comment_id}",
                token=intruder_token,
                body={"body": "Intruder comment body"},
            ),
            403,
            "forbidden",
        )
        updated_comment = request_json(
            "PATCH",
            f"{base_url}/api/v1/comments/{comment_id}",
            token=author_token,
            body={"body": "Updated Markdown-like comment body"},
        )
        assert_true(updated_comment["comment"]["body"] == "Updated Markdown-like comment body", "comment body was not updated")
        assert_true(updated_comment["comment"]["format"] == "nexus_markdown", "comment format should remain nexus_markdown")
        tree = request_json(
            "GET",
            f"{base_url}/api/v1/posts/{post_id}/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6",
            token=author_token,
        )
        comment_from_tree = next((item for item in tree.get("comments") or [] if item.get("id") == comment_id), None)
        assert_true(comment_from_tree is not None, "updated comment missing from tree")
        assert_true(comment_from_tree.get("body") == "Updated Markdown-like comment body", "tree did not reflect update")

        assert_http_error_code(
            lambda: request_json("DELETE", f"{base_url}/api/v1/comments/{comment_id}", token=intruder_token),
            403,
            "forbidden",
        )
        request_json("DELETE", f"{base_url}/api/v1/comments/{comment_id}", token=author_token)
        after_delete = request_json(
            "GET",
            f"{base_url}/api/v1/posts/{post_id}/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6",
            token=author_token,
        )
        deleted_comment = next((item for item in after_delete.get("comments") or [] if item.get("id") == comment_id), None)
        assert_true(deleted_comment is None, "deleted comment should not appear in tree")

        assert_http_error_code(
            lambda: request_json("DELETE", f"{base_url}/api/v1/posts/{post_id}", token=intruder_token),
            403,
            "forbidden",
        )
        request_json("DELETE", f"{base_url}/api/v1/posts/{post_id}", token=author_token)
        assert_http_error_code(
            lambda: request_json("GET", f"{base_url}/api/v1/posts/{post_id}", token=author_token),
            404,
            "not_found",
        )

        return {
            "status": "passed",
            "author": author_name,
            "intruder": intruder_name,
            "post_id": post_id,
            "comment_id": comment_id,
            "base_url": base_url,
        }

    try:
        result = with_api("s14-smoke", port, args.skip_migration, env, action)
        print(json.dumps(result, separators=(",", ":")))
    finally:
        shutil.rmtree(uploads, ignore_errors=True)


def read_dotenv(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    if not path.exists():
        return values
    for line in path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or "=" not in stripped:
            continue
        key, value = stripped.split("=", 1)
        value = value.strip()
        if (value.startswith('"') and value.endswith('"')) or (value.startswith("'") and value.endswith("'")):
            value = value[1:-1]
        values[key.strip()] = value
    return values


def config_value(key: str, dotenv: dict[str, str], default: str = "") -> str:
    value = os.environ.get(key)
    if value and value.strip():
        return value.strip()
    value = dotenv.get(key, "")
    if value and value.strip():
        return value.strip()
    return default


def public_image_readable(url: str) -> dict[str, Any]:
    method = "HEAD"
    request = urllib.request.Request(url, method="HEAD")
    try:
        response = urllib.request.urlopen(request, timeout=15)
    except Exception:
        method = "GET"
        request = urllib.request.Request(url, method="GET")
        try:
            response = urllib.request.urlopen(request, timeout=15)
        except Exception as exc:
            raise RuntimeError(f"attachment public url is not readable: {url} error={exc}") from exc
    with response:
        status = response.status
        content_type = response.headers.get("Content-Type", "")
    assert_true(status == 200, f"attachment public url returned HTTP {status}: {url}")
    assert_true(bool(content_type.strip()), f"attachment public url did not return Content-Type: {url}")
    assert_true(content_type.startswith("image/"), f"attachment public url returned non-image Content-Type {content_type}: {url}")
    return {"method": method, "status_code": status, "content_type": content_type}


def stage15(args: argparse.Namespace) -> None:
    port = args.port
    dotenv = read_dotenv(REPO / ".env")
    provider = config_value("OBJECT_STORAGE_PROVIDER", dotenv)
    if provider != "r2":
        result = {
            "status": "skipped",
            "reason": "r2_not_configured",
            "missing": ["OBJECT_STORAGE_PROVIDER=r2"],
        }
        if args.skip_when_missing_credentials:
            print(json.dumps(result, separators=(",", ":")))
            return
        raise RuntimeError("OBJECT_STORAGE_PROVIDER must be r2 for Stage 15 R2 smoke")

    r2 = {
        "OBJECT_STORAGE_ENDPOINT": config_value("OBJECT_STORAGE_ENDPOINT", dotenv),
        "OBJECT_STORAGE_REGION": config_value("OBJECT_STORAGE_REGION", dotenv, "auto"),
        "OBJECT_STORAGE_BUCKET": config_value("OBJECT_STORAGE_BUCKET", dotenv),
        "OBJECT_STORAGE_ACCESS_KEY_ID": config_value("OBJECT_STORAGE_ACCESS_KEY_ID", dotenv),
        "OBJECT_STORAGE_SECRET_ACCESS_KEY": config_value("OBJECT_STORAGE_SECRET_ACCESS_KEY", dotenv),
        "OBJECT_STORAGE_PUBLIC_BASE_URL": config_value("OBJECT_STORAGE_PUBLIC_BASE_URL", dotenv),
        "OBJECT_STORAGE_FORCE_PATH_STYLE": config_value("OBJECT_STORAGE_FORCE_PATH_STYLE", dotenv, "true"),
    }
    required = [
        "OBJECT_STORAGE_ENDPOINT",
        "OBJECT_STORAGE_BUCKET",
        "OBJECT_STORAGE_ACCESS_KEY_ID",
        "OBJECT_STORAGE_SECRET_ACCESS_KEY",
        "OBJECT_STORAGE_PUBLIC_BASE_URL",
    ]
    missing = [key for key in required if not r2.get(key)]
    if missing:
        result = {"status": "skipped", "reason": "missing_r2_credentials", "missing": missing}
        if args.skip_when_missing_credentials:
            print(json.dumps(result, separators=(",", ":")))
            return
        raise RuntimeError("missing R2 configuration: " + ", ".join(missing))

    env = smoke_env(
        port,
        "SmokeAuthSecretForStage15R2",
        {
            "OBJECT_STORAGE_PROVIDER": "r2",
            **r2,
        },
    )
    env["APP_NAME"] = "cumt-nexus-api-r2-smoke"

    def action(base_url: str, root: Path) -> dict[str, Any]:
        suffix = time.strftime("%y%m%d%H%M%S") + f"{int(time.time() * 1000) % 1000:03d}"
        username = f"s15_r2_smoke_{suffix}"
        token = register(base_url, username)
        png_path = root / "r2-smoke.png"
        png_path.write_bytes(PNG_BYTES)

        upload = upload_image(base_url, token, png_path, "r2 smoke")
        attachment = upload["attachment"]
        attachment_id = attachment["id"]
        assert_true(bool(attachment_id), f"upload missing attachment id: {upload}")
        assert_true(attachment.get("status") == "ready", f"expected ready attachment, got {attachment.get('status')}")
        assert_true(attachment.get("mime_type") == "image/png", f"expected image/png, got {attachment.get('mime_type')}")

        public_base = r2["OBJECT_STORAGE_PUBLIC_BASE_URL"].rstrip("/")
        assert_true(attachment["url"].startswith(public_base + "/"), "attachment url does not use configured public base url")
        public_url_check = public_image_readable(attachment["url"])

        post = request_json(
            "POST",
            f"{base_url}/api/v1/communities/public/posts",
            token=token,
            body={"title": "Stage 15 R2 smoke post", "body": "R2 smoke Markdown-like post body", "attachment_ids": [attachment_id]},
        )
        post_id = post["post"]["id"]
        attachments = post["post"].get("attachments") or []
        assert_true(len(attachments) == 1, "post create did not return bound attachment")
        assert_true(attachments[0].get("id") == attachment_id, "post attachment id mismatch")
        assert_true(attachments[0].get("url", "").startswith(public_base + "/"), "post attachment url does not use configured public base url")

        return {
            "status": "passed",
            "user": username,
            "post_id": post_id,
            "attachment_id": attachment_id,
            "attachment_url": attachment["url"],
            "public_url_check": public_url_check,
            "storage_provider": "r2",
            "base_url": base_url,
        }

    result = with_api("s15-r2-smoke", port, args.skip_migration, env, action)
    print(json.dumps(result, separators=(",", ":")))


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Run API smoke scenarios.")
    sub = parser.add_subparsers(dest="command", required=True)

    for name, func, default_port in [
        ("stage13", stage13, 18080),
        ("stage14", stage14, 18081),
        ("stage15", stage15, 18082),
    ]:
        cmd = sub.add_parser(name)
        cmd.add_argument("--port", type=int, default=default_port)
        cmd.add_argument("--skip-migration", action="store_true")
        if name == "stage15":
            cmd.add_argument("--skip-when-missing-credentials", action="store_true")
        cmd.set_defaults(func=func)

    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
