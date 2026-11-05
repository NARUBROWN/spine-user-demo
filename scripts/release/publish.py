#!/usr/bin/env python3
"""검증된 예제를 Preview, RC 또는 Stable 태그로 발행합니다."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess

from example_channels import verify


TAG = re.compile(r"v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(preview|rc)\.([1-9]\d*))?\Z")
SHA = re.compile(r"[0-9a-f]{40}\Z")


def run(*args):
    return subprocess.run(args, check=True, text=True, capture_output=True).stdout.strip()


def parse_tag(tag):
    match = TAG.fullmatch(tag)
    if not match:
        raise ValueError("태그는 vX.Y.Z-preview.N, vX.Y.Z-rc.N 또는 vX.Y.Z 형식이어야 합니다")
    return "v" + ".".join(match.groups()[:3]), match.group(4) or "stable"


def validate_inputs(branch, version, sha, source):
    base, channel = parse_tag(version)
    if {"preview": "preview", "rc": "rc", "main": "stable"}.get(branch) != channel:
        raise ValueError("태그 채널과 실행 브랜치가 다릅니다")
    if not SHA.fullmatch(sha):
        raise ValueError("발행할 브랜치 HEAD의 전체 40자리 SHA가 필요합니다")
    previous = {"rc": "preview", "stable": "rc"}.get(channel)
    if previous:
        source_base, source_channel = parse_tag(source)
        if source_base != base or source_channel != previous:
            raise ValueError("같은 버전의 이전 단계 태그만 승격할 수 있습니다")
    elif source:
        raise ValueError("Preview에는 이전 단계 태그를 지정하지 않습니다")
    return channel


def require_checks(checks, branch, application_ci, sha):
    required = ["Example Channel / " + branch]
    if application_ci:
        required.append("Example Verification / " + branch)
    for name in required:
        matches = [check for check in checks if check["name"] == name and
                   check.get("app", {}).get("id") == 15368 and check.get("head_sha") == sha]
        latest = max(matches, key=lambda check: check["id"], default={})
        if latest.get("status") != "completed" or latest.get("conclusion") != "success":
            raise ValueError("브랜치 HEAD에서 성공한 검사가 필요합니다: " + name)


class GitHub:
    def __init__(self, repository):
        self.repository = repository

    def api(self, path, data=None):
        command = ["gh", "api", "repos/" + self.repository + "/" + path]
        payload = None
        if data is not None:
            command.extend(["--method", "POST", "--input", "-"])
            payload = json.dumps(data)
        output = subprocess.run(command, input=payload, check=True, text=True, capture_output=True).stdout
        return json.loads(output)


def publish(github, root, branch, version, sha, source):
    channel = validate_inputs(branch, version, sha, source)
    if run("git", "rev-parse", "HEAD") != sha:
        raise ValueError("체크아웃한 커밋이 요청한 SHA와 다릅니다")
    if github.api("git/ref/heads/" + branch)["object"]["sha"] != sha:
        raise ValueError("브랜치 HEAD가 바뀌었습니다. 새 SHA로 다시 실행하세요")
    report = verify(root, branch)
    pages = json.loads(run("gh", "api", "--paginate", "--slurp",
                           f"repos/{github.repository}/commits/{sha}/check-runs?per_page=100"))
    checks = [check for page in pages for check in page["check_runs"]]
    policy = json.loads((root / ".github/example-channels.json").read_text())
    require_checks(checks, branch, policy["application_ci"], sha)
    if source:
        # 같은 소스를 앞 단계에서 발행했는지 확인하고, merge 커밋 차이만 허용합니다.
        source_sha = run("git", "rev-parse", f"refs/tags/{source}^{{commit}}")
        run("git", "merge-base", "--is-ancestor", source_sha, sha)
        if run("git", "rev-parse", f"{source_sha}^{{tree}}") != run("git", "rev-parse", f"{sha}^{{tree}}"):
            raise ValueError("이전 단계 태그와 소스 트리가 다릅니다. 새 Preview부터 검증하세요")
    # 발행 중 끊긴 작업은 같은 SHA에서만 재시도하며 기존 태그는 덮어쓰지 않습니다.
    ref = "refs/tags/" + version
    existing = [item for item in github.api("git/matching-refs/tags/" + version) if item["ref"] == ref]
    if existing:
        if existing[0]["object"]["type"] != "commit" or existing[0]["object"]["sha"] != sha:
            raise ValueError("기존 태그가 다른 커밋을 가리킵니다: " + version)
    else:
        github.api("git/refs", {"ref": ref, "sha": sha})
    if channel == "stable":
        notes = (f"예제 Stable 버전 `{version}`입니다.\n\n"
                 f"- 소스: `{sha}`\n- 승격한 RC: `{source}`\n"
                 f"- Spine CLI: `{report['cli_version']}`\n\n"
                 "Spine과 플랫폼의 버전은 이 태그의 `go.mod`를 확인하세요.\n")
        github.api("releases", {"tag_name": version, "name": version, "body": notes,
                                "draft": False, "prerelease": False, "make_latest": "true"})
    return {"repository": github.repository, "version": version, "channel": channel,
            "source_sha": sha, "source_tag": source or None, "github_release": channel == "stable"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True)
    parser.add_argument("--sha", required=True)
    parser.add_argument("--source", default="")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    try:
        report = publish(GitHub(os.environ["GITHUB_REPOSITORY"]), root, os.environ["GITHUB_REF_NAME"],
                         args.version, args.sha, args.source)
    except (KeyError, ValueError, subprocess.CalledProcessError) as error:
        detail = error.stderr if isinstance(error, subprocess.CalledProcessError) else str(error)
        parser.exit(1, detail + "\n")
    output = json.dumps(report, ensure_ascii=False, indent=2)
    print(output)
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
            summary.write("```json\n" + output + "\n```\n")


if __name__ == "__main__":
    main()
