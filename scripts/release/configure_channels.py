#!/usr/bin/env python3
"""main의 채널 설정을 바탕으로 예제 브랜치와 보호 규칙을 구성합니다."""

import argparse
import json
from pathlib import Path
import re
import subprocess

from example_channels import BRANCHES, load_policy


REQUIRED_FILES = (
    ".github/workflows/branch-flow.yml",
    ".github/workflows/example-channels.yml",
    ".github/example-channels.json",
    "scripts/release/branch_flow.cjs",
    "scripts/release/example_channels.py",
    ".github/workflows/release.yml",
    "scripts/release/publish.py",
)


def specification(branch, application_ci):
    contexts = ["Branch Flow / " + branch, "Example Channel / " + branch]
    if application_ci:
        contexts.append("Example Verification / " + branch)
    return {
        "name": "example-channel-" + branch,
        "target": "branch",
        "enforcement": "active",
        "bypass_actors": [],
        "conditions": {"ref_name": {"include": ["refs/heads/" + branch], "exclude": []}},
        "rules": [
            {"type": "deletion"},
            {"type": "non_fast_forward"},
            {"type": "pull_request", "parameters": {
                "allowed_merge_methods": ["merge", "squash", "rebase"] if branch == "preview" else ["merge"],
                "required_approving_review_count": 0,
                "dismiss_stale_reviews_on_push": True,
                "require_code_owner_review": False,
                "require_last_push_approval": False,
                "required_review_thread_resolution": True,
            }},
            {"type": "required_status_checks", "parameters": {
                # 승격 브랜치를 최신 상태로 만들기 위한 역방향 merge를 요구하지 않습니다.
                "strict_required_status_checks_policy": branch == "preview",
                "do_not_enforce_on_create": True,
                "required_status_checks": [{"context": context, "integration_id": 15368} for context in contexts],
            }},
        ],
    }


class GitHub:
    def __init__(self, repository):
        if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
            raise ValueError("저장소는 owner/name 형식이어야 합니다")
        self.repository = repository

    def api(self, path="", data=None, method=None):
        command = ["gh", "api", "repos/" + self.repository + ("/" + path if path else "")]
        payload = None
        if data is not None:
            command.extend(["--method", method or "POST", "--input", "-"])
            payload = json.dumps(data)
        result = subprocess.run(command, input=payload, check=True, text=True, capture_output=True)
        return json.loads(result.stdout) if result.stdout.strip() else None


def verify_rule(actual, expected):
    for key in ("name", "target", "enforcement", "bypass_actors", "conditions"):
        if actual.get(key) != expected[key]:
            raise ValueError("보호 규칙이 다릅니다: " + key)
    rules = {rule["type"]: rule.get("parameters", {}) for rule in actual["rules"]}
    for rule in expected["rules"]:
        if rule["type"] not in rules:
            raise ValueError("필수 보호 규칙이 없습니다: " + rule["type"])
        for key, value in rule.get("parameters", {}).items():
            if rules[rule["type"]].get(key) != value:
                raise ValueError("보호 규칙의 값이 다릅니다: " + key)


def configure(github, application_ci, apply=False):
    repo = github.api()
    if not repo.get("permissions", {}).get("admin") or repo.get("archived"):
        raise ValueError("보관되지 않은 저장소의 관리자 권한이 필요합니다")
    branches = {branch["name"]: branch for branch in github.api("branches?per_page=100")}
    if "main" not in branches:
        raise ValueError("기준 main 브랜치가 없습니다")
    if not apply:
        return {"repository": github.repository, "default_branch": "main",
                "create_from_main": [branch for branch in BRANCHES if branch not in branches],
                "rulesets": [specification(branch, application_ci) for branch in BRANCHES]}

    # 설정 코드가 공개된 다음에만 브랜치를 만들고 필수 상태를 활성화합니다.
    main_sha = branches["main"]["commit"]["sha"]
    for branch in BRANCHES:
        sha = branches.get(branch, {}).get("commit", {}).get("sha", main_sha)
        for path in REQUIRED_FILES + ((".github/workflows/ci.yml",) if application_ci else ()):
            github.api(f"contents/{path}?ref={sha}")
    rulesets = {rule["name"]: rule for rule in github.api("rulesets?per_page=100")}
    for branch in BRANCHES:
        if branch not in branches:
            github.api("git/refs", {"ref": "refs/heads/" + branch, "sha": main_sha})
        expected = specification(branch, application_ci)
        previous = rulesets.get(expected["name"])
        if previous:
            created = github.api(f"rulesets/{previous['id']}", expected, "PUT")
        else:
            created = github.api("rulesets", expected)
        verify_rule(github.api(f"rulesets/{created['id']}"), expected)
    github.api(data={"default_branch": "main"}, method="PATCH")
    if github.api()["default_branch"] != "main":
        raise ValueError("기본 브랜치가 main으로 설정되지 않았습니다")
    for branch in BRANCHES:
        github.api("git/ref/heads/" + branch)
    return {"repository": github.repository, "result": "configured", "default_branch": "main",
            "branches": list(BRANCHES)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository", required=True, help="owner/name")
    parser.add_argument("--apply", action="store_true", help="브랜치와 보호 규칙을 실제로 적용")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    try:
        plan = configure(GitHub(args.repository), load_policy(root)["application_ci"], args.apply)
    except (ValueError, subprocess.CalledProcessError) as error:
        detail = error.stderr if isinstance(error, subprocess.CalledProcessError) else str(error)
        parser.exit(1, detail + "\n")
    print(json.dumps(plan, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
