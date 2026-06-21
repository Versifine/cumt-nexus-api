#!/usr/bin/env python3
from __future__ import annotations

import argparse
import ast
import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any


REPO = Path(__file__).resolve().parents[1]


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8-sig")


def read_lines(path: Path) -> list[str]:
    return read_text(path).splitlines()


def rel(path: Path) -> str:
    return path.resolve().relative_to(REPO).as_posix()


def sorted_values(values: Any) -> list[Any]:
    return sorted(list(values))


def print_json(payload: dict[str, Any]) -> None:
    print(json.dumps(payload, ensure_ascii=False, indent=2))


def fail(payload: dict[str, Any], message: str) -> None:
    payload = {"status": "failed", **payload}
    print_json(payload)
    raise SystemExit(message)


def route_key(method: str, path: str) -> str:
    return f"{method.upper()} {path}"


@dataclass
class Route:
    method: str
    path: str
    auth: str
    handler: str = ""
    source: str = ""


def add_route(
    routes: dict[str, Route],
    method: str,
    path: str,
    auth: str,
    handler: str = "",
    source: str = "",
) -> None:
    key = route_key(method, path)
    if key in routes:
        raise RuntimeError(f"duplicate API route: {key}")
    routes[key] = Route(method.upper(), path, auth, handler, source)


def add_routes_from_file(
    routes: dict[str, Route],
    source: str,
    prefix: str,
    auth: str,
    include_handlers: set[str] | None = None,
) -> None:
    content = read_text(REPO / source)
    for match in re.finditer(r'group\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)",\s*handler\.(\w+)\)', content):
        method, path, handler = match.group(1), match.group(2), match.group(3)
        if include_handlers is not None and handler not in include_handlers:
            continue
        add_route(routes, method, prefix + path, auth, handler, source)


def find_handler_signature(content: str, handler: str) -> re.Match[str] | None:
    pattern = rf"func\s+\(h \*Handler\)\s+{re.escape(handler)}\s*\(c \*gin\.Context(?:,\s*[^)]*)?\)\s*\{{"
    return re.search(pattern, content)


def brace_body(content: str, signature: re.Match[str]) -> str:
    body_start = signature.end()
    depth = 1
    for idx in range(body_start, len(content)):
        char = content[idx]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return content[body_start:idx]
    return ""


def get_handler_body(source: str, handler: str, seen: set[tuple[str, str]] | None = None) -> str:
    if not source or not handler:
        return ""
    seen = seen or set()
    identity = (source, handler)
    if identity in seen:
        return ""
    seen.add(identity)

    source_path = REPO / source
    candidates = [source_path]
    candidates.extend(
        sorted(
            path
            for path in source_path.parent.glob("*.go")
            if path != source_path and not path.name.endswith("_test.go")
        )
    )
    for candidate in candidates:
        content = read_text(candidate)
        signature = find_handler_signature(content, handler)
        if not signature:
            continue
        body = brace_body(content, signature)
        for helper in re.findall(r"\bh\.(listModQueue|listUserStates)\(c", body):
            helper_body = get_handler_body(source, helper, seen)
            if helper_body.strip():
                body += "\n" + helper_body
        return body
    return ""


def query_params_from_body(body: str) -> set[str]:
    params = set(re.findall(r'\b(?:Query|DefaultQuery|GetQuery)\("([^"]+)"', body))
    params.update(re.findall(r'\b\w*Query\w*\(c,\s*"([^"]+)"', body))
    return params


def read_query_param_docs(path: Path) -> dict[str, list[str]]:
    docs: dict[str, list[str]] = {}
    for line in read_lines(path):
        match = re.match(r"^\|\s*`([^`]+)`\s*\|\s*((?:`[^`]+`(?:,\s*)?)+)\s*\|", line)
        if not match:
            continue
        key = match.group(1).strip()
        if key in docs:
            raise RuntimeError(f"duplicate API query parameter doc: {key}")
        docs[key] = sorted(re.findall(r"`([^`]+)`", match.group(2)))
    return docs


def actual_api_routes() -> dict[str, Route]:
    routes: dict[str, Route] = {}

    if re.search(r'router\.GET\("/healthz"', read_text(REPO / "internal/platform/httpserver/health.go")):
        add_route(routes, "GET", "/healthz", "public")

    if re.search(r'router\.Static\("/uploads"', read_text(REPO / "cmd/api/main.go")):
        add_route(routes, "GET", "/uploads/*filepath", "public, local only")

    add_routes_from_file(routes, "internal/auth/delivery/authhttp/register.go", "/api/v1/auth", "public")

    for source in [
        "internal/auth/delivery/authhttp/security.go",
        "internal/admin/delivery/adminhttp/handler.go",
        "internal/vote/delivery/votehttp/handler.go",
        "internal/moderation/delivery/moderationhttp/handler.go",
        "internal/notification/delivery/notificationhttp/handler.go",
        "internal/media/delivery/mediahttp/handler.go",
        "internal/contentref/delivery/contentrefhttp/handler.go",
    ]:
        add_routes_from_file(routes, source, "/api/v1", "Bearer")

    add_routes_from_file(
        routes,
        "internal/message/delivery/messagehttp/handler.go",
        "/api/v1",
        "Bearer",
        {
            "GetSummary",
            "ListConversations",
            "StartConversation",
            "ListMessages",
            "SendMessage",
            "MarkConversationRead",
            "ArchiveConversation",
            "UnarchiveConversation",
            "PinConversation",
            "UnpinConversation",
            "MuteConversation",
            "UnmuteConversation",
            "DeleteConversation",
            "ReportConversation",
            "AcceptRequest",
            "RejectRequest",
            "RecallMessage",
            "DeleteMessage",
            "ReportMessage",
            "BlockUser",
            "UnblockUser",
            "GetPrivacy",
            "UpdatePrivacy",
            "CreateRealtimeTicket",
        },
    )
    add_routes_from_file(
        routes,
        "internal/message/delivery/messagehttp/handler.go",
        "/api/v1",
        "ticket",
        {"RealtimeMessages"},
    )
    add_routes_from_file(
        routes,
        "internal/user/delivery/userhttp/handler.go",
        "/api/v1",
        "optional Bearer",
        {"GetPublicUser"},
    )
    add_routes_from_file(
        routes,
        "internal/user/delivery/userhttp/handler.go",
        "/api/v1",
        "Bearer",
        {"Me", "UpdateProfile", "ListFollowedUsers", "FollowUser", "DeleteUserFollow"},
    )
    add_routes_from_file(
        routes,
        "internal/community/delivery/communityhttp/handler.go",
        "/api/v1",
        "optional Bearer",
        {"ListCommunities", "GetCommunity"},
    )
    add_routes_from_file(
        routes,
        "internal/community/delivery/communityhttp/handler.go",
        "/api/v1",
        "Bearer",
        {
            "SubmitCommunityApplication",
            "ListCommunityApplications",
            "GetCommunityApplication",
            "ApproveCommunityApplication",
            "RejectCommunityApplication",
            "ListFollowedCommunities",
            "ListCommunityOwnerTransfers",
            "FollowCommunity",
            "DeleteCommunityFollow",
            "GetCommunityManageContext",
            "ListCommunityMembers",
            "AddCommunityModerator",
            "RemoveCommunityModerator",
            "GetCurrentCommunityOwnerTransfer",
            "GetCommunityOwnerTransfer",
            "CreateCommunityOwnerTransfer",
            "AcceptCommunityOwnerTransfer",
            "CancelCommunityOwnerTransfer",
            "ListCommunityManagePosts",
            "ListCommunityManageComments",
            "ListCommunityManageReports",
            "GetCommunityManageSettings",
            "UpdateCommunityManageSettings",
            "ListCommunityRules",
            "CreateCommunityRule",
            "UpdateCommunityRule",
            "DeleteCommunityRule",
        },
    )
    add_routes_from_file(
        routes,
        "internal/post/delivery/posthttp/handler.go",
        "/api/v1",
        "optional Bearer",
        {"ListCommunityPosts", "ListLatestPosts", "ListUserPosts", "GetPost"},
    )
    add_routes_from_file(
        routes,
        "internal/post/delivery/posthttp/handler.go",
        "/api/v1",
        "Bearer",
        {"PublishPost", "ListSavedPosts", "SavePost", "DeletePostSave", "UpdatePost", "DeletePost"},
    )
    add_routes_from_file(
        routes,
        "internal/comment/delivery/commenthttp/handler.go",
        "/api/v1",
        "optional Bearer",
        {"ListPostComments", "ListUserComments"},
    )
    add_routes_from_file(routes, "internal/search/delivery/searchhttp/handler.go", "/api/v1", "optional Bearer")
    add_routes_from_file(
        routes,
        "internal/effect/delivery/effecthttp/handler.go",
        "/api/v1",
        "optional Bearer",
        {"ListEffectsCatalog"},
    )
    add_routes_from_file(
        routes,
        "internal/comment/delivery/commenthttp/handler.go",
        "/api/v1",
        "Bearer",
        {"PublishComment", "SetCommentVote", "DeleteCommentVote", "UpdateComment", "DeleteComment"},
    )
    add_routes_from_file(
        routes,
		"internal/effect/delivery/effecthttp/handler.go",
		"/api/v1",
		"Bearer",
		{"GetMyPoints", "ListMyPointTransactions", "ApplyPostEffect", "ApplyCommentEffect"},
	)
    add_routes_from_file(routes, "internal/progression/delivery/progressionhttp/handler.go", "/api/v1", "Bearer")

    return routes


def verify_api_contract(args: argparse.Namespace) -> None:
    doc_path = REPO / args.doc_path
    if not doc_path.exists():
        raise SystemExit(f"contract doc not found: {args.doc_path}")

    actual_routes = actual_api_routes()
    documented_routes: dict[str, Route] = {}
    for line in read_lines(doc_path):
        match = re.match(r"^\|\s*(GET|POST|PUT|PATCH|DELETE)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|", line)
        if not match:
            continue
        add_route(
            documented_routes,
            match.group(1),
            match.group(2).strip(),
            match.group(3).strip(),
        )

    missing_in_doc = sorted(key for key in actual_routes if key not in documented_routes)
    stale_in_doc = sorted(key for key in documented_routes if key not in actual_routes)
    auth_mismatches = []
    actual_query_docs: dict[str, list[str]] = {}
    documented_query_docs = read_query_param_docs(doc_path)

    for key in sorted(actual_routes):
        if key in documented_routes and actual_routes[key].auth != documented_routes[key].auth:
            auth_mismatches.append(
                {"route": key, "actual": actual_routes[key].auth, "documented": documented_routes[key].auth}
            )
        route = actual_routes[key]
        params = query_params_from_body(get_handler_body(route.source, route.handler))
        if params:
            actual_query_docs[key] = sorted(params)

    missing_query_docs = sorted(key for key in actual_query_docs if key not in documented_query_docs)
    stale_query_docs = sorted(key for key in documented_query_docs if key not in actual_query_docs)
    query_mismatches = []
    for key in sorted(actual_query_docs):
        if key not in documented_query_docs:
            continue
        if actual_query_docs[key] != sorted(documented_query_docs[key]):
            query_mismatches.append(
                {"route": key, "actual": actual_query_docs[key], "documented": sorted(documented_query_docs[key])}
            )

    if any([missing_in_doc, stale_in_doc, auth_mismatches, missing_query_docs, stale_query_docs, query_mismatches]):
        fail(
            {
                "missing_in_doc": missing_in_doc,
                "stale_in_doc": stale_in_doc,
                "auth_mismatches": auth_mismatches,
                "missing_query_param_docs": missing_query_docs,
                "stale_query_param_docs": stale_query_docs,
                "query_param_mismatches": query_mismatches,
            },
            "API contract doc route/auth/query inventory is out of sync",
        )

    print_json(
        {
            "status": "passed",
            "route_count": len(actual_routes),
            "query_param_route_count": len(actual_query_docs),
            "doc": args.doc_path,
            "routes": sorted(actual_routes),
            "auth_boundaries": [
                {"Method": route.method, "Path": route.path, "Auth": route.auth}
                for route in sorted(actual_routes.values(), key=lambda item: (item.method, item.path))
            ],
            "query_params": [
                {"route": key, "params": actual_query_docs[key]} for key in sorted(actual_query_docs)
            ],
        }
    )


@dataclass
class Schema:
    package: str
    type: str
    fields: list[str]
    required_fields: list[str]
    source: str


def schema_key(package: str, type_name: str) -> str:
    return f"{package}.{type_name}"


def struct_schemas_from_file(path: Path) -> list[Schema]:
    content = read_text(path)
    package_match = re.search(r"(?m)^package\s+(\w+)", content)
    if not package_match:
        return []
    package = package_match.group(1)
    schemas: list[Schema] = []
    for struct_match in re.finditer(r"(?s)type\s+(\w+)\s+struct\s*\{(.*?)\n\}", content):
        type_name = struct_match.group(1)
        fields: list[str] = []
        required_fields: list[str] = []
        for tag_match in re.finditer(r"`([^`]+)`", struct_match.group(2)):
            tag = tag_match.group(1)
            json_match = re.search(r'json:"([^"]+)"', tag)
            if not json_match:
                continue
            field_name = json_match.group(1).split(",", 1)[0]
            if not field_name or field_name == "-":
                continue
            fields.append(field_name)
            binding_match = re.search(r'binding:"([^"]+)"', tag)
            if binding_match and "required" in [item.strip() for item in binding_match.group(1).split(",")]:
                required_fields.append(field_name)
        if fields:
            schemas.append(Schema(package, type_name, fields, required_fields, rel(path)))
    return schemas


def read_route_keys_from_doc(path: Path) -> set[str]:
    routes: set[str] = set()
    for line in read_lines(path):
        match = re.match(r"^\|\s*(GET|POST|PUT|PATCH|DELETE)\s*\|\s*([^|]+?)\s*\|", line)
        if match:
            routes.add(route_key(match.group(1), match.group(2).strip()))
    return routes


def read_schema_route_mappings(path: Path) -> dict[str, dict[str, Any]]:
    mappings: dict[str, dict[str, Any]] = {}
    for line in read_lines(path):
        match = re.match(
            r"^\|\s*(GET|POST|PUT|PATCH|DELETE)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*(\d+)\s*\|",
            line,
        )
        if not match:
            continue
        key = route_key(match.group(1), match.group(2).strip())
        if key in mappings:
            raise RuntimeError(f"duplicate API schema route mapping: {key}")
        mappings[key] = {
            "Method": match.group(1),
            "Path": match.group(2).strip(),
            "Request": match.group(3).strip(),
            "Success": match.group(4).strip(),
            "Status": int(match.group(5)),
        }
    return mappings


def schema_refs(cell: str) -> list[str]:
    return [value for value in re.findall(r"`([^`]+)`", cell) if re.match(r"^\w+http\.\w+$", value)]


def read_required_fields_from_doc(path: Path) -> dict[str, list[str]]:
    required: dict[str, list[str]] = {}
    for line in read_lines(path):
        match = re.match(r"^\|\s*`(\w+http\.\w+)`\s*\|\s*((?:`[^`]+`(?:,\s*)?)+)\s*\|", line)
        if not match:
            continue
        key = match.group(1)
        if key in required:
            raise RuntimeError(f"duplicate documented required-field schema key: {key}")
        required[key] = re.findall(r"`([^`]+)`", match.group(2))
    return required


def verify_api_schema(args: argparse.Namespace) -> None:
    doc_path = REPO / args.doc_path
    route_doc_path = REPO / args.route_doc_path
    if not doc_path.exists():
        raise SystemExit(f"API schema doc not found: {args.doc_path}")
    if not route_doc_path.exists():
        raise SystemExit(f"API route contract doc not found: {args.route_doc_path}")

    delivery_files = [
        path
        for path in (REPO / "internal").rglob("*.go")
        if "/delivery/" in path.as_posix() and "http/" in path.as_posix() and not path.name.endswith("_test.go")
    ]
    actual_schemas: dict[str, Schema] = {}
    for path in delivery_files:
        for schema in struct_schemas_from_file(path):
            key = schema_key(schema.package, schema.type)
            if key in actual_schemas:
                raise RuntimeError(f"duplicate handler schema key: {key}")
            actual_schemas[key] = schema

    documented_schemas: dict[str, list[str]] = {}
    for line in read_lines(doc_path):
        match = re.match(r"^\|\s*`([^`]+)`\s*\|\s*`([^`]+)`\s*\|\s*((?:`[^`]+`(?:,\s*)?)+)\s*\|", line)
        if not match:
            continue
        key = schema_key(match.group(1), match.group(2))
        if key in documented_schemas:
            raise RuntimeError(f"duplicate documented schema key: {key}")
        documented_schemas[key] = re.findall(r"`([^`]+)`", match.group(3))

    actual_keys = sorted(actual_schemas)
    documented_keys = sorted(documented_schemas)
    missing_in_doc = sorted(key for key in actual_keys if key not in documented_schemas)
    stale_in_doc = sorted(key for key in documented_keys if key not in actual_schemas)
    field_mismatches = []
    for key in actual_keys:
        if key in documented_schemas and actual_schemas[key].fields != documented_schemas[key]:
            field_mismatches.append(
                {
                    "schema": key,
                    "source": actual_schemas[key].source,
                    "actual": actual_schemas[key].fields,
                    "documented": documented_schemas[key],
                }
            )

    actual_required = {
        key: actual_schemas[key].required_fields for key in actual_keys if actual_schemas[key].required_fields
    }
    documented_required = read_required_fields_from_doc(doc_path)
    missing_required_docs = sorted(key for key in actual_required if key not in documented_required)
    stale_required_docs = sorted(key for key in documented_required if key not in actual_required)
    required_mismatches = []
    for key in sorted(actual_required):
        if key in documented_required and actual_required[key] != documented_required[key]:
            required_mismatches.append(
                {
                    "schema": key,
                    "source": actual_schemas[key].source,
                    "actual": actual_required[key],
                    "documented": documented_required[key],
                }
            )

    contract_routes = read_route_keys_from_doc(route_doc_path)
    schema_mappings = read_schema_route_mappings(doc_path)
    schema_routes = set(schema_mappings)
    missing_route_mappings = sorted(contract_routes - schema_routes)
    stale_route_mappings = sorted(schema_routes - contract_routes)
    invalid_schema_refs = []
    invalid_statuses = []
    for key in sorted(schema_mappings):
        mapping = schema_mappings[key]
        for schema_ref in schema_refs(mapping["Request"]) + schema_refs(mapping["Success"]):
            if schema_ref not in documented_schemas or schema_ref not in actual_schemas:
                invalid_schema_refs.append({"route": key, "schema": schema_ref})
        if mapping["Status"] not in {200, 201, 204}:
            invalid_statuses.append({"route": key, "status": mapping["Status"]})
        if mapping["Status"] == 204 and mapping["Success"] != "none":
            invalid_statuses.append({"route": key, "status": mapping["Status"], "reason": "204 success must be none"})

    if any(
        [
            missing_in_doc,
            stale_in_doc,
            field_mismatches,
            missing_required_docs,
            stale_required_docs,
            required_mismatches,
            missing_route_mappings,
            stale_route_mappings,
            invalid_schema_refs,
            invalid_statuses,
        ]
    ):
        fail(
            {
                "missing_in_doc": missing_in_doc,
                "stale_in_doc": stale_in_doc,
                "field_mismatches": field_mismatches,
                "missing_required_field_docs": missing_required_docs,
                "stale_required_field_docs": stale_required_docs,
                "required_field_mismatches": required_mismatches,
                "missing_route_mappings": missing_route_mappings,
                "stale_route_mappings": stale_route_mappings,
                "invalid_schema_refs": invalid_schema_refs,
                "invalid_statuses": invalid_statuses,
            },
            "API schema doc is out of sync",
        )

    print_json(
        {
            "status": "passed",
            "schema_count": len(actual_schemas),
            "route_mapping_count": len(schema_mappings),
            "required_field_schema_count": len(actual_required),
            "doc": args.doc_path,
            "route_doc": args.route_doc_path,
            "schemas": actual_keys,
        }
    )


def verify_config_contract(args: argparse.Namespace) -> None:
    load_path = REPO / "internal/platform/config/load.go"
    doc_path = REPO / args.doc_path
    env_example_path = REPO / args.env_example_path
    for label, path in [("config loader", load_path), ("configuration doc", doc_path), ("env example", env_example_path)]:
        if not path.exists():
            raise SystemExit(f"{label} not found: {path}")

    load_content = read_text(load_path)
    loaded = set(
        re.findall(r'(?:requiredString|stringDefault|stringListDefault|intDefault|boolDefault|durationDefault)\("([A-Z0-9_]+)"', load_content)
    )
    env_keys = set()
    for line in read_lines(env_example_path):
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        match = re.match(r"^([A-Z0-9_]+)\s*=", stripped)
        if match:
            env_keys.add(match.group(1))
    doc_keys = set(re.findall(r"`([A-Z][A-Z0-9_]+)`", read_text(doc_path)))

    missing_env = sorted(loaded - env_keys)
    unknown_env = sorted(env_keys - loaded)
    missing_doc = sorted(loaded - doc_keys)
    unknown_doc = sorted(doc_keys - loaded)
    if any([missing_env, unknown_env, missing_doc, unknown_doc]):
        fail(
            {
                "loaded_key_count": len(loaded),
                "missing_in_env_example": missing_env,
                "unknown_in_env_example": unknown_env,
                "missing_in_doc": missing_doc,
                "unknown_in_doc": unknown_doc,
            },
            "configuration contract is out of sync",
        )

    print_json(
        {
            "status": "passed",
            "loaded_key_count": len(loaded),
            "doc": args.doc_path,
            "env_example": args.env_example_path,
            "keys": sorted(loaded),
        }
    )


def normalize_doc_cell(value: str) -> str:
    return re.sub(r"\s+", " ", re.sub(r"<br\s*/?>", " ", value)).strip()


def remove_backticks(value: str) -> str:
    return value.replace("`", "").strip()


def normalize_required_cell(value: str) -> str:
    normalized = normalize_doc_cell(value)
    if normalized == "是":
        return "yes"
    if normalized == "否":
        return "no"
    match = re.match(r"^`(?P<provider>r2|local)`\s*(?:必需|必须)$", normalized)
    if match:
        return f"provider:{match.group('provider')}"
    return remove_backticks(normalized)


def normalize_default_cell(value: str) -> str:
    normalized = normalize_doc_cell(value)
    if normalized == "无":
        return "none"
    if normalized == "空":
        return "empty"
    prefix = "local 空值时补 "
    if normalized.startswith(prefix):
        return "local-empty-default:" + remove_backticks(normalized[len(prefix) :])
    return remove_backticks(normalized)


def convert_go_default_expression(expression: str) -> str:
    value = expression.strip()
    if re.match(r'^"(?:[^"\\]|\\.)*"$', value):
        text = ast.literal_eval(value)
        return "none" if text == "" else text
    if value == "nil":
        return "empty"
    if value in {"true", "false"}:
        return value
    match = re.match(r"^(?P<n>\d+)\s*\*\s*time\.(?P<unit>Second|Minute|Hour)$", value)
    if match:
        suffix = {"Second": "s", "Minute": "m", "Hour": "h"}[match.group("unit")]
        return f"{int(match.group('n'))}{suffix}"
    match = re.match(r"^time\.(?P<unit>Second|Minute|Hour)$", value)
    if match:
        suffix = {"Second": "s", "Minute": "m", "Hour": "h"}[match.group("unit")]
        return f"1{suffix}"
    match = re.match(r"^(?P<a>\d+)\s*\*\s*(?P<b>\d+)\s*\*\s*(?P<c>\d+)$", value)
    if match:
        return str(int(match.group("a")) * int(match.group("b")) * int(match.group("c")))
    if re.match(r"^\d+$", value):
        return value
    raise RuntimeError(f"unsupported default expression: {expression}")


def verify_config_semantics(args: argparse.Namespace) -> None:
    load_path = REPO / "internal/platform/config/load.go"
    validate_path = REPO / "internal/platform/config/validate.go"
    doc_path = REPO / args.doc_path
    for label, path in [("config loader", load_path), ("config validator", validate_path), ("configuration doc", doc_path)]:
        if not path.exists():
            raise SystemExit(f"{label} not found: {path}")

    load_content = read_text(load_path)
    validate_content = read_text(validate_path)
    expected: dict[str, dict[str, Any]] = {}
    for match in re.finditer(r'requiredString\("(?P<key>[A-Z0-9_]+)"', load_content):
        key = match.group("key")
        expected[key] = {"key": key, "required": "yes", "default": "none", "enum_values": []}

    default_pattern = r'(?:stringDefault|stringListDefault|intDefault|boolDefault|durationDefault)\("(?P<key>[A-Z0-9_]+)",\s*(?P<default>"(?:[^"\\]|\\.)*"|[^,\r\n\)]+)'
    for match in re.finditer(default_pattern, load_content):
        key = match.group("key")
        expected[key] = {
            "key": key,
            "required": "no",
            "default": convert_go_default_expression(match.group("default")),
            "enum_values": [],
        }

    if re.search(
        r'cfg\.Storage\.Provider\s*==\s*"local"\s*&&\s*cfg\.Storage\.PublicBaseURL\s*==\s*""[\s\S]*?cfg\.Storage\.PublicBaseURL\s*=\s*"http://localhost:8080/uploads"',
        load_content,
    ):
        expected["OBJECT_STORAGE_PUBLIC_BASE_URL"]["default"] = "local-empty-default:http://localhost:8080/uploads"

    for match in re.finditer(r"(?P<key>[A-Z0-9_]+) must be one of (?P<values>[a-z0-9_\-/]+)", validate_content):
        key = match.group("key")
        if key in expected:
            expected[key]["enum_values"] = [item for item in match.group("values").split("/") if item]

    for match in re.finditer(r"(?P<key>[A-Z0-9_]+) is required for r2 storage", validate_content):
        key = match.group("key")
        if key in expected and expected[key]["default"] == "none":
            expected[key]["required"] = "provider:r2"

    for match in re.finditer(r"(?P<key>[A-Z0-9_]+) cannot be empty for local storage", validate_content):
        key = match.group("key")
        if key in expected and expected[key]["default"] == "none":
            expected[key]["required"] = "provider:local"

    if "OBJECT_STORAGE_PUBLIC_BASE_URL" in expected:
        expected["OBJECT_STORAGE_PUBLIC_BASE_URL"]["required"] = "provider:r2"

    documented: dict[str, dict[str, str]] = {}
    for line in read_lines(doc_path):
        match = re.match(
            r"^\|\s*`(?P<key>[A-Z0-9_]+)`\s*\|\s*(?P<required>[^|]+?)\s*\|\s*(?P<default>[^|]+?)\s*(?:\|\s*(?P<description>[^|]*?))?\s*\|$",
            line,
        )
        if not match:
            continue
        key = match.group("key")
        documented[key] = {
            "key": key,
            "required": normalize_required_cell(match.group("required")),
            "default": normalize_default_cell(match.group("default")),
            "description": normalize_doc_cell(match.group("description") or ""),
        }

    missing_in_doc: list[str] = []
    stale_in_doc: list[str] = []
    required_mismatches: list[dict[str, str]] = []
    default_mismatches: list[dict[str, str]] = []
    enum_mismatches: list[dict[str, str]] = []

    for key, want in expected.items():
        if key not in documented:
            missing_in_doc.append(key)
            continue
        actual = documented[key]
        if actual["required"] != want["required"]:
            required_mismatches.append({"key": key, "expected": want["required"], "actual": actual["required"]})
        if actual["default"] != want["default"]:
            default_mismatches.append({"key": key, "expected": want["default"], "actual": actual["default"]})
        for enum_value in want["enum_values"]:
            if not re.search(rf"(^|[^a-z0-9_-]){re.escape(enum_value)}([^a-z0-9_-]|$)", actual["description"]):
                enum_mismatches.append(
                    {"key": key, "missing_enum_value": enum_value, "description": actual["description"]}
                )

    for key in documented:
        if key not in expected:
            stale_in_doc.append(key)

    if any([missing_in_doc, stale_in_doc, required_mismatches, default_mismatches, enum_mismatches]):
        fail(
            {
                "missing_in_doc": sorted(missing_in_doc),
                "stale_in_doc": sorted(stale_in_doc),
                "required_mismatches": required_mismatches,
                "default_mismatches": default_mismatches,
                "enum_mismatches": enum_mismatches,
            },
            "configuration semantic contract is out of sync",
        )

    print_json(
        {
            "status": "passed",
            "checked_key_count": len(expected),
            "enum_key_count": len([value for value in expected.values() if value["enum_values"]]),
            "doc": args.doc_path,
            "keys": sorted(expected),
        }
    )


def verify_http_errors(args: argparse.Namespace) -> None:
    doc_path = REPO / args.doc_path
    apperr_path = REPO / "internal/apperr/apperr.go"
    error_path = REPO / "internal/platform/httpserver/error.go"
    response_path = REPO / "internal/platform/httpserver/response.go"
    if not doc_path.exists():
        raise SystemExit(f"HTTP error contract doc not found: {args.doc_path}")

    http_status_values = {
        "StatusBadRequest": 400,
        "StatusUnauthorized": 401,
        "StatusForbidden": 403,
        "StatusNotFound": 404,
        "StatusConflict": 409,
        "StatusTooManyRequests": 429,
        "StatusInternalServerError": 500,
    }

    apperr_content = read_text(apperr_path)
    code_by_const = {
        match.group(1): match.group(2)
        for match in re.finditer(r'(?m)^\s*(Code\w+)\s+Code\s*=\s*"([^"]+)"', apperr_content)
    }

    error_content = read_text(error_path)
    status_by_code: dict[str, int] = {}
    for match in re.finditer(
        r"(?s)case\s+apperr\.(Code\w+):\s*return\s+http\.(Status\w+),\s*errorResponse\(err\)",
        error_content,
    ):
        const_name, status_name = match.group(1), match.group(2)
        if const_name not in code_by_const:
            raise RuntimeError(f"HTTP error mapper references unknown apperr constant: {const_name}")
        if status_name not in http_status_values:
            raise RuntimeError(f"HTTP error mapper references unsupported HTTP status: {status_name}")
        status_by_code[code_by_const[const_name]] = http_status_values[status_name]

    if "CodeInternal" in code_by_const and re.search(
        r'http\.StatusInternalServerError[\s\S]*Code:\s*string\(apperr\.CodeInternal\)[\s\S]*Message:\s*"internal server error"',
        error_content,
    ):
        status_by_code[code_by_const["CodeInternal"]] = 500

    response_fields = [
        field.split(",", 1)[0]
        for field in re.findall(r'json:"([^"]+)"', read_text(response_path))
        if field.split(",", 1)[0] and field.split(",", 1)[0] != "-"
    ]
    documented_status: dict[str, int] = {}
    for line in read_lines(doc_path):
        match = re.match(r"^\|\s*`([^`]+)`\s*\|\s*(\d+)\s*\|", line)
        if match:
            documented_status[match.group(1)] = int(match.group(2))

    code_values = sorted(code_by_const.values())
    actual_codes = sorted(status_by_code)
    documented_codes = sorted(documented_status)
    missing_mapper = sorted(code for code in code_values if code not in status_by_code)
    extra_mapper = sorted(code for code in actual_codes if code not in code_values)
    missing_doc = sorted(code for code in code_values if code not in documented_status)
    stale_doc = sorted(code for code in documented_codes if code not in code_values)
    status_mismatches = [
        {"code": code, "actual": status_by_code[code], "documented": documented_status[code]}
        for code in code_values
        if code in status_by_code and code in documented_status and status_by_code[code] != documented_status[code]
    ]

    doc_content = read_text(doc_path)
    required_shape = ["error", "code", "message"]
    missing_shape = [
        field
        for field in required_shape
        if field not in response_fields or (f'"{field}"' not in doc_content and f"`{field}`" not in doc_content)
    ]

    if any([missing_mapper, extra_mapper, missing_doc, stale_doc, status_mismatches, missing_shape]):
        fail(
            {
                "missing_mapper_codes": missing_mapper,
                "extra_mapper_codes": extra_mapper,
                "missing_doc_codes": missing_doc,
                "stale_doc_codes": stale_doc,
                "status_mismatches": status_mismatches,
                "missing_response_shape": missing_shape,
            },
            "HTTP error contract doc is out of sync",
        )

    print_json(
        {
            "status": "passed",
            "doc": args.doc_path,
            "code_count": len(code_values),
            "response_shape": required_shape,
            "mappings": {key: status_by_code[key] for key in sorted(status_by_code)},
        }
    )


def migration_key(version: str, name: str) -> str:
    return f"{version}_{name}"


def verify_migrations(args: argparse.Namespace) -> None:
    migration_dir = REPO / args.migration_dir
    doc_path = REPO / args.doc_path
    if not migration_dir.exists():
        raise SystemExit(f"migration directory not found: {args.migration_dir}")
    if not doc_path.exists():
        raise SystemExit(f"migration contract doc not found: {args.doc_path}")

    pattern = re.compile(r"^(?P<version>\d{6})_(?P<name>[a-z0-9_]+)\.(?P<direction>up|down)\.sql$")
    entries = []
    unexpected = []
    for path in sorted(migration_dir.iterdir()):
        if not path.is_file():
            continue
        match = pattern.match(path.name)
        if not match:
            unexpected.append(path.name)
            continue
        entries.append(
            {
                "version": match.group("version"),
                "version_number": int(match.group("version")),
                "name": match.group("name"),
                "direction": match.group("direction"),
                "file": path.name,
            }
        )
    if not entries:
        raise SystemExit("no migration files found")

    grouped: dict[str, list[dict[str, Any]]] = {}
    for entry in entries:
        grouped.setdefault(entry["version"], []).append(entry)

    versions = sorted(int(version) for version in grouped)
    contiguity_errors = [
        {"expected": f"{index + 1:06d}", "actual": f"{version:06d}"}
        for index, version in enumerate(versions)
        if version != index + 1
    ]
    pair_errors = []
    actual_inventory = []
    for version in sorted(grouped):
        items = grouped[version]
        ups = [item for item in items if item["direction"] == "up"]
        downs = [item for item in items if item["direction"] == "down"]
        if len(ups) != 1 or len(downs) != 1:
            pair_errors.append(
                {
                    "version": version,
                    "up_files": [item["file"] for item in ups],
                    "down_files": [item["file"] for item in downs],
                }
            )
            continue
        if ups[0]["name"] != downs[0]["name"]:
            pair_errors.append({"version": version, "up_name": ups[0]["name"], "down_name": downs[0]["name"]})
            continue
        actual_inventory.append(
            {"version": version, "name": ups[0]["name"], "up": ups[0]["file"], "down": downs[0]["file"]}
        )

    documented_inventory = []
    doc_version_counts: dict[str, int] = {}
    for line in read_lines(doc_path):
        match = re.match(r"^\|\s*(?P<version>\d{6})\s*\|\s*(?P<name>[a-z0-9_]+)\s*\|", line)
        if not match:
            continue
        version, name = match.group("version"), match.group("name")
        documented_inventory.append({"version": version, "name": name, "key": migration_key(version, name)})
        doc_version_counts[version] = doc_version_counts.get(version, 0) + 1

    actual_by_version = {item["version"]: item["name"] for item in actual_inventory}
    doc_by_version: dict[str, str] = {}
    for item in documented_inventory:
        doc_by_version.setdefault(item["version"], item["name"])

    duplicate_doc_versions = sorted(version for version, count in doc_version_counts.items() if count > 1)
    missing_in_doc = []
    stale_in_doc = []
    doc_name_mismatches = []
    for item in actual_inventory:
        if item["version"] not in doc_by_version:
            missing_in_doc.append(migration_key(item["version"], item["name"]))
        elif doc_by_version[item["version"]] != item["name"]:
            doc_name_mismatches.append(
                {
                    "version": item["version"],
                    "actual_name": item["name"],
                    "documented_name": doc_by_version[item["version"]],
                }
            )
    for item in documented_inventory:
        if item["version"] not in actual_by_version:
            stale_in_doc.append(item["key"])

    if any(
        [
            unexpected,
            pair_errors,
            contiguity_errors,
            duplicate_doc_versions,
            missing_in_doc,
            stale_in_doc,
            doc_name_mismatches,
        ]
    ):
        fail(
            {
                "unexpected_files": sorted(unexpected),
                "pair_errors": pair_errors,
                "contiguity_errors": contiguity_errors,
                "duplicate_documented_versions": duplicate_doc_versions,
                "missing_in_doc": sorted(missing_in_doc),
                "stale_in_doc": sorted(stale_in_doc),
                "doc_name_mismatches": doc_name_mismatches,
            },
            "migration contract is out of sync",
        )

    print_json(
        {
            "status": "passed",
            "migration_count": len(actual_inventory),
            "doc": args.doc_path,
            "migrations": sorted(actual_inventory, key=lambda item: item["version"]),
        }
    )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Verify source-derived contract documentation.")
    sub = parser.add_subparsers(dest="command", required=True)

    api_contract = sub.add_parser("api-contract")
    api_contract.add_argument("--doc-path", default="docs/contracts/http-api-contract.md")
    api_contract.set_defaults(func=verify_api_contract)

    api_schema = sub.add_parser("api-schema")
    api_schema.add_argument("--doc-path", default="docs/contracts/http-api-schema.md")
    api_schema.add_argument("--route-doc-path", default="docs/contracts/http-api-contract.md")
    api_schema.set_defaults(func=verify_api_schema)

    config_contract = sub.add_parser("config-contract")
    config_contract.add_argument("--doc-path", default="docs/contracts/configuration.md")
    config_contract.add_argument("--env-example-path", default=".env.example")
    config_contract.set_defaults(func=verify_config_contract)

    config_semantics = sub.add_parser("config-semantics")
    config_semantics.add_argument("--doc-path", default="docs/contracts/configuration.md")
    config_semantics.set_defaults(func=verify_config_semantics)

    http_errors = sub.add_parser("http-errors")
    http_errors.add_argument("--doc-path", default="docs/contracts/http-error-handling.md")
    http_errors.set_defaults(func=verify_http_errors)

    migrations = sub.add_parser("migrations")
    migrations.add_argument("--migration-dir", default="migrations")
    migrations.add_argument("--doc-path", default="docs/contracts/migrations.md")
    migrations.set_defaults(func=verify_migrations)

    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    try:
        args.func(args)
    except BrokenPipeError:
        raise
    except SystemExit:
        raise
    except Exception as exc:
        raise SystemExit(str(exc)) from exc


if __name__ == "__main__":
    main()
