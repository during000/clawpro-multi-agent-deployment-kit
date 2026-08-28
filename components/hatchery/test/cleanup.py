#!/usr/bin/env python3
"""
Cleanup script: remove cloud resources created during integration test.

Steps:
  1. Terminate CVM instances whose name contains the test identifier.
  2. Delete security groups whose name contains the test identifier.

Environment variables:
  API            - Hatchery API base URL (required)
  ADMIN_TOKEN    - Admin bearer token (required)
  IDENTIFIER     - Test identifier to match in resource names (required, must be non-empty)
"""

import os
import sys
import time
import requests

API = os.environ.get("API", "").rstrip("/")
ADMIN_TOKEN = os.environ.get("ADMIN_TOKEN", "")
IDENTIFIER = os.environ.get("IDENTIFIER", "")

if not API or not ADMIN_TOKEN:
    print("ERROR: API and ADMIN_TOKEN are required")
    sys.exit(1)

if not IDENTIFIER:
    print("ERROR: IDENTIFIER is required and must be non-empty")
    sys.exit(1)

HEADERS = {
    "Authorization": f"Bearer {ADMIN_TOKEN}",
    "Accept": "application/json",
}


# ========== CVM Cleanup ==========

def list_instances():
    """List instances managed by hatchery (already filtered by identifier via DB isolation).

    注意：/admin/instances 后端默认 page_size=20（parsePagination defaultPS=20），
    若不显式指定 page_size，cleanup 只能拿到最新 20 条记录，导致最早创建
    的实例被分页漏掉、最终残留在云上、并卡住 SG 删除。
    后端 maxPS=1000 足够覆盖一次集成测试中创建的所有实例。
    """
    resp = requests.get(
        f"{API}/admin/instances",
        headers=HEADERS,
        params={"page_size": 1000},
        timeout=30,
    )
    resp.raise_for_status()
    data = resp.json()
    return data.get("instances", [])


def delete_instances(db_ids):
    """Delete instances via admin API (handles CVM termination internally)."""
    resp = requests.post(
        f"{API}/admin/instances/delete",
        headers={**HEADERS, "Content-Type": "application/json"},
        json={"ids": db_ids},
        timeout=60,
    )
    resp.raise_for_status()
    data = resp.json()
    if not data.get("ok"):
        return data.get("error", "unknown error")
    return None


def cleanup_cvm_instances():
    """Delete instances managed by this hatchery instance via admin delete API."""
    print(f">>> Cleanup CVM instances for identifier: {IDENTIFIER}")

    instances = list_instances()
    # Filter to those with a cloud instance_id (already provisioned)
    matched = [
        inst for inst in instances
        if inst.get("InstanceId") or inst.get("instance_id")
    ]
    if not matched:
        print("    No instances to terminate")
        return True

    print(f"    Found {len(matched)} instance(s):")
    instance_ids = []
    db_ids = []
    for inst in matched:
        iid = inst.get("InstanceId") or inst.get("instance_id")
        db_id = inst.get("ID") or inst.get("id")
        name = inst.get("Name") or inst.get("name") or ""
        print(f"      - {iid} (db_id={db_id}, name={name})")
        instance_ids.append(iid)
        if db_id:
            db_ids.append(db_id)

    if not db_ids:
        print("    No valid DB IDs found, skip deletion")
        return False

    err = delete_instances(db_ids)
    if err:
        print(f"    FAILED to delete: {err}")
        return False

    print(f"    Delete request submitted for {len(db_ids)} instance(s)")

    # Wait until instances are completely gone from cloud
    print("    Waiting for instances to disappear from cloud...")
    timeout = 300
    start = time.time()
    while time.time() - start < timeout:
        resp = requests.post(
            f"{API}/admin/cloud/query/cvm",
            headers={**HEADERS, "X-TC-Action": "DescribeInstances"},
            json={"InstanceIds": instance_ids},
            timeout=30,
        )
        resp.raise_for_status()
        data = resp.json()
        remaining = data.get("Response", {}).get("InstanceSet", [])
        if not remaining:
            break
        print(f"    {len(remaining)} instance(s) still exist, waiting...", flush=True)
        time.sleep(10)
    else:
        print(f"    WARNING: timeout waiting for instances to disappear")
        return False

    print("    All instances gone")
    return True


# ========== Security Group Cleanup ==========

def list_security_groups():
    """List security groups whose name contains IDENTIFIER via cloud proxy."""
    all_sgs = []
    offset = 0
    limit = 100

    while True:
        resp = requests.post(
            f"{API}/admin/cloud/query/vpc",
            headers={**HEADERS, "X-TC-Action": "DescribeSecurityGroups"},
            json={
                "Offset": str(offset),
                "Limit": str(limit),
                "Filters": [{"Name": "security-group-name", "Values": [IDENTIFIER]}],
            },
            timeout=30,
        )
        resp.raise_for_status()
        data = resp.json()

        response = data.get("Response", {})
        sg_set = response.get("SecurityGroupSet", [])
        total = int(response.get("TotalCount", 0))

        all_sgs.extend(sg_set)

        if len(all_sgs) >= total or len(sg_set) == 0:
            break
        offset += limit

    return all_sgs


def delete_security_group(sg_id):
    """Delete a security group via cloud proxy."""
    resp = requests.post(
        f"{API}/admin/cloud/mutate/vpc",
        headers={**HEADERS, "X-TC-Action": "DeleteSecurityGroup"},
        json={"SecurityGroupId": sg_id},
        timeout=30,
    )
    resp.raise_for_status()
    data = resp.json()
    if data.get("Response", {}).get("Error"):
        return data["Response"]["Error"]
    return None


def cleanup_security_groups():
    """Delete security groups whose name contains IDENTIFIER, with retries."""
    print(f">>> Cleanup security groups matching: {IDENTIFIER}")

    sgs = list_security_groups()

    if not sgs:
        print("    No matching security groups")
        return True

    print(f"    Matched {len(sgs)} security group(s):")
    for sg in sgs:
        print(f"      - {sg['SecurityGroupId']} ({sg.get('SecurityGroupName', '')})")

    max_retries = 6
    retry_interval = 10
    remaining = list(sgs)

    for attempt in range(max_retries):
        failed = []
        for sg in remaining:
            sg_id = sg["SecurityGroupId"]
            sg_name = sg.get("SecurityGroupName", "")
            err = delete_security_group(sg_id)
            if err:
                failed.append(sg)
                if attempt == 0:
                    print(f"    FAILED to delete {sg_id} ({sg_name}): {err}")
            else:
                print(f"    Deleted {sg_id} ({sg_name})")

        remaining = failed
        if not remaining:
            break
        if attempt < max_retries - 1:
            print(f"    {len(remaining)} remaining, retrying in {retry_interval}s...")
            time.sleep(retry_interval)

    if remaining:
        print(f"    {len(remaining)} security group(s) could not be deleted after {max_retries} attempts")
        return False

    print(f"    All security groups deleted")
    return True


# ========== Main ==========

def main():
    ok = True

    if not cleanup_cvm_instances():
        ok = False

    if not cleanup_security_groups():
        ok = False

    if not ok:
        sys.exit(1)


if __name__ == "__main__":
    main()
