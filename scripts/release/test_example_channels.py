import copy
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from example_channels import check_module, check_version, verify


CORE = "github.com/NARUBROWN/spine"


class ExampleChannelTests(unittest.TestCase):
    def test_dependency_matrix(self):
        cases = {
            "v0.6.5": {"preview", "rc", "main"},
            "v0.6.6-preview.1": {"preview"},
            "v0.6.6-rc.1": {"preview", "rc"},
            "v0.6.6-0.20261103042116-77419ec0a2d2": {"preview"},
            "v0.6.6-beta.1": {"preview"},
            "v0.6.6-rc.0": {"preview"},
            "latest": set(),
            "main": set(),
            "v00.6.5": set(),
            "v0.6.5\ncli_version=latest": set(),
        }
        for version, allowed in cases.items():
            for branch in ("preview", "rc", "main"):
                with self.subTest(version=version, branch=branch):
                    if branch in allowed:
                        check_version(version, branch)
                    else:
                        with self.assertRaises(ValueError):
                            check_version(version, branch)

    def test_every_spine_platform_is_checked(self):
        for path in (CORE, CORE + "-echo-platform", CORE + "-gorilla-platform",
                     CORE + "-kafka-platform", CORE + "-validator-playground"):
            module = {"Require": [{"Path": CORE, "Version": "v0.6.5"},
                                  {"Path": path, "Version": "v0.6.6-preview.1"}]}
            with self.subTest(path=path), self.assertRaises(ValueError):
                check_module(module, "rc")

    def test_local_msa_module_is_allowed_but_spine_replacements_are_not(self):
        module = {"Require": [{"Path": CORE, "Version": "v0.6.5"}],
                  "Replace": [{"Old": {"Path": CORE + "-simple-msa"}, "New": {"Path": ".."}}]}
        self.assertEqual(check_module(module, "main"), {CORE: "v0.6.5"})
        for path in (CORE, CORE + "-kafka-platform"):
            replaced = copy.deepcopy(module)
            replaced["Replace"].append({"Old": {"Path": path}, "New": {"Path": "../local"}})
            with self.subTest(path=path), self.assertRaises(ValueError):
                check_module(replaced, "preview")

    def test_missing_spine_dependency_is_rejected(self):
        with self.assertRaises(ValueError):
            check_module({"Require": None}, "main")

    def test_real_multi_module_project_and_cli_policy(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            (root / ".github").mkdir()
            policy = {"schema_version": "spine.example-channels/v1", "cli_version": "v0.4.2",
                      "modules": ["order", "stock"], "application_ci": True}
            policy_path = root / ".github/example-channels.json"
            policy_path.write_text(json.dumps(policy))
            for name in policy["modules"]:
                (root / name).mkdir()
                (root / name / "go.mod").write_text(f"module example.com/{name}\n\ngo 1.25.0\n\nrequire {CORE} v0.6.5\n")
            self.assertEqual(len(verify(root, "main")["modules"]), 2)
            # 루트 외에 워커의 의존성도 검사합니다.
            stock = root / "stock/go.mod"
            stock.write_text(stock.read_text().replace("v0.6.5", "v0.6.6-preview.1"))
            with self.assertRaises(ValueError):
                verify(root, "rc")
            self.assertEqual(len(verify(root, "preview")["modules"]), 2)
            policy["cli_version"] = "v0.4.3-preview.1"
            policy_path.write_text(json.dumps(policy))
            with self.assertRaises(ValueError):
                verify(root, "rc")

    def test_module_path_cannot_escape_repository(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            (root / ".github").mkdir()
            (root / ".github/example-channels.json").write_text(json.dumps({
                "schema_version": "spine.example-channels/v1", "cli_version": "v0.4.2",
                "modules": [".."], "application_ci": False,
            }))
            with patch("example_channels.subprocess.run") as run:
                with self.assertRaises(ValueError):
                    verify(root, "preview")
                run.assert_not_called()


if __name__ == "__main__":
    unittest.main()
