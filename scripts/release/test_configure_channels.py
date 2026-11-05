import copy
import unittest

from configure_channels import REQUIRED_FILES, configure, specification, verify_rule


class FakeGitHub:
    repository = "NARUBROWN/example"

    def __init__(self, missing_file=False):
        self.calls = []
        self.branches = {"main": {"name": "main", "commit": {"sha": "a" * 40}}}
        self.rules = {}
        self.missing_file = missing_file

    def api(self, path="", data=None, method=None):
        self.calls.append((path, data, method))
        if not path:
            return {"permissions": {"admin": True}, "archived": False, "default_branch": "main"}
        if path.startswith("branches?"):
            return list(self.branches.values())
        if path.startswith("contents/"):
            if self.missing_file:
                raise ValueError("설정 코드가 아직 main에 없습니다")
            return {"sha": "b" * 40}
        if path.startswith("rulesets?"):
            return [{"name": rule["name"], "id": key} for key, rule in self.rules.items()]
        if path == "git/refs":
            name = data["ref"].removeprefix("refs/heads/")
            self.branches[name] = {"name": name, "commit": {"sha": data["sha"]}}
            return {}
        if path.startswith("git/ref/heads/"):
            return self.branches[path.removeprefix("git/ref/heads/")]
        if path == "rulesets":
            key = len(self.rules) + 1
            self.rules[key] = copy.deepcopy(data)
            return {"id": key}
        if path.startswith("rulesets/"):
            key = int(path.split("/")[1])
            if data is not None:
                self.rules[key] = copy.deepcopy(data)
            return {"id": key, **self.rules[key]}
        raise AssertionError((path, data, method))


class ConfigureChannelTests(unittest.TestCase):
    def test_dry_run_has_no_writes(self):
        github = FakeGitHub()
        plan = configure(github, False)
        self.assertEqual(plan["create_from_main"], ["preview", "rc"])
        self.assertTrue(all(data is None and method is None for _, data, method in github.calls))

    def test_bootstrap_preflight_precedes_all_writes(self):
        github = FakeGitHub(missing_file=True)
        with self.assertRaises(ValueError):
            configure(github, False, apply=True)
        self.assertTrue(all(data is None for _, data, _ in github.calls))

    def test_apply_is_idempotent_and_preserves_existing_channel_refs(self):
        github = FakeGitHub()
        github.branches["preview"] = {"name": "preview", "commit": {"sha": "c" * 40}}
        for _ in range(2):
            self.assertEqual(configure(github, True, apply=True)["result"], "configured")
        self.assertEqual(github.branches["preview"]["commit"]["sha"], "c" * 40)
        self.assertEqual(len(github.rules), 3)
        creates = [call for call in github.calls if call[0] == "git/refs"]
        self.assertEqual(len(creates), 1)
        self.assertEqual(creates[0][1]["ref"], "refs/heads/rc")
        files = [path for path, _, _ in github.calls if path.startswith("contents/")]
        self.assertIn("contents/" + REQUIRED_FILES[0] + "?ref=" + "c" * 40, files)

    def test_main_and_rc_require_merge_commits_and_channel_specific_checks(self):
        for branch in ("preview", "rc", "main"):
            rule = specification(branch, True)
            parameters = {item["type"]: item.get("parameters", {}) for item in rule["rules"]}
            self.assertEqual(parameters["pull_request"]["allowed_merge_methods"],
                             ["merge", "squash", "rebase"] if branch == "preview" else ["merge"])
            checks = parameters["required_status_checks"]["required_status_checks"]
            self.assertEqual([item["context"] for item in checks], [
                "Branch Flow / " + branch, "Example Channel / " + branch, "Example Verification / " + branch])
            self.assertEqual(parameters["required_status_checks"]["strict_required_status_checks_policy"],
                             branch == "preview")

    def test_user_example_does_not_require_removed_application_ci(self):
        rule = specification("main", False)
        checks = rule["rules"][-1]["parameters"]["required_status_checks"]
        self.assertEqual(len(checks), 2)

    def test_wrong_required_status_is_detected_after_write(self):
        expected = specification("main", False)
        actual = copy.deepcopy(expected)
        actual["rules"][-1]["parameters"]["required_status_checks"][0]["context"] = "Branch Flow / preview"
        with self.assertRaises(ValueError):
            verify_rule(actual, expected)


if __name__ == "__main__":
    unittest.main()
