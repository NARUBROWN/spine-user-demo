#!/usr/bin/env python3
"""예제 브랜치에 맞는 Spine과 CLI 의존성인지 확인합니다."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess


BRANCHES = ("preview", "rc", "main")
VERSION = re.compile(r"v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?\Z")
COMPONENT = re.compile(r"github\.com/NARUBROWN/spine(?:-.+-platform|-validator-playground)?\Z")


def check_version(version, branch):
    match = VERSION.fullmatch(version)
    if branch not in BRANCHES or not match:
        raise ValueError(f"지원하지 않는 채널 또는 버전입니다: {branch}, {version}")
    suffix = match.group(4)
    if branch == "main" and suffix:
        raise ValueError(f"main은 정식 버전만 사용합니다: {version}")
    if branch == "rc" and suffix and not re.fullmatch(r"rc\.[1-9]\d*", suffix):
        raise ValueError(f"rc는 정식 또는 RC 버전만 사용합니다: {version}")


def check_module(module, branch):
    components = {item["Path"]: item["Version"] for item in module.get("Require") or []
                  if COMPONENT.fullmatch(item["Path"])}
    if "github.com/NARUBROWN/spine" not in components:
        raise ValueError("검사 대상 모듈에 Spine 의존성이 없습니다")
    for item in module.get("Replace") or []:
        if COMPONENT.fullmatch(item["Old"]["Path"]):
            raise ValueError("Spine 구성 요소의 replace는 허용하지 않습니다: " + item["Old"]["Path"])
    for path, version in components.items():
        check_version(version, branch)
    return components


def load_policy(root):
    policy = json.loads((root / ".github/example-channels.json").read_text())
    if policy.get("schema_version") != "spine.example-channels/v1":
        raise ValueError("예제 채널 정책의 스키마가 다릅니다")
    if not policy.get("modules") or not isinstance(policy.get("application_ci"), bool):
        raise ValueError("검사할 모듈과 application_ci를 지정해야 합니다")
    return policy


def verify(root, branch):
    policy = load_policy(root)
    check_version(policy["cli_version"], branch)
    report = {"branch": branch, "cli_version": policy["cli_version"], "modules": {}}
    env = {**os.environ, "GOWORK": "off", "GOTOOLCHAIN": "local"}
    for directory in policy["modules"]:
        path = (root / directory).resolve()
        if not path.is_relative_to(root) or not (path / "go.mod").is_file():
            raise ValueError("검사 대상은 저장소 안의 Go 모듈이어야 합니다: " + directory)
        result = subprocess.run(["go", "mod", "edit", "-json"], cwd=path, env=env,
                                check=True, text=True, capture_output=True)
        report["modules"][directory] = check_module(json.loads(result.stdout), branch)
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--channel", required=True, choices=BRANCHES)
    parser.add_argument("--github-output", type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    try:
        report = verify(root, args.channel)
    except (ValueError, KeyError, subprocess.CalledProcessError) as error:
        parser.exit(1, str(error) + "\n")
    print(json.dumps(report, ensure_ascii=False, indent=2))
    if args.github_output:
        with args.github_output.open("a") as output:
            output.write("cli_version=" + report["cli_version"] + "\n")
    if os.environ.get("GITHUB_STEP_SUMMARY"):
        lines = [f"## 예제 채널: `{args.channel}`", "", f"CLI: `{report['cli_version']}`", ""]
        for module, components in report["modules"].items():
            lines.append(f"### `{module}`")
            lines.extend(f"- `{path}@{version}`" for path, version in components.items())
            lines.append("")
        with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
            summary.write("\n".join(lines) + "\n")


if __name__ == "__main__":
    main()
