#!/usr/bin/env python3
"""Enrich generated SCP data source schemas with MarkdownDescription from the OpenAPI spec."""

import json
import re
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
SCP_DIR = REPO_ROOT / "internal" / "provider" / "scp"
OPENAPI_URL = "https://www.servercontrolpanel.de/scp-core/api/v1/openapi"

# Map generated data source package names to the GET operation whose summary is used
# as the schema MarkdownDescription.
DATA_SOURCE_OPS = {
    "scpmaintenance": ("/api/v1/maintenance", "get"),
    "scprdnsipv4": ("/api/v1/rdns/ipv4/{ip}", "get"),
    "scprdnsipv6": ("/api/v1/rdns/ipv6/{ip}", "get"),
    "scpserver": ("/api/v1/servers/{serverId}", "get"),
    "scpserverdisk": ("/api/v1/servers/{serverId}/disks/{diskName}", "get"),
    "scpserverdisks": ("/api/v1/servers/{serverId}/disks", "get"),
    "scpserverdisksupporteddrivers": ("/api/v1/servers/{serverId}/disks/supported-drivers", "get"),
    "scpservergpudriver": ("/api/v1/servers/{serverId}/gpu-driver", "get"),
    "scpserverguestagent": ("/api/v1/servers/{serverId}/guest-agent", "get"),
    "scpserverguestagentstatus": ("/api/v1/servers/{serverId}/guest-agent/status", "get"),
    "scpserverimageflavours": ("/api/v1/servers/{serverId}/imageflavours", "get"),
    "scpserverinterface": ("/api/v1/servers/{serverId}/interfaces/{mac}", "get"),
    "scpserverinterfaces": ("/api/v1/servers/{serverId}/interfaces", "get"),
    "scpserveriso": ("/api/v1/servers/{serverId}/iso", "get"),
    "scpserverisoimages": ("/api/v1/servers/{serverId}/isoimages", "get"),
    "scpserverlogs": ("/api/v1/servers/{serverId}/logs", "get"),
    "scpserverrescuesystem": ("/api/v1/servers/{serverId}/rescuesystem", "get"),
    "scpservers": ("/api/v1/servers", "get"),
    "scpserversnapshot": ("/api/v1/servers/{serverId}/snapshots/{name}", "get"),
    "scpserversnapshots": ("/api/v1/servers/{serverId}/snapshots", "get"),
    "scptask": ("/api/v1/tasks/{uuid}", "get"),
    "scptasks": ("/api/v1/tasks", "get"),
    "scpuser": ("/api/v1/users/{userId}", "get"),
    "scpuserfailoveripsv4": ("/api/v1/users/{userId}/failoverips/v4", "get"),
    "scpuserfailoveripsv6": ("/api/v1/users/{userId}/failoverips/v6", "get"),
    "scpuserfirewallpolicies": ("/api/v1/users/{userId}/firewall-policies", "get"),
    "scpuserfirewallpolicy": ("/api/v1/users/{userId}/firewall-policies/{id}", "get"),
    "scpuserimage": ("/api/v1/users/{userId}/images/{key}", "get"),
    "scpuserimagepart": ("/api/v1/users/{userId}/images/{key}/{uploadId}/parts/{partNumber}", "get"),
    "scpuserimages": ("/api/v1/users/{userId}/images", "get"),
    "scpuseriso": ("/api/v1/users/{userId}/isos/{key}", "get"),
    "scpuserisopart": ("/api/v1/users/{userId}/isos/{key}/{uploadId}/parts/{partNumber}", "get"),
    "scpuserisos": ("/api/v1/users/{userId}/isos", "get"),
    "scpuserlogs": ("/api/v1/users/{userId}/logs", "get"),
    "scpusersshkeys": ("/api/v1/users/{userId}/ssh-keys", "get"),
    "scpuservlan": ("/api/v1/users/{userId}/vlans/{vlanId}", "get"),
    "scpuservlans": ("/api/v1/users/{userId}/vlans", "get"),
    "scpvlan": ("/api/v1/vlans/{vlanId}", "get"),
}


def load_openapi() -> dict:
    with urllib.request.urlopen(OPENAPI_URL, timeout=30) as resp:  # noqa: S310
        return json.load(resp)


def go_string_literal(s: str) -> str:
    """Return a Go double-quoted string literal for the given text."""
    return json.dumps(s)


def get_summary(spec: dict, path: str, method: str) -> str:
    op = spec["paths"][path][method]
    summary = op.get("summary", "")
    description = op.get("description", "")
    text = summary or description
    return " ".join(text.split())


def find_schema_file(pkg_dir: Path) -> Path | None:
    for f in pkg_dir.glob("*_data_source_gen.go"):
        return f
    return None


def patch_schema_markdown_description(file_path: Path, description: str) -> bool:
    content = file_path.read_text()
    desc_literal = go_string_literal(description)

    # Target the top-level schema.Schema opening, replacing any existing
    # MarkdownDescription and preserving the indentation of the Attributes block.
    pattern = re.compile(
        r"(return schema\.Schema\{\n)(?:\s*MarkdownDescription:[^,\n]+,\n)?(\s*)Attributes:",
    )
    if not pattern.search(content):
        return False

    replacement = f"\\1\\2MarkdownDescription: {desc_literal},\n\\2Attributes:"
    new_content, count = pattern.subn(replacement, content, count=1)
    if count == 0:
        return False

    file_path.write_text(new_content)
    return True


def main() -> None:
    spec = load_openapi()
    updated = 0

    for pkg, (path, method) in DATA_SOURCE_OPS.items():
        summary = get_summary(spec, path, method)
        if not summary:
            continue

        pkg_dir = SCP_DIR / pkg
        schema_file = find_schema_file(pkg_dir)
        if schema_file is None:
            print(f"warn: no generated schema for {pkg}")
            continue

        if patch_schema_markdown_description(schema_file, summary):
            print(f"updated {schema_file.relative_to(REPO_ROOT)}")
            updated += 1

    print(f"updated {updated} generated data source schemas")


if __name__ == "__main__":
    main()
