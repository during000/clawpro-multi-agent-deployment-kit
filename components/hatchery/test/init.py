#!/usr/bin/env python3
"""
Init script: configure security group, VPC, subnets, create test user and get API token.
"""

import json
import os
import sys
from datetime import datetime, timedelta, timezone

import requests

API = os.environ.get("API", "").rstrip("/")
ADMIN_TOKEN = os.environ.get("ADMIN_TOKEN", "")
VPC_ID = "vpc-2h0cpxlz"

TEST_USERNAME = os.environ.get("TEST_USERNAME", "testuser")
TEST_PASSWORD = os.environ.get("TEST_PASSWORD", "test123456")
ADMIN_USERNAME = os.environ.get("ADMIN_USERNAME", "testadmin")
ADMIN_PASSWORD = os.environ.get("ADMIN_PASSWORD", "test123456")

if not API or not ADMIN_TOKEN:
    print("ERROR: please set environment variables API and ADMIN_TOKEN")
    print("  export API=http://134.175.254.166/")
    print("  export ADMIN_TOKEN=clawpro-test-token")
    sys.exit(1)

HEADERS = {
    "Authorization": f"Bearer {ADMIN_TOKEN}",
    "Accept": "application/json",
}


def health_check():
    print(">>> Health check ...")
    resp = requests.get(f"{API}/health", timeout=10)
    resp.raise_for_status()
    data = resp.json()
    assert data.get("status") == "ok", f"Health check failed: {data}"
    print("    OK")


def setup_cvm_template():
    print(">>> Setup CVM template ...")

    action_time = (datetime.now(timezone.utc) + timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ")

    template = {
        "InstanceChargeType": "POSTPAID_BY_HOUR",
        "InstanceType": "Ai2.MEDIUM4",
        "SystemDisk": {
            "DiskType": "CLOUD_BSSD",
            "DiskSize": 50,
        },
        "InternetAccessible": {
            "InternetChargeType": "TRAFFIC_POSTPAID_BY_HOUR",
            "InternetMaxBandwidthOut": 5,
            "PublicIpAssigned": True,
        },
        "ActionTimer": {
            "TimerAction": "TerminateInstances",
            "ActionTime": action_time,
        },
    }

    resp = requests.post(
        f"{API}/admin/config/cvm",
        headers=HEADERS,
        data={"cvm_template": json.dumps(template)},
        timeout=10,
    )
    resp.raise_for_status()
    data = resp.json()
    if not data.get("ok"):
        print(f"    Failed: {data}")
        sys.exit(1)

    print(f"    Charge type: POSTPAID_BY_HOUR")
    print(f"    Auto destroy: {action_time}")


def setup_security_group():
    print(">>> Setup security group ...")

    resp = requests.get(
        f"{API}/admin/config/security-group/ruleset",
        headers=HEADERS,
        timeout=10,
    )
    resp.raise_for_status()
    data = resp.json()

    if data.get("initialized"):
        print(f"    Already exists: {data.get('name')} (skipped)")
        return

    resp = requests.post(
        f"{API}/admin/config/security-group/rulesets",
        headers=HEADERS,
        json={
            "name": "hatchery-test-sg",
            "description": "Hatchery test security group",
            "auto_fix_rules": True,
            "rules": [],
        },
        timeout=30,
    )
    if resp.status_code == 500:
        print(f"    ERROR: POST rulesets 返回 500 ({resp.text[:200]})")
        sys.exit(1)

    resp.raise_for_status()
    data = resp.json()

    if not data.get("initialized"):
        print(f"    Failed: {data}")
        sys.exit(1)

    sg_list = data.get("projected_to", [])
    sg_id = sg_list[0]["sg_id"] if sg_list else "unknown"
    print(f"    Created: {data.get('name')} -> sg {sg_id}")


def setup_vpc_subnet():
    print(">>> Setup VPC and subnets ...")

    vpc_id = VPC_ID

    if not vpc_id:
        resp = requests.get(
            f"{API}/admin/vpc/cloud",
            headers=HEADERS,
            params={"limit": 10},
            timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()
        vpcs = data.get("vpcs", [])
        if not vpcs:
            print("    ERROR: no VPC available")
            sys.exit(1)
        vpc_id = vpcs[0]["vpc_id"]
        print(f"    Auto selected VPC: {vpc_id} ({vpcs[0].get('name', '')})")
    else:
        resp = requests.get(
            f"{API}/admin/vpc/cloud",
            headers=HEADERS,
            params={"vpc_id": vpc_id},
            timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()
        vpcs = data.get("vpcs", [])
        if not vpcs:
            print(f"    ERROR: VPC {vpc_id} not found")
            sys.exit(1)
        print(f"    VPC: {vpc_id} ({vpcs[0].get('name', '')})")

    zones = [
        "ap-guangzhou-6", "ap-guangzhou-7",
    ]
    subnet_ids = {}

    for zone in zones:
        resp = requests.get(
            f"{API}/admin/subnet/cloud",
            headers=HEADERS,
            params={"vpc_id": vpc_id, "zone": zone},
            timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()
        subnets = data.get("subnets", [])
        if subnets:
            zone_subnet_ids = [s["subnet_id"] for s in subnets]
            subnet_ids[zone] = zone_subnet_ids
            names = ", ".join(f"{s['subnet_id']}({s.get('name', '')})" for s in subnets)
            print(f"    {zone}: {names}")

    if not subnet_ids:
        print("    ERROR: no subnets found")
        sys.exit(1)

    resp = requests.post(
        f"{API}/admin/config/cvm",
        headers=HEADERS,
        data={
            "vpc_id": vpc_id,
            "subnet_ids": json.dumps(subnet_ids),
        },
        timeout=10,
    )
    resp.raise_for_status()
    data = resp.json()
    if not data.get("ok"):
        print(f"    Failed: {data}")
        sys.exit(1)

    print(f"    VPC/subnet config saved ({len(subnet_ids)} zones)")


def _ensure_instance_quota(user_id, quota):
    """确保已有用户的 instance_quota 不低于期望值"""
    resp = requests.post(
        f"{API}/admin/update-user",
        headers=HEADERS,
        params={"id": user_id},
        json={"instance_quota": quota},
        timeout=10,
    )
    if resp.status_code == 200:
        print(f"    Updated instance_quota={quota}")
    else:
        print(f"    Warning: failed to update instance_quota: {resp.text[:200]}")


def create_user(username, password, role="user", instance_quota=5):
    print(f">>> Create {role} user ({username}) ...")
    resp = requests.post(
        f"{API}/admin/create",
        headers=HEADERS,
        json={
            "username": username,
            "password": password,
            "role": role,
            "instance_quota": instance_quota,
        },
        timeout=10,
    )
    data = resp.json()

    if data.get("ok"):
        user_id = data["id"]
        print(f"    Created: id={user_id}")
        return user_id

    error = data.get("error", "")
    if "exist" in error.lower() or "already" in error.lower() or "已存在" in error:
        print("    User already exists, looking up ...")
        resp = requests.get(
            f"{API}/admin/users",
            headers=HEADERS,
            timeout=10,
        )
        resp.raise_for_status()
        data = resp.json()
        users = data.get("users", [])
        for u in users:
            uname = u.get("username") or u.get("Username")
            uid = u.get("id") or u.get("ID")
            if uname == username:
                print(f"    Found: id={uid}")
                _ensure_instance_quota(uid, instance_quota)
                return uid
        print("    ERROR: user exists but not found in list")
        sys.exit(1)

    print(f"    Failed: {error}")
    sys.exit(1)


def get_token(user_id):
    print(f">>> Get API token (user_id={user_id}) ...")
    resp = requests.get(
        f"{API}/admin/user-token",
        headers=HEADERS,
        params={"id": user_id},
        timeout=10,
    )
    resp.raise_for_status()
    data = resp.json()

    token = data.get("token", "")
    if token:
        print(f"    Token: {token}")
        return token

    if data.get("exists"):
        mask = data.get("mask", "")
        print(f"    Token already exists (mask={mask}), cannot retrieve full token")
        print("    Hint: reset token via admin page if needed")
        return mask

    print(f"    Failed: {data}")
    sys.exit(1)


def main():
    global HEADERS

    health_check()
    print()

    # Setup CVM template using bootstrap admin-token (sensitive fields require it)
    setup_cvm_template()
    print()

    # Create admin user using bootstrap token, get its API token
    admin_user_id = create_user(ADMIN_USERNAME, ADMIN_PASSWORD, "admin")
    print()

    admin_token = get_token(admin_user_id)
    print(f"  export ADMIN_TOKEN={admin_token}")
    print()

    # Switch to the created admin's token for all subsequent operations
    HEADERS = {
        "Authorization": f"Bearer {admin_token}",
        "Accept": "application/json",
    }

    setup_security_group()
    print()

    setup_vpc_subnet()
    print()

    # Create regular test user using the admin token
    user_id = create_user(TEST_USERNAME, TEST_PASSWORD, "user")
    print()

    token = get_token(user_id)
    print(f"  export TOKEN={token}")
    print()


if __name__ == "__main__":
    main()
